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

package repositorylist

import (
	"context"
	"fmt"

	modulev1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufprint"
	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/netext"
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
		Use:   name + " <buf.build>",
		Short: "List BSR repositories",
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
		`The page size.`,
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
	registryHostname := container.Arg(0)
	if _, err := netext.ValidateHostname(registryHostname); err != nil {
		return err
	}
	format, err := bufprint.ParseFormat(flags.Format)
	if err != nil {
		return appcmd.NewInvalidArgumentError(err.Error())
	}

	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	moduleServiceClient := bufapi.NewClientProvider(clientConfig).V1ModuleServiceClient(registryHostname)
	order := modulev1.ListModulesRequest_ORDER_CREATE_TIME_ASC
	if flags.Reverse {
		order = modulev1.ListModulesRequest_ORDER_CREATE_TIME_DESC
	}
	resp, err := moduleServiceClient.ListModules(
		ctx,
		&connect.Request[modulev1.ListModulesRequest]{
			Msg: &modulev1.ListModulesRequest{
				PageSize:  flags.PageSize,
				PageToken: flags.PageToken,
				Order:     order,
			},
		},
	)
	if err != nil {
		return err
	}
	repositories, nextPageToken := resp.Msg.Modules, resp.Msg.NextPageToken
	return bufprint.NewRepositoryPrinter(
		clientConfig,
		registryHostname,
		container.Stdout(),
	).PrintRepositories(ctx, format, nextPageToken, repositories...)
}
