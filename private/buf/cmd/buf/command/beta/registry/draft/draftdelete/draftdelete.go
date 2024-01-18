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
	"strings"

	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const forceFlagName = "force"

// NewCommand returns a new Command
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <buf.build/owner/repository:draft>",
		Short: "Delete a repository draft",
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
	container appflag.Container,
	flags *flags,
) error {
	bufcli.WarnBetaCommand(ctx, container)
	moduleReference, err := bufmoduleref.ModuleReferenceForString(container.Arg(0))
	if err != nil {
		return appcmd.NewInvalidArgumentError(err.Error())
	}
	if moduleReference.Reference() == bufmoduleref.Main {
		// bufmoduleref.ModuleReferenceForString will give a default reference when user did not specify one
		// we need to check the origin input and return different errors for different cases.
		if strings.HasSuffix(container.Arg(0), ":"+bufmoduleref.Main) {
			return appcmd.NewInvalidArgumentErrorf("%q is not a valid draft name", bufmoduleref.Main)
		}
		return appcmd.NewInvalidArgumentError("a valid draft name need to be specified")
	}
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	service := connectclient.Make(
		clientConfig,
		moduleReference.Remote(),
		registryv1alpha1connect.NewRepositoryCommitServiceClient,
	)
	if !flags.Force {
		if err := bufcli.PromptUserForDelete(
			container,
			"draft",
			moduleReference.Reference(),
		); err != nil {
			return err
		}
	}
	if _, err := service.DeleteRepositoryDraftCommit(
		ctx,
		connect.NewRequest(&registryv1alpha1.DeleteRepositoryDraftCommitRequest{
			RepositoryOwner: moduleReference.Owner(),
			RepositoryName:  moduleReference.Repository(),
			DraftName:       moduleReference.Reference(),
		}),
	); err != nil {
		// not explicitly handling error with connect.CodeNotFound as it can be repository not found or draft not found.
		return err
	}
	if _, err := fmt.Fprintln(container.Stdout(), "Draft deleted."); err != nil {
		return bufcli.NewInternalError(err)
	}
	return nil
}
