// Copyright 2020-2024 Buf Technologies, Inc.
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

package bufpluginapi

import (
	"context"
	"log/slog"

	pluginv1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/plugin/v1beta1"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
)

// NewPluginKeyProvider returns a new PluginKeyProvider for the given API clients.
func NewPluginKeyProvider(
	logger *slog.Logger,
	clientProvider interface {
		bufapi.PluginV1Beta1CommitServiceClientProvider
		bufapi.PluginV1Beta1PluginServiceClientProvider
	},
) bufplugin.PluginKeyProvider {
	return newPluginKeyProvider(logger, clientProvider)
}

// *** PRIVATE ***

type pluginKeyProvider struct {
	logger         *slog.Logger
	clientProvider interface {
		bufapi.PluginV1Beta1CommitServiceClientProvider
		bufapi.PluginV1Beta1PluginServiceClientProvider
	}
}

func newPluginKeyProvider(
	logger *slog.Logger,
	clientProvider interface {
		bufapi.PluginV1Beta1CommitServiceClientProvider
		bufapi.PluginV1Beta1PluginServiceClientProvider
	},
) *pluginKeyProvider {
	return &pluginKeyProvider{
		logger:         logger,
		clientProvider: clientProvider,
	}
}

func (p *pluginKeyProvider) GetPluginKeysForPluginRefs(
	ctx context.Context,
	pluginRefs []bufplugin.PluginRef,
	digestType bufplugin.DigestType,
) ([]bufplugin.PluginKey, error) {
	if len(pluginRefs) == 0 {
		return nil, nil
	}
	// Check unique pluginRefs.
	if _, err := slicesext.ToUniqueValuesMapError(
		pluginRefs,
		func(pluginRef bufplugin.PluginRef) (string, error) {
			return pluginRef.String(), nil
		},
	); err != nil {
		return nil, err
	}
	registryToIndexedPluginRefs := slicesext.ToIndexedValuesMap(
		pluginRefs,
		func(pluginRef bufplugin.PluginRef) string {
			return pluginRef.PluginFullName().Registry()
		},
	)
	indexedPluginKeys := make([]slicesext.Indexed[bufplugin.PluginKey], 0, len(pluginRefs))
	for registry, indexedPluginRefs := range registryToIndexedPluginRefs {
		indexedRegistryPluginKeys, err := p.getIndexedPluginKeysForRegistryAndIndexedPluginRefs(
			ctx,
			registry,
			indexedPluginRefs,
			digestType,
		)
		if err != nil {
			return nil, err
		}
		indexedPluginKeys = append(indexedPluginKeys, indexedRegistryPluginKeys...)
	}
	return slicesext.IndexedToSortedValues(indexedPluginKeys), nil
}

func (p *pluginKeyProvider) getIndexedPluginKeysForRegistryAndIndexedPluginRefs(
	ctx context.Context,
	registry string,
	indexedPluginRefs []slicesext.Indexed[bufplugin.PluginRef],
	digestType bufplugin.DigestType,
) ([]slicesext.Indexed[bufplugin.PluginKey], error) {
	resourceRefs := slicesext.Map(indexedPluginRefs, func(indexedPluginRef slicesext.Indexed[bufplugin.PluginRef]) *pluginv1beta1.ResourceRef {
		resourceRefName := &pluginv1beta1.ResourceRef_Name{
			Owner:  indexedPluginRef.Value.PluginFullName().Owner(),
			Plugin: indexedPluginRef.Value.PluginFullName().Name(),
		}
		if ref := indexedPluginRef.Value.Ref(); ref != "" {
			resourceRefName.Child = &pluginv1beta1.ResourceRef_Name_Ref{
				Ref: ref,
			}
		}
		return &pluginv1beta1.ResourceRef{
			Value: &pluginv1beta1.ResourceRef_Name_{
				Name: resourceRefName,
			},
		}
	})

	pluginResponse, err := p.clientProvider.PluginV1Beta1CommitServiceClient(registry).GetCommits(
		ctx,
		connect.NewRequest(&pluginv1beta1.GetCommitsRequest{
			ResourceRefs: resourceRefs,
		}),
	)
	if err != nil {
		return nil, err
	}
	commits := pluginResponse.Msg.Commits
	if len(commits) != len(indexedPluginRefs) {
		return nil, syserror.New("did not get the expected number of plugin datas")
	}

	indexedPluginKeys := make([]slicesext.Indexed[bufplugin.PluginKey], len(commits))
	for i, commit := range commits {
		commitID, err := uuidutil.FromDashless(commit.Id)
		if err != nil {
			return nil, err
		}
		digest, err := V1Beta1ProtoToDigest(commit.Digest)
		if err != nil {
			return nil, err
		}
		pluginKey, err := bufplugin.NewPluginKey(
			// Note we don't have to resolve owner_name and plugin_name since we already have them.
			indexedPluginRefs[i].Value.PluginFullName(),
			commitID,
			func() (bufplugin.Digest, error) {
				return digest, nil
			},
		)
		if err != nil {
			return nil, err
		}
		indexedPluginKeys[i] = slicesext.Indexed[bufplugin.PluginKey]{
			Value: pluginKey,
			Index: indexedPluginRefs[i].Index,
		}
	}
	return indexedPluginKeys, nil
}
