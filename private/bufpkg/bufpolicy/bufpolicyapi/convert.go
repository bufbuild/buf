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
func V1Beta1ProtoToPolicyConfig(registry string, policyConfigV1Beta1 *policyv1beta1.PolicyConfig) (bufpolicy.PolicyConfig, error) {
	lintConfig, err := getLintConfigForV1Beta1LintConfig(policyConfigV1Beta1.GetLint())
	if err != nil {
		return nil, err
	}
	breakingConfig, err := getBreakingConfigForV1Beta1BreakingConfig(policyConfigV1Beta1.GetBreaking())
	if err != nil {
		return nil, err
	}
	pluginConfigs, err := xslices.MapError(
		policyConfigV1Beta1.GetPlugins(),
		func(pluginConfigV1Beta1 *policyv1beta1.PolicyConfig_CheckPluginConfig) (bufpolicy.PluginConfig, error) {
			return getPluginConfigForV1Beta1PluginConfig(registry, pluginConfigV1Beta1)
		},
	)
	if err != nil {
		return nil, err
	}
	return bufpolicy.NewPolicyConfig(
		lintConfig,
		breakingConfig,
		pluginConfigs,
	)
}

// PolicyConfigToV1Beta1Proto converts the given PolicyConfig to a proto PolicyConfig.
func PolicyConfigToV1Beta1Proto(policyConfig bufpolicy.PolicyConfig) (*policyv1beta1.PolicyConfig, error) {
	pluginConfigs, err := xslices.MapError(
		policyConfig.PluginConfigs(),
		func(pluginConfig bufpolicy.PluginConfig) (*policyv1beta1.PolicyConfig_CheckPluginConfig, error) {
			pluginRef := pluginConfig.Ref()
			if pluginRef == nil {
				return nil, fmt.Errorf("plugin config %q has no reference", pluginConfig.Name())
			}
			pluginOptions, err := pluginConfig.Options().ToProto()
			if err != nil {
				return nil, err
			}
			return &policyv1beta1.PolicyConfig_CheckPluginConfig{
				Name: &policyv1beta1.PolicyConfig_CheckPluginConfig_Name{
					Owner:  pluginRef.FullName().Owner(),
					Plugin: pluginRef.FullName().Name(),
					Ref:    pluginRef.Ref(),
				},
				Options: pluginOptions,
				Args:    pluginConfig.Args(),
			}, nil
		},
	)
	if err != nil {
		return nil, err
	}
	return &policyv1beta1.PolicyConfig{
		Lint: &policyv1beta1.PolicyConfig_LintConfig{
			Use:                                  policyConfig.LintConfig().UseIDsAndCategories(),
			Except:                               policyConfig.LintConfig().ExceptIDsAndCategories(),
			EnumZeroValueSuffix:                  policyConfig.LintConfig().EnumZeroValueSuffix(),
			RpcAllowSameRequestResponse:          policyConfig.LintConfig().RPCAllowSameRequestResponse(),
			RpcAllowGoogleProtobufEmptyRequests:  policyConfig.LintConfig().RPCAllowGoogleProtobufEmptyRequests(),
			RpcAllowGoogleProtobufEmptyResponses: policyConfig.LintConfig().RPCAllowGoogleProtobufEmptyResponses(),
			ServiceSuffix:                        policyConfig.LintConfig().ServiceSuffix(),
		},
		Breaking: &policyv1beta1.PolicyConfig_BreakingConfig{
			Use:                    policyConfig.BreakingConfig().UseIDsAndCategories(),
			Except:                 policyConfig.BreakingConfig().ExceptIDsAndCategories(),
			IgnoreUnstablePackages: policyConfig.BreakingConfig().IgnoreUnstablePackages(),
		},
		Plugins: pluginConfigs,
	}, nil
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

func getLintConfigForV1Beta1LintConfig(
	lintConfigV1Beta1 *policyv1beta1.PolicyConfig_LintConfig,
) (bufpolicy.LintConfig, error) {
	return bufpolicy.NewLintConfig(
		lintConfigV1Beta1.GetUse(),
		lintConfigV1Beta1.GetExcept(),
		lintConfigV1Beta1.GetEnumZeroValueSuffix(),
		lintConfigV1Beta1.GetRpcAllowSameRequestResponse(),
		lintConfigV1Beta1.GetRpcAllowGoogleProtobufEmptyRequests(),
		lintConfigV1Beta1.GetRpcAllowGoogleProtobufEmptyResponses(),
		lintConfigV1Beta1.GetServiceSuffix(),
		lintConfigV1Beta1.GetDisableBuiltin(),
	)
}

func getBreakingConfigForV1Beta1BreakingConfig(
	breakingConfigV1Beta1 *policyv1beta1.PolicyConfig_BreakingConfig,
) (bufpolicy.BreakingConfig, error) {
	return bufpolicy.NewBreakingConfig(
		breakingConfigV1Beta1.GetUse(),
		breakingConfigV1Beta1.GetExcept(),
		breakingConfigV1Beta1.GetIgnoreUnstablePackages(),
		breakingConfigV1Beta1.GetDisableBuiltin(),
	)
}

func getPluginConfigForV1Beta1PluginConfig(
	registry string,
	pluginConfigV1Beta1 *policyv1beta1.PolicyConfig_CheckPluginConfig,
) (bufpolicy.PluginConfig, error) {
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
	pluginOptions, err := option.OptionsForProtoOptions(pluginConfigV1Beta1.GetOptions())
	if err != nil {
		return nil, err
	}
	return bufpolicy.NewPluginConfig(
		nameV1Beta1.String(),
		pluginRef,
		pluginOptions,
		pluginConfigV1Beta1.GetArgs(),
	)
}
