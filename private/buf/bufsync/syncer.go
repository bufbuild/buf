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
	"time"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagegit"
	"go.uber.org/zap"
)

const (
	// lookbackCommitsLimit is the amount of commits that we will look back before the start sync
	// point to backfill old git tags. We might allow customizing this value in the future.
	lookbackCommitsLimit = 5
	// lookbackTimeLimit is how old we will look back (git commit timestamps) before the start sync
	// point to backfill old git tags. We might allow customizing this value in the future.
	lookbackTimeLimit = 24 * time.Hour
)

type syncer struct {
	logger                    *zap.Logger
	repo                      git.Repository
	storageGitProvider        storagegit.Provider
	errorHandler              ErrorHandler
	syncedGitCommitChecker    SyncedGitCommitChecker
	moduleDefaultBranchGetter ModuleDefaultBranchGetter
	syncPointResolver         SyncPointResolver
	tagsBackfiller            TagsBackfiller

	// flags received on creation
	sortedModulesDirsForSync             []string
	modulesDirsToIdentityOverrideForSync map[string]bufmoduleref.ModuleIdentity // moduleDir:moduleIdentityOverride
	syncAllBranches                      bool

	// commitsToTags holds all tags in the repo associated to its commit hash. (commits:[]tags)
	commitsToTags map[string][]string
	// modulesDirsToBranchesToIdentities holds all the module directories, branches, and module identity
	// targets for those directories+branches, prepared before syncing either from its identity
	// override or HEAD commit. (moduleDir:branch:targetModuleIdentity)
	modulesDirsToBranchesToIdentities map[string]map[string]bufmoduleref.ModuleIdentity
	// sortedBranchesForSync stores all git branches to sync, in their sort order
	sortedBranchesForSync []string
	// modulesToBranchesExpectedSyncPoints holds remote expected sync points for module identity and its
	// branches. (moduleIdentity:branch:lastSyncPointGitHash)
	modulesToBranchesExpectedSyncPoints map[string]map[string]string

	// modulesIdentitiesToCommitsSyncedCache (moduleIdentity:commit) caches commits already synced to
	// a given BSR module, so we don't ask twice the same module:commit when we already know it's
	// already synced. We don't cache "unsynced" git commits, because during the sync process we will
	// be syncing new git commits, which then will be added also to this cache.
	modulesIdentitiesToCommitsSyncedCache map[string]map[string]struct{}
}

func newSyncer(
	logger *zap.Logger,
	repo git.Repository,
	storageGitProvider storagegit.Provider,
	errorHandler ErrorHandler,
	options ...SyncerOption,
) (Syncer, error) {
	s := &syncer{
		logger:                                logger,
		repo:                                  repo,
		storageGitProvider:                    storageGitProvider,
		errorHandler:                          errorHandler,
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
	if s.moduleDefaultBranchGetter == nil {
		s.logger.Warn(
			"no module default branch getter, the default branch validation will be skipped for all modules and branches",
			zap.String("default_git_branch", s.repo.DefaultBranch()),
		)
	}
	if s.syncedGitCommitChecker == nil {
		s.logger.Warn("no sync git commit checker, all branches will attempt to sync from the start")
	}
	return s, nil
}

func (s *syncer) Sync(ctx context.Context, syncFunc SyncFunc) error {
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
			s.logger.Warn("module directory has no branches to sync", zap.String("module directory", moduleDir))
			continue
		}
		// for each branch in the right sync order
		for _, branch := range s.sortedBranchesForSync {
			moduleIdentity, branchHasIdentity := branchesToIdentities[branch]
			if !branchHasIdentity || moduleIdentity == nil {
				s.logger.Warn(
					"module directory has no module identity in branch",
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
			if err := s.syncModuleInBranch(ctx, moduleDir, moduleIdentity, branch, expectedSyncPoint, syncFunc); err != nil {
				return fmt.Errorf("sync module %s in branch %s: %w", moduleDir, branch, err)
			}
		}
	}
	return nil
}

func (s *syncer) prepareSync(ctx context.Context) error {
	// Populate all tags locations.
	if err := s.repo.ForEachTag(func(tag string, commitHash git.Hash) error {
		s.commitsToTags[commitHash.Hex()] = append(s.commitsToTags[commitHash.Hex()], tag)
		return nil
	}); err != nil {
		return fmt.Errorf("load tags: %w", err)
	}
	// Populate all branches to be synced.
	allRemoteBranches := make(map[string]struct{})
	if err := s.repo.ForEachBranch(func(branch string, _ git.Hash) error {
		allRemoteBranches[branch] = struct{}{}
		return nil
	}); err != nil {
		return fmt.Errorf("looping over repo remote branches: %w", err)
	}
	// sync default git branch, make sure it's present in the remote
	defaultBranch := s.repo.DefaultBranch()
	if _, isDefaultBranchPushedInRemote := allRemoteBranches[defaultBranch]; !isDefaultBranchPushedInRemote {
		return fmt.Errorf("default branch %s is not present in 'origin' remote", defaultBranch)
	}
	s.logger.Debug("default git branch", zap.String("name", defaultBranch))
	var branchesToSync []string
	if s.syncAllBranches {
		// sync all remote branches
		for remoteBranch := range allRemoteBranches {
			branchesToSync = append(branchesToSync, remoteBranch)
		}
	} else {
		// sync current branch, make sure it's present in the remote
		currentBranch := s.repo.CurrentBranch()
		if _, isCurrentBranchPushedInRemote := allRemoteBranches[currentBranch]; !isCurrentBranchPushedInRemote {
			return fmt.Errorf("current branch %s is not present in 'origin' remote", currentBranch)
		}
		branchesToSync = append(branchesToSync, defaultBranch, currentBranch)
		s.logger.Debug("current branch", zap.String("name", currentBranch))
	}
	var sortedBranchesForSync []string
	for _, branch := range branchesToSync {
		if branch == defaultBranch {
			continue // default branch will be injected manually
		}
		sortedBranchesForSync = append(sortedBranchesForSync, branch)
	}
	sort.Strings(sortedBranchesForSync)
	s.sortedBranchesForSync = append([]string{defaultBranch}, sortedBranchesForSync...)
	for _, moduleDir := range s.sortedModulesDirsForSync {
		s.modulesDirsToBranchesToIdentities[moduleDir] = make(map[string]bufmoduleref.ModuleIdentity)
		for _, branch := range s.sortedBranchesForSync {
			s.modulesDirsToBranchesToIdentities[moduleDir][branch] = nil
		}
	}
	// Populate module identities, from identity overrides or from HEAD, and its sync points if any
	allModulesIdentitiesForSync := make(map[string]bufmoduleref.ModuleIdentity) // moduleIdentityString:moduleIdentity
	for _, branch := range branchesToSync {
		headCommit, err := s.repo.HEADCommit(branch)
		if err != nil {
			return fmt.Errorf("reading head commit for branch %s: %w", branch, err)
		}
		for moduleDir, identityOverride := range s.modulesDirsToIdentityOverrideForSync {
			var targetModuleIdentity bufmoduleref.ModuleIdentity
			if identityOverride == nil {
				// no identity override, read from HEAD
				builtModule, readErr := s.readModuleAt(ctx, branch, headCommit, moduleDir)
				if readErr != nil {
					// any error reading module in HEAD, skip syncing that module in that branch
					s.logger.Warn(
						"read module from HEAD failed, module won't be synced for this branch",
						zap.Error(readErr),
					)
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
			// do we have a remote git sync point for this module+branch?
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
	// make sure all module identities we are about to sync in all branches have the same BSR default
	// branch as the local git default branch.
	for _, moduleIdentity := range allModulesIdentitiesForSync {
		if err := s.validateDefaultBranch(ctx, moduleIdentity); err != nil {
			return fmt.Errorf("validate default branch for module %s: %w", moduleIdentity.IdentityString(), err)
		}
	}
	return nil
}

// resolveSyncPoint resolves a sync point for a particular module identity and branch.
func (s *syncer) resolveSyncPoint(ctx context.Context, moduleIdentity bufmoduleref.ModuleIdentity, branch string) (git.Hash, error) {
	// If resumption is not enabled, we can bail early.
	if s.syncPointResolver == nil {
		return nil, nil
	}
	syncPoint, err := s.syncPointResolver(ctx, moduleIdentity, branch)
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
		return nil, s.errorHandler.InvalidRemoteSyncPoint(moduleIdentity, branch, syncPoint, isDefaultBranch, err)
	}
	return syncPoint, nil
}

// validateDefaultBranch checks that the passed module identity has the same default branch as the
// syncer Git repo's default branch.
func (s *syncer) validateDefaultBranch(ctx context.Context, moduleIdentity bufmoduleref.ModuleIdentity) error {
	if s.moduleDefaultBranchGetter == nil {
		return nil
	}
	expectedDefaultGitBranch := s.repo.DefaultBranch()
	bsrDefaultBranch, err := s.moduleDefaultBranchGetter(ctx, moduleIdentity)
	if err != nil {
		if errors.Is(err, ErrModuleDoesNotExist) {
			s.logger.Warn(
				"default branch validation skipped",
				zap.String("default_git_branch", expectedDefaultGitBranch),
				zap.String("module", moduleIdentity.IdentityString()),
				zap.Error(err),
			)
			return nil
		}
		return fmt.Errorf("getting bsr module: %w", err)
	}
	if bsrDefaultBranch != expectedDefaultGitBranch {
		return fmt.Errorf(
			"remote module default branch %s does not match the git repository's default branch %s, aborting sync",
			bsrDefaultBranch, expectedDefaultGitBranch,
		)
	}
	return nil
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
	syncFunc SyncFunc,
) error {
	commitsForSync, err := s.branchSyncableCommits(ctx, moduleDir, moduleIdentity, branch, expectedSyncPoint)
	if err != nil {
		return fmt.Errorf("finding commits to sync: %w", err)
	}
	// first sync tags in old commits
	if s.tagsBackfiller != nil {
		var startSyncPoint git.Hash
		if len(commitsForSync) == 0 {
			// no commits to sync for this branch, backfill from HEAD
			headCommit, err := s.repo.HEADCommit(branch)
			if err != nil {
				return fmt.Errorf("read HEAD commit for branch %s: %w", branch, err)
			}
			startSyncPoint = headCommit.Hash()
		} else {
			// backfill from the first commit to sync
			startSyncPoint = commitsForSync[0].commit.Hash()
		}
		if err := s.backfillTags(ctx, moduleDir, moduleIdentity, branch, startSyncPoint, &realClock{}); err != nil {
			return fmt.Errorf("sync looking back for branch %s: %w", branch, err)
		}
	}
	logger := s.logger.With(
		zap.String("module directory", branch),
		zap.String("module identity", moduleIdentity.IdentityString()),
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
		if builtModule.ModuleIdentity() == nil {
			return fmt.Errorf("built module for commit %s has no module identity", commitHash)
		}
		modIdentity := builtModule.ModuleIdentity().IdentityString()
		if err := syncFunc(
			ctx,
			newModuleCommit(
				branch,
				commitForSync.commit,
				s.commitsToTags[commitHash],
				moduleDir,
				builtModule.ModuleIdentity(),
				builtModule.Bucket,
			),
		); err != nil {
			return fmt.Errorf("sync module %s:%s in commit %s: %w", moduleDir, modIdentity, commitHash, err)
		}
		// module was synced successfully, add it to the cache
		if s.modulesIdentitiesToCommitsSyncedCache[modIdentity] == nil {
			s.modulesIdentitiesToCommitsSyncedCache[modIdentity] = make(map[string]struct{})
		}
		s.modulesIdentitiesToCommitsSyncedCache[modIdentity][commitHash] = struct{}{}
	}
	return nil
}

// branchSyncableCommits returns a sorted array of commit+module that are pending to sync for a
// moduleDir+branch. A commit in the array might have no modules to sync if those are skipped by the
// Syncer error handler, or are a found sync point.
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
	stopLoopErr := errors.New("stop loop")
	eachCommitFunc := func(commit git.Commit) error {
		commitHash := commit.Hash().Hex()
		logger := logger.With(zap.String("commit", commitHash))
		// check if this commit is already synced
		isSynced, err := s.isGitCommitSynced(ctx, moduleIdentity, commitHash)
		if err != nil {
			return fmt.Errorf(
				"checking if module %s already synced git commit %s: %w",
				targetModuleIdentity, commitHash, err,
			)
		}
		if isSynced {
			if expectedSyncPoint == "" {
				// no expected sync point
				logger.Debug("git commit already synced, stop looking back in branch")
			} else if commitHash != expectedSyncPoint {
				// unexpected sync point
				if s.repo.DefaultBranch() == branch {
					// default git branch
					return fmt.Errorf(
						"found synced git commit %s for default branch %s, but expected sync point was %s, "+
							"did you rebase or reset your default branch?",
						commitHash, branch, expectedSyncPoint,
					)
				}
				// any other branch
				logger.Warn("unexpected sync point reached, stop looking back in branch")
			} else {
				// reached expected sync point
				logger.Debug("expected sync point reached, stop looking back in branch")
			}
			return stopLoopErr
		}
		// git commit is not synced, attempt to read the module in the commit:moduleDir
		builtModule, readErr := s.readModuleAt(
			ctx, branch, commit, moduleDir,
			readModuleAtWithExpectedModuleIdentity(targetModuleIdentity),
		)
		if readErr != nil {
			decision := s.errorHandler.HandleReadModuleError(readErr)
			switch decision {
			case LookbackDecisionCodeFail:
				return fmt.Errorf("read module error: %w", readErr)
			case LookbackDecisionCodeSkip:
				logger.Debug("read module at commit failed, skipping commit", zap.Error(readErr))
			case LookbackDecisionCodeStop:
				logger.Debug("read module at commit failed, stop looking back in branch", zap.Error(readErr))
				return stopLoopErr
			case LookbackDecisionCodeOverride:
				logger.Debug("read module at commit failed, overriding module identity in commit", zap.Error(readErr))
				if builtModule == nil {
					return fmt.Errorf("cannot override commit, no built module: %w", readErr)
				}
				// rename the module to the target identity, and add it to the queue
				renamedModule, err := renameModule(ctx, builtModule, moduleIdentity)
				if err != nil {
					return fmt.Errorf("override module in commit: %s, rename module: %w", readErr.Error(), err)
				}
				commitsForSync = append(commitsForSync, &syncableCommit{commit: commit, module: renamedModule})
			default:
				return fmt.Errorf("unexpected decision code %d for read module error %w", decision, readErr)
			}
			return nil
		}
		commitsForSync = append(commitsForSync, &syncableCommit{commit: commit, module: builtModule})
		return nil
	}
	if err := s.repo.ForEachCommit(eachCommitFunc, git.ForEachCommitWithBranchStartPoint(branch)); err != nil && !errors.Is(err, stopLoopErr) {
		return nil, err
	}
	// if we have no commits to sync, no need to make more checks, bail early
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

// isGitCommitSynced checks if a commit hash is already synced to a remote BSR module.
func (s *syncer) isGitCommitSynced(ctx context.Context, moduleIdentity bufmoduleref.ModuleIdentity, commitHash string) (bool, error) {
	if s.syncedGitCommitChecker == nil {
		return false, nil
	}
	modIdentity := moduleIdentity.IdentityString()
	// check local cache first
	if syncedModuleCommits, ok := s.modulesIdentitiesToCommitsSyncedCache[modIdentity]; ok {
		if _, commitSynced := syncedModuleCommits[commitHash]; commitSynced {
			return true, nil
		}
	}
	// not in the cache, request remote check
	syncedModuleCommits, err := s.syncedGitCommitChecker(ctx, moduleIdentity, map[string]struct{}{commitHash: {}})
	if err != nil {
		return false, err
	}
	_, commitSynced := syncedModuleCommits[commitHash]
	if commitSynced {
		// populate local cache
		if s.modulesIdentitiesToCommitsSyncedCache[modIdentity] == nil {
			s.modulesIdentitiesToCommitsSyncedCache[modIdentity] = make(map[string]struct{})
		}
		s.modulesIdentitiesToCommitsSyncedCache[modIdentity][commitHash] = struct{}{}
	}
	return commitSynced, nil
}

// readModuleAt returns a module that has a name and builds correctly given a commit and a module
// directory, or a read module error. If the module builds, it might be returned alongside a not-nil
// error.
func (s *syncer) readModuleAt(
	ctx context.Context,
	branch string,
	commit git.Commit,
	moduleDir string,
	opts ...readModuleAtOption,
) (*bufmodulebuild.BuiltModule, *ReadModuleError) {
	var readOpts readModuleOpts
	for _, opt := range opts {
		opt(&readOpts)
	}
	// in case there is an error reading this module, it will have the same branch, commit, and module
	// dir that we can fill upfront. The actual `err` and `code` (if any) is populated in case-by-case
	// basis before returning.
	readModuleErr := &ReadModuleError{
		branch:    branch,
		commit:    commit.Hash().Hex(),
		moduleDir: moduleDir,
	}
	commitBucket, err := s.storageGitProvider.NewReadBucket(commit.Tree(), storagegit.ReadBucketWithSymlinksIfSupported())
	if err != nil {
		readModuleErr.err = fmt.Errorf("new read bucket: %w", err)
		return nil, readModuleErr
	}
	moduleBucket := storage.MapReadBucket(commitBucket, storage.MapOnPrefix(moduleDir))
	foundModule, err := bufconfig.ExistingConfigFilePath(ctx, moduleBucket)
	if err != nil {
		readModuleErr.err = fmt.Errorf("looking for an existing config file path: %w", err)
		return nil, readModuleErr
	}
	if foundModule == "" {
		readModuleErr.code = ReadModuleErrorCodeModuleNotFound
		readModuleErr.err = errors.New("module not found")
		return nil, readModuleErr
	}
	sourceConfig, err := bufconfig.GetConfigForBucket(ctx, moduleBucket)
	if err != nil {
		readModuleErr.code = ReadModuleErrorCodeInvalidModuleConfig
		readModuleErr.err = fmt.Errorf("invalid module config: %w", err)
		return nil, readModuleErr
	}
	builtModule, err := bufmodulebuild.NewModuleBucketBuilder().BuildForBucket(
		ctx,
		moduleBucket,
		sourceConfig.Build,
		bufmodulebuild.WithModuleIdentity(sourceConfig.ModuleIdentity),
	)
	if err != nil {
		readModuleErr.code = ReadModuleErrorCodeBuildModule
		readModuleErr.err = fmt.Errorf("build module: %w", err)
		return nil, readModuleErr
	}
	// module builds, unnamed and unexpectedName errors can be returned alongside the built module.
	if sourceConfig.ModuleIdentity == nil {
		readModuleErr.code = ReadModuleErrorCodeUnnamedModule
		readModuleErr.err = errors.New("found module does not have a name")
		return builtModule, readModuleErr
	}
	if readOpts.expectedModuleIdentity != "" {
		if sourceConfig.ModuleIdentity.IdentityString() != readOpts.expectedModuleIdentity {
			readModuleErr.code = ReadModuleErrorCodeUnexpectedName
			readModuleErr.err = fmt.Errorf(
				"read module has an unexpected module identity %s, expected %s",
				sourceConfig.ModuleIdentity.IdentityString(), readOpts.expectedModuleIdentity,
			)
			return builtModule, readModuleErr
		}
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
	clock clock,
) error {
	timeLimit := clock.Now().Add(-lookbackTimeLimit)
	stopLoopErr := errors.New("stop loop")
	var lookbackCommitsCount int
	forEachOldCommitFunc := func(oldCommit git.Commit) error {
		lookbackCommitsCount++
		// For the lookback into older commits to stop, both lookback limits (amount of commits and
		// timespan) need to be met.
		if lookbackCommitsCount > lookbackCommitsLimit &&
			oldCommit.Committer().Timestamp().Before(timeLimit) {
			return stopLoopErr
		}
		// Is there any tag in this commit to backfill?
		tagsToBackfill := s.commitsToTags[oldCommit.Hash().Hex()]
		if len(tagsToBackfill) == 0 {
			return nil
		}
		// For each older commit we travel, we need to make sure it's a valid module with the expected
		// module identity, or that the error handler would have chosen to override it.
		var shouldBackfillTagsForThisCommit bool
		if _, readErr := s.readModuleAt(
			ctx, branch, oldCommit, moduleDir,
			readModuleAtWithExpectedModuleIdentity(moduleIdentity.IdentityString()),
		); readErr == nil || s.errorHandler.HandleReadModuleError(readErr) == LookbackDecisionCodeOverride {
			shouldBackfillTagsForThisCommit = true
		}
		if !shouldBackfillTagsForThisCommit {
			// not a valid module, tags in this commit should not be backfilled to this module.
			return nil
		}
		logger := s.logger.With(
			zap.String("branch", branch),
			zap.String("commit", oldCommit.Hash().Hex()),
			zap.String("module directory", moduleDir),
			zap.String("module identity", moduleIdentity.IdentityString()),
			zap.String("module directory git start point", syncStartHash.Hex()),
			zap.Strings("tags", tagsToBackfill),
		)
		// Valid module in this commit to backfill tags. If backfilling the tags fails, we'll
		// WARN+continue to not block actual pending commits to sync in this run.
		bsrCommitName, err := s.tagsBackfiller(ctx, moduleIdentity, oldCommit.Hash(), oldCommit.Author(), oldCommit.Committer(), tagsToBackfill)
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
	); err != nil && !errors.Is(err, stopLoopErr) {
		return fmt.Errorf("looking back the start sync point: %w", err)
	}
	return nil
}

type readModuleOpts struct {
	expectedModuleIdentity string
}

type readModuleAtOption func(*readModuleOpts)

func readModuleAtWithExpectedModuleIdentity(moduleIdentity string) readModuleAtOption {
	return func(opts *readModuleOpts) { opts.expectedModuleIdentity = moduleIdentity }
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

// clock allows embedding a custom time.Now implementation, so it's easier to test.
type clock interface {
	Now() time.Time
}

// realClock returns the real time.Now.
type realClock struct{}

func (*realClock) Now() time.Time { return time.Now() }

// renameModule takes a module, and rebuilds it with a new module identity.
func renameModule(
	ctx context.Context,
	baseModule *bufmodulebuild.BuiltModule,
	newIdentity bufmoduleref.ModuleIdentity,
) (*bufmodulebuild.BuiltModule, error) {
	if baseModule == nil {
		return nil, errors.New("no base module to rebuild")
	}
	if newIdentity == nil {
		return nil, errors.New("no new identity to apply")
	}
	if baseModule.ModuleIdentity() != nil &&
		baseModule.ModuleIdentity().IdentityString() == newIdentity.IdentityString() {
		// same identity, no need to rename anything
		return baseModule, nil
	}
	sourceConfig, err := bufconfig.GetConfigForBucket(ctx, baseModule.Bucket)
	if err != nil {
		return nil, fmt.Errorf("invalid module config: %w", err)
	}
	renamedModule, err := bufmodulebuild.NewModuleBucketBuilder().BuildForBucket(
		ctx,
		baseModule.Bucket,
		sourceConfig.Build,
		bufmodulebuild.WithModuleIdentity(newIdentity),
	)
	if err != nil {
		return nil, fmt.Errorf("rebuild module with new identity: %w", err)
	}
	return renamedModule, nil
}

func syncableCommitsHashes(syncableCommits []*syncableCommit) []string {
	var hashes []string
	for _, sCommit := range syncableCommits {
		hashes = append(hashes, sCommit.commit.Hash().Hex())
	}
	return hashes
}
