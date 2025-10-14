// Copyright 2020-2025 Buf Technologies, Inc.
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

package plugincommitlist

import (
	"context"
	"fmt"

	pluginv1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/plugin/v1beta1"
	"buf.build/go/app/appcmd"
	"buf.build/go/app/appext"
	"buf.build/go/standard/xslices"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufprint"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/bufpkg/bufregistryapi/bufregistryapiplugin"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/spf13/pflag"
)

const (
	pageSizeFlagName          = "page-size"
	pageTokenFlagName         = "page-token"
	reverseFlagName           = "reverse"
	formatFlagName            = "format"
	digestChangesOnlyFlagName = "digest-changes-only"

	defaultPageSize = 10
)

// NewCommand returns a new Command
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
	deprecated string,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <remote/owner/plugin[:ref]>",
		Short: "List plugins commits",
		Long: `This command lists commits in a plugin based on the reference specified.
For a commit reference, it lists the commit itself.
For a label reference, it lists the current and past commits associated with this label.
If no reference is specified, it lists all commits in this plugin.
`,
		Args:       appcmd.ExactArgs(1),
		Deprecated: deprecated,
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	Format            string
	PageSize          uint32
	PageToken         string
	Reverse           bool
	DigestChangesOnly bool
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.Uint32Var(
		&f.PageSize,
		pageSizeFlagName,
		defaultPageSize,
		`The page size`,
	)
	flagSet.StringVar(
		&f.PageToken,
		pageTokenFlagName,
		"",
		`The page token. If more results are available, a "next_page" key is present in the --format=json output`,
	)
	flagSet.BoolVar(
		&f.Reverse,
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
	flagSet.BoolVar(
		&f.DigestChangesOnly,
		digestChangesOnlyFlagName,
		false,
		`Only commits that have changed digests. By default, all commits are listed`,
	)
}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) error {
	pluginRef, err := bufparse.ParseRef(container.Arg(0))
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
	registry := pluginRef.FullName().Registry()
	pluginClientProvider := bufregistryapiplugin.NewClientProvider(clientConfig)
	commitServiceClient := pluginClientProvider.V1Beta1CommitServiceClient(registry)
	labelServiceClient := pluginClientProvider.V1Beta1LabelServiceClient(registry)
	resourceServiceClient := pluginClientProvider.V1Beta1ResourceServiceClient(registry)

	resourceResp, err := resourceServiceClient.GetResources(
		ctx,
		connect.NewRequest(
			&pluginv1beta1.GetResourcesRequest{
				ResourceRefs: []*pluginv1beta1.ResourceRef{
					{
						Value: &pluginv1beta1.ResourceRef_Name_{
							Name: &pluginv1beta1.ResourceRef_Name{
								Owner:  pluginRef.FullName().Owner(),
								Plugin: pluginRef.FullName().Name(),
								Child: &pluginv1beta1.ResourceRef_Name_Ref{
									Ref: pluginRef.Ref(),
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
			return bufcli.NewRefNotFoundError(pluginRef)
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
			[]bufprint.Entity{bufprint.NewCommitEntity(commit, pluginRef.FullName(), commit.GetSourceControlUrl())},
		)
	}
	if resource.GetPlugin() != nil {
		// The ref is a plugin, ListCommits returns all the commits.
		commitOrder := pluginv1beta1.ListCommitsRequest_ORDER_CREATE_TIME_DESC
		if flags.Reverse {
			commitOrder = pluginv1beta1.ListCommitsRequest_ORDER_CREATE_TIME_ASC
		}
		resp, err := commitServiceClient.ListCommits(
			ctx,
			connect.NewRequest(
				&pluginv1beta1.ListCommitsRequest{
					PageSize:  flags.PageSize,
					PageToken: flags.PageToken,
					ResourceRef: &pluginv1beta1.ResourceRef{
						Value: &pluginv1beta1.ResourceRef_Name_{
							Name: &pluginv1beta1.ResourceRef_Name{
								Owner:  pluginRef.FullName().Owner(),
								Plugin: pluginRef.FullName().Name(),
							},
						},
					},
					Order: commitOrder,
				},
			),
		)
		if err != nil {
			if connect.CodeOf(err) == connect.CodeNotFound {
				return bufcli.NewRefNotFoundError(pluginRef)
			}
			return err
		}
		return bufprint.PrintPage(
			container.Stdout(),
			format,
			resp.Msg.NextPageToken,
			nextPageCommand(container, flags, resp.Msg.NextPageToken),
			xslices.Map(resp.Msg.Commits, func(commit *pluginv1beta1.Commit) bufprint.Entity {
				return bufprint.NewCommitEntity(commit, pluginRef.FullName(), commit.GetSourceControlUrl())
			}),
		)
	}
	label := resource.GetLabel()
	if label == nil {
		// This should be impossible because getLabelOrCommitForRef would've returned an error.
		return syserror.Newf("%s is neither a commit nor a label", pluginRef.String())
	}
	// The ref is a label. Call ListLabelHistory to get all commits.
	labelHistoryOrder := pluginv1beta1.ListLabelHistoryRequest_ORDER_DESC
	if flags.Reverse {
		labelHistoryOrder = pluginv1beta1.ListLabelHistoryRequest_ORDER_ASC
	}
	resp, err := labelServiceClient.ListLabelHistory(
		ctx,
		connect.NewRequest(
			&pluginv1beta1.ListLabelHistoryRequest{
				PageSize:  flags.PageSize,
				PageToken: flags.PageToken,
				LabelRef: &pluginv1beta1.LabelRef{
					Value: &pluginv1beta1.LabelRef_Name_{
						Name: &pluginv1beta1.LabelRef_Name{
							Owner:  pluginRef.FullName().Owner(),
							Plugin: pluginRef.FullName().Name(),
							Label:  pluginRef.Ref(),
						},
					},
				},
				Order:                         labelHistoryOrder,
				OnlyCommitsWithChangedDigests: flags.DigestChangesOnly,
			},
		),
	)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			// This should be impossible since we just checked that the ref is a label.
			return bufcli.NewRefNotFoundError(pluginRef)
		}
		return err
	}
	commits := xslices.Map(
		resp.Msg.Values,
		func(value *pluginv1beta1.ListLabelHistoryResponse_Value) *pluginv1beta1.Commit {
			return value.Commit
		},
	)
	return bufprint.PrintPage(
		container.Stdout(),
		format,
		resp.Msg.NextPageToken,
		nextPageCommand(container, flags, resp.Msg.NextPageToken),
		xslices.Map(commits, func(commit *pluginv1beta1.Commit) bufprint.Entity {
			return bufprint.NewCommitEntity(commit, pluginRef.FullName(), commit.GetSourceControlUrl())
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
