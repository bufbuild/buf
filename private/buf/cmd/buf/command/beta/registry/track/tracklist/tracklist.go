// Copyright 2020-2022 Buf Technologies, Inc.
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

package tracklist

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufprint"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/rpc"
	"github.com/spf13/cobra"
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
	builder appflag.Builder,
) *appcmd.Command {
	flgs := newFlags()
	return &appcmd.Command{
		Use:   name + " <buf.build/owner/repository>",
		Short: "List tracks for the specified repository",
		Args:  cobra.ExactArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appflag.Container) error {
				return run(ctx, container, flgs)
			},
			bufcli.NewErrorInterceptor(),
		),
		BindFlags: flgs.Bind,
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
		`The page token. If more results are available, a "next_page" key is present in the --format=json output.`,
	)
	flagSet.BoolVar(
		&f.Reverse,
		reverseFlagName,
		false,
		`Reverse the results.`,
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
	container appflag.Container,
	flags *flags,
) error {
	bufcli.WarnBetaCommand(ctx, container)
	if container.Arg(0) == "" {
		return appcmd.NewInvalidArgumentError("repository is required")
	}
	moduleReference, err := bufmoduleref.ModuleReferenceForString(container.Arg(0))
	if err != nil {
		return appcmd.NewInvalidArgumentError(err.Error())
	}
	format, err := bufprint.ParseFormat(flags.Format)
	if err != nil {
		return appcmd.NewInvalidArgumentError(err.Error())
	}

	apiProvider, err := bufcli.NewRegistryProvider(ctx, container)
	if err != nil {
		return err
	}
	repositoryService, err := apiProvider.NewRepositoryService(ctx, moduleReference.Remote())
	if err != nil {
		return err
	}
	repository, _, err := repositoryService.GetRepositoryByFullName(
		ctx,
		moduleReference.Owner()+"/"+moduleReference.Repository(),
	)
	if err != nil {
		if rpc.GetErrorCode(err) == rpc.ErrorCodeNotFound {
			return bufcli.NewRepositoryNotFoundError(container.Arg(0))
		}
		return err
	}
	repositoryTrackService, err := apiProvider.NewRepositoryTrackService(ctx, moduleReference.Remote())
	if err != nil {
		return err
	}
	repositoryTracks, nextPageToken, err := repositoryTrackService.ListRepositoryTracks(
		ctx,
		repository.Id,
		flags.PageSize,
		flags.PageToken,
		flags.Reverse,
	)
	if err != nil {
		return err
	}
	return bufprint.NewRepositoryTrackPrinter(container.Stdout()).PrintRepositoryTracks(
		format,
		nextPageToken,
		repositoryTracks...,
	)
}
