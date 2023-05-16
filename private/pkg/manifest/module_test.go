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
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/bufbuild/buf/private/pkg/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMemoryBlob(t *testing.T) {
	t.Parallel()
	const content = "some file content"
	digest := mustDigestShake256(t, []byte(content))
	blob, err := manifest.NewMemoryBlob(
		*digest,
		[]byte(content),
		manifest.MemoryBlobWithDigestValidation(),
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

	t.Run("NoValidateDigest", func(t *testing.T) {
		t.Parallel()
		_, err := manifest.NewMemoryBlob(*digest, []byte("different content"))
		assert.NoError(t, err)
	})
	t.Run("ValidatingDigest", func(t *testing.T) {
		t.Parallel()
		_, err := manifest.NewMemoryBlob(
			*digest,
			[]byte("different content"),
			manifest.MemoryBlobWithDigestValidation(),
		)
		assert.Error(t, err)
	})
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

func TestNewBlobSetWithNilBlobs(t *testing.T) {
	t.Parallel()
	blobsWithNil := append(newBlobsArray(t), nil)
	t.Run("DefaultRejectNils", func(t *testing.T) {
		_, err := manifest.NewBlobSet(
			context.Background(),
			blobsWithNil,
		)
		require.Error(t, err)
	})
	t.Run("AllowNils", func(t *testing.T) {
		blobSet, err := manifest.NewBlobSet(
			context.Background(),
			blobsWithNil,
			manifest.BlobSetWithSkipNilBlobs(),
		)
		require.NoError(t, err)
		assert.NotNil(t, blobSet)
	})
}

func TestNewBlobInvalidDuplicates(t *testing.T) {
	t.Parallel()
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

func TestAllBlobs(t *testing.T) {
	t.Parallel()
	blobs := newBlobsArray(t)
	set, err := manifest.NewBlobSet(context.Background(), blobs)
	require.NoError(t, err)
	retBlobs := set.Blobs()
	assertBlobsAreEqual(t, blobs, retBlobs)
}

type mockBlob struct {
	digest  *manifest.Digest
	content io.Reader
	openErr bool
}

func (mb *mockBlob) Digest() *manifest.Digest { return mb.digest }
func (mb *mockBlob) Open(_ context.Context) (io.ReadCloser, error) {
	if mb.openErr {
		return nil, errors.New("open error")
	}
	return io.NopCloser(mb.content), nil
}

type errAtEndReader struct{ reader io.Reader }

func (e *errAtEndReader) Read(p []byte) (int, error) {
	n, err := e.reader.Read(p)
	if err == io.EOF {
		return n, errors.New("test erroring at EOF")
	}
	return n, err
}

func TestBlobEqual(t *testing.T) {
	testBlobEqual(
		t,
		"does equal",
		strings.NewReader("foo"),
		strings.NewReader("foo"),
		true,
		false,
	)
	testBlobEqual(
		t,
		"equal with differing length reads",
		strings.NewReader("foo"),
		iotest.OneByteReader(strings.NewReader("foo")),
		true,
		false,
	)
	testBlobEqual(
		t,
		"mismatched equal-length content",
		strings.NewReader("foo"),
		strings.NewReader("bar"),
		false,
		false,
	)
	testBlobEqual(
		t,
		"mismatched equal-length content data with error",
		iotest.DataErrReader(strings.NewReader("foo")),
		iotest.DataErrReader(strings.NewReader("bar")),
		false,
		false,
	)
	testBlobEqual(
		t,
		"mismatched longer left content",
		strings.NewReader("foofoo"),
		strings.NewReader("foo"),
		false,
		false,
	)
	testBlobEqual(
		t,
		"mismatched longer right content",
		strings.NewReader("foo"),
		strings.NewReader("foofoo"),
		false,
		false,
	)
	testBlobEqual(
		t,
		"fast error left read",
		iotest.ErrReader(errors.New("testing error")),
		strings.NewReader("foo"),
		false,
		true,
	)
	testBlobEqual(
		t,
		"fast error right read",
		strings.NewReader("foo"),
		iotest.ErrReader(errors.New("testing error")),
		false,
		true,
	)
	testBlobEqual(
		t,
		"late error left read",
		&errAtEndReader{reader: strings.NewReader("foo")},
		strings.NewReader("foo"),
		false,
		true,
	)
	testBlobEqual(
		t,
		"late error right read",
		strings.NewReader("foo"),

		&errAtEndReader{reader: strings.NewReader("foo")},
		false,
		true,
	)
	testBlobEqual(
		t,
		"middle error left read",
		iotest.TimeoutReader(iotest.OneByteReader(strings.NewReader("foo"))),
		strings.NewReader("foo"),
		false,
		true,
	)
	testBlobEqual(
		t,
		"middle error right read",
		strings.NewReader("foo"),
		iotest.TimeoutReader(iotest.OneByteReader(strings.NewReader("foo"))),
		false,
		true,
	)
}

func TestBlobEqualDigestMismatch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	digester, err := manifest.NewDigester(manifest.DigestTypeShake256)
	require.NoError(t, err)
	const foo = "foo"
	const bar = "bar"
	aDigest, err := digester.Digest(strings.NewReader(foo))
	require.NoError(t, err)
	bDigest, err := digester.Digest(strings.NewReader(bar))
	require.NoError(t, err)
	aBlob := &mockBlob{
		digest:  aDigest,
		content: strings.NewReader(foo),
	}
	bBlob := &mockBlob{
		digest:  bDigest,
		content: strings.NewReader(""),
	}
	equal, err := manifest.BlobEqual(ctx, aBlob, bBlob)
	assert.False(t, equal)
	assert.NoError(t, err)
}

func TestBlobEqualOpenError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	aBlob := &mockBlob{
		digest:  &manifest.Digest{},
		openErr: true,
	}
	bBlob := &mockBlob{
		digest: &manifest.Digest{},
	}
	equal, err := manifest.BlobEqual(ctx, aBlob, bBlob)
	assert.False(t, equal)
	assert.Error(t, err)
	aBlob = &mockBlob{
		digest: &manifest.Digest{},
	}
	bBlob = &mockBlob{
		digest:  &manifest.Digest{},
		openErr: true,
	}
	equal, err = manifest.BlobEqual(ctx, aBlob, bBlob)
	assert.False(t, equal)
	assert.Error(t, err)
}

func testBlobEqual(
	t *testing.T,
	desc string,
	a, b io.Reader,
	isEqual bool,
	isError bool,
) {
	t.Helper()
	t.Run(desc, func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		digest := &manifest.Digest{} // Avoid digest equality test.
		aBlob := &mockBlob{
			digest:  digest,
			content: a,
		}
		bBlob := &mockBlob{
			digest:  digest,
			content: b,
		}
		equal, err := manifest.BlobEqual(ctx, aBlob, bBlob)
		assert.Equal(t, isEqual, equal)
		if isError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	})
}

func newBlobsArray(t *testing.T) []manifest.Blob {
	var blobs []manifest.Blob
	for i := 0; i < 10; i++ {
		content := fmt.Sprintf("some content %d", i)
		digest := mustDigestShake256(t, []byte(content))
		blob, err := manifest.NewMemoryBlob(
			*digest, []byte(content),
			manifest.MemoryBlobWithDigestValidation(),
		)
		require.NoError(t, err)
		blobs = append(blobs, blob)
	}
	return blobs
}

// assertBlobsAreEqual makes sure all the blobs digests in the array are the
// same (assuming they're correctly built), ignoring order in the blobs arrays.
func assertBlobsAreEqual(t *testing.T, expectedBlobs []manifest.Blob, actualBlobs []manifest.Blob) {
	expectedDigests := make(map[string]struct{}, len(expectedBlobs))
	for _, expectedBlob := range expectedBlobs {
		expectedDigests[expectedBlob.Digest().String()] = struct{}{}
	}
	actualDigests := make(map[string]struct{}, len(actualBlobs))
	for _, actualBlob := range actualBlobs {
		actualDigests[actualBlob.Digest().String()] = struct{}{}
	}
	assert.Equal(t, expectedDigests, actualDigests)
}
