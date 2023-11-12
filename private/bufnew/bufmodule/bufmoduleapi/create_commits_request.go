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

	modulev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1beta1"
	storagev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/storage/v1beta1"
	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufcas"
)

// ModuleSetToProtoModuleNodesAndBlobs creates new
// *modulev1beta.CreateCommitsRequest_ModuleNodes and *storagev1beta1.Blobs for target Modules
// in the ModuleSet.
//
// This creates ModuleNodes and Blobs for all local targets, as well as their local dependencies.
// DepNodes are created for all remote dependencies.
// All Modules in the ModuleSet that will be pushed or are dependencies are required to have ModuleFullNames.
func ModuleSetToProtoModuleNodesAndBlobs(
	ctx context.Context,
	moduleSet bufmodule.ModuleSet,
) ([]*modulev1beta1.CreateCommitsRequest_ModuleNode, []*storagev1beta1.Blob, error) {
	opaqueIDToProtoModuleNode := make(map[string]*modulev1beta1.CreateCommitsRequest_ModuleNode)
	var blobs []bufcas.Blob
	for _, module := range moduleSet.Modules() {
		if !module.IsTarget() || !module.IsLocal() {
			// We only create ModuleNodes for local targets or their local dependencies.
			continue
		}
		moduleBlobs, err := addProtoModuleNodeAndGetBlobsForLocalModule(
			ctx,
			opaqueIDToProtoModuleNode,
			module,
		)
		if err != nil {
			return nil, nil, err
		}
		blobs = append(blobs, moduleBlobs...)
	}
	protoModuleNodes := make([]*modulev1beta1.CreateCommitsRequest_ModuleNode, 0, len(opaqueIDToProtoModuleNode))
	for _, protoModuleNode := range opaqueIDToProtoModuleNode {
		protoModuleNodes = append(protoModuleNodes, protoModuleNode)
	}
	blobSet, err := bufcas.NewBlobSet(blobs)
	if err != nil {
		return nil, nil, err
	}
	protoBlobs, err := bufcas.BlobSetToProtoBlobs(blobSet)
	if err != nil {
		return nil, nil, err
	}
	return protoModuleNodes, protoBlobs, nil
}

func addProtoModuleNodeAndGetBlobsForLocalModule(
	ctx context.Context,
	opaqueIDToProtoModuleNode map[string]*modulev1beta1.CreateCommitsRequest_ModuleNode,
	module bufmodule.Module,
) ([]bufcas.Blob, error) {
	if _, ok := opaqueIDToProtoModuleNode[module.OpaqueID()]; ok {
		// We've already processed this Module.
		return nil, nil
	}
	moduleFullName := module.ModuleFullName()
	if moduleFullName == nil {
		return nil, fmt.Errorf("module %s had no name, which is required", module.OpaqueID())
	}
	protoModuleRef := &modulev1beta1.ModuleRef{
		Value: &modulev1beta1.ModuleRef_Name_{
			Name: &modulev1beta1.ModuleRef_Name{
				Owner:  moduleFullName.Owner(),
				Module: moduleFullName.Name(),
			},
		},
	}
	fileSet, err := bufcas.NewFileSetForBucket(
		ctx,
		bufmodule.ModuleReadBucketToStorageReadBucket(
			module,
		),
	)
	blobs := fileSet.BlobSet().Blobs()
	protoFileNodes, err := bufcas.FileNodesToProto(fileSet.Manifest().FileNodes())
	if err != nil {
		return nil, err
	}
	moduleDeps, err := module.ModuleDeps()
	if err != nil {
		return nil, err
	}
	var protoDepNodes []*modulev1beta1.CreateCommitsRequest_DepNode
	for _, moduleDep := range moduleDeps {
		// TODO: Should we only add direct dependencies? Talk about API.
		if moduleDep.IsLocal() {
			moduleDepBlobs, err := addProtoModuleNodeAndGetBlobsForLocalModule(
				ctx,
				opaqueIDToProtoModuleNode,
				moduleDep,
			)
			if err != nil {
				return nil, err
			}
			blobs = append(blobs, moduleDepBlobs...)
		} else {
			moduleDepFullName := moduleDep.ModuleFullName()
			if moduleDepFullName == nil {
				return nil, fmt.Errorf("module %s had no name, which is required", moduleDep.OpaqueID())
			}
			protoDepModuleRef := &modulev1beta1.ModuleRef{
				Value: &modulev1beta1.ModuleRef_Name_{
					Name: &modulev1beta1.ModuleRef_Name{
						Owner:  moduleDepFullName.Owner(),
						Module: moduleDepFullName.Name(),
					},
				},
			}
			depDigest, err := moduleDep.Digest()
			if err != nil {
				return nil, err
			}
			protoDepDigest, err := bufcas.DigestToProto(depDigest)
			if err != nil {
				return nil, err
			}
			protoDepNodes = append(
				protoDepNodes,
				&modulev1beta1.CreateCommitsRequest_DepNode{
					ModuleRef: protoDepModuleRef,
					Digest:    protoDepDigest,
				},
			)
		}
	}
	opaqueIDToProtoModuleNode[module.OpaqueID()] = &modulev1beta1.CreateCommitsRequest_ModuleNode{
		ModuleRef: protoModuleRef,
		FileNodes: protoFileNodes,
		DepNodes:  protoDepNodes,
	}
	return blobs, nil
}
