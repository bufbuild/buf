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
