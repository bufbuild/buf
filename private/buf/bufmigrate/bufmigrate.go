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

package bufmigrate

import (
	"context"
	"errors"
	"io"

	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
)

// Migrate migrates buf configuration files.
//
// TODO: document behavior with different examples of module and workspace directories, ie what happens
// when I specify a module directory that is within a workspace directory I always specify, what happens
// if I don't specify the module directory, what if I only specify some module directories in a
// workspace directory, etc.
func Migrate(
	ctx context.Context,
	messageWriter io.Writer,
	storageProvider storageos.Provider,
	clientProvider bufapi.ClientProvider,
	// TODO: use these values
	workspaceDirPaths []string,
	moduleDirPaths []string,
	generateTemplatePaths []string,
	options ...MigrateOption,
) (retErr error) {
	migrateOptions := newMigrateOptions()
	for _, option := range options {
		option(migrateOptions)
	}
	if migrateOptions.workspaceDirectory == "" && len(migrateOptions.moduleDirectories) == 0 && len(migrateOptions.bufGenYAMLPaths) == 0 {
		return errors.New("no directory or file specified")
	}
	bucket, err := storageProvider.NewReadWriteBucket(
		".",
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return err
	}
	destionationDirectory := "."
	if migrateOptions.workspaceDirectory != "" && len(migrateOptions.moduleDirectories) == 0 {
		destionationDirectory = migrateOptions.workspaceDirectory
	}
	migrator := newMigrator(
		messageWriter,
		clientProvider,
		bucket,
		destionationDirectory,
	)
	if migrateOptions.workspaceDirectory != "" {
		// TODO: should we allow multiple workspaces?
		if err := migrator.addWorkspaceDirectory(
			ctx,
			migrateOptions.workspaceDirectory,
		); err != nil {
			return err
		}
	}
	for _, bufYAMLPath := range migrateOptions.moduleDirectories {
		// TODO: read upwards to make sure it's not in a workspace
		// i.e. for ./foo/bar/buf.yaml, check none of "./foo", ".", "../", "../..", and etc. is a workspace.
		if err := migrator.addModuleDirectory(
			ctx,
			bufYAMLPath,
		); err != nil {
			return err
		}
	}
	for _, bufGenYAMLPath := range migrateOptions.bufGenYAMLPaths {
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

// MigrateAsDryRun write to the writer the summary of the changes to be made,
// without writing to the disk.
func MigrateAsDryRun() MigrateOption {
	return func(migrateOptions *migrateOptions) {
		migrateOptions.dryRun = true
	}
}

// TODO: move this validaton inside Migrate

//// MigrateWorkspaceDirectory migrates buf.work.yaml, and all buf.yamls in directories
//// pointed to by this workspace, as well as their co-resident buf.locks.
//func MigrateWorkspaceDirectory(directory string) (MigrateOption, error) {
//// TODO: Looking at IsLocal's doc, it seems to validate for what we want: relative and does not jump context.
//if !filepath.IsLocal(directory) {
//return nil, fmt.Errorf("%s is not a relative path", directory)
//}
//return func(migrateOptions *migrateOptions) {
//migrateOptions.workspaceDirectory = filepath.Clean(directory)
//}, nil
//}

//// MigrateModuleDirectories migrates buf.yamls buf.locks in directories.
//func MigrateModuleDirectories(directories []string) (MigrateOption, error) {
//for _, path := range directories {
//if !filepath.IsLocal(path) {
//return nil, fmt.Errorf("%s is not a relative path", path)
//}
//}
//return func(migrateOptions *migrateOptions) {
//migrateOptions.moduleDirectories = slicesext.Map(directories, filepath.Clean)
//}, nil
//}

//// MigrateGenerationTemplates migrates buf.gen.yamls.
//func MigrateGenerationTemplates(paths []string) MigrateOption {
//return func(migrateOptions *migrateOptions) {
//migrateOptions.bufGenYAMLPaths = slicesext.Map(paths, filepath.Clean)
//}
//}

/// *** PRIVATE ***

type migrateOptions struct {
	dryRun bool
	//workspaceDirectory string
	//moduleDirectories  []string
	//bufGenYAMLPaths    []string
}

func newMigrateOptions() *migrateOptions {
	return &migrateOptions{}
}
