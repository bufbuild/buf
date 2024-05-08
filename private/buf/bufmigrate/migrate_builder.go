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

// Generated. DO NOT EDIT.

package bufmigrate

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/gofrs/uuid/v5"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type migrateBuilder struct {
	logger             *zap.Logger
	commitProvider     bufmodule.CommitProvider
	bucket             storage.ReadBucket
	destinationDirPath string

	addedBufGenYAMLFilePaths map[string]struct{}
	addedWorkspaceDirPaths   map[string]struct{}
	addedModuleDirPaths      map[string]struct{}

	moduleConfigs                    []bufconfig.ModuleConfig
	configuredDepModuleRefs          []bufmodule.ModuleRef
	hasSeenBufLockFile               bool
	depModuleKeys                    []bufmodule.ModuleKey
	pathToMigratedBufGenYAMLFile     map[string]bufconfig.BufGenYAMLFile
	moduleFullNameStringToParentPath map[string]string
	pathsToDelete                    map[string]struct{}
}

func newMigrateBuilder(
	logger *zap.Logger,
	commitProvider bufmodule.CommitProvider,
	bucket storage.ReadBucket,
	destinationDirPath string,
) *migrateBuilder {
	return &migrateBuilder{
		logger:                           logger,
		commitProvider:                   commitProvider,
		bucket:                           bucket,
		destinationDirPath:               destinationDirPath,
		addedBufGenYAMLFilePaths:         make(map[string]struct{}),
		addedWorkspaceDirPaths:           make(map[string]struct{}),
		addedModuleDirPaths:              make(map[string]struct{}),
		pathToMigratedBufGenYAMLFile:     make(map[string]bufconfig.BufGenYAMLFile),
		moduleFullNameStringToParentPath: make(map[string]string),
		pathsToDelete:                    make(map[string]struct{}),
	}
}

// addBufGenYAML adds a buf.gen.yaml to the list of files to migrate. It returns nil
// nil if the file is already in v2.
//
// If the file is in v1 and has a 'types' section on the top level, this function will
// ignore 'types' and print a warning, while migrating everything else in the file.
//
// bufGenYAMLPath is relative to the call site of CLI or an absolute path.
func (m *migrateBuilder) addBufGenYAML(ctx context.Context, bufGenYAMLFilePath string) (retErr error) {
	if _, ok := m.addedBufGenYAMLFilePaths[bufGenYAMLFilePath]; ok {
		return nil
	}
	m.addedBufGenYAMLFilePaths[bufGenYAMLFilePath] = struct{}{}

	file, err := m.bucket.Get(ctx, bufGenYAMLFilePath)
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, file.Close())
	}()
	bufGenYAML, err := bufconfig.ReadBufGenYAMLFile(file)
	if err != nil {
		return err
	}
	if bufGenYAML.FileVersion() == bufconfig.FileVersionV2 {
		m.logger.Sugar().Warnf("%s is a v2 file, no migration required", bufGenYAMLFilePath)
		return nil
	}
	if typeConfig := bufGenYAML.GenerateConfig().GenerateTypeConfig(); typeConfig != nil && len(typeConfig.IncludeTypes()) > 0 {
		// TODO FUTURE: what does this sentence mean? Get someone else to read it and understand it without any explanation.
		m.logger.Sugar().Warnf(
			"%s is a v1 generation template with a top-level 'types' section including %s. In a v2 generation template, 'types' can"+
				" only exist within an input in the 'inputs' section. Since the migration command does not have information"+
				" on inputs, the migrated generation will not have an 'inputs' section. To add these types in the migrated file, you can"+
				" first add an input to 'inputs' and then add these types to the input.",
			bufGenYAMLFilePath,
			stringutil.SliceToHumanString(typeConfig.IncludeTypes()),
		)
	}
	// No special transformation needed, writeBufGenYAMLFile handles it correctly.
	migratedBufGenYAMLFile := bufconfig.NewBufGenYAMLFile(
		bufconfig.FileVersionV2,
		bufGenYAML.GenerateConfig(),
		// Types is always nil in v2.
		nil,
	)
	// Even though we're just writing over this, we store this so that Diff can pick it up.
	m.pathsToDelete[bufGenYAMLFilePath] = struct{}{}
	m.pathToMigratedBufGenYAMLFile[bufGenYAMLFilePath] = migratedBufGenYAMLFile
	return nil
}

// addWorkspace adds the buf.work.yaml at the root of the workspace directory
// to the list of files to migrate, the buf.yamls and buf.locks at the root of each
// directory pointed to by this workspace.
//
// workspaceDirectory is relative to the root bucket of the migrator.
func (m *migrateBuilder) addWorkspace(ctx context.Context, workspaceDirPath string) (retErr error) {
	if _, ok := m.addedWorkspaceDirPaths[workspaceDirPath]; ok {
		return nil
	}
	m.addedWorkspaceDirPaths[workspaceDirPath] = struct{}{}

	bufWorkYAML, err := bufconfig.GetBufWorkYAMLFileForPrefix(ctx, m.bucket, workspaceDirPath)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%q does not have a workspace configuration file (i.e. typically a buf.work.yaml)", workspaceDirPath)
	}
	if err != nil {
		return err
	}
	objectData := bufWorkYAML.ObjectData()
	if objectData == nil {
		return syserror.New("ObjectData was nil on BufWorkYAMLFile created for prefix")
	}
	m.pathsToDelete[normalpath.Join(workspaceDirPath, objectData.Name())] = struct{}{}
	for _, moduleDirRelativeToWorkspace := range bufWorkYAML.DirPaths() {
		if err := m.addModule(ctx, normalpath.Join(workspaceDirPath, moduleDirRelativeToWorkspace)); err != nil {
			return err
		}
	}
	return nil
}

// addModule adds buf.yaml and buf.lock at the root of moduleDir to the list
// of files to migrate. More specifically, it adds module configs and dependency module
// keys to the migrator.
//
// moduleDir is relative to the root bucket of the migrator.
func (m *migrateBuilder) addModule(ctx context.Context, moduleDirPath string) (retErr error) {
	if _, ok := m.addedModuleDirPaths[moduleDirPath]; ok {
		return nil
	}
	m.addedModuleDirPaths[moduleDirPath] = struct{}{}

	// First get module configs from the buf.yaml at moduleDir.
	bufYAMLFile, err := bufconfig.GetBufYAMLFileForPrefix(ctx, m.bucket, moduleDirPath)
	if errors.Is(err, fs.ErrNotExist) {
		// If buf.yaml isn't present, migration does not fail. Instead we add an
		// empty module config representing this directory.
		moduleRootRelativeToDestination, err := normalpath.Rel(m.destinationDirPath, moduleDirPath)
		if err != nil {
			return err
		}
		emptyModuleConfig, err := bufconfig.NewModuleConfig(
			moduleRootRelativeToDestination,
			nil,
			map[string][]string{
				".": {},
			},
			bufconfig.NewLintConfig(
				bufconfig.NewEnabledCheckConfigForUseIDsAndCategories(
					bufconfig.FileVersionV2,
					nil,
				),
				"",
				false,
				false,
				false,
				"",
				false,
			),
			bufconfig.NewBreakingConfig(
				bufconfig.NewEnabledCheckConfigForUseIDsAndCategories(
					bufconfig.FileVersionV2,
					nil,
				),
				false,
			),
		)
		if err != nil {
			return err
		}
		if err := m.appendModuleConfig(
			emptyModuleConfig,
			normalpath.Join(moduleDirPath, bufconfig.DefaultBufYAMLFileName),
		); err != nil {
			return err
		}
		// Assuming there is no co-resident buf.lock when there is no buf.yaml,
		// we return early here.
		return nil
	}
	if err != nil {
		return err
	}
	objectData := bufYAMLFile.ObjectData()
	if objectData == nil {
		return syserror.New("ObjectData was nil on BufYAMLFile created for prefix")
	}
	bufYAMLFilePath := normalpath.Join(moduleDirPath, objectData.Name())
	// If this module is already visited, we don't add it for a second time. It's
	// possbile to visit the same module directory twice when the user specifies both
	// a workspace and a module in this workspace.
	if _, ok := m.pathsToDelete[bufYAMLFilePath]; ok {
		return nil
	}
	switch bufYAMLFile.FileVersion() {
	case bufconfig.FileVersionV1Beta1:
		if len(bufYAMLFile.ModuleConfigs()) != 1 {
			// This should never happen because it's guaranteed by the bufYAMLFile interface.
			return syserror.Newf("expect exactly 1 module config from buf yaml, got %d", len(bufYAMLFile.ModuleConfigs()))
		}
		moduleConfig := bufYAMLFile.ModuleConfigs()[0]
		moduleFullName := moduleConfig.ModuleFullName()
		// If a buf.yaml v1beta1 has a non-empty name and multiple roots, the
		// resulting buf.yaml v2 should have these roots as module directories,
		// but they should not share the same module name. Instead we just give
		// them empty module names.
		if len(moduleConfig.RootToExcludes()) > 1 && moduleFullName != nil {
			m.logger.Sugar().Warnf(
				"%s has name %s and multiple roots. These roots are now separate unnamed modules.",
				bufYAMLFilePath,
				moduleFullName.String(),
			)
			moduleFullName = nil
		}
		// Each root in buf.yaml v1beta1 should become its own module config in v2,
		// and we iterate through these roots in deterministic order.
		sortedRoots := slicesext.MapKeysToSortedSlice(moduleConfig.RootToExcludes())
		for _, root := range sortedRoots {
			moduleRootRelativeToDestination, err := normalpath.Rel(
				m.destinationDirPath,
				normalpath.Join(moduleDirPath, root),
			)
			if err != nil {
				return err
			}
			lintConfigForRoot, err := equivalentLintConfigInV2(moduleConfig.LintConfig())
			if err != nil {
				return err
			}
			breakingConfigForRoot, err := equivalentBreakingConfigInV2(moduleConfig.BreakingConfig())
			if err != nil {
				return err
			}
			moduleConfigForRoot, err := bufconfig.NewModuleConfig(
				moduleRootRelativeToDestination,
				moduleFullName,
				// We do not need to handle paths in root-to-excludes, lint or breaking config specially,
				// because the paths are transformed correctly by readBufYAMLFile and writeBufYAMLFile.
				map[string][]string{".": moduleConfig.RootToExcludes()[root]},
				lintConfigForRoot,
				breakingConfigForRoot,
			)
			if err != nil {
				return err
			}
			if err := m.appendModuleConfig(moduleConfigForRoot, bufYAMLFilePath); err != nil {
				return err
			}
		}
		m.configuredDepModuleRefs = append(m.configuredDepModuleRefs, bufYAMLFile.ConfiguredDepModuleRefs()...)
	case bufconfig.FileVersionV1:
		if len(bufYAMLFile.ModuleConfigs()) != 1 {
			// This should never happen because it's guaranteed by the bufYAMLFile interface.
			return syserror.Newf("expect exactly 1 module config from buf yaml, got %d", len(bufYAMLFile.ModuleConfigs()))
		}
		moduleConfig := bufYAMLFile.ModuleConfigs()[0]
		moduleRootRelativeToDestination, err := normalpath.Rel(m.destinationDirPath, normalpath.Dir(bufYAMLFilePath))
		if err != nil {
			return err
		}
		lintConfig, err := equivalentLintConfigInV2(moduleConfig.LintConfig())
		if err != nil {
			return err
		}
		breakingConfig, err := equivalentBreakingConfigInV2(moduleConfig.BreakingConfig())
		if err != nil {
			return err
		}
		moduleConfig, err = bufconfig.NewModuleConfig(
			moduleRootRelativeToDestination,
			moduleConfig.ModuleFullName(),
			// We do not need to handle paths in root-to-excludes, lint or breaking config specially,
			// because the paths are transformed correctly by readBufYAMLFile and writeBufYAMLFile.
			moduleConfig.RootToExcludes(),
			lintConfig,
			breakingConfig,
		)
		if err != nil {
			return err
		}
		if err := m.appendModuleConfig(moduleConfig, bufYAMLFilePath); err != nil {
			return err
		}
		m.configuredDepModuleRefs = append(m.configuredDepModuleRefs, bufYAMLFile.ConfiguredDepModuleRefs()...)
	case bufconfig.FileVersionV2:
		m.logger.Sugar().Warnf("%s is a v2 file, no migration required", bufYAMLFilePath)
		return nil
	default:
		return syserror.Newf("unexpected version: %v", bufYAMLFile.FileVersion())
	}
	m.pathsToDelete[bufYAMLFilePath] = struct{}{}
	// Now we read buf.lock and add its lock entries to the list of candidate lock entries
	// for the migrated buf.lock. These lock entries are candidates because different buf.locks
	// can have lock entries for the same module but for different commits.
	bufLockFile, err := bufconfig.GetBufLockFileForPrefix(
		ctx,
		m.bucket,
		moduleDirPath,
		bufconfig.BufLockFileWithDigestResolver(
			func(ctx context.Context, remote string, commitID uuid.UUID) (bufmodule.Digest, error) {
				commitKey, err := bufmodule.NewCommitKey(remote, commitID, bufmodule.DigestTypeB4)
				if err != nil {
					return nil, err
				}
				commits, err := m.commitProvider.GetCommitsForCommitKeys(ctx, []bufmodule.CommitKey{commitKey})
				if err != nil {
					return nil, err
				}
				return commits[0].ModuleKey().Digest()
			},
		),
	)
	if errors.Is(errors.Unwrap(err), fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	objectData = bufLockFile.ObjectData()
	if objectData == nil {
		return syserror.New("ObjectData was nil on BufLockFile created for prefix")
	}
	bufLockFilePath := normalpath.Join(moduleDirPath, objectData.Name())
	// We don't need to check whether it's already in the map, but because if it were,
	// its co-resident buf.yaml would also have been a duplicate and made this
	// function return at an earlier point.
	m.pathsToDelete[bufLockFilePath] = struct{}{}
	m.hasSeenBufLockFile = true
	switch bufLockFile.FileVersion() {
	case bufconfig.FileVersionV1Beta1, bufconfig.FileVersionV1:
		m.depModuleKeys = append(m.depModuleKeys, bufLockFile.DepModuleKeys()...)
	case bufconfig.FileVersionV2:
		m.logger.Sugar().Warnf("%s is a v2 file, no migration required", bufLockFilePath)
		return nil
	default:
		return syserror.Newf("unrecognized version: %v", bufLockFile.FileVersion())
	}
	return nil
}

func (m *migrateBuilder) appendModuleConfig(moduleConfig bufconfig.ModuleConfig, parentPath string) error {
	m.moduleConfigs = append(m.moduleConfigs, moduleConfig)
	if moduleConfig.ModuleFullName() == nil {
		return nil
	}
	if file, ok := m.moduleFullNameStringToParentPath[moduleConfig.ModuleFullName().String()]; ok {
		return fmt.Errorf("module %s is found in both %s and %s", moduleConfig.ModuleFullName(), file, parentPath)
	}
	m.moduleFullNameStringToParentPath[moduleConfig.ModuleFullName().String()] = parentPath
	return nil
}
