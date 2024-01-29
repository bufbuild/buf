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

package modprune

import (
	"context"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/slicesext"
)

// NewCommand returns a new prune Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	return &appcmd.Command{
		Use:   name + " <directory>",
		Short: "Prune unused dependencies from buf.lock",
		Long: `The first argument is the directory of your buf.yaml configuration file.
Defaults to "." if no argument is specified.`,
		Args: appcmd.MaximumNArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container)
			},
		),
	}
}

func run(
	ctx context.Context,
	container appext.Container,
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
	depModules, err := bufmodule.RemoteDepsForModuleSet(updateableWorkspace)
	if err != nil {
		return err
	}
	depModuleKeys, err := slicesext.MapError(
		depModules,
		func(remoteDep bufmodule.RemoteDep) (bufmodule.ModuleKey, error) {
			return bufmodule.ModuleToModuleKey(remoteDep, updateableWorkspace.BufLockFileDigestType())
		},
	)
	if err != nil {
		return err
	}
	return updateableWorkspace.UpdateBufLockFile(ctx, depModuleKeys)
}
