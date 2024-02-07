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

package bufmigrate

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
)

// Migrate migrates buf configuration files.
//
// A buf.yaml v2 is written if any workspace directory or module directory is
// specified. The modules directories in the buf.yaml v2 will contain:
//
//   - directories at moduleDirPaths
//   - directories pointed to by buf.work.yamls at workspaceDirPaths
//
// More specifically:
//
//   - If a workspace is specified, then all of its module directories are also migrated,
//     regardless whether these module directories are specified in moduleDirPaths. Same
//     behavior with multiple workspaces. For example, if workspace foo has directories
//     bar and baz, then specifying foo, foo + bar and foo + bar + baz are the same.
//   - If a workspace is specfied, and modules not from this workspace are specified, the
//     buf.yaml will contain all directories from the workspace, as well as the module
//     directories specified.
//   - If only module directories are specified, then the buf.yaml will contain exactly
//     these directories.
//   - If a module specified is within some workspace not from workspaceDirPaths, we migrate
//     the module directory only (updating/deciding on this behavior is still a todo).
//   - If only one workspace directory is specified and no module directory is specified,
//     the buf.yaml will be written at <workspace directory>/buf.yaml. Otherwise, it will
//     be written at ./buf.yaml.
//
// Each generation template will be overwritten by a file in v2.
func Migrate(
	ctx context.Context,
	messageWriter io.Writer,
	storageProvider storageos.Provider,
	// TODO: This code should be reworked to use bufmodule.CommitProvider and bufmodule.ModuleKeyProvider.
	clientProvider bufapi.ClientProvider,
	commitProvider bufmodule.CommitProvider,
	workspaceDirPaths []string,
	moduleDirPaths []string,
	generateTemplatePaths []string,
	options ...MigrateOption,
) (retErr error) {
	migrateOptions := newMigrateOptions()
	for _, option := range options {
		option(migrateOptions)
	}
	if len(workspaceDirPaths) == 0 && len(moduleDirPaths) == 0 && len(generateTemplatePaths) == 0 {
		return errors.New("no directory or file specified")
	}
	var err error
	// Directories cannot jump context because in the migrated buf.yaml v2, each
	// directory path cannot jump context. I.e. it's not valid to have `- directory: ..`
	// in a buf.yaml v2.
	workspaceDirPaths, err = slicesext.MapError(
		workspaceDirPaths,
		func(workspaceDirPath string) (string, error) {
			if _, err := normalpath.NormalizeAndValidate(workspaceDirPath); err != nil {
				return "", fmt.Errorf("%s is not a relative path", workspaceDirPath)
			}
			return filepath.Clean(workspaceDirPath), nil
		},
	)
	if err != nil {
		return err
	}
	moduleDirPaths, err = slicesext.MapError(
		moduleDirPaths,
		func(moduleDirPath string) (string, error) {
			if _, err := normalpath.NormalizeAndValidate(moduleDirPath); err != nil {
				return "", fmt.Errorf("%s is not a relative path", moduleDirPath)
			}
			return filepath.Clean(moduleDirPath), nil
		},
	)
	if err != nil {
		return err
	}
	generateTemplatePaths = slicesext.Map(generateTemplatePaths, filepath.Clean)
	bucket, err := storageProvider.NewReadWriteBucket(
		".",
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return err
	}
	destionationDirectory := "."
	if len(workspaceDirPaths) == 1 && len(moduleDirPaths) == 0 {
		destionationDirectory = workspaceDirPaths[0]
	}
	migrator := newMigrator(
		messageWriter,
		clientProvider,
		commitProvider,
		bucket,
		destionationDirectory,
	)
	for _, workspaceDirPath := range workspaceDirPaths {
		if err := migrator.addWorkspaceDirectory(
			ctx,
			workspaceDirPath,
		); err != nil {
			return err
		}
	}
	for _, bufYAMLPath := range moduleDirPaths {
		// TODO: read upwards to make sure it's not in a workspace.
		// i.e. for ./foo/bar/buf.yaml, check none of "./foo", ".", "../", "../..", and etc. is a workspace.
		// The logic for this is in getMapPathAndSubDirPath from buffetch/internal
		if err := migrator.addModuleDirectory(
			ctx,
			bufYAMLPath,
		); err != nil {
			return err
		}
	}
	for _, bufGenYAMLPath := range generateTemplatePaths {
		if err := migrator.addBufGenYAML(bufGenYAMLPath); err != nil {
			return err
		}
	}
	if migrateOptions.dryRun {
		return migrator.migrateAsDryRun(ctx)
	}
	return migrator.migrate(ctx)
}

// MigrateOption is a migrate option.
type MigrateOption func(*migrateOptions)

// MigrateAsDryRun print the summary of the changes to be made, without writing to the disk.
func MigrateAsDryRun() MigrateOption {
	return func(migrateOptions *migrateOptions) {
		migrateOptions.dryRun = true
	}
}

/// *** PRIVATE ***

type migrateOptions struct {
	dryRun bool
}

func newMigrateOptions() *migrateOptions {
	return &migrateOptions{}
}
