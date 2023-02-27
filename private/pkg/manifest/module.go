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

package manifest

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"go.uber.org/multierr"
)

// Blob is an anonymous file associated with a digest.
type Blob interface {
	Digest() *Digest
	Open(context.Context) (io.ReadCloser, error)
}

type memoryBlob struct {
	digest  Digest
	content []byte
}

var _ Blob = (*memoryBlob)(nil)

type memoryBlobOptions struct {
	validateDigest bool
}

// MemoryBlobOption are options passed when creating a new memory blob.
type MemoryBlobOption func(*memoryBlobOptions)

// MemoryBlobWithDigestValidation checks that the passed content and digest match.
func MemoryBlobWithDigestValidation() MemoryBlobOption {
	return func(opts *memoryBlobOptions) {
		opts.validateDigest = true
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
	if config.validateDigest {
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
	if b == nil {
		return nil
	}
	return &b.digest
}

func (b *memoryBlob) Open(context.Context) (io.ReadCloser, error) {
	if b == nil {
		return nil, os.ErrNotExist
	}
	return io.NopCloser(bytes.NewReader(b.content)), nil
}

// BlobSet represents a set of deduplicated blobs by their digests.
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

// NewBlobSet receives a slice of blobs, and de-duplicates them into a BlobSet.
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
				equalContent, err := BlobEqual(ctx, b, existingBlob)
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

// Blobs returns a slice of the blobs in the set.
func (s *BlobSet) Blobs() []Blob {
	blobs := make([]Blob, 0, len(s.digestToBlob))
	for _, b := range s.digestToBlob {
		blobs = append(blobs, b)
	}
	return blobs
}

// NewMemoryBlobFromReader creates a memory blob from content, which is read
// until completion. The returned blob contains all bytes read. If you are using
// this in a loop, you might better use NewMemoryBlobFromReaderWithDigester so
// you can reuse your digester.
func NewMemoryBlobFromReader(content io.Reader) (Blob, error) {
	digester, err := NewDigester(DigestTypeShake256)
	if err != nil {
		return nil, err
	}
	return NewMemoryBlobFromReaderWithDigester(content, digester)
}

// NewMemoryBlobFromReaderWithDigester creates a memory blob from content with
// the passed digester. The content is read until completion. The returned blob
// contains all bytes read.
func NewMemoryBlobFromReaderWithDigester(content io.Reader, digester Digester) (Blob, error) {
	var contentInMemory bytes.Buffer
	tee := io.TeeReader(content, &contentInMemory)
	digest, err := digester.Digest(tee)
	if err != nil {
		return nil, err
	}
	return &memoryBlob{
		digest:  *digest,
		content: contentInMemory.Bytes(),
	}, nil
}

// BlobEqual returns true if blob a is the same as blob b. The digest is
// checked for equality and the content bytes compared.
//
// An error is returned if an unexpected I/O error occurred when opening,
// reading, or closing either blob.
func BlobEqual(ctx context.Context, a, b Blob) (_ bool, retErr error) {
	const blockSize = 4096
	if !a.Digest().Equal(*b.Digest()) {
		// digests don't match
		return false, nil
	}
	aFile, err := a.Open(ctx)
	if err != nil {
		return false, err
	}
	defer func() { retErr = multierr.Append(retErr, aFile.Close()) }()
	bFile, err := b.Open(ctx)
	if err != nil {
		return false, err
	}
	defer func() { retErr = multierr.Append(retErr, bFile.Close()) }()
	// Read blockSize from a, then from b, and compare.
	aBlock := make([]byte, blockSize)
	bBlock := make([]byte, blockSize)
	for {
		aN, aErr := aFile.Read(aBlock)
		bN, bErr := io.ReadAtLeast(bFile, bBlock[:aN], aN) // exactly aN bytes
		// We're running unexpected error processing (not EOF) before comparing
		// bytes because it doesn't matter if the returned bytes match if an
		// error occurred before an expected EOF.
		if bErr == io.ErrUnexpectedEOF {
			// b is shorter; we can error early
			return false, nil
		}
		if aErr != nil && aErr != io.EOF {
			// unexpected read error
			return false, aErr
		}
		if bErr != nil && bErr != io.EOF {
			// unexpected read error
			return false, bErr
		}
		if !bytes.Equal(aBlock[:aN], bBlock[:bN]) {
			// Read content doesn't match.
			return false, nil
		}
		if aErr == io.EOF || bErr == io.EOF {
			// EOF
			break
		}
	}
	aN, aErr := aFile.Read(aBlock[:1])
	bN, bErr := bFile.Read(bBlock[:1])
	if aN == 0 && bN == 0 && aErr == io.EOF && bErr == io.EOF {
		// a and b are at EOF with no more data for us
		return true, nil
	}
	// either a or b are longer
	return false, multierr.Append(nilEOF(aErr), nilEOF(bErr))
}

// nilEOF maps io.EOF to nil
func nilEOF(err error) error {
	if err == io.EOF {
		return nil
	}
	return err
}
