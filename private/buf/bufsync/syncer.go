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

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagegit"
	"go.uber.org/zap"
)

type syncer struct {
	logger                    *zap.Logger
	repo                      git.Repository
	storageGitProvider        storagegit.Provider
	errorHandler              ErrorHandler
	syncedGitCommitChecker    SyncedGitCommitChecker
	moduleDefaultBranchGetter ModuleDefaultBranchGetter
	syncPointResolver         SyncPointResolver

	// flags received on creation
	sortedModulesDirsForSync             []string
	modulesDirsToIdentityOverrideForSync map[string]bufmoduleref.ModuleIdentity // moduleDir:moduleIdentityOverride
	syncAllBranches                      bool

	commitsToTags                   map[string][]string                               // commits:[]tags
	branchesToModulesForSync        map[string]map[string]bufmoduleref.ModuleIdentity // branch:moduleDir:targetModuleIdentity
	modulesToBranchesLastSyncPoints map[string]map[string]string                      // moduleIdentity:branch:lastSyncPointGitHash

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
		branchesToModulesForSync:              make(map[string]map[string]bufmoduleref.ModuleIdentity),
		modulesToBranchesLastSyncPoints:       make(map[string]map[string]string),
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
	// first, default branch, if present
	defaultBranch := s.repo.DefaultBranch()
	if _, shouldSyncDefaultBranch := s.branchesToModulesForSync[defaultBranch]; shouldSyncDefaultBranch {
		if err := s.syncBranch(ctx, defaultBranch, syncFunc); err != nil {
			return fmt.Errorf("sync default branch %s: %w", defaultBranch, err)
		}
	}
	// then the rest of the branches, in a deterministic order
	var sortedBranchesForSync []string
	for branch := range s.branchesToModulesForSync {
		if branch == defaultBranch {
			continue // default branch was already synced
		}
		sortedBranchesForSync = append(sortedBranchesForSync, branch)
	}
	sort.Strings(sortedBranchesForSync)
	for _, branch := range sortedBranchesForSync {
		if err := s.syncBranch(ctx, branch, syncFunc); err != nil {
			return fmt.Errorf("sync branch %s: %w", branch, err)
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
	if s.syncAllBranches {
		// make sure the default branch is present in the branches to sync
		defaultBranch := s.repo.DefaultBranch()
		if _, isDefaultBranchPushedInRemote := allRemoteBranches[defaultBranch]; !isDefaultBranchPushedInRemote {
			return fmt.Errorf("default branch %s is not present in 'origin' remote", defaultBranch)
		}
		for remoteBranch := range allRemoteBranches {
			s.branchesToModulesForSync[remoteBranch] = make(map[string]bufmoduleref.ModuleIdentity)
		}
	} else {
		// only sync current branch, make sure it's present in the remote
		currentBranch := s.repo.CurrentBranch()
		if _, isCurrentBranchPushedInRemote := allRemoteBranches[currentBranch]; !isCurrentBranchPushedInRemote {
			return fmt.Errorf("current branch %s is not present in 'origin' remote", currentBranch)
		}
		s.branchesToModulesForSync[currentBranch] = make(map[string]bufmoduleref.ModuleIdentity)
		s.logger.Debug("current branch", zap.String("name", currentBranch))
	}
	// Populate module identities, from identity overrides or from HEAD, and its sync points if any
	allModulesIdentitiesForSync := make(map[string]bufmoduleref.ModuleIdentity) // moduleIdentityString:moduleIdentity
	for branch := range s.branchesToModulesForSync {
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
			s.branchesToModulesForSync[branch][moduleDir] = targetModuleIdentity
			targetModuleIdentityString := targetModuleIdentity.IdentityString()
			// do we have a remote git sync point for this module+branch?
			moduleBranchSyncPoint, err := s.resolveSyncPoint(ctx, targetModuleIdentity, branch)
			if err != nil {
				return fmt.Errorf("resolve sync point for module %s in branch %s: %w", targetModuleIdentityString, branch, err)
			}
			allModulesIdentitiesForSync[targetModuleIdentityString] = targetModuleIdentity
			if s.modulesToBranchesLastSyncPoints[targetModuleIdentityString] == nil {
				s.modulesToBranchesLastSyncPoints[targetModuleIdentityString] = make(map[string]string)
			}
			if moduleBranchSyncPoint != nil {
				s.modulesToBranchesLastSyncPoints[targetModuleIdentityString][branch] = moduleBranchSyncPoint.Hex()
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

// hasSomethingForSync returns true if there is at least one module in a branch to sync.
func (s *syncer) hasSomethingForSync() bool {
	for _, modules := range s.branchesToModulesForSync {
		for range modules {
			return true
		}
	}
	return false
}

// syncBranch modules from a branch.
//
// It first navigates the branch calculating the commits+modules that are to be synced. Once all
// modules have their initial git sync point we loop over those commits and invoke the sync
// function.
//
// Any error from the sync func aborts the sync process and leaves it partially complete, safe to
// resume.
func (s *syncer) syncBranch(ctx context.Context, branch string, syncFunc SyncFunc) error {
	commitsForSync, err := s.branchSyncableCommits(ctx, branch)
	if err != nil {
		return fmt.Errorf("finding commits to sync: %w", err)
	}
	// first lookback from the starting point, syncing old tags
	var startSyncPoint git.Hash
	if len(commitsForSync) == 0 {
		headCommit, err := s.repo.HEADCommit(branch)
		if err != nil {
			return fmt.Errorf("read HEAD commit for branch %s: %w", branch, err)
		}
		startSyncPoint = headCommit.Hash()
	} else {
		startSyncPoint = commitsForSync[0].commit.Hash()
	}
	if err := s.syncLookback(ctx, branch, commitsForSync); err != nil {
		return fmt.Errorf("sync looking back for branch %s at commit %s: %w", branch, startSyncPoint.Hex(), err)
	}
	// then sync the new commits
	if len(commitsForSync) == 0 {
		s.logger.Debug(
			"no modules to sync in branch",
			zap.String("branch", branch),
		)
		return nil
	}
	s.printCommitsForSync(branch, commitsForSync)
	for _, commitForSync := range commitsForSync {
		commitHash := commitForSync.commit.Hash().Hex()
		if len(commitForSync.modules) == 0 {
			s.logger.Debug(
				"branch commit with no modules to sync, skipping commit",
				zap.String("branch", branch),
				zap.String("commit", commitHash),
			)
			continue
		}
		for _, moduleDir := range s.sortedModulesDirsForSync { // looping over the original sort order of modules
			builtModule, shouldSyncModule := commitForSync.modules[moduleDir]
			if !shouldSyncModule {
				s.logger.Debug(
					"module directory not present as a module to sync, skipping module in commit",
					zap.String("branch", branch),
					zap.String("commit", commitHash),
					zap.String("module directory", moduleDir),
				)
				continue
			}
			if builtModule == nil {
				s.logger.Debug(
					"module directory has no module to sync, skipping module in commit",
					zap.String("branch", branch),
					zap.String("commit", commitHash),
					zap.String("module directory", moduleDir),
				)
				continue
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
	}
	return nil
}

// branchSyncableCommits returns a sorted commit+modules tuples array that are pending to sync for a
// branch. A commit in the array might have no modules to sync if those are skipped by the Syncer
// error handler, or are a found sync point.
func (s *syncer) branchSyncableCommits(ctx context.Context, branch string) ([]*syncableCommit, error) {
	branchModulesForSync, ok := s.branchesToModulesForSync[branch]
	if !ok || len(branchModulesForSync) == 0 {
		// branch should not be synced, or no modules to sync in that branch
		return nil, nil
	}
	// Copy all branch modules to sync and mark them as pending, until its starting sync point is
	// reached. They'll be removed from this list as its initial sync point is found.
	pendingModules := s.copyBranchModulesSync(branch, branchModulesForSync)
	var commitsForSync []*syncableCommit
	stopLoopErr := errors.New("stop loop")
	eachCommitFunc := func(commit git.Commit) error {
		if len(pendingModules) == 0 {
			// no more pending modules to sync, no need to keep navigating the branch
			return stopLoopErr
		}
		commitHash := commit.Hash().Hex()
		syncModules := make(map[string]*bufmodulebuild.BuiltModule) // modules to be queued for sync in this commit
		stopModules := make(map[string]struct{})                    // modules to stop looking in this commit
		for moduleDir, pendingModule := range pendingModules {
			targetModuleIdentity := pendingModule.targetModuleIdentity.IdentityString()
			logger := s.logger.With(
				zap.String("branch", branch),
				zap.String("commit", commit.Hash().Hex()),
				zap.String("module directory", moduleDir),
				zap.String("target module identity", targetModuleIdentity),
				zap.String("expected sync point", pendingModule.expectedSyncPoint),
			)
			// check if the remote module already synced this commit
			isSynced, err := s.isGitCommitSynced(ctx, pendingModule.targetModuleIdentity, commitHash)
			if err != nil {
				return fmt.Errorf(
					"checking if module %s already synced git commit %s: %w",
					targetModuleIdentity, commitHash, err,
				)
			}
			if isSynced {
				// reached a commit that is already synced for this module, we can stop looking for this
				// module dir
				stopModules[moduleDir] = struct{}{}
				if pendingModule.expectedSyncPoint == "" {
					// this module did not have an expected sync point for this branch, we probably reached
					// the beginning of the branch off another branch that is already synced.
					logger.Debug("git commit already synced, stop looking back in branch")
					continue
				}
				if commitHash != pendingModule.expectedSyncPoint {
					if s.repo.DefaultBranch() == branch {
						// if we reached a commit that is already synced, but it's not the expected one in the
						// default branch, abort sync.
						return fmt.Errorf(
							"found synced git commit %s for default branch %s, but expected sync point was %s, "+
								"did you rebase or reset your default branch?",
							commitHash,
							branch,
							pendingModule.expectedSyncPoint,
						)
					}
					// syncing non-default branches from an unexpected sync point can be a common scenario in
					// PRs, we can just WARN and stop looking back for this branch.
					logger.Warn(
						"unexpected sync point reached, stop looking back in branch",
						zap.String("found_sync_point", commitHash),
					)
				} else {
					logger.Debug("expected sync point reached, stop looking back in branch")
				}
				continue
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
					stopModules[moduleDir] = struct{}{}
				case LookbackDecisionCodeOverride:
					logger.Debug("read module at commit failed, overriding module identity in commit", zap.Error(readErr))
					if builtModule == nil {
						return fmt.Errorf("cannot override commit, no built module: %w", readErr)
					}
					// rename the module to the target identity, and add it to the queue
					renamedModule, err := renameModule(ctx, builtModule, pendingModule.targetModuleIdentity)
					if err != nil {
						return fmt.Errorf("override module in commit: %s, rename module: %w", readErr.Error(), err)
					}
					syncModules[moduleDir] = renamedModule
				default:
					return fmt.Errorf("unexpected decision code %d for read module error %w", decision, readErr)
				}
				continue
			}
			// add the read module to sync
			syncModules[moduleDir] = builtModule
		}
		// clear modules that are set to stop in this commit
		for moduleDir := range stopModules {
			delete(pendingModules, moduleDir)
		}
		commitsForSync = append(commitsForSync, &syncableCommit{
			commit:  commit,
			modules: syncModules,
		})
		return nil
	}
	if err := s.repo.ForEachCommit(eachCommitFunc, git.ForEachCommitWithBranchStartPoint(branch)); err != nil && !errors.Is(err, stopLoopErr) {
		return nil, err
	}
	// if we have no commits to sync, no need to make more checks, bail early
	if len(commitsForSync) == 0 {
		return nil, nil
	}
	// we reached a stopping point for all modules or the branch starting point (no  more commit
	// parents), do we still have pending modules?
	for moduleDir, pendingModule := range pendingModules {
		targetModuleIdentity := pendingModule.targetModuleIdentity.IdentityString()
		logger := s.logger.With(
			zap.String("branch", branch),
			zap.String("module directory", moduleDir),
			zap.String("target module identity", targetModuleIdentity),
		)
		if pendingModule.expectedSyncPoint != "" {
			if branch == s.repo.DefaultBranch() {
				return nil, fmt.Errorf(
					"module %s in directory %s in the default branch %s did not find its expected sync point %s, aborting sync",
					targetModuleIdentity,
					moduleDir,
					branch,
					pendingModule.expectedSyncPoint,
				)
			}
			logger.Warn(
				"module did not find its expected sync point, or any other synced git commit, "+
					"will sync all the way from the beginning of the branch",
				zap.String("expected sync point", pendingModule.expectedSyncPoint),
			)
		}
		logger.Debug(
			"module without expected sync point did not find any synced git commit, " +
				"will sync all the way from the beginning of the branch",
		)
	}
	// reverse commits to sync, to leave them in time order parent -> children
	// https://github.com/golang/go/wiki/SliceTricks#reversing
	for i := len(commitsForSync)/2 - 1; i >= 0; i-- {
		opp := len(commitsForSync) - 1 - i
		commitsForSync[i], commitsForSync[opp] = commitsForSync[opp], commitsForSync[i]
	}
	return commitsForSync, nil
}

// copyBranchModulesSync makes a copy of the modules to sync in the branch and returns it in the
// format of moduleDir:moduleTarget, which have the module identity and the expected sync point, if
// any.
func (s *syncer) copyBranchModulesSync(branch string, modulesDirsToIdentity map[string]bufmoduleref.ModuleIdentity) map[string]moduleTarget {
	pendingModules := make(map[string]moduleTarget, len(modulesDirsToIdentity))
	for moduleDir, moduleIdentity := range modulesDirsToIdentity {
		var expectedSyncPoint string
		if moduleSyncPoints, ok := s.modulesToBranchesLastSyncPoints[moduleIdentity.IdentityString()]; ok {
			if moduleBranchSyncPoint, ok := moduleSyncPoints[branch]; ok {
				expectedSyncPoint = moduleBranchSyncPoint
			}
		}
		pendingModules[moduleDir] = moduleTarget{
			targetModuleIdentity: moduleIdentity,
			expectedSyncPoint:    expectedSyncPoint,
		}
	}
	return pendingModules
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

// syncLookback takes calculated syncable commits for a branch, and looks back
func (s *syncer) syncLookback(ctx context.Context, branch string, syncableCommits []*syncableCommit) error {
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
		zap.Any("branchesModulesToSync", s.branchesToModulesForSync),
		zap.Any("modulesBranchesSyncPoints", s.modulesToBranchesLastSyncPoints),
	)
}

// printCommitsForSync prints syncable commits for a given branch.
func (s *syncer) printCommitsForSync(branch string, syncableCommits []*syncableCommit) {
	printableCommits := make([]map[string]string, 0)
	for _, sCommit := range syncableCommits {
		var commitModules []string
		for moduleDir, builtModule := range sCommit.modules {
			commitModules = append(commitModules, moduleDir+":"+builtModule.ModuleIdentity().IdentityString())
		}
		printableCommits = append(printableCommits, map[string]string{
			sCommit.commit.Hash().Hex(): fmt.Sprintf("(%d)[%s]", len(commitModules), strings.Join(commitModules, ", ")),
		})
	}
	s.logger.Debug(
		"branch commits to sync",
		zap.String("branch", branch),
		zap.Any("commits", printableCommits),
	)
}

// syncableCommit holds the modules that need to be synced in a git commit.
type syncableCommit struct {
	commit  git.Commit
	modules map[string]*bufmodulebuild.BuiltModule // moduleDir:builtModule
}

// moduleTarget is the format to use for pending modules while looking back in a branch for a
// stopping point. When looking back the syncer needs to know what's the target module identity for
// that branch (either read from in HEAD, or the passed module identity override) to compare with
// each commit's module identity, and if there is any BSR expected sync point for that module in
// that branch.
type moduleTarget struct {
	targetModuleIdentity bufmoduleref.ModuleIdentity
	expectedSyncPoint    string
}

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
