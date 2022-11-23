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

package bufmoduleref

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewModulePin(t *testing.T) {
	testNewModulePin(t, "no digest", "", true)
	testNewModulePin(t, "nominal digest", "shake256:11223344", false)
	testNewModulePin(t, "b1 digest", "b1-11223344", true)
	testNewModulePin(t, "b3 digest", "b3-11223344", true)
}

func testNewModulePin(
	t *testing.T,
	desc string,
	digest string,
	zeroDigest bool,
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
		if zeroDigest {
			assert.Equal(t, "", pin.Digest())
		} else {
			assert.Equal(t, digest, pin.Digest())
		}
	})
}
