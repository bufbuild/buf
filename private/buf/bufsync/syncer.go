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
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/slicesextended"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagegit"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

const (
	// LookbackCommitsLimit is the amount of commits that we will look back before the start sync
	// point to backfill old git tags. We might allow customizing this value in the future.
	LookbackCommitsLimit = 5
	// LookbackTimeLimit is how old we will look back (git commit timestamps) before the start sync
	// point to backfill old git tags. We might allow customizing this value in the future.
	LookbackTimeLimit = 24 * time.Hour
)

var (
	errReadModuleInvalidModule       = errors.New("invalid module")
	errReadModuleInvalidModuleConfig = errors.New("invalid module config")
)

type syncer struct {
	logger             *zap.Logger
	repo               git.Repository
	storageGitProvider storagegit.Provider
	handler            Handler
	clock              Clock

	// flags received on creation
	gitRemoteName                        string
	sortedModulesDirsForSync             []string
	modulesDirsToIdentityOverrideForSync map[string]bufmoduleref.ModuleIdentity // moduleDir:moduleIdentityOverride
	syncAllBranches                      bool

	// commitsToTags holds all tags in the repo associated to its commit hash. (commit:[]tags)
	commitsToTags map[string][]string
	// modulesDirsToBranchesToIdentities holds all the module directories, branches, and module identity
	// targets for those directories+branches, prepared before syncing either from its identity
	// override or HEAD commit. (moduleDir:branch:targetModuleIdentity)
	modulesDirsToBranchesToIdentities map[string]map[string]bufmoduleref.ModuleIdentity
	// sortedBranchesForSync stores all git branches to sync, in their sort order
	sortedBranchesForSync []string
	// modulesToBranchesExpectedSyncPoints holds expected sync points for module identity and its
	// branches. (moduleIdentity:branch:lastSyncPointGitHash)
	modulesToBranchesExpectedSyncPoints map[string]map[string]string
	// modulesIdentitiesToCommitsSyncedCache caches commits already synced to a given BSR module, so
	// we don't ask twice the same module:commit when we already know it's already synced. We don't
	// cache "unsynced" git commits, because during the sync process we will be syncing new git
	// commits, which then will be added also to this cache. (moduleIdentity:commits)
	modulesIdentitiesToCommitsSyncedCache map[string]map[string]struct{}
}

func newSyncer(
	logger *zap.Logger,
	clock Clock,
	repo git.Repository,
	storageGitProvider storagegit.Provider,
	handler Handler,
	options ...SyncerOption,
) (Syncer, error) {
	s := &syncer{
		logger:                                logger,
		clock:                                 clock,
		repo:                                  repo,
		storageGitProvider:                    storageGitProvider,
		handler:                               handler,
		modulesDirsToIdentityOverrideForSync:  make(map[string]bufmoduleref.ModuleIdentity),
		commitsToTags:                         make(map[string][]string),
		modulesDirsToBranchesToIdentities:     make(map[string]map[string]bufmoduleref.ModuleIdentity),
		modulesToBranchesExpectedSyncPoints:   make(map[string]map[string]string),
		modulesIdentitiesToCommitsSyncedCache: make(map[string]map[string]struct{}),
	}
	for _, opt := range options {
		if err := opt(s); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *syncer) Sync(ctx context.Context) error {
	if err := s.prepareSync(ctx); err != nil {
		return fmt.Errorf("sync preparation: %w", err)
	}
	s.printSyncPreparation()
	if !s.hasSomethingForSync() {
		s.logger.Warn("branches and modules directories scanned, nothing to sync")
		return nil
	}
	// for each module directory in its original order
	for _, moduleDir := range s.sortedModulesDirsForSync {
		branchesToIdentities, shouldSyncModuleDir := s.modulesDirsToBranchesToIdentities[moduleDir]
		if !shouldSyncModuleDir {
			s.logger.Warn("module directory has no module identity target in any branch", zap.String("module directory", moduleDir))
			continue
		}
		// for each branch in the right sync order
		for _, branch := range s.sortedBranchesForSync {
			moduleIdentity, branchHasIdentity := branchesToIdentities[branch]
			if !branchHasIdentity || moduleIdentity == nil {
				s.logger.Warn(
					"module directory has no module identity target for branch",
					zap.String("module directory", moduleDir),
					zap.String("branch", branch),
				)
				continue
			}
			var expectedSyncPoint string
			if moduleLastSyncPoints, ok := s.modulesToBranchesExpectedSyncPoints[moduleIdentity.IdentityString()]; ok {
				expectedSyncPoint = moduleLastSyncPoints[branch]
			}
			if expectedSyncPoint == "" {
				s.logger.Debug(
					"module identity has no expected sync point for branch",
					zap.String("module identity", moduleIdentity.IdentityString()),
					zap.String("branch", branch),
				)
			}
			if err := s.syncModuleInBranch(ctx, moduleDir, moduleIdentity, branch, expectedSyncPoint); err != nil {
				return fmt.Errorf("sync module %s in branch %s: %w", moduleDir, branch, err)
			}
		}
	}
	return nil
}

func (s *syncer) prepareSync(ctx context.Context) error {
	// (1) Prepare all tags locations.
	if err := s.repo.ForEachTag(func(tag string, commitHash git.Hash) error {
		s.commitsToTags[commitHash.Hex()] = append(s.commitsToTags[commitHash.Hex()], tag)
		return nil
	}); err != nil {
		return fmt.Errorf("load tags: %w", err)
	}
	// (2) Prepare branches to be synced.
	allBranches := make(map[string]struct{})
	if err := s.repo.ForEachBranch(func(branch string, _ git.Hash) error {
		allBranches[branch] = struct{}{}
		return nil
	}, git.ForEachBranchWithRemote(s.gitRemoteName)); err != nil {
		return fmt.Errorf("looping over repository branches: %w", err)
	}
	remoteErrMsg := "on local branches"
	if s.gitRemoteName != "" {
		remoteErrMsg = fmt.Sprintf("on git remote %s", s.gitRemoteName)
	}
	// sync default git branch, make sure it's present
	gitDefaultBranch := s.repo.DefaultBranch()
	if _, gitDefaultBranchPresent := allBranches[gitDefaultBranch]; !gitDefaultBranchPresent {
		return fmt.Errorf("default branch %s is not present %s", gitDefaultBranch, remoteErrMsg)
	}
	s.logger.Debug("default git branch", zap.String("name", gitDefaultBranch))
	var branchesToSync []string
	if s.syncAllBranches {
		// sync all branches
		branchesToSync = slicesextended.MapToSlice(allBranches)
	} else {
		// sync current branch, make sure it's present
		currentBranch, err := s.repo.CheckedOutBranch()
		if err != nil {
			return fmt.Errorf("determine checked out branch")
		}
		if _, currentBranchPresent := allBranches[currentBranch]; !currentBranchPresent {
			return fmt.Errorf("current branch %s is not present %s", currentBranch, remoteErrMsg)
		}
		branchesToSync = append(branchesToSync, gitDefaultBranch, currentBranch)
		s.logger.Debug("current git branch", zap.String("name", currentBranch))
	}
	var sortedBranchesForSync []string
	for _, branch := range branchesToSync {
		if branch == gitDefaultBranch {
			continue // default branch will be injected manually
		}
		sortedBranchesForSync = append(sortedBranchesForSync, branch)
	}
	sort.Strings(sortedBranchesForSync)
	s.sortedBranchesForSync = append([]string{gitDefaultBranch}, sortedBranchesForSync...) // default first, then the rest A-Z
	for _, moduleDir := range s.sortedModulesDirsForSync {
		s.modulesDirsToBranchesToIdentities[moduleDir] = make(map[string]bufmoduleref.ModuleIdentity)
		for _, branch := range s.sortedBranchesForSync {
			s.modulesDirsToBranchesToIdentities[moduleDir][branch] = nil
		}
	}
	// (3) Prepare module targets for all module directories and branches.
	allModulesIdentitiesForSync := make(map[string]bufmoduleref.ModuleIdentity) // moduleIdentityString:moduleIdentity
	for _, branch := range branchesToSync {
		headCommit, err := s.repo.HEADCommit(
			git.HEADCommitWithBranch(branch),
			git.HEADCommitWithRemote(s.gitRemoteName),
		)
		if err != nil {
			return fmt.Errorf("reading head commit for branch %s: %w", branch, err)
		}
		for moduleDir, identityOverride := range s.modulesDirsToIdentityOverrideForSync {
			var targetModuleIdentity bufmoduleref.ModuleIdentity
			if identityOverride == nil {
				// no identity override, read from HEAD
				builtModule, readErr := s.readModuleAt(ctx, headCommit, moduleDir)
				if readErr != nil {
					// any error reading module in HEAD, skip syncing that module in that branch
					s.logger.Warn(
						"read module from HEAD failed, module won't be synced for this branch",
						zap.Error(readErr),
					)
					continue
				}
				if builtModule == nil {
					s.logger.Debug(
						"no module on HEAD, skipping branch",
						zap.String("branch", branch),
						zap.String("moduleDir", moduleDir),
					)
					// no module on branch
					continue
				}
				targetModuleIdentity = builtModule.ModuleIdentity()
			} else {
				// disregard module name in HEAD, use the identity override
				targetModuleIdentity = identityOverride
			}
			// enqueue this branch+module for sync to the right target
			s.modulesDirsToBranchesToIdentities[moduleDir][branch] = targetModuleIdentity
			targetModuleIdentityString := targetModuleIdentity.IdentityString()
			// do we have an expected git sync point for this module+branch?
			moduleBranchSyncPoint, err := s.resolveSyncPoint(ctx, targetModuleIdentity, branch)
			if err != nil {
				return fmt.Errorf("resolve expected sync point for module %s in branch %s: %w", targetModuleIdentityString, branch, err)
			}
			allModulesIdentitiesForSync[targetModuleIdentityString] = targetModuleIdentity
			if s.modulesToBranchesExpectedSyncPoints[targetModuleIdentityString] == nil {
				s.modulesToBranchesExpectedSyncPoints[targetModuleIdentityString] = make(map[string]string)
			}
			if moduleBranchSyncPoint != nil {
				s.modulesToBranchesExpectedSyncPoints[targetModuleIdentityString][branch] = moduleBranchSyncPoint.Hex()
			}
		}
	}
	// make sure no duplicate identities for different directories in the same branch
	var duplicatedIdentitiesErr error
	for _, branch := range s.sortedBranchesForSync {
		identitiesInBranch := make(map[string][]string) // moduleIdentity:[]moduleDir
		for _, moduleDir := range s.sortedModulesDirsForSync {
			branchToIdentity, ok := s.modulesDirsToBranchesToIdentities[moduleDir]
			if !ok {
				continue // this module directory won't be synced by any branch
			}
			identity, ok := branchToIdentity[branch]
			if !ok || identity == nil {
				continue // this module directory won't be synced by this branch
			}
			identitiesInBranch[identity.IdentityString()] = append(identitiesInBranch[identity.IdentityString()], moduleDir)
		}
		for moduleIdentity, moduleDirs := range identitiesInBranch {
			if len(moduleDirs) > 1 {
				duplicatedIdentitiesErr = multierr.Append(duplicatedIdentitiesErr, fmt.Errorf(
					"module identity %s cannot be synced in branch %s: present in multiple module directories: [%s]",
					moduleIdentity, branch, strings.Join(moduleDirs, ", "),
				))
			}
		}
	}
	if duplicatedIdentitiesErr != nil {
		return duplicatedIdentitiesErr
	}
	return nil
}

// resolveSyncPoint resolves a sync point for a particular module identity and branch.
func (s *syncer) resolveSyncPoint(ctx context.Context, moduleIdentity bufmoduleref.ModuleIdentity, branch string) (git.Hash, error) {
	syncPoint, err := s.handler.ResolveSyncPoint(ctx, moduleIdentity, branch)
	if err != nil {
		return nil, fmt.Errorf("resolve sync point for module %s: %w", moduleIdentity.IdentityString(), err)
	}
	if syncPoint == nil {
		// no sync point for that module in that branch
		return nil, nil
	}
	// Validate that the commit pointed to by the sync point exists in the git repo.
	if _, err := s.repo.Objects().Commit(syncPoint); err != nil {
		isDefaultBranch := branch == s.repo.DefaultBranch()
		return nil, s.handler.InvalidBSRSyncPoint(moduleIdentity, branch, syncPoint, isDefaultBranch, err)
	}
	return syncPoint, nil
}

// hasSomethingForSync returns true if there is at least one valid module identity for any module
// directory in any branch.
func (s *syncer) hasSomethingForSync() bool {
	for _, branchesToIdentities := range s.modulesDirsToBranchesToIdentities {
		for _, identity := range branchesToIdentities {
			if identity != nil {
				return true
			}
		}
	}
	return false
}

// syncModuleInBranch syncs a module directory in a branch.
func (s *syncer) syncModuleInBranch(
	ctx context.Context,
	moduleDir string,
	moduleIdentity bufmoduleref.ModuleIdentity,
	branch string,
	expectedSyncPoint string,
) error {
	commitsForSync, err := s.branchSyncableCommits(ctx, moduleDir, moduleIdentity, branch, expectedSyncPoint)
	if err != nil {
		return fmt.Errorf("finding commits to sync: %w", err)
	}
	// first sync tags in old commits
	var startSyncPoint git.Hash
	if len(commitsForSync) == 0 {
		// no commits to sync for this branch, backfill from HEAD
		headCommit, err := s.repo.HEADCommit(
			git.HEADCommitWithBranch(branch),
			git.HEADCommitWithRemote(s.gitRemoteName),
		)
		if err != nil {
			return fmt.Errorf("read HEAD commit for branch %s: %w", branch, err)
		}
		startSyncPoint = headCommit.Hash()
	} else {
		// backfill from the first commit to sync
		startSyncPoint = commitsForSync[0].commit.Hash()
	}
	if err := s.backfillTags(ctx, moduleDir, moduleIdentity, branch, startSyncPoint); err != nil {
		return fmt.Errorf("sync looking back for branch %s: %w", branch, err)
	}
	// now sync
	targetModuleIdentity := moduleIdentity.IdentityString() // all syncable modules in the branch have the same target
	logger := s.logger.With(
		zap.String("module directory", branch),
		zap.String("module identity", targetModuleIdentity),
		zap.String("branch", branch),
	)
	// then sync the new commits
	if len(commitsForSync) == 0 {
		logger.Debug("no commits to sync for module in branch")
		return nil
	}
	s.logger.Debug("branch syncable commits for module", zap.Strings("git commits", syncableCommitsHashes(commitsForSync)))
	for _, commitForSync := range commitsForSync {
		commitHash := commitForSync.commit.Hash().Hex()
		builtModule := commitForSync.module
		if builtModule == nil {
			return fmt.Errorf("syncable commit %s has no built module to sync", commitHash)
		}
		if err := s.handler.SyncModuleCommit(
			ctx,
			newModuleCommit(
				branch,
				commitForSync.commit,
				s.commitsToTags[commitHash],
				moduleDir,
				moduleIdentity, // all syncable modules in the branch have the same target
				builtModule.Bucket,
			),
		); err != nil {
			return fmt.Errorf("sync module %s:%s in commit %s: %w", moduleDir, targetModuleIdentity, commitHash, err)
		}
		// module was synced successfully, add it to the cache
		if s.modulesIdentitiesToCommitsSyncedCache[targetModuleIdentity] == nil {
			s.modulesIdentitiesToCommitsSyncedCache[targetModuleIdentity] = make(map[string]struct{})
		}
		s.modulesIdentitiesToCommitsSyncedCache[targetModuleIdentity][commitHash] = struct{}{}
	}
	return nil
}

// branchSyncableCommits returns a sorted array of commit+module that are pending to sync for a
// moduleDir+branch. Every syncable commit contains a valid git commit and a built named module.
func (s *syncer) branchSyncableCommits(
	ctx context.Context,
	moduleDir string,
	moduleIdentity bufmoduleref.ModuleIdentity,
	branch string,
	expectedSyncPoint string,
) ([]*syncableCommit, error) {
	targetModuleIdentity := moduleIdentity.IdentityString()
	logger := s.logger.With(
		zap.String("module directory", moduleDir),
		zap.String("target module identity", targetModuleIdentity),
		zap.String("branch", branch),
		zap.String("expected sync point", expectedSyncPoint),
	)
	var commitsForSync []*syncableCommit
	eachCommitFunc := func(commit git.Commit) error {
		commitHash := commit.Hash().Hex()
		logger := logger.With(zap.String("commit", commitHash))
		// check if this commit is already synced
		isSynced, err := s.isGitCommitSynced(ctx, moduleIdentity, commit.Hash())
		if err != nil {
			return fmt.Errorf(
				"checking if module %s already synced git commit %s: %w",
				targetModuleIdentity, commitHash, err,
			)
		}
		if isSynced {
			if expectedSyncPoint == "" {
				// we did not expect a sync point for this branch, it's ok to stop
				logger.Debug("git commit already synced, stop looking back in branch")
			} else if commitHash != expectedSyncPoint {
				// we expected a different sync point for this branch, it's ok to stop as long as it's not a
				// protected branch
				isProtectedBranch, err := s.handler.IsProtectedBranch(ctx, moduleIdentity, branch)
				if err != nil {
					return fmt.Errorf("check if branch %q is protected for module %q: %w", branch, moduleIdentity, err)
				}
				if isProtectedBranch {
					return fmt.Errorf(
						"branch protection: "+
							"found synced git commit %s for branch %s, but expected sync point was %s, "+
							"did you rebase or reset this branch?",
						commitHash, branch, expectedSyncPoint,
					)
				}
				logger.Warn("unexpected sync point reached, stop looking back in branch")
			} else {
				// we reached the expected sync point for this branch, it's ok to stop
				logger.Debug("expected sync point reached, stop looking back in branch")
			}
			return git.ErrStopForEach
		}
		// git commit is not synced, attempt to read the module in the commit:moduleDir
		builtModule, err := s.readModuleAt(ctx, commit, moduleDir)
		if err != nil {
			if errors.Is(err, errReadModuleInvalidModule) || errors.Is(err, errReadModuleInvalidModuleConfig) {
				logger.Debug("read module at commit failed, skipping commit", zap.Error(err))
				return nil
			}
			return err
		}
		if builtModule == nil {
			logger.Debug("module not found, skipping commit")
			return nil
		}
		if builtModule.ModuleIdentity() == nil {
			if _, hasOverride := s.modulesDirsToIdentityOverrideForSync[moduleDir]; !hasOverride {
				logger.Debug("unnamed module, no override, skipping commit")
				return nil
			}
		}
		commitsForSync = append(commitsForSync, &syncableCommit{
			commit: commit,
			module: builtModule,
		})
		return nil
	}
	if err := s.repo.ForEachCommit(
		eachCommitFunc,
		git.ForEachCommitWithBranchStartPoint(
			branch,
			git.ForEachCommitWithBranchStartPointWithRemote(s.gitRemoteName),
		),
	); err != nil {
		return nil, err
	}
	// if we have no commits to sync we can bail early
	if len(commitsForSync) == 0 {
		return nil, nil
	}
	// reverse commits to sync, to leave them in time order parent -> children
	// https://github.com/golang/go/wiki/SliceTricks#reversing
	for i := len(commitsForSync)/2 - 1; i >= 0; i-- {
		opp := len(commitsForSync) - 1 - i
		commitsForSync[i], commitsForSync[opp] = commitsForSync[opp], commitsForSync[i]
	}
	return commitsForSync, nil
}

// isGitCommitSynced checks if a commit hash is already synced to a BSR module.
func (s *syncer) isGitCommitSynced(ctx context.Context, moduleIdentity bufmoduleref.ModuleIdentity, commitHash git.Hash) (bool, error) {
	modIdentity := moduleIdentity.IdentityString()
	// check local cache first
	if syncedModuleCommits, ok := s.modulesIdentitiesToCommitsSyncedCache[modIdentity]; ok {
		if _, commitSynced := syncedModuleCommits[commitHash.Hex()]; commitSynced {
			return true, nil
		}
	}
	// not in the cache, request BSR check
	commitSynced, err := s.handler.IsGitCommitSynced(ctx, moduleIdentity, commitHash)
	if err != nil {
		return false, err
	}
	if commitSynced {
		// populate local cache
		if s.modulesIdentitiesToCommitsSyncedCache[modIdentity] == nil {
			s.modulesIdentitiesToCommitsSyncedCache[modIdentity] = make(map[string]struct{})
		}
		s.modulesIdentitiesToCommitsSyncedCache[modIdentity][commitHash.Hex()] = struct{}{}
	}
	return commitSynced, nil
}

// readModule returns a module that has a name and builds correctly given a commit and a module
// directory. If the module builds, it might be returned alongside a non-nil error.
func (s *syncer) readModuleAt(
	ctx context.Context,
	commit git.Commit,
	moduleDir string,
) (*bufmodulebuild.BuiltModule, error) {
	commitBucket, err := s.storageGitProvider.NewReadBucket(commit.Tree(), storagegit.ReadBucketWithSymlinksIfSupported())
	if err != nil {
		return nil, fmt.Errorf("new read bucket: %w", err)
	}
	moduleBucket := storage.MapReadBucket(commitBucket, storage.MapOnPrefix(moduleDir))
	foundModule, err := bufconfig.ExistingConfigFilePath(ctx, moduleBucket)
	if err != nil {
		return nil, fmt.Errorf("looking for an existing config file path: %w", err)
	}
	if foundModule == "" {
		// No module at this commit.
		return nil, nil
	}
	sourceConfig, err := bufconfig.GetConfigForBucket(ctx, moduleBucket)
	if err != nil {
		// Invalid config.
		return nil, fmt.Errorf("%w: %s", errReadModuleInvalidModuleConfig, err)
	}
	builtModule, err := bufmodulebuild.NewModuleBucketBuilder().BuildForBucket(
		ctx,
		moduleBucket,
		sourceConfig.Build,
		bufmodulebuild.WithModuleIdentity(sourceConfig.ModuleIdentity),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errReadModuleInvalidModule, err)
	}
	return builtModule, nil
}

// backfillTags takes syncable commits for a branch already calculated, and looks back for each
// module a given amount of commits or timestamps, syncing tags in case they were created or moved
// after those commits were synced.
func (s *syncer) backfillTags(
	ctx context.Context,
	moduleDir string,
	moduleIdentity bufmoduleref.ModuleIdentity,
	branch string,
	syncStartHash git.Hash,
) error {
	var (
		lookbackCommitsCount int
		timeLimit            = s.clock.Now().Add(-LookbackTimeLimit)
		logger               = s.logger.With(
			zap.String("branch", branch),
			zap.String("module directory", moduleDir),
			zap.String("module identity", moduleIdentity.IdentityString()),
			zap.String("start point", syncStartHash.Hex()),
		)
	)
	forEachOldCommitFunc := func(oldCommit git.Commit) error {
		lookbackCommitsCount++
		// For the lookback into older commits to stop, both lookback limits (amount of commits and
		// timespan) need to be met.
		if lookbackCommitsCount > LookbackCommitsLimit &&
			oldCommit.Committer().Timestamp().Before(timeLimit) {
			return git.ErrStopForEach
		}
		// Is there any tag in this commit to backfill?
		tagsToBackfill := s.commitsToTags[oldCommit.Hash().Hex()]
		if len(tagsToBackfill) == 0 {
			return nil
		}
		// For each older commit we travel, we need to make sure it's a valid module with a module
		// identity or an overridden module identity.
		if builtModule, err := s.readModuleAt(ctx, oldCommit, moduleDir); err != nil {
			if errors.Is(err, errReadModuleInvalidModule) || errors.Is(err, errReadModuleInvalidModuleConfig) {
				// read module failed, skip commit
				return nil
			}
			return fmt.Errorf("read module at commit %q in dir %q: %w", oldCommit, moduleDir, err)
		} else if builtModule == nil {
			// no module, skip commit
			return nil
		} else if builtModule.ModuleIdentity() == nil {
			if _, hasOverride := s.modulesDirsToIdentityOverrideForSync[moduleDir]; !hasOverride {
				// no override for this module
				return nil
			}
		}
		logger := logger.With(
			zap.String("commit", oldCommit.Hash().Hex()),
			zap.Strings("tags", tagsToBackfill),
		)
		// Valid module in this commit to backfill tags. If backfilling the tags fails, we'll
		// WARN+continue to not block actual pending commits to sync in this run.
		bsrCommitName, err := s.handler.BackfillTags(ctx, moduleIdentity, oldCommit.Hash(), oldCommit.Author(), oldCommit.Committer(), tagsToBackfill)
		if err != nil {
			logger.Warn("backfill older tags failed", zap.Error(err))
			return nil
		}
		logger.Debug("older tags backfilled", zap.String("BSR commit", bsrCommitName))
		return nil
	}
	if err := s.repo.ForEachCommit(
		forEachOldCommitFunc,
		git.ForEachCommitWithHashStartPoint(syncStartHash.Hex()),
	); err != nil {
		return fmt.Errorf("looking back past the start sync point: %w", err)
	}
	return nil
}

// printSyncPreparation prints information gathered at the sync preparation step.
func (s *syncer) printSyncPreparation() {
	s.logger.Debug(
		"sync preparation",
		zap.Any("modulesDirsToSync", s.modulesDirsToIdentityOverrideForSync),
		zap.Any("commitsTags", s.commitsToTags),
		zap.Any("branchesModulesToSync", s.modulesDirsToBranchesToIdentities),
		zap.Any("modulesBranchesSyncPoints", s.modulesToBranchesExpectedSyncPoints),
	)
}

// syncableCommit holds the built module that need to be synced in a git commit.
type syncableCommit struct {
	commit git.Commit
	module *bufmodulebuild.BuiltModule
}

func syncableCommitsHashes(syncableCommits []*syncableCommit) []string {
	var hashes []string
	for _, sCommit := range syncableCommits {
		hashes = append(hashes, sCommit.commit.Hash().Hex())
	}
	return hashes
}
