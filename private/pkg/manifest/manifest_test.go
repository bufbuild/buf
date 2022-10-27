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

package manifest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"
	"testing/iotest"

	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
)

func Example() {
	ctx := context.Background()
	bucket, _ := storagemem.NewReadBucket(
		map[string][]byte{
			"foo": []byte("bar"),
		},
	)
	manifest, _ := NewManifestFromBucket(ctx, bucket)
	digest, _ := manifest.DigestFor("foo")
	fmt.Printf("digest[:16]: %s\n", digest.Hex()[:16])
	path, _ := manifest.PathsFor(digest)
	fmt.Printf("path at digest: %s\n", path[0])
	// Output:
	// digest[:16]: a15163728ed24e1c
	// path at digest: foo
}

func TestRoundTripManifest(t *testing.T) {
	t.Parallel()
	// read a manifest using the unmarshaling method
	null := mkdigest([]byte{})
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
	var m Manifest
	err := m.UnmarshalText(manifestContent)
	require.NoError(t, err)

	// marshaling the manifest back should produce an identical result
	retContent, err := m.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, manifestContent, retContent, "round trip failed")
}

func TestEmptyManifest(t *testing.T) {
	t.Parallel()
	content, err := NewManifest().MarshalText()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(content))
}

func TestAddContent(t *testing.T) {
	t.Parallel()
	// single entry
	manifest := NewManifest()
	err := manifest.AddContent("my/path", bytes.NewReader(nil))
	require.NoError(t, err)
	expect := fmt.Sprintf("%s  my/path\n", mkdigest(nil))
	retContent, err := manifest.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, expect, string(retContent))

	// failing content read
	expectedErr := errors.New("testing error")
	err = manifest.AddContent("my/path", iotest.ErrReader(expectedErr))
	assert.ErrorIs(t, err, expectedErr)
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
		"unsupported hash",
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
		fmt.Sprintf("%s  null", mkdigest(nil)),
	)
}

func TestBrokenRead(t *testing.T) {
	t.Parallel()
	expected := errors.New("testing error")
	_, err := NewManifestFromReader(iotest.ErrReader(expected))
	assert.ErrorIs(t, err, expected)
}

func TestUnmarshalBrokenManifest(t *testing.T) {
	t.Parallel()
	var m Manifest
	err := m.UnmarshalText([]byte("foo"))
	assert.Error(t, err)
}

func TestFromBucket(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	bucket, err := storagemem.NewReadBucket(
		map[string][]byte{
			"null": nil,
			"foo":  []byte("bar"),
		})
	require.NoError(t, err)
	m, err := NewManifestFromBucket(ctx, bucket)
	require.NoError(t, err)
	expected := fmt.Sprintf("%s  foo\n", mkdigest([]byte("bar")))
	expected += fmt.Sprintf("%s  null\n", mkdigest(nil))
	retContent, err := m.MarshalText()
	assert.NoError(t, err)
	assert.Equal(t, expected, string(retContent))
}

func TestDigestPaths(t *testing.T) {
	t.Parallel()
	m := NewManifest()
	err := m.AddContent("path/one", bytes.NewReader(nil))
	require.NoError(t, err)
	err = m.AddContent("path/two", bytes.NewReader(nil))
	require.NoError(t, err)
	paths, ok := m.PathsFor(mkdigest(nil))
	assert.True(t, ok)
	assert.ElementsMatch(t, []string{"path/one", "path/two"}, paths)
	paths, ok = m.PathsFor(mkdigest([]byte{0}))
	assert.False(t, ok)
	assert.Empty(t, paths)
}

func TestPathDigest(t *testing.T) {
	t.Parallel()
	m := NewManifest()
	err := m.AddContent("my/path", bytes.NewReader(nil))
	require.NoError(t, err)
	digest, ok := m.DigestFor("my/path")
	assert.True(t, ok)
	assert.Equal(t, mkdigest(nil), digest)
	digest, ok = m.DigestFor("foo")
	assert.False(t, ok)
	assert.Empty(t, digest)
}

func mkdigest(content []byte) *Digest {
	hash := sha3.NewShake256()
	if _, err := hash.Write(content); err != nil {
		panic(err)
	}
	digest := make([]byte, 64)
	if _, err := hash.Read(digest); err != nil {
		panic(err)
	}
	return NewDigestFromBytes("shake256", digest)
}

func testInvalidManifest(
	t *testing.T,
	desc string,
	line string,
) {
	t.Helper()
	t.Run(desc, func(t *testing.T) {
		t.Parallel()
		_, err := NewManifestFromReader(strings.NewReader(line))
		assert.ErrorContains(t, err, desc)
	})
}

func TestDigestValidator(t *testing.T) {
	t.Parallel()
	const content = "the content"
	digest := NewDigestFromBytes("not-supported-dtype", []byte(content))
	require.NotNil(t, digest)
	_, err := digest.Valid(strings.NewReader(content))
	require.ErrorContains(t, err, "unsupported hash")
}

func TestValidateContent(t *testing.T) {
	t.Parallel()
	const (
		filePath    = "path/to/file"
		fileContent = "one line\nanother line\nyet another one\n"
	)
	m := NewManifest()
	require.NoError(t, m.AddContent(filePath, strings.NewReader(fileContent)))
	fileDigest, ok := m.DigestFor(filePath)
	require.True(t, ok)
	require.NotNil(t, fileDigest)
	valid, err := fileDigest.Valid(strings.NewReader(fileContent))
	require.NoError(t, err)
	assert.True(t, valid)
	valid, err = fileDigest.Valid(strings.NewReader("some other content"))
	require.NoError(t, err)
	assert.False(t, valid)
}

func TestDigestFromBlobHash(t *testing.T) {
	t.Parallel()
	const (
		filePath    = "path/to/file"
		fileContent = "one line\nanother line\nyet another one\n"
	)
	typesToKinds := map[string]modulev1alpha1.HashKind{
		"shake256": modulev1alpha1.HashKind_HASH_KIND_SHAKE256,
	}
	for supportedType, supportedKind := range typesToKinds {
		digestFromFile := NewDigestFromBytes(supportedType, []byte(fileContent))
		require.NotNil(t, digestFromFile)
		assert.Equal(t, supportedType, digestFromFile.Type())
		blobHash := modulev1alpha1.Hash{
			Kind:   supportedKind,
			Digest: digestFromFile.Bytes(),
		}
		digestFromBlobHash, err := NewDigestFromBlobHash(&blobHash)
		require.NoError(t, err)
		assert.Equal(t, digestFromFile.String(), digestFromBlobHash.String())
	}
}

func TestManifestPaths(t *testing.T) {
	t.Parallel()
	m := NewManifest()
	for i := 0; i < 20; i++ {
		err := m.AddContent(
			fmt.Sprintf("path/to/file%0d", i),
			bytes.NewReader(nil),
		)
		require.NoError(t, err)
	}
	sortedPaths := m.Paths()
	assert.True(t, sort.SliceIsSorted(sortedPaths, func(i, j int) bool {
		return sortedPaths[i] < sortedPaths[j] // lexicographically sorted
	}))
}
