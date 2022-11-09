// Copyright 2020-2022 Buf Technologies, Inc.
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

package manifest

import (
	"bytes"
	"context"
	"fmt"
	"io"

	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
)

var hashKindToDigestType = map[modulev1alpha1.HashKind]DigestType{
	modulev1alpha1.HashKind_HASH_KIND_SHAKE256: DigestTypeShake256,
}

// Blob is a blob with a digest and a content.
type Blob interface {
	Digest() *Digest
	Open(context.Context) (io.ReadCloser, error)
	EqualContent(ctx context.Context, other Blob) (bool, error)
}

type memoryBlob struct {
	digest  Digest
	content []byte
}

var _ Blob = (*memoryBlob)(nil)

type memoryBlobOptions struct {
	validateHash bool
}

// MemoryBlobOption are options passed when creating a new memory blob.
type MemoryBlobOption func(*memoryBlobOptions)

// NewMemoryBlob takes a digest and a content, and turns it into an in-memory
// representation of a blob, which returns the digest and an io.ReadCloser for
// its content.
func NewMemoryBlob(digest Digest, content []byte, opts ...MemoryBlobOption) (Blob, error) {
	var config memoryBlobOptions
	for _, option := range opts {
		option(&config)
	}
	if config.validateHash {
		digester, err := NewDigester(digest.Type())
		if err != nil {
			return nil, err
		}
		contentDigest, err := digester.Digest(bytes.NewReader(content))
		if err != nil {
			return nil, err
		}
		if !digest.Equal(*contentDigest) {
			return nil, fmt.Errorf("digest and content mismatch")
		}
	}
	return &memoryBlob{
		digest:  digest,
		content: content,
	}, nil
}

func (b *memoryBlob) Digest() *Digest {
	return &b.digest
}

func (b *memoryBlob) Open(_ context.Context) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(b.content)), nil
}

func (b *memoryBlob) EqualContent(ctx context.Context, other Blob) (bool, error) {
	otherContentRC, err := other.Open(ctx)
	if err != nil {
		return false, fmt.Errorf("open other blob: %w", err)
	}
	otherContent, err := io.ReadAll(otherContentRC)
	if err != nil {
		return false, fmt.Errorf("read other blob: %w", err)
	}
	if c := bytes.Compare(b.content, otherContent); c != 0 {
		return false, nil
	}
	return true, nil
}

// BlobSet represents a set of deduplicated blobs, by digests.
type BlobSet struct {
	digestToBlob map[string]Blob
}

type blobSetOptions struct {
	validateContent bool
}

// BlobSetOption are options passed when creating a new blob set.
type BlobSetOption func(*blobSetOptions)

// MemoryBlobWithHashValidation checks that the passed content and digest match.
func MemoryBlobWithHashValidation() MemoryBlobOption {
	return func(opts *memoryBlobOptions) {
		opts.validateHash = true
	}
}

// BlobSetWithContentValidation turns on content validation for all the blobs
// when creating a new BlobSet. If this option is on, multiple blobs with the
// same digest might be passed, as long as the contents match. If this option is
// not passed, then the latest content digest will prevail in the set.
func BlobSetWithContentValidation() BlobSetOption {
	return func(opts *blobSetOptions) {
		opts.validateContent = true
	}
}

// NewBlobSet receives an slice of blobs, and deduplicates them into a BlobSet.
func NewBlobSet(ctx context.Context, blobs []Blob, opts ...BlobSetOption) (*BlobSet, error) {
	var config blobSetOptions
	for _, option := range opts {
		option(&config)
	}
	digestToBlobs := make(map[string]Blob, len(blobs))
	for _, b := range blobs {
		digestStr := b.Digest().String()
		if config.validateContent {
			existingBlob, alreadyPresent := digestToBlobs[digestStr]
			if alreadyPresent {
				equalContent, err := b.EqualContent(ctx, existingBlob)
				if err != nil {
					return nil, fmt.Errorf("compare duplicated blobs with digest %q: %w", digestStr, err)
				}
				if !equalContent {
					return nil, fmt.Errorf("duplicated blobs with digest %q have different contents", digestStr)
				}
			}
		}
		digestToBlobs[digestStr] = b
	}
	return &BlobSet{digestToBlob: digestToBlobs}, nil
}

// NewDigestFromBlobHash returns a Digest based on a proto Hash.
func NewDigestFromBlobHash(hash *modulev1alpha1.Hash) (*Digest, error) {
	if hash == nil {
		return nil, fmt.Errorf("nil hash")
	}
	dType, ok := hashKindToDigestType[hash.Kind]
	if !ok {
		return nil, fmt.Errorf("unsupported hash kind: %s", hash.Kind.String())
	}
	return NewDigestFromBytes(dType, hash.Digest)
}
