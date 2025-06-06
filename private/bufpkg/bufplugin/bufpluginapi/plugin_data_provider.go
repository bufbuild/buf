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

package bufpluginapi

import (
	"context"
	"fmt"
	"log/slog"

	pluginv1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/plugin/v1beta1"
	"buf.build/go/standard/xslices"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin"
	"github.com/bufbuild/buf/private/bufpkg/bufregistryapi/bufregistryapiplugin"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"
)

// NewPluginDataProvider returns a new PluginDataProvider for the given API client.
//
// A warning is printed to the logger if a given Plugin is deprecated.
func NewPluginDataProvider(
	logger *slog.Logger,
	clientProvider interface {
		bufregistryapiplugin.V1Beta1DownloadServiceClientProvider
	},
) bufplugin.PluginDataProvider {
	return newPluginDataProvider(logger, clientProvider)
}

// *** PRIVATE ***

type pluginDataProvider struct {
	logger         *slog.Logger
	clientProvider interface {
		bufregistryapiplugin.V1Beta1DownloadServiceClientProvider
	}
}

func newPluginDataProvider(
	logger *slog.Logger,
	clientProvider interface {
		bufregistryapiplugin.V1Beta1DownloadServiceClientProvider
	},
) *pluginDataProvider {
	return &pluginDataProvider{
		logger:         logger,
		clientProvider: clientProvider,
	}
}

func (p *pluginDataProvider) GetPluginDatasForPluginKeys(
	ctx context.Context,
	pluginKeys []bufplugin.PluginKey,
) ([]bufplugin.PluginData, error) {
	if len(pluginKeys) == 0 {
		return nil, nil
	}
	digestType, err := bufplugin.UniqueDigestTypeForPluginKeys(pluginKeys)
	if err != nil {
		return nil, err
	}
	if digestType != bufplugin.DigestTypeP1 {
		return nil, syserror.Newf("unsupported digest type: %v", digestType)
	}
	if _, err := bufparse.FullNameStringToUniqueValue(pluginKeys); err != nil {
		return nil, err
	}

	registryToIndexedPluginKeys := xslices.ToIndexedValuesMap(
		pluginKeys,
		func(pluginKey bufplugin.PluginKey) string {
			return pluginKey.FullName().Registry()
		},
	)
	indexedPluginDatas := make([]xslices.Indexed[bufplugin.PluginData], 0, len(pluginKeys))
	for registry, indexedPluginKeys := range registryToIndexedPluginKeys {
		indexedRegistryPluginDatas, err := p.getIndexedPluginDatasForRegistryAndIndexedPluginKeys(
			ctx,
			registry,
			indexedPluginKeys,
		)
		if err != nil {
			return nil, err
		}
		indexedPluginDatas = append(indexedPluginDatas, indexedRegistryPluginDatas...)
	}
	return xslices.IndexedToSortedValues(indexedPluginDatas), nil
}

func (p *pluginDataProvider) getIndexedPluginDatasForRegistryAndIndexedPluginKeys(
	ctx context.Context,
	registry string,
	indexedPluginKeys []xslices.Indexed[bufplugin.PluginKey],
) ([]xslices.Indexed[bufplugin.PluginData], error) {
	values := xslices.Map(indexedPluginKeys, func(indexedPluginKey xslices.Indexed[bufplugin.PluginKey]) *pluginv1beta1.DownloadRequest_Value {
		return &pluginv1beta1.DownloadRequest_Value{
			ResourceRef: &pluginv1beta1.ResourceRef{
				Value: &pluginv1beta1.ResourceRef_Name_{
					Name: &pluginv1beta1.ResourceRef_Name{
						Owner:  indexedPluginKey.Value.FullName().Owner(),
						Plugin: indexedPluginKey.Value.FullName().Name(),
						Child: &pluginv1beta1.ResourceRef_Name_Ref{
							Ref: uuidutil.ToDashless(indexedPluginKey.Value.CommitID()),
						},
					},
				},
			},
		}
	})

	pluginResponse, err := p.clientProvider.V1Beta1DownloadServiceClient(registry).Download(
		ctx,
		connect.NewRequest(&pluginv1beta1.DownloadRequest{
			Values: values,
		}),
	)
	if err != nil {
		return nil, err
	}
	pluginContents := pluginResponse.Msg.Contents
	if len(pluginContents) != len(indexedPluginKeys) {
		return nil, syserror.New("did not get the expected number of plugin datas")
	}

	commitIDToIndexedPluginKeys, err := xslices.ToUniqueValuesMapError(
		indexedPluginKeys,
		func(indexedPluginKey xslices.Indexed[bufplugin.PluginKey]) (uuid.UUID, error) {
			return indexedPluginKey.Value.CommitID(), nil
		},
	)
	if err != nil {
		return nil, err
	}

	indexedPluginDatas := make([]xslices.Indexed[bufplugin.PluginData], 0, len(indexedPluginKeys))
	for _, pluginContent := range pluginContents {
		commitID, err := uuid.Parse(pluginContent.Commit.Id)
		if err != nil {
			return nil, err
		}
		indexedPluginKey, ok := commitIDToIndexedPluginKeys[commitID]
		if !ok {
			return nil, syserror.Newf("did not get plugin key from store with commitID %q", commitID)
		}
		var getData func() ([]byte, error)
		switch compressionType := pluginContent.CompressionType; compressionType {
		case pluginv1beta1.CompressionType_COMPRESSION_TYPE_NONE:
			getData = func() ([]byte, error) {
				return pluginContent.Content, nil
			}
		case pluginv1beta1.CompressionType_COMPRESSION_TYPE_ZSTD:
			getData = func() ([]byte, error) {
				zstdDecoder, err := zstd.NewReader(nil)
				if err != nil {
					return nil, err
				}
				defer zstdDecoder.Close() // Does not return an error.
				return zstdDecoder.DecodeAll(pluginContent.Content, nil)
			}
		default:
			return nil, fmt.Errorf("unknown CompressionType: %v", compressionType)
		}
		pluginData, err := bufplugin.NewPluginData(ctx, indexedPluginKey.Value, getData)
		if err != nil {
			return nil, err
		}
		indexedPluginDatas = append(
			indexedPluginDatas,
			xslices.Indexed[bufplugin.PluginData]{
				Value: pluginData,
				Index: indexedPluginKey.Index,
			},
		)
	}
	return indexedPluginDatas, nil
}
