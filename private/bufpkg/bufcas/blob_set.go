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
	"sort"
)

// BlobSet is a set of deduplicated Blobs.
type BlobSet interface {
	// GetBlob gets the Blob for the given Digest, or nil if no such Blob exists.
	GetBlob(digest Digest) Blob
	// Blobs returns the Blobs associated with this BlobSet, ordered by
	// the sort value of the Digest.
	//
	// TODO: The former version of this package returns the Blobs in unspecified
	// order. We generally try to give a deterministic order in our codebase. There
	// are schools of arguments both ways, but we'd like to stay consistent.
	Blobs() []Blob

	// Protect against creation of a BlobSet outside of this package, as we
	// do very careful validation.
	isBlobSet()
}

// NewBlobSet returns a new BlobSet.
//
// Blobs are deduplicated upon construction.
func NewBlobSet(blobs []Blob) (BlobSet, error) {
	return newBlobSet(blobs)
}

// *** PRIVATE ***

type blobSet struct {
	digestStringToBlob  map[string]Blob
	sortedDigestStrings []string
}

func newBlobSet(blobs []Blob) (*blobSet, error) {
	digestStringToBlob := make(map[string]Blob, len(blobs))
	sortedDigestStrings := make([]string, 0, len(blobs))
	for _, blob := range blobs {
		// In the former version of this package, we validated that Blob contents matched for Blobs
		// with the same Digest via BlobEqual, however we no longer do this as BlobEqual no longer
		// validates content matching. See the comment on BlobEqual for why.
		digestString := blob.Digest().String()
		if _, ok := digestStringToBlob[digestString]; !ok {
			digestStringToBlob[digestString] = blob
			sortedDigestStrings = append(sortedDigestStrings, digestString)
		}
	}
	sort.Strings(sortedDigestStrings)
	return &blobSet{
		digestStringToBlob:  digestStringToBlob,
		sortedDigestStrings: sortedDigestStrings,
	}, nil
}

func (b *blobSet) GetBlob(digest Digest) Blob {
	return b.digestStringToBlob[digest.String()]
}

func (b *blobSet) Blobs() []Blob {
	blobs := make([]Blob, 0, len(b.digestStringToBlob))
	for _, digestString := range b.sortedDigestStrings {
		blobs = append(blobs, b.digestStringToBlob[digestString])
	}
	return blobs
}

func (*blobSet) isBlobSet() {}
