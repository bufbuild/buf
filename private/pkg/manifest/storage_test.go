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

package manifest_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/bufbuild/buf/private/pkg/manifest"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromBucket(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	bucket, err := storagemem.NewReadBucket(
		map[string][]byte{
			"null": nil,
			"foo":  []byte("bar"),
		})
	require.NoError(t, err)
	m, err := manifest.NewFromBucket(ctx, bucket)
	require.NoError(t, err)
	// sorted by paths
	expected := fmt.Sprintf("%s  foo\n", mustDigestShake256(t, []byte("bar")))
	expected += fmt.Sprintf("%s  null\n", mustDigestShake256(t, nil))
	retContent, err := m.MarshalText()
	assert.NoError(t, err)
	assert.Equal(t, expected, string(retContent))
}

func TestToBucket(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"some_empty_file":    {},
		"buf.yaml":           []byte("buf yaml contents"),
		"buf.lock":           []byte("buf lock contents"),
		"mypkg/v1/foo.proto": []byte("repeated proto content"),
		"mypkg/v1/bar.proto": []byte("repeated proto content"),
	}
	m := manifest.New()
	digestToContent := make(map[string][]byte, 0)
	digester, err := manifest.NewDigester(manifest.DigestTypeShake256)
	require.NoError(t, err)
	for path, content := range files {
		digest, err := digester.Digest(bytes.NewReader(content))
		require.NoError(t, err)
		require.NoError(t, m.AddEntry(path, *digest))
		digestToContent[digest.String()] = content
	}

	t.Run("InvalidDigestsMaps", func(t *testing.T) {
		t.Parallel()
		_, err = m.ToBucket(nil)
		assert.Error(t, err)
		_, err = m.ToBucket(map[string][]byte{
			"some":  {},
			"other": []byte("foo"),
			"files": []byte("bar"),
		})
		assert.Error(t, err)
	})

	t.Run("ValidDigestsMap", func(t *testing.T) {
		t.Parallel()
		bucket, err := m.ToBucket(digestToContent)
		require.NoError(t, err)
		// make sure all files are present and have the right content
		for path, content := range files {
			obj, err := bucket.Get(context.Background(), path)
			require.NoError(t, err)
			retContent, err := io.ReadAll(obj)
			require.NoError(t, err)
			require.Equal(t, content, retContent)
			require.NoError(t, obj.Close())
		}
		// make sure there are no extra files in the bucket
		require.NoError(t, bucket.Walk(context.Background(), "", func(obj storage.ObjectInfo) error {
			_, presentInOriginalFiles := files[obj.Path()]
			require.True(t, presentInOriginalFiles, "path %q in bucket is not present in original files", obj.Path())
			return nil
		}))
	})
}

func TestToBucketEmpty(t *testing.T) {
	t.Parallel()
	m := manifest.New()
	bucket, err := m.ToBucket(nil)
	require.NoError(t, err)
	// make sure there are no files in the bucket
	require.NoError(t, bucket.Walk(context.Background(), "", func(obj storage.ObjectInfo) error {
		require.Fail(t, "unexpected file %q in the bucket", obj.Path())
		return nil
	}))
}
