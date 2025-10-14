// Copyright 2020-2025 Buf Technologies, Inc.
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

package pluginprune

import (
	"context"

	"buf.build/go/app/appcmd"
	"buf.build/go/app/appext"
	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufworkspace"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	return &appcmd.Command{
		Use:   name + " <directory>",
		Short: "Prune unused plugins from buf.lock",
		Long: `Plugins that are no longer configured in buf.yaml are removed from the buf.lock file.

The first argument is the directory of your buf.yaml configuration file.
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
	workspaceDepManager, err := controller.GetWorkspaceDepManager(ctx, dirPath)
	if err != nil {
		return err
	}
	configuredRemotePluginRefs, err := workspaceDepManager.ConfiguredRemotePluginRefs(ctx)
	if err != nil {
		return err
	}
	return prune(
		ctx,
		xslices.Map(
			configuredRemotePluginRefs,
			func(pluginRef bufparse.Ref) string {
				return pluginRef.FullName().String()
			},
		),
		workspaceDepManager,
	)
}

func prune(
	ctx context.Context,
	bufYAMLBasedRemotePluginNames []string,
	workspaceDepManager bufworkspace.WorkspaceDepManager,
) error {
	bufYAMLRemotePluginNames := xslices.ToStructMap(bufYAMLBasedRemotePluginNames)
	existingRemotePluginKeys, err := workspaceDepManager.ExistingBufLockFileRemotePluginKeys(ctx)
	if err != nil {
		return err
	}
	var prunedBufLockPluginKeys []bufplugin.PluginKey
	for _, existingRemotePluginKey := range existingRemotePluginKeys {
		// Check if an existing plugin key from the buf.lock is configured in the buf.yaml.
		if _, ok := bufYAMLRemotePluginNames[existingRemotePluginKey.FullName().String()]; ok {
			// If yes, then we keep it for the updated buf.lock.
			prunedBufLockPluginKeys = append(prunedBufLockPluginKeys, existingRemotePluginKey)
		}
	}
	// We keep the existing dep module keys as-is.
	existingDepModuleKeys, err := workspaceDepManager.ExistingBufLockFileDepModuleKeys(ctx)
	if err != nil {
		return err
	}
	existingRemotePolicyKeys, err := workspaceDepManager.ExistingBufLockFileRemotePolicyKeys(ctx)
	if err != nil {
		return err
	}
	existingPolicyNameToRemotePluginKeys, err := workspaceDepManager.ExistingBufLockFilePolicyNameToRemotePluginKeys(ctx)
	if err != nil {
		return err
	}
	return workspaceDepManager.UpdateBufLockFile(ctx, existingDepModuleKeys, prunedBufLockPluginKeys, existingRemotePolicyKeys, existingPolicyNameToRemotePluginKeys)
}
