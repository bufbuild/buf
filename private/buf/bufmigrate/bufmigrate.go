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
	"fmt"
	"io"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"go.uber.org/zap"
)

// Migrator migrates buf configuration files.
type Migrator interface {
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
	Migrate(
		ctx context.Context,
		bucket storage.ReadWriteBucket,
		workspaceDirPaths []string,
		moduleDirPaths []string,
		bufGenYAMLFilePaths []string,
	) error
	// Diff runs migrate, but produces a diff instead of writing the migration.
	Diff(
		ctx context.Context,
		bucket storage.ReadBucket,
		writer io.Writer,
		workspaceDirPaths []string,
		moduleDirPaths []string,
		bufGenYAMLFilePaths []string,
	) error
}

func NewMigrator(
	logger *zap.Logger,
	runner command.Runner,
	moduleKeyProvider bufmodule.ModuleKeyProvider,
	commitProvider bufmodule.CommitProvider,
) Migrator {
	return newMigrator(
		logger,
		runner,
		moduleKeyProvider,
		commitProvider,
	)
}

// MigrateAll uses bufconfig.WalkFileInfos to discover all known module, workspace, and buf.gen.yaml
// paths in the Bucket, and migrates them.
//
// ignoreDirPaths should be normalized and relative to the root of the bucket, if specified.
func MigrateAll(
	ctx context.Context,
	migrator Migrator,
	bucket storage.ReadWriteBucket,
	ignoreDirPaths []string,
) error {
	workspaceDirPaths, moduleDirPaths, bufGenYAMLFilePaths, err := getWorkspaceModuleBufGenYAMLPaths(ctx, bucket, ignoreDirPaths)
	if err != nil {
		return err
	}
	return migrator.Migrate(ctx, bucket, workspaceDirPaths, moduleDirPaths, bufGenYAMLFilePaths)
}

// DiffAll uses bufconfig.WalkFileInfos to discover all known module, workspace, and buf.gen.yaml
// paths in the Bucket, and diffs them.
//
// ignoreDirPaths should be normalized and relative to the root of the bucket, if specified.
func DiffAll(
	ctx context.Context,
	migrator Migrator,
	bucket storage.ReadBucket,
	writer io.Writer,
	ignoreDirPaths []string,
) error {
	workspaceDirPaths, moduleDirPaths, bufGenYAMLFilePaths, err := getWorkspaceModuleBufGenYAMLPaths(ctx, bucket, ignoreDirPaths)
	if err != nil {
		return err
	}
	return migrator.Diff(ctx, bucket, writer, workspaceDirPaths, moduleDirPaths, bufGenYAMLFilePaths)
}

// *** PRIVATE ***

func getWorkspaceModuleBufGenYAMLPaths(
	ctx context.Context,
	bucket storage.ReadBucket,
	ignoreDirPaths []string,
) (workspaceDirPaths []string, moduleDirPaths []string, bufGenYAMLFilePaths []string, retErr error) {
	ignoreDirPathMap := make(map[string]struct{}, len(ignoreDirPaths))
	for _, ignoreDirPath := range ignoreDirPaths {
		ignoreDirPath, err := normalpath.NormalizeAndValidate(ignoreDirPath)
		if err != nil {
			return nil, nil, nil, err
		}
		ignoreDirPathMap[ignoreDirPath] = struct{}{}
	}
	var dirPath string
	if err := bufconfig.WalkFileInfos(
		ctx,
		bucket,
		func(path string, fileInfo bufconfig.FileInfo) error {
			dirPath = normalpath.Dir(path)
			if len(ignoreDirPathMap) > 0 {
				if normalpath.MapHasEqualOrContainingPath(ignoreDirPathMap, dirPath, normalpath.Relative) {
					return nil
				}
			}
			fileType := fileInfo.FileType()
			fileVersion := fileInfo.FileVersion()
			switch fileType {
			case bufconfig.FileTypeBufYAML:
				switch fileVersion {
				case bufconfig.FileVersionV1Beta1, bufconfig.FileVersionV1:
					moduleDirPaths = append(moduleDirPaths, dirPath)
					return nil
				case bufconfig.FileVersionV2:
					// ignore
					return nil
				default:
					return syserror.Newf("unknown FileVersion: %v", fileVersion)
				}
			case bufconfig.FileTypeBufGenYAML:
				switch fileVersion {
				case bufconfig.FileVersionV1Beta1, bufconfig.FileVersionV1:
					bufGenYAMLFilePaths = append(bufGenYAMLFilePaths, path)
					return nil
				case bufconfig.FileVersionV2:
					// ignore
					return nil
				default:
					return syserror.Newf("unknown FileVersion: %v", fileVersion)
				}
			case bufconfig.FileTypeBufWorkYAML:
				switch fileVersion {
				case bufconfig.FileVersionV1Beta1, bufconfig.FileVersionV1:
					workspaceDirPaths = append(workspaceDirPaths, dirPath)
					return nil
				case bufconfig.FileVersionV2:
					return syserror.Newf("invalid FileVersion for %q: %v", path, fileVersion)
				default:
					return syserror.Newf("unknown FileVersion: %v", fileVersion)
				}
			case bufconfig.FileTypeBufLock:
				// ignore
				return nil
			default:
				return syserror.Newf("unknown FileType: %v", fileType)
			}
		},
	); err != nil {
		return nil, nil, nil, fmt.Errorf("unable to parse %q: %w", dirPath, err)
	}
	return workspaceDirPaths, moduleDirPaths, bufGenYAMLFilePaths, nil
}
