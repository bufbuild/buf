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

package bufpolicy

import (
	"testing"

	"buf.build/go/bufplugin/option"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestO1Digest(t *testing.T) {
	t.Parallel()
	lintConfig, err := newLintConfig(
		[]string{"LINT_ID_1", "LINT_ID_2"},
		[]string{},
		"enumZeroValueSuffix",
		true,
		true,
		true,
		"serviceSuffix",
		false,
	)
	require.NoError(t, err)
	breakingConfig, err := newBreakingConfig(
		[]string{"BREAKING_ID_1", "BREAKING_ID_2"},
		nil,
		true,
		false,
	)
	require.NoError(t, err)
	policyConfig, err := newPolicyConfig(lintConfig, breakingConfig, nil)
	require.NoError(t, err)

	const policyConfigDigest = "o1:960327c2f0e3382b3e2caee5a5b8043b7a17287e82dc06a5c3a5ed76773e8622c18ddeead96bb5abe44b340965b49103462f4e5a1c11dbd05d1188d9afd7cff8"
	testPolicyConfigO1Digest(
		t,
		policyConfig,
		policyConfigDigest,
	)

	// Add a remote plugin config.
	options, err := option.NewOptions(map[string]any{
		"a": "b",
		"c": 3,
		"d": 1.2,
		"e": []string{"a", "b", "c"},
	})
	require.NoError(t, err)
	args := []string{"arg1", "arg2"}
	remotePluginRef, err := bufparse.NewRef("buf.build", "acme", "my-plugin", "v1.0.0")
	require.NoError(t, err)
	pluginConfig1, err := newPluginConfig(
		remotePluginRef.String(),
		remotePluginRef,
		options,
		args,
	)
	require.NoError(t, err)

	policyConfig, err = newPolicyConfig(lintConfig, breakingConfig, []PluginConfig{pluginConfig1})
	require.NoError(t, err)
	testPolicyConfigO1Digest(
		t,
		policyConfig,
		"o1:dee43e33da2570aa194ab5a3abe427c0c98012690bebe4e70540fb520e096a9a84b2272b3df85eeebf39d8f839b671d466517639a2495e325d619c1a9fe4eda3",
	)

	remotePluginRef2, err := bufparse.NewRef("buf.build", "acme", "a-plugin", "")
	require.NoError(t, err)
	pluginConfig2, err := newPluginConfig(
		remotePluginRef2.String(),
		remotePluginRef2,
		options,
		args,
	)
	require.NoError(t, err)

	// We should get the same digest regardless of the order of the remote plugins.
	const multiPluginDigest = "o1:d2c302094a3884a7e6afef19e774ce814308dcc0c52e3ad8f5e0ff6b5b7ec412cc61a93a39063db7418d6f7965ac7800bb41cc4f7d6eca6ac7dd5a0ee786fc70"
	policyConfig2, err := newPolicyConfig(lintConfig, breakingConfig, []PluginConfig{pluginConfig2, pluginConfig1})
	require.NoError(t, err)
	policyConfig3, err := newPolicyConfig(lintConfig, breakingConfig, []PluginConfig{pluginConfig1, pluginConfig2})
	require.NoError(t, err)
	testPolicyConfigO1Digest(t, policyConfig2, multiPluginDigest)
	testPolicyConfigO1Digest(t, policyConfig3, multiPluginDigest)

	// Check the new added disable_builtin affects the digest.
	lintConfigDisableBuiltin, err := newLintConfig(
		lintConfig.UseIDsAndCategories(),
		lintConfig.ExceptIDsAndCategories(),
		lintConfig.EnumZeroValueSuffix(),
		lintConfig.RPCAllowSameRequestResponse(),
		lintConfig.RPCAllowGoogleProtobufEmptyRequests(),
		lintConfig.RPCAllowGoogleProtobufEmptyResponses(),
		lintConfig.ServiceSuffix(),
		true, // Disable builtin
	)
	require.NoError(t, err)
	breakingConfigDisableBuiltin, err := newBreakingConfig(
		breakingConfig.UseIDsAndCategories(),
		breakingConfig.ExceptIDsAndCategories(),
		breakingConfig.IgnoreUnstablePackages(),
		true, // Disable builtin
	)
	require.NoError(t, err)
	// The digest should change if we disable builtins, for lint.
	policyConfigLintDisableBuiltin, err := newPolicyConfig(lintConfigDisableBuiltin, breakingConfig, nil)
	require.NoError(t, err)
	const lintDisableBuiltinDigest = "o1:35ecbf9c4da28e11314d548096722d6fccb41a8f0e7e05c8624d7b137cf55a6113070ad74b82b6d9abf59a6850b18aeb6dacee07d309ba36b4092f6c17720ac1"
	assert.NotEqual(t, lintDisableBuiltinDigest, policyConfigDigest)
	testPolicyConfigO1Digest(t, policyConfigLintDisableBuiltin, lintDisableBuiltinDigest)
	// And disable for breaking.
	policyConfigBreakingDisableBuiltin, err := newPolicyConfig(lintConfig, breakingConfigDisableBuiltin, nil)
	require.NoError(t, err)
	const breakingDisableBuiltinDigest = "o1:b2ab151374efa2df978583cbc8615f0fc601ed12998a3dcf1acb389712869adea57fdb7776c72e2ed0b03742913be80c14dde68b42fece8b6cf5484ed6c42338"
	assert.NotEqual(t, lintDisableBuiltinDigest, policyConfigDigest)
	testPolicyConfigO1Digest(t, policyConfigBreakingDisableBuiltin, breakingDisableBuiltinDigest)
	// And disable both.
	policyConfigBothDisableBuiltin, err := newPolicyConfig(lintConfigDisableBuiltin, breakingConfigDisableBuiltin, nil)
	require.NoError(t, err)
	const bothDisableBuiltinDigest = "o1:8588a08057e59f9b446efe60b7b782397acd37d87fa13734793b5d05c414b7e8569459b993217895e34f7985ad8ed2283fa03900b6d4a7b86789521408d13715"
	assert.NotEqual(t, bothDisableBuiltinDigest, lintDisableBuiltinDigest)
	assert.NotEqual(t, bothDisableBuiltinDigest, breakingDisableBuiltinDigest)
	assert.NotEqual(t, bothDisableBuiltinDigest, policyConfigDigest)
	testPolicyConfigO1Digest(t, policyConfigBothDisableBuiltin, bothDisableBuiltinDigest)
}

func testPolicyConfigO1Digest(t *testing.T, policyConfig PolicyConfig, expectDigest string) {
	digestFromPolicyConfig, err := getO1Digest(policyConfig)
	require.NoError(t, err)
	expectedDigest, err := ParseDigest(expectDigest)
	require.NoError(t, err)
	assert.True(t, DigestEqual(expectedDigest, digestFromPolicyConfig), "Digest mismatch, expected %q got %q", expectedDigest.String(), digestFromPolicyConfig.String())
}
