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

	modulev1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1"
	modulev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1beta1"
	ownerv1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/owner/v1"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"go.uber.org/zap"
)

// NewUploader returns a new Uploader for the given API client.
func NewUploader(
	logger *zap.Logger,
	clientProvider interface {
		bufapi.V1ModuleServiceClientProvider
		bufapi.V1UploadServiceClientProvider
		bufapi.V1Beta1UploadServiceClientProvider
	},
	options ...UploaderOption,
) bufmodule.Uploader {
	return newUploader(logger, clientProvider, options...)
}

// UploaderOption is an option for a new Uploader.
type UploaderOption func(*uploader)

// UploaderWithPublicRegistry returns a new UploaderOption that specifies
// the hostname of the public registry. By default this is "buf.build", however in testing,
// this may be something else. This is needed to discern which which registry to make calls
// against in the case where there is >1 registries represented in the ModuleKeys - we always
// want to call the non-public registry.
func UploaderWithPublicRegistry(publicRegistry string) UploaderOption {
	return func(uploader *uploader) {
		if publicRegistry != "" {
			uploader.publicRegistry = publicRegistry
		}
	}
}

// *** PRIVATE ***

type uploader struct {
	logger         *zap.Logger
	clientProvider interface {
		bufapi.V1ModuleServiceClientProvider
		bufapi.V1UploadServiceClientProvider
		bufapi.V1Beta1UploadServiceClientProvider
	}
	publicRegistry string
}

func newUploader(
	logger *zap.Logger,
	clientProvider interface {
		bufapi.V1ModuleServiceClientProvider
		bufapi.V1UploadServiceClientProvider
		bufapi.V1Beta1UploadServiceClientProvider
	},
	options ...UploaderOption,
) *uploader {
	uploader := &uploader{
		logger:         logger,
		clientProvider: clientProvider,
		publicRegistry: defaultPublicRegistry,
	}
	for _, option := range options {
		option(uploader)
	}
	return uploader
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

	var modules []*modulev1.Module
	if uploadOptions.CreateIfNotExist() {
		// We must attempt to create each module one at a time, since CreateModules is atomic.
		// For example, if contentModules contains 3 modules, a, b, and c, where a and b both
		// already exist, calling CreateModules on all 3 at once will result in an `AlreadyExists`
		// and `c` will not be created.
		for _, contentModule := range contentModules {
			modulesResponse, err := a.createContentModuleIfNotExist(
				ctx,
				primaryRegistry,
				contentModule,
				uploadOptions.CreateModuleVisibility(),
			)
			if err != nil {
				return nil, err
			}
			modules = append(modules, modulesResponse...)
		}
	} else {
		modules, err = a.validateContentModulesExist(
			ctx,
			primaryRegistry,
			contentModules,
		)
		if err != nil {
			return nil, err
		}
	}

	var v1beta1ProtoUploadRequestContents []*modulev1beta1.UploadRequest_Content
	if len(uploadOptions.Labels()) > 0 {
		// While the API allows different labels per reference, we don't expose this through
		// the use of the `--label` flag, so all references will have the same labels.
		// We just pre-compute them now.
		v1beta1ProtoScopedLabelRefs := slicesext.Map(
			uploadOptions.Labels(),
			labelNameToV1Beta1ProtoScopedLabelRef,
		)
		// Maintains ordering, important for when we create bufmodule.Commit objects below.
		v1beta1ProtoUploadRequestContents, err = slicesext.MapError(
			contentModules,
			func(module bufmodule.Module) (*modulev1beta1.UploadRequest_Content, error) {
				return getV1Beta1ProtoUploadRequestContent(
					ctx,
					v1beta1ProtoScopedLabelRefs,
					primaryRegistry,
					module,
				)
			},
		)
		if err != nil {
			return nil, err
		}
	}
	if uploadOptions.BranchOrDraft() != "" {
		if len(contentModules) > 1 {
			return nil, fmt.Errorf("--branch and --draft are disallowed for use when pushing a workspace with more than one module")
		}
		// We know that there is only one module we are uploading contents for.
		v1beta1ProtoUploadRequestContent, err := getV1Beta1ProtoUploadRequestContent(
			ctx,
			[]*modulev1beta1.ScopedLabelRef{
				labelNameToV1Beta1ProtoScopedLabelRef(uploadOptions.BranchOrDraft()),
			},
			primaryRegistry,
			contentModules[0],
		)
		if err != nil {
			return nil, err
		}
		v1beta1ProtoUploadRequestContents = append(v1beta1ProtoUploadRequestContents, v1beta1ProtoUploadRequestContent)
	}
	if len(uploadOptions.Tags()) > 0 {
		if len(contentModules) > 1 {
			return nil, fmt.Errorf("--tag is disallowed for use when pushing a workspace with more than one module")
		}
		v1beta1ProtoScopedLabelRefs := slicesext.Map(
			uploadOptions.Tags(),
			labelNameToV1Beta1ProtoScopedLabelRef,
		)
		// Add the default label to this
		v1beta1ProtoScopedLabelRefs = append(
			v1beta1ProtoScopedLabelRefs,
			labelNameToV1Beta1ProtoScopedLabelRef(modules[0].DefaultLabelName),
		)
		// We know that there is only one module we are uploading contents for.
		v1beta1ProtoUploadRequestContent, err := getV1Beta1ProtoUploadRequestContent(
			ctx,
			v1beta1ProtoScopedLabelRefs,
			primaryRegistry,
			contentModules[0],
		)
		if err != nil {
			return nil, err
		}
		v1beta1ProtoUploadRequestContents = append(v1beta1ProtoUploadRequestContents, v1beta1ProtoUploadRequestContent)
	}

	remoteDeps, err := bufmodule.RemoteDepsForModules(contentModules)
	if err != nil {
		return nil, err
	}

	v1beta1ProtoUploadRequestDepRefs, err := slicesext.MapError(
		remoteDeps,
		remoteDepToV1Beta1ProtoUploadRequestDepRef,
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
	if err := validateDepRegistries(primaryRegistry, remoteDepRegistries, a.publicRegistry); err != nil {
		return nil, err
	}

	var universalProtoCommits []*universalProtoCommit
	if len(remoteDepRegistries) > 0 && (len(remoteDepRegistries) > 1 || remoteDepRegistries[0] != primaryRegistry) {
		// If we have dependencies on other registries, or we have multiple registries we depend on, we have
		// to use legacy federation.
		response, err := a.clientProvider.V1Beta1UploadServiceClient(primaryRegistry).Upload(
			ctx,
			connect.NewRequest(
				&modulev1beta1.UploadRequest{
					Contents: v1beta1ProtoUploadRequestContents,
					DepRefs:  v1beta1ProtoUploadRequestDepRefs,
				},
			),
		)
		if err != nil {
			return nil, err
		}
		universalProtoCommits, err = slicesext.MapError(response.Msg.Commits, newUniversalProtoCommitForV1Beta1)
		if err != nil {
			return nil, err
		}
	} else {
		// If we only have a single registry, invoke the new API endpoint that does not allow
		// for federation. Do this so that we can maintain federated API endpoint metrics.
		//
		// Maintains ordering, important for when we create bufmodule.Commit objects below.
		v1ProtoUploadRequestContents := slicesext.Map(
			v1beta1ProtoUploadRequestContents,
			v1beta1ProtoUploadRequestContentToV1ProtoUploadRequestContent,
		)
		protoDepCommitIds := slicesext.Map(
			v1beta1ProtoUploadRequestDepRefs,
			func(v1beta1ProtoDepRef *modulev1beta1.UploadRequest_DepRef) string {
				return v1beta1ProtoDepRef.CommitId
			},
		)
		response, err := a.clientProvider.V1UploadServiceClient(primaryRegistry).Upload(
			ctx,
			connect.NewRequest(
				&modulev1.UploadRequest{
					Contents:     v1ProtoUploadRequestContents,
					DepCommitIds: protoDepCommitIds,
				},
			),
		)
		if err != nil {
			return nil, err
		}
		universalProtoCommits, err = slicesext.MapError(response.Msg.Commits, newUniversalProtoCommitForV1)
		if err != nil {
			return nil, err
		}
	}

	if len(universalProtoCommits) != len(v1beta1ProtoUploadRequestContents) {
		return nil, fmt.Errorf("expected %d Commits, got %d", len(v1beta1ProtoUploadRequestContents), len(universalProtoCommits))
	}
	commits := make([]bufmodule.Commit, len(universalProtoCommits))
	for i, universalProtoCommit := range universalProtoCommits {
		universalProtoCommit := universalProtoCommit
		// This is how we get the ModuleFullName without calling the ModuleService or OwnerService.
		//
		// We've maintained ordering throughout this function, so we can do this.
		// The API returns Commits in the same order as the Contents.
		moduleFullName := contentModules[i].ModuleFullName()
		commitID, err := uuidutil.FromDashless(universalProtoCommit.ID)
		if err != nil {
			return nil, err
		}
		moduleKey, err := bufmodule.NewModuleKey(
			moduleFullName,
			commitID,
			func() (bufmodule.Digest, error) {
				return universalProtoCommit.Digest, nil
			},
		)
		if err != nil {
			return nil, err
		}
		commits[i] = bufmodule.NewCommit(
			moduleKey,
			func() (time.Time, error) {
				return universalProtoCommit.CreateTime, nil
			},
		)
	}
	return commits, nil
}

func (a *uploader) createContentModuleIfNotExist(
	ctx context.Context,
	primaryRegistry string,
	contentModule bufmodule.Module,
	createModuleVisibility bufmodule.ModuleVisibility,
) ([]*modulev1.Module, error) {
	v1ProtoCreateModuleVisibility, err := moduleVisibilityToV1Proto(createModuleVisibility)
	if err != nil {
		return nil, err
	}
	response, err := a.clientProvider.V1ModuleServiceClient(primaryRegistry).CreateModules(
		ctx,
		connect.NewRequest(
			&modulev1.CreateModulesRequest{
				Values: []*modulev1.CreateModulesRequest_Value{
					{
						OwnerRef: &ownerv1.OwnerRef{
							Value: &ownerv1.OwnerRef_Name{
								Name: contentModule.ModuleFullName().Owner(),
							},
						},
						Name:       contentModule.ModuleFullName().Name(),
						Visibility: v1ProtoCreateModuleVisibility,
					},
				},
			},
		),
	)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeAlreadyExists {
			// If a module already existed, then we check validate its contents.
			return a.validateContentModulesExist(ctx, primaryRegistry, []bufmodule.Module{contentModule})
		}
		return nil, err
	}
	// Otherwise we return the module we created
	return response.Msg.Modules, nil
}

func (a *uploader) validateContentModulesExist(
	ctx context.Context,
	primaryRegistry string,
	contentModules []bufmodule.Module,
) ([]*modulev1.Module, error) {
	response, err := a.clientProvider.V1ModuleServiceClient(primaryRegistry).GetModules(
		ctx,
		connect.NewRequest(
			&modulev1.GetModulesRequest{
				ModuleRefs: slicesext.Map(
					contentModules,
					func(module bufmodule.Module) *modulev1.ModuleRef {
						return &modulev1.ModuleRef{
							Value: &modulev1.ModuleRef_Name_{
								Name: &modulev1.ModuleRef_Name{
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
	if err != nil {
		return nil, err
	}
	return response.Msg.Modules, nil
}

func getV1Beta1ProtoUploadRequestContent(
	ctx context.Context,
	v1beta1ProtoScopedLabelRefs []*modulev1beta1.ScopedLabelRef,
	primaryRegistry string,
	module bufmodule.Module,
) (*modulev1beta1.UploadRequest_Content, error) {
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

	v1beta1ProtoFiles, err := bucketToV1Beta1ProtoFiles(ctx, bufmodule.ModuleReadBucketToStorageReadBucket(module))
	if err != nil {
		return nil, err
	}

	return &modulev1beta1.UploadRequest_Content{
		ModuleRef: &modulev1beta1.ModuleRef{
			Value: &modulev1beta1.ModuleRef_Name_{
				Name: &modulev1beta1.ModuleRef_Name{
					Owner:  module.ModuleFullName().Owner(),
					Module: module.ModuleFullName().Name(),
				},
			},
		},
		Files:           v1beta1ProtoFiles,
		ScopedLabelRefs: v1beta1ProtoScopedLabelRefs,
		// TODO FUTURE: vcs_commit
	}, nil
}

func remoteDepToV1Beta1ProtoUploadRequestDepRef(
	remoteDep bufmodule.RemoteDep,
) (*modulev1beta1.UploadRequest_DepRef, error) {
	if remoteDep.ModuleFullName() == nil {
		return nil, newRequireModuleFullNameOnUploadError(remoteDep)
	}
	depCommitID := remoteDep.CommitID()
	if depCommitID.IsNil() {
		return nil, syserror.Newf("did not have a commit ID for a remote module dependency %q", remoteDep.OpaqueID())
	}
	return &modulev1beta1.UploadRequest_DepRef{
		CommitId: uuidutil.ToDashless(depCommitID),
		Registry: remoteDep.ModuleFullName().Registry(),
	}, nil
}

func newRequireModuleFullNameOnUploadError(module bufmodule.Module) error {
	// This error will likely actually go back to users.
	return fmt.Errorf("A name must be specified in buf.yaml for module %s for push. All modules that are being pushed, and all of their dependencies that are part of the workspace, must have a name.", module.OpaqueID())
}
