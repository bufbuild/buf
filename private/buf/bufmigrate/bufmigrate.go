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
	"fmt"
	"io"
	"path/filepath"

	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"go.uber.org/zap"
)

// Migrate migrate buf configuration files.
func Migrate(
	ctx context.Context,
	logger *zap.Logger,
	storageProvider storageos.Provider,
	clientProvider bufapi.ClientProvider,
	options ...MigrateOption,
) (retErr error) {
	migrateOptions := newMigrateOptions()
	for _, option := range options {
		option(migrateOptions)
	}
	// TODO: check buf.gen.yaml as well when it's added
	if migrateOptions.bufWorkYAMLFilePath == "" && len(migrateOptions.bufYAMLFilePaths) == 0 {
		return errors.New("no ")
	}
	bucket, err := storageProvider.NewReadWriteBucket(
		".",
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return err
	}
	migrator := newMigrator(
		logger,
		clientProvider,
		bucket,
		".",
	)
	if migrateOptions.bufWorkYAMLFilePath != "" {
		// TODO: should we allow multiple workspaces?
		if err := migrator.addWorkspaceForBufWorkYAML(
			ctx,
			migrateOptions.bufWorkYAMLFilePath,
		); err != nil {
			return err
		}
	}
	for _, bufYAMLPath := range migrateOptions.bufYAMLFilePaths {
		// TODO: read upwards to make sure it's not in a workspace
		// i.e. for ./foo/bar/buf.yaml, check none of "./foo", ".", "../", "../..", and etc. is a workspace.
		if err := migrator.addModuleDirectoryForBufYAML(
			ctx,
			bufYAMLPath,
		); err != nil {
			return err
		}
	}
	if migrateOptions.dryRun {
		return migrator.migrateAsDryRun(ctx, migrateOptions.dryRunWriter)
	}
	return migrator.migrate(ctx)
}

// MigrateOption is a migrate option.
type MigrateOption func(*migrateOptions)

// MigrateAsDryRun write to the writer the summary of the changes to be made, without writing to the disk.
func MigrateAsDryRun(writer io.Writer) MigrateOption {
	return func(migrateOptions *migrateOptions) {
		migrateOptions.dryRun = true
		migrateOptions.dryRunWriter = writer
	}
}

// MigrateBufWorkYAMLFile migrates a buf.work.yaml.
func MigrateBufWorkYAMLFile(path string) (MigrateOption, error) {
	// TODO: Looking at IsLocal's doc, it seems to validate for what we want: relative and does not jump context.
	if !filepath.IsLocal(path) {
		return nil, fmt.Errorf("%s is not a relative path", path)
	}
	return func(migrateOptions *migrateOptions) {
		migrateOptions.bufWorkYAMLFilePath = filepath.Clean(path)
	}, nil
}

// MigrateBufYAMLFile migrates buf.yaml files.
func MigrateBufYAMLFile(paths []string) (MigrateOption, error) {
	for _, path := range paths {
		if !filepath.IsLocal(path) {
			return nil, fmt.Errorf("%s is not a relative path", path)
		}
	}
	return func(migrateOptions *migrateOptions) {
		migrateOptions.bufYAMLFilePaths = slicesext.Map(paths, filepath.Clean)
	}, nil
}

/// *** PRIVATE ***

type migrateOptions struct {
	dryRun              bool
	dryRunWriter        io.Writer
	bufWorkYAMLFilePath string
	bufYAMLFilePaths    []string
}

func newMigrateOptions() *migrateOptions {
	return &migrateOptions{}
}
