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

import "sort"

type blobSet struct {
	digestStringToBlob  map[string]Blob
	sortedDigestStrings []string
}

func newBlobSet(blobs []Blob) *blobSet {
	digestStringToBlob := make(map[string]Blob, len(blobs))
	sortedDigestStrings := make([]string, 0, len(blobs))
	for _, blob := range blobs {
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
	}
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
