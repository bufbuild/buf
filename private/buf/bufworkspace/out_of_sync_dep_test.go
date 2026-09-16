// Copyright 2020-2026 Buf Technologies, Inc.
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

package bufworkspace

import (
	"context"
	"errors"
	"io/fs"
	"testing"

	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	commitIDOne   = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	commitIDTwo   = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	commitIDThree = uuid.MustParse("00000000-0000-0000-0000-000000000003")
)

func TestOutOfSyncDeps(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		// The refs as declared in the buf.yaml.
		configuredRefStrings []string
		// The commit that each ref resolves to on the BSR.
		refStringToResolvedCommitID map[string]uuid.UUID
		// The commit that each dep is pinned to in the buf.lock.
		fullNameStringToExistingCommitID map[string]uuid.UUID
		// The refs expected to be reported as out of sync.
		expectedOutOfSyncRefStrings []string
	}{
		{
			name:                 "in sync",
			configuredRefStrings: []string{"buf.build/foo/one:v1.0.0"},
			refStringToResolvedCommitID: map[string]uuid.UUID{
				"buf.build/foo/one:v1.0.0": commitIDOne,
			},
			fullNameStringToExistingCommitID: map[string]uuid.UUID{
				"buf.build/foo/one": commitIDOne,
			},
		},
		{
			name:                 "pinned ref does not match buf.lock",
			configuredRefStrings: []string{"buf.build/foo/one:v1.0.0"},
			refStringToResolvedCommitID: map[string]uuid.UUID{
				"buf.build/foo/one:v1.0.0": commitIDOne,
			},
			fullNameStringToExistingCommitID: map[string]uuid.UUID{
				"buf.build/foo/one": commitIDTwo,
			},
			expectedOutOfSyncRefStrings: []string{"buf.build/foo/one:v1.0.0"},
		},
		{
			name:                 "unpinned ref is never out of sync",
			configuredRefStrings: []string{"buf.build/foo/one"},
			refStringToResolvedCommitID: map[string]uuid.UUID{
				"buf.build/foo/one": commitIDOne,
			},
			fullNameStringToExistingCommitID: map[string]uuid.UUID{
				"buf.build/foo/one": commitIDTwo,
			},
		},
		{
			name:                 "pinned ref pruned from buf.lock is not out of sync",
			configuredRefStrings: []string{"buf.build/foo/one:v1.0.0"},
			refStringToResolvedCommitID: map[string]uuid.UUID{
				"buf.build/foo/one:v1.0.0": commitIDOne,
			},
			fullNameStringToExistingCommitID: map[string]uuid.UUID{},
		},
		{
			name: "only the out of sync deps are reported",
			configuredRefStrings: []string{
				"buf.build/foo/one:v1.0.0",
				"buf.build/foo/three:v3.0.0",
				"buf.build/foo/two",
			},
			refStringToResolvedCommitID: map[string]uuid.UUID{
				"buf.build/foo/one:v1.0.0":   commitIDOne,
				"buf.build/foo/two":          commitIDTwo,
				"buf.build/foo/three:v3.0.0": commitIDThree,
			},
			fullNameStringToExistingCommitID: map[string]uuid.UUID{
				"buf.build/foo/one":   commitIDOne,
				"buf.build/foo/two":   commitIDOne,
				"buf.build/foo/three": commitIDOne,
			},
			expectedOutOfSyncRefStrings: []string{"buf.build/foo/three:v3.0.0"},
		},
		{
			name:                             "no deps",
			fullNameStringToExistingCommitID: map[string]uuid.UUID{},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			configuredRefs, err := xslices.MapError(testCase.configuredRefStrings, bufparse.ParseRef)
			require.NoError(t, err)
			existingKeys, err := testModuleKeysForFullNameStringToCommitID(testCase.fullNameStringToExistingCommitID)
			require.NoError(t, err)
			actualOutOfSyncDeps, err := outOfSyncDeps(
				t.Context(),
				OutOfSyncDepTypeModule,
				configuredRefs,
				existingKeys,
				newTestKeysForRefsFunc(testCase.refStringToResolvedCommitID, nil),
			)
			require.NoError(t, err)
			actualOutOfSyncRefStrings := xslices.Map(
				actualOutOfSyncDeps,
				func(outOfSyncDep OutOfSyncDep) string {
					return outOfSyncDep.Ref().String()
				},
			)
			if len(testCase.expectedOutOfSyncRefStrings) == 0 {
				assert.Empty(t, actualOutOfSyncRefStrings)
			} else {
				assert.Equal(t, testCase.expectedOutOfSyncRefStrings, actualOutOfSyncRefStrings)
			}
		})
	}
}

func TestOutOfSyncDepsOnlyResolvesConstrainedDeps(t *testing.T) {
	t.Parallel()

	// Resolving a ref is a call to the BSR - we should not be making one for a dep that
	// cannot be out of sync.
	var requestedRefStrings []string
	configuredRefs, err := xslices.MapError(
		[]string{"buf.build/foo/one:v1.0.0", "buf.build/foo/two", "buf.build/foo/three:v3.0.0"},
		bufparse.ParseRef,
	)
	require.NoError(t, err)
	existingKeys, err := testModuleKeysForFullNameStringToCommitID(
		map[string]uuid.UUID{
			"buf.build/foo/one": commitIDOne,
			"buf.build/foo/two": commitIDTwo,
		},
	)
	require.NoError(t, err)
	_, err = outOfSyncDeps(
		t.Context(),
		OutOfSyncDepTypeModule,
		configuredRefs,
		existingKeys,
		newTestKeysForRefsFunc(
			map[string]uuid.UUID{"buf.build/foo/one:v1.0.0": commitIDOne},
			&requestedRefStrings,
		),
	)
	require.NoError(t, err)
	assert.Equal(t, []string{"buf.build/foo/one:v1.0.0"}, requestedRefStrings)
}

func TestNewOutOfSyncDepsError(t *testing.T) {
	t.Parallel()

	assert.NoError(t, NewOutOfSyncDepsError(nil))
	moduleRef, err := bufparse.ParseRef("buf.build/foo/one:v1.0.0")
	require.NoError(t, err)
	pluginRef, err := bufparse.ParseRef("buf.build/foo/two:v2.0.0")
	require.NoError(t, err)
	err = NewOutOfSyncDepsError(
		[]OutOfSyncDep{
			newOutOfSyncDep(moduleRef, OutOfSyncDepTypeModule, commitIDOne, commitIDTwo),
			newOutOfSyncDep(pluginRef, OutOfSyncDepTypePlugin, commitIDTwo, commitIDThree),
		},
	)
	require.Error(t, err)
	assert.Equal(
		t,
		`buf.lock is out of sync with buf.yaml:
	module buf.build/foo/one:v1.0.0 is declared in buf.yaml, but buf.lock pins commit 00000000000000000000000000000002 instead of 00000000000000000000000000000001
	plugin buf.build/foo/two:v2.0.0 is declared in buf.yaml, but buf.lock pins commit 00000000000000000000000000000003 instead of 00000000000000000000000000000002
Run "buf dep update" and "buf plugin update" to update buf.lock.`,
		err.Error(),
	)
}

func testModuleKeysForFullNameStringToCommitID(
	fullNameStringToCommitID map[string]uuid.UUID,
) ([]bufmodule.ModuleKey, error) {
	moduleKeys := make([]bufmodule.ModuleKey, 0, len(fullNameStringToCommitID))
	for fullNameString, commitID := range fullNameStringToCommitID {
		fullName, err := bufparse.ParseFullName(fullNameString)
		if err != nil {
			return nil, err
		}
		moduleKey, err := bufmodule.NewModuleKey(fullName, commitID, testDigestFunc)
		if err != nil {
			return nil, err
		}
		moduleKeys = append(moduleKeys, moduleKey)
	}
	return moduleKeys, nil
}

// newTestKeysForRefsFunc resolves Refs to commits based on the full Ref string, so that a
// pinned Ref and an unpinned Ref for the same module can resolve to different commits.
//
// If requestedRefStrings is non-nil, each requested Ref string is appended to it.
func newTestKeysForRefsFunc(
	refStringToCommitID map[string]uuid.UUID,
	requestedRefStrings *[]string,
) func(context.Context, []bufparse.Ref) ([]bufmodule.ModuleKey, error) {
	return func(_ context.Context, refs []bufparse.Ref) ([]bufmodule.ModuleKey, error) {
		moduleKeys := make([]bufmodule.ModuleKey, len(refs))
		for i, ref := range refs {
			if requestedRefStrings != nil {
				*requestedRefStrings = append(*requestedRefStrings, ref.String())
			}
			commitID, ok := refStringToCommitID[ref.String()]
			if !ok {
				return nil, &fs.PathError{Op: "read", Path: ref.String(), Err: fs.ErrNotExist}
			}
			moduleKey, err := bufmodule.NewModuleKey(ref.FullName(), commitID, testDigestFunc)
			if err != nil {
				return nil, err
			}
			moduleKeys[i] = moduleKey
		}
		return moduleKeys, nil
	}
}

func testDigestFunc() (bufmodule.Digest, error) {
	return nil, errors.New("digest not available in test")
}
