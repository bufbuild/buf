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
	"fmt"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/storage/storagegit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestPrepareSyncDuplicateIdentities(t *testing.T) {
	t.Parallel()
	moduleDirs := map[string]struct{}{
		"dir1": {},
		"dir2": {},
		"dir3": {},
	}
	var (
		moduleDirsToDuplicatedIdentities = make(map[string]bufmoduleref.ModuleIdentity, len(moduleDirs))
		moduleDirsToDifferentIdentities  = make(map[string]bufmoduleref.ModuleIdentity, len(moduleDirs))
		moduleDirsToNilIdentities        = make(map[string]bufmoduleref.ModuleIdentity, len(moduleDirs))
	)
	repeatedIdentity, err := bufmoduleref.NewModuleIdentity("buf.build", "acme", "foo")
	require.NoError(t, err)
	for moduleDir := range moduleDirs {
		moduleDirsToDuplicatedIdentities[moduleDir] = repeatedIdentity
		differentIdentity, err := bufmoduleref.NewModuleIdentity("buf.build", "acme", moduleDir)
		require.NoError(t, err)
		moduleDirsToDifferentIdentities[moduleDir] = differentIdentity
		moduleDirsToNilIdentities[moduleDir] = nil
	}
	type testCase struct {
		name                    string
		modulesIdentitiesInHEAD map[string]bufmoduleref.ModuleIdentity
		modulesOverrides        map[string]bufmoduleref.ModuleIdentity
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
				const defaultBranchName = "main"
				repo, repoDir := scaffoldGitRepository(t, defaultBranchName)
				prepareGitRepoMultiModule(t, repoDir, tc.modulesIdentitiesInHEAD)
				var moduleDirs []string
				for moduleDir := range tc.modulesIdentitiesInHEAD {
					moduleDirs = append(moduleDirs, moduleDir)
				}
				testSyncer := syncer{
					repo:                                  repo,
					storageGitProvider:                    storagegit.NewProvider(repo.Objects()),
					logger:                                zaptest.NewLogger(t),
					sortedModulesDirsForSync:              moduleDirs,
					modulesDirsToIdentityOverrideForSync:  tc.modulesOverrides,
					commitsToTags:                         make(map[string][]string),
					modulesDirsToBranchesToIdentities:     make(map[string]map[string]bufmoduleref.ModuleIdentity),
					modulesToBranchesExpectedSyncPoints:   make(map[string]map[string]string),
					modulesIdentitiesToCommitsSyncedCache: make(map[string]map[string]struct{}),
					errorHandler:                          &mockErrorHandler{},
				}
				prepareErr := testSyncer.prepareSync(context.Background())
				require.Error(t, prepareErr)
				assert.Contains(t, prepareErr.Error(), repeatedIdentity.IdentityString())
				assert.Contains(t, prepareErr.Error(), defaultBranchName)
				for _, moduleDir := range moduleDirs {
					assert.Contains(t, prepareErr.Error(), moduleDir)
				}
			})
		}(tc)
	}
}

// prepareGitRepoMultiModule commits valid modules to the passed directories and module identities.
func prepareGitRepoMultiModule(t *testing.T, repoDir string, moduleDirsToIdentities map[string]bufmoduleref.ModuleIdentity) {
	runner := command.NewRunner()
	for moduleDir, moduleIdentity := range moduleDirsToIdentities {
		writeFiles(t, repoDir, map[string]string{
			moduleDir + "/buf.yaml":         fmt.Sprintf("version: v1\nname: %s\n", moduleIdentity.IdentityString()),
			moduleDir + "/foo/v1/foo.proto": "syntax = \"proto3\";\n\npackage foo.v1;\n\nmessage Foo {}\n",
		})
	}
	runInDir(t, runner, repoDir, "git", "add", ".")
	runInDir(t, runner, repoDir, "git", "commit", "-m", "commit")
}
