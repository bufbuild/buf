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
	modulesToSync      []syncableModule

	knownTags map[string][]string
}

func newSyncer(
	logger *zap.Logger,
	repo git.Repository,
	storageGitProvider storagegit.Provider,
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

func (s *syncer) Sync(ctx context.Context, pushFunc PushFunc) error {
	s.knownTags = map[string][]string{}
	if err := s.repo.ForEachTag(func(tag string, commitHash git.Hash) error {
		s.knownTags[commitHash.Hex()] = append(s.knownTags[commitHash.Hex()], tag)
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
					pushFunc,
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

func (s *syncer) visitCommit(
	ctx context.Context,
	module syncableModule,
	branch string,
	commit git.Commit,
	pushFunc PushFunc,
) error {
	sourceBucket, err := s.storageGitProvider.NewReadBucket(
		commit.Tree(),
		storagegit.ReadBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return err
	}
	sourceBucket = storage.MapReadBucket(sourceBucket, storage.MapOnPrefix(module.dir))
	foundModule, err := bufconfig.ExistingConfigFilePath(ctx, sourceBucket)
	if err != nil {
		return err
	}
	if foundModule == "" {
		// We did not find a module. Carry on to the next commit.
		return nil
	}
	sourceConfig, err := bufconfig.GetConfigForBucket(ctx, sourceBucket)
	if err != nil {
		// We found a module but the module config is invalid. We can warn on this
		// and carry on. Note that because of resumption, we will typically only come
		// across this commit once, we will not log this warning again.
		s.logger.Warn(
			"invalid module",
			zap.String("commit", commit.Hash().String()),
			zap.String("module", module.dir),
			zap.Error(err),
		)
		return nil
	}
	if sourceConfig.ModuleIdentity == nil {
		// Unnamed module. Carry on.
		return nil
	}
	moduleIdentity := sourceConfig.ModuleIdentity
	if module.identityOverride != nil {
		moduleIdentity = module.identityOverride
	}
	builtModule, err := bufmodulebuild.BuildForBucket(
		ctx,
		sourceBucket,
		sourceConfig.Build,
	)
	if err != nil {
		// We failed to build the module. We can warn on this
		// and carry on. Note that because of resumption, we will typically only come
		// across this commit once, we will not log this warning again.
		s.logger.Warn(
			"invalid module",
			zap.String("commit", commit.Hash().String()),
			zap.String("module", module.dir),
			zap.Error(err),
		)
		return nil
	}
	return pushFunc(
		ctx,
		newModuleCommit(
			moduleIdentity,
			builtModule.Bucket,
			commit,
			branch,
			s.knownTags[commit.Hash().Hex()],
		),
	)
}
