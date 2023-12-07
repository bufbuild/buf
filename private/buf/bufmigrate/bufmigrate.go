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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"buf.build/gen/go/bufbuild/registry/connectrpc/go/buf/registry/module/v1beta1/modulev1beta1connect"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/buflint"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

// Migrate migrate buf configuration files.
func Migrate(
	ctx context.Context,
	storageProvider storageos.Provider,
	commitService modulev1beta1connect.CommitServiceClient,
	options ...MigrateOption,
) (retErr error) {
	// TODO: is it an error if a buf.yaml is v1beta1 but its buf.lock is v1? It's probably OK.
	// TODO: is it an error if some buf.yamls in a workspace are in v1beta1 while others are v1? It's probably also OK.
	migrateOptions := newMigrateOptions()
	for _, option := range options {
		option(migrateOptions)
	}
	if migrateOptions.bufWorkYAMLFilePath == "" && len(migrateOptions.bufYAMLFilePaths) == 0 {
		return errors.New("unimplmented")
	}
	// TODO: refine behavior for where to write this file
	destinationDir := "."
	// Alternatively, we can do the following: (but "." is probably better)
	//
	// if migrateOptions.bufWorkYAMLFilePath != "" {
	// 	destinationDir = filepath.Base(migrateOptions.bufWorkYAMLFilePath)
	// } else if len(migrateOptions.bufYAMLFilePaths) == 1 {
	// 	// TODO: maybe use "." (or maybe add --dest flag)
	// 	destinationDir = filepath.Base(migrateOptions.bufYAMLFilePaths[0])
	// } else {
	// 	destinationDir = "."
	// }
	var moduleConfigs []bufconfig.ModuleConfig
	var moduleDependencies []bufmodule.ModuleRef
	var depModuleKeys []bufmodule.ModuleKey
	// non-dependency modules names
	seenModuleNames := make(map[string]struct{})
	bufYAMLFilesFromWorkspace := make(map[string]struct{})
	bufYAMLFilesSpecified := make(map[string]struct{})
	var bufLockFiles []string
	if migrateOptions.bufWorkYAMLFilePath != "" {
		// We read the file from the exact path specified, instead of calling GetBufWorkYAMLFileForPrefix.
		// This is so that if the user passes something like buf.tmp.work.yaml, we can still find it.
		// Alternatively, we could say it's an invalid name and the only valid name is buf.work.yaml, and return an error.
		file, err := os.Open(migrateOptions.bufWorkYAMLFilePath)
		if err != nil {
			return err
		}
		bufWorkYAMLFile, err := bufconfig.ReadBufWorkYAMLFile(file)
		if err != nil {
			return err
		}
		if bufWorkYAMLFile.FileVersion() != bufconfig.FileVersionV1 {
			return fmt.Errorf("invalid version %v", bufWorkYAMLFile.FileVersion())
		}
		workspaceDirectory := filepath.Dir(migrateOptions.bufWorkYAMLFilePath)
		workspaceBucket, err := storageProvider.NewReadWriteBucket(
			workspaceDirectory,
			storageos.ReadWriteBucketWithSymlinksIfSupported(),
		)
		if err != nil {
			return err
		}
		for _, moduleDirectory := range bufWorkYAMLFile.DirPaths() {
			if err != nil {
				return err
			}
			bufYAMLFile, err := bufconfig.GetBufYAMLFileForPrefix(
				ctx,
				workspaceBucket,
				moduleDirectory,
			)
			if errors.Is(errors.Unwrap(err), fs.ErrNotExist) {
				workspaceDirectoryRelativeToDestination, err := filepath.Rel(workspaceDirectory, destinationDir)
				if err != nil {
					return err
				}
				moduleConfig, err := bufconfig.NewModuleConfig(
					filepath.Join(workspaceDirectoryRelativeToDestination, moduleDirectory),
					nil,
					nil,
					nil,
					nil,
				)
				if err != nil {
					return err
				}
				moduleConfigs = append(moduleConfigs, moduleConfig)
				// Assume there is no co-resident buf.lock
				continue
			}
			if err != nil {
				return err
			}
			// TODO: get file path properly
			bufYAMLFilePath := filepath.Join(workspaceDirectory, moduleDirectory, "buf.yaml")
			bufYAMLFilesFromWorkspace[bufYAMLFilePath] = struct{}{}
			for _, moduleConfig := range bufYAMLFile.ModuleConfigs() {
				// Not checking if module full name has been seen because it is guaranteed to be unique within the workspace.
				if moduleConfig.ModuleFullName() != nil {
					seenModuleNames[moduleConfig.ModuleFullName().String()] = struct{}{}
				}
			}
			moduleConfigsFromFile, moduleDependenciesFromFile, err := modulesAndDependenciesForBufYAMLFile(
				bufYAMLFile,
				bufYAMLFilePath,
				destinationDir,
			)
			if err != nil {
				return err
			}
			moduleConfigs = append(moduleConfigs, moduleConfigsFromFile...)
			moduleDependencies = append(moduleDependencies, moduleDependenciesFromFile...)
			bufLockFile, err := bufconfig.GetBufLockFileForPrefix(
				ctx,
				workspaceBucket,
				moduleDirectory,
			)
			if errors.Is(errors.Unwrap(err), fs.ErrNotExist) {
				continue
			}
			if err != nil {
				return err
			}
			// TODO: get file name properly
			bufLockFiles = append(bufLockFiles, filepath.Join(workspaceDirectory, moduleDirectory, "buf.lock"))
			switch bufLockFile.FileVersion() {
			case bufconfig.FileVersionV1Beta1, bufconfig.FileVersionV1:
				depModuleKeys = append(depModuleKeys, bufLockFile.DepModuleKeys()...)
			case bufconfig.FileVersionV2:
				// TODO: how to get full path, we have the prefix already, just need the file name
				return errors.New("file is already at v2")
			default:
				return syserror.Newf("unrecognized version: %v", bufLockFile.FileVersion())
			}
		}
		moduleDependencies = slicesext.Filter(
			moduleDependencies,
			func(moduleRef bufmodule.ModuleRef) bool {
				_, ok := seenModuleNames[moduleRef.ModuleFullName().String()]
				return !ok
			},
		)
	}
	bucket, err := storageProvider.NewReadWriteBucket(".", storageos.ReadWriteBucketWithSymlinksIfSupported())
	if err != nil {
		return err
	}
	for _, bufYAMLPath := range migrateOptions.bufYAMLFilePaths {
		if _, ok := bufYAMLFilesFromWorkspace[bufYAMLPath]; ok {
			return fmt.Errorf("%s is already discovered when migrating %s", bufYAMLPath, migrateOptions.bufWorkYAMLFilePath)
		}
		if _, ok := bufYAMLFilesSpecified[bufYAMLPath]; ok {
			return fmt.Errorf("%s is specified mutliple times", bufYAMLPath)
		}
		file, err := os.Open(bufYAMLPath)
		if err != nil {
			return err
		}
		moduleDirectory := filepath.Dir(bufYAMLPath)
		directoryForModuleConfig, err := filepath.Rel(destinationDir, moduleDirectory)
		if err != nil {
			return err
		}
		// TODO: close this file
		bufYAMLFile, err := bufconfig.ReadBufYAMLFile(file)
		if err != nil {
			return err
		}
		switch {
		case errors.Is(errors.Unwrap(err), fs.ErrNotExist):
			moduleConfig, err := bufconfig.NewModuleConfig(directoryForModuleConfig, nil, nil, nil, nil)
			if err != nil {
				return err
			}
			moduleConfigs = append(moduleConfigs, moduleConfig)
			// Assume there is no co-resident buf.lock
			continue
		case err != nil:
			return err
		}
		for _, moduleConfig := range bufYAMLFile.ModuleConfigs() {
			// Not checking if module full name has been seen because it is guaranteed to be unique within the workspace.
			if moduleConfig.ModuleFullName() != nil {
				seenModuleNames[moduleConfig.ModuleFullName().String()] = struct{}{}
			}
		}
		moduleConfigsFromFile, moduleDependenciesFromFile, err := modulesAndDependenciesForBufYAMLFile(
			bufYAMLFile,
			bufYAMLPath,
			destinationDir,
		)
		if err != nil {
			return err
		}
		moduleConfigs = append(moduleConfigs, moduleConfigsFromFile...)
		moduleDependencies = append(moduleDependencies, moduleDependenciesFromFile...)
		bufLockFile, err := bufconfig.GetBufLockFileForPrefix(
			ctx,
			bucket,
			moduleDirectory,
		)
		if errors.Is(errors.Unwrap(err), fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		bufLockFiles = append(bufLockFiles, filepath.Join(moduleDirectory, "buf.lock"))
		switch bufLockFile.FileVersion() {
		case bufconfig.FileVersionV1Beta1, bufconfig.FileVersionV1:
			depModuleKeys = append(depModuleKeys, bufLockFile.DepModuleKeys()...)
		case bufconfig.FileVersionV2:
			// TODO: how to get full path, we have the prefix already, but need the file name to print better errors
			return errors.New("file is already at v2")
		default:
			return syserror.Newf("unrecognized version: %v", bufLockFile.FileVersion())
		}
	}
	depModuleFullNameToModuleKeys := make(map[string][]bufmodule.ModuleKey)
	for _, depModuleKey := range depModuleKeys {
		depModuleFullName := depModuleKey.ModuleFullName().String()
		depModuleFullNameToModuleKeys[depModuleFullName] = append(depModuleFullNameToModuleKeys[depModuleFullName], depModuleKey)
	}
	// TODO: these are resolved arbitrarily right now, resolve them by commit time
	resolvedDepModuleKeys := make([]bufmodule.ModuleKey, 0, len(depModuleFullNameToModuleKeys))
	for _, depModuleKeys := range depModuleFullNameToModuleKeys {
		// TODO: actually resolve dependencies by time
		// The alternative is to build the workspace with tentative dependencies and
		// find the latest one that does not break. However, what if there are 3 dependencies
		// in question, each has 4 potential versions. We don't want to build 4*4*4 times in the worst case.
		resolvedDepModuleKeys = append(resolvedDepModuleKeys, depModuleKeys[0])
	}
	migratedBufYAMLFile, err := bufconfig.NewBufYAMLFile(
		bufconfig.FileVersionV2,
		moduleConfigs,
		moduleDependencies,
	)
	if err != nil {
		return err
	}
	migratedBufLockFile, err := bufconfig.NewBufLockFile(
		bufconfig.FileVersionV2,
		resolvedDepModuleKeys,
	)
	if err != nil {
		return err
	}
	writeBucket, err := storageProvider.NewReadWriteBucket(
		destinationDir,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return err
	}
	var filesToDelete []string
	if migrateOptions.bufWorkYAMLFilePath != "" {
		filesToDelete = append(filesToDelete, migrateOptions.bufWorkYAMLFilePath)
	}
	filesToDelete = append(filesToDelete, slicesext.MapKeysToSlice(bufYAMLFilesFromWorkspace)...)
	filesToDelete = append(filesToDelete, slicesext.MapKeysToSlice(bufYAMLFilesSpecified)...)
	filesToDelete = append(filesToDelete, bufLockFiles...)
	sort.Strings(filesToDelete)
	if migrateOptions.dryRun {
		var bufYAMLBuffer bytes.Buffer
		if err := bufconfig.WriteBufYAMLFile(&bufYAMLBuffer, migratedBufYAMLFile); err != nil {
			return err
		}
		var bufLockBuffer bytes.Buffer
		if err := bufconfig.WriteBufLockFile(&bufLockBuffer, migratedBufLockFile); err != nil {
			return err
		}
		fmt.Fprintf(
			migrateOptions.dryRunWriter,
			`in an actual run, these files will be removed:
%s

these files will be written:
%s:
%s
%s:
%s
`,
			strings.Join(filesToDelete, "\n"),
			// TODO: find a way to get file name
			filepath.Join(destinationDir, "buf.yaml"),
			bufYAMLBuffer.String(),
			// TODO: find a way to get file name
			filepath.Join(destinationDir, "buf.lock"),
			bufLockBuffer.String(),
		)
		return nil
	}
	if err := bufconfig.PutBufYAMLFileForPrefix(
		ctx,
		writeBucket,
		".",
		migratedBufYAMLFile,
	); err != nil {
		return err
	}
	if err := bufconfig.PutBufLockFileForPrefix(
		ctx,
		writeBucket,
		".",
		migratedBufLockFile,
	); err != nil {
		return err
	}
	// TODO: delete files to delete
	return nil
}

func modulesAndDependenciesForBufYAMLFile(
	bufYAMLFile bufconfig.BufYAMLFile,
	bufYAMLFilePath string,
	targetFileDirectory string,
) (
	[]bufconfig.ModuleConfig,
	[]bufmodule.ModuleRef,
	error,
) {
	var moduleConfigs []bufconfig.ModuleConfig
	var moduleDependencies []bufmodule.ModuleRef
	switch bufYAMLFile.FileVersion() {
	case bufconfig.FileVersionV1Beta1:
		// TODO: this has multiple configs
		// TODO: whether something needs to be done about root to exclude mapping
		if len(bufYAMLFile.ModuleConfigs()) != 1 {
			return nil, nil, syserror.Newf("expect exactly 1 module config from buf yaml, got %d", len(bufYAMLFile.ModuleConfigs()))
		}
		moduleConfig := bufYAMLFile.ModuleConfigs()[0]
		for root, excludes := range moduleConfig.RootToExcludes() {
			lintConfig := moduleConfig.LintConfig()
			// TODO: this list expands to individual rules, we could process
			// this list and make it shorter by substituting some rules with
			// a single group, if all rules in that group are present.
			lintRules, err := buflint.RulesForConfig(lintConfig)
			if err != nil {
				return nil, nil, err
			}
			lintConfigForRoot := bufconfig.NewLintConfig(
				bufconfig.NewCheckConfig(
					bufconfig.FileVersionV2,
					slicesext.Map(lintRules, func(rule bufcheck.Rule) string { return rule.ID() }),
					lintConfig.ExceptIDsAndCategories(),
					lintConfig.IgnorePaths(),
					lintConfig.IgnoreIDOrCategoryToPaths(),
				),
				lintConfig.EnumZeroValueSuffix(),
				lintConfig.RPCAllowSameRequestResponse(),
				lintConfig.RPCAllowGoogleProtobufEmptyRequests(),
				lintConfig.RPCAllowGoogleProtobufEmptyResponses(),
				lintConfig.ServiceSuffix(),
				lintConfig.AllowCommentIgnores(),
			)
			breakingConfig := moduleConfig.BreakingConfig()
			breakingConfigForRoot := bufconfig.NewBreakingConfig(
				bufconfig.NewCheckConfig(
					bufconfig.FileVersionV2,
					breakingConfig.UseIDsAndCategories(),
					breakingConfig.ExceptIDsAndCategories(),
					breakingConfig.IgnorePaths(),
					breakingConfig.IgnoreIDOrCategoryToPaths(),
				),
				breakingConfig.IgnoreUnstablePackages(),
			)
			rootRelativePath, err := filepath.Rel(targetFileDirectory, filepath.Dir(bufYAMLFilePath))
			if err != nil {
				return nil, nil, err
			}
			configForRoot, err := bufconfig.NewModuleConfig(
				filepath.Join(rootRelativePath, root),
				// TODO: if this is not nil, there will be multiple modules in the buf.yaml v2 with the same name.
				moduleConfig.ModuleFullName(),
				map[string][]string{".": excludes},
				// TODO: filter these by root
				// TODO: possibly convert them to be relative to workspace
				lintConfigForRoot,
				breakingConfigForRoot,
			)
			if err != nil {
				return nil, nil, err
			}
			moduleConfigs = append(moduleConfigs, configForRoot)
		}
	case bufconfig.FileVersionV1:
		if len(bufYAMLFile.ModuleConfigs()) != 1 {
			return nil, nil, syserror.Newf("expect exactly 1 module config from buf yaml, got %d", len(bufYAMLFile.ModuleConfigs()))
		}
		moduleConfig := bufYAMLFile.ModuleConfigs()[0]
		// use the same lint and breaking config, except that they are v2.
		lintConfig := moduleConfig.LintConfig()
		lintConfig = bufconfig.NewLintConfig(
			bufconfig.NewCheckConfig(
				bufconfig.FileVersionV2,
				lintConfig.UseIDsAndCategories(),
				lintConfig.ExceptIDsAndCategories(),
				lintConfig.IgnorePaths(),
				lintConfig.IgnoreIDOrCategoryToPaths(),
			),
			lintConfig.EnumZeroValueSuffix(),
			lintConfig.RPCAllowSameRequestResponse(),
			lintConfig.RPCAllowGoogleProtobufEmptyRequests(),
			lintConfig.RPCAllowGoogleProtobufEmptyResponses(),
			lintConfig.ServiceSuffix(),
			lintConfig.AllowCommentIgnores(),
		)
		breakingConfig := moduleConfig.BreakingConfig()
		breakingConfig = bufconfig.NewBreakingConfig(
			bufconfig.NewCheckConfig(
				bufconfig.FileVersionV2,
				breakingConfig.UseIDsAndCategories(),
				breakingConfig.ExceptIDsAndCategories(),
				breakingConfig.IgnorePaths(),
				breakingConfig.IgnoreIDOrCategoryToPaths(),
			),
			breakingConfig.IgnoreUnstablePackages(),
		)
		moduleRelativePath, err := filepath.Rel(targetFileDirectory, filepath.Dir(bufYAMLFilePath))
		if err != nil {
			return nil, nil, err
		}
		moduleConfig, err = bufconfig.NewModuleConfig(
			moduleRelativePath,
			moduleConfig.ModuleFullName(),
			moduleConfig.RootToExcludes(),
			lintConfig,
			breakingConfig,
		)
		if err != nil {
			return nil, nil, err
		}
		moduleConfigs = append(moduleConfigs, moduleConfig)
		moduleDependencies = append(moduleDependencies, bufYAMLFile.ConfiguredDepModuleRefs()...)
	case bufconfig.FileVersionV2:
		// TODO: how to get full path, we have the prefix already, just need the file name
		return nil, nil, errors.New("already at v2")
	}
	return moduleConfigs, moduleDependencies, nil
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
	// TODO: Looking at IsLocal's doc, it seems to does what we want: relative and does not jump context.
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
