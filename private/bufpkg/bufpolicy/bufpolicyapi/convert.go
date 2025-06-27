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

package bufpolicyapi

import (
	"fmt"

	policyv1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/policy/v1beta1"
	"buf.build/go/bufplugin/option"
	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/bufpkg/bufpolicy"
)

var (
	v1beta1ProtoDigestTypeToDigestType = map[policyv1beta1.DigestType]bufpolicy.DigestType{
		policyv1beta1.DigestType_DIGEST_TYPE_O1: bufpolicy.DigestTypeO1,
	}
)

// V1Beta1ProtoToDigest converts the given proto Digest to a Digest.
//
// Validation is performed to ensure the DigestType is known, and the value
// is a valid digest value for the given DigestType.
func V1Beta1ProtoToDigest(protoDigest *policyv1beta1.Digest) (bufpolicy.Digest, error) {
	digestType, err := v1beta1ProtoToDigestType(protoDigest.Type)
	if err != nil {
		return nil, err
	}
	bufcasDigest, err := bufcas.NewDigest(protoDigest.Value)
	if err != nil {
		return nil, err
	}
	return bufpolicy.NewDigest(digestType, bufcasDigest)
}

// V1Beta1ProtoToPolicyConfig converts the given proto PolicyConfig to a PolicyConfig.
// The registry is used to resolve plugin references.
func V1Beta1ProtoToPolicyConfig(registry string, protoPolicyConfig *policyv1beta1.PolicyConfig) (bufpolicy.PolicyConfig, error) {
	return newPolicyConfig(registry, protoPolicyConfig)
}

// *** PRIVATE ***

func policyVisibilityToV1Beta1Proto(policyVisibility bufpolicy.PolicyVisibility) (policyv1beta1.PolicyVisibility, error) {
	switch policyVisibility {
	case bufpolicy.PolicyVisibilityPublic:
		return policyv1beta1.PolicyVisibility_POLICY_VISIBILITY_PUBLIC, nil
	case bufpolicy.PolicyVisibilityPrivate:
		return policyv1beta1.PolicyVisibility_POLICY_VISIBILITY_PRIVATE, nil
	default:
		return 0, fmt.Errorf("unknown PolicyVisibility: %v", policyVisibility)
	}
}

func v1beta1ProtoToDigestType(protoDigestType policyv1beta1.DigestType) (bufpolicy.DigestType, error) {
	digestType, ok := v1beta1ProtoDigestTypeToDigestType[protoDigestType]
	if !ok {
		return 0, fmt.Errorf("unknown policyv1beta1.DigestType: %v", protoDigestType)
	}
	return digestType, nil
}

// policyConfig implements bufpolicy.PolicyConfig.
type policyConfig struct {
	lintConfig     bufconfig.LintConfig
	breakingConfig bufconfig.BreakingConfig
	pluginConfigs  []bufconfig.PluginConfig
}

func newPolicyConfig(
	registry string,
	policyConfigV1Beta1 *policyv1beta1.PolicyConfig,
) (*policyConfig, error) {
	if policyConfigV1Beta1 == nil {
		return nil, fmt.Errorf("policyConfigV1Beta1 must not be nil")
	}
	lintConfig, err := getLintConfigForV1Beta1LintConfig(policyConfigV1Beta1.Lint)
	if err != nil {
		return nil, err
	}
	breakingConfig, err := getBreakingConfigForV1Beta1BreakingConfig(policyConfigV1Beta1.Breaking)
	if err != nil {
		return nil, err
	}
	pluginConfigs, err := xslices.MapError(
		policyConfigV1Beta1.Plugins,
		func(pluginConfigV1Beta1 *policyv1beta1.PolicyConfig_CheckPluginConfig) (bufconfig.PluginConfig, error) {
			return getPluginConfigForV1Beta1PluginConfig(registry, pluginConfigV1Beta1)
		},
	)
	if err != nil {
		return nil, err
	}
	return &policyConfig{
		lintConfig:     lintConfig,
		breakingConfig: breakingConfig,
		pluginConfigs:  pluginConfigs,
	}, nil
}

// LintConfig returns the LintConfig for the File.
func (p *policyConfig) LintConfig() bufconfig.LintConfig {
	return p.lintConfig
}

// BreakingConfig returns the BreakingConfig for the File.
func (p *policyConfig) BreakingConfig() bufconfig.BreakingConfig {
	return p.breakingConfig
}

// PluginConfigs returns the PluginConfigs for the File.
func (p *policyConfig) PluginConfigs() []bufconfig.PluginConfig {
	return p.pluginConfigs
}

func getLintConfigForV1Beta1LintConfig(
	lintConfigV1Beta1 *policyv1beta1.PolicyConfig_LintConfig,
) (bufconfig.LintConfig, error) {
	checkConfig, err := bufconfig.NewEnabledCheckConfig(
		bufconfig.FileVersionV2,
		lintConfigV1Beta1.GetUse(),
		lintConfigV1Beta1.GetExcept(),
		nil,
		nil,
		false,
	)
	if err != nil {
		return nil, err
	}
	return bufconfig.NewLintConfig(
		checkConfig,
		lintConfigV1Beta1.GetEnumZeroValueSuffix(),
		lintConfigV1Beta1.GetRpcAllowSameRequestResponse(),
		lintConfigV1Beta1.GetRpcAllowGoogleProtobufEmptyRequests(),
		lintConfigV1Beta1.GetRpcAllowGoogleProtobufEmptyResponses(),
		lintConfigV1Beta1.GetServiceSuffix(),
		false, // Comment ignores are not allowed in Policy files.
	), nil
}

func getBreakingConfigForV1Beta1BreakingConfig(
	breakingConfigV1Beta1 *policyv1beta1.PolicyConfig_BreakingConfig,
) (bufconfig.BreakingConfig, error) {
	checkConfig, err := bufconfig.NewEnabledCheckConfig(
		bufconfig.FileVersionV2,
		breakingConfigV1Beta1.GetUse(),
		breakingConfigV1Beta1.GetExcept(),
		nil,
		nil,
		false,
	)
	if err != nil {
		return nil, err
	}
	return bufconfig.NewBreakingConfig(
		checkConfig,
		breakingConfigV1Beta1.GetIgnoreUnstablePackages(),
	), nil
}

func getPluginConfigForV1Beta1PluginConfig(
	registry string,
	pluginConfigV1Beta1 *policyv1beta1.PolicyConfig_CheckPluginConfig,
) (bufconfig.PluginConfig, error) {
	nameV1Beta1 := pluginConfigV1Beta1.GetName()
	pluginRef, err := bufparse.NewRef(
		registry,
		nameV1Beta1.GetOwner(),
		nameV1Beta1.GetPlugin(),
		nameV1Beta1.GetRef(),
	)
	if err != nil {
		return nil, err
	}
	options, err := option.OptionsForProtoOptions(pluginConfigV1Beta1.GetOptions())
	if err != nil {
		return nil, err
	}
	optionsMap := make(map[string]any)
	options.Range(func(key string, value any) {
		optionsMap[key] = value
	})
	return bufconfig.NewRemoteWasmPluginConfig(
		pluginRef,
		optionsMap,
		pluginConfigV1Beta1.GetArgs(),
	)
}
