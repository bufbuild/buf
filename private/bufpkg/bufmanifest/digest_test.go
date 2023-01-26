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
	"errors"
	"fmt"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/bufbuild/buf/private/bufpkg/bufmanifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDigestBytes(t *testing.T) {
	t.Parallel()
	testInvalidDigestBytes(
		t,
		"empty",
		"",
		mustDigestShake256(t, nil).Bytes(),
	)
	testInvalidDigestBytes(
		t,
		"unsupported digest type",
		"md5",
		mustDigestShake256(t, nil).Bytes(),
	)
	testInvalidDigestBytes(
		t,
		"invalid digest",
		bufmanifest.DigestTypeShake256,
		nil,
	)
	testInvalidDigestBytes(
		t,
		"invalid digest",
		bufmanifest.DigestTypeShake256,
		mustDigestShake256(t, nil).Bytes()[:10],
	)
}

func TestNewDigestHex(t *testing.T) {
	t.Parallel()
	testInvalidDigestHex(
		t,
		"empty",
		"",
		mustDigestShake256(t, nil).Hex(),
	)
	testInvalidDigestHex(
		t,
		"unsupported digest type",
		"md5",
		mustDigestShake256(t, nil).Hex(),
	)
	testInvalidDigestHex(
		t,
		"invalid digest",
		bufmanifest.DigestTypeShake256,
		"",
	)
	testInvalidDigestHex(
		t,
		"encoding/hex",
		bufmanifest.DigestTypeShake256,
		"not-a_hex/string",
	)
	testInvalidDigestHex(
		t,
		"invalid digest",
		bufmanifest.DigestTypeShake256,
		mustDigestShake256(t, nil).Hex()[:10],
	)
}

func TestNewDigestString(t *testing.T) {
	t.Parallel()
	testInvalidDigestString(
		t,
		"malformed",
		"",
	)
	testInvalidDigestString(
		t,
		"malformed",
		"foo",
	)
	testInvalidDigestString(
		t,
		"malformed",
		"shake256 foo",
	)
	testInvalidDigestString(
		t,
		"encoding/hex",
		"shake256:_",
	)
	validDigest := mustDigestShake256(t, nil)
	testInvalidDigestString(
		t,
		"invalid digest",
		fmt.Sprintf("%s:%s", validDigest.Type(), validDigest.Hex()[:10]),
	)
	testInvalidDigestString(
		t,
		"unsupported digest type",
		fmt.Sprintf("md5:%s", validDigest.Hex()),
	)
}

func TestNewDigester(t *testing.T) {
	t.Parallel()
	digester, err := bufmanifest.NewDigester(bufmanifest.DigestTypeShake256)
	require.NoError(t, err)
	require.NotNil(t, digester)
	digester, err = bufmanifest.NewDigester("some unrecognized digest type")
	require.Error(t, err)
	require.Nil(t, digester)
}

func TestDigesterDigest(t *testing.T) {
	t.Parallel()
	digester, err := bufmanifest.NewDigester(bufmanifest.DigestTypeShake256)
	require.NoError(t, err)
	digest, err := digester.Digest(strings.NewReader("some content"))
	require.NoError(t, err)
	assert.Equal(t, bufmanifest.DigestTypeShake256, digest.Type())
	assert.NotEmpty(t, digest.Bytes())
	assert.NotEmpty(t, digest.String())

	// failing digesting content
	expectedErr := errors.New("testing error")
	digest, err = digester.Digest(iotest.ErrReader(expectedErr))
	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, digest)
}

func TestEqualDigests(t *testing.T) {
	t.Parallel()
	const fileContent = "one line\nanother line\nyet another one\n"
	d1 := mustDigestShake256(t, []byte(fileContent))
	d2 := mustDigestShake256(t, []byte(fileContent))
	d3 := mustDigestShake256(t, []byte("some other content"))
	assert.True(t, d1.Equal(*d2))
	assert.True(t, d2.Equal(*d1))
	assert.False(t, d1.Equal(*d3))
}

func mustDigestShake256(t *testing.T, content []byte) *bufmanifest.Digest {
	digester, err := bufmanifest.NewDigester(bufmanifest.DigestTypeShake256)
	require.NoError(t, err)
	require.NotNil(t, digester)
	digest, err := digester.Digest(bytes.NewReader(content))
	require.NoError(t, err)
	return digest
}

func testInvalidDigestString(
	t *testing.T,
	desc string,
	digest string,
) {
	t.Helper()
	t.Run(desc, func(t *testing.T) {
		t.Parallel()
		_, err := bufmanifest.NewDigestFromString(digest)
		assert.ErrorContains(t, err, desc)
	})
}

func testInvalidDigestHex(
	t *testing.T,
	desc string,
	dtype bufmanifest.DigestType,
	hexstr string,
) {
	t.Helper()
	t.Run(desc, func(t *testing.T) {
		t.Parallel()
		_, err := bufmanifest.NewDigestFromHex(dtype, hexstr)
		assert.ErrorContains(t, err, desc)
	})
}

func testInvalidDigestBytes(
	t *testing.T,
	desc string,
	dtype bufmanifest.DigestType,
	digest []byte,
) {
	t.Helper()
	t.Run(desc, func(t *testing.T) {
		t.Parallel()
		_, err := bufmanifest.NewDigestFromBytes(dtype, digest)
		assert.ErrorContains(t, err, desc)
	})
}
