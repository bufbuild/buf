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

package bufsynctest

import (
	"fmt"
	"testing"

	"github.com/bufbuild/buf/private/buf/bufsync"
	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/pkg/git/gittest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDuplicateIdentities(t *testing.T, handler TestHandler, run runFunc) {
	moduleDirs := map[string]struct{}{
		"dir1": {},
		"dir2": {},
		"dir3": {},
	}
	var (
		moduleDirsToDuplicatedIdentities = make(map[string]bufmodule.ModuleFullName, len(moduleDirs))
		moduleDirsToDifferentIdentities  = make(map[string]bufmodule.ModuleFullName, len(moduleDirs))
		moduleDirsToNilIdentities        = make(map[string]bufmodule.ModuleFullName, len(moduleDirs))
	)
	repeatedIdentity, err := bufmodule.NewModuleFullName("buf.build", "acme", "foo")
	require.NoError(t, err)
	for moduleDir := range moduleDirs {
		moduleDirsToDuplicatedIdentities[moduleDir] = repeatedIdentity
		differentIdentity, err := bufmodule.NewModuleFullName("buf.build", "acme", moduleDir)
		require.NoError(t, err)
		moduleDirsToDifferentIdentities[moduleDir] = differentIdentity
		moduleDirsToNilIdentities[moduleDir] = nil
	}
	type testCase struct {
		name                    string
		modulesIdentitiesInHEAD map[string]bufmodule.ModuleFullName
		modulesOverrides        map[string]bufmodule.ModuleFullName
	}
	testCases := []testCase{
		{
			name:                    "when_dirs_have_duplicated_identities_in_HEAD_no_overrides",
			modulesIdentitiesInHEAD: moduleDirsToDuplicatedIdentities,
			modulesOverrides:        moduleDirsToNilIdentities,
		},
		{
			name:                    "when_dirs_have_different_identities_in_HEAD_but_duplicated_overrides",
			modulesIdentitiesInHEAD: moduleDirsToDifferentIdentities,
			modulesOverrides:        moduleDirsToDuplicatedIdentities,
		},
	}
	for _, tc := range testCases {
		func(tc testCase) {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				gitRepo := gittest.ScaffoldGitRepository(t)
				prepareGitRepoDuplicateIdentities(t, gitRepo, tc.modulesIdentitiesInHEAD)
				var moduleDirs []string
				for moduleDir := range tc.modulesIdentitiesInHEAD {
					moduleDirs = append(moduleDirs, moduleDir)
				}
				var opts []bufsync.SyncerOption
				for moduleDir, identityOverride := range tc.modulesOverrides {
					opts = append(opts, bufsync.SyncerWithModule(moduleDir, identityOverride))
				}
				_, err := run(t, gitRepo, opts...)
				require.Error(t, err)
				// TODO: not the greatest way to test this, maybe we should mak this a structured error
				assert.Contains(t, err.Error(), repeatedIdentity.IdentityString())
				assert.Contains(t, err.Error(), gittest.DefaultBranch)
				for _, moduleDir := range moduleDirs {
					assert.Contains(t, err.Error(), moduleDir)
				}
			})
		}(tc)
	}
}

// prepareGitRepoDuplicateIdentities commits valid modules to the passed directories and module identities.
func prepareGitRepoDuplicateIdentities(t *testing.T, repo gittest.Repository, moduleDirsToIdentities map[string]bufmodule.ModuleFullName) {
	files := make(map[string]string)
	for moduleDir, moduleFullName := range moduleDirsToIdentities {
		files[moduleDir+"/buf.yaml"] = fmt.Sprintf("version: v1\nname: %s\n", moduleFullName.String())
		files[moduleDir+"/foo/v1/foo.proto"] = "syntax = \"proto3\";\n\npackage foo.v1;\n\nmessage Foo {}\n"
	}
	repo.Commit(t, "commit", files)
}
