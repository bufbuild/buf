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

	"buf.build/gen/go/bufbuild/registry/connectrpc/go/buf/registry/module/v1/modulev1connect"
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
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <buf.build/owner/repository[:ref]>",
		Short: "List repository commits",
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
	moduleRef, err := bufmodule.ParseModuleRef(container.Arg(0))
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
	clientProvider := bufapi.NewClientProvider(clientConfig)
	commitServiceClient := clientProvider.V1CommitServiceClient(moduleRef.ModuleFullName().Registry())
	labelServiceClient := clientProvider.V1LabelServiceClient(moduleRef.ModuleFullName().Registry())
	// The moduleRef is a module, ListCommits returns all the commits.
	if moduleRef.Ref() == "" {
		commitOrder := modulev1.ListCommitsRequest_ORDER_CREATE_TIME_ASC
		if flags.Reverse {
			commitOrder = modulev1.ListCommitsRequest_ORDER_CREATE_TIME_DESC
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
		return bufprint.NewRepositoryCommitPrinter(container.Stdout()).
			PrintRepositoryCommits(ctx, format, resp.Msg.NextPageToken, resp.Msg.Commits...)
	}
	label, commit, err := getLabelOrCommitForRef(ctx, labelServiceClient, commitServiceClient, moduleRef)
	if err != nil {
		return err
	}
	if commit != nil {
		// If the ref is a commit, the commit is the only result and there is no next page.
		return bufprint.NewRepositoryCommitPrinter(container.Stdout()).PrintRepositoryCommits(ctx, format, "", commit)
	}
	if label == nil {
		// This should be impossible.
		return syserror.Newf("%s is neither a commit nor a label", moduleRef.String())
	}
	// The ref is a label. Call ListLabelHistory to get all the commits.
	labelHistoryOrder := modulev1.ListLabelHistoryRequest_ORDER_ASC
	if flags.Reverse {
		labelHistoryOrder = modulev1.ListLabelHistoryRequest_ORDER_DESC
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
	return bufprint.NewRepositoryCommitPrinter(container.Stdout()).
		PrintRepositoryCommits(ctx, format, resp.Msg.NextPageToken, commits...)
}

// A ref string could either be a label or a commit.
//
// If the ref is a label or both, returns a non-nil label.
// If the ref is a commit, returns a non-nil commit.
func getLabelOrCommitForRef(
	ctx context.Context,
	labelServiceClient modulev1connect.LabelServiceClient,
	commitServiceClient modulev1connect.CommitServiceClient,
	moduleRef bufmodule.ModuleRef,
) (*modulev1.Label, *modulev1.Commit, error) {
	// First check if it's a label.
	labelResp, err := labelServiceClient.GetLabels(
		ctx,
		connect.NewRequest(
			&modulev1.GetLabelsRequest{
				LabelRefs: []*modulev1.LabelRef{
					{
						Value: &modulev1.LabelRef_Name_{
							Name: &modulev1.LabelRef_Name{
								Owner:  moduleRef.ModuleFullName().Owner(),
								Module: moduleRef.ModuleFullName().Name(),
								Label:  moduleRef.Ref(),
							},
						},
					},
				},
			},
		),
	)
	if err == nil {
		labels := labelResp.Msg.Labels
		if len(labels) != 1 {
			return nil, nil, syserror.Newf("expect 1 label from response, got %d", len(labels))
		}
		return labels[0], nil, nil
	}
	if connect.CodeOf(err) != connect.CodeNotFound {
		return nil, nil, err
	}
	// Now check if it's a commit.
	commitResp, err := commitServiceClient.GetCommits(
		ctx,
		connect.NewRequest(
			&modulev1.GetCommitsRequest{
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
			return nil, nil, bufcli.NewModuleRefNotFoundError(moduleRef)
		}
		return nil, nil, err
	}
	commits := commitResp.Msg.Commits
	if len(commits) != 1 {
		return nil, nil, syserror.Newf("expect 1 commit from response, got %d", len(commits))
	}
	return nil, commits[0], nil
}
