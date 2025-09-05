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
	"sort"
	"strings"

	pluginoptionv1 "buf.build/gen/go/bufbuild/bufplugin/protocolbuffers/go/buf/plugin/option/v1"
	policyv1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/policy/v1beta1"
	"buf.build/go/bufplugin/option"
	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

// PolicyConfig is the configuration for a Policy.
type PolicyConfig interface {
	// LintConfig returns the LintConfig for the Policy.
	LintConfig() LintConfig
	// BreakingConfig returns the BreakingConfig for the Policy.
	BreakingConfig() BreakingConfig
	// PluginConfigs returns an iterator over PluginConfig for the Policy.
	//
	// Sorted by plugin name.
	PluginConfigs() []PluginConfig

	isPolicyConfig()
}

// LintConfig is the configuration for a Policy Lint.
type LintConfig interface {
	// The list of check rules and/or categories used for the Policy.
	//
	// Sorted.
	UseIDsAndCategories() []string
	// The list of check rules and/or categories to exclude for the Policy.
	//
	// Sorted.
	ExceptIDsAndCategories() []string
	// EnumZeroValueSuffix returns the suffix that controls the behavior of the
	// ENUM_ZERO_VALUE_SUFFIX lint rule. By default, this rule verifies that the zero value of all
	// enums ends in `_UNSPECIFIED`, however, this allows organizations to choose a different
	// suffix.
	EnumZeroValueSuffix() string
	// RPCAllowSameRequestResponse returns true to allow the same message type to be used for a
	// single RPC's request and response type.
	RPCAllowSameRequestResponse() bool
	// RPCAllowGoogleProtobufEmptyRequests returns true to allow RPC requests to be
	// google.protobuf.Empty messages.
	RPCAllowGoogleProtobufEmptyRequests() bool
	// RPCAllowGoogleProtobufEmptyResponses returns true to allow RPC responses to be
	// google.protobuf.Empty messages.
	RPCAllowGoogleProtobufEmptyResponses() bool
	// ServiceSuffix returns the suffix that controls the behavior of the SERVICE_SUFFIX lint rule.
	// By default, this rule verifies that all service names are suffixed with `Service`, however
	// this allows organizations to choose a different suffix.
	ServiceSuffix() string
	// DisableBuiltin says to disable the Rules and Categories builtin to the Buf CLI and only
	// use plugins.
	//
	// This will make it as if these rules did not exist.
	DisableBuiltin() bool

	isLintConfig()
}

// BreakingConfig is the configuration for a Policy Breaking.
type BreakingConfig interface {
	// The list of check rules and/or categories used for the Policy.
	//
	// Sorted.
	UseIDsAndCategories() []string
	// The list of check rules and/or categories to exclude for the Policy.
	//
	// Sorted.
	ExceptIDsAndCategories() []string
	// IgnoreUnstablePackages returns true if unstable packages should be ignored:
	//   - v\d+test.*
	//   - v\d+(alpha|beta)\d*
	//   - v\d+p\d+(alpha|beta)\d*
	IgnoreUnstablePackages() bool
	// DisableBuiltin says to disable the Rules and Categories builtin to the Buf CLI and only
	// use plugins.
	//
	// This will make it as if these rules did not exist.
	DisableBuiltin() bool

	isBreakingConfig()
}

// PluginConfig is a configuration for a Policy Plugin.
type PluginConfig interface {
	// Name returns the name of the plugin.
	Name() string
	// Ref returns the reference to the plugin, may be nil.
	Ref() bufparse.Ref
	// Options returns the options for the plugin, which may be empty.
	Options() option.Options
	// Args returns the arguments for the plugin, which may be empty.
	Args() []string

	isPluginConfig()
}

// NewPolicyConfig creates a new PolicyConfig.
func NewPolicyConfig(
	lintConfig LintConfig,
	breakingConfig BreakingConfig,
	pluginConfigs []PluginConfig,
) (PolicyConfig, error) {
	return newPolicyConfig(
		lintConfig,
		breakingConfig,
		pluginConfigs,
	)
}

// NewLintConfig creates a new LintConfig.
func NewLintConfig(
	use []string,
	except []string,
	enumZeroValueSuffix string,
	rpcAllowSameRequestResponse bool,
	rpcAllowGoogleProtobufEmptyRequests bool,
	rpcAllowGoogleProtobufEmptyResponses bool,
	serviceSuffix string,
	disableBuiltin bool,
) (LintConfig, error) {
	return newLintConfig(
		use,
		except,
		enumZeroValueSuffix,
		rpcAllowSameRequestResponse,
		rpcAllowGoogleProtobufEmptyRequests,
		rpcAllowGoogleProtobufEmptyResponses,
		serviceSuffix,
		disableBuiltin,
	)
}

// NewBreakingConfig creates a new BreakingConfig.
func NewBreakingConfig(
	use []string,
	except []string,
	ignoreUnstablePackages bool,
	disableBuiltin bool,
) (BreakingConfig, error) {
	return newBreakingConfig(
		use,
		except,
		ignoreUnstablePackages,
		disableBuiltin,
	)
}

// NewPluginConfig creates a new PluginConfig.
func NewPluginConfig(
	name string,
	ref bufparse.Ref,
	options option.Options,
	args []string,
) (PluginConfig, error) {
	return newPluginConfig(
		name,
		ref,
		options,
		args,
	)
}

// MarshalPolicyConfigAsJSON marshals the PolicyConfig to a stable JSON representation.
//
// It is a valid JSON encoding of the type buf.registry.policy.v1beta1.PolicyConfig.
// This is used to calculate the O1 digest.
func MarshalPolicyConfigAsJSON(policyConfig PolicyConfig) ([]byte, error) {
	return marshalPolicyConfigAsJSON(policyConfig)
}

// UnmarshalJSONPolicyConfig unmarshals the given JSON data into a PolicyConfig.
//
// Data is a valid JSON encoding of the type buf.registry.policy.v1beta1.PolicyConfig.
func UnmarshalJSONPolicyConfig(registry string, data []byte) (PolicyConfig, error) {
	return unmarshalJSONPolicyConfig(registry, data)
}

// *** PRIVATE ***

type policyConfig struct {
	lintConfig     LintConfig
	breakingConfig BreakingConfig
	pluginConfigs  []PluginConfig
}

func newPolicyConfig(
	lintConfig LintConfig,
	breakingConfig BreakingConfig,
	pluginConfigs []PluginConfig,
) (PolicyConfig, error) {
	pluginConfigs = slices.Clone(pluginConfigs)
	sort.Slice(pluginConfigs, func(i, j int) bool {
		return pluginConfigs[i].Name() < pluginConfigs[j].Name()
	})
	var registry string
	for _, pluginConfig := range pluginConfigs {
		ref := pluginConfig.Ref()
		if ref == nil {
			continue // Local plugin, no need to validate.
		}
		if ref.FullName().Registry() == "" {
			return nil, syserror.Newf("plugin config %q must have a non-empty registry", pluginConfig.Name())
		}
		if registry != "" && ref.FullName().Registry() != registry {
			return nil, fmt.Errorf("all plugin configs must have the same registry, got %q and %q", registry, ref.FullName().Registry())
		}
	}
	return &policyConfig{
		lintConfig:     lintConfig,
		breakingConfig: breakingConfig,
		pluginConfigs:  pluginConfigs,
	}, nil
}

func (p *policyConfig) LintConfig() LintConfig         { return p.lintConfig }
func (p *policyConfig) BreakingConfig() BreakingConfig { return p.breakingConfig }
func (p *policyConfig) PluginConfigs() []PluginConfig  { return slices.Clone(p.pluginConfigs) }
func (p *policyConfig) isPolicyConfig()                {}

type lintConfig struct {
	use                                  []string
	except                               []string
	enumZeroValueSuffix                  string
	rpcAllowSameRequestResponse          bool
	rpcAllowGoogleProtobufEmptyRequests  bool
	rpcAllowGoogleProtobufEmptyResponses bool
	serviceSuffix                        string
	disableBuiltin                       bool
}

func newLintConfig(
	use []string,
	except []string,
	enumZeroValueSuffix string,
	rpcAllowSameRequestResponse bool,
	rpcAllowGoogleProtobufEmptyRequests bool,
	rpcAllowGoogleProtobufEmptyResponses bool,
	serviceSuffix string,
	disableBuiltin bool,
) (*lintConfig, error) {
	use = slices.Clone(use)
	sort.Strings(use)
	except = slices.Clone(except)
	sort.Strings(except)
	return &lintConfig{
		use:                                  use,
		except:                               except,
		enumZeroValueSuffix:                  enumZeroValueSuffix,
		rpcAllowSameRequestResponse:          rpcAllowSameRequestResponse,
		rpcAllowGoogleProtobufEmptyRequests:  rpcAllowGoogleProtobufEmptyRequests,
		rpcAllowGoogleProtobufEmptyResponses: rpcAllowGoogleProtobufEmptyResponses,
		serviceSuffix:                        serviceSuffix,
		disableBuiltin:                       disableBuiltin,
	}, nil
}

func (c *lintConfig) UseIDsAndCategories() []string     { return slices.Clone(c.use) }
func (c *lintConfig) ExceptIDsAndCategories() []string  { return slices.Clone(c.except) }
func (c *lintConfig) EnumZeroValueSuffix() string       { return c.enumZeroValueSuffix }
func (c *lintConfig) RPCAllowSameRequestResponse() bool { return c.rpcAllowSameRequestResponse }
func (c *lintConfig) RPCAllowGoogleProtobufEmptyRequests() bool {
	return c.rpcAllowGoogleProtobufEmptyRequests
}
func (c *lintConfig) RPCAllowGoogleProtobufEmptyResponses() bool {
	return c.rpcAllowGoogleProtobufEmptyResponses
}
func (c *lintConfig) ServiceSuffix() string { return c.serviceSuffix }
func (c *lintConfig) DisableBuiltin() bool  { return c.disableBuiltin }
func (c *lintConfig) isLintConfig()         {}

type breakingConfig struct {
	use                    []string
	except                 []string
	ignoreUnstablePackages bool
	disableBuiltin         bool
}

func newBreakingConfig(
	use []string,
	except []string,
	ignoreUnstablePackages bool,
	disableBuiltin bool,
) (*breakingConfig, error) {
	use = slices.Clone(use)
	sort.Strings(use)
	except = slices.Clone(except)
	sort.Strings(except)
	return &breakingConfig{
		use:                    use,
		except:                 except,
		ignoreUnstablePackages: ignoreUnstablePackages,
		disableBuiltin:         disableBuiltin,
	}, nil
}

func (c *breakingConfig) UseIDsAndCategories() []string    { return slices.Clone(c.use) }
func (c *breakingConfig) ExceptIDsAndCategories() []string { return slices.Clone(c.except) }
func (c *breakingConfig) IgnoreUnstablePackages() bool     { return c.ignoreUnstablePackages }
func (c *breakingConfig) DisableBuiltin() bool             { return c.disableBuiltin }
func (c *breakingConfig) isBreakingConfig()                {}

type pluginConfig struct {
	name    string
	ref     bufparse.Ref
	options option.Options
	args    []string
}

func newPluginConfig(
	name string,
	ref bufparse.Ref,
	options option.Options,
	args []string,
) (*pluginConfig, error) {
	return &pluginConfig{
		name:    name,
		ref:     ref,
		options: options,
		args:    args,
	}, nil
}

func (p *pluginConfig) Name() string            { return p.name }
func (p *pluginConfig) Ref() bufparse.Ref       { return p.ref }
func (p *pluginConfig) Options() option.Options { return p.options }
func (p *pluginConfig) Args() []string          { return slices.Clone(p.args) }
func (p *pluginConfig) isPluginConfig()         {}

func marshalPolicyConfigAsJSON(policyConfig PolicyConfig) ([]byte, error) {
	var lintConfigV1Beta1 *policyV1Beta1PolicyConfig_LintConfig
	if lintConfig := policyConfig.LintConfig(); lintConfig != nil {
		lintConfigV1Beta1 = &policyV1Beta1PolicyConfig_LintConfig{
			Use:                                  lintConfig.UseIDsAndCategories(),
			Except:                               lintConfig.ExceptIDsAndCategories(),
			EnumZeroValueSuffix:                  lintConfig.EnumZeroValueSuffix(),
			RpcAllowSameRequestResponse:          lintConfig.RPCAllowSameRequestResponse(),
			RpcAllowGoogleProtobufEmptyRequests:  lintConfig.RPCAllowGoogleProtobufEmptyRequests(),
			RpcAllowGoogleProtobufEmptyResponses: lintConfig.RPCAllowGoogleProtobufEmptyResponses(),
			ServiceSuffix:                        lintConfig.ServiceSuffix(),
			DisableBuiltin:                       lintConfig.DisableBuiltin(),
		}
	}
	var breakingConfigV1Beta1 *policyV1Beta1PolicyConfig_BreakingConfig
	if breakingConfig := policyConfig.BreakingConfig(); breakingConfig != nil {
		breakingConfigV1Beta1 = &policyV1Beta1PolicyConfig_BreakingConfig{
			Use:                    breakingConfig.UseIDsAndCategories(),
			Except:                 breakingConfig.ExceptIDsAndCategories(),
			IgnoreUnstablePackages: breakingConfig.IgnoreUnstablePackages(),
			DisableBuiltin:         breakingConfig.DisableBuiltin(),
		}
	}
	pluginConfigs, err := xslices.MapError(policyConfig.PluginConfigs(), func(pluginConfig PluginConfig) (*policyV1Beta1PolicyConfig_PluginConfig, error) {
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
		return nil, fmt.Errorf("failed to convert plugin configs: %w", err)
	}
	slices.SortFunc(pluginConfigs, func(a, b *policyV1Beta1PolicyConfig_PluginConfig) int {
		// Sort by owner, plugin, and ref.
		return strings.Compare(
			fmt.Sprintf("%s/%s:%s", a.Name.Owner, a.Name.Plugin, a.Name.Ref),
			fmt.Sprintf("%s/%s:%s", b.Name.Owner, b.Name.Plugin, b.Name.Ref),
		)
	})
	config := policyV1Beta1PolicyConfig{
		Lint:     lintConfigV1Beta1,
		Breaking: breakingConfigV1Beta1,
		Plugins:  pluginConfigs,
	}
	data, err := json.Marshal(config)
	if err != nil {
		return nil, syserror.Newf("failed to marshal PolicyConfig as JSON: %w", err)
	}
	// Assert the data is a valid JSON representation of the type
	// buf.registry.policy.v1beta1.PolicyConfig.
	var policyConfigProto policyv1beta1.PolicyConfig
	if err := protoencoding.NewJSONUnmarshaler(nil, protoencoding.JSONUnmarshalerWithDisallowUnknown()).Unmarshal(data, &policyConfigProto); err != nil {
		return nil, syserror.Newf("not a valid JSON representation of type buf.registry.policy.v1beta1.PolicyConfig: %w", err)
	}
	return data, nil
}

func unmarshalJSONPolicyConfig(registry string, data []byte) (PolicyConfig, error) {
	var policyConfigV1Beta1 policyv1beta1.PolicyConfig
	if err := protoencoding.NewJSONUnmarshaler(nil, protoencoding.JSONUnmarshalerWithDisallowUnknown()).Unmarshal(data, &policyConfigV1Beta1); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON as PolicyConfig: %w", err)
	}
	lintConfigV1Beta1 := policyConfigV1Beta1.GetLint()
	lintConfig, err := newLintConfig(
		lintConfigV1Beta1.GetUse(),
		lintConfigV1Beta1.GetExcept(),
		lintConfigV1Beta1.GetEnumZeroValueSuffix(),
		lintConfigV1Beta1.GetRpcAllowSameRequestResponse(),
		lintConfigV1Beta1.GetRpcAllowGoogleProtobufEmptyRequests(),
		lintConfigV1Beta1.GetRpcAllowGoogleProtobufEmptyResponses(),
		lintConfigV1Beta1.GetServiceSuffix(),
		lintConfigV1Beta1.GetDisableBuiltin(),
	)
	if err != nil {
		return nil, err
	}
	breakingConfigV1Beta1 := policyConfigV1Beta1.GetBreaking()
	breakingConfig, err := newBreakingConfig(
		breakingConfigV1Beta1.GetUse(),
		breakingConfigV1Beta1.GetExcept(),
		breakingConfigV1Beta1.GetIgnoreUnstablePackages(),
		breakingConfigV1Beta1.GetDisableBuiltin(),
	)
	if err != nil {
		return nil, err
	}
	pluginConfigs, err := xslices.MapError(
		policyConfigV1Beta1.GetPlugins(),
		func(pluginConfigV1Beta1 *policyv1beta1.PolicyConfig_CheckPluginConfig) (PluginConfig, error) {
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
			return newPluginConfig(
				nameV1Beta1.String(),
				pluginRef,
				pluginOptions,
				pluginConfigV1Beta1.GetArgs(),
			)
		},
	)
	if err != nil {
		return nil, err
	}
	return newPolicyConfig(
		lintConfig,
		breakingConfig,
		pluginConfigs,
	)
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
	EnumZeroValueSuffix                  string   `json:"enumZeroValueSuffix,omitempty"`
	RpcAllowSameRequestResponse          bool     `json:"rpcAllowSameRequestResponse,omitempty"`
	RpcAllowGoogleProtobufEmptyRequests  bool     `json:"rpcAllowGoogleProtobufEmptyRequests,omitempty"`
	RpcAllowGoogleProtobufEmptyResponses bool     `json:"rpcAllowGoogleProtobufEmptyResponses,omitempty"`
	ServiceSuffix                        string   `json:"serviceSuffix,omitempty"`
	DisableBuiltin                       bool     `json:"disableBuiltin,omitempty"`
}

// policyV1Beta1PolicyConfig_BreakingConfig is a stable JSON representation of the buf.registry.policy.v1beta1.PolicyConfig.BreakingConfig.
type policyV1Beta1PolicyConfig_BreakingConfig struct {
	Use                    []string `json:"use,omitempty"`
	Except                 []string `json:"except,omitempty"`
	IgnoreUnstablePackages bool     `json:"ignoreUnstablePackages,omitempty"`
	DisableBuiltin         bool     `json:"disableBuiltin,omitempty"`
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
	Int64Value  int64              `json:"int64Value,omitempty"`
	DoubleValue float64            `json:"doubleValue,omitempty"`
	StringValue string             `json:"stringValue,omitempty"`
	BytesValue  []byte             `json:"bytesValue,omitempty"`
	ListValue   *optionV1ListValue `json:"listValue,omitempty"`
}

// optionV1ListValue is a stable JSON representation of the buf.plugin.option.v1.ListValue.
type optionV1ListValue struct {
	Values []*optionV1Value `json:"values,omitempty"`
}

// optionsToOptionsV1Options converts a set of options to a slice of optionV1Option.
func optionsToOptionConfig(options option.Options) ([]*optionV1Option, error) {
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
