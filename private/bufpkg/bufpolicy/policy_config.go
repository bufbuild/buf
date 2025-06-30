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
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	pluginoptionv1 "buf.build/gen/go/bufbuild/bufplugin/protocolbuffers/go/buf/plugin/option/v1"
	"buf.build/go/bufplugin/option"
	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

// PolicyConfig is the configuration for a Policy.
type PolicyConfig interface {
	// LintConfig returns the LintConfig for the File.
	LintConfig() bufconfig.LintConfig
	// BreakingConfig returns the BreakingConfig for the File.
	BreakingConfig() bufconfig.BreakingConfig
	// PluginConfigs returns the PluginConfigs for the File.
	PluginConfigs() []bufconfig.PluginConfig
}

// MarshalPolicyConfigAsJSON marshals the PolicyConfig to a stable JSON representation.
//
// It is a valid JSON encoding of the type buf.registry.policy.v1beta1.PolicyConfig.
// This is used to calculate the O1 digest.
func MarshalPolicyConfigAsJSON(policyConfig PolicyConfig) ([]byte, error) {
	return marshalPolicyConfigAsJSON(policyConfig)
}

/// *** PRIVATE ***

// marshalPolicyConfigAsJSON implements the stable JSON representation of the PolicyConfig.
func marshalPolicyConfigAsJSON(policyConfig PolicyConfig) ([]byte, error) {
	lintConfig := policyConfig.LintConfig()
	if lintConfig == nil {
		return nil, syserror.Newf("policyConfig.LintConfig() must not be nil")
	}
	breakingConfig := policyConfig.BreakingConfig()
	if breakingConfig == nil {
		return nil, syserror.Newf("policyConfig.BreakingConfig() must not be nil")
	}
	pluginConfigs, err := xslices.MapError(policyConfig.PluginConfigs(), func(pluginConfig bufconfig.PluginConfig) (*policyV1Beta1PolicyConfig_PluginConfig, error) {
		ref := pluginConfig.Ref()
		if ref == nil {
			return nil, fmt.Errorf("PluginConfig must have a non-nil Ref")
		}
		optionsConfig, err := optionsToOptionConfig(pluginConfig.Options())
		if err != nil {
			return nil, err
		}
		return &policyV1Beta1PolicyConfig_PluginConfig{
			Name: policyV1Beta1PolicyConfig_PluginConfig_Name{
				Owner:  ref.FullName().Owner(),
				Plugin: ref.FullName().Name(),
				Ref:    ref.Ref(),
			},
			Options: optionsConfig,
			Args:    pluginConfig.Args(),
		}, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed converting PluginConfigs to PolicyConfig_CheckPluginConfig: %w", err)
	}
	slices.SortFunc(pluginConfigs, func(a, b *policyV1Beta1PolicyConfig_PluginConfig) int {
		// Sort by owner, plugin, and ref.
		return strings.Compare(
			fmt.Sprintf("%s/%s:%s", a.Name.Owner, a.Name.Plugin, a.Name.Ref),
			fmt.Sprintf("%s/%s:%s", b.Name.Owner, b.Name.Plugin, b.Name.Ref),
		)
	})
	config := policyV1Beta1PolicyConfig{
		Lint: &policyV1Beta1PolicyConfig_LintConfig{
			Use:                                  lintConfig.UseIDsAndCategories(),
			Except:                               lintConfig.ExceptIDsAndCategories(),
			EnumZeroValueSuffix:                  lintConfig.EnumZeroValueSuffix(),
			RpcAllowSameRequestResponse:          lintConfig.RPCAllowSameRequestResponse(),
			RpcAllowGoogleProtobufEmptyRequests:  lintConfig.RPCAllowGoogleProtobufEmptyRequests(),
			RpcAllowGoogleProtobufEmptyResponses: lintConfig.RPCAllowGoogleProtobufEmptyResponses(),
			ServiceSuffix:                        lintConfig.ServiceSuffix(),
		},
		Breaking: &policyV1Beta1PolicyConfig_BreakingConfig{
			Use:                    breakingConfig.UseIDsAndCategories(),
			Except:                 breakingConfig.ExceptIDsAndCategories(),
			IgnoreUnstablePackages: breakingConfig.IgnoreUnstablePackages(),
		},
		Plugins: pluginConfigs,
	}
	return json.Marshal(config)
}

// policyV1Beta1PolicyConfig is a stable JSON representation of the buf.registry.policy.v1beta1.PolicyConfig.
type policyV1Beta1PolicyConfig struct {
	Lint     *policyV1Beta1PolicyConfig_LintConfig     `json:"lint,omitempty"`
	Breaking *policyV1Beta1PolicyConfig_BreakingConfig `json:"breaking,omitempty"`
	Plugins  []*policyV1Beta1PolicyConfig_PluginConfig `json:"plugins,omitempty"`
}

// policyV1Beta1PolicyConfig_LintConfig is a stable JSON representation of the buf.registry.policy.v1beta1.PolicyConfig.LintConfig.
type policyV1Beta1PolicyConfig_LintConfig struct {
	Use                                  []string `json:"use,omitempty"`
	Except                               []string `json:"except,omitempty"`
	EnumZeroValueSuffix                  string   `json:"enumZeroValue_suffix,omitempty"`
	RpcAllowSameRequestResponse          bool     `json:"rpcAllowSame_request_response,omitempty"`
	RpcAllowGoogleProtobufEmptyRequests  bool     `json:"rpcAllowGoogleProtobufEmptyRequests,omitempty"`
	RpcAllowGoogleProtobufEmptyResponses bool     `json:"rpcAllowGoogleProtobufEmptyResponses,omitempty"`
	ServiceSuffix                        string   `json:"serviceSuffix,omitempty"`
}

// policyV1Beta1PolicyConfig_BreakingConfig is a stable JSON representation of the buf.registry.policy.v1beta1.PolicyConfig.BreakingConfig.
type policyV1Beta1PolicyConfig_BreakingConfig struct {
	Use                    []string `json:"use,omitempty"`
	Except                 []string `json:"except,omitempty"`
	IgnoreUnstablePackages bool     `json:"ignoreUnstablePackages,omitempty"`
}

// policyV1Beta1PolicyConfig_PluginConfig is a stable JSON representation of the buf.registry.policy.v1beta1.PolicyConfig.PluginConfig.
type policyV1Beta1PolicyConfig_PluginConfig struct {
	Name    policyV1Beta1PolicyConfig_PluginConfig_Name `json:"name,omitempty"`
	Options []*optionV1Option                           `json:"options,omitempty"`
	Args    []string                                    `json:"args,omitempty"`
}

// policyV1Beta1PolicyConfig_PluginConfig_Name is a stable JSON representation of the buf.registry.policy.v1beta1.PolicyConfig.PluginConfig.Name.
type policyV1Beta1PolicyConfig_PluginConfig_Name struct {
	Owner  string `json:"owner,omitempty"`
	Plugin string `json:"plugin,omitempty"`
	Ref    string `json:"ref,omitempty"`
}

// optionV1Option is a stable JSON representation of the buf.plugin.option.v1.Option.
type optionV1Option struct {
	Key   string         `json:"key,omitempty"`
	Value *optionV1Value `json:"value,omitempty"`
}

// optionV1Value is a stable JSON representation of the buf.plugin.option.v1.Value.
type optionV1Value struct {
	BoolValue   bool               `json:"boolValue,omitempty"`
	Int64Value  int64              `json:"intValue,omitempty"`
	DoubleValue float64            `json:"floatValue,omitempty"`
	StringValue string             `json:"stringValue,omitempty"`
	BytesValue  []byte             `json:"bytesValue,omitempty"`
	ListValue   *optionV1ListValue `json:"listValue,omitempty"`
}

// optionV1ListValue is a stable JSON representation of the buf.plugin.option.v1.ListValue.
type optionV1ListValue struct {
	Values []*optionV1Value `json:"values,omitempty"`
}

// optionsToOptionsV1Options converts a map of options to a slice of optionV1Option.
func optionsToOptionConfig(keyToValue map[string]any) ([]*optionV1Option, error) {
	options, err := option.NewOptions(keyToValue) // This will validate the options.
	if err != nil {
		return nil, fmt.Errorf("failed to convert options: %w", err)
	}
	optionsProto, err := options.ToProto()
	if err != nil {
		return nil, fmt.Errorf("failed to convert options to proto: %w", err)
	}
	// Sort the options by key to ensure a stable order.
	slices.SortFunc(optionsProto, func(a, b *pluginoptionv1.Option) int {
		return strings.Compare(a.Key, b.Key)
	})
	optionsV1Options := make([]*optionV1Option, len(optionsProto))
	for i, optionProto := range optionsProto {
		optionValue, err := optionV1ValueProtoToOptionValue(optionProto.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to convert option value: %w", err)
		}
		optionsV1Options[i] = &optionV1Option{
			Key:   optionProto.Key,
			Value: optionValue,
		}
	}
	return optionsV1Options, nil
}

func optionV1ValueProtoToOptionValue(optionValue *pluginoptionv1.Value) (*optionV1Value, error) {
	if optionValue == nil {
		return nil, nil
	}
	switch optionValue.Type.(type) {
	case *pluginoptionv1.Value_BoolValue:
		return &optionV1Value{BoolValue: optionValue.GetBoolValue()}, nil
	case *pluginoptionv1.Value_Int64Value:
		return &optionV1Value{Int64Value: optionValue.GetInt64Value()}, nil
	case *pluginoptionv1.Value_DoubleValue:
		return &optionV1Value{DoubleValue: optionValue.GetDoubleValue()}, nil
	case *pluginoptionv1.Value_StringValue:
		return &optionV1Value{StringValue: optionValue.GetStringValue()}, nil
	case *pluginoptionv1.Value_BytesValue:
		return &optionV1Value{BytesValue: optionValue.GetBytesValue()}, nil
	case *pluginoptionv1.Value_ListValue:
		listValue, err := optionV1ListValueProtoToOptionListValue(optionValue.GetListValue())
		if err != nil {
			return nil, err
		}
		return &optionV1Value{ListValue: listValue}, nil
	default:
		return nil, fmt.Errorf("unknown option value type: %T", optionValue.Type)
	}
}

func optionV1ListValueProtoToOptionListValue(listValue *pluginoptionv1.ListValue) (*optionV1ListValue, error) {
	if listValue == nil {
		return nil, nil
	}
	values := make([]*optionV1Value, len(listValue.Values))
	for i, value := range listValue.Values {
		optionValue, err := optionV1ValueProtoToOptionValue(value)
		if err != nil {
			return nil, fmt.Errorf("failed to convert option value: %w", err)
		}
		values[i] = optionValue
	}
	return &optionV1ListValue{Values: values}, nil
}
