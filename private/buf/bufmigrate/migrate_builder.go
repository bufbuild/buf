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
	"os"
	"path/filepath"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/gofrs/uuid/v5"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type migrateBuilder struct {
	logger                   *zap.Logger
	commitProvider           bufmodule.CommitProvider
	rootBucket               storage.ReadWriteBucket
	destinationDirPath       string
	moduleConfigs            []bufconfig.ModuleConfig
	moduleDependencies       []bufmodule.ModuleRef
	hasSeenBufLock           bool
	depModuleKeys            []bufmodule.ModuleKey
	pathToMigratedBufGenYAML map[string]bufconfig.BufGenYAMLFile
	moduleNameToParentFile   map[string]string
	filePathsToDelete        map[string]struct{}
}

func newMigrateBuilder(
	logger *zap.Logger,
	commitProvider bufmodule.CommitProvider,
	rootBucket storage.ReadWriteBucket,
	destinationDirPath string,
) *migrateBuilder {
	return &migrateBuilder{
		logger:                   logger,
		commitProvider:           commitProvider,
		rootBucket:               rootBucket,
		destinationDirPath:       destinationDirPath,
		pathToMigratedBufGenYAML: map[string]bufconfig.BufGenYAMLFile{},
		moduleNameToParentFile:   map[string]string{},
		filePathsToDelete:        map[string]struct{}{},
	}
}

// addBufGenYAML adds a buf.gen.yaml to the list of files to migrate. It returns nil
// nil if the file is already in v2.
//
// If the file is in v1 and has a 'types' section on the top level, this function will
// ignore 'types' and print a warning, while migrating everything else in the file.
//
// bufGenYAMLPath is relative to the call site of CLI or an absolute path.
func (m *migrateBuilder) addBufGenYAML(
	bufGenYAMLPath string,
) (retErr error) {
	file, err := os.Open(bufGenYAMLPath)
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
		m.logger.Sugar().Warnf("%s is a v2 file, no migration required", bufGenYAMLPath)
		return nil
	}
	if typeConfig := bufGenYAML.GenerateConfig().GenerateTypeConfig(); typeConfig != nil && len(typeConfig.IncludeTypes()) > 0 {
		// TODO FUTURE: what does this sentence mean? Get someone else to read it and understand it without any explanation.
		m.logger.Sugar().Warnf(
			"%s is a v1 generation template with a top-level 'types' section including %s. In a v2 generation template, 'types' can"+
				" only exist within an input in the 'inputs' section. Since the migration command does not have information"+
				" on inputs, the migrated generation will not have an 'inputs' section. To add these types in the migrated file, you can"+
				" first add an input to 'inputs' and then add these types to the input.",
			bufGenYAMLPath,
			stringutil.SliceToHumanString(typeConfig.IncludeTypes()),
		)
	}
	// No special transformation needed, writeBufGenYAMLFile handles it correctly.
	migratedBufGenYAML := bufconfig.NewBufGenYAMLFile(
		bufconfig.FileVersionV2,
		bufGenYAML.GenerateConfig(),
		// Types is always nil in v2.
		nil,
	)
	m.filePathsToDelete[bufGenYAMLPath] = struct{}{}
	m.pathToMigratedBufGenYAML[bufGenYAMLPath] = migratedBufGenYAML
	return nil
}

// addWorkspaceDirectory adds the buf.work.yaml at the root of the workspace directory
// to the list of files to migrate, the buf.yamls and buf.locks at the root of each
// directory pointed to by this workspace.
//
// workspaceDirectory is relative to the root bucket of the migrator.
func (m *migrateBuilder) addWorkspaceDirectory(
	ctx context.Context,
	workspaceDirectory string,
) (retErr error) {
	bufWorkYAML, err := bufconfig.GetBufWorkYAMLFileForPrefix(
		ctx,
		m.rootBucket,
		workspaceDirectory,
	)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%q does not have a workspace configuration file (i.e. typically a buf.work.yaml)", workspaceDirectory)
	}
	if err != nil {
		return err
	}
	objectData := bufWorkYAML.ObjectData()
	if objectData == nil {
		return syserror.New("ObjectData was nil on BufWorkYAMLFile created for prefix")
	}
	m.filePathsToDelete[filepath.Join(workspaceDirectory, objectData.Name())] = struct{}{}
	for _, moduleDirRelativeToWorkspace := range bufWorkYAML.DirPaths() {
		if err := m.addModuleDirectory(ctx, filepath.Join(workspaceDirectory, moduleDirRelativeToWorkspace)); err != nil {
			return err
		}
	}
	return nil
}

// addModuleDirectory adds buf.yaml and buf.lock at the root of moduleDir to the list
// of files to migrate. More specifically, it adds module configs and dependency module
// keys to the migrator.
//
// moduleDir is relative to the root bucket of the migrator.
func (m *migrateBuilder) addModuleDirectory(
	ctx context.Context,
	moduleDir string,
) (retErr error) {
	// First get module configs from the buf.yaml at moduleDir.
	bufYAML, err := bufconfig.GetBufYAMLFileForPrefix(
		ctx,
		m.rootBucket,
		moduleDir,
	)
	if errors.Is(errors.Unwrap(err), fs.ErrNotExist) {
		// If buf.yaml isn't present, migration does not fail. Instead we add an
		// empty module config representing this directory.
		moduleRootRelativeToDestination, err := filepath.Rel(m.destinationDirPath, moduleDir)
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
				bufconfig.NewCheckConfig(
					bufconfig.FileVersionV2,
					nil,
					nil,
					nil,
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
				bufconfig.NewCheckConfig(
					bufconfig.FileVersionV2,
					nil,
					nil,
					nil,
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
			filepath.Join(moduleDir, bufconfig.DefaultBufYAMLFileName),
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
	objectData := bufYAML.ObjectData()
	if objectData == nil {
		return syserror.New("ObjectData was nil on BufYAMLFile created for prefix")
	}
	bufYAMLPath := filepath.Join(moduleDir, objectData.Name())
	// If this module is already visited, we don't add it for a second time. It's
	// possbile to visit the same module directory twice when the user specifies both
	// a workspace and a module in this workspace.
	if _, ok := m.filePathsToDelete[bufYAMLPath]; ok {
		return nil
	}
	switch bufYAML.FileVersion() {
	case bufconfig.FileVersionV1Beta1:
		if len(bufYAML.ModuleConfigs()) != 1 {
			// This should never happen because it's guaranteed by the bufYAMLFile interface.
			return syserror.Newf("expect exactly 1 module config from buf yaml, got %d", len(bufYAML.ModuleConfigs()))
		}
		moduleConfig := bufYAML.ModuleConfigs()[0]
		moduleFullName := moduleConfig.ModuleFullName()
		// If a buf.yaml v1beta1 has a non-empty name and multiple roots, the
		// resulting buf.yaml v2 should have these roots as module directories,
		// but they should not share the same module name. Instead we just give
		// them empty module names.
		if len(moduleConfig.RootToExcludes()) > 1 && moduleFullName != nil {
			m.logger.Sugar().Warnf(
				"%s has name %s and multiple roots. These roots are now separate unnamed modules.",
				bufYAMLPath,
				moduleFullName.String(),
			)
			moduleFullName = nil
		}
		// Each root in buf.yaml v1beta1 should become its own module config in v2,
		// and we iterate through these roots in deterministic order.
		sortedRoots := slicesext.MapKeysToSortedSlice(moduleConfig.RootToExcludes())
		for _, root := range sortedRoots {
			moduleRootRelativeToDestination, err := filepath.Rel(
				m.destinationDirPath,
				filepath.Join(moduleDir, root),
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
			if err := m.appendModuleConfig(moduleConfigForRoot, bufYAMLPath); err != nil {
				return err
			}
		}
		m.moduleDependencies = append(m.moduleDependencies, bufYAML.ConfiguredDepModuleRefs()...)
	case bufconfig.FileVersionV1:
		if len(bufYAML.ModuleConfigs()) != 1 {
			// This should never happen because it's guaranteed by the bufYAMLFile interface.
			return syserror.Newf("expect exactly 1 module config from buf yaml, got %d", len(bufYAML.ModuleConfigs()))
		}
		moduleConfig := bufYAML.ModuleConfigs()[0]
		moduleRootRelativeToDestination, err := filepath.Rel(m.destinationDirPath, filepath.Dir(bufYAMLPath))
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
		if err := m.appendModuleConfig(moduleConfig, bufYAMLPath); err != nil {
			return err
		}
		m.moduleDependencies = append(m.moduleDependencies, bufYAML.ConfiguredDepModuleRefs()...)
	case bufconfig.FileVersionV2:
		m.logger.Sugar().Warnf("%s is a v2 file, no migration required", bufYAMLPath)
		return nil
	default:
		return syserror.Newf("unexpected version: %v", bufYAML.FileVersion())
	}
	m.filePathsToDelete[bufYAMLPath] = struct{}{}
	// Now we read buf.lock and add its lock entries to the list of candidate lock entries
	// for the migrated buf.lock. These lock entries are candidates because different buf.locks
	// can have lock entries for the same module but for different commits.
	bufLockFile, err := bufconfig.GetBufLockFileForPrefix(
		ctx,
		m.rootBucket,
		moduleDir,
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
	bufLockFilePath := filepath.Join(moduleDir, objectData.Name())
	// We don't need to check whether it's already in the map, but because if it were,
	// its co-resident buf.yaml would also have been a duplicate and made this
	// function return at an earlier point.
	m.filePathsToDelete[bufLockFilePath] = struct{}{}
	m.hasSeenBufLock = true
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

func (m *migrateBuilder) appendModuleConfig(moduleConfig bufconfig.ModuleConfig, parentFile string) error {
	m.moduleConfigs = append(m.moduleConfigs, moduleConfig)
	if moduleConfig.ModuleFullName() == nil {
		return nil
	}
	if file, ok := m.moduleNameToParentFile[moduleConfig.ModuleFullName().String()]; ok {
		return fmt.Errorf("module %s is found in both %s and %s", moduleConfig.ModuleFullName(), file, parentFile)
	}
	m.moduleNameToParentFile[moduleConfig.ModuleFullName().String()] = parentFile
	return nil
}
