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

package bufmodulestore

import (
	"context"
	"testing"

	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduletesting"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/pkg/filelock"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slogtestext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/require"
)

func TestCommitStoreBasic(t *testing.T) {
	t.Parallel()
	bucket := storagemem.NewReadWriteBucket()
	locker := filelock.NewNopLocker()
	testCommitStore(t, bucket, locker)
}

func TestCommitStoreOS(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	bucket, err := storageos.NewProvider().NewReadWriteBucket(tempDir)
	require.NoError(t, err)
	locker, err := filelock.NewLocker(tempDir)
	require.NoError(t, err)
	testCommitStore(t, bucket, locker)
}

func TestCommitStoreCorruptedEntry(t *testing.T) {
	t.Parallel()
	t.Run("invalid_json", func(t *testing.T) {
		t.Parallel()
		testCommitStoreCorruptedEntry(
			t,
			[]byte("invalid_json"),
		)
	})
	t.Run("invalid_fields", func(t *testing.T) {
		t.Parallel()
		// Valid JSON but missing required fields, so isValid() returns false.
		testCommitStoreCorruptedEntry(
			t,
			[]byte(`{"version":"v1","owner":"","module":"mod1","digest":"b5:fake"}`),
		)
	})
	t.Run("mismatched_digest_type", func(t *testing.T) {
		t.Parallel()
		// Valid JSON with all fields present, but digest type shake256 (B4) does not
		// match the b5 digest type used by the commit key.
		testCommitStoreCorruptedEntry(
			t,
			[]byte(`{"version":"v1","owner":"foo","module":"mod1","create_time":"2024-01-01T00:00:00Z","digest":"shake256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"}`),
		)
	})
}

func testCommitStoreCorruptedEntry(
	t *testing.T,
	corruptData []byte,
) {
	bucket := storagemem.NewReadWriteBucket()
	locker := filelock.NewNopLocker()
	ctx := t.Context()
	logger := slogtestext.NewLogger(t)
	commitStore := NewCommitStore(logger, bucket, locker)
	moduleKeys, commits := testGetModuleKeysAndCommits(t, ctx)

	err := commitStore.PutCommits(ctx, commits)
	require.NoError(t, err)

	// Corrupt the first entry.
	commitKey, err := bufmodule.ModuleKeyToCommitKey(moduleKeys[0])
	require.NoError(t, err)
	corruptPath := normalpath.Join(
		getCommitStoreDirPath(commitKey),
		getCommitStoreFilePath(commitKey),
	)
	require.NoError(t, storage.PutPath(ctx, bucket, corruptPath, corruptData))

	foundCommits, notFoundModuleKeys, err := commitStore.GetCommitsForModuleKeys(ctx, moduleKeys)
	require.NoError(t, err)
	testRequireCommitNamesEqual(
		t,
		[]string{
			"buf.build/foo/mod3",
			"buf.build/foo/mod2",
		},
		foundCommits,
	)
	testRequireModuleKeyNamesEqual(
		t,
		[]string{
			"buf.build/foo/mod1",
		},
		notFoundModuleKeys,
	)
}

func TestCommitStorePutIdempotent(t *testing.T) {
	t.Parallel()
	bucket := storagemem.NewReadWriteBucket()
	locker := filelock.NewNopLocker()
	ctx := t.Context()
	logger := slogtestext.NewLogger(t)
	commitStore := NewCommitStore(logger, bucket, locker)
	_, commits := testGetModuleKeysAndCommits(t, ctx)

	// Put twice -- second put should be a no-op.
	require.NoError(t, commitStore.PutCommits(ctx, commits))
	require.NoError(t, commitStore.PutCommits(ctx, commits))
}

func testCommitStore(
	t *testing.T,
	bucket storage.ReadWriteBucket,
	filelocker filelock.Locker,
) {
	ctx := t.Context()
	logger := slogtestext.NewLogger(t)
	commitStore := NewCommitStore(logger, bucket, filelocker)
	moduleKeys, commits := testGetModuleKeysAndCommits(t, ctx)

	// Nothing in cache yet.
	foundCommits, notFoundModuleKeys, err := commitStore.GetCommitsForModuleKeys(ctx, moduleKeys)
	require.NoError(t, err)
	testRequireCommitNamesEqual(t, nil, foundCommits)
	testRequireModuleKeyNamesEqual(
		t,
		[]string{
			"buf.build/foo/mod1",
			"buf.build/foo/mod3",
			"buf.build/foo/mod2",
		},
		notFoundModuleKeys,
	)

	// Put commits.
	err = commitStore.PutCommits(ctx, commits)
	require.NoError(t, err)

	// All should be found now via module keys.
	foundCommits, notFoundModuleKeys, err = commitStore.GetCommitsForModuleKeys(ctx, moduleKeys)
	require.NoError(t, err)
	testRequireCommitNamesEqual(
		t,
		[]string{
			"buf.build/foo/mod1",
			"buf.build/foo/mod3",
			"buf.build/foo/mod2",
		},
		foundCommits,
	)
	testRequireModuleKeyNamesEqual(t, nil, notFoundModuleKeys)

	// All should be found via commit keys.
	commitKeys, err := xslices.MapError(moduleKeys, bufmodule.ModuleKeyToCommitKey)
	require.NoError(t, err)
	foundCommits, notFoundCommitKeys, err := commitStore.GetCommitsForCommitKeys(ctx, commitKeys)
	require.NoError(t, err)
	testRequireCommitNamesEqual(
		t,
		[]string{
			"buf.build/foo/mod1",
			"buf.build/foo/mod3",
			"buf.build/foo/mod2",
		},
		foundCommits,
	)
	require.Empty(t, notFoundCommitKeys)
}

func testGetModuleKeysAndCommits(t *testing.T, ctx context.Context) ([]bufmodule.ModuleKey, []bufmodule.Commit) {
	bsrProvider, err := bufmoduletesting.NewOmniProvider(
		bufmoduletesting.ModuleData{
			Name: "buf.build/foo/mod1",
			PathToData: map[string][]byte{
				"mod1.proto": []byte(
					`syntax = "proto3"; package mod1;`,
				),
			},
		},
		bufmoduletesting.ModuleData{
			Name: "buf.build/foo/mod2",
			PathToData: map[string][]byte{
				"mod2.proto": []byte(
					`syntax = "proto3"; package mod2; import "mod1.proto";`,
				),
			},
		},
		bufmoduletesting.ModuleData{
			Name: "buf.build/foo/mod3",
			PathToData: map[string][]byte{
				"mod3.proto": []byte(
					`syntax = "proto3"; package mod3;`,
				),
			},
		},
	)
	require.NoError(t, err)
	moduleRefMod1, err := bufparse.NewRef("buf.build", "foo", "mod1", "")
	require.NoError(t, err)
	moduleRefMod2, err := bufparse.NewRef("buf.build", "foo", "mod2", "")
	require.NoError(t, err)
	moduleRefMod3, err := bufparse.NewRef("buf.build", "foo", "mod3", "")
	require.NoError(t, err)
	moduleKeys, err := bsrProvider.GetModuleKeysForModuleRefs(
		ctx,
		[]bufparse.Ref{
			moduleRefMod1,
			// Switching order on purpose.
			moduleRefMod3,
			moduleRefMod2,
		},
		bufmodule.DigestTypeB5,
	)
	require.NoError(t, err)
	testRequireModuleKeyNamesEqual(
		t,
		[]string{
			"buf.build/foo/mod1",
			"buf.build/foo/mod3",
			"buf.build/foo/mod2",
		},
		moduleKeys,
	)
	commits, err := bsrProvider.GetCommitsForModuleKeys(ctx, moduleKeys)
	require.NoError(t, err)
	testRequireCommitNamesEqual(
		t,
		[]string{
			"buf.build/foo/mod1",
			"buf.build/foo/mod3",
			"buf.build/foo/mod2",
		},
		commits,
	)
	return moduleKeys, commits
}

func testRequireCommitNamesEqual(t *testing.T, expected []string, actual []bufmodule.Commit) {
	if len(expected) == 0 {
		require.Empty(t, actual)
	} else {
		require.Equal(
			t,
			expected,
			xslices.Map(
				actual,
				func(value bufmodule.Commit) string {
					return value.ModuleKey().FullName().String()
				},
			),
		)
	}
}
