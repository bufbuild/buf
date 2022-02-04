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

package trackappend

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

const formatFlagName = "format"

// NewCommand returns a new Command
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <buf.build/owner/repository:track> <commit>",
		Short: "Append commit to a track",
		Args:  cobra.ExactArgs(2),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appflag.Container) error {
				return run(ctx, container, flags)
			},
			bufcli.NewErrorInterceptor(),
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	Format string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
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
	moduleReference, err := bufmoduleref.ModuleReferenceForString(container.Arg(0))
	if err != nil {
		return appcmd.NewInvalidArgumentError(err.Error())
	}
	commitReference := container.Arg(1)
	if err := bufmoduleref.ValidateReference(commitReference); err != nil {
		return appcmd.NewInvalidArgumentError(err.Error())
	}
	format, err := bufprint.ParseFormat(flags.Format)
	if err != nil {
		return appcmd.NewInvalidArgumentError(err.Error())
	}
	registryProvider, err := bufcli.NewRegistryProvider(ctx, container)
	if err != nil {
		return err
	}
	repositoryTrackService, err := registryProvider.NewRepositoryTrackService(ctx, moduleReference.Remote())
	if err != nil {
		return err
	}
	repositoryTrack, err := repositoryTrackService.GetRepositoryTrackByName(
		ctx,
		moduleReference.Owner(),
		moduleReference.Repository(),
		moduleReference.Reference(),
	)
	if err != nil {
		if rpc.GetErrorCode(err) == rpc.ErrorCodeNotFound {
			return bufcli.NewTrackNotFoundError(moduleReference.String())
		}
		return err
	}
	repositoryCommitService, err := registryProvider.NewRepositoryCommitService(ctx, moduleReference.Remote())
	if err != nil {
		return err
	}
	repositoryCommit, err := repositoryCommitService.GetRepositoryCommitByReference(
		ctx,
		moduleReference.Owner(),
		moduleReference.Repository(),
		commitReference,
	)
	if err != nil {
		reference, refErr := bufmoduleref.NewModuleReference(
			moduleReference.Remote(),
			moduleReference.Owner(),
			moduleReference.Repository(),
			commitReference,
		)
		// This should not be possible because all arguments have already been validated.
		if refErr != nil {
			return err
		}
		if rpc.GetErrorCode(err) == rpc.ErrorCodeNotFound {
			return bufcli.NewModuleReferenceNotFoundError(reference)
		}
		return err
	}
	repositoryTrackCommitService, err := registryProvider.NewRepositoryTrackCommitService(ctx, moduleReference.Remote())
	if err != nil {
		return err
	}
	repositoryTrackCommit, err := repositoryTrackCommitService.CreateRepositoryTrackCommit(
		ctx,
		repositoryTrack.Id,
		repositoryCommit.Id,
	)
	if err != nil {
		return err
	}
	return bufprint.NewRepositoryTrackCommitPrinter(container.Stdout()).PrintRepositoryTrackCommit(
		format,
		repositoryTrackCommit,
	)
}
