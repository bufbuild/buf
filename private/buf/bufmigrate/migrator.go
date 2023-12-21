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

	modulev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1beta1"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufbreaking"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/buflint"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleapi"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
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
			"%s has types configuration and is migrated to v2 without it. To add these types your v2 configuration, first add an input and add these types to it.",
		)
	}
	migratedBufGenYAML := bufconfig.NewBufGenYAMLFile(
		bufconfig.FileVersionV2,
		bufGenYAML.GenerateConfig(),
		nil,
	)
	m.filesToDelete[bufGenYAMLPath] = struct{}{}
	m.pathToMigratedBufGenYAML[bufGenYAMLPath] = migratedBufGenYAML
	return nil
}

// TODO: document is workspaceDirectory always relative to root of bucket?
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
	m.filesToDelete[filepath.Join(workspaceDirectory, bufWorkYAML.FileName())] = struct{}{}
	for _, moduleDirRelativeToWorkspace := range bufWorkYAML.DirPaths() {
		if err := m.addModuleDirectory(ctx, filepath.Join(workspaceDirectory, moduleDirRelativeToWorkspace)); err != nil {
			return err
		}
	}
	return nil
}

// both buf.yaml and buf.lock TODO what does this mean?
// TODO: document is workspaceDirectory always relative to root of bucket?
func (m *migrator) addModuleDirectory(
	ctx context.Context,
	// moduleDir is the relative path (relative to ".") to the module directory
	moduleDir string,
) (retErr error) {
	bufYAML, err := bufconfig.GetBufYAMLFileForPrefix(
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
			moduleConfig,
			filepath.Join(moduleDir, bufconfig.DefaultBufYAMLFileName),
		); err != nil {
			return err
		}
		// Assume there is no co-resident buf.lock
		return nil
	}
	if err != nil {
		return err
	}
	bufYAMLPath := filepath.Join(moduleDir, bufconfig.DefaultBufYAMLFileName)
	if _, ok := m.filesToDelete[bufYAMLPath]; ok {
		return nil
	}
	switch bufYAML.FileVersion() {
	case bufconfig.FileVersionV1Beta1:
		if len(bufYAML.ModuleConfigs()) != 1 {
			return syserror.Newf("expect exactly 1 module config from buf yaml, got %d", len(bufYAML.ModuleConfigs()))
		}
		moduleConfig := bufYAML.ModuleConfigs()[0]
		// If a v1beta buf.yaml has multiple roots, they are split into multiple
		// module configs, but they cannot share the same module full name.
		moduleFullName := moduleConfig.ModuleFullName()
		if len(moduleConfig.RootToExcludes()) > 1 && moduleFullName != nil {
			m.warnf(
				"%s has name %s and multiple roots. These roots are now separate unnamed modules.",
				bufYAMLPath,
				moduleFullName.String(),
			)
			moduleFullName = nil
		}
		// Iterate through root-to-excludes in deterministic order.
		sortedRoots := slicesext.MapKeysToSortedSlice(moduleConfig.RootToExcludes())
		for _, root := range sortedRoots {
			excludes := moduleConfig.RootToExcludes()[root]
			lintConfigForRoot, err := equivalentLintConfigInV2(moduleConfig.LintConfig())
			if err != nil {
				return err
			}
			breakingConfigForRoot, err := equivalentBreakingConfigInV2(moduleConfig.BreakingConfig())
			if err != nil {
				return err
			}
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
			configForRoot, err := bufconfig.NewModuleConfig(
				dirPathRelativeToDestination,
				moduleFullName,
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
	case bufconfig.FileVersionV1:
		if len(bufYAML.ModuleConfigs()) != 1 {
			return syserror.Newf("expect exactly 1 module config from buf yaml, got %d", len(bufYAML.ModuleConfigs()))
		}
		moduleConfig := bufYAML.ModuleConfigs()[0]
		lintConfig, err := equivalentLintConfigInV2(moduleConfig.LintConfig())
		if err != nil {
			return err
		}
		breakingConfig, err := equivalentBreakingConfigInV2(moduleConfig.BreakingConfig())
		if err != nil {
			return err
		}
		dirPathRelativeToDestination, err := filepath.Rel(m.destinationDir, filepath.Dir(bufYAMLPath))
		if err != nil {
			return err
		}
		moduleConfig, err = bufconfig.NewModuleConfig(
			dirPathRelativeToDestination,
			moduleConfig.ModuleFullName(),
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
		m.warnf("%s is already at v2", bufYAMLPath)
		return nil
	default:
		return syserror.Newf("unexpected version: %v", bufYAML.FileVersion())
	}
	m.filesToDelete[bufYAMLPath] = struct{}{}
	bufLockFile, err := bufconfig.GetBufLockFileForPrefix(
		ctx,
		m.rootBucket,
		moduleDir,
		bufconfig.BufLockFileWithDigestResolver(
			func(ctx context.Context, remote, commitID string) (bufcas.Digest, error) {
				return bufmoduleapi.CommitIDToDigest(ctx, m.clientProvider, remote, commitID)
			},
		),
	)
	if errors.Is(errors.Unwrap(err), fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	bufLockFilePath := filepath.Join(moduleDir, bufconfig.DefaultBufLockFileName)
	// We don't need to check whether it's already in the map, but because if it were,
	// its co-resident buf.yaml would also have been a duplicate and made this
	// function return at an earlier point.
	m.filesToDelete[bufLockFilePath] = struct{}{}
	switch bufLockFile.FileVersion() {
	case bufconfig.FileVersionV1Beta1, bufconfig.FileVersionV1:
		m.depModuleKeys = append(m.depModuleKeys, bufLockFile.DepModuleKeys()...)
	case bufconfig.FileVersionV2:
		return fmt.Errorf("%s is already at v2", bufLockFilePath)
	default:
		return syserror.Newf("unrecognized version: %v", bufLockFile.FileVersion())
	}
	m.hasSeenBufLock = true
	return nil
}

func (m *migrator) migrateAsDryRun(ctx context.Context) (retErr error) {
	m.infof(
		"In an actual run, these files will be removed:\n%s\n\nThe following files will be overwritten or created:\n",
		strings.Join(slicesext.MapKeysToSortedSlice(m.filesToDelete), "\n"),
	)
	if len(m.moduleConfigs) > 0 {
		migratedBufYAML, migratedBufLock, err := m.buildBufYAMLAndBufLock(ctx)
		if err != nil {
			return err
		}
		var bufYAMLBuffer bytes.Buffer
		if err := bufconfig.WriteBufYAMLFile(&bufYAMLBuffer, migratedBufYAML); err != nil {
			return err
		}
		m.infof(
			"%s:\n%s",
			filepath.Join(m.destinationDir, bufconfig.DefaultBufYAMLFileName),
			bufYAMLBuffer.String(),
		)
		if migratedBufLock != nil {
			var bufLockBuffer bytes.Buffer
			if err := bufconfig.WriteBufLockFile(&bufLockBuffer, migratedBufLock); err != nil {
				return err
			}
			m.infof(
				"%s:\n%s",
				filepath.Join(m.destinationDir, bufconfig.DefaultBufLockFileName),
				bufLockBuffer.String(),
			)
		}
	}
	for bufGenYAMLPath, migratedBufGenYAML := range m.pathToMigratedBufGenYAML {
		var bufGenYAMLBuffer bytes.Buffer
		if err := bufconfig.WriteBufGenYAMLFile(&bufGenYAMLBuffer, migratedBufGenYAML); err != nil {
			return err
		}
		m.infof(
			"%s will be written:\n%s",
			bufGenYAMLPath,
			bufGenYAMLBuffer.String(),
		)
	}
	return nil
}

func (m *migrator) migrate(
	ctx context.Context,
) (retErr error) {
	for bufGenYAMLPath, migratedBufGenYAML := range m.pathToMigratedBufGenYAML {
		file, err := os.OpenFile(bufGenYAMLPath, os.O_WRONLY|os.O_TRUNC, 0644)
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

// If no error, the BufYAMLFile returned is never nil, but the BufLockFile returne may be nil.
func (m *migrator) buildBufYAMLAndBufLock(
	ctx context.Context,
) (bufconfig.BufYAMLFile, bufconfig.BufLockFile, error) {
	depModuleToRefs := make(map[string][]bufmodule.ModuleRef)
	for _, depModuleRef := range m.moduleDependencies {
		moduleFullName := depModuleRef.ModuleFullName().String()
		if _, ok := m.moduleNameToParentFile[moduleFullName]; ok {
			continue
		}
		depModuleToRefs[moduleFullName] = append(depModuleToRefs[moduleFullName], depModuleRef)
	}
	depModuleToKeys := make(map[string][]bufmodule.ModuleKey)
	for _, depModuleKey := range m.depModuleKeys {
		moduleFullName := depModuleKey.ModuleFullName().String()
		// We are only removing lock entries that are in the workspace. A lock entry
		// could be for an indirect dependenceny not listed in deps in any buf.yaml.
		if _, ok := m.moduleNameToParentFile[moduleFullName]; ok {
			continue
		}
		depModuleToKeys[moduleFullName] = append(depModuleToKeys[moduleFullName], depModuleKey)
	}
	areDependenciesResolved := true
	for depModule, depModuleRefs := range depModuleToRefs {
		refStringToRef := make(map[string]bufmodule.ModuleRef)
		for _, ref := range depModuleRefs {
			// Add ref even if ref.Ref() is empty. Therefore, slicesext.ToValuesMap is not used.
			refStringToRef[ref.Ref()] = ref
		}
		// If there are both buf.build/foo/bar and buf.build/foo/bar:some_ref, take the latter.
		if len(refStringToRef) > 1 {
			delete(refStringToRef, "")
		}
		if len(refStringToRef) > 1 {
			areDependenciesResolved = false
		}
		depModuleToRefs[depModule] = slicesext.MapValuesToSlice(refStringToRef)
	}
	for depModule, depModuleKeys := range depModuleToKeys {
		commitIDToKey := slicesext.ToValuesMap(
			depModuleKeys,
			func(moduleKey bufmodule.ModuleKey) string {
				return moduleKey.CommitID()
			},
		)
		if len(commitIDToKey) > 1 {
			areDependenciesResolved = false
		}
		depModuleToKeys[depModule] = slicesext.MapValuesToSlice(commitIDToKey)
	}
	if areDependenciesResolved {
		resolvedDepModuleRefs := make([]bufmodule.ModuleRef, 0, len(depModuleToRefs))
		for _, depModuleRefs := range depModuleToRefs {
			// depModuleRefs is guaranteed to have length 1
			resolvedDepModuleRefs = append(resolvedDepModuleRefs, depModuleRefs...)
		}
		bufYAML, err := bufconfig.NewBufYAMLFile(
			bufconfig.FileVersionV2,
			m.moduleConfigs,
			resolvedDepModuleRefs,
		)
		if err != nil {
			return nil, nil, err
		}
		resolvedDepModuleKeys := make([]bufmodule.ModuleKey, 0, len(depModuleToKeys))
		for _, depModuleKeys := range depModuleToKeys {
			resolvedDepModuleKeys = append(resolvedDepModuleKeys, depModuleKeys...)
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
	// TODO: remove entire if-clause when commit service is implemented
	if true {
		resolvedDepModuleRefs := make([]bufmodule.ModuleRef, 0, len(depModuleToRefs))
		for _, depModuleRefs := range depModuleToRefs {
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
		resolvedDepModuleKeys := make([]bufmodule.ModuleKey, 0, len(depModuleToKeys))
		for _, depModuleKeys := range depModuleToKeys {
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
		depModuleToRefs,
		depModuleToKeys,
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

func (m *migrator) warn(message string) {
	_, _ = m.messageWriter.Write([]byte(fmt.Sprintf("Warning: %s\n", message)))
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
		resolvedRef := depModuleFullNameToResolvedRef[moduleFullName]
		if resolvedRef.Ref() != "" {
			resolvedCommit := moduleToRefToCommit[moduleFullName][resolvedRef.Ref()]
			key, err := bufmodule.NewModuleKey(
				resolvedRef.ModuleFullName(),
				resolvedCommit.GetId(),
				func() (bufcas.Digest, error) { return bufcas.ProtoToDigest(resolvedCommit.GetDigest()) },
			)
			if err != nil {
				return nil, nil, err
			}
			resolvedDepModuleKeys = append(resolvedDepModuleKeys, key)
			continue
		}
		// Use the lastest
		sort.Slice(lockKeys, func(i, j int) bool {
			iTime := commitIDToCommit[lockKeys[i].CommitID()].GetCreateTime().AsTime()
			jTime := commitIDToCommit[lockKeys[j].CommitID()].GetCreateTime().AsTime()
			return iTime.After(jTime)
		})
		resolvedDepModuleKeys = append(resolvedDepModuleKeys, lockKeys[0])
	}
	resolvedDeclaredDependencies := slicesext.MapValuesToSlice(depModuleFullNameToResolvedRef)
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
		response, err := clientProvider.CommitServiceClient(moduleFullName.Registry()).ResolveCommits(
			ctx,
			connect.NewRequest(
				&modulev1beta1.ResolveCommitsRequest{
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
		response, err := clientProvider.CommitServiceClient(moduleFullName.Registry()).ResolveCommits(
			ctx,
			connect.NewRequest(
				&modulev1beta1.ResolveCommitsRequest{
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

func equivalentCheckConfigInV2(
	checkConfig bufconfig.CheckConfig,
	getRulesFunc func(bufconfig.CheckConfig) ([]bufcheck.Rule, error),
) (bufconfig.CheckConfig, error) {
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
		return simplyTranslatedCheckConfig, nil
	}
	expectedIDsMap := slicesext.ToStructMap(expectedIDs)
	simplyTranslatedIDsMap := slicesext.ToStructMap(simplyTranslatedIDs)
	extraUse := slicesext.Filter(
		expectedIDs,
		func(expectedID string) bool {
			_, ok := simplyTranslatedIDsMap[expectedID]
			return !ok
		},
	)
	extraExcept := slicesext.Filter(
		simplyTranslatedIDs,
		func(simplyTranslatedID string) bool {
			_, ok := expectedIDsMap[simplyTranslatedID]
			return !ok
		},
	)
	return bufconfig.NewCheckConfig(
		bufconfig.FileVersionV2,
		append(checkConfig.UseIDsAndCategories(), extraUse...),
		append(checkConfig.ExceptIDsAndCategories(), extraExcept...),
		checkConfig.IgnorePaths(),
		checkConfig.IgnoreIDOrCategoryToPaths(),
	), nil
}
