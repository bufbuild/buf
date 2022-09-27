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

package moduleundeprecate

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/connect-go"
	"github.com/spf13/cobra"
)

// NewCommand returns a new Command
func NewCommand(cmdName string, builder appflag.Builder) *appcmd.Command {
	return &appcmd.Command{
		Use:   cmdName + " <buf.build/owner/name>",
		Short: "Undeprecate a BSR module.",
		Args:  cobra.ExactArgs(1),
		Run:   builder.NewRunFunc(run, bufcli.NewErrorInterceptor()),
	}
}

func run(ctx context.Context, container appflag.Container) error {
	bufcli.WarnBetaCommand(ctx, container)
	moduleIdentity, err := bufmoduleref.ModuleIdentityForString(container.Arg(0))
	if err != nil {
		return appcmd.NewInvalidArgumentError(err.Error())
	}
	apiProvider, err := bufcli.NewRegistryProvider(ctx, container)
	if err != nil {
		return err
	}
	service, err := apiProvider.NewRepositoryService(ctx, moduleIdentity.Remote())
	if err != nil {
		return err
	}
	if _, err = service.UndeprecateRepositoryByName(
		ctx,
		moduleIdentity.Owner(),
		moduleIdentity.Repository(),
	); err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return bufcli.NewRepositoryNotFoundError(container.Arg(0))
		}
		return err
	}
	if _, err := fmt.Fprintln(container.Stdout(), "Module undeprecated."); err != nil {
		return bufcli.NewInternalError(err)
	}
	return nil
}
