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

package branchcreate

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufprint"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/connect-go"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	formatFlagName      = "format"
	parentFlagName      = "parent"
	parentFlagShortName = "p"
)

// NewCommand returns a new Command
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <buf.build/owner/repository:branch>",
		Short: "Creates a branch for the specified repository.",
		Args:  cobra.ExactArgs(1),
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
	Parent string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(
		&f.Format,
		formatFlagName,
		bufprint.FormatText.String(),
		fmt.Sprintf(`The output format to use. Must be one of %s.`, bufprint.AllFormatsString),
	)
	flagSet.StringVarP(
		&f.Parent,
		parentFlagName,
		parentFlagShortName,
		bufmoduleref.MainBranch,
		`The parent branch.`,
	)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) error {
	bufcli.WarnBetaCommand(ctx, container)
	if flags.Parent == "" {
		return appcmd.NewInvalidArgumentErrorf("required flag %q not set", parentFlagName)
	}
	moduleReference, err := bufmoduleref.ModuleReferenceForString(
		container.Arg(0),
	)
	if err != nil {
		return appcmd.NewInvalidArgumentError(err.Error())
	}
	if bufmoduleref.IsCommitModuleReference(moduleReference) {
		return fmt.Errorf("branch is required but commit was given: %q", container.Arg(0))
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
	repositoryBranchService, err := apiProvider.NewRepositoryBranchService(ctx, moduleReference.Remote())
	if err != nil {
		return err
	}
	// TODO: We can add another RPC for creating a repository branch by name so that we don't
	// have to get the repository separately.
	repository, _, err := repositoryService.GetRepositoryByFullName(ctx, moduleReference.Owner()+"/"+moduleReference.Repository())
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return bufcli.NewRepositoryNotFoundError(moduleReference.Remote() + "/" + moduleReference.Owner() + "/" + moduleReference.Repository())
		}
		return err
	}
	repositoryBranch, err := repositoryBranchService.CreateRepositoryBranch(ctx, repository.Id, moduleReference.Reference(), flags.Parent)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeAlreadyExists {
			return bufcli.NewBranchOrTagNameAlreadyExistsError(moduleReference.String())
		}
		return err
	}
	return bufprint.NewRepositoryBranchPrinter(container.Stdout()).PrintRepositoryBranch(ctx, format, repositoryBranch)
}
