// Copyright 2020-2024 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package draftlist

import (
	"context"
	"fmt"

	modulev1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufprint"
	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/spf13/pflag"
)

const (
	pageSizeFlagName  = "page-size"
	pageTokenFlagName = "page-token"
	reverseFlagName   = "reverse"
	formatFlagName    = "format"
)

// NewCommand returns a new Command
func NewCommand(name string, builder appext.SubCommandBuilder) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <buf.build/owner/repository>",
		Short: "List repository drafts",
		Args:  appcmd.ExactArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	PageSize  uint32
	PageToken string
	Reverse   bool
	Format    string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.Uint32Var(&f.PageSize,
		pageSizeFlagName,
		10,
		`The page size`,
	)
	flagSet.StringVar(&f.PageToken,
		pageTokenFlagName,
		"",
		`The page token. If more results are available, a "next_page" key is present in the --format=json output`,
	)
	flagSet.BoolVar(&f.Reverse,
		reverseFlagName,
		false,
		`Reverse the results`,
	)
	flagSet.StringVar(
		&f.Format,
		formatFlagName,
		bufprint.FormatText.String(),
		fmt.Sprintf(`The output format to use. Must be one of %s`, bufprint.AllFormatsString),
	)
}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) error {
	bufcli.WarnBetaCommand(ctx, container)
	moduleFullName, err := bufmodule.ParseModuleFullName(container.Arg(0))
	if err != nil {
		return appcmd.NewInvalidArgumentError(err.Error())
	}
	format, err := bufprint.ParseFormat(flags.Format)
	if err != nil {
		return appcmd.NewInvalidArgumentError(err.Error())
	}
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	moduleServiceClient := bufapi.NewClientProvider(clientConfig).V1ModuleServiceClient(moduleFullName.Registry())
	moduleResp, err := moduleServiceClient.GetModules(
		ctx,
		&connect.Request[modulev1.GetModulesRequest]{
			Msg: &modulev1.GetModulesRequest{
				ModuleRefs: []*modulev1.ModuleRef{
					{
						Value: &modulev1.ModuleRef_Name_{
							Name: &modulev1.ModuleRef_Name{
								Owner:  moduleFullName.Owner(),
								Module: moduleFullName.Name(),
							},
						},
					},
				},
			},
		},
	)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return bufcli.NewRepositoryNotFoundError(container.Arg(0))
		}
		return err
	}
	modules := moduleResp.Msg.GetModules()
	if len(modules) != 1 {
		return syserror.Newf("expected 1 module from response, got %d", len(modules))
	}
	defaultLabelName := modules[0].GetDefaultLabelName()
	labelServiceClient := bufapi.NewClientProvider(clientConfig).V1LabelServiceClient(moduleFullName.Registry())
	order := modulev1.ListLabelsRequest_ORDER_CREATE_TIME_ASC
	if flags.Reverse {
		order = modulev1.ListLabelsRequest_ORDER_CREATE_TIME_DESC
	}
	labelsResp, err := labelServiceClient.ListLabels(
		ctx,
		&connect.Request[modulev1.ListLabelsRequest]{
			Msg: &modulev1.ListLabelsRequest{
				// Get 1 extra label to account for the default label returned. We will need to
				// remove the default label from the response. It's also possible that the response
				// does not contain the default label, in which case we will remove the last one.
				PageSize:  flags.PageSize + 1,
				PageToken: flags.PageToken,
				Order:     *order.Enum(),
				ResourceRef: &modulev1.ResourceRef{
					Value: &modulev1.ResourceRef_Name_{
						Name: &modulev1.ResourceRef_Name{
							Owner:  moduleFullName.Owner(),
							Module: moduleFullName.Name(),
						},
					},
				},
				ArchiveFilter: modulev1.ListLabelsRequest_ARCHIVE_FILTER_UNARCHIVED_ONLY,
			},
		},
	)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return bufcli.NewRepositoryNotFoundError(container.Arg(0))
		}
		return err
	}
	labels := labelsResp.Msg.GetLabels()
	nextPageToken := labelsResp.Msg.NextPageToken
	if len(labels) > 0 {
		respLabelCount := len(labels)
		labels = slicesext.Filter(
			labels,
			func(label *modulev1.Label) bool {
				return label.GetName() != defaultLabelName
			},
		)
		switch {
		case len(labels) == respLabelCount-1:
			// This means the default label was filtered out.
			break
		case len(labels) == respLabelCount && respLabelCount <= int(flags.PageSize):
			// Less than or equal to page size of labels are returned, no need to trim the slice.
			break
		case len(labels) == respLabelCount && respLabelCount == int(flags.PageSize)+1:
			// We got pageSize+1 labels back.
			labels = labels[0:flags.PageSize]
			// We also need a different next page token:
			// Say there are 10 labels (1 - 10), page size is 2, we send a request
			// for 3 tokens and we get 3, 4, 5 back. The token that comes with this response
			// indicates that the next page starts with 6. However, we only display 3 and 4
			// to the user, we need a token that indicates 5 next.
			resp, err := labelServiceClient.ListLabels(
				ctx,
				&connect.Request[modulev1.ListLabelsRequest]{
					Msg: &modulev1.ListLabelsRequest{
						// We ask for the correct size.
						PageSize:  flags.PageSize,
						PageToken: flags.PageToken,
						Order:     *order.Enum(),
						ResourceRef: &modulev1.ResourceRef{
							Value: &modulev1.ResourceRef_Name_{
								Name: &modulev1.ResourceRef_Name{
									Owner:  moduleFullName.Owner(),
									Module: moduleFullName.Name(),
								},
							},
						},
						ArchiveFilter: modulev1.ListLabelsRequest_ARCHIVE_FILTER_UNARCHIVED_ONLY,
					},
				},
			)
			if err != nil {
				if connect.CodeOf(err) == connect.CodeNotFound {
					return bufcli.NewRepositoryNotFoundError(container.Arg(0))
				}
				return err
			}
			nextPageToken = resp.Msg.GetNextPageToken()
		default:
			return syserror.Newf("incorrect number of labels after filtering: %d", len(labels))
		}
	}
	return bufprint.NewRepositoryDraftPrinter(container.Stdout()).
		PrintRepositoryDrafts(ctx, format, nextPageToken, labels...)
}
