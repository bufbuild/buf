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

package modupdate

import (
	"context"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/mod/internal"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/spf13/pflag"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

const (
	onlyFlagName = "only"
)

// NewCommand returns a new update Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <directory>",
		Short: "Update a module's locked dependencies in buf.lock",
		Long: `Fetch the latest digests for the specified references in buf.yaml,
and write them and their transitive dependencies to buf.lock.

The first argument is the directory of the local module to update.
Defaults to "." if no argument is specified.

Note that updating is only allowed for v2 buf.yaml files. Run "buf migrate" to migrate to v2.`,
		Args: appcmd.MaximumNArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	Only []string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringSliceVar(
		&f.Only,
		onlyFlagName,
		nil,
		"The name of the dependency to update. When set, only this dependency and its transitive dependencies are updated. May be passed multiple times",
	)
	// TODO: implement
	_ = flagSet.MarkHidden(onlyFlagName)
}

// run update the buf.lock file for a specific module.
func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) (retErr error) {
	dirPath := "."
	if container.NumArgs() > 0 {
		dirPath = container.Arg(0)
	}
	if len(flags.Only) > 0 {
		// TODO: implement
		return syserror.Newf("--%s is not yet implemented", onlyFlagName)
	}

	logger := container.Logger()
	controller, err := bufcli.NewController(container)
	if err != nil {
		return err
	}
	moduleKeyProvider, err := bufcli.NewModuleKeyProvider(container)
	if err != nil {
		return err
	}
	graphProvider, err := bufcli.NewGraphProvider(container)
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
	// Get all the configured deps.
	configuredDepModuleKeys, err := moduleKeyProvider.GetModuleKeysForModuleRefs(
		ctx,
		configuredDepModuleRefs,
		workspaceDepManager.BufLockFileDigestType(),
	)
	if err != nil {
		return err
	}
	logger.Debug(
		"deps from buf.yaml",
		zap.Strings("deps", slicesext.Map(configuredDepModuleKeys, bufmodule.ModuleKey.String)),
	)
	// Walk the graph to get all configured deps and their transitive dependencies.
	graph, err := graphProvider.GetGraphForModuleKeys(ctx, configuredDepModuleKeys)
	if err != nil {
		return err
	}
	var newDepModuleKeys []bufmodule.ModuleKey
	if err := graph.WalkNodes(
		func(depModuleKey bufmodule.ModuleKey, _ []bufmodule.ModuleKey, _ []bufmodule.ModuleKey) error {
			newDepModuleKeys = append(newDepModuleKeys, depModuleKey)
			return nil
		},
	); err != nil {
		return err
	}
	logger.Debug(
		"all deps",
		zap.Strings("deps", slicesext.Map(newDepModuleKeys, bufmodule.ModuleKey.String)),
	)

	// Store the existing buf.lock data.
	existingDepModuleKeys, err := workspaceDepManager.ExistingBufLockFileDepModuleKeys(ctx)
	if err != nil {
		return err
	}
	// We're about to edit the buf.lock file on disk. If we have a subsequent error,
	// attempt to revert the buf.lock file.
	defer func() {
		if retErr != nil {
			retErr = multierr.Append(retErr, workspaceDepManager.UpdateBufLockFile(ctx, existingDepModuleKeys))
		}
	}()
	// Edit the buf.lock file with the unpruned dependencies.
	if err := workspaceDepManager.UpdateBufLockFile(ctx, newDepModuleKeys); err != nil {
		return err
	}
	// Prune the buf.lock. This also verifies the workspace builds again.
	// Building also has the side effect of doing tamper-proofing.
	return internal.Prune(ctx, logger, controller, workspaceDepManager, dirPath)
}
