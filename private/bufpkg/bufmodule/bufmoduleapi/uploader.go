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

package bufmoduleapi

import (
	"context"
	"fmt"
	"time"

	federationv1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/legacy/federation/v1beta1"
	modulev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1beta1"
	ownerv1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/owner/v1beta1"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/gofrs/uuid/v5"
	"go.uber.org/zap"
)

// NewUploader returns a new Uploader for the given API client.
func NewUploader(
	logger *zap.Logger,
	clientProvider interface {
		bufapi.LegacyFederationUploadServiceClientProvider
		bufapi.ModuleServiceClientProvider
		bufapi.UploadServiceClientProvider
	},
) bufmodule.Uploader {
	return newUploader(logger, clientProvider)
}

// *** PRIVATE ***

type uploader struct {
	logger         *zap.Logger
	clientProvider interface {
		bufapi.LegacyFederationUploadServiceClientProvider
		bufapi.ModuleServiceClientProvider
		bufapi.UploadServiceClientProvider
	}
}

func newUploader(
	logger *zap.Logger,
	clientProvider interface {
		bufapi.LegacyFederationUploadServiceClientProvider
		bufapi.ModuleServiceClientProvider
		bufapi.UploadServiceClientProvider
	},
) *uploader {
	return &uploader{
		logger:         logger,
		clientProvider: clientProvider,
	}
}

func (a *uploader) Upload(
	ctx context.Context,
	moduleSet bufmodule.ModuleSet,
	options ...bufmodule.UploadOption,
) ([]bufmodule.Commit, error) {
	uploadOptions, err := bufmodule.NewUploadOptions(options)
	if err != nil {
		return nil, err
	}

	contentModules, err := bufmodule.ModuleSetTargetLocalModulesAndTransitiveLocalDeps(moduleSet)
	if err != nil {
		return nil, err
	}
	primaryRegistry, err := getSingleRegistryForContentModules(contentModules)
	if err != nil {
		return nil, err
	}

	if uploadOptions.CreateIfNotExist() {
		if err := a.createContentModulesIfNotExist(
			ctx,
			primaryRegistry,
			contentModules,
			uploadOptions.CreateModuleVisibility(),
		); err != nil {
			return nil, err
		}
	} else {
		if err := a.validateContentModulesExist(
			ctx,
			primaryRegistry,
			contentModules,
		); err != nil {
			return nil, err
		}
	}

	// While the API allows different labels per reference, we don't have a use case for this
	// in the CLI, so all references will have the same labels. We just pre-compute them now.
	protoScopedLabelRefs := slicesext.Map(
		slicesext.ToUniqueSorted(uploadOptions.Labels()),
		labelNameToProtoScopedLabelRef,
	)
	remoteDeps, err := bufmodule.RemoteDepsForModules(contentModules)
	if err != nil {
		return nil, err
	}

	// Maintains ordering, important for when we create bufmodule.Commit objects below.
	protoLegacyFederationUploadRequestContents, err := slicesext.MapError(
		contentModules,
		func(module bufmodule.Module) (*federationv1beta1.UploadRequest_Content, error) {
			return getProtoLegacyFederationUploadRequestContent(
				ctx,
				protoScopedLabelRefs,
				primaryRegistry,
				module,
			)
		},
	)
	if err != nil {
		return nil, err
	}
	protoLegacyFederationDepRefs, err := slicesext.MapError(
		remoteDeps,
		getProtoLegacyFederationUploadRequestDepRef,
	)
	if err != nil {
		return nil, err
	}

	// A sorted slice of unique registries for the RemoteDeps.
	remoteDepRegistries := slicesext.MapKeysToSortedSlice(
		// A map from registry to RemoteDeps for that reigsry.
		slicesext.ToValuesMap(
			remoteDeps,
			func(remoteDep bufmodule.RemoteDep) string {
				// We've already validated two or three times that ModuleFullName is present here.
				return remoteDep.ModuleFullName().Registry()
			},
		),
	)
	if err := validateDepRegistries(primaryRegistry, remoteDepRegistries); err != nil {
		return nil, err
	}

	var protoCommits []*modulev1beta1.Commit
	if len(remoteDepRegistries) > 0 && (len(remoteDepRegistries) > 1 || remoteDepRegistries[0] != primaryRegistry) {
		// If we have dependencies on other registries, or we have multiple registries we depend on, we have
		// to use legacy federation.
		response, err := a.clientProvider.LegacyFederationUploadServiceClient(primaryRegistry).Upload(
			ctx,
			connect.NewRequest(
				&federationv1beta1.UploadRequest{
					Contents: protoLegacyFederationUploadRequestContents,
					DepRefs:  protoLegacyFederationDepRefs,
				},
			),
		)
		if err != nil {
			return nil, err
		}
		protoCommits = response.Msg.Commits
	} else {
		// If we only have a single registry, invoke the new API endpoint that does not allow
		// for federation. Do this so that we can maintain federated API endpoint metrics.
		//
		// Maintains ordering, important for when we create bufmodule.Commit objects below.
		protoUploadRequestContents := slicesext.Map(
			protoLegacyFederationUploadRequestContents,
			protoLegacyFederationUploadRequestContentToProtoUploadRequestContent,
		)
		protoDepCommitIds := slicesext.Map(
			protoLegacyFederationDepRefs,
			func(protoLegacyFederationDepRef *federationv1beta1.UploadRequest_DepRef) string {
				return protoLegacyFederationDepRef.CommitId
			},
		)
		response, err := a.clientProvider.UploadServiceClient(primaryRegistry).Upload(
			ctx,
			connect.NewRequest(
				&modulev1beta1.UploadRequest{
					Contents:     protoUploadRequestContents,
					DepCommitIds: protoDepCommitIds,
				},
			),
		)
		if err != nil {
			return nil, err
		}
		protoCommits = response.Msg.Commits
	}

	if len(protoCommits) != len(protoLegacyFederationUploadRequestContents) {
		return nil, fmt.Errorf("expected %d Commits, got %d", len(protoLegacyFederationUploadRequestContents), len(protoCommits))
	}
	commits := make([]bufmodule.Commit, len(protoCommits))
	for i, protoCommit := range protoCommits {
		protoCommit := protoCommit
		// This is how we get the ModuleFullName without calling the ModuleService or OwnerService.
		//
		// We've maintained ordering throughout this function, so we can do this.
		// The API returns Commits in the same order as the Contents.
		moduleFullName := contentModules[i].ModuleFullName()
		commitID, err := uuid.FromString(protoCommit.Id)
		if err != nil {
			return nil, err
		}
		moduleKey, err := bufmodule.NewModuleKey(
			moduleFullName,
			commitID,
			func() (bufmodule.Digest, error) {
				return ProtoToDigest(protoCommit.Digest)
			},
		)
		if err != nil {
			return nil, err
		}
		commits[i] = bufmodule.NewCommit(
			moduleKey,
			func() (time.Time, error) {
				return protoCommit.CreateTime.AsTime(), nil
			},
		)
	}
	return commits, nil
}

func (a *uploader) createContentModulesIfNotExist(
	ctx context.Context,
	primaryRegistry string,
	contentModules []bufmodule.Module,
	createModuleVisibility bufmodule.ModuleVisibility,
) error {
	protoCreateModuleVisibility, err := moduleVisibilityToProto(createModuleVisibility)
	if err != nil {
		return err
	}
	if _, err := a.clientProvider.ModuleServiceClient(primaryRegistry).CreateModules(
		ctx,
		connect.NewRequest(
			&modulev1beta1.CreateModulesRequest{
				Values: slicesext.Map(
					contentModules,
					func(module bufmodule.Module) *modulev1beta1.CreateModulesRequest_Value {
						return &modulev1beta1.CreateModulesRequest_Value{
							OwnerRef: &ownerv1beta1.OwnerRef{
								Value: &ownerv1beta1.OwnerRef_Name{
									Name: module.ModuleFullName().Owner(),
								},
							},
							Name:       module.ModuleFullName().Name(),
							Visibility: protoCreateModuleVisibility,
						}
					},
				),
			},
		),
	); err != nil && connect.CodeOf(err) != connect.CodeAlreadyExists {
		return err
	}
	return nil
}

func (a *uploader) validateContentModulesExist(
	ctx context.Context,
	primaryRegistry string,
	contentModules []bufmodule.Module,
) error {
	_, err := a.clientProvider.ModuleServiceClient(primaryRegistry).GetModules(
		ctx,
		connect.NewRequest(
			&modulev1beta1.GetModulesRequest{
				ModuleRefs: slicesext.Map(
					contentModules,
					func(module bufmodule.Module) *modulev1beta1.ModuleRef {
						return &modulev1beta1.ModuleRef{
							Value: &modulev1beta1.ModuleRef_Name_{
								Name: &modulev1beta1.ModuleRef_Name{
									Owner:  module.ModuleFullName().Owner(),
									Module: module.ModuleFullName().Name(),
								},
							},
						}
					},
				),
			},
		),
	)
	return err
}

func getSingleRegistryForContentModules(contentModules []bufmodule.Module) (string, error) {
	var registry string
	for _, module := range contentModules {
		moduleFullName := module.ModuleFullName()
		if moduleFullName == nil {
			return "", newRequireModuleFullNameOnUploadError(module)
		}
		moduleRegistry := moduleFullName.Registry()
		if registry != "" && moduleRegistry != registry {
			// We don't allow the upload of content across multiple registries, but in the legacy federation
			// case, we DO allow for depending on other registries.
			return "", fmt.Errorf(
				"cannot upload content for multiple registries at once: %s, %s",
				registry,
				moduleRegistry,
			)
		}
		registry = moduleRegistry
	}
	return registry, nil
}

func getProtoLegacyFederationUploadRequestContent(
	ctx context.Context,
	protoScopedLabelRefs []*modulev1beta1.ScopedLabelRef,
	primaryRegistry string,
	module bufmodule.Module,
) (*federationv1beta1.UploadRequest_Content, error) {
	if !module.IsLocal() {
		return nil, syserror.New("expected local Module in getProtoLegacyFederationUploadRequestContent")
	}
	if module.ModuleFullName() == nil {
		return nil, syserror.Wrap(newRequireModuleFullNameOnUploadError(module))
	}
	if module.ModuleFullName().Registry() != primaryRegistry {
		// This should never happen - the upload Modules should already be verified above to come from one registry.
		return nil, syserror.Newf("attempting to upload content for registry other than %s in getProtoLegacyFederationUploadRequestContent", primaryRegistry)
	}

	protoFiles, err := bucketToProtoFiles(ctx, bufmodule.ModuleReadBucketToStorageReadBucket(module))
	if err != nil {
		return nil, err
	}
	v1BufYAMLObjectData, err := module.V1Beta1OrV1BufYAMLObjectData()
	if err != nil {
		return nil, err
	}
	v1BufLockObjectData, err := module.V1Beta1OrV1BufLockObjectData()
	if err != nil {
		return nil, err
	}

	return &federationv1beta1.UploadRequest_Content{
		Files:           protoFiles,
		ScopedLabelRefs: protoScopedLabelRefs,
		// TODO: We may end up synthesizing v1 buf.yamls/buf.locks on bufmodule.Module,
		// if we do, we should consider whether we should be sending them over, as the
		// backend may come to rely on this.
		V1BufYamlFile: objectDataToProtoFile(v1BufYAMLObjectData),
		V1BufLockFile: objectDataToProtoFile(v1BufLockObjectData),
		// TODO FUTURE: vcs_commit
	}, nil
}

func getProtoLegacyFederationUploadRequestDepRef(
	remoteDep bufmodule.RemoteDep,
) (*federationv1beta1.UploadRequest_DepRef, error) {
	if remoteDep.ModuleFullName() == nil {
		return nil, newRequireModuleFullNameOnUploadError(remoteDep)
	}
	depCommitID := remoteDep.CommitID()
	if depCommitID.IsNil() {
		return nil, syserror.Newf("did not have a commit ID for a remote module dependency %q", remoteDep.OpaqueID())
	}
	return &federationv1beta1.UploadRequest_DepRef{
		CommitId: depCommitID.String(),
		Registry: remoteDep.ModuleFullName().Registry(),
	}, nil
}

func newRequireModuleFullNameOnUploadError(module bufmodule.Module) error {
	// This error will likely actually go back to users.
	return fmt.Errorf("A name must be specified in buf.yaml for module %s for push.", module.OpaqueID())
}
