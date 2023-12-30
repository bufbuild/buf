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
	"strings"

	modulev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1beta1"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

// NewUploadRequest creates a new UploadRequest for the given ModuleSet.
//
// Only target Modules will be added as references.
//
// Note that registry hostname is effectively stripped! This means that if you have multiple registry
// hostnames represented by Modules in the ModuleSet, *all* of the Modules (targets and dependencies) are
// referenced as if they are on the registry you call against.
//
// Right now, we error if the Modules do not all have the same registry. However, this may cause issues
// with legagy federation situations TODO.
//
// TODO: This function should either move directly into push, or the create helpers in push should
// move into this package.
func NewUploadRequest(
	ctx context.Context,
	moduleSet bufmodule.ModuleSet,
	options ...UploadRequestOption,
) (*modulev1beta1.UploadRequest, error) {
	uploadRequestOptions := newUploadRequestOptions()
	for _, option := range options {
		option(uploadRequestOptions)
	}

	modules := moduleSet.Modules()
	targetModules := bufmodule.ModuleSetTargetModules(moduleSet)

	// We check upfront if all modules have names, before contining onwards.
	for _, module := range modules {
		if module.ModuleFullName() == nil {
			return nil, newRequireModuleFullNameOnUploadError(module)
		}
	}
	// Validate we're all within one registry for now.
	if registries := slicesext.ToUniqueSorted(
		slicesext.Map(
			modules,
			func(module bufmodule.Module) string { return module.ModuleFullName().Registry() },
		),
	); len(registries) > 1 {
		// TODO: This messes up legacy federation.
		return nil, fmt.Errorf("multiple registries detected: %s", strings.Join(registries, ", "))
	}

	// While the API allows different labels per reference, we don't have a use case for this
	// in the CLI, so all references will have the same labels. We just pre-compute them now.
	protoScopedLabelRefs := slicesext.Map(
		slicesext.ToUniqueSorted(uploadRequestOptions.labels),
		labelNameToProtoScopedLabelRef,
	)

	// We only add references for target Modules.
	// TODO: We should add references for any module without a CommitID! See below!
	targetOpaqueIDToProtoModuleRef, err := getOpaqueIDToProtoModuleRef(targetModules)
	if err != nil {
		return nil, err
	}

	protoReferences, err := slicesext.MapError(
		targetModules,
		func(module bufmodule.Module) (*modulev1beta1.UploadRequest_Reference, error) {
			protoModuleRef, ok := targetOpaqueIDToProtoModuleRef[module.OpaqueID()]
			if !ok {
				return nil, syserror.Newf("no Module found for OpaqueID %q", module.OpaqueID())
			}
			// Includes transitive dependencies.
			// Sorted by OpaqueID.
			moduleDeps, err := module.ModuleDeps()
			if err != nil {
				return nil, err
			}
			depProtoResourceRefs := make([]*modulev1beta1.ResourceRef, 0, len(moduleDeps))
			for _, moduleDep := range moduleDeps {
				moduleFullName := moduleDep.ModuleFullName()
				if moduleFullName == nil {
					return nil, newRequireModuleFullNameOnUploadError(moduleDep)
				}
				if moduleDep.IsTarget() {
					// If the dependency is a target, it will have a corresponding Reference
					// and Content. As per the Upload documentation, we should only specify the
					// Module as part of the ResourceRef.
					depProtoResourceRefs = append(
						depProtoResourceRefs,
						&modulev1beta1.ResourceRef{
							Value: &modulev1beta1.ResourceRef_Name_{
								Name: &modulev1beta1.ResourceRef_Name{
									Owner:  moduleFullName.Owner(),
									Module: moduleFullName.Name(),
								},
							},
						},
					)
				} else {
					commitID := moduleDep.CommitID()
					if commitID == "" {
						// TODO: THIS IS A MAJOR TODO. We might NOT have commit IDs for other modules
						// in the workspace. In this case, we need to add their data to the upload.
						return nil, fmt.Errorf("did not have a commit ID for a non-target module dependency %q", moduleDep.OpaqueID())
					}
					protoCommitID, err := CommitIDToProto(commitID)
					if err != nil {
						return nil, err
					}
					depProtoResourceRefs = append(
						depProtoResourceRefs,
						&modulev1beta1.ResourceRef{
							Value: &modulev1beta1.ResourceRef_Id{
								Id: protoCommitID,
							},
						},
					)
				}
			}
			return &modulev1beta1.UploadRequest_Reference{
				ModuleRef:       protoModuleRef,
				DepResourceRefs: depProtoResourceRefs,
				ScopedLabelRefs: protoScopedLabelRefs,
				// TODO: vcs_commit
			}, nil
		},
	)
	if err != nil {
		return nil, err
	}

	protoContents, err := slicesext.MapError(
		targetModules,
		func(module bufmodule.Module) (*modulev1beta1.UploadRequest_Content, error) {
			protoModuleRef, ok := targetOpaqueIDToProtoModuleRef[module.OpaqueID()]
			if !ok {
				return nil, syserror.Newf("no Module found for OpaqueID %q", module.OpaqueID())
			}
			protoFiles, err := bucketToProtoFiles(ctx, bufmodule.ModuleReadBucketToStorageReadBucket(module))
			if err != nil {
				return nil, err
			}
			return &modulev1beta1.UploadRequest_Content{
				ModuleRef: protoModuleRef,
				Files:     protoFiles,
			}, nil
		},
	)
	if err != nil {
		return nil, err
	}

	return &modulev1beta1.UploadRequest{
		References: protoReferences,
		Contents:   protoContents,
	}, nil
}

// UploadRequestOption is an option for a new UploadRequest.
type UploadRequestOption func(*uploadRequestOptions)

// UploadRequestWithLabels returns a new UploadRequestOption that adds the given labels.
//
// This can be called multiple times. The unique result set of labels will be used.
func UploadRequestWithLabels(labels ...string) UploadRequestOption {
	return func(uploadRequestOptions *uploadRequestOptions) {
		uploadRequestOptions.labels = append(uploadRequestOptions.labels, labels...)
	}
}

// *** PRIVATE ***

func getOpaqueIDToProtoModuleRef(modules []bufmodule.Module) (map[string]*modulev1beta1.ModuleRef, error) {
	opaqueIDToProtoModuleRef := make(map[string]*modulev1beta1.ModuleRef, len(modules))
	for _, module := range modules {
		moduleFullName := module.ModuleFullName()
		if moduleFullName == nil {
			return nil, newRequireModuleFullNameOnUploadError(module)
		}
		opaqueIDToProtoModuleRef[module.OpaqueID()] = &modulev1beta1.ModuleRef{
			Value: &modulev1beta1.ModuleRef_Name_{
				Name: &modulev1beta1.ModuleRef_Name{
					// Note registry is not used here! See note on NewUploadRequest.
					Owner:  moduleFullName.Owner(),
					Module: moduleFullName.Name(),
				},
			},
		}
	}
	return opaqueIDToProtoModuleRef, nil
}

func newRequireModuleFullNameOnUploadError(module bufmodule.Module) error {
	// This error will likely actually go back to users.
	return fmt.Errorf("A name must be specified in buf.yaml for module %s for push.", module.OpaqueID())
}

type uploadRequestOptions struct {
	labels []string
}

func newUploadRequestOptions() *uploadRequestOptions {
	return &uploadRequestOptions{}
}
