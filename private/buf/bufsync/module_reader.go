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

// moduleAt retrieves a built named module at a passed commit and directory, if any. If it fails
// during the process it invokes the syncer error handler, and returns its error if any.
func (s *syncer) builtNamedModuleAt(
	ctx context.Context,
	branch string,
	commit git.Commit,
	moduleDir string,
) (*bufmodulebuild.BuiltModule, error) {
	// in case there is an error to handle, it will have the same branch, commit, and module dir. The
	// actual `err` and `code` is populated in case-by-case basis before sending it to the handler.
	errToHandle := ReadModuleError{
		branch:    branch,
		commit:    commit.Hash().Hex(),
		moduleDir: moduleDir,
	}
	commitBucket, err := s.storageGitProvider.NewReadBucket(commit.Tree(), storagegit.ReadBucketWithSymlinksIfSupported())
	if err != nil {
		errToHandle.err = fmt.Errorf("new read bucket: %w", err)
		return nil, s.errorHandler(errToHandle)
	}
	moduleBucket := storage.MapReadBucket(commitBucket, storage.MapOnPrefix(moduleDir))
	foundModule, err := bufconfig.ExistingConfigFilePath(ctx, moduleBucket)
	if err != nil {
		errToHandle.err = fmt.Errorf("looking for an existing config file path: %w", err)
		return nil, s.errorHandler(errToHandle)
	}
	if foundModule == "" {
		errToHandle.err = errors.New("module not found in commit and module dir")
		errToHandle.code = ReadModuleErrorCodeModuleNotFound
		return nil, s.errorHandler(errToHandle)
	}
	sourceConfig, err := bufconfig.GetConfigForBucket(ctx, moduleBucket)
	if err != nil {
		errToHandle.err = fmt.Errorf("invalid module config: %w", err)
		errToHandle.code = ReadModuleErrorCodeInvalidModuleConfig
		return nil, s.errorHandler(errToHandle)
	}
	if sourceConfig.ModuleIdentity == nil {
		errToHandle.err = errors.New("found module does not have a name")
		errToHandle.code = ReadModuleErrorCodeUnnamedModule
		return nil, s.errorHandler(errToHandle)
	}
	builtModule, err := bufmodulebuild.NewModuleBucketBuilder().BuildForBucket(
		ctx,
		moduleBucket,
		sourceConfig.Build,
	)
	if err != nil {
		return nil, &buildModuleError{err: err}
	}
	return builtModule, nil
}

func (e *invalidModuleConfigError) Error() string {
	return
}

func (e *buildModuleError) Error() string {
	return "build module: " + e.err.Error()
}
