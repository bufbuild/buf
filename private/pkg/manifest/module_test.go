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

	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDigestFromBlobHash(t *testing.T) {
	t.Parallel()
	const (
		filePath    = "path/to/file"
		fileContent = "one line\nanother line\nyet another one\n"
	)
	digestFromContent, err := manifest.NewDigestFromBytes(
		manifest.DigestTypeShake256,
		mustDigestShake256(t, []byte(fileContent)).Bytes(),
	)
	require.NoError(t, err)
	assert.Equal(t, manifest.DigestTypeShake256, digestFromContent.Type())
	blobHash := modulev1alpha1.Hash{
		Kind:   modulev1alpha1.HashKind_HASH_KIND_SHAKE256,
		Digest: digestFromContent.Bytes(),
	}
	digestFromBlobHash, err := manifest.NewDigestFromBlobHash(&blobHash)
	require.NoError(t, err)
	assert.Equal(t, digestFromContent.String(), digestFromBlobHash.String())
}

func TestNewMemoryBlob(t *testing.T) {
	t.Parallel()
	const content = "some file content"
	digest := mustDigestShake256(t, []byte(content))
	blob, err := manifest.NewMemoryBlob(
		*digest,
		[]byte(content),
		manifest.MemoryBlobWithHashValidation(),
	)
	require.NoError(t, err)
	assert.True(t, blob.Digest().Equal(*digest))
	file, err := blob.Open(context.Background())
	require.NoError(t, err)
	blobContent, err := io.ReadAll(file)
	require.NoError(t, err)
	assert.Equal(t, []byte(content), blobContent)
}

func TestInvalidMemoryBlob(t *testing.T) {
	t.Parallel()
	const content = "some file content"
	digest := mustDigestShake256(t, []byte(content))

	t.Run("NoValidateHash", func(t *testing.T) {
		t.Parallel()
		_, err := manifest.NewMemoryBlob(*digest, []byte("different content"))
		assert.NoError(t, err)
	})
	t.Run("ValidatingHash", func(t *testing.T) {
		t.Parallel()
		_, err := manifest.NewMemoryBlob(
			*digest,
			[]byte("different content"),
			manifest.MemoryBlobWithHashValidation(),
		)
		assert.Error(t, err)
	})
}

func TestNewDigestFromBlobHash(t *testing.T) {
	t.Parallel()
	digest := mustDigestShake256(t, []byte("my content"))
	retDigest, err := manifest.NewDigestFromBlobHash(&modulev1alpha1.Hash{
		Kind:   modulev1alpha1.HashKind_HASH_KIND_SHAKE256,
		Digest: digest.Bytes(),
	})
	require.NoError(t, err)
	assert.True(t, digest.Equal(*retDigest))
}

func TestInvalidNewDigestFromBlobHash(t *testing.T) {
	t.Parallel()
	_, err := manifest.NewDigestFromBlobHash(nil)
	assert.Error(t, err)
	_, err = manifest.NewDigestFromBlobHash(&modulev1alpha1.Hash{
		Kind: modulev1alpha1.HashKind_HASH_KIND_UNSPECIFIED,
	})
	assert.Error(t, err)
	_, err = manifest.NewDigestFromBlobHash(&modulev1alpha1.Hash{
		Kind:   modulev1alpha1.HashKind_HASH_KIND_SHAKE256,
		Digest: []byte("invalid digest"),
	})
	assert.Error(t, err)
}

func TestNewBlobSet(t *testing.T) {
	t.Parallel()
	blobSet, err := manifest.NewBlobSet(
		context.Background(),
		newBlobsArray(t),
		manifest.BlobSetWithContentValidation(),
	)
	require.NoError(t, err)
	assert.NotNil(t, blobSet)
}

func TestNewBlobValidDuplicates(t *testing.T) {
	t.Parallel()
	blobs := newBlobsArray(t)
	blobSet, err := manifest.NewBlobSet(
		context.Background(),
		append(blobs, blobs[0]), // send the first blob twice
		manifest.BlobSetWithContentValidation(),
	)
	require.NoError(t, err)
	assert.NotNil(t, blobSet)
}

func TestNewBlobInvalidDuplicates(t *testing.T) {
	blobs := newBlobsArray(t)
	incorrectBlob, err := manifest.NewMemoryBlob(
		*blobs[0].Digest(),
		[]byte("not blobs[0] content"),
	)
	require.NoError(t, err)
	require.NotNil(t, incorrectBlob)
	_, err = manifest.NewBlobSet(
		context.Background(),
		append(blobs, incorrectBlob), // send first digest twice, with diff content
		manifest.BlobSetWithContentValidation(),
	)
	require.Error(t, err)
}

func TestBlobFromReader(t *testing.T) {
	testBlobFromReader(
		t,
		[]byte("hello"),
		[]byte{
			0x12, 0x34, 0x07, 0x5a, 0xe4, 0xa1, 0xe7, 0x73, 0x16, 0xcf, 0x2d,
			0x80, 0x00, 0x97, 0x45, 0x81, 0xa3, 0x43, 0xb9, 0xeb, 0xbc, 0xa7,
			0xe3, 0xd1, 0xdb, 0x83, 0x39, 0x4c, 0x30, 0xf2, 0x21, 0x62, 0x6f,
			0x59, 0x4e, 0x4f, 0x0d, 0xe6, 0x39, 0x02, 0x34, 0x9a, 0x5e, 0xa5,
			0x78, 0x12, 0x13, 0x21, 0x58, 0x13, 0x91, 0x9f, 0x92, 0xa4, 0xd8,
			0x6d, 0x12, 0x74, 0x66, 0xe3, 0xd0, 0x7e, 0x8b, 0xe3,
		},
	)
}

func testBlobFromReader(t *testing.T, content []byte, digest []byte) {
	t.Helper()
	t.Parallel()
	blob, err := manifest.NewBlobFromReader(bytes.NewReader(content))
	require.NoError(t, err)
	expect := &modulev1alpha1.Blob{
		Hash: &modulev1alpha1.Hash{
			Kind:   modulev1alpha1.HashKind_HASH_KIND_SHAKE256,
			Digest: digest,
		},
		Content: content,
	}
	assert.Equal(t, expect, blob)
}

func newBlobsArray(t *testing.T) []manifest.Blob {
	var blobs []manifest.Blob
	for i := 0; i < 10; i++ {
		content := fmt.Sprintf("some content %d", i)
		digest := mustDigestShake256(t, []byte(content))
		blob, err := manifest.NewMemoryBlob(
			*digest, []byte(content),
			manifest.MemoryBlobWithHashValidation(),
		)
		require.NoError(t, err)
		blobs = append(blobs, blob)
	}
	return blobs
}
