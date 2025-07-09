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
	lintConfig := testLintConfig{
		use:                                  []string{"LINT_ID_1", "LINT_ID_2"},
		except:                               []string{},
		enumZeroValueSuffix:                  "enumZeroValueSuffix",
		rpcAllowSameRequestResponse:          true,
		rpcAllowGoogleProtobufEmptyRequests:  true,
		rpcAllowGoogleProtobufEmptyResponses: true,
		serviceSuffix:                        "serviceSuffix",
	}
	breakingConfig := testBreakingConfig{
		use:                    []string{"BREAKING_ID_1", "BREAKING_ID_2"},
		except:                 nil,
		ignoreUnstablePackages: true,
	}
	policyConfig := &testPolicyConfig{
		lintConfig:     lintConfig,
		breakingConfig: breakingConfig,
	}
	testPolicyConfigO1Digest(t, policyConfig, "o1:db2906a09cca66da39f800207c75a8a2134d1c7918eca793e19ab07d76bea7a8f1282827205a6b7013ee0c1196af4ae7ef3abc6c41dab039bc24153d2e2dc4af")
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
	remotePluginConfig := testPluginConfig{
		name:    remotePluginRef.String(),
		ref:     remotePluginRef,
		options: options,
		args:    args,
	}
	require.NoError(t, err)
	policyConfig.pluginConfigs = append(policyConfig.pluginConfigs, remotePluginConfig)
	testPolicyConfigO1Digest(t, policyConfig, "o1:6862edf26139073f77846d2afa6d3c23016f4f0ae9abce74ec5485bb8c65ee2c32a9da80263bdf1ea1736ca46fd8fa31c5e14610c2c3dbee4fab96985122fa14")
	remotePluginRef2, err := bufparse.NewRef("buf.build", "acme", "a-plugin", "")
	require.NoError(t, err)
	remotePluginConfig2 := testPluginConfig{
		name:    remotePluginRef2.String(),
		ref:     remotePluginRef2,
		options: options,
		args:    args,
	}
	// We should get the same digest regardless of the order of the remote plugins.
	policyConfig.pluginConfigs = append(policyConfig.pluginConfigs, remotePluginConfig2)
	const multiPluginDigest = "o1:8612d6270b3ea1e222554eb40aadd9194dcfedf772ffc00ac053abed3ce8e201487088ede5f889b1bfc6236f280e0cab47cf434f91de2a9ccc1ad562334582f7"
	testPolicyConfigO1Digest(t, policyConfig, multiPluginDigest)
	// Swap the order and assert that the digest is the same.
	policyConfig.pluginConfigs[0], policyConfig.pluginConfigs[1] = policyConfig.pluginConfigs[1], policyConfig.pluginConfigs[0]
	testPolicyConfigO1Digest(t, policyConfig, multiPluginDigest)
}

func testPolicyConfigO1Digest(t *testing.T, policyConfig PolicyConfig, expectDigest string) {
	digestFromPolicyConfig, err := getO1Digest(policyConfig)
	require.NoError(t, err)
	expectedDigest, err := ParseDigest(expectDigest)
	require.NoError(t, err)
	assert.True(t, DigestEqual(expectedDigest, digestFromPolicyConfig), "Digest mismatch, expected %q got %q", expectedDigest.String(), digestFromPolicyConfig.String())
}

type testPolicyConfig struct {
	lintConfig     LintConfig
	breakingConfig BreakingConfig
	pluginConfigs  []PluginConfig
}

func (p *testPolicyConfig) LintConfig() LintConfig {
	return p.lintConfig
}
func (p *testPolicyConfig) BreakingConfig() BreakingConfig {
	return p.breakingConfig
}
func (p *testPolicyConfig) PluginConfigs() []PluginConfig {
	return p.pluginConfigs
}

type testLintConfig struct {
	use                                  []string
	except                               []string
	enumZeroValueSuffix                  string
	rpcAllowSameRequestResponse          bool
	rpcAllowGoogleProtobufEmptyRequests  bool
	rpcAllowGoogleProtobufEmptyResponses bool
	serviceSuffix                        string
}

func (c testLintConfig) UseIDsAndCategories() []string     { return c.use }
func (c testLintConfig) ExceptIDsAndCategories() []string  { return c.except }
func (c testLintConfig) EnumZeroValueSuffix() string       { return c.enumZeroValueSuffix }
func (c testLintConfig) RPCAllowSameRequestResponse() bool { return c.rpcAllowSameRequestResponse }
func (c testLintConfig) RPCAllowGoogleProtobufEmptyRequests() bool {
	return c.rpcAllowGoogleProtobufEmptyRequests
}
func (c testLintConfig) RPCAllowGoogleProtobufEmptyResponses() bool {
	return c.rpcAllowGoogleProtobufEmptyResponses
}
func (c testLintConfig) ServiceSuffix() string { return c.serviceSuffix }

type testBreakingConfig struct {
	use                    []string
	except                 []string
	ignoreUnstablePackages bool
}

func (c testBreakingConfig) UseIDsAndCategories() []string    { return c.use }
func (c testBreakingConfig) ExceptIDsAndCategories() []string { return c.except }
func (c testBreakingConfig) IgnoreUnstablePackages() bool     { return c.ignoreUnstablePackages }

type testPluginConfig struct {
	name    string
	ref     bufparse.Ref
	options option.Options
	args    []string
}

func (c testPluginConfig) Name() string            { return c.name }
func (c testPluginConfig) Ref() bufparse.Ref       { return c.ref }
func (c testPluginConfig) Options() option.Options { return c.options }
func (c testPluginConfig) Args() []string          { return c.args }
