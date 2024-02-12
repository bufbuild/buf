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

	federationv1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/legacy/federation/v1beta1"
	modulev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1beta1"
	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

var (
	digestTypeToProtoDigestType = map[bufmodule.DigestType]modulev1beta1.DigestType{
		bufmodule.DigestTypeB4: modulev1beta1.DigestType_DIGEST_TYPE_B4,
		bufmodule.DigestTypeB5: modulev1beta1.DigestType_DIGEST_TYPE_B5,
	}
	protoDigestTypeToDigestType = map[modulev1beta1.DigestType]bufmodule.DigestType{
		modulev1beta1.DigestType_DIGEST_TYPE_B4: bufmodule.DigestTypeB4,
		modulev1beta1.DigestType_DIGEST_TYPE_B5: bufmodule.DigestTypeB5,
	}
)

// ParseModuleVisibility parses the ModuleVisibility from the string.
func ParseModuleVisibility(s string) (modulev1beta1.ModuleVisibility, error) {
	switch s {
	case "public":
		return modulev1beta1.ModuleVisibility_MODULE_VISIBILITY_PUBLIC, nil
	case "private":
		return modulev1beta1.ModuleVisibility_MODULE_VISIBILITY_PRIVATE, nil
	default:
		return 0, fmt.Errorf("unknown visibility: %q", s)
	}
}

// DigestToProto converts the given Digest to a proto Digest.
func DigestToProto(digest bufmodule.Digest) (*modulev1beta1.Digest, error) {
	protoDigestType, err := digestTypeToProto(digest.Type())
	if err != nil {
		return nil, err
	}
	return &modulev1beta1.Digest{
		Type:  protoDigestType,
		Value: digest.Value(),
	}, nil
}

// ProtoToDigest converts the given proto Digest to a Digest.
//
// Validation is performed to ensure the DigestType is known, and the value
// is a valid digest value for the given DigestType.
func ProtoToDigest(protoDigest *modulev1beta1.Digest) (bufmodule.Digest, error) {
	digestType, err := protoToDigestType(protoDigest.Type)
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

func digestTypeToProto(digestType bufmodule.DigestType) (modulev1beta1.DigestType, error) {
	protoDigestType, ok := digestTypeToProtoDigestType[digestType]
	// Technically we have already done this validation but just to be safe.
	if !ok {
		return 0, fmt.Errorf("unknown DigestType: %v", digestType)
	}
	return protoDigestType, nil
}

func protoToDigestType(protoDigestType modulev1beta1.DigestType) (bufmodule.DigestType, error) {
	digestType, ok := protoDigestTypeToDigestType[protoDigestType]
	if !ok {
		return 0, fmt.Errorf("unknown modulev1beta.DigestType: %v", protoDigestType)
	}
	return digestType, nil
}

// It is assumed that the bucket is already filtered to just module files.
func bucketToProtoFiles(ctx context.Context, bucket storage.ReadBucket) ([]*modulev1beta1.File, error) {
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

func protoFilesToBucket(protoFiles []*modulev1beta1.File) (storage.ReadBucket, error) {
	pathToData := make(map[string][]byte, len(protoFiles))
	for _, protoFile := range protoFiles {
		pathToData[protoFile.Path] = protoFile.Content
	}
	return storagemem.NewReadBucket(pathToData)
}

func protoFileToObjectData(protoFile *modulev1beta1.File) (bufmodule.ObjectData, error) {
	if protoFile == nil {
		return nil, nil
	}
	return bufmodule.NewObjectData(normalpath.Base(protoFile.Path), protoFile.Content)
}

func objectDataToProtoFile(objectData bufmodule.ObjectData) *modulev1beta1.File {
	if objectData == nil {
		return nil
	}
	return &modulev1beta1.File{
		Path:    objectData.Name(),
		Content: objectData.Data(),
	}
}

func labelNameToProtoScopedLabelRef(labelName string) *modulev1beta1.ScopedLabelRef {
	return &modulev1beta1.ScopedLabelRef{
		Value: &modulev1beta1.ScopedLabelRef_Name{
			Name: labelName,
		},
	}
}

func protoGraphToProtoLegacyFederationGraph(
	registry string,
	protoGraph *modulev1beta1.Graph,
) *federationv1beta1.Graph {
	protoLegacyFederationGraph := &federationv1beta1.Graph{
		Commits: make([]*federationv1beta1.Graph_Commit, len(protoGraph.Commits)),
		Edges:   make([]*federationv1beta1.Graph_Edge, len(protoGraph.Edges)),
	}
	for i, protoCommit := range protoGraph.Commits {
		protoLegacyFederationGraph.Commits[i] = &federationv1beta1.Graph_Commit{
			Commit:   protoCommit,
			Registry: registry,
		}
	}
	for i, protoEdge := range protoGraph.Edges {
		protoLegacyFederationGraph.Edges[i] = &federationv1beta1.Graph_Edge{
			FromNode: &federationv1beta1.Graph_Node{
				CommitId: protoEdge.FromNode.CommitId,
				Registry: registry,
			},
			ToNode: &federationv1beta1.Graph_Node{
				CommitId: protoEdge.ToNode.CommitId,
				Registry: registry,
			},
		}
	}
	return protoLegacyFederationGraph
}

// We have to make sure this is updated if a field is added?
// TODO FUTURE: Can we automate this to make sure this is true?
func protoLegacyFederationUploadRequestContentToProtoUploadRequestContent(
	registry string,
	protoLegacyFederationUploadRequestContent *federationv1beta1.UploadRequest_Content,
) (*modulev1beta1.UploadRequest_Content, error) {
	protoUploadRequestContent := &modulev1beta1.UploadRequest_Content{
		ModuleRef:        protoLegacyFederationUploadRequestContent.ModuleRef,
		DepRefs:          make([]*modulev1beta1.UploadRequest_DepRef, len(protoLegacyFederationUploadRequestContent.DepRefs)),
		Files:            protoLegacyFederationUploadRequestContent.Files,
		V1BufYamlFile:    protoLegacyFederationUploadRequestContent.V1BufYamlFile,
		V1BufLockFile:    protoLegacyFederationUploadRequestContent.V1BufLockFile,
		ScopedLabelRefs:  protoLegacyFederationUploadRequestContent.ScopedLabelRefs,
		SourceControlUrl: protoLegacyFederationUploadRequestContent.SourceControlUrl,
	}
	for i, legacyDepRef := range protoLegacyFederationUploadRequestContent.DepRefs {
		if legacyDepRef.Registry != registry {
			return nil, syserror.Newf("tried to convert a legacy federation UploadRequest_Content to a module UploadRequest_Content with registry %q but found registry %q in DepRefs", registry, legacyDepRef.Registry)
		}
		protoUploadRequestContent.DepRefs[i] = &modulev1beta1.UploadRequest_DepRef{
			ModuleRef: legacyDepRef.ModuleRef,
			CommitId:  legacyDepRef.CommitId,
		}
	}
	return protoUploadRequestContent, nil
}
