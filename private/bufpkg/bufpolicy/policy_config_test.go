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

package bufpolicy_test

import (
	"testing"

	optionv1 "buf.build/gen/go/bufbuild/bufplugin/protocolbuffers/go/buf/plugin/option/v1"
	policyv1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/policy/v1beta1"
	"github.com/bufbuild/buf/private/bufpkg/bufpolicy"
	"github.com/bufbuild/buf/private/bufpkg/bufpolicy/bufpolicyapi"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestMarshalPolicyConfigAsJSON(t *testing.T) {
	t.Parallel()

	testRoundTripPolicyConfigAsJSON(t, &policyv1beta1.PolicyConfig{
		Lint: &policyv1beta1.PolicyConfig_LintConfig{
			DisableBuiltin: true,
		},
		Breaking: &policyv1beta1.PolicyConfig_BreakingConfig{
			DisableBuiltin: true,
		},
	})
	testRoundTripPolicyConfigAsJSON(t, &policyv1beta1.PolicyConfig{
		Lint: &policyv1beta1.PolicyConfig_LintConfig{
			Use:                                  []string{"USE_ID"},
			Except:                               []string{"EXCEPT_ID"},
			EnumZeroValueSuffix:                  "EnumZeroValue",
			RpcAllowSameRequestResponse:          true,
			RpcAllowGoogleProtobufEmptyRequests:  true,
			RpcAllowGoogleProtobufEmptyResponses: true,
			ServiceSuffix:                        "ServiceSuffix",
		},
		Breaking: &policyv1beta1.PolicyConfig_BreakingConfig{
			Use:                    []string{"BREAKING_ID_1", "BREAKING_ID_2"},
			Except:                 []string{"BREAKING_EXCEPT_ID"},
			IgnoreUnstablePackages: true,
		},
		// Plugins are sorted by owner, plugin, and ref.
		Plugins: []*policyv1beta1.PolicyConfig_CheckPluginConfig{
			{
				Name: &policyv1beta1.PolicyConfig_CheckPluginConfig_Name{
					Owner:  "PluginOwner",
					Plugin: "PluginName",
					Ref:    "PluginRef1",
				},
				// Option keys are sorted in the JSON output.
				Options: []*optionv1.Option{
					{
						Key: "boolOption",
						Value: &optionv1.Value{
							Type: &optionv1.Value_BoolValue{
								BoolValue: true,
							},
						},
					},
					{
						Key: "bytesValue",
						Value: &optionv1.Value{
							Type: &optionv1.Value_BytesValue{
								BytesValue: []byte{0, 1, 2, 3, 4},
							},
						},
					},
					{
						Key: "doubleOption",
						Value: &optionv1.Value{
							Type: &optionv1.Value_DoubleValue{
								DoubleValue: 3.14159,
							},
						},
					},
					{
						Key: "int64Option",
						Value: &optionv1.Value{
							Type: &optionv1.Value_Int64Value{
								Int64Value: 1234567890,
							},
						},
					},
					{
						Key: "listOption",
						Value: &optionv1.Value{
							Type: &optionv1.Value_ListValue{
								ListValue: &optionv1.ListValue{
									Values: []*optionv1.Value{
										{
											Type: &optionv1.Value_StringValue{
												StringValue: "listValue1",
											},
										},
										{
											Type: &optionv1.Value_StringValue{
												StringValue: "listValue2",
											},
										},
									},
								},
							},
						},
					},
					{
						Key: "stringOption",
						Value: &optionv1.Value{
							Type: &optionv1.Value_StringValue{
								StringValue: "value1",
							},
						},
					},
				},
				Args: []string{"arg1", "arg2"},
			},
			{
				Name: &policyv1beta1.PolicyConfig_CheckPluginConfig_Name{
					Owner:  "PluginOwner",
					Plugin: "PluginName",
					Ref:    "PluginRef2",
				},
				Options: []*optionv1.Option{},
				Args:    []string{"arg1", "arg2"},
			},
		},
	})
}

func testRoundTripPolicyConfigAsJSON(t *testing.T, policyConfigV1Beta1 *policyv1beta1.PolicyConfig) {
	const registry = "bufbuild.test"
	policyConfig, err := bufpolicyapi.V1Beta1ProtoToPolicyConfig(registry, policyConfigV1Beta1)
	require.NoError(t, err)
	data, err := bufpolicy.MarshalPolicyConfigAsJSON(policyConfig)
	require.NoError(t, err)
	require.NotEmpty(t, data)
	var policyConfigV1Beta1Copy policyv1beta1.PolicyConfig
	require.NoError(t, protoencoding.NewJSONUnmarshaler(nil, protoencoding.JSONUnmarshalerWithDisallowUnknown()).Unmarshal(data, &policyConfigV1Beta1Copy))
	diff := cmp.Diff(policyConfigV1Beta1, &policyConfigV1Beta1Copy, protocmp.Transform())
	require.Empty(t, diff)
}
