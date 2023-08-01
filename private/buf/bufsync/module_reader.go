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

type invalidModuleConfigError struct {
	err error
}

type buildModuleError struct {
	err error
}

var (
	errModuleNotFound = errors.New("module not found in commit and module dir")
	errUnnamedModule  = errors.New("found module does not have a name")
)

// moduleAt retrieves a built named module at a passed commit and directory, if any.
func (s *syncer) builtNamedModuleAt(
	ctx context.Context,
	commit git.Commit,
	moduleDir string,
) (*bufmodulebuild.BuiltModule, error) {
	commitBucket, err := s.storageGitProvider.NewReadBucket(
		commit.Tree(),
		storagegit.ReadBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return nil, fmt.Errorf("new read bucket: %w", err)
	}
	moduleBucket := storage.MapReadBucket(commitBucket, storage.MapOnPrefix(moduleDir))
	foundModule, err := bufconfig.ExistingConfigFilePath(ctx, moduleBucket)
	if err != nil {
		return nil, fmt.Errorf("looking for an existing config file path: %w", err)
	}
	if foundModule == "" {
		return nil, errModuleNotFound
	}
	sourceConfig, err := bufconfig.GetConfigForBucket(ctx, moduleBucket)
	if err != nil {
		return nil, &invalidModuleConfigError{err: err}
	}
	if sourceConfig.ModuleIdentity == nil {
		return nil, errUnnamedModule
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
	return "invalid module config: " + e.err.Error()
}

func (e *buildModuleError) Error() string {
	return "build module: " + e.err.Error()
}
