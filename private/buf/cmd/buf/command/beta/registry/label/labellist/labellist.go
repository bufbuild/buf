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

package labellist

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
	"github.com/spf13/pflag"
)

const (
	archiveStatusName = "archive-status"
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
		Short: "List repository labels",
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
		10,
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
	bufcli.WarnBetaCommand(ctx, container)
	moduleRef, err := bufmodule.ParseModuleRef(container.Arg(0))
	if err != nil {
		return appcmd.NewInvalidArgumentError(err.Error())
	}
	archiveStatusFitler, err := bufcli.ArchiveStatusFlagToArchiveStatusFilter(flags.ArchiveStatus)
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
	labelServiceClient := clientProvider.V1LabelServiceClient(moduleRef.ModuleFullName().Registry())
	order := modulev1.ListLabelsRequest_ORDER_CREATE_TIME_ASC
	if flags.Reverse {
		order = modulev1.ListLabelsRequest_ORDER_CREATE_TIME_DESC
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
							Owner:  moduleRef.ModuleFullName().Owner(),
							Module: moduleRef.ModuleFullName().Name(),
							Child: &modulev1.ResourceRef_Name_Ref{
								Ref: moduleRef.Ref(),
							},
						},
					},
				},
				Order:         order,
				ArchiveFilter: archiveStatusFitler,
			},
		),
	)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return bufcli.NewModuleRefNotFoundError(moduleRef)
		}
		return err
	}
	return bufprint.NewRepositoryLabelPrinter(container.Stdout()).PrintRepositoryLabels(ctx, format, resp.Msg.NextPageToken, resp.Msg.Labels...)
}
