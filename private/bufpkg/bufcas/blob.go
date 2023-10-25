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

package bufcas

import (
	"bytes"
	"fmt"
	"io"

	storagev1beta1 "github.com/bufbuild/buf/private/gen/proto/go/buf/registry/storage/v1beta1"
)

// Blob is content with its associated Digest.
type Blob interface {
	// Digest returns the Digest of the Blob.
	//
	// Always non-nil.
	//
	// NewDigestForContent(blob.Digest.Type(), bytes.NewReader(blob.Content()) should
	// always match this value.
	Digest() Digest
	// Content returns the content of the Blob.
	//
	// Always non-empty.
	Content() []byte

	// Protect against creation of a Blob outside of this package, as we
	// do very careful validation.
	isBlob()
}

// NewBlobForContent returns a new Blob with a Digest of the given DigestType,
// and the content as read from the Reader.
//
// If the content is empty, returns nil.
//
// The reader is read until io.EOF.
// Validation is performed to ensure that the DigestType is known.
func NewBlobForContent(digestType DigestType, reader io.Reader) (Blob, error) {
	buffer := bytes.NewBuffer(nil)
	teeReader := io.TeeReader(reader, buffer)
	digest, err := NewDigestForContent(digestType, teeReader)
	if err != nil {
		return nil, err
	}
	// If the digest is nil, we have no content, return nil.
	if digest == nil {
		return nil, nil
	}
	return newBlob(digest, buffer.Bytes()), nil
}

// NewBlobForContentWithKnownDigest returns a new Blob for the given Digest and content
// as read from the Reader.
//
// If the content is empty, returns nil after verifying the known Digest is nil.
//
// The reader is read until io.EOF.
// Validation is performed to ensure that the Digest matches the computed Digest of the content.
func NewBlobForContentWithKnownDigest(knownDigest Digest, reader io.Reader) (Blob, error) {
	blob, err := NewBlobForContent(knownDigest.Type(), reader)
	if err != nil {
		return nil, err
	}
	if blob == nil {
		if knownDigest != nil {
			return nil, fmt.Errorf("Blob was empty but had a known non-empty Digest %v", knownDigest)
		}
		return nil, nil
	}
	if !DigestEqual(blob.Digest(), knownDigest) {
		return nil, fmt.Errorf("Digest %v did not match known Digest %v when creating a new Blob", blob.Digest(), knownDigest)
	}
	return blob, nil
}

// BlobToProto converts the given Blob to a proto Blob.
//
// If the given Blob is nil, returns nil.
//
// TODO: validate the returned Blob.
func BlobToProto(blob Blob) (*storagev1beta1.Blob, error) {
	if blob == nil {
		return nil, nil
	}
	protoDigest, err := DigestToProto(blob.Digest())
	if err != nil {
		return nil, err
	}
	return &storagev1beta1.Blob{
		Digest:  protoDigest,
		Content: blob.Content(),
	}, nil
}

// ProtoToBlob converts the given proto Blob to a Blob.
//
// If the given proto Blob is nil, returns nil.
//
// Validation is performed to ensure that the Digest matches the computed Digest of the content.
// TODO: validate the input proto Blob.
func ProtoToBlob(protoBlob *storagev1beta1.Blob) (Blob, error) {
	if protoBlob == nil {
		return nil, nil
	}
	digest, err := ProtoToDigest(protoBlob.Digest)
	if err != nil {
		return nil, err
	}
	return NewBlobForContentWithKnownDigest(digest, bytes.NewReader(protoBlob.Content))
}

// BlobEqual returns true if the given Blobs are considered equal.
//
// This checks both the Digest and the content.
//
// TODO: In the former version of this package, we compared content values as well.
// We should be able to remove this, and this is commented out for now. Technically we do not
// need to compare the contents, as we know that the Digest is a valid Digest for the
// given content via valiation we did at construction time.
func BlobEqual(a Blob, b Blob) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	if a == nil {
		return true
	}
	if !DigestEqual(a.Digest(), b.Digest()) {
		return false
	}
	//aContent := a.Content()
	//bContent := b.Content()
	//for i := 0; i < len(aContent); i += 4096 {
	//j := i + 4096
	//if j > len(aContent) {
	//j = len(aContent)
	//}
	//if !bytes.Equal(aContent[i:j], bContent[i:j]) {
	//return false
	//}
	//}
	return true
}

// *** PRIVATE ***

type blob struct {
	digest  Digest
	content []byte
}

func newBlob(digest Digest, content []byte) *blob {
	return &blob{
		digest:  digest,
		content: content,
	}
}

func (b *blob) Digest() Digest {
	return b.digest
}

func (b *blob) Content() []byte {
	return b.content
}

func (*blob) isBlob() {}
