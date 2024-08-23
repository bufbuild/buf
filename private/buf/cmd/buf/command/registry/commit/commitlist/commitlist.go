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

package commitlist

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

	defaultPageSize = 10
)

// NewCommand returns a new Command
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <remote/owner/module[:ref]>",
		Short: "List modules commits",
		Long: `This command lists commits in a module based on the reference specified.
For a commit reference, it lists the commit itself.
For a label reference, it lists the current and past commits associated with this label.
If no reference is specified, it lists all commits in this module.
`,
		Args: appcmd.ExactArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	Format    string
	PageSize  uint32
	PageToken string
	Reverse   bool
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.Uint32Var(&f.PageSize,
		pageSizeFlagName,
		defaultPageSize,
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
		`Reverse the results. By default, they are ordered with the newest first`,
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
	moduleRef, err := bufmodule.ParseModuleRef(container.Arg(0))
	if err != nil {
		return appcmd.WrapInvalidArgumentError(err)
	}
	format, err := bufprint.ParseFormat(flags.Format)
	if err != nil {
		return appcmd.WrapInvalidArgumentError(err)
	}
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	registry := moduleRef.ModuleFullName().Registry()
	clientProvider := bufapi.NewClientProvider(clientConfig)
	commitServiceClient := clientProvider.V1CommitServiceClient(registry)
	labelServiceClient := clientProvider.V1LabelServiceClient(registry)
	resourceServiceClient := clientProvider.V1ResourceServiceClient(registry)
	resourceResp, err := resourceServiceClient.GetResources(
		ctx,
		connect.NewRequest(
			&modulev1.GetResourcesRequest{
				ResourceRefs: []*modulev1.ResourceRef{
					{
						Value: &modulev1.ResourceRef_Name_{
							Name: &modulev1.ResourceRef_Name{
								Owner:  moduleRef.ModuleFullName().Owner(),
								Module: moduleRef.ModuleFullName().Name(),
								Child: &modulev1.ResourceRef_Name_Ref{
									Ref: moduleRef.Ref(),
								},
							},
						},
					},
				},
			},
		),
	)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return bufcli.NewModuleRefNotFoundError(moduleRef)
		}
		return err
	}
	resources := resourceResp.Msg.Resources
	if len(resources) != 1 {
		return syserror.Newf("expect 1 resource from response, got %d", len(resources))
	}
	resource := resources[0]
	if commit := resource.GetCommit(); commit != nil {
		// If the ref is a commit, the commit is the only result and there is no next page.
		return bufprint.PrintPage(
			container.Stdout(),
			format,
			"",
			"",
			[]bufprint.Entity{bufprint.NewCommitEntity(commit, moduleRef.ModuleFullName())},
		)
	}
	if resource.GetModule() != nil {
		// The ref is a module, ListCommits returns all the commits.
		commitOrder := modulev1.ListCommitsRequest_ORDER_CREATE_TIME_DESC
		if flags.Reverse {
			commitOrder = modulev1.ListCommitsRequest_ORDER_CREATE_TIME_ASC
		}
		resp, err := commitServiceClient.ListCommits(
			ctx,
			connect.NewRequest(
				&modulev1.ListCommitsRequest{
					PageSize:  flags.PageSize,
					PageToken: flags.PageToken,
					ResourceRef: &modulev1.ResourceRef{
						Value: &modulev1.ResourceRef_Name_{
							Name: &modulev1.ResourceRef_Name{
								Owner:  moduleRef.ModuleFullName().Owner(),
								Module: moduleRef.ModuleFullName().Name(),
							},
						},
					},
					Order: commitOrder,
				},
			),
		)
		if err != nil {
			if connect.CodeOf(err) == connect.CodeNotFound {
				return bufcli.NewModuleRefNotFoundError(moduleRef)
			}
			return err
		}
		return bufprint.PrintPage(
			container.Stdout(),
			format,
			resp.Msg.NextPageToken,
			nextPageCommand(container, flags, resp.Msg.NextPageToken),
			slicesext.Map(resp.Msg.Commits, func(commit *modulev1.Commit) bufprint.Entity {
				return bufprint.NewCommitEntity(commit, moduleRef.ModuleFullName())
			}),
		)
	}
	label := resource.GetLabel()
	if label == nil {
		// This should be impossible because getLabelOrCommitForRef would've returned an error.
		return syserror.Newf("%s is neither a commit nor a label", moduleRef.String())
	}
	// The ref is a label. Call ListLabelHistory to get all commits.
	labelHistoryOrder := modulev1.ListLabelHistoryRequest_ORDER_DESC
	if flags.Reverse {
		labelHistoryOrder = modulev1.ListLabelHistoryRequest_ORDER_ASC
	}
	resp, err := labelServiceClient.ListLabelHistory(
		ctx,
		connect.NewRequest(
			&modulev1.ListLabelHistoryRequest{
				PageSize:  flags.PageSize,
				PageToken: flags.PageToken,
				LabelRef: &modulev1.LabelRef{
					Value: &modulev1.LabelRef_Name_{
						Name: &modulev1.LabelRef_Name{
							Owner:  moduleRef.ModuleFullName().Owner(),
							Module: moduleRef.ModuleFullName().Name(),
							Label:  moduleRef.Ref(),
						},
					},
				},
				Order: labelHistoryOrder,
			},
		),
	)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			// This should be impossible since we just checked that the ref is a label.
			return bufcli.NewModuleRefNotFoundError(moduleRef)
		}
		return err
	}
	commits := slicesext.Map(
		resp.Msg.Values,
		func(value *modulev1.ListLabelHistoryResponse_Value) *modulev1.Commit {
			return value.Commit
		},
	)
	return bufprint.PrintPage(
		container.Stdout(),
		format,
		resp.Msg.NextPageToken,
		nextPageCommand(container, flags, resp.Msg.NextPageToken),
		slicesext.Map(commits, func(commit *modulev1.Commit) bufprint.Entity {
			return bufprint.NewCommitEntity(commit, moduleRef.ModuleFullName())
		}),
	)
}

func nextPageCommand(container appext.Container, flags *flags, nextPageToken string) string {
	if nextPageToken == "" {
		return ""
	}
	command := fmt.Sprintf("buf registry commit list %s", container.Arg(0))
	if flags.PageSize != defaultPageSize {
		command = fmt.Sprintf("%s --%s %d", command, pageSizeFlagName, flags.PageSize)
	}
	if flags.Reverse {
		command = fmt.Sprintf("%s --%s", command, reverseFlagName)
	}
	if flags.Format != bufprint.FormatText.String() {
		command = fmt.Sprintf("%s --%s %s", command, formatFlagName, flags.Format)
	}
	return fmt.Sprintf("%s --%s %s", command, pageTokenFlagName, nextPageToken)
}
