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

package depprune

import (
	"context"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/dep/internal"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
)

// NewCommand returns a new prune Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
	deprecated string,
	hidden bool,
) *appcmd.Command {
	return &appcmd.Command{
		Use:   name + " <directory>",
		Short: "Prune unused dependencies from a buf.lock",
		Long: `The first argument is the directory of your buf.yaml configuration file.
Defaults to "." if no argument is specified.`,
		Args:       appcmd.MaximumNArgs(1),
		Deprecated: deprecated,
		Hidden:     hidden,
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
	workspaceDepManager, err := controller.GetWorkspaceDepManager(ctx, dirPath)
	if err != nil {
		return err
	}
	configuredDepModuleRefs, err := workspaceDepManager.ConfiguredDepModuleRefs(ctx)
	if err != nil {
		return err
	}
	configuredDepModuleKeys, err := internal.ModuleKeysAndTransitiveDepModuleKeysForModuleRefs(
		ctx,
		container,
		configuredDepModuleRefs,
		workspaceDepManager.BufLockFileDigestType(),
	)
	if err != nil {
		return err
	}
	return internal.Prune(
		ctx,
		container.Logger(),
		controller,
		configuredDepModuleKeys,
		workspaceDepManager,
		dirPath,
	)
}
