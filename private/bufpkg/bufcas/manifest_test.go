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
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManifest(t *testing.T) {
	t.Parallel()
	var digests []Digest
	var fileNodes []FileNode
	for i := 0; i < 10; i++ {
		digest, err := NewDigestForContent(strings.NewReader(fmt.Sprintf("content%d", i)))
		require.NoError(t, err)
		digests = append(digests, digest)
		fileNode, err := NewFileNode(fmt.Sprintf("%d", i), digest)
		require.NoError(t, err)
		fileNodes = append(fileNodes, fileNode)
	}
	manifest, err := NewManifest(fileNodes)
	require.NoError(t, err)

	sort.Slice(
		fileNodes,
		func(i int, j int) bool {
			return fileNodes[i].Path() < fileNodes[j].Path()
		},
	)
	manifestFileNodes := manifest.FileNodes()
	for i := 0; i < 10; i++ {
		digest := manifest.GetDigest(fmt.Sprintf("%d", i))
		require.NotNil(t, digest)
		assert.Equal(t, digests[i], digest)
		assert.Equal(t, fileNodes[i], manifestFileNodes[i])
	}

	manifestString := manifest.String()
	parsedManifest, err := ParseManifest(manifestString)
	require.NoError(t, err)

	// Do not use fileNodes, FileNodes() are sorted.
	assert.Equal(t, manifestFileNodes, parsedManifest.FileNodes())
}

func TestEmptyManifest(t *testing.T) {
	t.Parallel()
	manifest, err := NewManifest(nil)
	require.NoError(t, err)

	manifestString := manifest.String()
	assert.Empty(t, manifestString)
	parsedManifest, err := ParseManifest(manifestString)
	require.NoError(t, err)

	// Do not use fileNodes, FileNodes() are sorted.
	assert.Equal(t, manifest.FileNodes(), parsedManifest.FileNodes())
}

func TestParseManifestError(t *testing.T) {
	t.Parallel()
	testParseManifestError(t, "\n")
	testParseManifestError(t, "whoops\n")
	testParseManifestError(t, "shake256:1234  foo\n")
	testParseManifestError(t, "md5:d41d8cd98f00b204e9800998ecf8427e  foo\n")
	testParseManifestError(t, "bar  foo\n")
	testParseManifestError(t, "shake256:_  foo\n")
}

func testParseManifestError(t *testing.T, manifestString string) {
	_, err := ParseManifest(manifestString)
	assert.Error(t, err)
}
