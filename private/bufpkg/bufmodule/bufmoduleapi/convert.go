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
	"io"

	modulev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1beta1"
	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
)

var (
	digestTypeToProto = map[bufmodule.DigestType]modulev1beta1.DigestType{
		bufmodule.DigestTypeB4: modulev1beta1.DigestType_DIGEST_TYPE_B4,
		bufmodule.DigestTypeB5: modulev1beta1.DigestType_DIGEST_TYPE_B5,
	}
	protoToDigestType = map[modulev1beta1.DigestType]bufmodule.DigestType{
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

// CommitIDToProto converts the CommitID to a BSR Commit ID.
//
// This just takes a dashless UUID and converts it to a dashful UUID.
func CommitIDToProto(commitID string) (string, error) {
	protoCommitID, err := uuidutil.FromDashless(commitID)
	if err != nil {
		return "", fmt.Errorf("invalid commit ID %s: %w", commitID, err)
	}
	return protoCommitID.String(), nil
}

// ProtoToCommitID converts the BSR Commit ID to a CommitID.
//
// This just takes a dashless UUID and converts it to a dashful UUID.
func ProtoToCommitID(protoCommitID string) (string, error) {
	id, err := uuidutil.FromString(protoCommitID)
	if err != nil {
		return "", fmt.Errorf("invalid BSR commit ID %s: %w", protoCommitID, err)
	}
	return uuidutil.ToDashless(id)
}

// DigestToProto converts the given Digest to a proto Digest.
func DigestToProto(digest bufmodule.Digest) (*modulev1beta1.Digest, error) {
	protoDigestType, ok := digestTypeToProto[digest.Type()]
	// Technically we have already done this validation but just to be safe.
	if !ok {
		return nil, fmt.Errorf("unknown DigestType: %v", digest.Type())
	}
	protoDigest := &modulev1beta1.Digest{
		Type:  protoDigestType,
		Value: digest.Value(),
	}
	return protoDigest, nil
}

// ProtoToDigest converts the given proto Digest to a Digest.
//
// Validation is performed to ensure the DigestType is known, and the value
// is a valid digest value for the given DigestType.
func ProtoToDigest(protoDigest *modulev1beta1.Digest) (bufmodule.Digest, error) {
	digestType, ok := protoToDigestType[protoDigest.Type]
	if !ok {
		return nil, fmt.Errorf("unknown proto Digest.Type: %v", protoDigest.Type)
	}
	bufcasDigest, err := bufcas.NewDigest(protoDigest.Value)
	if err != nil {
		return nil, err
	}
	return bufmodule.NewDigest(digestType, bufcasDigest)
}

// *** PRIVATE ***

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

func labelNameToProtoScopedLabelRef(labelName string) *modulev1beta1.ScopedLabelRef {
	return &modulev1beta1.ScopedLabelRef{
		Value: &modulev1beta1.ScopedLabelRef_Name{
			Name: labelName,
		},
	}
}
