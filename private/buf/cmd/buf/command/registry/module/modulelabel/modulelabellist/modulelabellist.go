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

package modulelabellist

import (
	"context"
	"fmt"

	modulev1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1"
	"buf.build/go/app/appcmd"
	"buf.build/go/app/appext"
	"buf.build/go/standard/xslices"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufprint"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/bufpkg/bufregistryapi/bufregistryapimodule"
	"github.com/spf13/pflag"
)

const (
	archiveStatusName = "archive-status"
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
	deprecated string,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:        name + " <remote/owner/module[:ref]>",
		Short:      "List module labels",
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
	ArchiveStatus string
	PageSize      uint32
	PageToken     string
	Reverse       bool
	Format        string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	bufcli.BindArchiveStatus(flagSet, &f.ArchiveStatus, archiveStatusName)
	flagSet.Uint32Var(
		&f.PageSize,
		pageSizeFlagName,
		defaultPageSize,
		`The page size.`,
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
	moduleRef, err := bufparse.ParseRef(container.Arg(0))
	if err != nil {
		return appcmd.WrapInvalidArgumentError(err)
	}
	archiveStatusFilter, err := bufcli.ArchiveStatusFlagToModuleArchiveStatusFilter(flags.ArchiveStatus)
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
	moduleClientProvider := bufregistryapimodule.NewClientProvider(clientConfig)
	moduleFullName := moduleRef.FullName()
	labelServiceClient := moduleClientProvider.V1LabelServiceClient(moduleFullName.Registry())
	order := modulev1.ListLabelsRequest_ORDER_UPDATE_TIME_DESC
	if flags.Reverse {
		order = modulev1.ListLabelsRequest_ORDER_UPDATE_TIME_ASC
	}
	resp, err := labelServiceClient.ListLabels(
		ctx,
		connect.NewRequest(
			&modulev1.ListLabelsRequest{
				PageSize:  flags.PageSize,
				PageToken: flags.PageToken,
				ResourceRef: &modulev1.ResourceRef{
					Value: &modulev1.ResourceRef_Name_{
						Name: &modulev1.ResourceRef_Name{
							Owner:  moduleFullName.Owner(),
							Module: moduleFullName.Name(),
							Child: &modulev1.ResourceRef_Name_Ref{
								Ref: moduleRef.Ref(),
							},
						},
					},
				},
				Order:         order,
				ArchiveFilter: archiveStatusFilter,
			},
		),
	)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return bufcli.NewRefNotFoundError(moduleRef)
		}
		return err
	}
	return bufprint.PrintPage(
		container.Stdout(),
		format,
		resp.Msg.NextPageToken,
		nextPageCommand(container, flags, resp.Msg.NextPageToken),
		xslices.Map(resp.Msg.Labels, func(label *modulev1.Label) bufprint.Entity {
			return bufprint.NewLabelEntity(label, moduleFullName)
		}),
	)
}

func nextPageCommand(container appext.Container, flags *flags, nextPageToken string) string {
	if nextPageToken == "" {
		return ""
	}
	command := fmt.Sprintf("buf registry module label list %s", container.Arg(0))
	if flags.ArchiveStatus != bufcli.DefaultArchiveStatus {
		command = fmt.Sprintf("%s --%s %s", command, archiveStatusName, flags.ArchiveStatus)
	}
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
