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
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDigestForContent(t *testing.T) {
	t.Parallel()
	digest, err := NewDigestForContent(bytes.NewBuffer(nil))
	require.NoError(t, err)
	assert.NotEqual(t, DigestType(0), digest.Type())
	assert.NotEmpty(t, digest.Value())

	digest, err = NewDigestForContent(strings.NewReader("some content"))
	require.NoError(t, err)
	assert.NotEqual(t, DigestType(0), digest.Type())
	assert.NotEmpty(t, digest.Value())

	// failing digesting content
	expectedErr := errors.New("testing error")
	digest, err = NewDigestForContent(iotest.ErrReader(expectedErr))
	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, digest)
}

func TestParseDigestError(t *testing.T) {
	t.Parallel()
	testParseDigestError(t, "")
	testParseDigestError(t, "foo")
	testParseDigestError(t, "shake256 foo")
	testParseDigestError(t, "shake256:_")
	validDigest, err := NewDigestForContent(bytes.NewBuffer(nil))
	require.NoError(t, err)
	validDigestHex := hex.EncodeToString(validDigest.Value())
	testParseDigestError(t, fmt.Sprintf("%s:%s", validDigest.Type(), validDigestHex[:10]))
	testParseDigestError(t, fmt.Sprintf("md5:%s", validDigestHex))
}

func TestDigestEqual(t *testing.T) {
	t.Parallel()
	fileContent := "one line\nanother line\nyet another one\n"
	d1, err := NewDigestForContent(strings.NewReader(fileContent))
	require.NoError(t, err)
	d2, err := NewDigestForContent(strings.NewReader(fileContent))
	require.NoError(t, err)
	d3, err := NewDigestForContent(strings.NewReader(fileContent + "foo"))
	require.NoError(t, err)
	assert.True(t, DigestEqual(d1, d2))
	assert.False(t, DigestEqual(d1, d3))
	assert.False(t, DigestEqual(d2, d3))
}

func testParseDigestError(t *testing.T, digestString string) {
	_, err := ParseDigest(digestString)
	assert.Error(t, err)
}
