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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBlobForContent(t *testing.T) {
	t.Parallel()
	content := "some file content"
	digest, err := NewDigestForContent(strings.NewReader(content))
	require.NoError(t, err)
	blob, err := NewBlobForContent(
		strings.NewReader(content),
		BlobWithKnownDigest(digest),
	)
	require.NoError(t, err)
	assert.True(t, DigestEqual(blob.Digest(), digest))
	assert.Equal(t, []byte(content), blob.Content())

	differentContent := "some different file content"
	differentDigest, err := NewDigestForContent(strings.NewReader(differentContent))
	_, err = NewBlobForContent(
		strings.NewReader(content),
		BlobWithKnownDigest(differentDigest),
	)
	require.Error(t, err)
}
