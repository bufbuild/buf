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
	"context"
	"encoding/json"
	"sync"

	pluginoptionv1 "buf.build/gen/go/bufbuild/bufplugin/protocolbuffers/go/buf/plugin/option/v1"
	policyv1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/policy/v1beta1"
	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// PolicyData presents the raw Policy data read by PolicyKey.
//
// A PolicyData generally represents the data on a Policy read from the BSR API
// or a cache.
//
// Tamper-proofing is done as part of every function.
type PolicyData interface {
	// PolicyKey used to download this PolicyData.
	//
	// The Digest from this PolicyKey is used for tamper-proofing. It will be checked
	// against the actual data downloaded before Data() returns.
	PolicyKey() PolicyKey
	// Data returns the bytes of the Policy configuration data.
	//
	// This is the stable JSON representation of the type buf.registry.policy.v1beta1.PolicyConfig.
	Config() (PolicyConfig, error)

	isPolicyData()
}

// NewPolicyData returns a new PolicyData.
//
// getData is expected to be lazily-loaded function where possible.
func NewPolicyData(
	ctx context.Context,
	policyKey PolicyKey,
	getConfig func() (PolicyConfig, error),
) (PolicyData, error) {
	return newPolicyData(
		ctx,
		policyKey,
		getConfig,
	)
}

// *** PRIVATE ***

type policyData struct {
	policyKey PolicyKey
	getConfig func() (PolicyConfig, error)

	checkDigest func() error
}

func newPolicyData(
	ctx context.Context,
	policyKey PolicyKey,
	getConfig func() (PolicyConfig, error),
) (*policyData, error) {
	policyData := &policyData{
		policyKey: policyKey,
		getConfig: getConfig,
	}
	policyData.checkDigest = sync.OnceValue(func() error {
		policyConfig, err := policyData.getConfig()
		if err != nil {
			return err
		}
		policyConfigProto, err := policyConfigToV1Beta1ProtoPolicyConfigForPolicyConfig(policyConfig)
		if err != nil {
			return err
		}
		policyDataJSON, err := marshalStable(policyConfigProto)
		if err != nil {
			return err
		}
		bufcasDigest, err := bufcas.NewDigestForContent(
			bytes.NewReader(policyDataJSON),
		)
		if err != nil {
			return err
		}
		actualDigest, err := NewDigest(DigestTypeO1, bufcasDigest)
		if err != nil {
			return err
		}
		expectedDigest, err := policyKey.Digest()
		if err != nil {
			return err
		}
		if !DigestEqual(actualDigest, expectedDigest) {
			return &DigestMismatchError{
				FullName:       policyKey.FullName(),
				CommitID:       policyKey.CommitID(),
				ExpectedDigest: expectedDigest,
				ActualDigest:   actualDigest,
			}
		}
		return nil
	})
	return policyData, nil
}

func (p *policyData) PolicyKey() PolicyKey {
	return p.policyKey
}

func (p *policyData) Config() (PolicyConfig, error) {
	if err := p.checkDigest(); err != nil {
		return nil, err
	}
	return p.getConfig()
}

func (*policyData) isPolicyData() {}

func policyConfigToV1Beta1ProtoPolicyConfigForPolicyConfig(policyConfig PolicyConfig) (*policyv1beta1.PolicyConfig, error) {
	lintConfig := policyConfig.LintConfig()
	if lintConfig == nil {
		return nil, syserror.Newf("policyConfig.LintConfig() must not be nil")
	}
	breakingConfig := policyConfig.BreakingConfig()
	if breakingConfig == nil {
		return nil, syserror.Newf("policyConfig.BreakingConfig() must not be nil")
	}
	pluginsProto, err := xslices.MapError(policyConfig.PluginConfigs(), func(pluginConfig bufconfig.PluginConfig) (*policyv1beta1.PolicyConfig_CheckPluginConfig, error) {
		if pluginConfig.Type() != bufconfig.PluginConfigTypeRemoteWasm {
			return nil, syserror.Newf("PolicyConfig.PluginConfigs elements must be of type RemoteWasm, got %s", pluginConfig.Type())
		}
		ref := pluginConfig.Ref()
		if ref == nil {
			return nil, syserror.Newf("PolicyConfig.PluginConfigs elements must have a non-nil Ref, got nil")
		}
		return &policyv1beta1.PolicyConfig_CheckPluginConfig{
			Name: &policyv1beta1.PolicyConfig_CheckPluginConfig_Name{
				Owner:  ref.FullName().Owner(),
				Plugin: ref.FullName().Name(),
			},
			Options: []*pluginoptionv1.Option{},
			Args:    []string{},
		}, nil
	})
	if err != nil {
		return nil, err
	}
	return &policyv1beta1.PolicyConfig{
		Lint: &policyv1beta1.PolicyConfig_LintConfig{
			Use:                                  lintConfig.UseIDsAndCategories(),
			Except:                               lintConfig.ExceptIDsAndCategories(),
			EnumZeroValueSuffix:                  lintConfig.EnumZeroValueSuffix(),
			RpcAllowSameRequestResponse:          lintConfig.RPCAllowSameRequestResponse(),
			RpcAllowGoogleProtobufEmptyRequests:  lintConfig.RPCAllowGoogleProtobufEmptyRequests(),
			RpcAllowGoogleProtobufEmptyResponses: lintConfig.RPCAllowGoogleProtobufEmptyResponses(),
			ServiceSuffix:                        lintConfig.ServiceSuffix(),
		},
		Breaking: &policyv1beta1.PolicyConfig_BreakingConfig{
			Use:                    breakingConfig.UseIDsAndCategories(),
			Except:                 breakingConfig.ExceptIDsAndCategories(),
			IgnoreUnstablePackages: breakingConfig.IgnoreUnstablePackages(),
		},
		Plugins: pluginsProto,
	}, nil
}

// marshalStable marshals a proto.Message to a stable JSON representation.
func marshalStable(message proto.Message) ([]byte, error) {
	// protojson does not offer a "deterministic" field ordering, but fields
	// are still ordered consistently by their index. However, protojson can
	// output inconsistent whitespace, therefore a formatter is applied after
	// marshaling to ensure consistent formatting.
	// See https://github.com/golang/protobuf/issues/1373
	messageJSON, err := protojson.Marshal(message)
	if err != nil {
		return nil, err
	}
	compactedJSON := bytes.NewBuffer(messageJSON[:0])
	if err = json.Compact(compactedJSON, messageJSON); err != nil {
		return nil, err
	}
	return compactedJSON.Bytes(), nil
}
