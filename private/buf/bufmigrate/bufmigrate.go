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
	"io"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
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
		options ...MigrateOption,
	) error
}

func NewMigrator(
	logger *zap.Logger,
	dryRunWriter io.Writer,
	moduleKeyProvider bufmodule.ModuleKeyProvider,
	commitProvider bufmodule.CommitProvider,
) Migrator {
	return newMigrator(
		logger,
		dryRunWriter,
		moduleKeyProvider,
		commitProvider,
	)
}

// MigrateOption is a migrate option.
type MigrateOption func(*migrateOptions)

// MigrateAsDryRun print the summary of the changes to be made, without writing to the disk.
func MigrateAsDryRun() MigrateOption {
	return func(migrateOptions *migrateOptions) {
		migrateOptions.dryRun = true
	}
}

// MigrateAll uses bufconfig.WalkFileInfos to discover all known module, workspace, and buf.gen.yaml
// paths in the Bucket, and migrates them.
func MigrateAll(
	ctx context.Context,
	migrator Migrator,
	bucket storage.ReadWriteBucket,
	options ...MigrateOption,
) error {
	var workspaceDirPaths []string
	var moduleDirPaths []string
	var bufGenYAMLFilePaths []string
	if err := bufconfig.WalkFileInfos(
		ctx,
		bucket,
		func(path string, fileInfo bufconfig.FileInfo) error {
			dirPath := normalpath.Dir(path)
			fileType := fileInfo.FileType()
			fileVersion := fileInfo.FileVersion()
			switch fileType {
			case bufconfig.FileTypeBufYAML:
				switch fileVersion {
				case bufconfig.FileVersionV1Beta1, bufconfig.FileVersionV1:
					moduleDirPaths = append(moduleDirPaths, dirPath)
				case bufconfig.FileVersionV2:
					// ignore
				default:
					return syserror.Newf("unknown FileVersion: %v", fileVersion)
				}
			case bufconfig.FileTypeBufGenYAML:
				switch fileVersion {
				case bufconfig.FileVersionV1Beta1, bufconfig.FileVersionV1:
					bufGenYAMLFilePaths = append(bufGenYAMLFilePaths, path)
				case bufconfig.FileVersionV2:
					// ignore
				default:
					return syserror.Newf("unknown FileVersion: %v", fileVersion)
				}
			case bufconfig.FileTypeBufWorkYAML:
				switch fileVersion {
				case bufconfig.FileVersionV1Beta1, bufconfig.FileVersionV1:
					workspaceDirPaths = append(workspaceDirPaths, dirPath)
				case bufconfig.FileVersionV2:
					return syserror.Newf("invalid FileVersion for %q: %v", path, fileVersion)
				default:
					return syserror.Newf("unknown FileVersion: %v", fileVersion)
				}
			case bufconfig.FileTypeBufLock:
				// ignore
			default:
				return syserror.Newf("unknown FileType: %v", fileType)
			}
			return syserror.New("should never get here")
		},
	); err != nil {
		return err
	}
	return migrator.Migrate(ctx, bucket, workspaceDirPaths, moduleDirPaths, bufGenYAMLFilePaths, options...)
}
