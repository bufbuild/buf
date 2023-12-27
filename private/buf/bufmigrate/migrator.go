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
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/buflint"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleapi"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

type migrator struct {
	clientProvider bufapi.ClientProvider
	// the directory where the migrated buf.yaml live, this is useful for computing
	// module directory paths, and possibly other paths.
	destinationDir string
	// the bucket at "."
	rootBucket storage.ReadWriteBucket

	// useful for creating new files
	moduleConfigs      []bufconfig.ModuleConfig
	moduleDependencies []bufmodule.ModuleRef
	depModuleKeys      []bufmodule.ModuleKey

	moduleNameToParentFile map[string]string
	seenFiles              map[string]struct{}
}

func newMigrator(
	clientProvider bufapi.ClientProvider,
	rootBucket storage.ReadWriteBucket,
	destinationDir string,
) *migrator {
	return &migrator{
		clientProvider:         clientProvider,
		destinationDir:         destinationDir,
		rootBucket:             rootBucket,
		moduleNameToParentFile: map[string]string{},
		seenFiles:              map[string]struct{}{},
	}
}

func (m *migrator) addWorkspaceDirectory(
	ctx context.Context,
	workspaceDir string,
) error {
	bufWorkYAML, err := bufconfig.GetBufWorkYAMLFileForPrefix(ctx, m.rootBucket, workspaceDir)
	if err != nil {
		return err
	}
	// TODO: get path properly
	m.seenFiles[filepath.Join(workspaceDir, "buf.work.yaml")] = struct{}{}
	for _, moduleDirRelativeToWorkspace := range bufWorkYAML.DirPaths() {
		if err := m.addModuleDirectory(ctx, filepath.Join(workspaceDir, moduleDirRelativeToWorkspace)); err != nil {
			return err
		}
	}
	return nil
}

// both buf.yaml and buf.lock
func (m *migrator) addModuleDirectory(
	ctx context.Context,
	// moduleDir is the relative path (relative to ".") to the module directory
	moduleDir string,
) error {
	// TODO: get file path properly
	bufYAMLPath := filepath.Join(moduleDir, "buf.yaml")
	bufYAMLFile, err := bufconfig.GetBufYAMLFileForPrefix(
		ctx,
		m.rootBucket,
		moduleDir,
	)
	if errors.Is(errors.Unwrap(err), fs.ErrNotExist) {
		moduleDirInMigratedBufYAML, err := filepath.Rel(m.destinationDir, moduleDir)
		if err != nil {
			return err
		}
		moduleConfig, err := bufconfig.NewModuleConfig(
			moduleDirInMigratedBufYAML,
			nil,
			nil,
			nil,
			nil,
		)
		if err != nil {
			return err
		}
		if err := m.appendModuleConfig(moduleConfig, bufYAMLPath); err != nil {
			return err
		}
		// Assume there is no co-resident buf.lock
		return nil
	}
	if err != nil {
		return err
	}
	if err := m.addBufYAML(
		ctx,
		bufYAMLFile,
		bufYAMLPath,
	); err != nil {
		return err
	}
	bufLockFile, err := bufconfig.GetBufLockFileForPrefix(
		ctx,
		m.rootBucket,
		moduleDir,
		bufconfig.BufLockFileWithModuleDigestResolver(
			func(ctx context.Context, remote, commitID string) (bufmodule.ModuleDigest, error) {
				return bufmoduleapi.ModuleDigestForCommitID(ctx, m.clientProvider, remote, commitID)
			},
		),
	)
	if errors.Is(errors.Unwrap(err), fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	// TODO: get file paths properly
	bufLockFilePath := filepath.Join(moduleDir, "buf.lock")
	// We don't need to check whether it's already in the map, but because if it were,
	// its co-resident buf.yaml would also have been a duplicate, which would make this
	// function return early.
	m.seenFiles[bufLockFilePath] = struct{}{}
	switch bufLockFile.FileVersion() {
	case bufconfig.FileVersionV1Beta1, bufconfig.FileVersionV1:
		m.depModuleKeys = append(m.depModuleKeys, bufLockFile.DepModuleKeys()...)
	case bufconfig.FileVersionV2:
		return fmt.Errorf("%s is already at v2", bufLockFilePath)
	default:
		return syserror.Newf("unrecognized version: %v", bufLockFile.FileVersion())
	}
	return nil
}

func (m *migrator) migrateAsDryRun(
	writer io.Writer,
) error {
	migratedBufYAML, err := m.getBufYAML()
	if err != nil {
		return err
	}
	migratedBufLock, err := m.getBufLock()
	if err != nil {
		return err
	}
	filesToDelete := m.filesToDelete()
	var bufYAMLBuffer bytes.Buffer
	if err := bufconfig.WriteBufYAMLFile(&bufYAMLBuffer, migratedBufYAML); err != nil {
		return err
	}
	var bufLockBuffer bytes.Buffer
	if err := bufconfig.WriteBufLockFile(&bufLockBuffer, migratedBufLock); err != nil {
		return err
	}
	fmt.Fprintf(
		writer,
		`in an actual run, these files will be removed:
%s

these files will be written:
%s:
%s
%s:
%s
`,
		strings.Join(filesToDelete, "\n"),
		// TODO: find a way to get file name properly
		filepath.Join(m.destinationDir, "buf.yaml"),
		bufYAMLBuffer.String(),
		// TODO: find a way to get file name properly
		filepath.Join(m.destinationDir, "buf.lock"),
		bufLockBuffer.String(),
	)
	return nil
}

func (m *migrator) migrate(
	ctx context.Context,
) error {
	migratedBufYAML, err := m.getBufYAML()
	if err != nil {
		return err
	}
	migratedBufLock, err := m.getBufLock()
	if err != nil {
		return err
	}
	filesToDelete := m.filesToDelete()
	for _, fileToDelete := range filesToDelete {
		if err := os.Remove(fileToDelete); err != nil {
			return err
		}
	}
	if err := bufconfig.PutBufYAMLFileForPrefix(
		ctx,
		m.rootBucket,
		m.destinationDir,
		migratedBufYAML,
	); err != nil {
		return err
	}
	if err := bufconfig.PutBufLockFileForPrefix(
		ctx,
		m.rootBucket,
		m.destinationDir,
		migratedBufLock,
	); err != nil {
		return err
	}
	return nil
}

func (m *migrator) addBufYAML(
	ctx context.Context,
	bufYAML bufconfig.BufYAMLFile,
	bufYAMLPath string,
) error {
	if _, ok := m.seenFiles[bufYAMLPath]; ok {
		// TODO: this isn't always the case, perhaps the first time it was read as part of the workspace.
		// TODO: we could also return nil here.
		return fmt.Errorf("%s is specified multiple times", bufYAMLPath)
	}
	m.seenFiles[bufYAMLPath] = struct{}{}
	// TODO: transform paths so that they are relative to the new buf.yaml v2 or module root (depending on buf.yaml v2 semantics)
	switch bufYAML.FileVersion() {
	case bufconfig.FileVersionV1Beta1:
		// TODO: whether something needs to be done about root to exclude mapping (what is it relative to now?)
		if len(bufYAML.ModuleConfigs()) != 1 {
			return syserror.Newf("expect exactly 1 module config from buf yaml, got %d", len(bufYAML.ModuleConfigs()))
		}
		moduleConfig := bufYAML.ModuleConfigs()[0]
		// TODO: iterate through this map in deterministic order
		for root, excludes := range moduleConfig.RootToExcludes() {
			lintConfig := moduleConfig.LintConfig()
			// TODO: this list expands to individual rules, we could process
			// this list and make it shorter by substituting some rules with
			// a single group, if all rules in that group are present.
			lintRules, err := buflint.RulesForConfig(lintConfig)
			if err != nil {
				return err
			}
			lintRuleNames := slicesext.Map(lintRules, func(rule bufcheck.Rule) string { return rule.ID() })
			lintConfigForRoot := bufconfig.NewLintConfig(
				bufconfig.NewCheckConfig(
					bufconfig.FileVersionV2,
					lintRuleNames,
					lintConfig.ExceptIDsAndCategories(),
					// TODO: filter these paths by root
					lintConfig.IgnorePaths(),
					// TODO: filter these paths by root
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
					// TODO: FIELD_SAME_TYPE
					breakingConfig.UseIDsAndCategories(),
					breakingConfig.ExceptIDsAndCategories(),
					// TODO: filter these paths by root
					breakingConfig.IgnorePaths(),
					// TODO: filter these paths by root
					breakingConfig.IgnoreIDOrCategoryToPaths(),
				),
				breakingConfig.IgnoreUnstablePackages(),
			)
			dirPathRelativeToDestination, err := filepath.Rel(
				m.destinationDir,
				filepath.Join(
					filepath.Dir(bufYAMLPath),
					root,
				),
			)
			if err != nil {
				return err
			}
			// If a v1beta buf.yaml has multiple roots, they are split into multiple
			// module configs, but they cannot share the same module full name.
			moduleFullName := moduleConfig.ModuleFullName()
			if len(moduleConfig.RootToExcludes()) > 1 && moduleFullName != nil {
				moduleFullName, err = bufmodule.NewModuleFullName(
					moduleFullName.Registry(),
					moduleFullName.Owner(),
					// Note: roots are normalized, "/" is universal
					moduleFullName.Name()+"-"+strings.ReplaceAll(root, "/", "-"),
				)
				if err != nil {
					return err
				}
			}
			configForRoot, err := bufconfig.NewModuleConfig(
				dirPathRelativeToDestination,
				moduleFullName,
				// TODO: excludes might need to be transformed WRT what it's relative to
				map[string][]string{".": excludes},
				lintConfigForRoot,
				breakingConfigForRoot,
			)
			if err != nil {
				return err
			}
			if err := m.appendModuleConfig(configForRoot, bufYAMLPath); err != nil {
				return err
			}
		}
		m.moduleDependencies = append(m.moduleDependencies, bufYAML.ConfiguredDepModuleRefs()...)
		return nil
	case bufconfig.FileVersionV1:
		// TODO: smiliar to the above, make paths (root to excludes, lint ignore, ...) relative to the correct root (either buf.yaml v2 or module root)
		if len(bufYAML.ModuleConfigs()) != 1 {
			return syserror.Newf("expect exactly 1 module config from buf yaml, got %d", len(bufYAML.ModuleConfigs()))
		}
		moduleConfig := bufYAML.ModuleConfigs()[0]
		// use the same lint and breaking config, except that they are v2.
		lintConfig := moduleConfig.LintConfig()
		lintConfig = bufconfig.NewLintConfig(
			bufconfig.NewCheckConfig(
				bufconfig.FileVersionV2,
				lintConfig.UseIDsAndCategories(),
				lintConfig.ExceptIDsAndCategories(),
				// TODO: paths
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
				// TODO: paths
				breakingConfig.IgnorePaths(),
				breakingConfig.IgnoreIDOrCategoryToPaths(),
			),
			breakingConfig.IgnoreUnstablePackages(),
		)
		dirPathRelativeToDestination, err := filepath.Rel(m.destinationDir, filepath.Dir(bufYAMLPath))
		if err != nil {
			return err
		}
		moduleConfig, err = bufconfig.NewModuleConfig(
			dirPathRelativeToDestination,
			moduleConfig.ModuleFullName(),
			// TODO: paths
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
		return nil
	case bufconfig.FileVersionV2:
		return fmt.Errorf("%s is already at v2", bufYAMLPath)
	default:
		return syserror.Newf("unexpected version: %v", bufYAML.FileVersion())
	}
}

func (m *migrator) getBufYAML() (bufconfig.BufYAMLFile, error) {
	// TODO: where do we update seenModuleNames?
	filteredModuleDependencies := slicesext.Filter(
		m.moduleDependencies,
		func(moduleRef bufmodule.ModuleRef) bool {
			_, ok := m.moduleNameToParentFile[moduleRef.ModuleFullName().String()]
			return !ok
		},
	)
	// TODO: deduplicate/resolve dependency list
	return bufconfig.NewBufYAMLFile(
		bufconfig.FileVersionV2,
		m.moduleConfigs,
		filteredModuleDependencies,
	)
}

func (m *migrator) getBufLock() (bufconfig.BufLockFile, error) {
	depModuleFullNameToModuleKeys := make(map[string][]bufmodule.ModuleKey)
	for _, depModuleKey := range m.depModuleKeys {
		depModuleFullName := depModuleKey.ModuleFullName().String()
		depModuleFullNameToModuleKeys[depModuleFullName] = append(depModuleFullNameToModuleKeys[depModuleFullName], depModuleKey)
	}
	// TODO: these are resolved arbitrarily right now, we need to resolve them by commit time
	resolvedDepModuleKeys := make([]bufmodule.ModuleKey, 0, len(depModuleFullNameToModuleKeys))
	for _, depModuleKeys := range depModuleFullNameToModuleKeys {
		// TODO: actually resolve dependencies by time
		// The alternative is to build the workspace with tentative dependencies and
		// find the latest one that does not break. However, what if there are 3 dependencies
		// in question, each has 4 potential versions. We don't want to build 4*4*4 times in the worst case.
		resolvedDepModuleKeys = append(resolvedDepModuleKeys, depModuleKeys[0])
	}
	return bufconfig.NewBufLockFile(
		bufconfig.FileVersionV2,
		resolvedDepModuleKeys,
	)
}

func (m *migrator) appendModuleConfig(moduleConfig bufconfig.ModuleConfig, parentFile string) error {
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

func (m *migrator) filesToDelete() []string {
	return slicesext.MapKeysToSortedSlice(m.seenFiles)
}
