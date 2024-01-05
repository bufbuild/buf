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
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	modulev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1beta1"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufbreaking"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/buflint"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleapi"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"go.uber.org/multierr"
)

type migrator struct {
	messageWriter  io.Writer
	clientProvider bufapi.ClientProvider
	// the bucket at "."
	rootBucket storage.ReadWriteBucket
	// the directory where the migrated buf.yaml live, this is useful for computing
	// module directory paths, and possibly other paths.
	destinationDir string

	moduleConfigs            []bufconfig.ModuleConfig
	moduleDependencies       []bufmodule.ModuleRef
	hasSeenBufLock           bool
	depModuleKeys            []bufmodule.ModuleKey
	pathToMigratedBufGenYAML map[string]bufconfig.BufGenYAMLFile
	moduleNameToParentFile   map[string]string
	filesToDelete            map[string]struct{}
}

func newMigrator(
	// usually stderr
	messageWriter io.Writer,
	clientProvider bufapi.ClientProvider,
	rootBucket storage.ReadWriteBucket,
	destinationDir string,
) *migrator {
	return &migrator{
		messageWriter:            messageWriter,
		clientProvider:           clientProvider,
		destinationDir:           destinationDir,
		rootBucket:               rootBucket,
		pathToMigratedBufGenYAML: map[string]bufconfig.BufGenYAMLFile{},
		moduleNameToParentFile:   map[string]string{},
		filesToDelete:            map[string]struct{}{},
	}
}

// addBufGenYAML adds a buf.gen.yaml to the list of files to migrate. It returns nil
// nil if the file is already in v2.
//
// If the file is in v1 and has a 'types' section on the top level, this function will
// ignore 'types' and print a warning, while migrating everything else in the file.
//
// bufGenYAMLPath is relative to the call site of CLI or an absolute path.
func (m *migrator) addBufGenYAML(
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
		m.warnf("%s is a v2 file, no migration required", bufGenYAMLPath)
		return nil
	}
	if typeConfig := bufGenYAML.GenerateConfig().GenerateTypeConfig(); typeConfig != nil && len(typeConfig.IncludeTypes()) > 0 {
		// TODO: what does this sentence mean? Get someone else to read it and understand it without any explanation.
		m.warnf(
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
	m.filesToDelete[bufGenYAMLPath] = struct{}{}
	m.pathToMigratedBufGenYAML[bufGenYAMLPath] = migratedBufGenYAML
	return nil
}

// addWorkspaceDirectory adds the buf.work.yaml at the root of the workspace directory
// to the list of files to migrate, the buf.yamls and buf.locks at the root of each
// directory pointed to by this workspace.
//
// workspaceDirectory is relative to the root bucket of the migrator.
func (m *migrator) addWorkspaceDirectory(
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
	m.filesToDelete[filepath.Join(workspaceDirectory, objectData.Name())] = struct{}{}
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
func (m *migrator) addModuleDirectory(
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
		moduleRootRelativeToDestination, err := filepath.Rel(m.destinationDir, moduleDir)
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
	if _, ok := m.filesToDelete[bufYAMLPath]; ok {
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
			m.warnf(
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
				m.destinationDir,
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
		moduleRootRelativeToDestination, err := filepath.Rel(m.destinationDir, filepath.Dir(bufYAMLPath))
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
		m.warnf("%s is a v2 file, no migration required", bufYAMLPath)
		return nil
	default:
		return syserror.Newf("unexpected version: %v", bufYAML.FileVersion())
	}
	m.filesToDelete[bufYAMLPath] = struct{}{}
	// Now we read buf.lock and add its lock entries to the list of candidate lock entries
	// for the migrated buf.lock. These lock entries are candiates because different buf.locks
	// can have lock entries for the same module but for different commits.
	bufLockFile, err := bufconfig.GetBufLockFileForPrefix(
		ctx,
		m.rootBucket,
		moduleDir,
		bufconfig.BufLockFileWithDigestResolver(
			func(ctx context.Context, remote, commitID string) (bufmodule.Digest, error) {
				return bufmoduleapi.DigestForCommitID(ctx, m.clientProvider, remote, commitID)
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
	m.filesToDelete[bufLockFilePath] = struct{}{}
	m.hasSeenBufLock = true
	switch bufLockFile.FileVersion() {
	case bufconfig.FileVersionV1Beta1, bufconfig.FileVersionV1:
		m.depModuleKeys = append(m.depModuleKeys, bufLockFile.DepModuleKeys()...)
	case bufconfig.FileVersionV2:
		m.warnf("%s is a v2 file, no migration required", bufLockFilePath)
		return nil
	default:
		return syserror.Newf("unrecognized version: %v", bufLockFile.FileVersion())
	}
	return nil
}

func (m *migrator) migrateAsDryRun(ctx context.Context) (retErr error) {
	if len(m.filesToDelete) > 0 {
		m.infof(
			"In an actual run, these files will be removed:\n%s\n\nThe following files will be overwritten or created:\n",
			strings.Join(slicesext.MapKeysToSortedSlice(m.filesToDelete), "\n"),
		)
	} else {
		m.info("In an actual run:\n")
	}
	// We create a buf.yaml if we have seen visited any module directory. Note
	// we add a module config even for a module directory without a buf.yaml.
	if len(m.moduleConfigs) > 0 {
		migratedBufYAML, migratedBufLock, err := m.buildBufYAMLAndBufLock(ctx)
		if err != nil {
			return err
		}
		m.infof(
			"%s will be written:\n",
			filepath.Join(m.destinationDir, bufconfig.DefaultBufWorkYAMLFileName),
		)
		if err := bufconfig.WriteBufYAMLFile(m.messageWriter, migratedBufYAML); err != nil {
			return err
		}
		if migratedBufLock != nil {
			m.infof(
				"%s will be written:\n",
				filepath.Join(m.destinationDir, bufconfig.DefaultBufLockFileName),
			)
			if err := bufconfig.WriteBufLockFile(m.messageWriter, migratedBufLock); err != nil {
				return err
			}
		}
	}
	for bufGenYAMLPath, migratedBufGenYAML := range m.pathToMigratedBufGenYAML {
		m.infof(
			"%s will be written:\n",
			bufGenYAMLPath,
		)
		if err := bufconfig.WriteBufGenYAMLFile(m.messageWriter, migratedBufGenYAML); err != nil {
			return err
		}
	}
	return nil
}

func (m *migrator) migrate(
	ctx context.Context,
) (retErr error) {
	for bufGenYAMLPath, migratedBufGenYAML := range m.pathToMigratedBufGenYAML {
		// os.Create truncates the existing file.
		file, err := os.Create(bufGenYAMLPath)
		if err != nil {
			return err
		}
		defer func() {
			retErr = multierr.Append(retErr, file.Close())
		}()
		if err := bufconfig.WriteBufGenYAMLFile(file, migratedBufGenYAML); err != nil {
			return err
		}
	}
	// We create a buf.yaml if we have seen visited any module directory. Note
	// we add a module config even for a module directory without a buf.yaml.
	if len(m.moduleConfigs) > 0 {
		migratedBufYAML, migratedBufLock, err := m.buildBufYAMLAndBufLock(ctx)
		if err != nil {
			return err
		}
		for _, fileToDelete := range slicesext.MapKeysToSortedSlice(m.filesToDelete) {
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
		if migratedBufLock != nil {
			if err := bufconfig.PutBufLockFileForPrefix(
				ctx,
				m.rootBucket,
				m.destinationDir,
				migratedBufLock,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

// If this function doesn't return an error, the BufYAMLFile returned is never nil,
// but the BufLockFile returned may be nil.
func (m *migrator) buildBufYAMLAndBufLock(
	ctx context.Context,
) (bufconfig.BufYAMLFile, bufconfig.BufLockFile, error) {
	// module full name --> the list of declared dependencies that are this module.
	depModuleToDeclaredRefs := make(map[string][]bufmodule.ModuleRef)
	for _, declaredRef := range m.moduleDependencies {
		moduleFullName := declaredRef.ModuleFullName().String()
		// If a declared dependency also shows up in the workspace, it's not a dependency.
		if _, ok := m.moduleNameToParentFile[moduleFullName]; ok {
			continue
		}
		depModuleToDeclaredRefs[moduleFullName] = append(depModuleToDeclaredRefs[moduleFullName], declaredRef)
	}
	// module full name --> the list of lock entries that are this module.
	depModuleToLockEntries := make(map[string][]bufmodule.ModuleKey)
	for _, lockEntry := range m.depModuleKeys {
		moduleFullName := lockEntry.ModuleFullName().String()
		// If a declared dependency also shows up in the workspace, it's not a dependency.
		//
		// We are only removing lock entries that are in the workspace. A lock entry
		// could be for an indirect dependenceny not listed in deps in any buf.yaml.
		if _, ok := m.moduleNameToParentFile[moduleFullName]; ok {
			continue
		}
		depModuleToLockEntries[moduleFullName] = append(depModuleToLockEntries[moduleFullName], lockEntry)
	}
	// This will be set to false if the duplicate dependencies cannot be resolved locally.
	areDependenciesResolved := true
	for depModule, declaredRefs := range depModuleToDeclaredRefs {
		refStringToRef := make(map[string]bufmodule.ModuleRef)
		for _, ref := range declaredRefs {
			// Add ref even if ref.Ref() is empty. Therefore, slicesext.ToValuesMap is not used.
			refStringToRef[ref.Ref()] = ref
		}
		// If there are both buf.build/foo/bar and buf.build/foo/bar:some_ref, the former will
		// not be used.
		if len(refStringToRef) > 1 {
			delete(refStringToRef, "")
		}
		depModuleToDeclaredRefs[depModule] = slicesext.MapValuesToSlice(refStringToRef)
		if len(refStringToRef) > 1 {
			areDependenciesResolved = false
		}
	}
	for depModule, lockEntries := range depModuleToLockEntries {
		commitIDToKey, err := slicesext.ToUniqueValuesMapError(
			lockEntries,
			func(moduleKey bufmodule.ModuleKey) (string, error) {
				return moduleKey.CommitID(), nil
			},
		)
		if err != nil {
			return nil, nil, err
		}
		depModuleToLockEntries[depModule] = slicesext.MapValuesToSlice(commitIDToKey)
		if len(commitIDToKey) > 1 {
			areDependenciesResolved = false
		}
	}
	if areDependenciesResolved {
		resolvedDeclaredRefs := make([]bufmodule.ModuleRef, 0, len(depModuleToDeclaredRefs))
		for _, depModuleRefs := range depModuleToDeclaredRefs {
			// depModuleRefs is guaranteed to have length 1, because areDependenciesResolved is true.
			resolvedDeclaredRefs = append(resolvedDeclaredRefs, depModuleRefs...)
		}
		bufYAML, err := bufconfig.NewBufYAMLFile(
			bufconfig.FileVersionV2,
			m.moduleConfigs,
			resolvedDeclaredRefs,
		)
		if err != nil {
			return nil, nil, err
		}
		resolvedLockEntries := make([]bufmodule.ModuleKey, 0, len(depModuleToLockEntries))
		for _, lockEntry := range depModuleToLockEntries {
			resolvedLockEntries = append(resolvedLockEntries, lockEntry...)
		}
		var bufLock bufconfig.BufLockFile
		if m.hasSeenBufLock {
			bufLock, err = bufconfig.NewBufLockFile(
				bufconfig.FileVersionV2,
				resolvedLockEntries,
			)
			if err != nil {
				return nil, nil, err
			}
		}
		// bufLock could be nil here, but that's OK, see docs for this function.
		return bufYAML, bufLock, nil
	}
	// TODO: remove entire if-clause when commit service is implemented
	if true {
		resolvedDepModuleRefs := make([]bufmodule.ModuleRef, 0, len(depModuleToDeclaredRefs))
		for _, depModuleRefs := range depModuleToDeclaredRefs {
			resolvedDepModuleRefs = append(resolvedDepModuleRefs, depModuleRefs[0])
		}
		bufYAML, err := bufconfig.NewBufYAMLFile(
			bufconfig.FileVersionV2,
			m.moduleConfigs,
			resolvedDepModuleRefs,
		)
		if err != nil {
			return nil, nil, err
		}
		resolvedDepModuleKeys := make([]bufmodule.ModuleKey, 0, len(depModuleToLockEntries))
		for _, depModuleKeys := range depModuleToLockEntries {
			resolvedDepModuleKeys = append(resolvedDepModuleKeys, depModuleKeys[0])
		}
		var bufLock bufconfig.BufLockFile
		if m.hasSeenBufLock {
			bufLock, err = bufconfig.NewBufLockFile(
				bufconfig.FileVersionV2,
				resolvedDepModuleKeys,
			)
			if err != nil {
				return nil, nil, err
			}
		}
		return bufYAML, bufLock, nil
	}
	// TODO: the code below this line isn't currently reachable.
	moduleToRefToCommit, err := getModuleToRefToCommit(ctx, m.clientProvider, m.moduleDependencies)
	if err != nil {
		return nil, nil, err
	}
	commitIDToCommit, err := getCommitIDToCommit(ctx, m.clientProvider, m.depModuleKeys)
	if err != nil {
		return nil, nil, err
	}
	resolvedDepModuleRefs, resolvedDepModuleKeys, err := resolvedDeclaredAndLockedDependencies(
		moduleToRefToCommit,
		commitIDToCommit,
		depModuleToDeclaredRefs,
		depModuleToLockEntries,
	)
	if err != nil {
		return nil, nil, err
	}
	bufYAML, err := bufconfig.NewBufYAMLFile(
		bufconfig.FileVersionV2,
		m.moduleConfigs,
		resolvedDepModuleRefs,
	)
	if err != nil {
		return nil, nil, err
	}
	var bufLock bufconfig.BufLockFile
	if m.hasSeenBufLock {
		bufLock, err = bufconfig.NewBufLockFile(
			bufconfig.FileVersionV2,
			resolvedDepModuleKeys,
		)
		if err != nil {
			return nil, nil, err
		}
	}
	return bufYAML, bufLock, nil
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

func (m *migrator) info(message string) {
	_, _ = m.messageWriter.Write([]byte(fmt.Sprintf("%s\n", message)))
}

func (m *migrator) infof(format string, args ...any) {
	_, _ = m.messageWriter.Write([]byte(fmt.Sprintf("%s\n", fmt.Sprintf(format, args...))))
}

func (m *migrator) warnf(format string, args ...any) {
	_, _ = m.messageWriter.Write([]byte(fmt.Sprintf("Warning: %s\n", fmt.Sprintf(format, args...))))
}

func resolvedDeclaredAndLockedDependencies(
	moduleToRefToCommit map[string]map[string]*modulev1beta1.Commit,
	commitIDToCommit map[string]*modulev1beta1.Commit,
	moduleFullNameToDeclaredRefs map[string][]bufmodule.ModuleRef,
	moduleFullNameToLockKeys map[string][]bufmodule.ModuleKey,
) ([]bufmodule.ModuleRef, []bufmodule.ModuleKey, error) {
	depModuleFullNameToResolvedRef := make(map[string]bufmodule.ModuleRef)
	for moduleFullName, refs := range moduleFullNameToDeclaredRefs {
		// There are multiple pinned versions of the same dependency, we use the latest one.
		sort.Slice(refs, func(i, j int) bool {
			refToCommit := moduleToRefToCommit[moduleFullName]
			iTime := refToCommit[refs[i].Ref()].GetCreateTime().AsTime()
			jTime := refToCommit[refs[j].Ref()].GetCreateTime().AsTime()
			return iTime.After(jTime)
		})
		depModuleFullNameToResolvedRef[moduleFullName] = refs[0]
	}
	resolvedDepModuleKeys := make([]bufmodule.ModuleKey, 0, len(moduleFullNameToLockKeys))
	for moduleFullName, lockKeys := range moduleFullNameToLockKeys {
		resolvedRef, ok := depModuleFullNameToResolvedRef[moduleFullName]
		if ok && resolvedRef.Ref() != "" {
			// If we have already picked a pinned dependency ref for this dependency,
			// we use that as the lock entry as well.
			resolvedCommit := moduleToRefToCommit[moduleFullName][resolvedRef.Ref()]
			key, err := bufmodule.NewModuleKey(
				resolvedRef.ModuleFullName(),
				resolvedCommit.GetId(),
				func() (bufmodule.Digest, error) {
					return bufmoduleapi.ProtoToDigest(resolvedCommit.GetDigest())
				},
			)
			if err != nil {
				return nil, nil, err
			}
			resolvedDepModuleKeys = append(resolvedDepModuleKeys, key)
			continue
		}
		// Otherwise, we pick the latest key from the buf.locks we have read.
		sort.Slice(lockKeys, func(i, j int) bool {
			iTime := commitIDToCommit[lockKeys[i].CommitID()].GetCreateTime().AsTime()
			jTime := commitIDToCommit[lockKeys[j].CommitID()].GetCreateTime().AsTime()
			return iTime.After(jTime)
		})
		resolvedDepModuleKeys = append(resolvedDepModuleKeys, lockKeys[0])
	}
	resolvedDeclaredDependencies := slicesext.MapValuesToSlice(depModuleFullNameToResolvedRef)
	// Sort the resolved dependencies for deterministic results.
	sort.Slice(resolvedDeclaredDependencies, func(i, j int) bool {
		return resolvedDeclaredDependencies[i].ModuleFullName().String() < resolvedDeclaredDependencies[j].ModuleFullName().String()
	})
	sort.Slice(resolvedDepModuleKeys, func(i, j int) bool {
		return resolvedDepModuleKeys[i].ModuleFullName().String() < resolvedDepModuleKeys[j].ModuleFullName().String()
	})
	return resolvedDeclaredDependencies, resolvedDepModuleKeys, nil
}

func getModuleToRefToCommit(
	ctx context.Context,
	clientProvider bufapi.ClientProvider,
	moduleRefs []bufmodule.ModuleRef,
) (map[string]map[string]*modulev1beta1.Commit, error) {
	moduleToRefToCommit := make(map[string]map[string]*modulev1beta1.Commit)
	for _, moduleRef := range moduleRefs {
		if moduleRef.Ref() == "" {
			continue
		}
		moduleFullName := moduleRef.ModuleFullName()
		response, err := clientProvider.CommitServiceClient(moduleFullName.Registry()).GetCommits(
			ctx,
			connect.NewRequest(
				&modulev1beta1.GetCommitsRequest{
					ResourceRefs: []*modulev1beta1.ResourceRef{
						{
							Value: &modulev1beta1.ResourceRef_Name_{
								Name: &modulev1beta1.ResourceRef_Name{
									Owner:  moduleFullName.Owner(),
									Module: moduleFullName.Name(),
									Child: &modulev1beta1.ResourceRef_Name_Ref{
										Ref: moduleRef.Ref(),
									},
								},
							},
						},
					},
				},
			),
		)
		if err != nil {
			if connect.CodeOf(err) == connect.CodeNotFound {
				return nil, &fs.PathError{Op: "read", Path: moduleRef.String(), Err: fs.ErrNotExist}
			}
			return nil, err
		}
		if len(response.Msg.Commits) != 1 {
			return nil, fmt.Errorf("expected 1 Commit, got %d", len(response.Msg.Commits))
		}
		if moduleToRefToCommit[moduleFullName.String()] == nil {
			moduleToRefToCommit[moduleFullName.String()] = make(map[string]*modulev1beta1.Commit)
		}
		moduleToRefToCommit[moduleFullName.String()][moduleRef.Ref()] = response.Msg.Commits[0]
	}
	return moduleToRefToCommit, nil
}

func getCommitIDToCommit(
	ctx context.Context,
	clientProvider bufapi.ClientProvider,
	moduleKeys []bufmodule.ModuleKey,
) (map[string]*modulev1beta1.Commit, error) {
	commitIDToCommit := make(map[string]*modulev1beta1.Commit)
	for _, moduleKey := range moduleKeys {
		moduleFullName := moduleKey.ModuleFullName()
		response, err := clientProvider.CommitServiceClient(moduleFullName.Registry()).GetCommits(
			ctx,
			connect.NewRequest(
				&modulev1beta1.GetCommitsRequest{
					ResourceRefs: []*modulev1beta1.ResourceRef{
						{
							Value: &modulev1beta1.ResourceRef_Id{
								// TODO: is this in the correct dashless/dashful form?
								Id: moduleKey.CommitID(),
							},
						},
					},
				},
			),
		)
		if err != nil {
			if connect.CodeOf(err) == connect.CodeNotFound {
				return nil, &fs.PathError{Op: "read", Path: moduleKey.CommitID(), Err: fs.ErrNotExist}
			}
			return nil, err
		}
		if len(response.Msg.Commits) != 1 {
			return nil, fmt.Errorf("expected 1 Commit, got %d", len(response.Msg.Commits))
		}
		commitIDToCommit[moduleKey.CommitID()] = response.Msg.Commits[0]
	}
	return commitIDToCommit, nil
}

func equivalentLintConfigInV2(lintConfig bufconfig.LintConfig) (bufconfig.LintConfig, error) {
	equivalentCheckConfigV2, err := equivalentCheckConfigInV2(
		lintConfig,
		func(checkConfig bufconfig.CheckConfig) ([]bufcheck.Rule, error) {
			lintConfig := bufconfig.NewLintConfig(
				checkConfig,
				lintConfig.EnumZeroValueSuffix(),
				lintConfig.RPCAllowSameRequestResponse(),
				lintConfig.RPCAllowGoogleProtobufEmptyRequests(),
				lintConfig.RPCAllowGoogleProtobufEmptyResponses(),
				lintConfig.ServiceSuffix(),
				lintConfig.AllowCommentIgnores(),
			)
			return buflint.RulesForConfig(lintConfig)
		},
	)
	if err != nil {
		return nil, err
	}
	return bufconfig.NewLintConfig(
		equivalentCheckConfigV2,
		lintConfig.EnumZeroValueSuffix(),
		lintConfig.RPCAllowSameRequestResponse(),
		lintConfig.RPCAllowGoogleProtobufEmptyRequests(),
		lintConfig.RPCAllowGoogleProtobufEmptyResponses(),
		lintConfig.ServiceSuffix(),
		lintConfig.AllowCommentIgnores(),
	), nil
}

func equivalentBreakingConfigInV2(breakingConfig bufconfig.BreakingConfig) (bufconfig.BreakingConfig, error) {
	equivalentCheckConfigV2, err := equivalentCheckConfigInV2(
		breakingConfig,
		func(checkConfig bufconfig.CheckConfig) ([]bufcheck.Rule, error) {
			breakingConfig := bufconfig.NewBreakingConfig(
				checkConfig,
				breakingConfig.IgnoreUnstablePackages(),
			)
			return bufbreaking.RulesForConfig(breakingConfig)
		},
	)
	if err != nil {
		return nil, err
	}
	return bufconfig.NewBreakingConfig(
		equivalentCheckConfigV2,
		breakingConfig.IgnoreUnstablePackages(),
	), nil
}

// Returns an equivalent check config with (close to) minimal difference in the
// list of rules and categories specified.
func equivalentCheckConfigInV2(
	checkConfig bufconfig.CheckConfig,
	getRulesFunc func(bufconfig.CheckConfig) ([]bufcheck.Rule, error),
) (bufconfig.CheckConfig, error) {
	// These are the rules we want the returned config to have in effect.
	// i.e. getRulesFunc(returnedConfig) should return this list.
	expectedRules, err := getRulesFunc(checkConfig)
	if err != nil {
		return nil, err
	}
	expectedIDs := slicesext.Map(
		expectedRules,
		func(rule bufcheck.Rule) string {
			return rule.ID()
		},
	)
	// First create a check config with the exact same UseIDsAndCategories. This
	// is a simple translation. It may or may not be equivalent to the given check config.
	simplyTranslatedCheckConfig := bufconfig.NewCheckConfig(
		bufconfig.FileVersionV2,
		checkConfig.UseIDsAndCategories(),
		checkConfig.ExceptIDsAndCategories(),
		checkConfig.IgnorePaths(),
		checkConfig.IgnoreIDOrCategoryToPaths(),
	)
	simplyTranslatedRules, err := getRulesFunc(simplyTranslatedCheckConfig)
	if err != nil {
		return nil, err
	}
	simplyTranslatedIDs := slicesext.Map(
		simplyTranslatedRules,
		func(rule bufcheck.Rule) string {
			return rule.ID()
		},
	)
	if slicesext.ElementsEqual(expectedIDs, simplyTranslatedIDs) {
		// If the simple translation is equivalent to before, use it.
		return simplyTranslatedCheckConfig, nil
	}
	// Otherwise, find what's missing and what's extra.
	expectedIDsMap := slicesext.ToStructMap(expectedIDs)
	simplyTranslatedIDsMap := slicesext.ToStructMap(simplyTranslatedIDs)
	missingIDs := slicesext.Filter(
		expectedIDs,
		func(expectedID string) bool {
			_, ok := simplyTranslatedIDsMap[expectedID]
			return !ok
		},
	)
	extraIDs := slicesext.Filter(
		simplyTranslatedIDs,
		func(simplyTranslatedID string) bool {
			_, ok := expectedIDsMap[simplyTranslatedID]
			return !ok
		},
	)
	return bufconfig.NewCheckConfig(
		bufconfig.FileVersionV2,
		append(checkConfig.UseIDsAndCategories(), missingIDs...),
		append(checkConfig.ExceptIDsAndCategories(), extraIDs...),
		checkConfig.IgnorePaths(),
		checkConfig.IgnoreIDOrCategoryToPaths(),
	), nil
}
