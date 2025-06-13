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
	"github.com/bufbuild/buf/private/bufpkg/bufpolicy/bufpolicyconfig"
)

func newPolicyLintConfig(
	policyConfig bufconfig.PolicyConfig,
	policyFile bufpolicyconfig.BufPolicyYAMLFile,
) (bufconfig.LintConfig, error) {
	policyFileLintConfig := policyFile.LintConfig()
	checkConfig, err := bufconfig.NewEnabledCheckConfig(
		policyFileLintConfig.FileVersion(),
		policyFileLintConfig.UseIDsAndCategories(),
		policyFileLintConfig.ExceptIDsAndCategories(),
		policyConfig.IgnorePaths(),
		policyConfig.IgnoreIDOrCategoryToPaths(),
		policyFileLintConfig.DisableBuiltin(),
	)
	if err != nil {
		return nil, err
	}
	return bufconfig.NewLintConfig(
		checkConfig,
		policyFileLintConfig.EnumZeroValueSuffix(),
		policyFileLintConfig.RPCAllowSameRequestResponse(),
		policyFileLintConfig.RPCAllowGoogleProtobufEmptyRequests(),
		policyFileLintConfig.RPCAllowGoogleProtobufEmptyResponses(),
		policyFileLintConfig.ServiceSuffix(),
		policyFileLintConfig.AllowCommentIgnores(),
	), nil
}

func newPolicyBreakingConfig(
	policyConfig bufconfig.PolicyConfig,
	policyFile bufpolicyconfig.BufPolicyYAMLFile,
) (bufconfig.BreakingConfig, error) {
	policyFileBreakingConfig := policyFile.BreakingConfig()
	checkConfig, err := bufconfig.NewEnabledCheckConfig(
		policyFileBreakingConfig.FileVersion(),
		policyFileBreakingConfig.UseIDsAndCategories(),
		policyFileBreakingConfig.ExceptIDsAndCategories(),
		policyConfig.IgnorePaths(),
		policyConfig.IgnoreIDOrCategoryToPaths(),
		policyFileBreakingConfig.DisableBuiltin(),
	)
	if err != nil {
		return nil, err
	}
	return bufconfig.NewBreakingConfig(
		checkConfig,
		policyFileBreakingConfig.IgnoreUnstablePackages(),
	), nil
}
