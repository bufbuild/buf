// Copyright 2020-2023 Buf Technologies, Inc.
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

package modprune

import (
	"context"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/spf13/cobra"
)

// NewCommand returns a new prune Command.
func NewCommand(
	name string,
	builder appflag.SubCommandBuilder,
) *appcmd.Command {
	return &appcmd.Command{
		Use:   name + " <directory>",
		Short: "Prune unused dependencies from buf.lock",
		Long: `The first argument is the directory of your buf.yaml configuration file.
Defaults to "." if no argument is specified.

Note that pruning is only allowed for v2 buf.yaml files. Run "buf migrate" to migrate to v2.`,
		Args: cobra.MaximumNArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appflag.Container) error {
				return run(ctx, container)
			},
		),
	}
}

func run(
	ctx context.Context,
	container appflag.Container,
) error {
	dirPath := "."
	if container.NumArgs() > 0 {
		dirPath = container.Arg(0)
	}
	controller, err := bufcli.NewController(container)
	if err != nil {
		return err
	}
	updateableWorkspace, err := controller.GetUpdateableWorkspace(ctx, dirPath)
	if err != nil {
		return err
	}
	depModules, err := bufmodule.ModuleSetRemoteDepsOfLocalModules(updateableWorkspace)
	if err != nil {
		return err
	}
	depModuleKeys, err := slicesext.MapError(depModules, bufmodule.ModuleToModuleKey)
	if err != nil {
		return err
	}
	bufLockFile, err := bufconfig.NewBufLockFile(bufconfig.FileVersionV2, depModuleKeys)
	if err != nil {
		return err
	}
	return updateableWorkspace.PutBufLockFile(ctx, bufLockFile)
}
