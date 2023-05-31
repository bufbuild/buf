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
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagegit"
	"go.uber.org/zap"
)

// ErrorHandler handles errors reported by the Syncer. If a non-nil
// error is returned by the handler, sync will abort in a partially-synced
// state.
type ErrorHandler interface {
	// InvalidModuleConfig is invoked by Syncer upon encountering a module
	// with an invalid module config.
	InvalidModuleConfig(
		module string,
		commit git.Commit,
		err error,
	) error
	// BuildFailure is invoked by Syncer upon encountering a module that fails
	// build.
	BuildFailure(
		module string,
		moduleIdentity bufmoduleref.ModuleIdentity,
		commit git.Commit,
		err error,
	) error
}

// Module is a module that will be synced by Syncer.
type Module interface {
	// Dir is the path to the module relative to the repository root.
	Dir() string
	// IdentityOverride is an optional module identity override. If empty,
	// the identity specified in the module config file will be used.
	//
	// Unnamed modules will not have their identity overridden, as they are
	// not pushable.
	IdentityOverride() bufmoduleref.ModuleIdentity
}

// NewModule constructs a new module that can be synced with a Syncer.
func NewModule(dir string, identityOverride bufmoduleref.ModuleIdentity) (Module, error) {
	path, err := normalpath.NormalizeAndValidate(dir)
	if err != nil {
		return nil, err
	}
	return newSyncableModule(
		path,
		identityOverride,
	), nil
}

// Syncer syncs a modules in a git.Repository.
type Syncer interface {
	// Sync syncs the repository using the provided SyncFunc. It processes
	// commits in reverse topological order, loads any configured named
	// modules, extracts any Git metadata for that commit, and invokes
	// SyncFunc with a ModuleCommit.
	//
	// Only commits/branches belonging to the remote named 'origin' are
	// processed. All tags are processed.
	Sync(context.Context, SyncFunc) error
}

// NewSyncer creates a new Syncer.
func NewSyncer(
	logger *zap.Logger,
	repo git.Repository,
	storageGitProvider storagegit.Provider,
	errorHandler ErrorHandler,
	options ...SyncerOption,
) (Syncer, error) {
	return newSyncer(
		logger,
		repo,
		storageGitProvider,
		errorHandler,
		options...,
	)
}

// SyncerOption configures the creation of a new Syncer.
type SyncerOption func(*syncer) error

// SyncerWithModule configures a Syncer to sync the specified module.
//
// This option can be provided multiple times to sync multiple distinct modules.
func SyncerWithModule(module Module) SyncerOption {
	return func(s *syncer) error {
		for _, existingModule := range s.modulesToSync {
			if existingModule.Dir() != module.Dir() {
				continue
			}
			if module.IdentityOverride() == nil && existingModule.IdentityOverride() == nil {
				return fmt.Errorf("duplicate module %s", module.Dir())
			}
			if module.IdentityOverride() != nil &&
				existingModule.IdentityOverride() != nil &&
				module.IdentityOverride().IdentityString() == existingModule.IdentityOverride().IdentityString() {
				return fmt.Errorf("duplicate module %s:%s", module.Dir(), module.IdentityOverride().IdentityString())
			}
		}
		s.modulesToSync = append(s.modulesToSync, module)
		return nil
	}
}

// SyncFunc is invoked by Syncer to process a sync point.
type SyncFunc func(ctx context.Context, commit ModuleCommit) error

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
