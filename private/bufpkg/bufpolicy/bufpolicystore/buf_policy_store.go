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

package bufpolicystore

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"log/slog"

	policyv1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/policy/v1beta1"
	"buf.build/go/bufplugin/option"
	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/bufpkg/bufpolicy"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"google.golang.org/protobuf/encoding/protojson"
)

// PolicyDataStore reads and writes PolicysDatas.
type PolicyDataStore interface {
	// GetPolicyDatasForPolicyKeys gets the PolicyDatas from the store for the PolicyKeys.
	//
	// Returns the found PolicyDatas, and the input PolicyKeys that were not found, each
	// ordered by the order of the input PolicyKeys.
	GetPolicyDatasForPolicyKeys(context.Context, []bufpolicy.PolicyKey) (
		foundPolicyDatas []bufpolicy.PolicyData,
		notFoundPolicyKeys []bufpolicy.PolicyKey,
		err error,
	)
	// PutPolicyDatas puts the PolicyDatas to the store.
	PutPolicyDatas(ctx context.Context, moduleDatas []bufpolicy.PolicyData) error
}

// NewPolicyDataStore returns a new PolicyDataStore for the given bucket.
//
// It is assumed that the PolicyDataStore has complete control of the bucket.
//
// This is typically used to interact with a cache directory.
func NewPolicyDataStore(
	logger *slog.Logger,
	bucket storage.ReadWriteBucket,
) PolicyDataStore {
	return newPolicyDataStore(logger, bucket)
}

/// *** PRIVATE ***

type policyDataStore struct {
	logger *slog.Logger
	bucket storage.ReadWriteBucket
}

func newPolicyDataStore(
	logger *slog.Logger,
	bucket storage.ReadWriteBucket,
) *policyDataStore {
	return &policyDataStore{
		logger: logger,
		bucket: bucket,
	}
}

func (p *policyDataStore) GetPolicyDatasForPolicyKeys(
	ctx context.Context,
	policyKeys []bufpolicy.PolicyKey,
) ([]bufpolicy.PolicyData, []bufpolicy.PolicyKey, error) {
	var foundPolicyDatas []bufpolicy.PolicyData
	var notFoundPolicyKeys []bufpolicy.PolicyKey
	for _, policyKey := range policyKeys {
		policyData, err := p.getPolicyDataForPolicyKey(ctx, policyKey)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return nil, nil, err
			}
			notFoundPolicyKeys = append(notFoundPolicyKeys, policyKey)
		} else {
			foundPolicyDatas = append(foundPolicyDatas, policyData)
		}
	}
	return foundPolicyDatas, notFoundPolicyKeys, nil
}

func (p *policyDataStore) PutPolicyDatas(
	ctx context.Context,
	policyDatas []bufpolicy.PolicyData,
) error {
	for _, policyData := range policyDatas {
		if err := p.putPolicyData(ctx, policyData); err != nil {
			return err
		}
	}
	return nil
}

// getPolicyDataForPolicyKey reads the policy data for the policy key from the cache.
func (p *policyDataStore) getPolicyDataForPolicyKey(
	ctx context.Context,
	policyKey bufpolicy.PolicyKey,
) (bufpolicy.PolicyData, error) {
	policyDataStorePath, err := getPolicyDataStorePath(policyKey)
	if err != nil {
		return nil, err
	}
	if exists, err := storage.Exists(ctx, p.bucket, policyDataStorePath); err != nil {
		return nil, err
	} else if !exists {
		return nil, fs.ErrNotExist
	}
	getConfig := func() (bufpolicy.PolicyConfig, error) {
		data, err := storage.ReadPath(ctx, p.bucket, policyDataStorePath)
		if err != nil {
			return nil, err
		}
		// Validate the digest, before parsing the config.
		bufcasDigest, err := bufcas.NewDigestForContent(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		expectedDigest, err := policyKey.Digest()
		if err != nil {
			return nil, err
		}
		actualDigest, err := bufpolicy.NewDigest(expectedDigest.Type(), bufcasDigest)
		if err != nil {
			return nil, err
		}
		if !bufpolicy.DigestEqual(actualDigest, expectedDigest) {
			return nil, &bufpolicy.DigestMismatchError{
				FullName:       policyKey.FullName(),
				CommitID:       policyKey.CommitID(),
				ExpectedDigest: expectedDigest,
				ActualDigest:   actualDigest,
			}
		}
		// Create the policy config from the data.
		var configProto policyv1beta1.PolicyConfig
		if err := protojson.Unmarshal(data, &configProto); err != nil {
			return nil, err
		}
		return newPolicyConfig(policyKey.FullName().Registry(), &configProto)
	}
	return bufpolicy.NewPolicyData(
		ctx,
		policyKey,
		getConfig,
	)
}

// putPolicyData puts the policy data into the policy cache.
func (p *policyDataStore) putPolicyData(
	ctx context.Context,
	policyData bufpolicy.PolicyData,
) error {
	policyKey := policyData.PolicyKey()
	policyDataStorePath, err := getPolicyDataStorePath(policyKey)
	if err != nil {
		return err
	}
	config, err := policyData.Config()
	if err != nil {
		return err
	}
	data, err := bufpolicy.MarshalPolicyConfigAsJSON(config)
	if err != nil {
		return err
	}
	// Data is stored uncompressed.
	return storage.PutPath(ctx, p.bucket, policyDataStorePath, data)
}

// getPolicyDataStorePath returns the path for the policy data store for the policy key.
//
// This is "digestType/registry/owner/name/dashlessCommitID", e.g. the policy
// "buf.build/acme/check-policy" with commit "12345-abcde" and digest type "o1"
// will return "o1/buf.build/acme/check-policy/12345abcde.yaml".
func getPolicyDataStorePath(policyKey bufpolicy.PolicyKey) (string, error) {
	digest, err := policyKey.Digest()
	if err != nil {
		return "", err
	}
	fullName := policyKey.FullName()
	return normalpath.Join(
		digest.Type().String(),
		fullName.Registry(),
		fullName.Owner(),
		fullName.Name(),
		uuidutil.ToDashless(policyKey.CommitID())+".yaml",
	), nil
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
