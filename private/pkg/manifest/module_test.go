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
	"context"
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
	blob, err := manifest.NewMemoryBlob(*digest, []byte(content), true)
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
		_, err := manifest.NewMemoryBlob(*digest, []byte("different content"), false)
		assert.NoError(t, err)
	})
	t.Run("ValidatingHash", func(t *testing.T) {
		t.Parallel()
		_, err := manifest.NewMemoryBlob(*digest, []byte("different content"), true)
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
