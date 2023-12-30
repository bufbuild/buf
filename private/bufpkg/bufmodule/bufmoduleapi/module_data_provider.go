// Copyright 2020-2023 Buf Technologies, Inc.
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

package bufmoduleapi

import (
	"context"
	"fmt"
	"io/fs"
	"sort"

	modulev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1beta1"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"go.uber.org/zap"
)

// NewModuleDataProvider returns a new ModuleDataProvider for the given API client.
//
// A warning is printed to the logger if a given Module is deprecated.
func NewModuleDataProvider(
	logger *zap.Logger,
	clientProvider bufapi.ClientProvider,
	graphProvider bufmodule.GraphProvider,
) bufmodule.ModuleDataProvider {
	return newModuleDataProvider(logger, clientProvider, graphProvider)
}

// *** PRIVATE ***

type moduleDataProvider struct {
	logger         *zap.Logger
	clientProvider bufapi.ClientProvider
	graphProvider  bufmodule.GraphProvider
}

func newModuleDataProvider(
	logger *zap.Logger,
	clientProvider bufapi.ClientProvider,
	graphProvider bufmodule.GraphProvider,
) *moduleDataProvider {
	return &moduleDataProvider{
		logger:         logger,
		clientProvider: clientProvider,
		graphProvider:  graphProvider,
	}
}
func (a *moduleDataProvider) GetModuleDatasForModuleKeys(
	ctx context.Context,
	moduleKeys []bufmodule.ModuleKey,
) ([]bufmodule.ModuleData, error) {
	if _, err := bufmodule.ModuleFullNameStringToUniqueValue(moduleKeys); err != nil {
		return nil, err
	}

	// We don't want to persist this across calls - this could grow over time and this cache
	// isn't an LRU cache, and the information also may change over time.
	protoModuleProvider := newProtoModuleProvider(a.logger, a.clientProvider)

	registryToModuleKeys := toValuesMap(
		moduleKeys,
		func(moduleKey bufmodule.ModuleKey) string {
			return moduleKey.ModuleFullName().Registry()
		},
	)
	moduleDatas := make([]bufmodule.ModuleData, 0, len(moduleKeys))
	for registry, iModuleKeys := range registryToModuleKeys {
		iModuleDatas, err := a.getModuleDatasForRegistryAndModuleKeys(
			ctx,
			protoModuleProvider,
			registry,
			iModuleKeys,
		)
		if err != nil {
			return nil, err
		}
		moduleDatas = append(moduleDatas, iModuleDatas...)
	}
	sort.Slice(
		moduleDatas,
		func(i int, j int) bool {
			return moduleDatas[i].ModuleKey().ModuleFullName().String() < moduleDatas[j].ModuleKey().ModuleFullName().String()
		},
	)
	return moduleDatas, nil
}

func (a *moduleDataProvider) getModuleDatasForRegistryAndModuleKeys(
	ctx context.Context,
	protoModuleProvider *protoModuleProvider,
	registry string,
	moduleKeys []bufmodule.ModuleKey,
) ([]bufmodule.ModuleData, error) {
	graph, err := a.graphProvider.GetGraphForModuleKeys(ctx, moduleKeys)
	if err != nil {
		return nil, err
	}
	protoContents, err := a.getProtoContentsForRegistryAndModuleKeys(
		ctx,
		protoModuleProvider,
		registry,
		moduleKeys,
	)
	if err != nil {
		return nil, err
	}
	commitIDToBucket, err := getCommitIDToBucketForProtoContents(protoContents)
	if err != nil {
		return nil, err
	}
	var moduleDatas []bufmodule.ModuleData
	if err := graph.WalkNodes(
		func(
			moduleKey bufmodule.ModuleKey,
			_ []bufmodule.ModuleKey,
			depModuleKeys []bufmodule.ModuleKey,
		) error {
			bucket, ok := commitIDToBucket[moduleKey.CommitID()]
			if !ok {
				return fmt.Errorf("no files returned for commit id %s", moduleKey.CommitID())
			}
			moduleDatas = append(
				moduleDatas,
				bufmodule.NewModuleData(
					ctx,
					moduleKey,
					func() (storage.ReadBucket, error) { return bucket, nil },
					func() ([]bufmodule.ModuleKey, error) { return depModuleKeys, nil },
				),
			)
			return nil
		},
	); err != nil {
		return nil, err
	}
	return moduleDatas, nil
}

func (a *moduleDataProvider) getProtoContentsForRegistryAndModuleKeys(
	ctx context.Context,
	protoModuleProvider *protoModuleProvider,
	registry string,
	moduleKeys []bufmodule.ModuleKey,
) ([]*modulev1beta1.DownloadResponse_Content, error) {
	protoCommitIDToModuleKey, err := slicesext.ToUniqueValuesMapError(
		moduleKeys,
		func(moduleKey bufmodule.ModuleKey) (string, error) {
			return CommitIDToProto(moduleKey.CommitID())
		},
	)
	if err != nil {
		return nil, err
	}
	protoCommitIDs := slicesext.MapKeysToSortedSlice(protoCommitIDToModuleKey)

	response, err := a.clientProvider.DownloadServiceClient(registry).Download(
		ctx,
		connect.NewRequest(
			&modulev1beta1.DownloadRequest{
				// TODO: chunking
				Values: slicesext.Map(
					protoCommitIDs,
					func(protoCommitID string) *modulev1beta1.DownloadRequest_Value {
						return &modulev1beta1.DownloadRequest_Value{
							ResourceRef: &modulev1beta1.ResourceRef{
								Value: &modulev1beta1.ResourceRef_Id{
									Id: protoCommitID,
								},
							},
						}
					},
				),
				DigestType: modulev1beta1.DigestType_DIGEST_TYPE_B5,
			},
		),
	)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			// Kind of an abuse of fs.PathError. Is there a way to get a specific ModuleKey out of this?
			return nil, &fs.PathError{Op: "read", Path: err.Error(), Err: fs.ErrNotExist}
		}
		return nil, err
	}
	if len(response.Msg.Contents) != len(moduleKeys) {
		return nil, fmt.Errorf("expected %d Contents, got %d", len(moduleKeys), len(response.Msg.Contents))
	}
	protoCommitIDToProtoContent, err := slicesext.ToUniqueValuesMapError(
		response.Msg.Contents,
		func(protoContent *modulev1beta1.DownloadResponse_Content) (string, error) {
			return protoContent.Commit.Id, nil
		},
	)
	if err != nil {
		return nil, err
	}
	for protoCommitID, moduleKey := range protoCommitIDToModuleKey {
		protoContent, ok := protoCommitIDToProtoContent[protoCommitID]
		if !ok {
			return nil, fmt.Errorf("no content returned for BSR commit ID %s", protoCommitID)
		}
		if err := a.warnIfDeprecated(
			ctx,
			protoModuleProvider,
			registry,
			protoContent.Commit,
			moduleKey,
		); err != nil {
			return nil, err
		}
	}
	return response.Msg.Contents, nil
}

func (a *moduleDataProvider) warnIfDeprecated(
	ctx context.Context,
	protoModuleProvider *protoModuleProvider,
	registry string,
	protoCommit *modulev1beta1.Commit,
	moduleKey bufmodule.ModuleKey,
) error {
	protoModule, err := protoModuleProvider.getProtoModuleForModuleID(
		ctx,
		registry,
		protoCommit.ModuleId,
	)
	if err != nil {
		return err
	}
	if protoModule.State == modulev1beta1.ModuleState_MODULE_STATE_DEPRECATED {
		a.logger.Warn(fmt.Sprintf("%s is deprecated", moduleKey.ModuleFullName().String()))
	}
	return nil
}

func getCommitIDToBucketForProtoContents(
	protoContents []*modulev1beta1.DownloadResponse_Content,
) (map[string]storage.ReadBucket, error) {
	commitIDToBucket := make(map[string]storage.ReadBucket, len(protoContents))
	for _, protoContent := range protoContents {
		commitID, err := ProtoToCommitID(protoContent.Commit.Id)
		if err != nil {
			return nil, err
		}
		bucket, err := protoFilesToBucket(protoContent.Files)
		if err != nil {
			return nil, err
		}
		commitIDToBucket[commitID] = bucket
	}
	return commitIDToBucket, nil
}
