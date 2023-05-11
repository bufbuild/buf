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

package manifest_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/bufbuild/buf/private/pkg/manifest"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Example() {
	ctx := context.Background()
	bucket, _ := storagemem.NewReadBucket(
		map[string][]byte{
			"foo": []byte("bar"),
		},
	)
	m, _, _ := manifest.NewFromBucket(ctx, bucket)
	digest, _ := m.DigestFor("foo")
	fmt.Printf("digest[:16]: %s\n", digest.Hex()[:16])
	path, _ := m.PathsFor(digest.String())
	fmt.Printf("path at digest: %s\n", path[0])
	// Output:
	// digest[:16]: a15163728ed24e1c
	// path at digest: foo
}

func TestRoundTripManifest(t *testing.T) {
	t.Parallel()
	// read a manifest using the unmarshaling method
	null := mustDigestShake256(t, []byte{})
	var manifestBuilder bytes.Buffer
	for i := 0; i < 2; i++ {
		fmt.Fprintf(
			&manifestBuilder,
			"%s:%s  null%d\n",
			null.Type(),
			null.Hex(),
			i,
		)
	}
	manifestContent := manifestBuilder.Bytes()
	var m manifest.Manifest
	err := m.UnmarshalText(manifestContent)
	require.NoError(t, err)

	// marshaling the manifest back should produce an identical result
	retContent, err := m.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, manifestContent, retContent, "round trip failed")
}

func TestEmptyManifest(t *testing.T) {
	t.Parallel()
	content, err := new(manifest.Manifest).MarshalText()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(content))
}

func TestAddEntry(t *testing.T) {
	t.Parallel()
	var m manifest.Manifest
	fileDigest := mustDigestShake256(t, nil)
	const filePath = "my/path"
	require.NoError(t, m.AddEntry(filePath, *fileDigest))
	require.NoError(t, m.AddEntry(filePath, *fileDigest)) // adding the same entry twice is fine

	require.Error(t, m.AddEntry("", *fileDigest))
	require.Error(t, m.AddEntry("other/path", manifest.Digest{}))
	require.Error(t, m.AddEntry(filePath, *mustDigestShake256(t, []byte("other content"))))

	expect := fmt.Sprintf("%s  %s\n", fileDigest, filePath)
	retContent, err := m.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, expect, string(retContent))

	t.Run("clean-up-paths", func(t *testing.T) {
		t.Parallel()
		var m manifest.Manifest
		validDigest := mustDigestShake256(t, nil)
		// adding same entry many times, should be cleaned up.
		assert.NoError(t, m.AddEntry("./my/valid/path", *validDigest))
		assert.NoError(t, m.AddEntry("my///valid///path", *validDigest))
		assert.NoError(t, m.AddEntry("././my/valid/path", *validDigest))
		assert.NoError(t, m.AddEntry("./my/./valid/./path", *validDigest))
		assert.NoError(t, m.AddEntry("./my/valid/../../my/valid/path", *validDigest))
		assert.NoError(t, m.AddEntry("foo/../my/valid/path", *validDigest))
		paths := m.Paths()
		require.Len(t, paths, 1)
		assert.Equal(t, paths[0], "my/valid/path")
	})
	t.Run("invalid-paths", func(t *testing.T) {
		t.Parallel()
		var m manifest.Manifest
		validDigest := mustDigestShake256(t, nil)
		assert.Error(t, m.AddEntry("/invalid/absolute/path", *validDigest))
		assert.Error(t, m.AddEntry("../invalid/outer/path", *validDigest))
		assert.Error(t, m.AddEntry("invalid/../../outer/path", *validDigest))
		assert.Empty(t, m.Paths())
	})
}

func TestInvalidManifests(t *testing.T) {
	testInvalidManifest(
		t,
		"invalid entry",
		"\n",
	)
	testInvalidManifest(
		t,
		"invalid entry",
		"whoops\n",
	)
	testInvalidManifest(
		t,
		"invalid digest",
		"shake256:1234  foo\n",
	)
	testInvalidManifest(
		t,
		"unsupported digest type",
		"md5:d41d8cd98f00b204e9800998ecf8427e  foo\n",
	)
	testInvalidManifest(
		t,
		"malformed digest string",
		"bar  foo\n",
	)
	testInvalidManifest(
		t,
		"encoding/hex",
		"shake256:_  foo\n",
	)
	testInvalidManifest(
		t,
		"partial record",
		fmt.Sprintf("%s  null", mustDigestShake256(t, nil)),
	)
}

func TestBrokenRead(t *testing.T) {
	t.Parallel()
	expected := errors.New("testing error")
	_, err := manifest.NewFromReader(iotest.ErrReader(expected))
	assert.ErrorIs(t, err, expected)
}

func TestUnmarshalBrokenManifest(t *testing.T) {
	t.Parallel()
	var m manifest.Manifest
	err := m.UnmarshalText([]byte("foo"))
	assert.Error(t, err)
}

func TestDigestPaths(t *testing.T) {
	t.Parallel()
	var m manifest.Manifest
	sharedDigest := mustDigestShake256(t, nil)
	err := m.AddEntry("path/one", *sharedDigest)
	require.NoError(t, err)
	err = m.AddEntry("path/two", *sharedDigest)
	require.NoError(t, err)
	paths, ok := m.PathsFor(sharedDigest.String())
	assert.True(t, ok)
	assert.ElementsMatch(t, []string{"path/one", "path/two"}, paths)
	paths, ok = m.PathsFor(mustDigestShake256(t, []byte{0}).String())
	assert.False(t, ok)
	assert.Empty(t, paths)
}

func TestPathDigest(t *testing.T) {
	t.Parallel()
	var m manifest.Manifest
	digest := mustDigestShake256(t, nil)
	err := m.AddEntry("my/path", *digest)
	require.NoError(t, err)
	retDigest, ok := m.DigestFor("my/path")
	assert.True(t, ok)
	assert.Equal(t, digest, retDigest)
	retDigest, ok = m.DigestFor("foo")
	assert.False(t, ok)
	assert.Empty(t, retDigest)
}

func testInvalidManifest(
	t *testing.T,
	desc string,
	line string,
) {
	t.Helper()
	t.Run(desc, func(t *testing.T) {
		t.Parallel()
		_, err := manifest.NewFromReader(strings.NewReader(line))
		assert.ErrorContains(t, err, desc)
	})
}

func TestPathsDigestsRange(t *testing.T) {
	t.Parallel()
	var (
		m          manifest.Manifest
		addedPaths []string
		nilDigest  = mustDigestShake256(t, nil)
	)
	for i := 0; i < 20; i++ {
		path := fmt.Sprintf("path/to/file%0d", i)
		require.NoError(t, m.AddEntry(path, *nilDigest)) // all of the paths have the same digest/content
		addedPaths = append(addedPaths, path)
	}
	t.Run("AllPaths", func(t *testing.T) {
		t.Parallel()
		allPaths := m.Paths()
		assert.Equal(t, len(addedPaths), len(allPaths))
		assert.ElementsMatch(t, addedPaths, allPaths)
	})
	t.Run("AllDigests", func(t *testing.T) {
		t.Parallel()
		allDigests := m.Digests()
		require.Equal(t, 1, len(allDigests))
		assert.True(t, nilDigest.Equal(allDigests[0]))
	})
	t.Run("Range", func(t *testing.T) {
		t.Parallel()
		var iteratedPaths []string
		require.NoError(t, m.Range(func(path string, digest manifest.Digest) error {
			iteratedPaths = append(iteratedPaths, path)
			assert.True(t, digest.Equal(*nilDigest))
			return nil
		}))
		assert.ElementsMatch(t, addedPaths, iteratedPaths)
	})
}

func TestManifestZeroValue(t *testing.T) {
	t.Parallel()
	var m manifest.Manifest
	blob, err := m.Blob()
	require.NoError(t, err)
	assert.NotEmpty(t, blob.Digest().Hex())
	assert.True(t, m.Empty())
	assert.Empty(t, m.Paths())
	paths, ok := m.PathsFor("anything")
	assert.Empty(t, paths)
	assert.False(t, ok)
	digest, ok := m.DigestFor("anything")
	assert.Nil(t, digest)
	assert.False(t, ok)
	digester, err := manifest.NewDigester(manifest.DigestTypeShake256)
	require.NoError(t, err)
	emptyDigest, err := digester.Digest(bytes.NewReader(nil))
	require.NoError(t, err)
	require.NoError(t, m.AddEntry("a", *emptyDigest))
	digest, ok = m.DigestFor("a")
	assert.True(t, ok)
	assert.Equal(t, emptyDigest.Hex(), digest.Hex())
	assert.False(t, m.Empty())
}
