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

	"buf.build/gen/go/bufbuild/registry/connectrpc/go/buf/registry/module/v1beta1/modulev1beta1connect"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
)

// Migrate migrate buf configuration files.
func Migrate(
	ctx context.Context,
	storageProvider storageos.Provider,
	commitService modulev1beta1connect.CommitServiceClient,
	options ...MigrateOption,
) (retErr error) {
	migrateOptions := newMigrateOptions()
	for _, option := range options {
		option(migrateOptions)
	}
	if migrateOptions.bufWorkYAMLFilePath == "" && len(migrateOptions.bufYAMLFilePaths) == 0 {
		return errors.New("unimplmented")
	}
	// TODO: decide on behavior for where to write this file
	destinationDir := "."
	// Alternatively, we could do the following: (but "." is probably better, since we want users to
	// have buf.yaml v2 at their repository root and they are likely running this command from there)
	//
	// if migrateOptions.bufWorkYAMLFilePath != "" {
	// 	destinationDir = filepath.Base(migrateOptions.bufWorkYAMLFilePath)
	// } else if len(migrateOptions.bufYAMLFilePaths) == 1 {
	// 	// TODO: maybe use "." (or maybe add --dest flag)
	// 	destinationDir = filepath.Base(migrateOptions.bufYAMLFilePaths[0])
	// } else {
	// 	destinationDir = "."
	// }
	bucket, err := storageProvider.NewReadWriteBucket(".", storageos.ReadWriteBucketWithSymlinksIfSupported())
	if err != nil {
		return err
	}
	migrator := newMigrator(
		bucket,
		destinationDir,
	)
	if migrateOptions.bufWorkYAMLFilePath != "" {
		if err := migrator.addWorkspace(ctx, filepath.Dir(migrateOptions.bufWorkYAMLFilePath)); err != nil {
			return err
		}
	}
	for _, bufYAMLPath := range migrateOptions.bufYAMLFilePaths {
		// TODO: read upwards to make sure it's not in a workspace
		// i.e. for ./foo/bar/buf.yaml, check none of "./foo", ".", "../", "../..", and etc. is a workspace.
		if err := migrator.addModule(
			ctx,
			filepath.Dir(bufYAMLPath),
		); err != nil {
			return err
		}
	}
	if migrateOptions.dryRun {
		return migrator.migrateAsDryRun(migrateOptions.dryRunWriter)
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

// MigrateBufWorkYAMLFile migrates a v1 buf.work.yaml.
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

type migrateOptions struct {
	dryRun              bool
	dryRunWriter        io.Writer
	bufWorkYAMLFilePath string
	bufYAMLFilePaths    []string
}

func newMigrateOptions() *migrateOptions {
	return &migrateOptions{}
}
