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
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
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

// ProtoToV1Digest converts the given proto Digest to a Digest.
//
// Validation is performed to ensure the DigestType is known, and the value
// is a valid digest value for the given DigestType.
func ProtoToV1Digest(protoDigest *modulev1.Digest) (bufmodule.Digest, error) {
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

// ProtoToV1Beta1Digest converts the given proto Digest to a Digest.
//
// Validation is performed to ensure the DigestType is known, and the value
// is a valid digest value for the given DigestType.
func ProtoToV1Beta1Digest(protoDigest *modulev1beta1.Digest) (bufmodule.Digest, error) {
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

func moduleVisibilityToProto(moduleVisibility bufmodule.ModuleVisibility) (modulev1.ModuleVisibility, error) {
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
		return 0, fmt.Errorf("unknown modulev1beta.DigestType: %v", protoDigestType)
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
		return 0, fmt.Errorf("unknown modulev1beta1beta.DigestType: %v", protoDigestType)
	}
	return digestType, nil
}

// It is assumed that the bucket is already filtered to just module files.
func bucketToV1ProtoFiles(ctx context.Context, bucket storage.ReadBucket) ([]*modulev1.File, error) {
	var protoFiles []*modulev1.File
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
				&modulev1.File{
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

func v1ProtoFilesToBucket(protoFiles []*modulev1.File) (storage.ReadBucket, error) {
	pathToData := make(map[string][]byte, len(protoFiles))
	for _, protoFile := range protoFiles {
		pathToData[protoFile.Path] = protoFile.Content
	}
	return storagemem.NewReadBucket(pathToData)
}

func v1beta1ProtoFilesToBucket(protoFiles []*modulev1beta1.File) (storage.ReadBucket, error) {
	pathToData := make(map[string][]byte, len(protoFiles))
	for _, protoFile := range protoFiles {
		pathToData[protoFile.Path] = protoFile.Content
	}
	return storagemem.NewReadBucket(pathToData)
}

func v1ProtoFileToObjectData(protoFile *modulev1.File) (bufmodule.ObjectData, error) {
	if protoFile == nil {
		return nil, nil
	}
	return bufmodule.NewObjectData(normalpath.Base(protoFile.Path), protoFile.Content)
}

func v1beta1ProtoFileToObjectData(protoFile *modulev1beta1.File) (bufmodule.ObjectData, error) {
	if protoFile == nil {
		return nil, nil
	}
	return bufmodule.NewObjectData(normalpath.Base(protoFile.Path), protoFile.Content)
}

func objectDataToV1ProtoFile(objectData bufmodule.ObjectData) *modulev1.File {
	if objectData == nil {
		return nil
	}
	return &modulev1.File{
		Path:    objectData.Name(),
		Content: objectData.Data(),
	}
}

func objectDataToV1Beta1ProtoFile(objectData bufmodule.ObjectData) *modulev1beta1.File {
	if objectData == nil {
		return nil
	}
	return &modulev1beta1.File{
		Path:    objectData.Name(),
		Content: objectData.Data(),
	}
}

func labelNameToV1ProtoScopedLabelRef(labelName string) *modulev1.ScopedLabelRef {
	return &modulev1.ScopedLabelRef{
		Value: &modulev1.ScopedLabelRef_Name{
			Name: labelName,
		},
	}
}

func labelNameToV1Beta1ProtoScopedLabelRef(labelName string) *modulev1beta1.ScopedLabelRef {
	return &modulev1beta1.ScopedLabelRef{
		Value: &modulev1beta1.ScopedLabelRef_Name{
			Name: labelName,
		},
	}
}

func v1ProtoGraphToV1Beta1ProtoGraph(
	registry string,
	protoGraph *modulev1.Graph,
) *modulev1beta1.Graph {
	v1Beta1ProtoGraph := &modulev1beta1.Graph{
		Commits: make([]*modulev1beta1.Graph_Commit, len(protoGraph.Commits)),
		Edges:   make([]*modulev1beta1.Graph_Edge, len(protoGraph.Edges)),
	}
	for i, protoCommit := range protoGraph.Commits {
		v1beta1ProtoGraph.Commits[i] = &modulev1beta1.Graph_Commit{
			Commit:   protoCommit,
			Registry: registry,
		}
	}
	for i, protoEdge := range protoGraph.Edges {
		v1beta1ProtoGraph.Edges[i] = &modulev1beta1.Graph_Edge{
			FromNode: &modulev1beta1.Graph_Node{
				CommitId: protoEdge.FromNode.CommitId,
				Registry: registry,
			},
			ToNode: &modulev1beta1.Graph_Node{
				CommitId: protoEdge.ToNode.CommitId,
				Registry: registry,
			},
		}
	}
	return v1beta1ProtoGraph
}

// We have to make sure this is updated if a field is added?
// TODO FUTURE: Can we automate this to make sure this is true?
func v1beta1ProtoUploadRequestContentToV1ProtoUploadRequestContent(
	v1beta1ProtoUploadRequestContent *modulev1beta1.UploadRequest_Content,
) *modulev1.UploadRequest_Content {
	return &modulev1.UploadRequest_Content{
		ModuleRef:        v1beta1ProtoUploadRequestContent.ModuleRef,
		Files:            v1beta1ProtoUploadRequestContent.Files,
		V1BufYamlFile:    v1beta1ProtoUploadRequestContent.V1BufYamlFile,
		V1BufLockFile:    v1beta1ProtoUploadRequestContent.V1BufLockFile,
		ScopedLabelRefs:  v1beta1ProtoUploadRequestContent.ScopedLabelRefs,
		SourceControlUrl: v1beta1ProtoUploadRequestContent.SourceControlUrl,
	}
}
