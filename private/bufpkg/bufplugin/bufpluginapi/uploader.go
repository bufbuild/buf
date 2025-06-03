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
	"time"

	ownerv1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/owner/v1"
	pluginv1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/plugin/v1beta1"
	"buf.build/go/standard/xslices"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin"
	"github.com/bufbuild/buf/private/bufpkg/bufregistryapi/bufregistryapiplugin"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"github.com/klauspost/compress/zstd"
)

// NewUploader returns a new Uploader for the given API client.
func NewUploader(
	logger *slog.Logger,
	pluginClientProvider interface {
		bufregistryapiplugin.V1Beta1PluginServiceClientProvider
		bufregistryapiplugin.V1Beta1UploadServiceClientProvider
	},
	options ...UploaderOption,
) bufplugin.Uploader {
	return newUploader(logger, pluginClientProvider, options...)
}

// UploaderOption is an option for a new Uploader.
type UploaderOption func(*uploader)

// *** PRIVATE ***

type uploader struct {
	logger               *slog.Logger
	pluginClientProvider interface {
		bufregistryapiplugin.V1Beta1PluginServiceClientProvider
		bufregistryapiplugin.V1Beta1UploadServiceClientProvider
	}
}

func newUploader(
	logger *slog.Logger,
	pluginClientProvider interface {
		bufregistryapiplugin.V1Beta1PluginServiceClientProvider
		bufregistryapiplugin.V1Beta1UploadServiceClientProvider
	},
	options ...UploaderOption,
) *uploader {
	uploader := &uploader{
		logger:               logger,
		pluginClientProvider: pluginClientProvider,
	}
	for _, option := range options {
		option(uploader)
	}
	return uploader
}

func (u *uploader) Upload(
	ctx context.Context,
	plugins []bufplugin.Plugin,
	options ...bufplugin.UploadOption,
) ([]bufplugin.Commit, error) {
	uploadOptions, err := bufplugin.NewUploadOptions(options)
	if err != nil {
		return nil, err
	}
	registryToIndexedPluginKeys := xslices.ToIndexedValuesMap(
		plugins,
		func(plugin bufplugin.Plugin) string {
			return plugin.FullName().Registry()
		},
	)
	indexedCommits := make([]xslices.Indexed[bufplugin.Commit], 0, len(plugins))
	for registry, indexedPluginKeys := range registryToIndexedPluginKeys {
		indexedRegistryPluginDatas, err := u.uploadIndexedPluginsForRegistry(
			ctx,
			registry,
			indexedPluginKeys,
			uploadOptions,
		)
		if err != nil {
			return nil, err
		}
		indexedCommits = append(indexedCommits, indexedRegistryPluginDatas...)
	}
	return xslices.IndexedToSortedValues(indexedCommits), nil
}

func (u *uploader) uploadIndexedPluginsForRegistry(
	ctx context.Context,
	registry string,
	indexedPlugins []xslices.Indexed[bufplugin.Plugin],
	uploadOptions bufplugin.UploadOptions,
) ([]xslices.Indexed[bufplugin.Commit], error) {
	if uploadOptions.CreateIfNotExist() {
		// We must attempt to create each Plugin one at a time, since CreatePlugins will return
		// an `AlreadyExists` if any of the Plugins we are attempting to create already exists,
		// and no new Plugins will be created.
		for _, indexedPlugin := range indexedPlugins {
			plugin := indexedPlugin.Value
			if _, err := u.createPluginIfNotExist(
				ctx,
				registry,
				plugin,
				uploadOptions.CreatePluginVisibility(),
				uploadOptions.CreatePluginType(),
			); err != nil {
				return nil, err
			}
		}
	}
	contents, err := xslices.MapError(indexedPlugins, func(indexedPlugin xslices.Indexed[bufplugin.Plugin]) (*pluginv1beta1.UploadRequest_Content, error) {
		plugin := indexedPlugin.Value
		if !plugin.IsLocal() {
			return nil, syserror.New("expected local Plugin in uploadIndexedPluginsForRegistry")
		}
		if plugin.FullName() == nil {
			return nil, syserror.Newf("expected Plugin name for local Plugin: %s", plugin.Description())
		}
		data, err := plugin.Data()
		if err != nil {
			return nil, err
		}
		compressedWasmBinary, err := zstdCompress(data)
		if err != nil {
			return nil, fmt.Errorf("could not compress Plugin data %q: %w", plugin.OpaqueID(), err)
		}
		return &pluginv1beta1.UploadRequest_Content{
			PluginRef: &pluginv1beta1.PluginRef{
				Value: &pluginv1beta1.PluginRef_Name_{
					Name: &pluginv1beta1.PluginRef_Name{
						Owner:  plugin.FullName().Owner(),
						Plugin: plugin.FullName().Name(),
					},
				},
			},
			CompressionType: pluginv1beta1.CompressionType_COMPRESSION_TYPE_ZSTD,
			Content:         compressedWasmBinary,
			ScopedLabelRefs: xslices.Map(uploadOptions.Labels(), func(label string) *pluginv1beta1.ScopedLabelRef {
				return &pluginv1beta1.ScopedLabelRef{
					Value: &pluginv1beta1.ScopedLabelRef_Name{
						Name: label,
					},
				}
			}),
			SourceControlUrl: uploadOptions.SourceControlURL(),
		}, nil
	})
	if err != nil {
		return nil, err
	}

	uploadResponse, err := u.pluginClientProvider.V1Beta1UploadServiceClient(registry).Upload(
		ctx,
		connect.NewRequest(&pluginv1beta1.UploadRequest{
			Contents: contents,
		}))
	if err != nil {
		return nil, err
	}
	pluginCommits := uploadResponse.Msg.Commits
	if len(pluginCommits) != len(indexedPlugins) {
		return nil, syserror.Newf("expected %d Commits, found %d", len(indexedPlugins), len(pluginCommits))
	}

	indexedCommits := make([]xslices.Indexed[bufplugin.Commit], 0, len(indexedPlugins))
	for i, pluginCommit := range pluginCommits {
		pluginFullName := indexedPlugins[i].Value.FullName()
		commitID, err := uuidutil.FromDashless(pluginCommit.Id)
		if err != nil {
			return nil, err
		}
		pluginKey, err := bufplugin.NewPluginKey(
			pluginFullName,
			commitID,
			func() (bufplugin.Digest, error) {
				return V1Beta1ProtoToDigest(pluginCommit.Digest)
			},
		)
		if err != nil {
			return nil, err
		}
		commit := bufplugin.NewCommit(
			pluginKey,
			func() (time.Time, error) {
				return pluginCommit.CreateTime.AsTime(), nil
			},
		)
		indexedCommits = append(
			indexedCommits,
			xslices.Indexed[bufplugin.Commit]{
				Value: commit,
				Index: i,
			},
		)
	}
	return indexedCommits, nil
}

func (u *uploader) createPluginIfNotExist(
	ctx context.Context,
	primaryRegistry string,
	plugin bufplugin.Plugin,
	createPluginVisibility bufplugin.PluginVisibility,
	createPluginType bufplugin.PluginType,
) (*pluginv1beta1.Plugin, error) {
	v1Beta1ProtoCreatePluginVisibility, err := pluginVisibilityToV1Beta1Proto(createPluginVisibility)
	if err != nil {
		return nil, err
	}
	v1Beta1ProtoCreatePluginType, err := pluginTypeToV1Beta1Proto(createPluginType)
	if err != nil {
		return nil, err
	}
	response, err := u.pluginClientProvider.V1Beta1PluginServiceClient(primaryRegistry).CreatePlugins(
		ctx,
		connect.NewRequest(
			&pluginv1beta1.CreatePluginsRequest{
				Values: []*pluginv1beta1.CreatePluginsRequest_Value{
					{
						OwnerRef: &ownerv1.OwnerRef{
							Value: &ownerv1.OwnerRef_Name{
								Name: plugin.FullName().Owner(),
							},
						},
						Name:       plugin.FullName().Name(),
						Visibility: v1Beta1ProtoCreatePluginVisibility,
						Type:       v1Beta1ProtoCreatePluginType,
					},
				},
			},
		),
	)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeAlreadyExists {
			// If a plugin already existed, then we check validate its contents.
			plugins, err := u.validatePluginsExist(ctx, primaryRegistry, []bufplugin.Plugin{plugin})
			if err != nil {
				return nil, err
			}
			if len(plugins) != 1 {
				return nil, syserror.Newf("expected 1 Plugin, found %d", len(plugins))
			}
			return plugins[0], nil
		}
		return nil, err
	}
	if len(response.Msg.Plugins) != 1 {
		return nil, syserror.Newf("expected 1 Plugin, found %d", len(response.Msg.Plugins))
	}
	// Otherwise we return the plugin we created.
	return response.Msg.Plugins[0], nil
}

func (u *uploader) validatePluginsExist(
	ctx context.Context,
	primaryRegistry string,
	plugins []bufplugin.Plugin,
) ([]*pluginv1beta1.Plugin, error) {
	response, err := u.pluginClientProvider.V1Beta1PluginServiceClient(primaryRegistry).GetPlugins(
		ctx,
		connect.NewRequest(
			&pluginv1beta1.GetPluginsRequest{
				PluginRefs: xslices.Map(
					plugins,
					func(plugin bufplugin.Plugin) *pluginv1beta1.PluginRef {
						return &pluginv1beta1.PluginRef{
							Value: &pluginv1beta1.PluginRef_Name_{
								Name: &pluginv1beta1.PluginRef_Name{
									Owner:  plugin.FullName().Owner(),
									Plugin: plugin.FullName().Name(),
								},
							},
						}
					},
				),
			},
		),
	)
	if err != nil {
		return nil, err
	}
	return response.Msg.Plugins, nil
}

func zstdCompress(data []byte) ([]byte, error) {
	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		return nil, err
	}
	defer encoder.Close()
	return encoder.EncodeAll(data, nil), nil
}
