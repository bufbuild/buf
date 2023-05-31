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

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagegit"
	"go.uber.org/zap"
)

type syncer struct {
	logger             *zap.Logger
	repo               git.Repository
	storageGitProvider storagegit.Provider
	errorHandler       ErrorHandler
	modulesToSync      []Module

	knownTagsByCommitHash map[string][]string
}

func newSyncer(
	logger *zap.Logger,
	repo git.Repository,
	storageGitProvider storagegit.Provider,
	errorHandler ErrorHandler,
	options ...SyncerOption,
) (Syncer, error) {
	s := &syncer{
		logger:             logger,
		repo:               repo,
		storageGitProvider: storageGitProvider,
	}
	for _, opt := range options {
		if err := opt(s); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *syncer) Sync(ctx context.Context, syncFunc SyncFunc) error {
	s.knownTagsByCommitHash = map[string][]string{}
	if err := s.repo.ForEachTag(func(tag string, commitHash git.Hash) error {
		s.knownTagsByCommitHash[commitHash.Hex()] = append(s.knownTagsByCommitHash[commitHash.Hex()], tag)
		return nil
	}); err != nil {
		return fmt.Errorf("process tags: %w", err)
	}
	// TODO: sync other branches
	for _, branch := range []string{s.repo.BaseBranch()} {
		// TODO: resume from last sync point
		if err := s.repo.ForEachCommit(branch, func(commit git.Commit) error {
			for _, module := range s.modulesToSync {
				if err := s.visitCommit(
					ctx,
					module,
					branch,
					commit,
					syncFunc,
				); err != nil {
					return fmt.Errorf("process commit %s: %w", commit.Hash().Hex(), err)
				}
			}
			return nil
		}); err != nil {
			return fmt.Errorf("process commits: %w", err)
		}
	}
	return nil
}

// visitCommit looks for the module in the commit, and if found tries to validate it.
// If it is valid, it invokes `syncFunc`.
//
// It does not return errors on invalid modules, but it will return any errors from
// `syncFunc` as those may be transient.
func (s *syncer) visitCommit(
	ctx context.Context,
	module Module,
	branch string,
	commit git.Commit,
	syncFunc SyncFunc,
) error {
	sourceBucket, err := s.storageGitProvider.NewReadBucket(
		commit.Tree(),
		storagegit.ReadBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return err
	}
	sourceBucket = storage.MapReadBucket(sourceBucket, storage.MapOnPrefix(module.Dir()))
	foundModule, err := bufconfig.ExistingConfigFilePath(ctx, sourceBucket)
	if err != nil {
		return err
	}
	if foundModule == "" {
		// We did not find a module. Carry on to the next commit.
		s.logger.Debug(
			"module not found, skipping commit",
			zap.String("commit", commit.Hash().String()),
			zap.String("module", module.Dir()),
		)
		return nil
	}
	sourceConfig, err := bufconfig.GetConfigForBucket(ctx, sourceBucket)
	if err != nil {
		return s.errorHandler.InvalidModuleConfig(module.Dir(), commit, err)
	}
	if sourceConfig.ModuleIdentity == nil {
		// Unnamed module. Carry on.
		s.logger.Debug(
			"unnamed module, skipping commit",
			zap.String("commit", commit.Hash().String()),
			zap.String("module", module.Dir()),
		)
		return nil
	}
	moduleIdentity := sourceConfig.ModuleIdentity
	if module.IdentityOverride() != nil {
		moduleIdentity = module.IdentityOverride()
	}
	builtModule, err := bufmodulebuild.BuildForBucket(
		ctx,
		sourceBucket,
		sourceConfig.Build,
	)
	if err != nil {
		return s.errorHandler.BuildFailure(module.Dir(), moduleIdentity, commit, err)
	}
	return syncFunc(
		ctx,
		newModuleCommit(
			moduleIdentity,
			builtModule.Bucket,
			commit,
			branch,
			s.knownTagsByCommitHash[commit.Hash().Hex()],
		),
	)
}
