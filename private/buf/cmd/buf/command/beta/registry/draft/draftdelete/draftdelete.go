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

package draftdelete

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/spf13/pflag"
)

const forceFlagName = "force"

// NewCommand returns a new Command
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <buf.build/owner/repository:draft>",
		Short: "Delete a repository draft",
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
	Force bool
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.BoolVar(
		&f.Force,
		forceFlagName,
		false,
		"Force deletion without confirming. Use with caution",
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
	if moduleRef.Ref() == "" {
		return appcmd.NewInvalidArgumentError("a valid draft name need to be specified")
	}
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	service := connectclient.Make(
		clientConfig,
		moduleRef.ModuleFullName().Registry(),
		registryv1alpha1connect.NewRepositoryCommitServiceClient,
	)
	if !flags.Force {
		if err := bufcli.PromptUserForDelete(
			container,
			"draft",
			moduleRef.Ref(),
		); err != nil {
			return err
		}
	}
	if _, err := service.DeleteRepositoryDraftCommit(
		ctx,
		connect.NewRequest(
			&registryv1alpha1.DeleteRepositoryDraftCommitRequest{
				RepositoryOwner: moduleRef.ModuleFullName().Owner(),
				RepositoryName:  moduleRef.ModuleFullName().Name(),
				DraftName:       moduleRef.Ref(),
			},
		),
	); err != nil {
		// not explicitly handling error with connect.CodeNotFound as it can be repository not found or draft not found.
		return err
	}
	if _, err := fmt.Fprintln(container.Stdout(), "Draft deleted."); err != nil {
		return syserror.Wrap(err)
	}
	return nil
}
