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
	"go.uber.org/multierr"
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

// MemoryBlobWithHashValidation checks that the passed content and digest match.
func MemoryBlobWithHashValidation() MemoryBlobOption {
	return func(opts *memoryBlobOptions) {
		opts.validateHash = true
	}
}

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

func (b *memoryBlob) Open(context.Context) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(b.content)), nil
}

func (b *memoryBlob) EqualContent(ctx context.Context, other Blob) (_ bool, retErr error) {
	otherContentRC, err := other.Open(ctx)
	if err != nil {
		return false, fmt.Errorf("open other blob: %w", err)
	}
	defer func() { retErr = multierr.Append(retErr, otherContentRC.Close()) }()
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

// BlobSetWithContentValidation turns on content validation for all the blobs
// when creating a new BlobSet. If this option is on, blobs with the same digest
// must have the same content (in case blobs with the same digest are sent). If
// this option is not passed, then the latest duplicated blob digest content
// will prevail in the set.
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

// BlobFor returns the blob for the passed digest string, or nil, ok=false if
// the digest has no blob in the set.
func (s *BlobSet) BlobFor(digest string) (Blob, bool) {
	blob, ok := s.digestToBlob[digest]
	if !ok {
		return nil, false
	}
	return blob, true
}

// NewDigestFromBlobHash maps a module Hash to a digest.
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

// NewBlobFromReader creates a module Blob from content, which is read until
// completion. The returned blob contains all bytes read.
func NewBlobFromReader(content io.Reader) (*modulev1alpha1.Blob, error) {
	digester, err := NewDigester(DigestTypeShake256)
	if err != nil {
		return nil, err
	}
	var contentInMemory bytes.Buffer
	tee := io.TeeReader(content, &contentInMemory)
	digest, err := digester.Digest(tee)
	if err != nil {
		return nil, err
	}
	blob := &modulev1alpha1.Blob{
		Hash: &modulev1alpha1.Hash{
			Kind:   modulev1alpha1.HashKind_HASH_KIND_SHAKE256,
			Digest: digest.Bytes(),
		},
		Content: contentInMemory.Bytes(),
	}
	return blob, nil
}
