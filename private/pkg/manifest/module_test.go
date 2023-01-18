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
	"io"
	"strings"
	"testing"
	"testing/iotest"

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
	digest, err := manifest.NewDigestFromProtoDigest(&protoDigest)
	require.NoError(t, err)
	assert.Equal(t, digestFromContent.String(), digest.String())
}

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

func TestNewDigestFromProtoDigest(t *testing.T) {
	t.Parallel()
	digest := mustDigestShake256(t, []byte("my content"))
	retDigest, err := manifest.NewDigestFromProtoDigest(&modulev1alpha1.Digest{
		DigestType: modulev1alpha1.DigestType_DIGEST_TYPE_SHAKE256,
		Digest:     digest.Bytes(),
	})
	require.NoError(t, err)
	assert.True(t, digest.Equal(*retDigest))
}

func TestInvalidNewDigestFromProtoDigest(t *testing.T) {
	t.Parallel()
	_, err := manifest.NewDigestFromProtoDigest(nil)
	assert.Error(t, err)
	_, err = manifest.NewDigestFromProtoDigest(&modulev1alpha1.Digest{
		DigestType: modulev1alpha1.DigestType_DIGEST_TYPE_UNSPECIFIED,
	})
	assert.Error(t, err)
	_, err = manifest.NewDigestFromProtoDigest(&modulev1alpha1.Digest{
		DigestType: modulev1alpha1.DigestType_DIGEST_TYPE_SHAKE256,
		Digest:     []byte("invalid digest"),
	})
	assert.Error(t, err)
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

func TestBlobFromReader(t *testing.T) {
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
	protoBlob, err := manifest.AsProtoBlob(ctx, blob)
	require.NoError(t, err)
	rtBlob, err := manifest.NewBlobFromProto(protoBlob)
	require.NoError(t, err)
	equal, err := manifest.BlobEqual(ctx, blob, rtBlob)
	require.NoError(t, err)
	assert.True(t, equal)
}

func testBlobFromReader(t *testing.T, content []byte, digest []byte) {
	t.Helper()
	t.Parallel()
	blob, err := manifest.NewMemoryBlobFromReader(bytes.NewReader(content))
	require.NoError(t, err)
	protoBlob, err := manifest.AsProtoBlob(context.Background(), blob)
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
