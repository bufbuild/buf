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

package storagemanifest

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/pkg/manifest"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO: adapt to the storagetesting framework

func TestNewReadBucket(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"some_empty_file":    {},
		"buf.yaml":           []byte("buf yaml contents"),
		"buf.lock":           []byte("buf lock contents"),
		"mypkg/v1/foo.proto": []byte("repeated proto content"),
		"mypkg/v1/bar.proto": []byte("repeated proto content"),
		// same "mypkg" prefix for `Walk` test purposes
		"mypkglongername/v1/baz.proto": []byte("repeated proto content"),
	}
	m := &manifest.Manifest{}
	var blobs []manifest.Blob
	digester, err := manifest.NewDigester(manifest.DigestTypeShake256)
	require.NoError(t, err)
	for path, content := range files {
		digest, err := digester.Digest(bytes.NewReader(content))
		require.NoError(t, err)
		require.NoError(t, m.AddEntry(path, *digest))
		blob, err := manifest.NewMemoryBlob(
			*digest,
			content,
			manifest.MemoryBlobWithDigestValidation(),
		)
		require.NoError(t, err)
		blobs = append(blobs, blob)
	}
	blobSet, err := manifest.NewBlobSet(
		context.Background(),
		blobs,
		manifest.BlobSetWithContentValidation(),
	)
	require.NoError(t, err)

	t.Run("BucketWithAllManifestBlobs", func(t *testing.T) {
		t.Parallel()
		// only send 3 blobs: there are 6 files with 4 different contents,
		// regardless of which blobs are sent, there will always be missing at least
		// one.
		const blobsToSend = 3
		incompleteBlobSet, err := manifest.NewBlobSet(
			context.Background(),
			blobs[:blobsToSend],
		)
		require.NoError(t, err)

		_, err = NewReadBucket(
			m, incompleteBlobSet,
			ReadBucketWithAllManifestBlobs(),
		)
		assert.Error(t, err)

		bucket, err := NewReadBucket(m, incompleteBlobSet)
		assert.NoError(t, err)
		assert.NotNil(t, bucket)
		var bucketFilesCount int
		require.NoError(t, bucket.Walk(context.Background(), "", func(obj storage.ObjectInfo) error {
			bucketFilesCount++
			return nil
		}))
		assert.Less(t, bucketFilesCount, len(files)) // incomplete bucket
	})

	t.Run("BucketWithNoExtraBlobs", func(t *testing.T) {
		t.Parallel()
		const content = "some other file contents"
		digest := mustDigestShake256(t, []byte(content))
		orphanBlob, err := manifest.NewMemoryBlob(*digest, []byte(content))
		require.NoError(t, err)
		tooLargeBlobSet, err := manifest.NewBlobSet(
			context.Background(),
			append(blobs, orphanBlob),
		)
		require.NoError(t, err)
		_, err = NewReadBucket(
			m, tooLargeBlobSet,
			ReadBucketWithNoExtraBlobs(),
		)
		assert.Error(t, err)
	})

	t.Run("Valid", func(t *testing.T) {
		t.Parallel()
		bucket, err := NewReadBucket(
			m, blobSet,
			ReadBucketWithAllManifestBlobs(),
			ReadBucketWithNoExtraBlobs(),
		)
		require.NoError(t, err)

		t.Run("BucketGet", func(t *testing.T) {
			t.Parallel()
			// make sure all files are present and have the right content
			for path, content := range files {
				obj, err := bucket.Get(context.Background(), path)
				require.NoError(t, err)
				retContent, err := io.ReadAll(obj)
				require.NoError(t, err)
				assert.Equal(t, content, retContent)
				assert.NoError(t, obj.Close())
			}
			// non existent files
			_, err = bucket.Get(context.Background(), "path/not/present")
			assert.Error(t, err)
		})

		t.Run("BucketWalk", func(t *testing.T) {
			t.Parallel()
			// make sure there are no extra files in the bucket
			require.NoError(t, bucket.Walk(context.Background(), "", func(obj storage.ObjectInfo) error {
				_, presentInOriginalFiles := files[obj.Path()]
				require.True(t, presentInOriginalFiles, "path %q in bucket is not present in original files", obj.Path())
				return nil
			}))
			// walking a non existent dir
			assert.NoError(t, bucket.Walk(context.Background(), "nonexistentpkg", func(obj storage.ObjectInfo) error {
				require.Fail(t, "unexpected file %q in the bucket", obj.Path())
				return nil
			}))
			// walking a valid dir
			const prefix = "mypkg"
			expectedPaths := make(map[string]struct{}, 0)
			for path := range files {
				if strings.HasPrefix(path, prefix+"/") {
					expectedPaths[path] = struct{}{}
				}
			}
			assert.NoError(t, bucket.Walk(context.Background(), prefix, func(obj storage.ObjectInfo) error {
				_, expected := expectedPaths[obj.Path()]
				require.True(t, expected, "walking path %q was not expected", obj.Path())
				return nil
			}))
		})
	})
}

func TestNewReadBucketEmpty(t *testing.T) {
	t.Parallel()
	bucket, err := NewReadBucket(&manifest.Manifest{}, &manifest.BlobSet{})
	require.NoError(t, err)
	// make sure there are no files in the bucket
	require.NoError(t, bucket.Walk(context.Background(), "", func(obj storage.ObjectInfo) error {
		require.Fail(t, "unexpected file %q in the bucket", obj.Path())
		return nil
	}))
}

func mustDigestShake256(t *testing.T, content []byte) *manifest.Digest {
	digester, err := manifest.NewDigester(manifest.DigestTypeShake256)
	require.NoError(t, err)
	require.NotNil(t, digester)
	digest, err := digester.Digest(bytes.NewReader(content))
	require.NoError(t, err)
	return digest
}
