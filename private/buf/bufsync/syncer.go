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
	sortedModulesDirsToSync []string
	modulesDirsToSync       map[string]struct{}
	syncAllBranches         bool

	commitsTags                   map[string][]string                               // commits:[]tags
	branchesModulesToSync         map[string]map[string]bufmoduleref.ModuleIdentity // branch:moduleDir:moduleIdentityInHEAD
	modulesBranchesLastSyncPoints map[string]map[string]string                      // moduleIdentity:branch:lastSyncPointGitHash

	// syncedModulesCommitsCache (moduleIdentity:commit) caches commits already synced to a given BSR
	// module, so we don't ask twice the same module:commit when we already know it's already synced.
	// We don't cache "unsynced" git commits, because during the sync process we will be syncing new
	// git commits, which then will be added also to this cache.
	syncedModulesCommitsCache map[string]map[string]struct{}
}

func newSyncer(
	logger *zap.Logger,
	repo git.Repository,
	storageGitProvider storagegit.Provider,
	errorHandler ErrorHandler,
	options ...SyncerOption,
) (Syncer, error) {
	s := &syncer{
		logger:                        logger,
		repo:                          repo,
		storageGitProvider:            storageGitProvider,
		errorHandler:                  errorHandler,
		modulesDirsToSync:             make(map[string]struct{}),
		commitsTags:                   make(map[string][]string),
		branchesModulesToSync:         make(map[string]map[string]bufmoduleref.ModuleIdentity),
		modulesBranchesLastSyncPoints: make(map[string]map[string]string),
		syncedModulesCommitsCache:     make(map[string]map[string]struct{}),
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
	if !s.somethingToSync() {
		s.logger.Warn("branches and modules directories scanned, nothing to sync")
		return nil
	}
	// first, default branch, if present
	defaultBranch := s.repo.DefaultBranch()
	if _, shouldSyncDefaultBranch := s.branchesModulesToSync[defaultBranch]; shouldSyncDefaultBranch {
		if err := s.syncBranch(ctx, defaultBranch, syncFunc); err != nil {
			return fmt.Errorf("sync default branch %s: %w", defaultBranch, err)
		}
	}
	// then the rest of the branches, in a deterministic order
	var sortedBranchesToSync []string
	for branch := range s.branchesModulesToSync {
		if branch == defaultBranch {
			continue // default branch was already synced
		}
		sortedBranchesToSync = append(sortedBranchesToSync, branch)
	}
	sort.Strings(sortedBranchesToSync)
	for _, branch := range sortedBranchesToSync {
		if err := s.syncBranch(ctx, branch, syncFunc); err != nil {
			return fmt.Errorf("sync branch %s: %w", branch, err)
		}
	}
	return nil
}

func (s *syncer) prepareSync(ctx context.Context) error {
	// Populate all tags locations.
	if err := s.repo.ForEachTag(func(tag string, commitHash git.Hash) error {
		s.commitsTags[commitHash.Hex()] = append(s.commitsTags[commitHash.Hex()], tag)
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
			s.branchesModulesToSync[remoteBranch] = make(map[string]bufmoduleref.ModuleIdentity)
		}
	} else {
		// only sync current branch, make sure it's present in the remote
		currentBranch := s.repo.CurrentBranch()
		if _, isCurrentBranchPushedInRemote := allRemoteBranches[currentBranch]; !isCurrentBranchPushedInRemote {
			return fmt.Errorf("current branch %s is not present in 'origin' remote", currentBranch)
		}
		s.branchesModulesToSync[currentBranch] = make(map[string]bufmoduleref.ModuleIdentity)
		s.logger.Debug("current branch", zap.String("name", currentBranch))
	}
	// Populate module identities from HEAD, and its sync points if any
	allModulesIdentitiesToSync := make(map[string]bufmoduleref.ModuleIdentity) // moduleIdentityString:moduleIdentity
	for branch := range s.branchesModulesToSync {
		headCommit, err := s.repo.HEADCommit(branch)
		if err != nil {
			return fmt.Errorf("reading head commit for branch %s: %w", branch, err)
		}
		for moduleDir := range s.modulesDirsToSync {
			builtModule, readErr := s.readModuleAt(ctx, branch, headCommit, moduleDir, nil /* no specific module identity expected */)
			if readErr != nil {
				// any error reading module in HEAD, skip syncing that module in that branch
				s.logger.Warn(
					"read module from HEAD failed, module won't be synced for this branch",
					zap.Error(readErr),
				)
				continue
			}
			// there is a valid module in the module dir at the HEAD of this branch, enqueue it for sync
			s.branchesModulesToSync[branch][moduleDir] = builtModule.ModuleIdentity()
			// do we have a remote git sync point for this module+branch?
			moduleIdentityInHEAD := builtModule.ModuleIdentity().IdentityString()
			moduleBranchSyncPoint, err := s.resolveSyncPoint(ctx, builtModule.ModuleIdentity(), branch)
			if err != nil {
				return fmt.Errorf(
					"resolve sync point for module %s in branch %s HEAD commit %s: %w",
					moduleIdentityInHEAD, branch, headCommit.Hash().Hex(), err,
				)
			}
			allModulesIdentitiesToSync[moduleIdentityInHEAD] = builtModule.ModuleIdentity()
			if s.modulesBranchesLastSyncPoints[moduleIdentityInHEAD] == nil {
				s.modulesBranchesLastSyncPoints[moduleIdentityInHEAD] = make(map[string]string)
			}
			if moduleBranchSyncPoint != nil {
				s.modulesBranchesLastSyncPoints[moduleIdentityInHEAD][branch] = moduleBranchSyncPoint.Hex()
			}
		}
	}
	// make sure all module identities we are about to sync in all branches have the same BSR default
	// branch as the local git default branch.
	for _, moduleIdentity := range allModulesIdentitiesToSync {
		if err := s.validateDefaultBranch(ctx, moduleIdentity); err != nil {
			return fmt.Errorf("validate default branch for module %s: %w", moduleIdentity.IdentityString(), err)
		}
	}
	return nil
}

// resolveSyncPoint resolves a sync point for a particular module identity and branch.
func (s *syncer) resolveSyncPoint(ctx context.Context, module bufmoduleref.ModuleIdentity, branch string) (git.Hash, error) {
	// If resumption is not enabled, we can bail early.
	if s.syncPointResolver == nil {
		return nil, nil
	}
	syncPoint, err := s.syncPointResolver(ctx, module, branch)
	if err != nil {
		return nil, fmt.Errorf("resolve sync point for module %s: %w", module.IdentityString(), err)
	}
	if syncPoint == nil {
		// no sync point for that module in that branch
		return nil, nil
	}
	// Validate that the commit pointed to by the sync point exists in the git repo.
	if _, err := s.repo.Objects().Commit(syncPoint); err != nil {
		isDefaultBranch := branch == s.repo.DefaultBranch()
		return nil, s.errorHandler.InvalidRemoteSyncPoint(module, branch, syncPoint, isDefaultBranch, err)
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

// somethingToSync returns true if there is at least one module in a branch to sync.
func (s *syncer) somethingToSync() bool {
	for _, modules := range s.branchesModulesToSync {
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
	commitsToSync, err := s.branchCommitsToSync(ctx, branch)
	if err != nil {
		return fmt.Errorf("finding commits to sync: %w", err)
	}
	if len(commitsToSync) == 0 {
		s.logger.Debug(
			"no modules to sync in branch",
			zap.String("branch", branch),
		)
		return nil
	}
	s.printCommitsToSync(branch, commitsToSync)
	for _, commitToSync := range commitsToSync {
		commitHash := commitToSync.commit.Hash().Hex()
		if len(commitToSync.modules) == 0 {
			s.logger.Debug(
				"branch commit with no modules to sync, skipping commit",
				zap.String("branch", branch),
				zap.String("commit", commitHash),
			)
			continue
		}
		for _, moduleDir := range s.sortedModulesDirsToSync { // looping over the original sort order of modules
			builtModule, shouldSyncModule := commitToSync.modules[moduleDir]
			if !shouldSyncModule {
				s.logger.Debug(
					"module directory not present as a module to sync, skipping module in commit",
					zap.String("branch", branch),
					zap.String("commit", commitHash),
					zap.String("module directory", moduleDir),
				)
				continue
			}
			modIdentity := builtModule.ModuleIdentity().IdentityString()
			if err := syncFunc(
				ctx,
				newModuleCommitToSync(
					branch,
					commitToSync.commit,
					s.commitsTags[commitHash],
					moduleDir,
					builtModule.ModuleIdentity(),
					builtModule.Bucket,
				),
			); err != nil {
				return fmt.Errorf("sync module %s:%s in commit %s: %w", moduleDir, modIdentity, commitHash, err)
			}
			// module was synced successfully, add it to the cache
			if s.syncedModulesCommitsCache[modIdentity] == nil {
				s.syncedModulesCommitsCache[modIdentity] = make(map[string]struct{})
			}
			s.syncedModulesCommitsCache[modIdentity][commitHash] = struct{}{}
		}
	}
	return nil
}

// branchCommitsToSync returns a sorted commit+modules tuples array that are pending to sync for a
// branch. A commit in the array might have no modules to sync if those are skipped by the
// Syncer error handler, or are a found sync point.
func (s *syncer) branchCommitsToSync(ctx context.Context, branch string) ([]*syncableCommit, error) {
	branchModulesToSync, ok := s.branchesModulesToSync[branch]
	if !ok || len(branchModulesToSync) == 0 {
		// branch should not be synced, or no modules to sync in that branch
		return nil, nil
	}
	// Copy all branch modules to sync and mark them as pending, until its starting sync point is
	// reached. They'll be removed from this list as its initial sync point is found.
	pendingModules := s.copyBranchModulesSync(branch, branchModulesToSync)
	var commitsToSync []*syncableCommit
	// travel branch commits from HEAD and check if they're already synced, until finding a synced git
	// commit, or adding them all to be synced
	stopLoopErr := errors.New("stop loop")
	if err := s.repo.ForEachCommit(branch, func(commit git.Commit) error {
		if len(pendingModules) == 0 {
			// no more pending modules to sync, no need to keep navigating the branch
			return stopLoopErr
		}
		commitHash := commit.Hash().Hex()
		modulesDirsToSyncInThisCommit := make(map[string]bufmodulebuild.BuiltModule)
		modulesDirsToStopInThisCommit := make(map[string]struct{})
		for moduleDir, pendingModule := range pendingModules {
			moduleIdentityInHEAD := pendingModule.moduleIdentityInHEAD.IdentityString()
			logger := s.logger.With(
				zap.String("branch", branch),
				zap.String("commit", commit.Hash().Hex()),
				zap.String("module directory", moduleDir),
				zap.String("module identity in branch HEAD", moduleIdentityInHEAD),
				zap.Stringp("expected sync point", pendingModule.expectedSyncPoint),
			)
			// check if the remote module already synced this commit
			isSynced, err := s.isGitCommitSynced(ctx, pendingModule.moduleIdentityInHEAD, commitHash)
			if err != nil {
				return fmt.Errorf(
					"checking if module %s already synced git commit %s: %w",
					moduleIdentityInHEAD, commitHash, err,
				)
			}
			if isSynced {
				// reached a commit that is already synced for this module, we can stop looking for this
				// module dir
				modulesDirsToStopInThisCommit[moduleDir] = struct{}{}
				if pendingModule.expectedSyncPoint == nil {
					// this module did not have an expected sync point for this branch, we probably reached
					// the beginning of the branch off another branch that is already synced.
					logger.Debug("git commit already synced, stop looking back in branch")
					continue
				}
				if commitHash != *pendingModule.expectedSyncPoint {
					if s.repo.DefaultBranch() == branch {
						// if we reached a commit that is already synced, but it's not the expected one in the
						// default branch, abort sync.
						return fmt.Errorf(
							"found synced git commit %s for default branch %s, but expected sync point was %s, "+
								"did you rebase or reset your default branch?",
							commitHash,
							branch,
							*pendingModule.expectedSyncPoint,
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
			builtModule, readErr := s.readModuleAt(ctx, branch, commit, moduleDir, &moduleIdentityInHEAD)
			if readErr != nil {
				if s.errorHandler.StopLookback(readErr) {
					logger.Warn("read module at commit failed, stop looking back in branch", zap.Error(readErr))
					modulesDirsToStopInThisCommit[moduleDir] = struct{}{}
					continue
				}
				logger.Debug("read module at commit failed, skipping commit", zap.Error(readErr))
				continue
			}
			// add the read module to sync
			modulesDirsToSyncInThisCommit[moduleDir] = *builtModule
		}
		// clear modules that are set to stop in this commit
		for moduleDir := range modulesDirsToStopInThisCommit {
			delete(pendingModules, moduleDir)
		}
		commitsToSync = append(commitsToSync, &syncableCommit{
			commit:  commit,
			modules: modulesDirsToSyncInThisCommit,
		})
		return nil
	}); err != nil && !errors.Is(err, stopLoopErr) {
		return nil, err
	}
	// if we have no commits to sync, no need to make more checks, bail early
	if len(commitsToSync) == 0 {
		return nil, nil
	}
	// we reached a stopping point for all modules or the branch starting point (no  more commit
	// parents), do we still have pending modules?
	for moduleDir, pendingModule := range pendingModules {
		moduleIdentityInHEAD := pendingModule.moduleIdentityInHEAD.IdentityString()
		logger := s.logger.With(
			zap.String("branch", branch),
			zap.String("module directory", moduleDir),
			zap.String("module identity in branch HEAD", moduleIdentityInHEAD),
		)
		if pendingModule.expectedSyncPoint != nil {
			if branch == s.repo.DefaultBranch() {
				return nil, fmt.Errorf(
					"module %s in directory %s in the default branch %s did not find its expected sync point %s, aborting sync",
					moduleIdentityInHEAD,
					moduleDir,
					branch,
					*pendingModule.expectedSyncPoint,
				)
			}
			logger.Warn(
				"module did not find its expected sync point, or any other synced git commit, "+
					"will sync all the way from the beginning of the branch",
				zap.String("expected sync point", *pendingModule.expectedSyncPoint),
			)
		}
		logger.Debug(
			"module without expected sync point did not find any synced git commit, " +
				"will sync all the way from the beginning of the branch",
		)
	}
	// reverse commits to sync, to leave them in time order parent -> children
	// https://github.com/golang/go/wiki/SliceTricks#reversing
	for i := len(commitsToSync)/2 - 1; i >= 0; i-- {
		opp := len(commitsToSync) - 1 - i
		commitsToSync[i], commitsToSync[opp] = commitsToSync[opp], commitsToSync[i]
	}
	return commitsToSync, nil
}

// copyBranchModulesSync makes a copy of the modules to sync in the branch and returns it in the format of
// moduleDir:moduleTarget, which have the module identity in HEAD, and the expected sync point, if
// any.
func (s *syncer) copyBranchModulesSync(branch string, branchModulesToSync map[string]bufmoduleref.ModuleIdentity) map[string]moduleTarget {
	pendingModules := make(map[string]moduleTarget, len(branchModulesToSync))
	for moduleDir, moduleIdentityInHEAD := range branchModulesToSync {
		var expectedSyncPoint *string
		if moduleSyncPoints, ok := s.modulesBranchesLastSyncPoints[moduleIdentityInHEAD.IdentityString()]; ok {
			if moduleBranchSyncPoint, ok := moduleSyncPoints[branch]; ok {
				expectedSyncPoint = &moduleBranchSyncPoint
			}
		}
		pendingModules[moduleDir] = moduleTarget{
			moduleIdentityInHEAD: moduleIdentityInHEAD,
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
	if syncedModuleCommits, ok := s.syncedModulesCommitsCache[modIdentity]; ok {
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
		if s.syncedModulesCommitsCache[modIdentity] == nil {
			s.syncedModulesCommitsCache[modIdentity] = make(map[string]struct{})
		}
		s.syncedModulesCommitsCache[modIdentity][commitHash] = struct{}{}
	}
	return commitSynced, nil
}

// readModuleAt returns a module that has a name and builds correctly given a commit and a module
// directory, or a read error.
func (s *syncer) readModuleAt(
	ctx context.Context,
	branch string,
	commit git.Commit,
	moduleDir string,
	expectedModuleIdentity *string, // optional
) (*bufmodulebuild.BuiltModule, *ReadModuleError) {
	// in case there is an error reading this module, it will have the same branch, commit, and module
	// dir that we can fill upfront. The actual `err` and `code` (if any) is populated in case-by-case
	// basis before returning.
	readErr := &ReadModuleError{
		branch:    branch,
		commit:    commit.Hash().Hex(),
		moduleDir: moduleDir,
	}
	commitBucket, err := s.storageGitProvider.NewReadBucket(commit.Tree(), storagegit.ReadBucketWithSymlinksIfSupported())
	if err != nil {
		readErr.err = fmt.Errorf("new read bucket: %w", err)
		return nil, readErr
	}
	moduleBucket := storage.MapReadBucket(commitBucket, storage.MapOnPrefix(moduleDir))
	foundModule, err := bufconfig.ExistingConfigFilePath(ctx, moduleBucket)
	if err != nil {
		readErr.err = fmt.Errorf("looking for an existing config file path: %w", err)
		return nil, readErr
	}
	if foundModule == "" {
		readErr.code = ReadModuleErrorCodeModuleNotFound
		readErr.err = errors.New("module not found")
		return nil, readErr
	}
	sourceConfig, err := bufconfig.GetConfigForBucket(ctx, moduleBucket)
	if err != nil {
		readErr.code = ReadModuleErrorCodeInvalidModuleConfig
		readErr.err = fmt.Errorf("invalid module config: %w", err)
		return nil, readErr
	}
	if sourceConfig.ModuleIdentity == nil {
		readErr.code = ReadModuleErrorCodeUnnamedModule
		readErr.err = errors.New("found module does not have a name")
		return nil, readErr
	}
	if expectedModuleIdentity != nil {
		if sourceConfig.ModuleIdentity.IdentityString() != *expectedModuleIdentity {
			readErr.code = ReadModuleErrorCodeUnexpectedName
			readErr.err = fmt.Errorf(
				"read module has an unexpected module identity %s, expected %s",
				sourceConfig.ModuleIdentity.IdentityString(), *expectedModuleIdentity,
			)
			return nil, readErr
		}
	}
	builtModule, err := bufmodulebuild.NewModuleBucketBuilder().BuildForBucket(
		ctx,
		moduleBucket,
		sourceConfig.Build,
		bufmodulebuild.WithModuleIdentity(sourceConfig.ModuleIdentity),
	)
	if err != nil {
		readErr.code = ReadModuleErrorCodeBuildModule
		readErr.err = fmt.Errorf("build module: %w", err)
		return nil, readErr
	}
	return builtModule, nil
}

// syncableCommit holds the modules that need to be synced in a git commit.
type syncableCommit struct {
	commit  git.Commit
	modules map[string]bufmodulebuild.BuiltModule // moduleDir:builtModule
}

type moduleTarget struct {
	moduleIdentityInHEAD bufmoduleref.ModuleIdentity
	expectedSyncPoint    *string
}
