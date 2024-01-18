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
	"strings"
	"time"

	modulev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1beta1"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

// Upload uploads the given ModuleSet.
//
// Only target Modules will be added as references.
//
// Note that registry hostname is effectively stripped! This means that if you have multiple registry
// hostnames represented by Modules in the ModuleSet, *all* of the Modules (targets and dependencies) are
// referenced as if they are on the registry you call against.
//
// Right now, we error if the Modules do not all have the same registry. However, this may cause issues
// with legagy federation situations TODO.
func Upload(
	ctx context.Context,
	clientProvider bufapi.UploadServiceClientProvider,
	moduleSet bufmodule.ModuleSet,
	options ...UploadOption,
) ([]bufmodule.Commit, error) {
	uploadOptions := newUploadOptions()
	for _, option := range options {
		option(uploadOptions)
	}

	modules := moduleSet.Modules()

	// We check upfront if all modules have names, before contining onwards.
	for _, module := range modules {
		if module.ModuleFullName() == nil {
			return nil, newRequireModuleFullNameOnUploadError(module)
		}
	}
	// Validate we're all within one registry for now.
	registries := slicesext.ToUniqueSorted(
		slicesext.Map(
			modules,
			func(module bufmodule.Module) string { return module.ModuleFullName().Registry() },
		),
	)
	if len(registries) > 1 {
		// TODO: This messes up legacy federation.
		return nil, fmt.Errorf("multiple registries detected: %s", strings.Join(registries, ", "))
	}
	registry := registries[0]

	// While the API allows different labels per reference, we don't have a use case for this
	// in the CLI, so all references will have the same labels. We just pre-compute them now.
	protoScopedLabelRefs := slicesext.Map(
		slicesext.ToUniqueSorted(uploadOptions.labels),
		labelNameToProtoScopedLabelRef,
	)

	opaqueIDToProtoModuleRef, err := getOpaqueIDToProtoModuleRef(modules)
	if err != nil {
		return nil, err
	}

	targetModules := bufmodule.ModuleSetTargetModules(moduleSet)
	protoContents, err := slicesext.MapError(
		targetModules,
		func(module bufmodule.Module) (*modulev1beta1.UploadRequest_Content, error) {
			protoModuleRef, ok := opaqueIDToProtoModuleRef[module.OpaqueID()]
			if !ok {
				return nil, syserror.Newf("no Module found for OpaqueID %q", module.OpaqueID())
			}
			// Includes transitive dependencies.
			// Sorted by OpaqueID.
			moduleDeps, err := module.ModuleDeps()
			if err != nil {
				return nil, err
			}
			protoDepRefs := make([]*modulev1beta1.UploadRequest_DepRef, 0, len(moduleDeps))
			for _, moduleDep := range moduleDeps {
				depModuleFullName := moduleDep.ModuleFullName()
				if depModuleFullName == nil {
					return nil, newRequireModuleFullNameOnUploadError(moduleDep)
				}
				depProtoModuleRef, ok := opaqueIDToProtoModuleRef[module.OpaqueID()]
				if !ok {
					return nil, syserror.Newf("no Module found for OpaqueID %q", moduleDep.OpaqueID())
				}
				var depProtoCommitID string
				// TODO: This should probably just become !moduleDep.IsLocal()!!!!
				if !moduleDep.IsTarget() {
					depCommitID := moduleDep.CommitID()
					if depCommitID == "" {
						// TODO: THIS IS A MAJOR TODO. We might NOT have commit IDs for other modules
						// in the workspace. In this case, we need to add their data to the upload.
						return nil, fmt.Errorf("did not have a commit ID for a non-target module dependency %q", moduleDep.OpaqueID())
					}
					depProtoCommitID, err = CommitIDToProto(depCommitID)
					if err != nil {
						return nil, err
					}
				}
				protoDepRefs = append(
					protoDepRefs,
					&modulev1beta1.UploadRequest_DepRef{
						ModuleRef: depProtoModuleRef,
						CommitId:  depProtoCommitID,
					},
				)
			}
			protoFiles, err := bucketToProtoFiles(ctx, bufmodule.ModuleReadBucketToStorageReadBucket(module))
			if err != nil {
				return nil, err
			}
			return &modulev1beta1.UploadRequest_Content{
				ModuleRef:       protoModuleRef,
				Files:           protoFiles,
				DepRefs:         protoDepRefs,
				ScopedLabelRefs: protoScopedLabelRefs,
				// TODO: vcs_commit
			}, nil
		},
	)
	if err != nil {
		return nil, err
	}
	response, err := clientProvider.UploadServiceClient(registry).Upload(
		ctx,
		connect.NewRequest(
			&modulev1beta1.UploadRequest{
				Contents: protoContents,
			},
		),
	)
	if err != nil {
		return nil, err
	}
	if len(response.Msg.Commits) != len(protoContents) {
		return nil, fmt.Errorf("expected %d Commits, got %d", len(protoContents), len(response.Msg.Commits))
	}
	commits := make([]bufmodule.Commit, len(response.Msg.Commits))
	for i, protoCommit := range response.Msg.Commits {
		targetModule := targetModules[i]
		commitID, err := ProtoToCommitID(protoCommit.Id)
		if err != nil {
			return nil, err
		}
		getDigest := func() (bufmodule.Digest, error) {
			return ProtoToDigest(protoCommit.Digest)
		}
		moduleKey, err := bufmodule.NewModuleKey(
			targetModule.ModuleFullName(),
			commitID,
			getDigest,
		)
		if err != nil {
			return nil, err
		}
		commits[i] = bufmodule.NewCommit(
			moduleKey,
			func() (time.Time, error) {
				return protoCommit.CreateTime.AsTime(), nil
			},
			// Since we use the same getDigest for ModuleKey, the "tamper-proofing" will
			// always return true. We might just get rid of the bufmodule.Commit type, or
			// re-work it a bit. It makes the most sense in the context of the CommitProvider.
			getDigest,
		)
	}
	return commits, nil
}

// UploadOption is an option for a new Upload.
type UploadOption func(*uploadOptions)

// UploadWithLabels returns a new UploadOption that adds the given labels.
//
// This can be called multiple times. The unique result set of labels will be used.
func UploadWithLabels(labels ...string) UploadOption {
	return func(uploadOptions *uploadOptions) {
		uploadOptions.labels = append(uploadOptions.labels, labels...)
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

type uploadOptions struct {
	labels []string
}

func newUploadOptions() *uploadOptions {
	return &uploadOptions{}
}
