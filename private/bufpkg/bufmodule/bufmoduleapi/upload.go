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
	"github.com/gofrs/uuid/v5"
)

// Upload uploads the given ModuleSet.
//
// Targeted local Modules, and their local transitive dependencies, will have content uploaded.
// These Modules will have Commits returned. The returned Commit order is deterministic, but should
// not be relied upon to be any specific ordering.
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

	// Validate we're all within one registry for now.
	registryMap := make(map[string]struct{})
	for _, module := range moduleSet.Modules() {
		if moduleFullName := module.ModuleFullName(); moduleFullName != nil {
			registryMap[moduleFullName.Registry()] = struct{}{}
		}
	}
	registries := slicesext.MapKeysToSortedSlice(registryMap)
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

	// Pre-compute these.
	opaqueIDToProtoModuleRef, err := getOpaqueIDToProtoModuleRef(moduleSet.Modules())
	if err != nil {
		return nil, err
	}

	// We do this as a map so we can check if we've already visited a given
	// Module before adding a new uploaContent in addUploadContentForLocalModule.
	opaqueIDToUploadContent := make(map[string]*uploadContent)
	targetedLocalModules := slicesext.Filter(
		moduleSet.Modules(),
		func(module bufmodule.Module) bool {
			return module.IsTarget() && module.IsLocal()
		},
	)
	for _, targetedLocalModule := range targetedLocalModules {
		if err := addUploadContentForLocalModule(
			ctx,
			targetedLocalModule,
			opaqueIDToUploadContent,
			opaqueIDToProtoModuleRef,
			protoScopedLabelRefs,
		); err != nil {
			return nil, err
		}
	}
	uploadContents := slicesext.MapValuesToSlice(opaqueIDToUploadContent)

	response, err := clientProvider.UploadServiceClient(registry).Upload(
		ctx,
		connect.NewRequest(
			&modulev1beta1.UploadRequest{
				Contents: slicesext.Map(
					uploadContents,
					func(uploadContent *uploadContent) *modulev1beta1.UploadRequest_Content {
						return uploadContent.protoUploadRequestContent
					},
				),
			},
		),
	)
	if err != nil {
		return nil, err
	}
	if len(response.Msg.Commits) != len(uploadContents) {
		return nil, fmt.Errorf("expected %d Commits, got %d", len(uploadContents), len(response.Msg.Commits))
	}
	commits := make([]bufmodule.Commit, len(response.Msg.Commits))
	for i, protoCommit := range response.Msg.Commits {
		// This is how we get the ModuleFullName without calling the ModuleService or OwnerService.
		moduleFullName := uploadContents[i].moduleFullName
		commitID, err := uuid.FromString(protoCommit.Id)
		if err != nil {
			return nil, err
		}
		getDigest := func() (bufmodule.Digest, error) {
			return ProtoToDigest(protoCommit.Digest)
		}
		moduleKey, err := bufmodule.NewModuleKey(
			moduleFullName,
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

// Ignores any modules without ModuleFullNames.
func getOpaqueIDToProtoModuleRef(modules []bufmodule.Module) (map[string]*modulev1beta1.ModuleRef, error) {
	opaqueIDToProtoModuleRef := make(map[string]*modulev1beta1.ModuleRef, len(modules))
	for _, module := range modules {
		moduleFullName := module.ModuleFullName()
		if moduleFullName == nil {
			continue
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

// addUploadContentForLocalModule adds the Module as uploadContent to the opaqueIDToUploadContent map.
//
// The Module (which is assumed to  be local) and all of its transitive local dependencies are added
// to the map IF they have not already been added.
//
// This function is recursive.
func addUploadContentForLocalModule(
	ctx context.Context,
	module bufmodule.Module,
	// This is the map to fill up.
	opaqueIDToUploadContent map[string]*uploadContent,
	// This map is already populated.
	opaqueIDToProtoModuleRef map[string]*modulev1beta1.ModuleRef,
	// This slice is already populated.
	protoScopedLabelRefs []*modulev1beta1.ScopedLabelRef,
) error {
	if _, ok := opaqueIDToUploadContent[module.OpaqueID()]; ok {
		// We've already added this module.
		return nil
	}

	if !module.IsLocal() {
		return syserror.New("expected local Module in addUploadContentForLocalModule")
	}
	if module.ModuleFullName() == nil {
		// All local modules that will be pushed need a ModuleFullName.
		return newRequireModuleFullNameOnUploadError(module)
	}

	protoModuleRef, ok := opaqueIDToProtoModuleRef[module.OpaqueID()]
	if !ok {
		return syserror.Newf("no Module found for OpaqueID %q in opaqueIDToProtoModuleRef", module.OpaqueID())
	}

	// Includes transitive dependencies.
	// Sorted by OpaqueID.
	moduleDeps, err := module.ModuleDeps()
	if err != nil {
		return err
	}
	protoDepRefs := make([]*modulev1beta1.UploadRequest_DepRef, 0, len(moduleDeps))
	for _, moduleDep := range moduleDeps {
		if moduleDep.ModuleFullName() == nil {
			// All local modules that will be pushed need a ModuleFullName.
			return newRequireModuleFullNameOnUploadError(moduleDep)
		}
		depProtoModuleRef, ok := opaqueIDToProtoModuleRef[moduleDep.OpaqueID()]
		if !ok {
			return syserror.Newf("no Module found for OpaqueID %q in opaqueIDToProtoModuleRef", moduleDep.OpaqueID())
		}
		if moduleDep.IsLocal() {
			// If the dependency is local, add it to the map if it hasn't already been added,
			// and add it as a DepRef with no Commit ID.
			if err := addUploadContentForLocalModule(
				ctx,
				moduleDep,
				opaqueIDToUploadContent,
				opaqueIDToProtoModuleRef,
				protoScopedLabelRefs,
			); err != nil {
				return err
			}
			protoDepRefs = append(
				protoDepRefs,
				&modulev1beta1.UploadRequest_DepRef{
					ModuleRef: depProtoModuleRef,
				},
			)
		} else {
			// If the dependency is remote, add it as a dep ref.
			depCommitID := moduleDep.CommitID()
			if depCommitID.IsNil() {
				return syserror.Newf("did not have a commit ID for a remote module dependency %q", moduleDep.OpaqueID())
			}
			protoDepRefs = append(
				protoDepRefs,
				&modulev1beta1.UploadRequest_DepRef{
					ModuleRef: depProtoModuleRef,
					CommitId:  depCommitID.String(),
				},
			)
		}
	}
	protoFiles, err := bucketToProtoFiles(ctx, bufmodule.ModuleReadBucketToStorageReadBucket(module))
	if err != nil {
		return err
	}

	opaqueIDToUploadContent[module.OpaqueID()] = &uploadContent{
		moduleFullName: module.ModuleFullName(),
		protoUploadRequestContent: &modulev1beta1.UploadRequest_Content{
			ModuleRef:       protoModuleRef,
			Files:           protoFiles,
			DepRefs:         protoDepRefs,
			ScopedLabelRefs: protoScopedLabelRefs,
			// TODO: We may end up synthesizing v1 buf.yamls/buf.locks on bufmodule.Module,
			// if we do, we should consider whether we should be sending them over, as the
			// backend may come to rely on this.
			V1BufYamlFile: objectDataToProtoFile(module.V1Beta1OrV1BufYAMLObjectData()),
			V1BufLockFile: objectDataToProtoFile(module.V1Beta1OrV1BufLockObjectData()),
			// TODO: vcs_commit
		},
	}
	return nil
}

func newRequireModuleFullNameOnUploadError(module bufmodule.Module) error {
	// This error will likely actually go back to users.
	return fmt.Errorf("A name must be specified in buf.yaml for module %s for push.", module.OpaqueID())
}

// uploadContent is just the pair of ModuleFullName and UploadRequest_Content.
//
// We know the ModuleFullName at construction, but we need to keep track of it alongside
// the content we upload so that we can re-associate it when we get back a response from Upload.
//
// We add Modules to the uploaded content in a recursive manner (for deps), and its difficult
// to keep track of indices. This is as good a way as any. If we didn't do this, we would have
// to reconstruct a ModuleFullName from a proto Commit, which would mean calls out to the
// ModuleService and OwnerService that we don't have to make.
type uploadContent struct {
	moduleFullName            bufmodule.ModuleFullName
	protoUploadRequestContent *modulev1beta1.UploadRequest_Content
}

type uploadOptions struct {
	labels []string
}

func newUploadOptions() *uploadOptions {
	return &uploadOptions{}
}
