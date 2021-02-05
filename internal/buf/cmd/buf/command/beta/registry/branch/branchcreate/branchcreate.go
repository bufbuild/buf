// Copyright 2020-2021 Buf Technologies, Inc.
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

	"github.com/bufbuild/buf/internal/buf/bufcli"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/buf/bufprint"
	"github.com/bufbuild/buf/internal/pkg/app/appcmd"
	"github.com/bufbuild/buf/internal/pkg/app/appflag"
	"github.com/bufbuild/buf/internal/pkg/rpc"
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
		Short: "Create a branch for the specified repository.",
		Args:  cobra.ExactArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appflag.Container) error {
				return run(ctx, container, flags)
			},
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
		bufmodule.MainBranch,
		`The parent branch.`,
	)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) error {
	if flags.Parent == "" {
		return bufcli.NewFlagIsRequiredError(parentFlagName)
	}
	moduleReference, err := bufmodule.BranchModuleReferenceForString(
		container.Arg(0),
		bufmodule.BranchModuleReferenceForStringRequireBranch(),
	)
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
	ctx, err = bufcli.WithHeaders(ctx, container, moduleReference.Remote())
	if err != nil {
		return err
	}
	// TODO: We can add another RPC for creating a repository branch by name so that we don't
	// have to get the repository separately.
	repository, err := repositoryService.GetRepositoryByFullName(ctx, moduleReference.Owner()+"/"+moduleReference.Repository())
	if err != nil {
		if rpc.GetErrorCode(err) == rpc.ErrorCodeNotFound {
			return bufcli.NewRepositoryNotFoundError(moduleReference.Remote() + "/" + moduleReference.Owner() + "/" + moduleReference.Repository())
		}
		return bufcli.NewRPCError("get repository", moduleReference.Remote(), err)
	}
	repositoryBranch, err := repositoryBranchService.CreateRepositoryBranch(ctx, repository.Id, moduleReference.Branch(), flags.Parent)
	if err != nil {
		if rpc.GetErrorCode(err) == rpc.ErrorCodeAlreadyExists {
			return bufcli.NewBranchNameAlreadyExistsError(moduleReference.String())
		}
		return bufcli.NewRPCError("create repository branch", moduleReference.Remote(), err)
	}
	return bufcli.PrintRepositoryBranches(ctx, container.Stdout(), flags.Format, repositoryBranch)
}
