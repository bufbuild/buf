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

package bufcheck

import (
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufpolicy"
	"github.com/bufbuild/buf/private/bufpkg/bufpolicy/bufpolicyconfig"
)

// policyToBufConfigLintConfig creates a new bufconfig.LintConfig from the source bufpolicy.Policy
// applied through the buf.yaml file's policy configuration.
func policyToBufConfigLintConfig(
	policy bufpolicy.Policy,
	bufYamlPolicyConfig bufconfig.PolicyConfig,
) (bufconfig.LintConfig, error) {
	policyConfig, err := policy.Config()
	if err != nil {
		return nil, err
	}
	policyLintConfig := policyConfig.LintConfig()
	checkConfig, err := bufconfig.NewEnabledCheckConfig(
		bufconfig.FileVersionV2,
		policyLintConfig.UseIDsAndCategories(),
		policyLintConfig.ExceptIDsAndCategories(),
		bufYamlPolicyConfig.IgnorePaths(),
		bufYamlPolicyConfig.IgnoreIDOrCategoryToPaths(),
		policyLintConfig.DisableBuiltin(),
	)
	if err != nil {
		return nil, err
	}
	return bufconfig.NewLintConfig(
		checkConfig,
		policyLintConfig.EnumZeroValueSuffix(),
		policyLintConfig.RPCAllowSameRequestResponse(),
		policyLintConfig.RPCAllowGoogleProtobufEmptyRequests(),
		policyLintConfig.RPCAllowGoogleProtobufEmptyResponses(),
		policyLintConfig.ServiceSuffix(),
		false, // We do not allow comment ignores in policy files.
	), nil
}

// policyToBufConfigBreakingConfig creates a new bufconfig.BreakingConfig from the source bufpolicy.Policy
// applied through the buf.yaml file's policy configuration.
func policyToBufConfigBreakingConfig(
	policy bufpolicy.Policy,
	bufYamlPolicyConfig bufconfig.PolicyConfig,
) (bufconfig.BreakingConfig, error) {
	policyConfig, err := policy.Config()
	if err != nil {
		return nil, err
	}
	policyBreakingConfig := policyConfig.BreakingConfig()
	checkConfig, err := bufconfig.NewEnabledCheckConfig(
		bufconfig.FileVersionV2,
		policyBreakingConfig.UseIDsAndCategories(),
		policyBreakingConfig.ExceptIDsAndCategories(),
		bufYamlPolicyConfig.IgnorePaths(),
		bufYamlPolicyConfig.IgnoreIDOrCategoryToPaths(),
		policyBreakingConfig.DisableBuiltin(),
	)
	if err != nil {
		return nil, err
	}
	return bufconfig.NewBreakingConfig(
		checkConfig,
		policyBreakingConfig.IgnoreUnstablePackages(),
	), nil
}

// policyToBufConfigPluginConfigs converts the given policy to a slice of bufconfig.PluginConfig.
func policyToBufConfigPluginConfigs(
	policy bufpolicy.Policy,
) ([]bufconfig.PluginConfig, error) {
	policyConfig, err := policy.Config()
	if err != nil {
		return nil, err
	}
	pluginConfigs := policyConfig.PluginConfigs()
	return bufpolicyconfig.PluginConfigsToBufConfig(pluginConfigs)
}
