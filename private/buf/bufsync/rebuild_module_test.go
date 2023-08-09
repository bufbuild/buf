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

package bufsync

import (
	"context"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRebuildModule(t *testing.T) {
	t.Parallel()
	identity1, err := bufmoduleref.NewModuleIdentity("buf.first", "foo", "bar")
	require.NoError(t, err)
	identity2, err := bufmoduleref.NewModuleIdentity("buf.second", "baz", "qux")
	require.NoError(t, err)
	moduleBucket, err := storagemem.NewReadBucket(map[string][]byte{
		"buf.yaml":         []byte("version: v1\n"),
		"foo/v1/foo.proto": []byte(`syntax = "proto3";\nmessage Test {}\n`),
	})
	require.NoError(t, err)
	emptyConfig, err := bufmoduleconfig.NewConfigV1(bufmoduleconfig.ExternalConfigV1{})
	require.NoError(t, err)
	type testCase struct {
		name                   string
		originalModuleIdentity bufmoduleref.ModuleIdentity
	}
	testCases := []testCase{
		{
			name: "when_no_module_identity",
		},
		{
			name:                   "when_some_module_identity",
			originalModuleIdentity: identity1,
		},
	}
	for _, tc := range testCases {
		func(tc testCase) {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				originalModule, err := bufmodulebuild.NewModuleBucketBuilder().BuildForBucket(
					context.Background(),
					moduleBucket,
					emptyConfig,
					bufmodulebuild.WithModuleIdentity(tc.originalModuleIdentity),
				)
				require.NoError(t, err)
				renamedModule, err := renameModule(
					context.Background(),
					originalModule,
					identity2,
				)
				require.NoError(t, err)
				require.NotNil(t, renamedModule.ModuleIdentity())
				assert.Equal(t, identity2.IdentityString(), renamedModule.ModuleIdentity().IdentityString())
			})
		}(tc)
	}
}

func TestRebuildModuleFailures(t *testing.T) {
	t.Parallel()
	moduleBucket, err := storagemem.NewReadBucket(map[string][]byte{
		"buf.yaml":         []byte("version: v1\n"),
		"foo/v1/foo.proto": []byte(`syntax = "proto3";\nmessage Test {}\n`),
	})
	require.NoError(t, err)
	emptyConfig, err := bufmoduleconfig.NewConfigV1(bufmoduleconfig.ExternalConfigV1{})
	require.NoError(t, err)
	validModule, err := bufmodulebuild.NewModuleBucketBuilder().BuildForBucket(
		context.Background(),
		moduleBucket,
		emptyConfig,
	)
	require.NoError(t, err)
	validIdentity, err := bufmoduleref.NewModuleIdentity("buf.test", "acme", "foo")
	require.NoError(t, err)
	type testCase struct {
		name                    string
		missingOriginalModule   bool
		missingIdentityOverride bool
	}
	testCases := []testCase{
		{
			name:                  "when_no_original_module",
			missingOriginalModule: true,
		},
		{
			name:                    "when_no_new_identity",
			missingIdentityOverride: true,
		},
	}
	for _, tc := range testCases {
		func(tc testCase) {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var (
					originalModule *bufmodulebuild.BuiltModule
					newIdentity    bufmoduleref.ModuleIdentity
				)
				if !tc.missingOriginalModule {
					originalModule = validModule
				}
				if !tc.missingIdentityOverride {
					newIdentity = validIdentity
				}
				renamedModule, err := renameModule(
					context.Background(),
					originalModule,
					newIdentity,
				)
				require.Nil(t, renamedModule)
				require.Error(t, err)
			})
		}(tc)
	}
}
