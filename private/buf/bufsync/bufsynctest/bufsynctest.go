package bufsynctest

import (
	"context"
	"fmt"
	"testing"

	"github.com/bufbuild/buf/private/buf/bufsync"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/git/gittest"
	"github.com/bufbuild/buf/private/pkg/storage/storagegit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type HandlerProvider func() bufsync.Handler

type runFunc func(
	t *testing.T,
	repo gittest.Repository,
	options []bufsync.SyncerOption,
) (bufsync.ExecutionPlan, error)

func RunTestSuite(t *testing.T, handlerProvider HandlerProvider) {
	makeRunFunc := func(handlerProvider HandlerProvider) runFunc {
		// handler is shared across all runs of a test case
		handler := handlerProvider()
		return func(
			t *testing.T,
			repo gittest.Repository,
			options []bufsync.SyncerOption,
		) (bufsync.ExecutionPlan, error) {
			syncer := newSyncer(t, handler, repo, options...)
			plan, err := syncer.Plan(context.Background())
			if err != nil {
				return plan, err
			}
			return plan, syncer.Sync(context.Background())
		}
	}

	t.Run("no_previous_sync_points", func(t *testing.T) {
		testNoPreviousSyncPoints(t, makeRunFunc(handlerProvider))
	})
}

func testNoPreviousSyncPoints(t *testing.T, run runFunc) {
	moduleIdentityInHEAD, err := bufmoduleref.NewModuleIdentity("buf.build", "acme", "foo")
	require.NoError(t, err)
	moduleIdentityOverride, err := bufmoduleref.NewModuleIdentity("buf.build", "acme", "bar")
	require.NoError(t, err)
	repo := gittest.ScaffoldGitRepository(t)
	prepareGitRepoSyncWithNoPreviousSyncPoints(t, repo, moduleIdentityInHEAD, gittest.DefaultBranch)
	type testCase struct {
		name            string
		branch          string
		expectedCommits int
	}
	testCases := []testCase{
		{
			name:            "when_main",
			branch:          gittest.DefaultBranch,
			expectedCommits: 4, // doesn't include initial scaffolding empty commit
		},
		{
			name:            "when_foo",
			branch:          "foo",
			expectedCommits: 3, // +1 for the branch fork point, which is synced again
		},
	}
	for _, withOverride := range []bool{false, true} {
		for _, tc := range testCases {
			func(tc testCase) {
				t.Run(fmt.Sprintf("%s/override_%t", tc.name, withOverride), func(t *testing.T) {
					// check out the branch to sync
					repo.Checkout(t, tc.branch)
					const moduleDir = "."
					var opts []bufsync.SyncerOption
					if withOverride {
						opts = append(opts, bufsync.SyncerWithModule(moduleDir, moduleIdentityOverride))
					} else {
						opts = append(opts, bufsync.SyncerWithModule(moduleDir, nil))
					}
					plan, err := run(t, repo, opts)
					require.NoError(t, err)
					identity := moduleIdentityInHEAD
					if withOverride {
						identity = moduleIdentityOverride
					}
					assert.False(t, plan.Nop())
					require.Len(t, plan.ModuleBranchesToSync(), 1)
					branch := plan.ModuleBranchesToSync()[0]
					assert.Equal(t, tc.branch, branch.BranchName())
					assert.Equal(t, identity, branch.TargetModuleIdentity())
					assert.Len(t, branch.CommitsToSync(), tc.expectedCommits)
				})
			}(tc)
		}
	}
}

// prepareGitRepoSyncWithNoPreviousSyncPoints writes and pushes commits in the repo with the
// following commits:
//
// | o-o----------o-----------------o (master)
// |   └o-o (foo) └o--------o (bar)
// |               └o (baz)
func prepareGitRepoSyncWithNoPreviousSyncPoints(
	t *testing.T,
	repo gittest.Repository,
	moduleIdentity bufmoduleref.ModuleIdentity,
	defaultBranchName string,
) {
	var allBranches = []string{defaultBranchName, "foo", "bar", "baz"}

	var commitsCounter int
	doEmptyCommits := func(numOfCommits int) {
		for i := 0; i < numOfCommits; i++ {
			commitsCounter++
			repo.Commit(t, fmt.Sprintf("commit %d", commitsCounter), nil)
		}
	}
	// write the base module in the root
	repo.Commit(t, "commit 0", map[string]string{
		"buf.yaml": fmt.Sprintf("version: v1\nname: %s\n", moduleIdentity.IdentityString()),
	})

	doEmptyCommits(1)
	repo.CheckoutB(t, allBranches[1])
	doEmptyCommits(2)
	repo.Checkout(t, defaultBranchName)
	doEmptyCommits(1)
	repo.CheckoutB(t, allBranches[2])
	doEmptyCommits(1)
	repo.CheckoutB(t, allBranches[3])
	doEmptyCommits(1)
	repo.Checkout(t, allBranches[2])
	doEmptyCommits(1)
	repo.Checkout(t, defaultBranchName)
	doEmptyCommits(1)
}

func newSyncer(
	t *testing.T,
	handler bufsync.Handler,
	repo gittest.Repository,
	options ...bufsync.SyncerOption,
) bufsync.Syncer {
	syncer, err := bufsync.NewSyncer(
		zaptest.NewLogger(t),
		repo,
		storagegit.NewProvider(repo.Objects()),
		handler,
		options...,
	)
	require.NoError(t, err)
	return syncer
}
