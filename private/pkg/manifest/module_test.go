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

func TestBlobFromBytes(t *testing.T) {
	testBlobFromBytes(
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

func testBlobFromBytes(t *testing.T, content []byte, digest []byte) {
	t.Helper()
	t.Parallel()
	blob := manifest.NewBlobFromBytes(content)
	expect := &modulev1alpha1.Blob{
		Hash: &modulev1alpha1.Hash{
			Kind:   modulev1alpha1.HashKind_HASH_KIND_SHAKE256,
			Digest: digest,
		},
		Content: content,
	}
	assert.Equal(t, expect, blob)
}
