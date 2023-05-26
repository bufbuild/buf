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

package bufmanifest_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufmanifest"
	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDigestFromProtoDigest(t *testing.T) {
	t.Parallel()
	const (
		fileContent = "one line\nanother line\nyet another one\n"
	)
	digestFromContent, err := manifest.NewDigestFromBytes(
		manifest.DigestTypeShake256,
		mustDigestShake256(t, []byte(fileContent)).Bytes(),
	)
	require.NoError(t, err)
	assert.Equal(t, manifest.DigestTypeShake256, digestFromContent.Type())
	protoDigest := modulev1alpha1.Digest{
		DigestType: modulev1alpha1.DigestType_DIGEST_TYPE_SHAKE256,
		Digest:     digestFromContent.Bytes(),
	}
	digest, err := bufmanifest.NewDigestFromProtoDigest(&protoDigest)
	require.NoError(t, err)
	assert.Equal(t, digestFromContent.String(), digest.String())
}

func TestNewDigestFromProtoDigest(t *testing.T) {
	t.Parallel()
	digest := mustDigestShake256(t, []byte("my content"))
	retDigest, err := bufmanifest.NewDigestFromProtoDigest(&modulev1alpha1.Digest{
		DigestType: modulev1alpha1.DigestType_DIGEST_TYPE_SHAKE256,
		Digest:     digest.Bytes(),
	})
	require.NoError(t, err)
	assert.True(t, digest.Equal(*retDigest))
}

func TestInvalidNewDigestFromProtoDigest(t *testing.T) {
	t.Parallel()
	_, err := bufmanifest.NewDigestFromProtoDigest(nil)
	assert.Error(t, err)
	_, err = bufmanifest.NewDigestFromProtoDigest(&modulev1alpha1.Digest{
		DigestType: modulev1alpha1.DigestType_DIGEST_TYPE_UNSPECIFIED,
	})
	assert.Error(t, err)
	_, err = bufmanifest.NewDigestFromProtoDigest(&modulev1alpha1.Digest{
		DigestType: modulev1alpha1.DigestType_DIGEST_TYPE_SHAKE256,
		Digest:     []byte("invalid digest"),
	})
	assert.Error(t, err)
}

func TestProtoBlob(t *testing.T) {
	t.Parallel()
	content := []byte("hello world")
	digester, err := manifest.NewDigester(manifest.DigestTypeShake256)
	require.NoError(t, err)
	digest, err := digester.Digest(bytes.NewReader(content))
	require.NoError(t, err)
	blob, err := manifest.NewMemoryBlob(*digest, content)
	require.NoError(t, err)
	ctx := context.Background()
	protoBlob, err := bufmanifest.AsProtoBlob(ctx, blob)
	require.NoError(t, err)
	rtBlob, err := bufmanifest.NewBlobFromProto(protoBlob)
	require.NoError(t, err)
	equal, err := manifest.BlobEqual(ctx, blob, rtBlob)
	require.NoError(t, err)
	assert.True(t, equal)
}

func TestBlobFromReader(t *testing.T) {
	t.Parallel()
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
	blob, err := manifest.NewMemoryBlobFromReader(bytes.NewReader(content))
	require.NoError(t, err)
	protoBlob, err := bufmanifest.AsProtoBlob(context.Background(), blob)
	require.NoError(t, err)
	expect := &modulev1alpha1.Blob{
		Digest: &modulev1alpha1.Digest{
			DigestType: modulev1alpha1.DigestType_DIGEST_TYPE_SHAKE256,
			Digest:     digest,
		},
		Content: content,
	}
	assert.Equal(t, expect, protoBlob)
}

func mustDigestShake256(t *testing.T, content []byte) *manifest.Digest {
	digester, err := manifest.NewDigester(manifest.DigestTypeShake256)
	require.NoError(t, err)
	require.NotNil(t, digester)
	digest, err := digester.Digest(bytes.NewReader(content))
	require.NoError(t, err)
	return digest
}
