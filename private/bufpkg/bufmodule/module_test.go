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

package bufmodule

import (
	"context"
	"testing"
	"time"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testNewModuleForBucket(
	t *testing.T,
	desc string,
	files map[string][]byte,
	isError bool,
	isNil bool,
	pins []bufmoduleref.ModulePin,
	documentation string,
	license string,
) {
	t.Run(desc, func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		bucket, err := storagemem.NewReadBucket(files)
		require.NoError(t, err)
		module, err := newModuleForBucket(ctx, bucket)
		if isError {
			assert.Error(t, err, "isError")
			return
		}
		require.NoError(t, err)
		if isNil {
			assert.Nil(t, module, "isNil")
			return
		}
		require.NotNil(t, module, "!isNil")

		assert.Equal(t, pins, module.dependencyModulePins, "pins")
		assert.Equal(t, documentation, module.documentation, "documentation")
		assert.Equal(t, license, module.license, "license")
	})
}

func TestNewModuleForBucket(t *testing.T) {
	testNewModuleForBucket(t,
		"an empty bucket is a valid parse",
		map[string][]byte{},
		false,
		false,
		[]bufmoduleref.ModulePin{},
		"",
		"",
	)

	wantPin, err := bufmoduleref.NewModulePin(
		"foo",
		"bar",
		"baz",
		"",
		"62f35d8aed1149c291d606d958a7ce32",
		"shake512:11223344",
		time.Time{},
	)
	require.NoError(t, err)
	testNewModuleForBucket(t,
		"pins are consumed",
		map[string][]byte{
			"buf.lock": []byte(`
version: v1
deps:
  - remote: foo
    owner: bar
    repository: baz
    commit: 62f35d8aed1149c291d606d958a7ce32
    digest: shake512:11223344
`),
		},
		false,
		false,
		[]bufmoduleref.ModulePin{wantPin},
		"",
		"",
	)

	testNewModuleForBucket(t,
		"license and documentation are consumed",
		map[string][]byte{
			"buf.md":  []byte("foo"),
			"LICENSE": []byte("bar"),
		},
		false,
		false,
		[]bufmoduleref.ModulePin{},
		"foo",
		"bar",
	)

	testNewModuleForBucket(t,
		"invalid buf.lock",
		map[string][]byte{
			"buf.lock": []byte("version: v0"),
		},
		true,
		false,
		[]bufmoduleref.ModulePin{},
		"",
		"",
	)
}
