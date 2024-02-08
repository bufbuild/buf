// Copyright 2020-2024 Buf Technologies, Inc.
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
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagegit"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

var (
	// errReadModuleInvalidModule is returned by `readModuleAt` when the module fails to build.
	errReadModuleInvalidModule = errors.New("invalid module")
	// errReadModuleInvalidModule is returned by `readModuleAt` when the module's config is invalid.
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
		handler:                              newCachedHandler(handler),
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
	plan, err := s.Plan(ctx)
	if err != nil {
		return fmt.Errorf("determine plan: %w", err)
	}
	plan.log(s.logger)
	if plan.Nop() {
		s.logger.Warn("nothing to sync")
		return nil
	}
	if err := s.executePlan(ctx, plan); err != nil {
		return fmt.Errorf("execute plan: %w", err)
	}
	return nil
}

func (s *syncer) Plan(ctx context.Context) (ExecutionPlan, error) {
	branchesToSync, tagsToSync, err := s.determineEverythingToSync(ctx)
	if err != nil {
		return nil, err
	}
	return newExecutionPlan(
		s.sortedModulesDirsForSync,
		branchesToSync,
		tagsToSync,
	)
}

// executePlan executes an ExecutionPlan using the Handler.
func (s *syncer) executePlan(ctx context.Context, plan ExecutionPlan) error {
	// ModuleBranches must be synced prior to ModuleTags. This ensures that all
	// tagged commits in ModuleTags have been synced.
	for _, moduleBranch := range plan.ModuleBranchesToSync() {
		if err := s.handler.SyncModuleBranch(ctx, moduleBranch); err != nil {
			return fmt.Errorf(
				"sync module %s:%s branch %q: %w",
				moduleBranch.Directory(),
				moduleBranch.TargetModuleIdentity(),
				moduleBranch.BranchName(),
				err,
			)
		}
	}
	for _, moduleTags := range plan.ModuleTagsToSync() {
		if err := s.handler.SyncModuleTags(ctx, moduleTags); err != nil {
			return fmt.Errorf(
				"sync module %s tags: %w",
				moduleTags.TargetModuleIdentity(),
				err,
			)
		}
	}
	return nil
}

// readModule returns a module that has a name and builds correctly given a commit and a module directory.
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

// determineEverythingToSync determines the full set of module branches to sync and module tags to sync.
// The returned ModulesBranches and ModuleTags are not ordered in any particular way and must be ordered
// by the caller.
func (s *syncer) determineEverythingToSync(ctx context.Context) ([]ModuleBranch, []ModuleTags, error) {
	// Determine the branches to sync. The order of branches here doesn't matter, everything
	// will be ordered at the end.
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
	// Load all tagged commits ahead of time.
	commitHashToTags := make(map[string][]string)
	if err := s.repo.ForEachTag(func(tag string, commitHash git.Hash) error {
		commitHashToTags[commitHash.Hex()] = append(commitHashToTags[commitHash.Hex()], tag)
		return nil
	}); err != nil {
		return nil, nil, err
	}
	var (
		moduleBranches []ModuleBranch
		// All tagged commits for a module identity. The commits in this set are either already synced or
		// will be synced in at least one ModuleBranch.
		taggedCommitsToSyncForModuleIdentity = make(map[bufmoduleref.ModuleIdentity]map[git.Commit][]string)
	)
	// Walk branches and collect ModuleBranches and tagged commits to sync. The order of branches here
	// doesn't matter, everything will be ordered at the end.
	for _, branch := range branchesToSync {
		headCommit, err := s.repo.HEADCommit(
			git.HEADCommitWithBranch(branch),
			git.HEADCommitWithRemote(s.gitRemoteName),
		)
		if err != nil {
			return nil, nil, fmt.Errorf("reading head commit for branch %s: %w", branch, err)
		}
		moduleDirsFoundForModuleIdentity := make(map[string][]string) // moduleIdentity:[]moduleDir
		// Walk all module dirs for a branch. The order of module dirs here doesn't matter, everything will
		// be ordered at the end.
		for moduleDir, identityOverride := range s.modulesDirsToIdentityOverrideForSync {
			// Determine the module identity to sync this moduleDir with, in this branch.
			// If the user provided an override, prefer it always. Otherwise use the identity of the
			// module at HEAD.
			builtModuleAtHEAD, readErr := s.readModuleAt(ctx, headCommit, moduleDir)
			if readErr != nil {
				// Module read failed. Either this is a transient error, or the module is invalid, either
				// way, the reason doesn't really matter, syncing this branch should fail sync. There may
				// be other commits behind HEAD that are valid to sync, but for now nothing on this branch
				// will be synced.
				return nil, nil, fmt.Errorf("reading module %q at branch %q HEAD: %w", moduleDir, branch, err)
			} else if builtModuleAtHEAD == nil {
				// Module does not exist at HEAD. Skip syncing the branch. In theory there may be other
				// commits with modules on them behind HEAD, but the repository is no longer syncing a
				// module from this directory, so take the same position.
				s.logger.Debug(
					"no module on HEAD, skipping branch",
					zap.String("branch", branch),
					zap.String("moduleDir", moduleDir),
				)
				continue
			}
			var targetModuleIdentity bufmoduleref.ModuleIdentity
			if identityOverride == nil {
				targetModuleIdentity = builtModuleAtHEAD.ModuleIdentity()
			} else {
				targetModuleIdentity = identityOverride
			}
			moduleDirsFoundForModuleIdentity[targetModuleIdentity.IdentityString()] = append(
				moduleDirsFoundForModuleIdentity[targetModuleIdentity.IdentityString()],
				moduleDir,
			)
			// Determine commits to _visit_ since the last sync. Not all commits that are visitable
			// are syncable, as any of those commits may contain an invalid module.
			commitsToVisit, err := s.determineCommitsToVisitForModuleBranch(
				ctx,
				moduleDir,
				targetModuleIdentity,
				branch,
			)
			if err != nil {
				return nil, nil, err
			}
			// Determine commits to sync. All of these commits will be synced.
			var commitsToSync []ModuleCommit
			for i, commitToVisit := range commitsToVisit {
				commit, err := s.repo.Objects().Commit(commitToVisit)
				if err != nil {
					return nil, nil, fmt.Errorf("read commit %q: %w", commitsToVisit[i], err)
				}
				module, err := s.readModuleAt(ctx, commit, moduleDir)
				if err != nil {
					if errors.Is(err, errReadModuleInvalidModule) || errors.Is(err, errReadModuleInvalidModuleConfig) {
						// If this is not the last commit to sync for this branch (i.e., is not the HEAD of
						// the branch), it is safe to skip this commit and simply log a warning to the user.
						if i != len(commitsToVisit)-1 {
							s.logger.Warn(
								"module read failed",
								zap.Stringer("commit", commitToVisit),
								zap.String("moduleDir", moduleDir),
								zap.String("branch", branch),
								zap.Error(err),
							)
							continue
						}
					}
					// We know the last commit to sync for this branch (i.e., the branch HEAD) was already
					// read and is valid. Therefore this is a transient error. Fail immediately and let the
					// user try again.
					return nil, nil, fmt.Errorf(
						"module %q read failed at commit %q on branch %q: %w",
						targetModuleIdentity.IdentityString(),
						commit.Hash().Hex(),
						branch,
						err,
					)
				}
				if module == nil {
					s.logger.Debug(
						"no module, skipping commit",
						zap.Stringer("commit", commitToVisit),
						zap.Error(err),
					)
					continue
				}
				// Check that the module identity of the module at this commit matches the target module
				// identity. If it doesn't, this is still okay if the user configured an override for this
				// moduleDir. If the identities don't match, and user didn't configure an override for this
				// moduleDir, warn and skip the commit.
				if identityOverride == nil && module.ModuleIdentity().IdentityString() != targetModuleIdentity.IdentityString() {
					s.logger.Warn(
						"module identity at commit has a different module identity than branch HEAD; skipping commit",
						zap.Stringer("commit", commitToVisit),
						zap.String("moduleDir", moduleDir),
						zap.String("branch", branch),
						zap.String("moduleIdentityAtHead", targetModuleIdentity.IdentityString()),
						zap.String("moduleIdentityAtCommit", module.ModuleIdentity().IdentityString()),
					)
					continue
				}
				commitsToSync = append(commitsToSync, newModuleCommit(
					commit,
					commitHashToTags[commit.Hash().Hex()],
					func(ctx context.Context) (storage.ReadBucket, error) {
						// Don't retain the module read above to avoid storing a lot of modules in memory
						// at once. Instead, read it again and expect to be able to read it without error.
						// This prioritizes memory pressure over time pressure.
						module, err := s.readModuleAt(ctx, commit, moduleDir)
						if err != nil {
							return nil, fmt.Errorf("expected to read module: %w", err)
						}
						if module == nil {
							return nil, errors.New("expected bucket to be non-nil, but wasn't")
						}
						return module.Bucket, nil
					},
				))
				// If the commit is tagged, collect this tagged commit because we _will_ be syncing it.
				if len(commitHashToTags[commit.Hash().Hex()]) > 0 {
					if _, ok := taggedCommitsToSyncForModuleIdentity[targetModuleIdentity]; !ok {
						taggedCommitsToSyncForModuleIdentity[targetModuleIdentity] = make(map[git.Commit][]string)
					}
					taggedCommitsToSyncForModuleIdentity[targetModuleIdentity][commit] = commitHashToTags[commit.Hash().Hex()]
				}
			}
			moduleBranch := newModuleBranch(
				branch,
				moduleDir,
				targetModuleIdentity,
				commitsToSync,
			)
			moduleBranches = append(moduleBranches, moduleBranch)
			// Collect all previously synced tagged commits for this branch.
			taggedCommitsOnBranch, err := s.determineSyncedTaggedCommitsReachableFrom(
				ctx,
				targetModuleIdentity,
				// Start walking back from the first commit we'll sync.
				// There may not be a commit to sync, but there is always at least 1 commit
				// to visit because there is always at least one commit on a branch.
				commitsToVisit[0],
				commitHashToTags,
			)
			if err != nil {
				return nil, nil, fmt.Errorf("determine tagged commits on branch: %w", err)
			}
			for commit, tags := range taggedCommitsOnBranch {
				if _, ok := taggedCommitsToSyncForModuleIdentity[targetModuleIdentity]; !ok {
					taggedCommitsToSyncForModuleIdentity[targetModuleIdentity] = make(map[git.Commit][]string)
				}
				taggedCommitsToSyncForModuleIdentity[targetModuleIdentity][commit] = tags
			}
		}
		// If we encountered any duplicated target module identities, fail immediately as that
		// will wreak havoc on everything.
		var duplicatedIdentitiesErr error
		for moduleIdentity, moduleDirs := range moduleDirsFoundForModuleIdentity {
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
	// Convert all collected tagged commits into ModuleTags.
	var moduleTags []ModuleTags
	for targetModuleIdentity, commitsToTags := range taggedCommitsToSyncForModuleIdentity {
		var taggedCommits []TaggedCommit
		for commit, tags := range commitsToTags {
			taggedCommits = append(taggedCommits, newTaggedCommit(commit, tags))
		}
		moduleTags = append(moduleTags, newModuleTags(
			targetModuleIdentity,
			taggedCommits,
		))
	}
	return moduleBranches, moduleTags, nil
}

// determineSyncedTaggedCommitsReachableFrom walks back from a starting commit, collecting all commits that
// are tagged and synced to the target module identity.
func (s *syncer) determineSyncedTaggedCommitsReachableFrom(
	ctx context.Context,
	targetModuleIdentity bufmoduleref.ModuleIdentity,
	startingGitHash git.Hash,
	commitHashToTags map[string][]string,
) (map[git.Commit][]string, error) {
	taggedCommitsOnBranch := make(map[git.Commit][]string)
	if err := s.repo.ForEachCommit(
		func(commit git.Commit) error {
			if commit.Hash().Hex() == startingGitHash.Hex() {
				// skip starting commit
				return nil
			}
			if tags, commitIsTagged := commitHashToTags[commit.Hash().Hex()]; commitIsTagged {
				if synced, err := s.handler.IsGitCommitSynced(ctx, targetModuleIdentity, commit.Hash()); err != nil {
					return err
				} else if synced {
					taggedCommitsOnBranch[commit] = tags
				} else {
					s.logger.Debug(
						"skipping tags because the commit is not synced",
						zap.String("commit", commit.Hash().Hex()),
						zap.String("targetModuleIdentity", targetModuleIdentity.IdentityString()),
						zap.Strings("tags", tags),
					)
				}
			}
			return nil
		},
		git.ForEachCommitWithHashStartPoint(startingGitHash.Hex()),
	); err != nil {
		return nil, fmt.Errorf("walk branch looking for tags to sync: %w", err)
	}
	return taggedCommitsOnBranch, nil
}

// determineCommitsToVisitForModuleBranch determines the set of commits to visit for a particular
// branch for a module in this run of Syncer#Sync.
//
// This logic can be complicated, so here's a table of expected behavior:
//
//	CONDITION                        RESUME FROM (-> means fallback)
//	new remote branch:
//	  unprotected:                   any synced commit from any branch -> START of branch
//	  protected:
//	    not release branch:          START of branch
//	    release lineage:
//	      empty:                     START of branch
//	      not empty:                 content match(HEAD of Release) -> HEAD of branch
//	existing remote branch:
//	  not previously synced:         content match -> HEAD of branch
//	  previously synced:
//	    protected:                   protect branch && any synced commit from branch -> error
//	    unprotected:                 any synced commit from any branch -> content match -> HEAD of branch
func (s *syncer) determineCommitsToVisitForModuleBranch(
	ctx context.Context,
	moduleDir string,
	moduleIdentity bufmoduleref.ModuleIdentity,
	branch string,
) ([]git.Hash, error) {
	protected, err := s.handler.IsProtectedBranch(ctx, moduleIdentity, branch)
	if err != nil {
		return nil, err
	}
	bsrBranchHead, err := s.handler.GetBranchHead(ctx, moduleIdentity, branch)
	if err != nil {
		return nil, err
	}
	if bsrBranchHead == nil {
		// The remote branch is empty.
		if !protected {
			// The remote branch is empty and unprotected. Backfill history from any synced place, or sync
			// the whole branch.
			found, walkedCommits, err := s.visitAllCommitsOnBranchUntil(branch, func(commit git.Commit) (bool, error) {
				return s.handler.IsGitCommitSynced(ctx, moduleIdentity, commit.Hash())
			})
			if err != nil {
				return nil, err
			}
			if found != nil {
				return walkedCommits, nil
			}
			return s.visitAllCommitsOnBranch(branch)
		}
		// The remote branch is empty but protected. How to respond to this is based on whether this branch
		// represents the Release branch or not.
		if isReleaseBranch, err := s.handler.IsReleaseBranch(ctx, moduleIdentity, branch); err != nil {
			return nil, err
		} else if !isReleaseBranch {
			// The remote branch is empty but protected and does not represent the Release branch. Don't
			// trust the history of any other branch to backfill from, so sync the whole branch.
			return s.visitAllCommitsOnBranch(branch)
		}
		// The remote branch is empty but protected. It represents the Release branch, so any commit
		// synced from here is going to be released immediately.
		// As a special case, attempt to content-match to the _Release_ head if there is one. If there isn't,
		// fallback to syncing the whole branch.
		bsrReleasedHead, err := s.handler.GetReleaseHead(ctx, moduleIdentity)
		if err != nil {
			return nil, err
		}
		if bsrReleasedHead != nil {
			// This branch is the Release branch, and there is something to content-match with. This is the
			// typical case of onboarding the Release branch.
			return s.visitAllCommitsOnBranchUntilContentMatchOrHead(ctx, moduleDir, moduleIdentity, branch, bsrReleasedHead)
		}
		// This branch is the Release branch, but there is no released commit. This is the typical case
		// of a new module.
		return s.visitAllCommitsOnBranch(branch)
	}
	// The remote branch is non-empty.
	if isSynced, err := s.handler.IsBranchSynced(ctx, moduleIdentity, branch); err != nil {
		return nil, err
	} else if !isSynced {
		// The remote branch is non-empty, but unsynced. This is the typical case of onboarding
		// non-protected branches.
		return s.visitAllCommitsOnBranchUntilContentMatchOrHead(ctx, moduleDir, moduleIdentity, branch, bsrBranchHead)
	}
	// The remote branch is non-empty and it has been synced at least once.
	if protected {
		// The remote branch is non-empty but was synced and is protected. First protect the branch, and
		// then resume from the last synced commit for this branch specifically.
		// If there is no such commit, this is an error. protectSyncedModuleBranch should catch this
		// but return an error just in case.
		if err := s.protectSyncedModuleBranch(ctx, moduleIdentity, branch); err != nil {
			return nil, err
		}
		latestVcsCommitInRemoteBranch, walkedCommits, err := s.visitAllCommitsOnBranchUntil(branch, func(commit git.Commit) (bool, error) {
			return s.handler.IsGitCommitSyncedToBranch(ctx, moduleIdentity, branch, commit.Hash())
		})
		if err != nil {
			return nil, err
		}
		if latestVcsCommitInRemoteBranch != nil {
			return walkedCommits, nil
		}
		// Should not get here unless there's a bug in protectSyncedModuleBranch.
		return nil, errors.New("expected a synced commit to be found for a synced branch; did you rebase?")
	}
	// The remote branch is non-empty and non-protected but was synced. Backfill history from any synced commit.
	// If a synced commit cannot be found, attempt to recover by content matching again, and finally just
	// syncing the HEAD of the branch.
	latestVcsCommitInRemote, walkedCommits, err := s.visitAllCommitsOnBranchUntil(branch, func(commit git.Commit) (bool, error) {
		return s.handler.IsGitCommitSynced(ctx, moduleIdentity, commit.Hash())
	})
	if err != nil {
		return nil, err
	}
	if latestVcsCommitInRemote != nil {
		return walkedCommits, nil
	}
	s.logger.Warn(
		"expected to find resume point for synced branch for module, but didn't find one; onboarding branch again",
		zap.String("moduleDir", moduleDir),
		zap.String("moduleIdentity", moduleIdentity.IdentityString()),
		zap.String("branch", branch),
	)
	return s.visitAllCommitsOnBranchUntilContentMatchOrHead(ctx, moduleDir, moduleIdentity, branch, bsrBranchHead)
}

// visitAllCommitsOnBranchUntilContentMatchOrHead walks a branch at a module dir finding the first commit that
// contains a module whose content matches the content of the BSR commit provided. If no match is found, the
// HEAD of the branch is returned, regardless of whether its content matches.
//
// This comparison is via ManifestDigest only.
func (s *syncer) visitAllCommitsOnBranchUntilContentMatchOrHead(
	ctx context.Context,
	moduleDir string,
	moduleIdentity bufmoduleref.ModuleIdentity,
	branch string,
	bsrCommitToMatch *registryv1alpha1.RepositoryCommit,
) ([]git.Hash, error) {
	digestToMatch, err := bufcas.ParseDigest(bsrCommitToMatch.ManifestDigest)
	if err != nil {
		return nil, fmt.Errorf("parse digest: %w", err)
	}
	var head git.Hash
	matched, walked, err := s.visitAllCommitsOnBranchUntil(branch, func(commit git.Commit) (bool, error) {
		// Capture the branch head; if no content-match is found, it will be needed.
		if head == nil {
			head = commit.Hash()
		}
		// Try to content match against this commit.
		module, err := s.readModuleAt(ctx, commit, moduleDir)
		if err != nil {
			if errors.Is(err, errReadModuleInvalidModule) || errors.Is(err, errReadModuleInvalidModuleConfig) {
				// Bad module, skip this commit.
				return false, nil
			}
			return false, err
		}
		if module == nil {
			// No module, skip this commit.
			return false, nil
		}
		// TODO(BSR-3065):  The module returned from `readModuleAt` was not constructed from a fileset.
		// As a result, we have to manually construct it from the bucket ourselves.
		fileSet, err := bufcas.NewFileSetForBucket(ctx, module.Bucket)
		if err != nil {
			return false, err
		}
		manifestBlob, err := bufcas.ManifestToBlob(fileSet.Manifest())
		if err != nil {
			return false, fmt.Errorf("manifest to blob: %w", err)
		}
		if bufcas.DigestEqual(digestToMatch, manifestBlob.Digest()) {
			// Content-match found; select this commit.
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
			zap.String("moduleDir", moduleDir),
			zap.String("moduleIdentity", moduleIdentity.IdentityString()),
			zap.String("branch", branch),
			zap.String("gitHash", matched.Hex()),
			zap.String("bsrCommitID", bsrCommitToMatch.Id),
		)
		return walked, nil
	}
	// Select only the HEAD of the branch.
	return []git.Hash{head}, nil
}

// protectSyncedModuleBranch protects a synced branch according to the following rules:
//   - If a protected branch is _behind_ the BSR, this is allowed.
//   - If a protected branch is _ahead of_ the BSR, this is also allowed.
//   - If a protected branch is both _ahead of_ and _behind_ the BSR, this is disallowed.
//
// This must only be called for protected branches.
func (s *syncer) protectSyncedModuleBranch(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
	protectedBranch string,
) error {
	syncPoint, err := s.handler.ResolveSyncPoint(ctx, moduleIdentity, protectedBranch)
	if err != nil {
		return fmt.Errorf("resolve sync point for module %s: %w", moduleIdentity.IdentityString(), err)
	}
	if syncPoint == nil {
		// Branch has never been synced, there is nothing to protected against.
		return nil
	}
	if containsSyncPoint, err := s.branchContainsCommit(protectedBranch, syncPoint); err != nil {
		return err
	} else if containsSyncPoint {
		// SyncPoint is up-to-date or behind us.
		return nil
	}
	// The syncPoint is unknown to us. It may be in the future, if the copy of the repository locally
	// is stale. Check if everything locally has been synced by checking if the branch HEAD is synced.
	branchHead, err := s.repo.HEADCommit(git.HEADCommitWithBranch(protectedBranch))
	if err != nil {
		return fmt.Errorf("resolve branch %q head: %w", protectedBranch, err)
	}
	if headIsSyncedToRemoteBranch, err := s.handler.IsGitCommitSyncedToBranch(ctx, moduleIdentity, protectedBranch, branchHead.Hash()); err != nil {
		return fmt.Errorf("check if branch %q head %q is synced: %w", protectedBranch, branchHead.Hash(), err)
	} else if headIsSyncedToRemoteBranch {
		// Branch HEAD is synced, syncPoint is most likely ahead of the local branch and the local
		// repository is stale. This is okay.
		return nil
	}
	return fmt.Errorf(
		"history on protected branch %q has diverged: remote branch sync point %q is not in the local branch, and branch HEAD %q is not in the remote branch. Did you rebase?",
		protectedBranch,
		syncPoint,
		branchHead.Hash(),
	)
}

// visitAllCommitsOnBranch returns all commits on a branch.
func (s *syncer) visitAllCommitsOnBranch(branch string) ([]git.Hash, error) {
	_, walkedCommits, err := s.visitAllCommitsOnBranchUntil(branch, func(commit git.Commit) (bool, error) { return false, nil })
	return walkedCommits, err
}

// visitAllCommitsOnBranchUntil walks a branch starting from HEAD, accumulating the commits visited until f evaluates
// to true, including the commit for which f evaluated to true. It returns the commit stopped at and all commits walked,
// ancestors before descendants.
// If no commit was stopped at, it returns nil and all commits walked.
func (s *syncer) visitAllCommitsOnBranchUntil(branch string, f func(commit git.Commit) (bool, error)) (git.Hash, []git.Hash, error) {
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
	for i, j := 0, len(walked)-1; i < j; i, j = i+1, j-1 {
		walked[i], walked[j] = walked[j], walked[i]
	}
	return stoppedAt, walked, nil
}

// branchContainsCommit returns true if the branch contains the commit, and false otherwise.
func (s *syncer) branchContainsCommit(branch string, hash git.Hash) (bool, error) {
	var found bool
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
