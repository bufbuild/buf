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

package policyupdate

import (
	"context"
	"errors"
	"fmt"
	"maps"

	"buf.build/go/app/appcmd"
	"buf.build/go/app/appext"
	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin"
	"github.com/bufbuild/buf/private/bufpkg/bufpolicy"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/spf13/pflag"
)

const (
	onlyFlagName = "only"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <directory>",
		Short: "Update pinned remote policies in a buf.lock",
		Long: `Fetch the latest digests for the specified policy references in buf.yaml.

The first argument is the directory of the local module to update.
Defaults to "." if no argument is specified.`,
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
		"The name of the policy to update. When set, only this policy is updated. May be provided multiple times",
	)
	// TODO FUTURE: implement
	_ = flagSet.MarkHidden(onlyFlagName)
}

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
		// TODO FUTURE: implement
		return syserror.Newf("--%s is not implemented", onlyFlagName)
	}

	logger := container.Logger()
	controller, err := bufcli.NewController(container)
	if err != nil {
		return err
	}
	workspaceDepManager, err := controller.GetWorkspaceDepManager(ctx, dirPath)
	if err != nil {
		return err
	}
	configuredRemotePolicyRefs, err := workspaceDepManager.ConfiguredRemotePolicyRefs(ctx)
	if err != nil {
		return err
	}
	configuredLocalPolicyNameToRemotePluginRefs, err := workspaceDepManager.ConfiguredLocalPolicyNameToRemotePluginRefs(ctx)
	if err != nil {
		return err
	}
	policyKeyProvider, err := bufcli.NewPolicyKeyProvider(container)
	if err != nil {
		return err
	}
	configuredRemotePolicyKeys, err := policyKeyProvider.GetPolicyKeysForPolicyRefs(
		ctx,
		configuredRemotePolicyRefs,
		bufpolicy.DigestTypeO1,
	)
	if err != nil {
		return err
	}
	configuredRemotePolicyNameToRemotePluginKeys, err := getPolicyKeyPluginKeysForPolicyKeys(
		ctx,
		container,
		configuredRemotePolicyKeys,
	)
	if err != nil {
		return err
	}
	configuredLocalPolicyNameToRemotePluginKeys, err := getPolicyPluginKeysForPolicyNames(
		ctx,
		container,
		configuredLocalPolicyNameToRemotePluginRefs,
	)
	if err != nil {
		return err
	}
	var configuredPolicyNameToRemotePluginKeys map[string][]bufplugin.PluginKey
	if policyCount := len(configuredRemotePolicyNameToRemotePluginKeys) + len(configuredLocalPolicyNameToRemotePluginKeys); policyCount > 0 {
		configuredPolicyNameToRemotePluginKeys = make(map[string][]bufplugin.PluginKey, policyCount)
		maps.Copy(configuredPolicyNameToRemotePluginKeys, configuredRemotePolicyNameToRemotePluginKeys)
		maps.Copy(configuredPolicyNameToRemotePluginKeys, configuredLocalPolicyNameToRemotePluginKeys)
	}

	// Store the existing buf.lock data.
	existingRemotePolicyKeys, err := workspaceDepManager.ExistingBufLockFileRemotePolicyKeys(ctx)
	if err != nil {
		return err
	}
	existingPolicyNameToRemotePluginKeys, err := workspaceDepManager.ExistingBufLockFilePolicyNameToRemotePluginKeys(ctx)
	if err != nil {
		return err
	}
	if len(configuredRemotePolicyKeys) == 0 && len(configuredPolicyNameToRemotePluginKeys) == 0 &&
		len(existingRemotePolicyKeys) == 0 && len(existingPolicyNameToRemotePluginKeys) == 0 {
		// No new configured remote plugins were found, and no existing buf.lock deps were found, so there
		// is nothing to update, we can return here.
		// This ensures we do not create an empty buf.lock when one did not exist in the first
		// place and we do not need to go through the entire operation of updating non-existent
		// deps and building the image for tamper-proofing.
		logger.Warn(fmt.Sprintf("No configured remote policies were found to update in %q.", dirPath))
		return nil
	}
	existingDepModuleKeys, err := workspaceDepManager.ExistingBufLockFileDepModuleKeys(ctx)
	if err != nil {
		return err
	}
	existingRemotePluginKeys, err := workspaceDepManager.ExistingBufLockFileRemotePluginKeys(ctx)
	if err != nil {
		return err
	}
	// We're about to edit the buf.lock file on disk. If we have a subsequent error,
	// attempt to revert the buf.lock file.
	//
	// TODO FUTURE: We should be able to update the buf.lock file in an in-memory bucket, then do the rebuild,
	// and if the rebuild is successful, then actually write to disk. It shouldn't even be that much work - just
	// overlay the new buf.lock file in a union bucket.
	defer func() {
		if retErr != nil {
			retErr = errors.Join(retErr, workspaceDepManager.UpdateBufLockFile(
				ctx, existingDepModuleKeys, existingRemotePluginKeys, existingRemotePolicyKeys, existingPolicyNameToRemotePluginKeys,
			))
		}
	}()
	// Edit the buf.lock file with the updated remote plugins.
	if err := workspaceDepManager.UpdateBufLockFile(ctx, existingDepModuleKeys, existingRemotePluginKeys, configuredRemotePolicyKeys, configuredPolicyNameToRemotePluginKeys); err != nil {
		return err
	}
	return nil
}

func getPolicyKeyPluginKeysForPolicyKeys(
	ctx context.Context,
	container appext.Container,
	policyKeys []bufpolicy.PolicyKey,
) (map[string][]bufplugin.PluginKey, error) {
	if len(policyKeys) == 0 {
		return nil, nil
	}
	policyDataProvider, err := bufcli.NewPolicyDataProvider(container)
	if err != nil {
		return nil, err
	}
	pluginKeyProvider, err := bufcli.NewPluginKeyProvider(container)
	if err != nil {
		return nil, err
	}
	policyDatas, err := policyDataProvider.GetPolicyDatasForPolicyKeys(ctx, policyKeys)
	if err != nil {
		return nil, err
	}
	policyNameToRemotePluginKeys := make(map[string][]bufplugin.PluginKey)
	for _, policyData := range policyDatas {
		policyConfig, err := policyData.Config()
		if err != nil {
			return nil, err
		}
		pluginConfigs := policyConfig.PluginConfigs()
		pluginRefs, err := xslices.MapError(pluginConfigs, func(pluginConfig bufpolicy.PluginConfig) (bufparse.Ref, error) {
			pluginRef := pluginConfig.Ref()
			if pluginRef == nil {
				return nil, fmt.Errorf("plugin config %q does not have a valid ref", pluginConfig.Name())
			}
			return pluginRef, nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get plugin refs for policy %q: %w", policyData.PolicyKey(), err)
		}
		remotePluginKeys, err := pluginKeyProvider.GetPluginKeysForPluginRefs(
			ctx,
			pluginRefs,
			bufplugin.DigestTypeP1,
		)
		if err != nil {
			return nil, err
		}
		policyName := policyData.PolicyKey().FullName().String()
		if len(remotePluginKeys) > 0 {
			policyNameToRemotePluginKeys[policyName] = remotePluginKeys
		}
	}
	return policyNameToRemotePluginKeys, nil
}

func getPolicyPluginKeysForPolicyNames(
	ctx context.Context,
	container appext.Container,
	localPolicyNameToRemotePluginRefs map[string][]bufparse.Ref,
) (map[string][]bufplugin.PluginKey, error) {
	if len(localPolicyNameToRemotePluginRefs) == 0 {
		return nil, nil
	}
	pluginKeyProvider, err := bufcli.NewPluginKeyProvider(container)
	if err != nil {
		return nil, err
	}
	policyNameToRemotePluginKeys := make(map[string][]bufplugin.PluginKey)
	for policyName, pluginRefs := range localPolicyNameToRemotePluginRefs {
		remotePluginKeys, err := pluginKeyProvider.GetPluginKeysForPluginRefs(
			ctx,
			pluginRefs,
			bufplugin.DigestTypeP1,
		)
		if err != nil {
			return nil, err
		}
		if len(remotePluginKeys) > 0 {
			policyNameToRemotePluginKeys[policyName] = remotePluginKeys
		}
	}
	return policyNameToRemotePluginKeys, nil
}
