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
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type migrator struct {
	logger         *zap.Logger
	clientProvider bufapi.ClientProvider
	// the directory where the migrated buf.yaml live, this is useful for computing
	// module directory paths, and possibly other paths.
	destinationDir string
	// the bucket at "."
	rootBucket storage.ReadWriteBucket

	moduleConfigs            []bufconfig.ModuleConfig
	moduleDependencies       []bufmodule.ModuleRef
	depModuleKeys            []bufmodule.ModuleKey
	pathToMigratedBufGenYAML map[string]bufconfig.BufGenYAMLFile
	moduleNameToParentFile   map[string]string
	filesToDelete            map[string]struct{}
}

func newMigrator(
	logger *zap.Logger,
	clientProvider bufapi.ClientProvider,
	rootBucket storage.ReadWriteBucket,
	destinationDir string,
) *migrator {
	return &migrator{
		logger:                   logger,
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
		m.logger.Sugar().Warnf("%s is already in v2", bufGenYAMLPath)
		return nil
	}
	if typeConfig := bufGenYAML.GenerateConfig().GenerateTypeConfig(); typeConfig != nil && len(typeConfig.IncludeTypes()) > 0 {
		m.logger.Sugar().Warnf(
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

func (m *migrator) addWorkspace(
	ctx context.Context,
	workspaceDirectory string,
) (retErr error) {
	bufWorkYAML, err := bufconfig.GetBufWorkYAMLFileForPrefix(
		ctx,
		m.rootBucket,
		workspaceDirectory,
	)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%q does not have a workspace configuration file", workspaceDirectory)
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

// both buf.yaml and buf.lock
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
			nil,
			nil,
			nil,
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
	// TODO: transform paths so that they are relative to the new buf.yaml v2 or module root (depending on buf.yaml v2 semantics)
	// Paths include RootToExcludes, IgnorePaths, IgnoreIDOrCategoryToPaths
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
			m.logger.Sugar().Warnf(
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
			breakingRules, err := bufbreaking.RulesForConfig(breakingConfig)
			if err != nil {
				return err
			}
			breakingRuleNames := slicesext.Map(breakingRules, func(rule bufcheck.Rule) string { return rule.ID() })
			breakingConfigForRoot := bufconfig.NewBreakingConfig(
				bufconfig.NewCheckConfig(
					bufconfig.FileVersionV2,
					breakingRuleNames,
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
			configForRoot, err := bufconfig.NewModuleConfig(
				dirPathRelativeToDestination,
				moduleFullName,
				// TODO: make them relative to what they should be relative to.
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
	case bufconfig.FileVersionV2:
		m.logger.Sugar().Warnf("%s is already at v2", bufYAMLPath)
		return nil
	default:
		return syserror.Newf("unexpected version: %v", bufYAML.FileVersion())
	}
	m.filesToDelete[bufYAMLPath] = struct{}{}
	bufLockFile, err := bufconfig.GetBufLockFileForPrefix(
		ctx,
		m.rootBucket,
		moduleDir,
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
	return nil
}

func (m *migrator) migrateAsDryRun(
	ctx context.Context,
	writer io.Writer,
) (retErr error) {
	fmt.Fprintf(
		writer,
		"In an actual run, these files will be removed:\n%s\n\nThe following files will be overwritten or created:\n\n",
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
		fmt.Fprintf(
			writer,
			"%s:\n%s\n",
			filepath.Join(m.destinationDir, bufconfig.DefaultBufYAMLFileName),
			bufYAMLBuffer.String(),
		)
		var bufLockBuffer bytes.Buffer
		if err := bufconfig.WriteBufLockFile(&bufLockBuffer, migratedBufLock); err != nil {
			return err
		}
		fmt.Fprintf(
			writer,
			"%s:\n%s\n",
			filepath.Join(m.destinationDir, bufconfig.DefaultBufLockFileName),
			bufLockBuffer.String(),
		)
	}
	for bufGenYAMLPath, migratedBufGenYAML := range m.pathToMigratedBufGenYAML {
		var bufGenYAMLBuffer bytes.Buffer
		if err := bufconfig.WriteBufGenYAMLFile(&bufGenYAMLBuffer, migratedBufGenYAML); err != nil {
			return err
		}
		fmt.Fprintf(
			writer,
			"%s will be written:\n%s\n",
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
		if err := bufconfig.PutBufLockFileForPrefix(
			ctx,
			m.rootBucket,
			m.destinationDir,
			migratedBufLock,
		); err != nil {
			return err
		}
	}
	return nil
}

func (m *migrator) buildBufYAMLAndBufLock(
	ctx context.Context,
) (bufconfig.BufYAMLFile, bufconfig.BufLockFile, error) {
	// Remove declared dependencies that are also modules in this workspace.
	filteredModuleDependencies := slicesext.Filter(
		m.moduleDependencies,
		func(moduleRef bufmodule.ModuleRef) bool {
			_, ok := m.moduleNameToParentFile[moduleRef.ModuleFullName().String()]
			return !ok
		},
	)
	// TODO: the next two variables have placeholder values. Inside the if-statement
	// is what should happen.
	moduleRefs := slicesext.MapValuesToSlice(
		slicesext.ToValuesMap(
			filteredModuleDependencies,
			func(moduleRef bufmodule.ModuleRef) string {
				return moduleRef.ModuleFullName().String()
			},
		),
	)
	moduleKeys := slicesext.MapValuesToSlice(
		slicesext.ToValuesMap(
			m.depModuleKeys,
			func(moduleKey bufmodule.ModuleKey) string {
				return moduleKey.ModuleFullName().String()
			},
		),
	)
	// TODO: use this logic when it's tested and when commit service is ready.
	if false {
		moduleToRefToCommit, err := getModuleToRefToCommit(ctx, filteredModuleDependencies, m.clientProvider)
		if err != nil {
			return nil, nil, err
		}
		commitIDToCommit, err := getCommitIDToCommit(ctx, m.clientProvider, m.depModuleKeys)
		if err != nil {
			return nil, nil, err
		}
		moduleRefs, moduleKeys, err = resolvedDeclaredAndLockedDependencies(
			moduleToRefToCommit,
			commitIDToCommit,
			filteredModuleDependencies,
			m.depModuleKeys,
		)
		if err != nil {
			return nil, nil, err
		}
	}
	bufYAML, err := bufconfig.NewBufYAMLFile(
		bufconfig.FileVersionV2,
		m.moduleConfigs,
		moduleRefs,
	)
	if err != nil {
		return nil, nil, err
	}
	bufLock, err := bufconfig.NewBufLockFile(
		bufconfig.FileVersionV2,
		moduleKeys,
	)
	if err != nil {
		return nil, nil, err
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

func resolvedDeclaredAndLockedDependencies(
	moduleToRefToCommit map[string]map[string]*modulev1beta1.Commit,
	commitIDToCommit map[string]*modulev1beta1.Commit,
	declaredDependencies []bufmodule.ModuleRef,
	lockedDependencies []bufmodule.ModuleKey,
) ([]bufmodule.ModuleRef, []bufmodule.ModuleKey, error) {
	depModuleFullNameToRefs := make(map[string][]bufmodule.ModuleRef)
	for _, depModuleRef := range declaredDependencies {
		moduleFullName := depModuleRef.ModuleFullName().String()
		depModuleFullNameToRefs[moduleFullName] = append(depModuleFullNameToRefs[moduleFullName], depModuleRef)
	}
	depModuleFullNameToResolvedRef := make(map[string]bufmodule.ModuleRef)
	for moduleFullName, refs := range depModuleFullNameToRefs {
		nonEmptyRef := slicesext.Filter(
			refs,
			func(ref bufmodule.ModuleRef) bool {
				return ref.Ref() != ""
			},
		)
		switch len(nonEmptyRef) {
		case 0:
			// All refs are empty, we take the first one (they are all the same). refs is guaranteed not empty,
			// by the construction of depModuleFullNameToRefs.
			depModuleFullNameToResolvedRef[moduleFullName] = refs[0]
		default:
			// There are multiple pinned versions of the same dependency, we use the latest one.
			sort.Slice(nonEmptyRef, func(i, j int) bool {
				refToCommit := moduleToRefToCommit[moduleFullName]
				iTime := refToCommit[refs[i].Ref()].GetCreateTime().AsTime()
				jTime := refToCommit[refs[j].Ref()].GetCreateTime().AsTime()
				return iTime.After(jTime)
			})
			depModuleFullNameToResolvedRef[moduleFullName] = nonEmptyRef[0]
		}
	}
	// We only want locked dependencies that correspond to declared dependencies.
	lockedDependencies = slicesext.Filter(
		lockedDependencies,
		func(lockedDependency bufmodule.ModuleKey) bool {
			_, ok := depModuleFullNameToRefs[lockedDependency.ModuleFullName().String()]
			return ok
		},
	)
	depModuleFullNameToModuleKeys := make(map[string][]bufmodule.ModuleKey)
	for _, depModuleKey := range lockedDependencies {
		depModuleFullName := depModuleKey.ModuleFullName().String()
		depModuleFullNameToModuleKeys[depModuleFullName] = append(depModuleFullNameToModuleKeys[depModuleFullName], depModuleKey)
	}
	resolvedDepModuleKeys := make([]bufmodule.ModuleKey, 0, len(depModuleFullNameToModuleKeys))
	for moduleFullName, depModuleKeys := range depModuleFullNameToModuleKeys {
		resolvedRef := depModuleFullNameToResolvedRef[moduleFullName]
		if resolvedRef.Ref() != "" {
			// TODO: do we want to do all this? It feels like we are doing a partial `buf mod update`.
			// More specifically, it's possible that the declared dependency pin does not have a corresponding
			// lock entry. For example, the buf.lock might not exist.
			// Might as well do this for all lock entries.
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
		sort.Slice(depModuleKeys, func(i, j int) bool {
			iTime := commitIDToCommit[depModuleKeys[i].CommitID()].GetCreateTime().AsTime()
			jTime := commitIDToCommit[depModuleKeys[j].CommitID()].GetCreateTime().AsTime()
			return iTime.After(jTime)
		})
		resolvedDepModuleKeys = append(resolvedDepModuleKeys, depModuleKeys[0])
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
	moduleRefs []bufmodule.ModuleRef,
	clientProvider bufapi.ClientProvider,
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
