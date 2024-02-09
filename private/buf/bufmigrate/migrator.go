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
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/gofrs/uuid/v5"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type migrator struct {
	logger         *zap.Logger
	dryRunWriter   io.Writer
	clientProvider bufapi.ClientProvider
	commitProvider bufmodule.CommitProvider
}

func newMigrator(
	logger *zap.Logger,
	dryRunWriter io.Writer,
	clientProvider bufapi.ClientProvider,
	commitProvider bufmodule.CommitProvider,
) *migrator {
	return &migrator{
		logger:         logger,
		dryRunWriter:   dryRunWriter,
		clientProvider: clientProvider,
		commitProvider: commitProvider,
	}
}

func (m *migrator) Migrate(
	ctx context.Context,
	bucket storage.ReadWriteBucket,
	workspaceDirPaths []string,
	moduleDirPaths []string,
	generateTemplatePaths []string,
	options ...MigrateOption,
) (retErr error) {
	migrateOptions := newMigrateOptions()
	for _, option := range options {
		option(migrateOptions)
	}
	if len(workspaceDirPaths) == 0 && len(moduleDirPaths) == 0 && len(generateTemplatePaths) == 0 {
		return errors.New("no directory or file specified")
	}
	// Directories cannot jump context because in the migrated buf.yaml v2, each
	// directory path cannot jump context. I.e. it's not valid to have `- directory: ..`
	// in a buf.yaml v2.
	workspaceDirPaths, err := slicesext.MapError(workspaceDirPaths, normalpath.NormalizeAndValidate)
	if err != nil {
		return err
	}
	moduleDirPaths, err = slicesext.MapError(moduleDirPaths, normalpath.NormalizeAndValidate)
	if err != nil {
		return err
	}
	// This does mean that buf.gen.yamls need to be under the directory this is run at, but this is OK.
	generateTemplatePaths, err = slicesext.MapError(generateTemplatePaths, normalpath.NormalizeAndValidate)
	if err != nil {
		return err
	}
	// the directory where the migrated buf.yaml live, this is useful for computing
	// module directory paths, and possibly other paths.
	destinationDirPath := "."
	if len(workspaceDirPaths) == 1 && len(moduleDirPaths) == 0 {
		destinationDirPath = workspaceDirPaths[0]
	}
	migrateBuilder := newMigrateBuilder(
		m.logger,
		m.commitProvider,
		bucket,
		destinationDirPath,
	)
	for _, workspaceDirPath := range workspaceDirPaths {
		if err := migrateBuilder.addWorkspaceDirectory(
			ctx,
			workspaceDirPath,
		); err != nil {
			return err
		}
	}
	for _, bufYAMLPath := range moduleDirPaths {
		// TODO: read upwards to make sure it's not in a workspace.
		// i.e. for ./foo/bar/buf.yaml, check none of "./foo", ".", "../", "../..", and etc. is a workspace.
		// The logic for this is in getMapPathAndSubDirPath from buffetch/internal
		if err := migrateBuilder.addModuleDirectory(
			ctx,
			bufYAMLPath,
		); err != nil {
			return err
		}
	}
	for _, bufGenYAMLPath := range generateTemplatePaths {
		if err := migrateBuilder.addBufGenYAML(bufGenYAMLPath); err != nil {
			return err
		}
	}
	if migrateOptions.dryRun {
		return m.migrateAsDryRun(ctx, migrateBuilder)
	}
	return m.migrate(ctx, migrateBuilder)
}

func (m *migrator) migrateAsDryRun(ctx context.Context, migrateBuilder *migrateBuilder) (retErr error) {
	if len(migrateBuilder.filePathsToDelete) > 0 {
		m.dryRunPrintf(
			"In an actual run, these files will be removed:\n%s\n\nThe following files will be overwritten or created:\n",
			strings.Join(slicesext.MapKeysToSortedSlice(migrateBuilder.filePathsToDelete), "\n"),
		)
	} else {
		m.dryRunPrintln("In an actual run:")
	}
	// We create a buf.yaml if we have seen visited any module directory. Note
	// we add a module config even for a module directory without a buf.yaml.
	if len(migrateBuilder.moduleConfigs) > 0 {
		migratedBufYAML, migratedBufLock, err := m.buildBufYAMLAndBufLock(ctx, migrateBuilder)
		if err != nil {
			return err
		}
		m.dryRunPrintf(
			"%s will be written:\n",
			filepath.Join(migrateBuilder.destinationDirPath, bufconfig.DefaultBufWorkYAMLFileName),
		)
		if err := bufconfig.WriteBufYAMLFile(m.dryRunWriter, migratedBufYAML); err != nil {
			return err
		}
		if migratedBufLock != nil {
			m.dryRunPrintf(
				"%s will be written:\n",
				filepath.Join(migrateBuilder.destinationDirPath, bufconfig.DefaultBufLockFileName),
			)
			if err := bufconfig.WriteBufLockFile(m.dryRunWriter, migratedBufLock); err != nil {
				return err
			}
		}
	}
	for bufGenYAMLPath, migratedBufGenYAML := range migrateBuilder.pathToMigratedBufGenYAML {
		m.dryRunPrintf(
			"%s will be written:\n",
			bufGenYAMLPath,
		)
		if err := bufconfig.WriteBufGenYAMLFile(m.dryRunWriter, migratedBufGenYAML); err != nil {
			return err
		}
	}
	return nil
}

func (m *migrator) migrate(ctx context.Context, migrateBuilder *migrateBuilder) (retErr error) {
	for bufGenYAMLPath, migratedBufGenYAML := range migrateBuilder.pathToMigratedBufGenYAML {
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
	if len(migrateBuilder.moduleConfigs) > 0 {
		migratedBufYAML, migratedBufLock, err := m.buildBufYAMLAndBufLock(ctx, migrateBuilder)
		if err != nil {
			return err
		}
		for _, fileToDelete := range slicesext.MapKeysToSortedSlice(migrateBuilder.filePathsToDelete) {
			if err := os.Remove(fileToDelete); err != nil {
				return err
			}
		}
		if err := bufconfig.PutBufYAMLFileForPrefix(
			ctx,
			migrateBuilder.rootBucket,
			migrateBuilder.destinationDirPath,
			migratedBufYAML,
		); err != nil {
			return err
		}
		if migratedBufLock != nil {
			if err := bufconfig.PutBufLockFileForPrefix(
				ctx,
				migrateBuilder.rootBucket,
				migrateBuilder.destinationDirPath,
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
	migrateBuilder *migrateBuilder,
) (bufconfig.BufYAMLFile, bufconfig.BufLockFile, error) {
	// module full name --> the list of declared dependencies that are this module.
	depModuleToDeclaredRefs := make(map[string][]bufmodule.ModuleRef)
	for _, declaredRef := range migrateBuilder.moduleDependencies {
		moduleFullName := declaredRef.ModuleFullName().String()
		// If a declared dependency also shows up in the workspace, it's not a dependency.
		if _, ok := migrateBuilder.moduleNameToParentFile[moduleFullName]; ok {
			continue
		}
		depModuleToDeclaredRefs[moduleFullName] = append(depModuleToDeclaredRefs[moduleFullName], declaredRef)
	}
	// module full name --> the list of lock entries that are this module.
	depModuleToLockEntries := make(map[string][]bufmodule.ModuleKey)
	for _, lockEntry := range migrateBuilder.depModuleKeys {
		moduleFullName := lockEntry.ModuleFullName().String()
		// If a declared dependency also shows up in the workspace, it's not a dependency.
		//
		// We are only removing lock entries that are in the workspace. A lock entry
		// could be for an indirect dependenceny not listed in deps in any buf.yaml.
		if _, ok := migrateBuilder.moduleNameToParentFile[moduleFullName]; ok {
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
			func(moduleKey bufmodule.ModuleKey) (uuid.UUID, error) {
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
			migrateBuilder.moduleConfigs,
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
		if migrateBuilder.hasSeenBufLock {
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
	// TODO: This code should be reworked to use bufmodule.CommitProvider and bufmodule.ModuleKeyProvider
	moduleToRefToCommit, err := getModuleToRefToCommit(ctx, m.clientProvider, migrateBuilder.moduleDependencies)
	if err != nil {
		return nil, nil, err
	}
	commitIDToCommit, err := getCommitIDToCommit(ctx, m.clientProvider, migrateBuilder.depModuleKeys)
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
		migrateBuilder.moduleConfigs,
		resolvedDepModuleRefs,
	)
	if err != nil {
		return nil, nil, err
	}
	// TODO: We need to upgrade digests from b4 to b5, right?
	var bufLock bufconfig.BufLockFile
	if migrateBuilder.hasSeenBufLock {
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

func (m *migrator) dryRunPrintf(format string, args ...any) {
	_, _ = m.dryRunWriter.Write([]byte(fmt.Sprintf(format, args...)))
}

func (m *migrator) dryRunPrintln(message string) {
	_, _ = m.dryRunWriter.Write([]byte(message + "\n"))
}

func resolvedDeclaredAndLockedDependencies(
	moduleToRefToCommit map[string]map[string]*modulev1beta1.Commit,
	commitIDToCommit map[uuid.UUID]*modulev1beta1.Commit,
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
			commitID, err := uuid.FromString(resolvedCommit.GetId())
			if err != nil {
				return nil, nil, err
			}
			key, err := bufmodule.NewModuleKey(
				resolvedRef.ModuleFullName(),
				commitID,
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
) (map[uuid.UUID]*modulev1beta1.Commit, error) {
	commitIDToCommit := make(map[uuid.UUID]*modulev1beta1.Commit)
	for _, moduleKey := range moduleKeys {
		moduleFullName := moduleKey.ModuleFullName()
		response, err := clientProvider.CommitServiceClient(moduleFullName.Registry()).GetCommits(
			ctx,
			connect.NewRequest(
				&modulev1beta1.GetCommitsRequest{
					ResourceRefs: []*modulev1beta1.ResourceRef{
						{
							Value: &modulev1beta1.ResourceRef_Id{
								Id: moduleKey.CommitID().String(),
							},
						},
					},
				},
			),
		)
		if err != nil {
			if connect.CodeOf(err) == connect.CodeNotFound {
				return nil, &fs.PathError{Op: "read", Path: moduleKey.CommitID().String(), Err: fs.ErrNotExist}
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

type migrateOptions struct {
	dryRun bool
}

func newMigrateOptions() *migrateOptions {
	return &migrateOptions{}
}
