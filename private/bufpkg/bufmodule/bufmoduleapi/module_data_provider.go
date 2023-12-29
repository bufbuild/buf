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
	"errors"
	"fmt"
	"io/fs"

	modulev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1beta1"
	ownerv1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/owner/v1beta1"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/cache"
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
) bufmodule.ModuleDataProvider {
	return newModuleDataProvider(logger, clientProvider)
}

// *** PRIVATE ***

type moduleDataProvider struct {
	logger         *zap.Logger
	clientProvider bufapi.ClientProvider
}

func newModuleDataProvider(
	logger *zap.Logger,
	clientProvider bufapi.ClientProvider,
) *moduleDataProvider {
	return &moduleDataProvider{
		logger:         logger,
		clientProvider: clientProvider,
	}
}

func (a *moduleDataProvider) GetOptionalModuleDatasForModuleKeys(
	ctx context.Context,
	moduleKeys ...bufmodule.ModuleKey,
) ([]bufmodule.OptionalModuleData, error) {
	// We don't want to persist these across calls - this could grow over time and this cache
	// isn't an LRU cache, and the information also may change over time.
	var protoModuleCache cache.Cache[string, *modulev1beta1.Module]
	var protoOwnerCache cache.Cache[string, *ownerv1beta1.Owner]
	// TODO: Do the work to coalesce ModuleKeys by registry hostname, make calls out to the CommitService
	// per registry, then get back the resulting data, and order it in the same order as the input ModuleKeys.
	// Make sure to respect 250 max.
	optionalModuleDatas := make([]bufmodule.OptionalModuleData, len(moduleKeys))
	for i, moduleKey := range moduleKeys {
		moduleData, err := a.getModuleDataForModuleKey(
			ctx,
			&protoModuleCache,
			&protoOwnerCache,
			moduleKey,
		)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return nil, err
			}
		}
		optionalModuleDatas[i] = bufmodule.NewOptionalModuleData(moduleData)
	}
	return optionalModuleDatas, nil
}

func (a *moduleDataProvider) getModuleDataForModuleKey(
	ctx context.Context,
	protoModuleCache *cache.Cache[string, *modulev1beta1.Module],
	protoOwnerCache *cache.Cache[string, *ownerv1beta1.Owner],
	moduleKey bufmodule.ModuleKey,
) (bufmodule.ModuleData, error) {
	registryHostname := moduleKey.ModuleFullName().Registry()

	protoCommitID, err := CommitIDToProto(moduleKey.CommitID())
	if err != nil {
		return nil, err
	}
	response, err := a.clientProvider.DownloadServiceClient(registryHostname).Download(
		ctx,
		connect.NewRequest(
			&modulev1beta1.DownloadRequest{
				Values: []*modulev1beta1.DownloadRequest_Value{
					{
						ResourceRef: &modulev1beta1.ResourceRef{
							Value: &modulev1beta1.ResourceRef_Id{
								Id: protoCommitID,
							},
						},
					},
				},
				DigestType: modulev1beta1.DigestType_DIGEST_TYPE_B5,
			},
		),
	)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return nil, &fs.PathError{Op: "read", Path: moduleKey.ModuleFullName().String(), Err: fs.ErrNotExist}
		}
		return nil, err
	}
	if len(response.Msg.References) != 1 {
		return nil, fmt.Errorf("expected 1 Reference, got %d", len(response.Msg.References))
	}
	protoCommitIDToCommit, err := getProtoCommitIDToCommitForProtoDownloadResponse(response.Msg)
	if err != nil {
		return nil, err
	}
	protoCommitIDToBucket, err := getProtoCommitIDToBucketForProtoDownloadResponse(response.Msg)
	if err != nil {
		return nil, err
	}
	if err := a.warnIfDeprecated(
		ctx,
		registryHostname,
		moduleKey,
		protoCommitIDToCommit,
		protoModuleCache,
		response.Msg.References[0],
	); err != nil {
		return nil, err
	}
	return a.getModuleDataForProtoDownloadResponseReference(
		ctx,
		registryHostname,
		moduleKey,
		protoCommitIDToCommit,
		protoCommitIDToBucket,
		protoModuleCache,
		protoOwnerCache,
		response.Msg.References[0],
	)
}

func (a *moduleDataProvider) warnIfDeprecated(
	ctx context.Context,
	registryHostname string,
	moduleKey bufmodule.ModuleKey,
	protoCommitIDToCommit map[string]*modulev1beta1.Commit,
	protoModuleCache *cache.Cache[string, *modulev1beta1.Module],
	protoReference *modulev1beta1.DownloadResponse_Reference,
) error {
	protoCommit, ok := protoCommitIDToCommit[protoReference.CommitId]
	if !ok {
		return fmt.Errorf("commit_id %q was not present in Commits on DownloadModuleResponse", protoReference.CommitId)
	}
	protoModule, err := a.getProtoModuleForModuleID(
		ctx,
		registryHostname,
		protoModuleCache,
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

func (a *moduleDataProvider) getModuleDataForProtoDownloadResponseReference(
	ctx context.Context,
	registryHostname string,
	moduleKey bufmodule.ModuleKey,
	protoCommitIDToCommit map[string]*modulev1beta1.Commit,
	protoCommitIDToBucket map[string]storage.ReadBucket,
	protoModuleCache *cache.Cache[string, *modulev1beta1.Module],
	protoOwnerCache *cache.Cache[string, *ownerv1beta1.Owner],
	protoReference *modulev1beta1.DownloadResponse_Reference,
) (bufmodule.ModuleData, error) {
	bucket, ok := protoCommitIDToBucket[protoReference.CommitId]
	if !ok {
		return nil, fmt.Errorf("commit_id %q was not present in Contents on DownloadModuleResponse", protoReference.CommitId)
	}
	depProtoCommits, err := slicesext.MapError(
		protoReference.DepCommitIds,
		func(protoCommitID string) (*modulev1beta1.Commit, error) {
			commit, ok := protoCommitIDToCommit[protoCommitID]
			if !ok {
				return nil, fmt.Errorf("dep_commit_id %q was not present in Commits on DownloadModuleResponse", protoCommitID)
			}
			return commit, nil
		},
	)
	if err != nil {
		return nil, err
	}
	return bufmodule.NewModuleData(
		ctx,
		moduleKey,
		func() (storage.ReadBucket, error) {
			return bucket, nil
		},
		func() ([]bufmodule.ModuleKey, error) {
			return a.getModuleKeysForProtoCommits(
				ctx,
				registryHostname,
				protoModuleCache,
				protoOwnerCache,
				depProtoCommits,
			)
		},
	), nil
}

// TODO: We could call this for multiple Commits at once, but this is a bunch of extra work.
// We can do this later if we want to optimize. There's other coalescing we could do inside
// this function too (single call for one moduleID, single call for one ownerID, get
// multiple moduleIDs at once, multiple ownerIDs at once, etc). Lots of room for optimization.
func (a *moduleDataProvider) getModuleKeysForProtoCommits(
	ctx context.Context,
	registryHostname string,
	protoModuleCache *cache.Cache[string, *modulev1beta1.Module],
	protoOwnerCache *cache.Cache[string, *ownerv1beta1.Owner],
	protoCommits []*modulev1beta1.Commit,
) ([]bufmodule.ModuleKey, error) {
	moduleKeys := make([]bufmodule.ModuleKey, len(protoCommits))
	for i, protoCommit := range protoCommits {
		moduleKey, err := a.getModuleKeyForProtoCommit(
			ctx,
			registryHostname,
			protoModuleCache,
			protoOwnerCache,
			protoCommit,
		)
		if err != nil {
			return nil, err
		}
		moduleKeys[i] = moduleKey
	}
	return moduleKeys, nil
}

func (a *moduleDataProvider) getModuleKeyForProtoCommit(
	ctx context.Context,
	registryHostname string,
	protoModuleCache *cache.Cache[string, *modulev1beta1.Module],
	protoOwnerCache *cache.Cache[string, *ownerv1beta1.Owner],
	protoCommit *modulev1beta1.Commit,
) (bufmodule.ModuleKey, error) {
	protoModule, err := a.getProtoModuleForModuleID(ctx, registryHostname, protoModuleCache, protoCommit.ModuleId)
	if err != nil {
		return nil, err
	}
	protoOwner, err := a.getProtoOwnerForOwnerID(ctx, registryHostname, protoOwnerCache, protoCommit.OwnerId)
	if err != nil {
		return nil, err
	}
	var ownerName string
	switch {
	case protoOwner.GetUser() != nil:
		ownerName = protoOwner.GetUser().Name
	case protoOwner.GetOrganization() != nil:
		ownerName = protoOwner.GetOrganization().Name
	default:
		return nil, fmt.Errorf("proto Owner did not have a User or Organization: %v", protoOwner)
	}
	moduleFullName, err := bufmodule.NewModuleFullName(
		registryHostname,
		ownerName,
		protoModule.Name,
	)
	if err != nil {
		return nil, err
	}
	commitID, err := ProtoToCommitID(protoCommit.Id)
	if err != nil {
		return nil, err
	}
	return bufmodule.NewModuleKey(
		moduleFullName,
		commitID,
		func() (bufmodule.Digest, error) {
			return ProtoToDigest(protoCommit.Digest)
		},
	)
}

func (a *moduleDataProvider) getProtoModuleForModuleID(
	ctx context.Context,
	registryHostname string,
	protoModuleCache *cache.Cache[string, *modulev1beta1.Module],
	moduleID string,
) (*modulev1beta1.Module, error) {
	return protoModuleCache.GetOrAdd(
		registryHostname+"/"+moduleID,
		func() (*modulev1beta1.Module, error) {
			response, err := a.clientProvider.ModuleServiceClient(registryHostname).GetModules(
				ctx,
				connect.NewRequest(
					&modulev1beta1.GetModulesRequest{
						ModuleRefs: []*modulev1beta1.ModuleRef{
							{
								Value: &modulev1beta1.ModuleRef_Id{
									Id: moduleID,
								},
							},
						},
					},
				),
			)
			if err != nil {
				return nil, err
			}
			if len(response.Msg.Modules) != 1 {
				return nil, fmt.Errorf("expected 1 Module, got %d", len(response.Msg.Modules))
			}
			return response.Msg.Modules[0], nil
		},
	)
}

func (a *moduleDataProvider) getProtoOwnerForOwnerID(
	ctx context.Context,
	registryHostname string,
	protoOwnerCache *cache.Cache[string, *ownerv1beta1.Owner],
	ownerID string,
) (*ownerv1beta1.Owner, error) {
	return protoOwnerCache.GetOrAdd(
		registryHostname+"/"+ownerID,
		func() (*ownerv1beta1.Owner, error) {
			response, err := a.clientProvider.OwnerServiceClient(registryHostname).GetOwners(
				ctx,
				connect.NewRequest(
					&ownerv1beta1.GetOwnersRequest{
						OwnerRefs: []*ownerv1beta1.OwnerRef{
							{
								Value: &ownerv1beta1.OwnerRef_Id{
									Id: ownerID,
								},
							},
						},
					},
				),
			)
			if err != nil {
				return nil, err
			}
			if len(response.Msg.Owners) != 1 {
				return nil, fmt.Errorf("expected 1 Owner, got %d", len(response.Msg.Owners))
			}
			return response.Msg.Owners[0], nil
		},
	)
}

func getProtoCommitIDToCommitForProtoDownloadResponse(
	protoDownloadResponse *modulev1beta1.DownloadResponse,
) (map[string]*modulev1beta1.Commit, error) {
	return slicesext.ToUniqueValuesMapError(
		protoDownloadResponse.Commits,
		func(protoCommit *modulev1beta1.Commit) (string, error) {
			return protoCommit.Id, nil
		},
	)
}

func getProtoCommitIDToBucketForProtoDownloadResponse(
	protoDownloadResponse *modulev1beta1.DownloadResponse,
) (map[string]storage.ReadBucket, error) {
	protoCommitIDToBucket := make(map[string]storage.ReadBucket, len(protoDownloadResponse.Contents))
	for _, protoContent := range protoDownloadResponse.Contents {
		bucket, err := protoFilesToBucket(protoContent.Files)
		if err != nil {
			return nil, err
		}
		protoCommitIDToBucket[protoContent.CommitId] = bucket
	}
	return protoCommitIDToBucket, nil
}
