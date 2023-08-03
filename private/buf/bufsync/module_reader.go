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

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagegit"
)

// readModuleAt returns a module that has a name and builds correctly given a commit and a module
// directory.
func (s *syncer) readModuleAt(
	ctx context.Context,
	branch string,
	commit git.Commit,
	moduleDir string,
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
