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

package bufmanifest

import (
	"context"
	"fmt"
	"io"

	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/manifest"
	"go.uber.org/multierr"
)

var (
	protoDigestTypeToDigestType = map[modulev1alpha1.DigestType]manifest.DigestType{
		modulev1alpha1.DigestType_DIGEST_TYPE_SHAKE256: manifest.DigestTypeShake256,
	}
	digestTypeToProtoDigestType = map[manifest.DigestType]modulev1alpha1.DigestType{
		manifest.DigestTypeShake256: modulev1alpha1.DigestType_DIGEST_TYPE_SHAKE256,
	}
)

// NewDigestFromProtoDigest maps a modulev1alpha1.Digest to a Digest.
func NewDigestFromProtoDigest(digest *modulev1alpha1.Digest) (*manifest.Digest, error) {
	if digest == nil {
		return nil, fmt.Errorf("nil digest")
	}
	dType, ok := protoDigestTypeToDigestType[digest.DigestType]
	if !ok {
		return nil, fmt.Errorf("unsupported digest kind: %s", digest.DigestType.String())
	}
	return manifest.NewDigestFromBytes(dType, digest.Digest)
}

// AsProtoBlob returns the passed blob as a proto module blob.
func AsProtoBlob(ctx context.Context, b manifest.Blob) (_ *modulev1alpha1.Blob, retErr error) {
	digestType, ok := digestTypeToProtoDigestType[b.Digest().Type()]
	if !ok {
		return nil, fmt.Errorf("digest type %q not supported by module proto", b.Digest().Type())
	}
	rc, err := b.Open(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot open blob: %w", err)
	}
	defer func() {
		retErr = multierr.Append(retErr, rc.Close())
	}()
	content, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("cannot read blob contents: %w", err)
	}
	return &modulev1alpha1.Blob{
		Digest: &modulev1alpha1.Digest{
			DigestType: digestType,
			Digest:     b.Digest().Bytes(),
		},
		Content: content,
	}, nil
}

// NewBlobFromProto returns a Blob from a proto module blob. It makes sure the
// digest and content matches.
func NewBlobFromProto(b *modulev1alpha1.Blob) (manifest.Blob, error) {
	if b == nil {
		return nil, fmt.Errorf("nil blob")
	}
	digest, err := NewDigestFromProtoDigest(b.Digest)
	if err != nil {
		return nil, fmt.Errorf("digest from proto digest: %w", err)
	}
	memBlob, err := manifest.NewMemoryBlob(
		*digest,
		b.Content,
		manifest.MemoryBlobWithDigestValidation(),
	)
	if err != nil {
		return nil, fmt.Errorf("new memory blob: %w", err)
	}
	return memBlob, nil
}

// ToProtoManifestAndBlobs converts a Manifest and BlobSet to the protobuf types.
func ToProtoManifestAndBlobs(ctx context.Context, manifest *manifest.Manifest, blobs *manifest.BlobSet) (*modulev1alpha1.Blob, []*modulev1alpha1.Blob, error) {
	manifestBlob, err := manifest.Blob()
	if err != nil {
		return nil, nil, err
	}
	manifestProtoBlob, err := AsProtoBlob(ctx, manifestBlob)
	if err != nil {
		return nil, nil, err
	}
	filesBlobs := blobs.Blobs()
	filesProtoBlobs := make([]*modulev1alpha1.Blob, len(filesBlobs))
	for i, b := range filesBlobs {
		pb, err := AsProtoBlob(ctx, b)
		if err != nil {
			return nil, nil, err
		}
		filesProtoBlobs[i] = pb
	}
	return manifestProtoBlob, filesProtoBlobs, nil
}
