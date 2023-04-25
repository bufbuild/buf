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

func TestNewModulePinForString(t *testing.T) {
	testNewModulePinForString(t, "empty", "", true)
	testNewModulePinForString(t, "missing pieces", "foo/foo", true)
	testNewModulePinForString(t, "extra pieces", "foo/foo/foo/foo:foo:foo", true)
	testNewModulePinForString(t, "missing remote", "/owner/repo:commit", true)
	testNewModulePinForString(t, "missing owner", "remote//repo:commit", true)
	testNewModulePinForString(t, "missing repo", "remote/owner/:commit", true)
	testNewModulePinForString(t, "missing commit", "remote/owner/repo:", true)
	testNewModulePinForString(t, "valid", "remote/owner/repo:commit", false)
	testNewModulePinForString(
		t, "valid with options", "remote/owner/repo:commit", false,
		NewModulePinForStringWithBranch("branch"),
		NewModulePinForStringWithCreateTime(time.Now()),
		NewModulePinForStringWithDigest("digest"),
	)
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

func testNewModulePinForString(
	t *testing.T,
	desc string,
	pin string,
	expectErr bool,
	opts ...NewModulePinForStringOption,
) {
	t.Helper()
	t.Run(desc, func(t *testing.T) {
		t.Parallel()
		_, err := newModulePinForString(pin, opts...)
		if expectErr {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	})
}
