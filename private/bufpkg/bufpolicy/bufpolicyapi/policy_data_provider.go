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
	"context"
	"log/slog"

	policyv1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/policy/v1beta1"
	"buf.build/go/bufplugin/option"
	"buf.build/go/standard/xslices"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/bufpkg/bufpolicy"
	"github.com/bufbuild/buf/private/bufpkg/bufregistryapi/bufregistryapipolicy"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"github.com/google/uuid"
)

// NewPolicyDataProvider returns a new PolicyDataProvider for the given API client.
//
// A warning is printed to the logger if a given Policy is deprecated.
func NewPolicyDataProvider(
	logger *slog.Logger,
	clientProvider interface {
		bufregistryapipolicy.V1Beta1DownloadServiceClientProvider
	},
) bufpolicy.PolicyDataProvider {
	return newPolicyDataProvider(logger, clientProvider)
}

// *** PRIVATE ***

type policyDataProvider struct {
	logger         *slog.Logger
	clientProvider interface {
		bufregistryapipolicy.V1Beta1DownloadServiceClientProvider
	}
}

func newPolicyDataProvider(
	logger *slog.Logger,
	clientProvider interface {
		bufregistryapipolicy.V1Beta1DownloadServiceClientProvider
	},
) *policyDataProvider {
	return &policyDataProvider{
		logger:         logger,
		clientProvider: clientProvider,
	}
}

func (p *policyDataProvider) GetPolicyDatasForPolicyKeys(
	ctx context.Context,
	policyKeys []bufpolicy.PolicyKey,
) ([]bufpolicy.PolicyData, error) {
	if len(policyKeys) == 0 {
		return nil, nil
	}
	digestType, err := bufpolicy.UniqueDigestTypeForPolicyKeys(policyKeys)
	if err != nil {
		return nil, err
	}
	if digestType != bufpolicy.DigestTypeO1 {
		return nil, syserror.Newf("unsupported digest type: %v", digestType)
	}
	if _, err := bufparse.FullNameStringToUniqueValue(policyKeys); err != nil {
		return nil, err
	}

	registryToIndexedPolicyKeys := xslices.ToIndexedValuesMap(
		policyKeys,
		func(policyKey bufpolicy.PolicyKey) string {
			return policyKey.FullName().Registry()
		},
	)
	indexedPolicyDatas := make([]xslices.Indexed[bufpolicy.PolicyData], 0, len(policyKeys))
	for registry, indexedPolicyKeys := range registryToIndexedPolicyKeys {
		indexedRegistryPolicyDatas, err := p.getIndexedPolicyDatasForRegistryAndIndexedPolicyKeys(
			ctx,
			registry,
			indexedPolicyKeys,
		)
		if err != nil {
			return nil, err
		}
		indexedPolicyDatas = append(indexedPolicyDatas, indexedRegistryPolicyDatas...)
	}
	return xslices.IndexedToSortedValues(indexedPolicyDatas), nil
}

func (p *policyDataProvider) getIndexedPolicyDatasForRegistryAndIndexedPolicyKeys(
	ctx context.Context,
	registry string,
	indexedPolicyKeys []xslices.Indexed[bufpolicy.PolicyKey],
) ([]xslices.Indexed[bufpolicy.PolicyData], error) {
	values := xslices.Map(indexedPolicyKeys, func(indexedPolicyKey xslices.Indexed[bufpolicy.PolicyKey]) *policyv1beta1.DownloadRequest_Value {
		return &policyv1beta1.DownloadRequest_Value{
			ResourceRef: &policyv1beta1.ResourceRef{
				Value: &policyv1beta1.ResourceRef_Id{
					Id: uuidutil.ToDashless(indexedPolicyKey.Value.CommitID()),
				},
			},
		}
	})

	policyResponse, err := p.clientProvider.V1Beta1DownloadServiceClient(registry).Download(
		ctx,
		connect.NewRequest(&policyv1beta1.DownloadRequest{
			Values: values,
		}),
	)
	if err != nil {
		return nil, err
	}
	policyContents := policyResponse.Msg.Contents
	if len(policyContents) != len(indexedPolicyKeys) {
		return nil, syserror.New("did not get the expected number of policy datas")
	}

	commitIDToIndexedPolicyKeys, err := xslices.ToUniqueValuesMapError(
		indexedPolicyKeys,
		func(indexedPolicyKey xslices.Indexed[bufpolicy.PolicyKey]) (uuid.UUID, error) {
			return indexedPolicyKey.Value.CommitID(), nil
		},
	)
	if err != nil {
		return nil, err
	}

	indexedPolicyDatas := make([]xslices.Indexed[bufpolicy.PolicyData], 0, len(indexedPolicyKeys))
	for _, policyContent := range policyContents {
		commitID, err := uuid.Parse(policyContent.Commit.Id)
		if err != nil {
			return nil, err
		}
		indexedPolicyKey, ok := commitIDToIndexedPolicyKeys[commitID]
		if !ok {
			return nil, syserror.Newf("did not get policy key from store with commitID %q", commitID)
		}
		getContent := func() (bufpolicy.PolicyConfig, error) {
			return newPolicyConfig(registry, policyContent.GetConfig())
		}
		policyData, err := bufpolicy.NewPolicyData(ctx, indexedPolicyKey.Value, getContent)
		if err != nil {
			return nil, err
		}
		indexedPolicyDatas = append(
			indexedPolicyDatas,
			xslices.Indexed[bufpolicy.PolicyData]{
				Value: policyData,
				Index: indexedPolicyKey.Index,
			},
		)
	}
	return indexedPolicyDatas, nil
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
		lintConfigV1Beta1.Use,
		lintConfigV1Beta1.Except,
		nil,
		nil,
		false,
	)
	if err != nil {
		return nil, err
	}
	return bufconfig.NewLintConfig(
		checkConfig,
		lintConfigV1Beta1.EnumZeroValueSuffix,
		lintConfigV1Beta1.RpcAllowSameRequestResponse,
		lintConfigV1Beta1.RpcAllowGoogleProtobufEmptyRequests,
		lintConfigV1Beta1.RpcAllowGoogleProtobufEmptyResponses,
		lintConfigV1Beta1.ServiceSuffix,
		false, // Comment ignores are not allowed in Policy files.
	), nil
}

func getBreakingConfigForV1Beta1BreakingConfig(
	breakingConfigV1Beta1 *policyv1beta1.PolicyConfig_BreakingConfig,
) (bufconfig.BreakingConfig, error) {
	checkConfig, err := bufconfig.NewEnabledCheckConfig(
		bufconfig.FileVersionV2,
		breakingConfigV1Beta1.Use,
		breakingConfigV1Beta1.Except,
		nil,
		nil,
		false,
	)
	if err != nil {
		return nil, err
	}
	return bufconfig.NewBreakingConfig(
		checkConfig,
		breakingConfigV1Beta1.IgnoreUnstablePackages,
	), nil
}

func getPluginConfigForV1Beta1PluginConfig(
	registry string,
	pluginConfigV1Beta1 *policyv1beta1.PolicyConfig_CheckPluginConfig,
) (bufconfig.PluginConfig, error) {
	nameV1Beta1 := pluginConfigV1Beta1.Name
	pluginRef, err := bufparse.NewRef(
		registry,
		nameV1Beta1.Owner,
		nameV1Beta1.Plugin,
		nameV1Beta1.Ref,
	)
	if err != nil {
		return nil, err
	}
	options, err := option.OptionsForProtoOptions(pluginConfigV1Beta1.Options)
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
		pluginConfigV1Beta1.Args,
	)
}
