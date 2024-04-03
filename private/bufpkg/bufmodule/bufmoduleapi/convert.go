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
	"io"

	modulev1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1"
	modulev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1beta1"
	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"github.com/gofrs/uuid/v5"
)

var (
	digestTypeToV1ProtoDigestType = map[bufmodule.DigestType]modulev1.DigestType{
		bufmodule.DigestTypeB5: modulev1.DigestType_DIGEST_TYPE_B5,
	}
	v1ProtoDigestTypeToDigestType = map[modulev1.DigestType]bufmodule.DigestType{
		modulev1.DigestType_DIGEST_TYPE_B5: bufmodule.DigestTypeB5,
	}
	digestTypeToV1Beta1ProtoDigestType = map[bufmodule.DigestType]modulev1beta1.DigestType{
		bufmodule.DigestTypeB4: modulev1beta1.DigestType_DIGEST_TYPE_B4,
		bufmodule.DigestTypeB5: modulev1beta1.DigestType_DIGEST_TYPE_B5,
	}
	v1beta1ProtoDigestTypeToDigestType = map[modulev1beta1.DigestType]bufmodule.DigestType{
		modulev1beta1.DigestType_DIGEST_TYPE_B4: bufmodule.DigestTypeB4,
		modulev1beta1.DigestType_DIGEST_TYPE_B5: bufmodule.DigestTypeB5,
	}
	v1ProtoDigestTypeToV1Beta1ProtoDigestType = map[modulev1.DigestType]modulev1beta1.DigestType{
		modulev1.DigestType_DIGEST_TYPE_B5: modulev1beta1.DigestType_DIGEST_TYPE_B5,
	}
)

// DigestToV1Proto converts the given Digest to a proto Digest.
func DigestToV1Proto(digest bufmodule.Digest) (*modulev1.Digest, error) {
	protoDigestType, err := digestTypeToV1Proto(digest.Type())
	if err != nil {
		return nil, err
	}
	return &modulev1.Digest{
		Type:  protoDigestType,
		Value: digest.Value(),
	}, nil
}

// V1ProtoToDigest converts the given proto Digest to a Digest.
//
// Validation is performed to ensure the DigestType is known, and the value
// is a valid digest value for the given DigestType.
func V1ProtoToDigest(protoDigest *modulev1.Digest) (bufmodule.Digest, error) {
	digestType, err := v1ProtoToDigestType(protoDigest.Type)
	if err != nil {
		return nil, err
	}
	bufcasDigest, err := bufcas.NewDigest(protoDigest.Value)
	if err != nil {
		return nil, err
	}
	return bufmodule.NewDigest(digestType, bufcasDigest)
}

// DigestToV1Beta1Proto converts the given Digest to a proto Digest.
func DigestToV1Beta1Proto(digest bufmodule.Digest) (*modulev1beta1.Digest, error) {
	protoDigestType, err := digestTypeToV1Beta1Proto(digest.Type())
	if err != nil {
		return nil, err
	}
	return &modulev1beta1.Digest{
		Type:  protoDigestType,
		Value: digest.Value(),
	}, nil
}

// V1Beta1ProtoToDigest converts the given proto Digest to a Digest.
//
// Validation is performed to ensure the DigestType is known, and the value
// is a valid digest value for the given DigestType.
func V1Beta1ProtoToDigest(protoDigest *modulev1beta1.Digest) (bufmodule.Digest, error) {
	digestType, err := v1beta1ProtoToDigestType(protoDigest.Type)
	if err != nil {
		return nil, err
	}
	bufcasDigest, err := bufcas.NewDigest(protoDigest.Value)
	if err != nil {
		return nil, err
	}
	return bufmodule.NewDigest(digestType, bufcasDigest)
}

// *** PRIVATE ***

func moduleVisibilityToV1Proto(moduleVisibility bufmodule.ModuleVisibility) (modulev1.ModuleVisibility, error) {
	switch moduleVisibility {
	case bufmodule.ModuleVisibilityPublic:
		return modulev1.ModuleVisibility_MODULE_VISIBILITY_PUBLIC, nil
	case bufmodule.ModuleVisibilityPrivate:
		return modulev1.ModuleVisibility_MODULE_VISIBILITY_PRIVATE, nil
	default:
		return 0, fmt.Errorf("unknown ModuleVisibility: %v", moduleVisibility)
	}
}

func digestTypeToV1Proto(digestType bufmodule.DigestType) (modulev1.DigestType, error) {
	protoDigestType, ok := digestTypeToV1ProtoDigestType[digestType]
	// Technically we have already done this validation but just to be safe.
	if !ok {
		return 0, fmt.Errorf("unknown DigestType: %v", digestType)
	}
	return protoDigestType, nil
}

func v1ProtoToDigestType(protoDigestType modulev1.DigestType) (bufmodule.DigestType, error) {
	digestType, ok := v1ProtoDigestTypeToDigestType[protoDigestType]
	if !ok {
		return 0, fmt.Errorf("unknown modulev1.DigestType: %v", protoDigestType)
	}
	return digestType, nil
}

func digestTypeToV1Beta1Proto(digestType bufmodule.DigestType) (modulev1beta1.DigestType, error) {
	protoDigestType, ok := digestTypeToV1Beta1ProtoDigestType[digestType]
	// Technically we have already done this validation but just to be safe.
	if !ok {
		return 0, fmt.Errorf("unknown DigestType: %v", digestType)
	}
	return protoDigestType, nil
}

func v1beta1ProtoToDigestType(protoDigestType modulev1beta1.DigestType) (bufmodule.DigestType, error) {
	digestType, ok := v1beta1ProtoDigestTypeToDigestType[protoDigestType]
	if !ok {
		return 0, fmt.Errorf("unknown modulev1beta1.DigestType: %v", protoDigestType)
	}
	return digestType, nil
}

// It is assumed that the bucket is already filtered to just module files.
func bucketToV1Beta1ProtoFiles(ctx context.Context, bucket storage.ReadBucket) ([]*modulev1beta1.File, error) {
	var protoFiles []*modulev1beta1.File
	if err := storage.WalkReadObjects(
		ctx,
		bucket,
		"",
		func(readObject storage.ReadObject) error {
			data, err := io.ReadAll(readObject)
			if err != nil {
				return err
			}
			protoFiles = append(
				protoFiles,
				&modulev1beta1.File{
					Path:    readObject.Path(),
					Content: data,
				},
			)
			return nil
		},
	); err != nil {
		return nil, err
	}
	return protoFiles, nil
}

func labelNameToV1Beta1ProtoScopedLabelRef(labelName string) *modulev1beta1.ScopedLabelRef {
	return &modulev1beta1.ScopedLabelRef{
		Value: &modulev1beta1.ScopedLabelRef_Name{
			Name: labelName,
		},
	}
}

func commitIDToV1ProtoResourceRef(commitID uuid.UUID) *modulev1.ResourceRef {
	return &modulev1.ResourceRef{
		Value: &modulev1.ResourceRef_Id{
			Id: uuidutil.ToDashless(commitID),
		},
	}
}

func commitIDsToV1ProtoResourceRefs(commitIDs []uuid.UUID) []*modulev1.ResourceRef {
	return slicesext.Map(commitIDs, commitIDToV1ProtoResourceRef)
}

func commitIDToV1Beta1ProtoResourceRef(commitID uuid.UUID) *modulev1beta1.ResourceRef {
	return &modulev1beta1.ResourceRef{
		Value: &modulev1beta1.ResourceRef_Id{
			Id: uuidutil.ToDashless(commitID),
		},
	}
}

func commitIDsToV1Beta1ProtoResourceRefs(commitIDs []uuid.UUID) []*modulev1beta1.ResourceRef {
	return slicesext.Map(commitIDs, commitIDToV1Beta1ProtoResourceRef)
}

func moduleRefToV1ProtoResourceRef(moduleRef bufmodule.ModuleRef) *modulev1.ResourceRef {
	return &modulev1.ResourceRef{
		Value: &modulev1.ResourceRef_Name_{
			Name: &modulev1.ResourceRef_Name{
				Owner:  moduleRef.ModuleFullName().Owner(),
				Module: moduleRef.ModuleFullName().Name(),
				Child: &modulev1.ResourceRef_Name_Ref{
					Ref: moduleRef.Ref(),
				},
			},
		},
	}
}

func moduleRefsToV1ProtoResourceRefs(moduleRefs []bufmodule.ModuleRef) []*modulev1.ResourceRef {
	return slicesext.Map(moduleRefs, moduleRefToV1ProtoResourceRef)
}

func moduleRefToV1Beta1ProtoResourceRef(moduleRef bufmodule.ModuleRef) *modulev1beta1.ResourceRef {
	return &modulev1beta1.ResourceRef{
		Value: &modulev1beta1.ResourceRef_Name_{
			Name: &modulev1beta1.ResourceRef_Name{
				Owner:  moduleRef.ModuleFullName().Owner(),
				Module: moduleRef.ModuleFullName().Name(),
				Child: &modulev1beta1.ResourceRef_Name_Ref{
					Ref: moduleRef.Ref(),
				},
			},
		},
	}
}

func moduleRefsToV1Beta1ProtoResourceRefs(moduleRefs []bufmodule.ModuleRef) []*modulev1beta1.ResourceRef {
	return slicesext.Map(moduleRefs, moduleRefToV1Beta1ProtoResourceRef)
}

// We have to make sure all the below is updated if a field is added.
// This is enforced via exhaustruct using golangci-lint.
// Search .golangci.yml for convert.go to see where this is enabled.

func v1ProtoDigestToV1Beta1ProtoDigest(
	v1ProtoDigest *modulev1.Digest,
) (*modulev1beta1.Digest, error) {
	if v1ProtoDigest == nil {
		return nil, nil
	}
	v1beta1ProtoDigestType, ok := v1ProtoDigestTypeToV1Beta1ProtoDigestType[v1ProtoDigest.Type]
	if !ok {
		return nil, fmt.Errorf("unknown modulev1.DigestType: %v", v1ProtoDigest.Type)
	}
	return &modulev1beta1.Digest{
		Type:  v1beta1ProtoDigestType,
		Value: v1ProtoDigest.Value,
	}, nil
}

func v1ProtoCommitToV1Beta1ProtoCommit(
	v1ProtoCommit *modulev1.Commit,
) (*modulev1beta1.Commit, error) {
	if v1ProtoCommit == nil {
		return nil, nil
	}
	v1beta1ProtoDigest, err := v1ProtoDigestToV1Beta1ProtoDigest(v1ProtoCommit.Digest)
	if err != nil {
		return nil, err
	}
	return &modulev1beta1.Commit{
		Id:               v1ProtoCommit.Id,
		CreateTime:       v1ProtoCommit.CreateTime,
		OwnerId:          v1ProtoCommit.OwnerId,
		ModuleId:         v1ProtoCommit.ModuleId,
		Digest:           v1beta1ProtoDigest,
		CreatedByUserId:  v1ProtoCommit.CreatedByUserId,
		SourceControlUrl: v1ProtoCommit.SourceControlUrl,
	}, nil
}

func v1ProtoGraphToV1Beta1ProtoGraph(
	registry string,
	v1ProtoGraph *modulev1.Graph,
) (*modulev1beta1.Graph, error) {
	if v1ProtoGraph == nil {
		return nil, nil
	}
	v1beta1ProtoGraph := &modulev1beta1.Graph{
		Commits: make([]*modulev1beta1.Graph_Commit, len(v1ProtoGraph.Commits)),
		Edges:   make([]*modulev1beta1.Graph_Edge, len(v1ProtoGraph.Edges)),
	}
	for i, v1ProtoCommit := range v1ProtoGraph.Commits {
		v1beta1ProtoCommit, err := v1ProtoCommitToV1Beta1ProtoCommit(v1ProtoCommit)
		if err != nil {
			return nil, err
		}
		v1beta1ProtoGraph.Commits[i] = &modulev1beta1.Graph_Commit{
			Commit:   v1beta1ProtoCommit,
			Registry: registry,
		}
	}
	for i, v1ProtoEdge := range v1ProtoGraph.Edges {
		v1beta1ProtoGraph.Edges[i] = &modulev1beta1.Graph_Edge{
			FromNode: &modulev1beta1.Graph_Node{
				CommitId: v1ProtoEdge.FromNode.CommitId,
				Registry: registry,
			},
			ToNode: &modulev1beta1.Graph_Node{
				CommitId: v1ProtoEdge.ToNode.CommitId,
				Registry: registry,
			},
		}
	}
	return v1beta1ProtoGraph, nil
}

func v1beta1ProtoModuleRefToV1ProtoModuleRef(
	v1beta1ProtoModuleRef *modulev1beta1.ModuleRef,
) *modulev1.ModuleRef {
	if v1beta1ProtoModuleRef == nil {
		return nil
	}
	if id := v1beta1ProtoModuleRef.GetId(); id != "" {
		return &modulev1.ModuleRef{
			Value: &modulev1.ModuleRef_Id{
				Id: id,
			},
		}
	}
	if name := v1beta1ProtoModuleRef.GetName(); name != nil {
		return &modulev1.ModuleRef{
			Value: &modulev1.ModuleRef_Name_{
				Name: &modulev1.ModuleRef_Name{
					Owner:  name.Owner,
					Module: name.Module,
				},
			},
		}
	}
	return nil
}

func v1beta1ProtoFileToV1ProtoFile(
	v1beta1ProtoFile *modulev1beta1.File,
) *modulev1.File {
	if v1beta1ProtoFile == nil {
		return nil
	}
	return &modulev1.File{
		Path:    v1beta1ProtoFile.Path,
		Content: v1beta1ProtoFile.Content,
	}
}

func v1beta1ProtoFilesToV1ProtoFiles(
	v1beta1ProtoFiles []*modulev1beta1.File,
) []*modulev1.File {
	return slicesext.Map(v1beta1ProtoFiles, v1beta1ProtoFileToV1ProtoFile)
}

func v1beta1ProtoScopedLabelRefToV1ProtoScopedLabelRef(
	v1beta1ProtoScopedLabelRef *modulev1beta1.ScopedLabelRef,
) *modulev1.ScopedLabelRef {
	if v1beta1ProtoScopedLabelRef == nil {
		return nil
	}
	if id := v1beta1ProtoScopedLabelRef.GetId(); id != "" {
		return &modulev1.ScopedLabelRef{
			Value: &modulev1.ScopedLabelRef_Id{
				Id: id,
			},
		}
	}
	if name := v1beta1ProtoScopedLabelRef.GetName(); name != "" {
		return &modulev1.ScopedLabelRef{
			Value: &modulev1.ScopedLabelRef_Name{
				Name: name,
			},
		}
	}
	return nil
}

func v1beta1ProtoScopedLabelRefsToV1ProtoScopedLabelRefs(
	v1beta1ProtoScopedLabelRefs []*modulev1beta1.ScopedLabelRef,
) []*modulev1.ScopedLabelRef {
	return slicesext.Map(v1beta1ProtoScopedLabelRefs, v1beta1ProtoScopedLabelRefToV1ProtoScopedLabelRef)
}

func v1beta1ProtoUploadRequestContentToV1ProtoUploadRequestContent(
	v1beta1ProtoUploadRequestContent *modulev1beta1.UploadRequest_Content,
) *modulev1.UploadRequest_Content {
	return &modulev1.UploadRequest_Content{
		ModuleRef:        v1beta1ProtoModuleRefToV1ProtoModuleRef(v1beta1ProtoUploadRequestContent.ModuleRef),
		Files:            v1beta1ProtoFilesToV1ProtoFiles(v1beta1ProtoUploadRequestContent.Files),
		ScopedLabelRefs:  v1beta1ProtoScopedLabelRefsToV1ProtoScopedLabelRefs(v1beta1ProtoUploadRequestContent.ScopedLabelRefs),
		SourceControlUrl: v1beta1ProtoUploadRequestContent.SourceControlUrl,
	}
}
