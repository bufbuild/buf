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
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBlobSet(t *testing.T) {
	t.Parallel()
	blobs := testNewBlobs(t, 10)
	blobSet, err := NewBlobSet(blobs)
	require.NoError(t, err)
	testAssertBlobsEqual(t, blobs, blobSet.Blobs())
	assert.Equal(t, blobs[0], blobSet.GetBlob(blobs[0].Digest()))
}

func TestNewBlobSetDuplicatesValid(t *testing.T) {
	t.Parallel()
	blobs := testNewBlobs(t, 10)
	_, err := NewBlobSet(append(blobs, blobs[0]))
	require.NoError(t, err)
}

func testNewBlobs(t *testing.T, size int) []Blob {
	var blobs []Blob
	for i := 0; i < size; i++ {
		content := fmt.Sprintf("some file content %d", i)
		blob, err := NewBlobForContent(strings.NewReader(content))
		require.NoError(t, err)
		blobs = append(blobs, blob)
	}
	return blobs
}

// testAssertBlobsEqual makes sure all the blobs digests in the array are the
// same (assuming they're correctly built), ignoring order in the blobs arrays.
func testAssertBlobsEqual(t *testing.T, expectedBlobs []Blob, actualBlobs []Blob) {
	expectedDigests := make(map[string]struct{}, len(expectedBlobs))
	for _, expectedBlob := range expectedBlobs {
		expectedDigests[expectedBlob.Digest().String()] = struct{}{}
	}
	actualDigests := make(map[string]struct{}, len(actualBlobs))
	for _, actualBlob := range actualBlobs {
		actualDigests[actualBlob.Digest().String()] = struct{}{}
	}
	assert.Equal(t, expectedDigests, actualDigests)
}
