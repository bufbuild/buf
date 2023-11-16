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
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/slicesextended"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagegit"
	"go.uber.org/multierr"
	"go.uber.org/zap"
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

	// flags received on creation
	gitRemoteName                        string
	sortedModulesDirsForSync             []string
	modulesDirsToIdentityOverrideForSync map[string]bufmoduleref.ModuleIdentity // moduleDir:moduleIdentityOverride
	syncAllBranches                      bool
}

func newSyncer(
	logger *zap.Logger,
	repo git.Repository,
	storageGitProvider storagegit.Provider,
	handler Handler,
	options ...SyncerOption,
) (Syncer, error) {
	s := &syncer{
		logger:                               logger,
		repo:                                 repo,
		storageGitProvider:                   storageGitProvider,
		handler:                              handler,
		modulesDirsToIdentityOverrideForSync: make(map[string]bufmoduleref.ModuleIdentity),
	}
	for _, opt := range options {
		if err := opt(s); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *syncer) Sync(ctx context.Context) error {
	plan, err := s.makeExecutionPlan(ctx)
	if err != nil {
		return fmt.Errorf("sync preparation: %w", err)
	}
	s.logPlan(plan)
	if !plan.hasAnythingToSync() {
		s.logger.Warn("nothing to sync")
		return nil
	}
	for _, syncableBranch := range plan.branchesToSync {
		if err := s.syncSyncableBranch(ctx, syncableBranch); err != nil {
			return fmt.Errorf("sync branch %q for module dir %q: %w", syncableBranch.name, syncableBranch.moduleDir, err)
		}
	}
	if err := s.syncTags(ctx, plan.tagsToSync); err != nil {
		return fmt.Errorf("sync tags: %w", err)
	}
	return nil
}

// resolveSyncPoint resolves a sync point for a particular module identity and protected branch.
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
		// The most likely culprit for an invalid sync point is a rebase, where the last known commit has
		// been garbage collected. In this case, let's present a better error message.
		//
		// This is not trivial scenario if the branch that's been rebased is a long-lived branch (like
		// main) whose artifacts are consumed by other branches, as we may fail to sync those commits if
		// we continue.
		//
		// For now we simply error if this happens.
		if errors.Is(err, git.ErrObjectNotFound) {
			return nil, fmt.Errorf(
				"last synced git commit %q for default branch %q in module %q is not found in the git repo, did you rebase or reset your default branch?",
				syncPoint.Hex(), branch, moduleIdentity.IdentityString(),
			)
		}
		// Other error, let's abort sync.
		return nil, fmt.Errorf(
			"invalid sync point %q for branch %q in module %q: %w",
			syncPoint.Hex(), branch, moduleIdentity.IdentityString(), err,
		)
	}
	return syncPoint, nil
}

// syncModuleInBranch syncs a module directory in a branch.
func (s *syncer) syncSyncableBranch(ctx context.Context, syncableBranch *syncableBranch) error {
	logger := s.logger.With(
		zap.String("module directory", syncableBranch.moduleDir),
		zap.String("module identity", syncableBranch.moduleIdentity.IdentityString()),
		zap.String("branch", syncableBranch.name),
	)
	for i, commitToSync := range syncableBranch.commitsToSync {
		module, err := s.readModuleAt(ctx, commitToSync.commit, syncableBranch.moduleDir)
		if err != nil {
			if errors.Is(err, errReadModuleInvalidModule) || errors.Is(err, errReadModuleInvalidModuleConfig) {
				// If this is not the last commit to sync, i.e., the HEAD of the branch, we log a warning only.
				if i != len(syncableBranch.commitsToSync)-1 {
					logger.Debug(
						"module read failed",
						zap.Stringer("commit", commitToSync.commit.Hash()),
						zap.Error(err),
					)
				}
			}
			return fmt.Errorf(
				"module %q read failed @ HEAD on branch %q: %w",
				syncableBranch.moduleIdentity.IdentityString(),
				syncableBranch.name,
				err,
			)
		}
		if module == nil {
			logger.Debug(
				"no module, skipping commit",
				zap.Stringer("commit", commitToSync.commit.Hash()),
				zap.Error(err),
			)
			continue
		}
		if err := s.handler.SyncModuleCommit(
			ctx,
			newModuleCommit(
				syncableBranch.name,
				commitToSync.commit,
				commitToSync.tags,
				syncableBranch.moduleDir,
				syncableBranch.moduleIdentity,
				module.Bucket,
			),
		); err != nil {
			return fmt.Errorf(
				"sync module %s:%s in commit %s: %w",
				syncableBranch.moduleDir,
				syncableBranch.moduleIdentity,
				commitToSync.commit.Hash(),
				err,
			)
		}
	}
	return nil
}

// readModule returns a module that has a name and builds correctly given a commit and a module
// directory.
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

// syncTags syncs all tag commits for commits that are sycned. This should be run _at the end_ of syncing.
func (s *syncer) syncTags(ctx context.Context, commitTags []*syncableCommitTags) error {
	commitTagsToSyncPerModuleIdentity := make(map[bufmoduleref.ModuleIdentity]map[git.Hash][]string)
	for _, commit := range commitTags {
		if synced, err := s.handler.IsGitCommitSynced(ctx, commit.moduleIdentity, commit.commit); err != nil {
			return fmt.Errorf("check if tagged commit %q is synced: %w", commit.commit, err)
		} else if !synced {
			s.logger.Debug(
				"commit referenced by tags is not synced, skipping tags",
				zap.String("module identity", commit.moduleIdentity.IdentityString()),
				zap.Stringer("commit", commit.commit),
				zap.Strings("tags", commit.tagsToSync),
			)
			continue
		}
		if _, ok := commitTagsToSyncPerModuleIdentity[commit.moduleIdentity]; !ok {
			commitTagsToSyncPerModuleIdentity[commit.moduleIdentity] = make(map[git.Hash][]string)
		}
		commitTagsToSyncPerModuleIdentity[commit.moduleIdentity][commit.commit] = commit.tagsToSync
	}
	for moduleIdentity, commitTagsToSync := range commitTagsToSyncPerModuleIdentity {
		err := s.handler.SyncModuleTags(ctx, moduleIdentity, commitTagsToSync)
		if err != nil {
			return fmt.Errorf("put tags %q for module identity %q: %w", commitTagsToSync, moduleIdentity, err)
		}
		s.logger.Debug(
			"tags put",
			zap.String("module identity", moduleIdentity.IdentityString()),
			zap.Any("tags", commitTagsToSync),
		)
	}
	return nil
}

func (s *syncer) makeExecutionPlan(ctx context.Context) (*executionPlan, error) {
	branchesToSync, tagsToSync, err := s.determineEverythingToSync(ctx)
	if err != nil {
		return nil, fmt.Errorf("determine branches to sync: %w", err)
	}
	return newExecutionPlan(branchesToSync, tagsToSync), nil
}

func (s *syncer) determineEverythingToSync(ctx context.Context) ([]*syncableBranch, []*syncableCommitTags, error) {
	var branchesToSync []string
	if s.syncAllBranches {
		if err := s.repo.ForEachBranch(func(branch string, _ git.Hash) error {
			branchesToSync = append(branchesToSync, branch)
			return nil
		}, git.ForEachBranchWithRemote(s.gitRemoteName)); err != nil {
			return nil, nil, fmt.Errorf("looping over repository branches: %w", err)
		}
	} else {
		currentBranch, err := s.repo.CheckedOutBranch()
		if err != nil {
			return nil, nil, fmt.Errorf("determine checked out branch")
		}
		branchesToSync = []string{currentBranch}
	}
	allTags := make(map[git.Hash][]string)
	if err := s.repo.ForEachTag(func(tag string, commitHash git.Hash) error {
		allTags[commitHash] = append(allTags[commitHash], tag)
		return nil
	}); err != nil {
		return nil, nil, err
	}
	var (
		syncableBranches      []*syncableBranch
		tagsPerModuleIdentity = make(map[bufmoduleref.ModuleIdentity]map[git.Hash]map[string]struct{})
	)
	for _, branch := range branchesToSync {
		headCommit, err := s.repo.HEADCommit(
			git.HEADCommitWithBranch(branch),
			git.HEADCommitWithRemote(s.gitRemoteName),
		)
		if err != nil {
			return nil, nil, fmt.Errorf("reading head commit for branch %s: %w", branch, err)
		}
		moduleDirsForModuleIdentity := make(map[string][]string) // moduleIdentity:[]moduleDir
		for moduleDir, identityOverride := range s.modulesDirsToIdentityOverrideForSync {
			// First, determine the set of commits to sync.
			builtModule, readErr := s.readModuleAt(ctx, headCommit, moduleDir)
			if readErr != nil {
				return nil, nil, fmt.Errorf("reading module %q at branch %q HEAD: %w", moduleDir, branch, err)
			} else if builtModule == nil {
				s.logger.Debug(
					"no module on HEAD, skipping branch",
					zap.String("branch", branch),
					zap.String("moduleDir", moduleDir),
				)
				continue
			}
			var targetModuleIdentity bufmoduleref.ModuleIdentity
			if identityOverride == nil {
				targetModuleIdentity = builtModule.ModuleIdentity()
			} else {
				targetModuleIdentity = identityOverride
			}
			moduleDirsForModuleIdentity[targetModuleIdentity.IdentityString()] = append(
				moduleDirsForModuleIdentity[targetModuleIdentity.IdentityString()],
				moduleDir,
			)
			commitsToSync, err := s.determineCommitsAndTagsToSyncForModuleBranch(
				ctx,
				moduleDir,
				targetModuleIdentity,
				branch,
			)
			if err != nil {
				return nil, nil, err
			}
			var synableCommits []*syncableCommit
			// determineCommitsAndTagsToSyncForModuleBranch returns commits in the order in which
			// the branch is iterated:
			// 		HEAD -> parent1 -> .. -> parentN
			// syncableBranch expects commits in the order in which they should be synced:
			// 		parentN -> .. -> parent2 -> parent1 -> HEAD
			// So we iterate in reverse order.
			for i := len(commitsToSync) - 1; i >= 0; i-- {
				commit, err := s.repo.Objects().Commit(commitsToSync[i])
				if err != nil {
					return nil, nil, fmt.Errorf("read commit %q: %w", commitsToSync[i], err)
				}
				synableCommits = append(synableCommits, newSyncableCommit(
					commit,
					allTags[commit.Hash()],
				))
			}
			syncableBranches = append(syncableBranches, newSyncableBranch(
				branch,
				moduleDir,
				targetModuleIdentity,
				synableCommits,
			))
			// Next, collect all tags on this module branch and add them to the correct module.
			taggedCommitsOnBranch, err := s.determineTaggedCommitsOnBranch(ctx, branch, allTags)
			if err != nil {
				return nil, nil, fmt.Errorf("determine tagged commits on branch: %w", err)
			}
			if _, ok := tagsPerModuleIdentity[targetModuleIdentity]; !ok {
				tagsPerModuleIdentity[targetModuleIdentity] = make(map[git.Hash]map[string]struct{})
			}
			for commit, tags := range taggedCommitsOnBranch {
				if _, ok := tagsPerModuleIdentity[targetModuleIdentity][commit]; !ok {
					tagsPerModuleIdentity[targetModuleIdentity][commit] = make(map[string]struct{})
				}
				for _, tag := range tags {
					tagsPerModuleIdentity[targetModuleIdentity][commit][tag] = struct{}{}
				}
			}
		}
		var duplicatedIdentitiesErr error
		for moduleIdentity, moduleDirs := range moduleDirsForModuleIdentity {
			if len(moduleDirs) > 1 {
				duplicatedIdentitiesErr = multierr.Append(duplicatedIdentitiesErr, fmt.Errorf(
					"module identity %s cannot be synced in branch %s: present in multiple module directories: [%s]",
					moduleIdentity, branch, strings.Join(moduleDirs, ", "),
				))
			}
		}
		if duplicatedIdentitiesErr != nil {
			return nil, nil, duplicatedIdentitiesErr
		}
	}
	var syncableCommitTags []*syncableCommitTags
	for moduleIdentity, commits := range tagsPerModuleIdentity {
		for commit, tags := range commits {
			syncableCommitTags = append(syncableCommitTags, newSyncableCommitTags(
				moduleIdentity,
				commit,
				slicesextended.MapToSlice(tags),
			))
		}
	}
	return syncableBranches, syncableCommitTags, nil
}

func (s *syncer) determineTaggedCommitsOnBranch(
	ctx context.Context,
	branch string,
	allTags map[git.Hash][]string,
) (map[git.Hash][]string, error) {
	allTagsHashesAsString := make(map[string][]string)
	for hash, tags := range allTags {
		allTagsHashesAsString[hash.Hex()] = tags
	}
	taggedCommitsOnBranch := make(map[git.Hash][]string)
	if err := s.repo.ForEachCommit(func(commit git.Commit) error {
		if tags, found := allTagsHashesAsString[commit.Hash().Hex()]; found {
			taggedCommitsOnBranch[commit.Hash()] = tags
		}
		return nil
	}, git.ForEachCommitWithBranchStartPoint(branch, git.ForEachCommitWithBranchStartPointWithRemote(s.gitRemoteName))); err != nil {
		return nil, fmt.Errorf("walk branch looking for tags to sync: %w", err)
	}
	return taggedCommitsOnBranch, nil
}

func (s *syncer) determineCommitsAndTagsToSyncForModuleBranch(
	ctx context.Context,
	moduleDir string,
	moduleIdentity bufmoduleref.ModuleIdentity,
	branch string,
) ([]git.Hash, error) {
	bsrBranchHead, err := s.handler.GetBranchHead(ctx, moduleIdentity, branch)
	if err != nil {
		return nil, err
	}
	if bsrBranchHead == nil {
		// Remote branch is empty, let's see if we can resume from another branch we intersect with
		latestVcsCommitInRemote, walkedCommits, err := s.walkBranchUntil(branch, func(commit git.Commit) (bool, error) {
			return s.handler.IsGitCommitSynced(ctx, moduleIdentity, commit.Hash())
		})
		if err != nil {
			return nil, err
		}
		if latestVcsCommitInRemote != nil {
			return walkedCommits, nil
		}
		// No intersection with any other synced branch: sync from the start.
		_, walkedCommits, err = s.walkBranchUntil(branch, func(commit git.Commit) (bool, error) { return false, nil })
		return walkedCommits, err
	}
	if protected, err := s.handler.IsProtectedBranch(ctx, moduleIdentity, branch); err != nil {
		return nil, err
	} else if !protected {
		if isSynced, err := s.handler.IsBranchSynced(ctx, moduleIdentity, branch); err != nil {
			return nil, err
		} else if !isSynced {
			// Don't both looking for any synced commits, there are non. Proceed straight to content-matching.
			return s.contentMatchOrHead(ctx, moduleDir, moduleIdentity, branch, bsrBranchHead)
		}
		latestVcsCommitInRemote, walkedCommits, err := s.walkBranchUntil(branch, func(commit git.Commit) (bool, error) {
			return s.handler.IsGitCommitSynced(ctx, moduleIdentity, commit.Hash())
		})
		if err != nil {
			return nil, err
		}
		if latestVcsCommitInRemote != nil {
			return walkedCommits, nil
		}
		s.logger.Warn("expected to find resume point for synced branch %q in the BSR, but didn't find one; onboarding branch again", zap.String("branch", branch))
	}
	if isSynced, err := s.handler.IsBranchSynced(ctx, moduleIdentity, branch); err != nil {
		return nil, err
	} else if !isSynced {
		return s.contentMatchOrHead(ctx, moduleDir, moduleIdentity, branch, bsrBranchHead)
	}
	if err := s.protectSyncedModuleBranch(ctx, moduleIdentity, branch); err != nil {
		return nil, err
	}
	latestVcsCommitInRemoteBranch, walkedCommits, err := s.walkBranchUntil(branch, func(commit git.Commit) (bool, error) {
		return s.handler.IsGitCommitSyncedToBranch(ctx, moduleIdentity, branch, commit.Hash())
	})
	if err != nil {
		return nil, err
	}
	if latestVcsCommitInRemoteBranch != nil {
		return walkedCommits, nil
	}
	return nil, errors.New("expected a synced commit to be found for a synced branch; did you rebase?")
}

func (s *syncer) contentMatchOrHead(
	ctx context.Context,
	moduleDir string,
	moduleIdentity bufmoduleref.ModuleIdentity,
	branch string,
	bsrCommitToMatch *registryv1alpha1.RepositoryCommit,
) ([]git.Hash, error) {
	var head git.Hash
	matched, walked, err := s.walkBranchUntil(branch, func(commit git.Commit) (bool, error) {
		// capture the branch head if don't match anything and need it
		if head == nil {
			head = commit.Hash()
		}
		// try to content match against the commit
		module, err := s.readModuleAt(ctx, commit, moduleDir)
		if err != nil {
			if errors.Is(err, errReadModuleInvalidModule) || errors.Is(err, errReadModuleInvalidModuleConfig) {
				// skip this commit
				return false, nil
			}
			return false, err
		}
		manifestBlob, err := bufcas.ManifestToBlob(module.FileSet().Manifest())
		if err != nil {
			return false, fmt.Errorf("manifest to blob: %w", err)
		}
		manifestDigest := manifestBlob.Digest().String()
		if manifestDigest == bsrCommitToMatch.ManifestDigest {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return nil, err
	}
	if matched != nil {
		s.logger.Debug(
			"content matched to commit",
			zap.String("gitHash", matched.Hex()),
			zap.String("bsrCommitID", bsrCommitToMatch.Id),
		)
		return walked, nil
	}
	// sync only the HEAD of the branch
	return []git.Hash{head}, nil
}

func (s *syncer) protectSyncedModuleBranch(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
	branch string,
) error {
	syncPoint, err := s.resolveSyncPoint(ctx, moduleIdentity, branch)
	if err != nil {
		return err
	}
	if syncPoint != nil {
		// Branch has never been synced, there is nothing to protected against.
		return nil
	}
	if containsSyncPoint, err := s.branchContainsCommit(branch, syncPoint); err != nil {
		return err
	} else if containsSyncPoint {
		// SyncPoint is up-to-date or behind us.
		return nil
	}
	// We don't know about the syncPoint. It may be in the future, if we have a stale copy of the
	// repository. Let's check if everything locally has been synced by checking if the branch HEAD
	// is synced.
	branchHead, err := s.repo.HEADCommit(git.HEADCommitWithBranch(branch))
	if err != nil {
		return fmt.Errorf("resolve branch %q head: %w", branch, err)
	}
	if headIsSynced, err := s.handler.IsGitCommitSyncedToBranch(ctx, moduleIdentity, branch, branchHead.Hash()); err != nil {
		return fmt.Errorf("check if branch %q head %q is synced: %w", branch, branchHead.Hash(), err)
	} else if headIsSynced {
		// Branch HEAD is synced, syncPoint is most likely ahead of us and we have a stale copy of
		// the repository. This is okay.
		return nil
	}
	return fmt.Errorf(
		"history on protected branch %q has diverged: remote sync point %q is unknown locally, branch HEAD %q is unknown remotely. Did you rebase?",
		branch,
		syncPoint,
		branchHead.Hash(),
	)
}

// walkBranchUntil walks a branch starting from HEAD, accumulating the commits visited until f evaluates to true.
// It returns the commit stopped at and all commits walked. If no commit was stopped at, it returns nil and all
// commits walked.
func (s *syncer) walkBranchUntil(branch string, f func(commit git.Commit) (bool, error)) (git.Hash, []git.Hash, error) {
	var (
		walked    []git.Hash
		stoppedAt git.Hash
	)
	if err := s.repo.ForEachCommit(
		func(commit git.Commit) error {
			walked = append(walked, commit.Hash())
			if shouldStop, err := f(commit); err != nil {
				return err
			} else if shouldStop {
				stoppedAt = commit.Hash()
				return git.ErrStopForEach
			}
			return nil
		},
		git.ForEachCommitWithBranchStartPoint(branch, git.ForEachCommitWithBranchStartPointWithRemote(s.gitRemoteName)),
	); err != nil {
		return nil, nil, err
	}
	return stoppedAt, walked, nil
}

func (s *syncer) branchContainsCommit(branch string, hash git.Hash) (bool, error) {
	found := false
	err := s.repo.ForEachCommit(
		func(commit git.Commit) error {
			if commit.Hash().Hex() == hash.Hex() {
				found = true
				return git.ErrStopForEach
			}
			return nil
		},
		git.ForEachCommitWithBranchStartPoint(branch, git.ForEachCommitWithBranchStartPointWithRemote(s.gitRemoteName)),
	)
	return found, err
}

func (s *syncer) logPlan(plan *executionPlan) {
	if !s.logger.Level().Enabled(zap.DebugLevel) {
		return
	}
	for _, branch := range plan.branchesToSync {
		var commitSHAs []string
		for _, commit := range branch.commitsToSync {
			commitSHAs = append(commitSHAs, commit.commit.Hash().Hex())
		}
		s.logger.Debug(
			"branch plan for module",
			zap.String("branch", branch.name),
			zap.String("moduleDir", branch.moduleDir),
			zap.String("moduleIdentity", branch.moduleIdentity.IdentityString()),
			zap.Strings("commitsToSync", commitSHAs),
		)
	}
	for _, commitTags := range plan.tagsToSync {
		s.logger.Debug(
			"tag plan for module",
			zap.Stringer("commit", commitTags.commit),
			zap.String("moduleIdentity", commitTags.moduleIdentity.IdentityString()),
			zap.Strings("tagsToSync", commitTags.tagsToSync),
		)
	}
}
