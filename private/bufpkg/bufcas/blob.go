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

	storagev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/storage/v1beta1"
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
	// May be empty.
	Content() []byte

	// Protect against creation of a Blob outside of this package, as we
	// do very careful validation.
	isBlob()
}

// NewBlobForContent returns a new Blob for the content as read from the Reader.
//
// The reader is read until io.EOF.
func NewBlobForContent(reader io.Reader, options ...BlobOption) (Blob, error) {
	blobOptions := newBlobOptions()
	for _, option := range options {
		option(blobOptions)
	}
	buffer := bytes.NewBuffer(nil)
	teeReader := io.TeeReader(reader, buffer)
	digest, err := NewDigestForContent(teeReader, DigestWithDigestType(blobOptions.digestType))
	if err != nil {
		return nil, err
	}
	blob := newBlob(digest, buffer.Bytes())
	if blobOptions.knownDigest != nil && !DigestEqual(blob.Digest(), blobOptions.knownDigest) {
		return nil, fmt.Errorf("Digest %v did not match known Digest %v when creating a new Blob", blob.Digest(), blobOptions.knownDigest)
	}
	return blob, nil
}

// BlobOption is an option when constructing a new Blob
type BlobOption func(*blobOptions)

// BlobWithKnownDigest returns a new BlobOption that results in validation that the
// Digest for the new Blob matches an existing known Digest.
func BlobWithKnownDigest(knownDigest Digest) BlobOption {
	return func(blobOptions *blobOptions) {
		blobOptions.knownDigest = knownDigest
	}
}

// BlobWithDigestType returns a new BlobOption sets the DigestType to be used.
//
// The default is DigestTypeShake256.
func BlobWithDigestType(digestType DigestType) BlobOption {
	return func(blobOptions *blobOptions) {
		blobOptions.digestType = digestType
	}
}

// BlobToProto converts the given Blob to a proto Blob.
//
// TODO: validate the returned Blob.
func BlobToProto(blob Blob) (*storagev1beta1.Blob, error) {
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
// Validation is performed to ensure that the Digest matches the computed Digest of the content.
// TODO: validate the input proto Blob.
func ProtoToBlob(protoBlob *storagev1beta1.Blob) (Blob, error) {
	digest, err := ProtoToDigest(protoBlob.Digest)
	if err != nil {
		return nil, err
	}
	return NewBlobForContent(bytes.NewReader(protoBlob.Content), BlobWithKnownDigest(digest))
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

type blobOptions struct {
	knownDigest Digest
	digestType  DigestType
}

func newBlobOptions() *blobOptions {
	return &blobOptions{}
}
