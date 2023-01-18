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

package bufmoduleref

import (
	"bytes"
	"testing"
	"time"

	"github.com/bufbuild/buf/private/pkg/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewModulePin(t *testing.T) {
	digester, err := manifest.NewDigester(manifest.DigestTypeShake256)
	require.NoError(t, err)
	nullDigest, err := digester.Digest(&bytes.Buffer{})
	require.NoError(t, err)
	testNewModulePin(t, "no digest", "", true)
	testNewModulePin(t, "nominal digest", nullDigest.String(), false)
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
			"branch",
			"commit",
			digest,
			time.Now(),
		)
		assert.NoError(t, err)
		if expectEmptyDigest {
			assert.Equal(t, "", pin.Digest())
		} else {
			assert.Equal(t, digest, pin.Digest())
		}
	})
}
