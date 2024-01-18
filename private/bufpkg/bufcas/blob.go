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

package bufcas

import (
	"bytes"
	"fmt"
	"io"
)

// Blob is content with its associated Digest.
type Blob interface {
	// Digest returns the Digest of the Blob.
	//
	// Always non-nil.
	//
	// NewDigestForContent(bytes.NewReader(blob.Content())) should always match this value.
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
