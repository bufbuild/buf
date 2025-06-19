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
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	pluginoptionv1 "buf.build/gen/go/bufbuild/bufplugin/protocolbuffers/go/buf/plugin/option/v1"
	"buf.build/go/bufplugin/option"
	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

const (
	// DigestTypeO1 represents the o1 policy digest type.
	//
	// The string value of this is "o1".
	DigestTypeO1 DigestType = iota + 1
)

var (
	// AllDigestTypes are all known DigestTypes.
	AllDigestTypes = []DigestType{
		DigestTypeO1,
	}
	digestTypeToString = map[DigestType]string{
		DigestTypeO1: "o1",
	}
	stringToDigestType = map[string]DigestType{
		"o1": DigestTypeO1,
	}
)

// DigestType is a type of digest.
type DigestType int

// ParseDigestType parses a DigestType from its string representation.
//
// This reverses DigestType.String().
//
// Returns an error of type *bufparse.ParseError if the string could not be parsed.
func ParseDigestType(s string) (DigestType, error) {
	d, ok := stringToDigestType[s]
	if !ok {
		return 0, bufparse.NewParseError(
			"policy digest type",
			s,
			fmt.Errorf("unknown type: %q", s),
		)
	}
	return d, nil
}

// String prints the string representation of the DigestType.
func (d DigestType) String() string {
	s, ok := digestTypeToString[d]
	if !ok {
		return strconv.Itoa(int(d))
	}
	return s
}

// Digest is a digest of some content.
//
// It consists of a DigestType and a digest value.
type Digest interface {
	// String() prints typeString:hexValue.
	fmt.Stringer

	// Type returns the type of digest.
	//
	// Always a valid value.
	Type() DigestType
	// Value returns the digest value.
	//
	// Always non-empty.
	Value() []byte

	isDigest()
}

// NewDigest creates a new Digest.
func NewDigest(digestType DigestType, bufcasDigest bufcas.Digest) (Digest, error) {
	switch digestType {
	case DigestTypeO1:
		if bufcasDigest.Type() != bufcas.DigestTypeShake256 {
			return nil, syserror.Newf(
				"trying to create a %v Digest for a cas Digest of type %v",
				digestType,
				bufcasDigest.Type(),
			)
		}
		return newDigest(digestType, bufcasDigest), nil
	default:
		// This is a system error.
		return nil, syserror.Newf("unknown DigestType: %v", digestType)
	}
}

// ParseDigest parses a Digest from its string representation.
//
// A Digest string is of the form typeString:hexValue.
// The string is expected to be non-empty, If not, an error is returned.
//
// This reverses Digest.String().
//
// Returns an error of type *bufparse.ParseError if the string could not be parsed.
func ParseDigest(s string) (Digest, error) {
	if s == "" {
		// This should be considered a system error.
		return nil, errors.New("empty string passed to ParseDigest")
	}
	digestTypeString, hexValue, ok := strings.Cut(s, ":")
	if !ok {
		return nil, bufparse.NewParseError(
			"policy digest",
			s,
			errors.New(`must be in the form "digest_type:digest_hex_value"`),
		)
	}
	digestType, err := ParseDigestType(digestTypeString)
	if err != nil {
		return nil, bufparse.NewParseError(
			"policy digest",
			digestTypeString,
			err,
		)
	}
	value, err := hex.DecodeString(hexValue)
	if err != nil {
		return nil, bufparse.NewParseError(
			"policy digest",
			s,
			errors.New(`could not parse hex: must in the form "digest_type:digest_hex_value"`),
		)
	}
	switch digestType {
	case DigestTypeO1:
		bufcasDigest, err := bufcas.NewDigest(value)
		if err != nil {
			return nil, err
		}
		return NewDigest(digestType, bufcasDigest)
	default:
		return nil, syserror.Newf("unknown DigestType: %v", digestType)
	}
}

// DigestEqual returns true if the given Digests are considered equal.
//
// If both Digests are nil, this returns true.
//
// This checks both the DigestType and Digest value.
func DigestEqual(a Digest, b Digest) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	if a == nil {
		return true
	}
	if a.Type() != b.Type() {
		return false
	}
	return bytes.Equal(a.Value(), b.Value())
}

/// *** PRIVATE ***

type digest struct {
	digestType   DigestType
	bufcasDigest bufcas.Digest
	// Cache as we call String pretty often.
	// We could do this lazily but not worth it.
	stringValue string
}

// validation should occur outside of this function.
func newDigest(digestType DigestType, bufcasDigest bufcas.Digest) *digest {
	return &digest{
		digestType:   digestType,
		bufcasDigest: bufcasDigest,
		stringValue:  digestType.String() + ":" + hex.EncodeToString(bufcasDigest.Value()),
	}
}

func (d *digest) Type() DigestType {
	return d.digestType
}

func (d *digest) Value() []byte {
	return d.bufcasDigest.Value()
}

func (d *digest) String() string {
	return d.stringValue
}

func (*digest) isDigest() {}

// getO1Digest returns the O1 digest for the given PolicyConfig.
func getO1Digest(policyConfig PolicyConfig) (Digest, error) {
	policyDataJSON, err := marshalStablePolicyConfig(policyConfig)
	if err != nil {
		return nil, err
	}
	bufcasDigest, err := bufcas.NewDigestForContent(bytes.NewReader(policyDataJSON))
	if err != nil {
		return nil, err
	}
	return NewDigest(DigestTypeO1, bufcasDigest)
}

// marshalStablePolicyConfig marshals the given PolicyConfig to a stable JSON representation.
//
// This is used for the O1 digest and should not be used for other purposes.
// It is a valid JSON encoding of the type buf.registry.policy.v1beta1.PolicyConfig.
func marshalStablePolicyConfig(policyConfig PolicyConfig) ([]byte, error) {
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
