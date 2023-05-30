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

	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagegit"
	"go.uber.org/zap"
)

// Syncer syncs a modules in a git.Repository.
type Syncer interface {
	// Sync syncs the repository using the provided PushFunc. It processes
	// commits in reverse topological order, loads any configured named
	// modules, extracts any Git metadata for that commit, and invokes
	// PushFunc with a ModuleCommit.
	//
	// Only commits/branches belonging to the remote named 'origin' are
	// processed. All tags are processed.
	Sync(context.Context, PushFunc) error
}

// NewSyncer creates a new Syncer.
func NewSyncer(
	logger *zap.Logger,
	repo git.Repository,
	storageGitProvider storagegit.Provider,
	options ...SyncerOption,
) (Syncer, error) {
	return newSyncer(
		logger,
		repo,
		storageGitProvider,
		options...,
	)
}

// SyncerOption configures the creation of a new Syncer.
type SyncerOption func(*syncer) error

// SyncerWithModule configures a Syncer to sync the specified module. The module
// identity override is optional.
//
// This option can be provided multiple times to sync multiple distinct modules.
func SyncerWithModule(dir string, identityOverride bufmoduleref.ModuleIdentity) SyncerOption {
	return func(s *syncer) error {
		for _, existingModule := range s.modulesToSync {
			if existingModule.dir != dir {
				continue
			}
			if identityOverride == nil && existingModule.identityOverride == nil {
				return fmt.Errorf("duplicate module %s", dir)
			}
			if identityOverride != nil &&
				existingModule.identityOverride != nil &&
				identityOverride.IdentityString() == existingModule.identityOverride.IdentityString() {
				return fmt.Errorf("duplicate module %s:%s", dir, identityOverride.IdentityString())
			}
		}
		s.modulesToSync = append(s.modulesToSync, newSyncableModule(
			dir,
			identityOverride,
		))
		return nil
	}
}

// PushFunc is invoked by Syncer to process a sync point.
type PushFunc func(ctx context.Context, commit ModuleCommit) error

// ModuleCommit is a module at a particular commit.
type ModuleCommit interface {
	// Identity is the identity of the module, accounting for any configured override.
	Identity() bufmoduleref.ModuleIdentity
	// Bucket is the bucket for the module.
	Bucket() storage.ReadBucket
	// Commit is the commit that the module is sourced from.
	Commit() git.Commit
	// Branch is the git branch that this module is sourced from.
	Branch() string
	// Tags are the git tags associated with Commit.
	Tags() []string
}
