// Copyright 2020-2025 Buf Technologies, Inc.
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

// Making sure that ParseErrors work outside of the cas package.
package cas_test

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/bufbuild/buf/private/pkg/cas"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDigestForContent(t *testing.T) {
	t.Parallel()
	digest, err := cas.NewDigestForContent(bytes.NewBuffer(nil))
	require.NoError(t, err)
	assert.Equal(t, cas.DigestTypeShake256, digest.Type())
	assert.NotEmpty(t, digest.Value())

	digest, err = cas.NewDigestForContent(strings.NewReader("some content"))
	require.NoError(t, err)
	assert.Equal(t, cas.DigestTypeShake256, digest.Type())
	assert.NotEmpty(t, digest.Value())

	digest, err = cas.NewDigestForContent(bytes.NewBuffer(nil), cas.DigestWithDigestType(cas.DigestTypeSha256))
	require.NoError(t, err)
	assert.Equal(t, cas.DigestTypeSha256, digest.Type())
	assert.NotEmpty(t, digest.Value())

	digest, err = cas.NewDigestForContent(strings.NewReader("some content"), cas.DigestWithDigestType(cas.DigestTypeSha256))
	require.NoError(t, err)
	assert.Equal(t, cas.DigestTypeSha256, digest.Type())
	assert.NotEmpty(t, digest.Value())

	// failing digesting content
	expectedErr := errors.New("testing error")
	digest, err = cas.NewDigestForContent(iotest.ErrReader(expectedErr))
	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, digest)
}

func TestParseDigestError(t *testing.T) {
	t.Parallel()
	testParseDigestError(t, "", false)
	testParseDigestError(t, "foo", true)
	testParseDigestError(t, "shake256 foo", true)
	testParseDigestError(t, "shake256:_", true)
	testParseDigestError(t, "sha256 foo", true)
	testParseDigestError(t, "sha256:_", true)
	validDigest, err := cas.NewDigestForContent(bytes.NewBuffer(nil))
	require.NoError(t, err)
	validDigestHex := hex.EncodeToString(validDigest.Value())
	testParseDigestError(t, fmt.Sprintf("%s:%s", validDigest.Type(), validDigestHex[:10]), true)
	testParseDigestError(t, fmt.Sprintf("md5:%s", validDigestHex), true)
}

func TestDigestEqual(t *testing.T) {
	t.Parallel()
	fileContent := "one line\nanother line\nyet another one\n"
	d1, err := cas.NewDigestForContent(strings.NewReader(fileContent))
	require.NoError(t, err)
	d2, err := cas.NewDigestForContent(strings.NewReader(fileContent))
	require.NoError(t, err)
	d3, err := cas.NewDigestForContent(strings.NewReader(fileContent + "foo"))
	require.NoError(t, err)
	d4, err := cas.NewDigestForContent(strings.NewReader(fileContent), cas.DigestWithDigestType(cas.DigestTypeSha256))
	require.NoError(t, err)
	d5, err := cas.NewDigestForContent(strings.NewReader(fileContent), cas.DigestWithDigestType(cas.DigestTypeSha256))
	require.NoError(t, err)
	assert.True(t, cas.DigestEqual(d1, d2))
	assert.True(t, cas.DigestEqual(d4, d5))
	assert.False(t, cas.DigestEqual(d1, d3))
	assert.False(t, cas.DigestEqual(d1, d4))
	assert.False(t, cas.DigestEqual(d2, d3))
	assert.False(t, cas.DigestEqual(d2, d4))
}

func testParseDigestError(t *testing.T, digestString string, expectParseError bool) {
	_, err := cas.ParseDigest(digestString)
	assert.Error(t, err)
	parseError := &cas.ParseError{}
	isParseError := errors.As(err, &parseError)
	if expectParseError {
		assert.True(t, isParseError)
		assert.Equal(t, digestString, parseError.Input())
	} else {
		assert.False(t, isParseError)
	}
}
