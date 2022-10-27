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
	"errors"
	"fmt"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/bufbuild/buf/private/pkg/manifest"
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
	m, _ := manifest.NewFromBucket(ctx, bucket)
	digest, _ := m.Digest("foo")
	fmt.Printf("digest[:16]: %s\n", digest.Hex()[:16])
	path, _ := m.Paths(digest)
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
	content, err := manifest.New().MarshalText()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(content))
}

func TestAddContent(t *testing.T) {
	t.Parallel()
	// single entry
	m := manifest.New()
	err := m.AddContent("my/path", bytes.NewReader(nil))
	require.NoError(t, err)
	expect := fmt.Sprintf("%s  my/path\n", mkdigest(nil))
	retContent, err := m.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, expect, string(retContent))

	// failing content read
	expectedErr := errors.New("testing error")
	err = m.AddContent("my/path", iotest.ErrReader(expectedErr))
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
	_, err := manifest.NewFromReader(iotest.ErrReader(expected))
	assert.ErrorIs(t, err, expected)
}

func TestUnmarshalBrokenManifest(t *testing.T) {
	t.Parallel()
	var m manifest.Manifest
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
	m, err := manifest.NewFromBucket(ctx, bucket)
	require.NoError(t, err)
	expected := fmt.Sprintf("%s  foo\n", mkdigest([]byte("bar")))
	expected += fmt.Sprintf("%s  null\n", mkdigest(nil))
	retContent, err := m.MarshalText()
	assert.NoError(t, err)
	assert.Equal(t, expected, string(retContent))
}

func TestPaths(t *testing.T) {
	t.Parallel()
	m := manifest.New()
	err := m.AddContent("path/one", bytes.NewReader(nil))
	require.NoError(t, err)
	err = m.AddContent("path/two", bytes.NewReader(nil))
	require.NoError(t, err)
	paths, ok := m.Paths(mkdigest(nil))
	assert.True(t, ok)
	assert.ElementsMatch(t, []string{"path/one", "path/two"}, paths)
	paths, ok = m.Paths(mkdigest([]byte{0}))
	assert.False(t, ok)
	assert.Empty(t, paths)
}

func TestDigest(t *testing.T) {
	t.Parallel()
	m := manifest.New()
	err := m.AddContent("my/path", bytes.NewReader(nil))
	require.NoError(t, err)
	digest, ok := m.Digest("my/path")
	assert.True(t, ok)
	assert.Equal(t, mkdigest(nil), digest)
	digest, ok = m.Digest("foo")
	assert.False(t, ok)
	assert.Empty(t, digest)
}

func mkdigest(content []byte) *manifest.Digest {
	hash := sha3.NewShake256()
	if _, err := hash.Write(content); err != nil {
		panic(err)
	}
	digest := make([]byte, 64)
	if _, err := hash.Read(digest); err != nil {
		panic(err)
	}
	return manifest.NewDigestFromBytes("shake256", digest)
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
