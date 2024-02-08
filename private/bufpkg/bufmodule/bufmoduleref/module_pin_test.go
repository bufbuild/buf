// Copyright 2020-2024 Buf Technologies, Inc.
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

package bufmoduleref

import (
	"bytes"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewModulePin(t *testing.T) {
	t.Parallel()
	nilDigest, err := bufcas.NewDigestForContent(bytes.NewBuffer(nil))
	require.NoError(t, err)
	testNewModulePin(t, "no digest", "", true)
	testNewModulePin(t, "nominal digest", nilDigest.String(), false)
}

func testNewModulePin(
	t *testing.T,
	desc string,
	digest string,
	expectEmptyDigest bool,
) {
	t.Helper()
	t.Run(desc, func(t *testing.T) {
		t.Parallel()
		pin, err := NewModulePin(
			"remote",
			"owner",
			"repository",
			"commit",
			digest,
		)
		assert.NoError(t, err)
		if expectEmptyDigest {
			assert.Equal(t, "", pin.Digest())
		} else {
			assert.Equal(t, digest, pin.Digest())
		}
	})
}
