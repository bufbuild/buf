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
	"io"
	"io/fs"
	"sort"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufcheck"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufbreaking"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/buflint"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/gofrs/uuid/v5"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type migrator struct {
	logger            *zap.Logger
	runner            command.Runner
	moduleKeyProvider bufmodule.ModuleKeyProvider
	commitProvider    bufmodule.CommitProvider
}

func newMigrator(
	logger *zap.Logger,
	runner command.Runner,
	moduleKeyProvider bufmodule.ModuleKeyProvider,
	commitProvider bufmodule.CommitProvider,
) *migrator {
	return &migrator{
		logger:            logger,
		runner:            runner,
		moduleKeyProvider: moduleKeyProvider,
		commitProvider:    commitProvider,
	}
}

func (m *migrator) Migrate(
	ctx context.Context,
	bucket storage.ReadWriteBucket,
	workspaceDirPaths []string,
	moduleDirPaths []string,
	bufGenYAMLFilePaths []string,
) error {
	m.logPaths(workspaceDirPaths, moduleDirPaths, bufGenYAMLFilePaths)
	migrateBuilder, err := m.getMigrateBuilder(ctx, bucket, workspaceDirPaths, moduleDirPaths, bufGenYAMLFilePaths)
	if err != nil {
		return err
	}
	return m.migrate(ctx, bucket, migrateBuilder)
}

func (m *migrator) Diff(
	ctx context.Context,
	bucket storage.ReadBucket,
	writer io.Writer,
	workspaceDirPaths []string,
	moduleDirPaths []string,
	bufGenYAMLFilePaths []string,
) error {
	m.logPaths(workspaceDirPaths, moduleDirPaths, bufGenYAMLFilePaths)
	migrateBuilder, err := m.getMigrateBuilder(ctx, bucket, workspaceDirPaths, moduleDirPaths, bufGenYAMLFilePaths)
	if err != nil {
		return err
	}
	return m.diff(ctx, writer, migrateBuilder)
}

func (m *migrator) logPaths(
	workspaceDirPaths []string,
	moduleDirPaths []string,
	bufGenYAMLFilePaths []string,
) {
	if len(workspaceDirPaths) > 0 {
		m.logger.Sugar().Debugf("workspace directory paths:\n%s", strings.Join(workspaceDirPaths, "\n"))
	}
	if len(moduleDirPaths) > 0 {
		m.logger.Sugar().Debugf("module directory paths:\n%s", strings.Join(moduleDirPaths, "\n"))
	}
	if len(bufGenYAMLFilePaths) > 0 {
		m.logger.Sugar().Debugf("buf.gen.yaml file paths:\n%s", strings.Join(bufGenYAMLFilePaths, "\n"))
	}
}

func (m *migrator) getMigrateBuilder(
	ctx context.Context,
	bucket storage.ReadBucket,
	workspaceDirPaths []string,
	moduleDirPaths []string,
	bufGenYAMLFilePaths []string,
) (*migrateBuilder, error) {
	if len(workspaceDirPaths) == 0 && len(moduleDirPaths) == 0 && len(bufGenYAMLFilePaths) == 0 {
		return nil, errors.New("no directory or file specified")
	}
	// Directories cannot jump context because in the migrated buf.yaml v2, each
	// directory path cannot jump context. I.e. it's not valid to have `- path: ..`
	// in a buf.yaml v2.
	workspaceDirPaths, err := slicesext.MapError(workspaceDirPaths, normalpath.NormalizeAndValidate)
	if err != nil {
		return nil, err
	}
	moduleDirPaths, err = slicesext.MapError(moduleDirPaths, normalpath.NormalizeAndValidate)
	if err != nil {
		return nil, err
	}
	// This does mean that buf.gen.yamls need to be under the directory this is run at, but this is OK.
	bufGenYAMLFilePaths, err = slicesext.MapError(bufGenYAMLFilePaths, normalpath.NormalizeAndValidate)
	if err != nil {
		return nil, err
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
		if err := migrateBuilder.addWorkspace(ctx, workspaceDirPath); err != nil {
			return nil, err
		}
	}
	for _, moduleDirPath := range moduleDirPaths {
		// TODO FUTURE: read upwards to make sure it's not in a workspace.
		// i.e. for ./foo/bar/buf.yaml, check none of "./foo", ".", "../", "../..", and etc. is a workspace.
		// The logic for this is in getMapPathAndSubDirPath from buffetch/internal
		if err := migrateBuilder.addModule(ctx, moduleDirPath); err != nil {
			return nil, err
		}
	}
	for _, bufGenYAMLFilePath := range bufGenYAMLFilePaths {
		if err := migrateBuilder.addBufGenYAML(ctx, bufGenYAMLFilePath); err != nil {
			return nil, err
		}
	}
	return migrateBuilder, nil
}

func (m *migrator) migrate(ctx context.Context, bucket storage.WriteBucket, migrateBuilder *migrateBuilder) (retErr error) {
	for _, path := range slicesext.MapKeysToSortedSlice(migrateBuilder.pathsToDelete) {
		if err := bucket.Delete(ctx, path); err != nil {
			return err
		}
	}
	for path, migratedBufGenYAMLFile := range migrateBuilder.pathToMigratedBufGenYAMLFile {
		if err := storage.ForWriteObject(ctx, bucket, path, func(writeObject storage.WriteObject) error {
			return bufconfig.WriteBufGenYAMLFile(writeObject, migratedBufGenYAMLFile)
		}); err != nil {
			return err
		}
	}
	// We create a buf.yaml if we have seen visited any module directory. Note
	// we add a module config even for a module directory without a buf.yaml.
	if len(migrateBuilder.moduleConfigs) > 0 {
		migratedBufYAMLFile, migratedBufLockFile, err := m.buildBufYAMLAndBufLockFiles(ctx, migrateBuilder)
		if err != nil {
			return err
		}
		if err := bufconfig.PutBufYAMLFileForPrefix(ctx, bucket, migrateBuilder.destinationDirPath, migratedBufYAMLFile); err != nil {
			return err
		}
		if migratedBufLockFile != nil {
			if err := bufconfig.PutBufLockFileForPrefix(ctx, bucket, migrateBuilder.destinationDirPath, migratedBufLockFile); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *migrator) diff(
	ctx context.Context,
	writer io.Writer,
	migrateBuilder *migrateBuilder,
) (retErr error) {
	originalFileBucket, addedFileBucket, err := m.getOriginalAndAddedFileBuckets(ctx, migrateBuilder)
	if err != nil {
		return err
	}
	return storage.Diff(
		ctx,
		m.runner,
		writer,
		originalFileBucket,
		addedFileBucket,
	)
}

// Doing this as a separate function so we can use defer for WriteObjectCloser.Close.
// Files are not written until Close is called.
func (m *migrator) getOriginalAndAddedFileBuckets(
	ctx context.Context,
	migrateBuilder *migrateBuilder,
) (_ storage.ReadBucket, _ storage.ReadBucket, retErr error) {
	// Contains the original files before modification. Includes both the deleted files
	// and the originals of any files that were added.
	originalFileBucket := storagemem.NewReadWriteBucket()
	// Contains the added files
	addedFileBucket := storagemem.NewReadWriteBucket()
	for pathToDelete := range migrateBuilder.pathsToDelete {
		if err := storage.CopyPath(
			ctx,
			migrateBuilder.bucket,
			pathToDelete,
			originalFileBucket,
			pathToDelete,
		); err != nil {
			return nil, nil, err
		}
	}
	// We create a buf.yaml if we have seen visited any module directory. Note
	// we add a module config even for a module directory without a buf.yaml.
	if len(migrateBuilder.moduleConfigs) > 0 {
		migratedBufYAMLFile, migratedBufLockFile, err := m.buildBufYAMLAndBufLockFiles(ctx, migrateBuilder)
		if err != nil {
			return nil, nil, err
		}
		migratedBufYAMLFilePath := normalpath.Join(migrateBuilder.destinationDirPath, bufconfig.DefaultBufYAMLFileName)
		if err := storage.CopyPath(
			ctx,
			migrateBuilder.bucket,
			migratedBufYAMLFilePath,
			originalFileBucket,
			migratedBufYAMLFilePath,
		); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return nil, nil, err
			}
		}
		writeObjectCloser, err := addedFileBucket.Put(ctx, migratedBufYAMLFilePath)
		if err != nil {
			return nil, nil, err
		}
		defer func() {
			retErr = multierr.Append(retErr, writeObjectCloser.Close())
		}()
		if err := bufconfig.WriteBufYAMLFile(writeObjectCloser, migratedBufYAMLFile); err != nil {
			return nil, nil, err
		}
		if migratedBufLockFile != nil {
			migratedBufLockFilePath := normalpath.Join(migrateBuilder.destinationDirPath, bufconfig.DefaultBufLockFileName)
			if err := storage.CopyPath(
				ctx,
				migrateBuilder.bucket,
				migratedBufLockFilePath,
				originalFileBucket,
				migratedBufLockFilePath,
			); err != nil {
				if !errors.Is(err, fs.ErrNotExist) {
					return nil, nil, err
				}
			}
			writeObjectCloser, err := addedFileBucket.Put(ctx, migratedBufLockFilePath)
			if err != nil {
				return nil, nil, err
			}
			defer func() {
				retErr = multierr.Append(retErr, writeObjectCloser.Close())
			}()
			if err := bufconfig.WriteBufLockFile(writeObjectCloser, migratedBufLockFile); err != nil {
				return nil, nil, err
			}
		}
	}
	for bufGenYAMLPath, migratedBufGenYAMLFile := range migrateBuilder.pathToMigratedBufGenYAMLFile {
		if err := storage.CopyPath(
			ctx,
			migrateBuilder.bucket,
			bufGenYAMLPath,
			originalFileBucket,
			bufGenYAMLPath,
		); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return nil, nil, err
			}
		}
		writeObjectCloser, err := addedFileBucket.Put(ctx, bufGenYAMLPath)
		if err != nil {
			return nil, nil, err
		}
		defer func() {
			retErr = multierr.Append(retErr, writeObjectCloser.Close())
		}()
		if err := bufconfig.WriteBufGenYAMLFile(writeObjectCloser, migratedBufGenYAMLFile); err != nil {
			return nil, nil, err
		}
	}
	return originalFileBucket, addedFileBucket, nil
}

// If this function doesn't return an error, the BufYAMLFile returned is never nil,
// but the BufLockFile returned may be nil.
func (m *migrator) buildBufYAMLAndBufLockFiles(
	ctx context.Context,
	migrateBuilder *migrateBuilder,
) (bufconfig.BufYAMLFile, bufconfig.BufLockFile, error) {
	// module full name --> the list of declared dependencies that are this module.
	depModuleToDeclaredRefs := make(map[string][]bufmodule.ModuleRef)
	for _, declaredRef := range migrateBuilder.configuredDepModuleRefs {
		moduleFullName := declaredRef.ModuleFullName().String()
		// If a declared dependency also shows up in the workspace, it's not a dependency.
		if _, ok := migrateBuilder.moduleFullNameStringToParentPath[moduleFullName]; ok {
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
		if _, ok := migrateBuilder.moduleFullNameStringToParentPath[moduleFullName]; ok {
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
		commitIDToKey := make(map[uuid.UUID]bufmodule.ModuleKey)
		for _, lockEntry := range lockEntries {
			// There may be duplicates, we ignore this. They should be the same.
			// We could check,
			commitIDToKey[lockEntry.CommitID()] = lockEntry
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
		if migrateBuilder.hasSeenBufLockFile {
			resolvedLockEntries, err := m.upgradeModuleKeysToB5(ctx, resolvedLockEntries)
			if err != nil {
				return nil, nil, err
			}
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
	moduleToRefToCommit, err := m.getModuleToRefToCommit(ctx, migrateBuilder.configuredDepModuleRefs)
	if err != nil {
		return nil, nil, err
	}
	commitIDToCommit, err := m.getCommitIDToCommit(ctx, migrateBuilder.depModuleKeys)
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
	var bufLock bufconfig.BufLockFile
	if migrateBuilder.hasSeenBufLockFile {
		resolvedDepModuleKeys, err := m.upgradeModuleKeysToB5(ctx, resolvedDepModuleKeys)
		if err != nil {
			return nil, nil, err
		}
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

func (m *migrator) getModuleToRefToCommit(
	ctx context.Context,
	moduleRefs []bufmodule.ModuleRef,
) (map[string]map[string]bufmodule.Commit, error) {
	// The module refs that are collected by migrateBuilder is across all modules being
	// migrated, so there may be duplicates. ModuleKeyProvider errors on duplicate module
	// refs because it is expensive to make multiple calls to resolve the same module ref,
	// so we deduplicate the module refs we are passing here.
	moduleRefs = slicesext.DeduplicateAny(
		moduleRefs,
		func(moduleRef bufmodule.ModuleRef) string { return moduleRef.String() },
	)
	moduleKeys, err := m.moduleKeyProvider.GetModuleKeysForModuleRefs(ctx, moduleRefs, bufmodule.DigestTypeB5)
	if err != nil {
		return nil, err
	}
	commits, err := m.commitProvider.GetCommitsForModuleKeys(ctx, moduleKeys)
	if err != nil {
		return nil, err
	}
	moduleToRefToCommit := make(map[string]map[string]bufmodule.Commit)
	for i, moduleRef := range moduleRefs {
		if moduleRef.Ref() == "" {
			continue
		}
		// We know that that the ModuleKeys and Commits match up with the ModuleRefs via the definition
		// of GetModuleKeysForModuleRefs and GetCommitsForModuleKeys.
		commit := commits[i]

		moduleFullName := moduleRef.ModuleFullName()
		if moduleToRefToCommit[moduleFullName.String()] == nil {
			moduleToRefToCommit[moduleFullName.String()] = make(map[string]bufmodule.Commit)
		}
		moduleToRefToCommit[moduleFullName.String()][moduleRef.Ref()] = commit
	}
	return moduleToRefToCommit, nil
}

func (m *migrator) getCommitIDToCommit(
	ctx context.Context,
	moduleKeys []bufmodule.ModuleKey,
) (map[uuid.UUID]bufmodule.Commit, error) {
	// The module keys that are collected by migrateBuilder is across all modules being
	// migrated, so there may be duplicates. CommitProvider errors on duplicate module
	// keys because it is expensive to make multiple calls to resolve the same module key,
	// so we deduplicate the module keys we are passing here.
	moduleKeys = slicesext.DeduplicateAny(
		moduleKeys,
		func(moduleKey bufmodule.ModuleKey) string { return moduleKey.String() },
	)
	commits, err := m.commitProvider.GetCommitsForModuleKeys(ctx, moduleKeys)
	if err != nil {
		return nil, err
	}
	commitIDToCommit := make(map[uuid.UUID]bufmodule.Commit, len(commits))
	for _, commit := range commits {
		// We don't know if these are unique, so we do not use slicesext.ToUniqueValuesMapError.
		commitIDToCommit[commit.ModuleKey().CommitID()] = commit
	}
	return commitIDToCommit, nil
}

func (m *migrator) upgradeModuleKeysToB5(
	ctx context.Context,
	moduleKeys []bufmodule.ModuleKey,
) ([]bufmodule.ModuleKey, error) {
	moduleKeys = slicesext.Copy(moduleKeys)

	b4IndexedModuleKeys, err := slicesext.FilterError(
		slicesext.ToIndexed(moduleKeys),
		func(indexedModuleKey slicesext.Indexed[bufmodule.ModuleKey]) (bool, error) {
			digest, err := indexedModuleKey.Value.Digest()
			if err != nil {
				return false, err
			}
			return digest.Type() == bufmodule.DigestTypeB4, nil
		},
	)
	if err != nil {
		return nil, err
	}

	if len(b4IndexedModuleKeys) == 0 {
		return moduleKeys, nil
	}

	commitKeys, err := slicesext.MapError(
		b4IndexedModuleKeys,
		func(indexedModuleKey slicesext.Indexed[bufmodule.ModuleKey]) (bufmodule.CommitKey, error) {
			return bufmodule.NewCommitKey(
				indexedModuleKey.Value.ModuleFullName().Registry(),
				indexedModuleKey.Value.CommitID(),
				bufmodule.DigestTypeB5,
			)
		},
	)
	if err != nil {
		return nil, err
	}
	commits, err := m.commitProvider.GetCommitsForCommitKeys(ctx, commitKeys)
	if err != nil {
		return nil, err
	}
	for i, commit := range commits {
		// The index into moduleKeys.
		moduleKeyIndex := b4IndexedModuleKeys[i].Index
		existingModuleKey := moduleKeys[moduleKeyIndex]
		newModuleKey, err := bufmodule.NewModuleKey(
			existingModuleKey.ModuleFullName(),
			existingModuleKey.CommitID(),
			commit.ModuleKey().Digest,
		)
		if err != nil {
			return nil, err
		}
		moduleKeys[moduleKeyIndex] = newModuleKey
	}
	return moduleKeys, nil
}

func resolvedDeclaredAndLockedDependencies(
	moduleToRefToCommit map[string]map[string]bufmodule.Commit,
	commitIDToCommit map[uuid.UUID]bufmodule.Commit,
	moduleFullNameToDeclaredRefs map[string][]bufmodule.ModuleRef,
	moduleFullNameToLockKeys map[string][]bufmodule.ModuleKey,
) ([]bufmodule.ModuleRef, []bufmodule.ModuleKey, error) {
	depModuleFullNameToResolvedRef := make(map[string]bufmodule.ModuleRef)
	for moduleFullName, refs := range moduleFullNameToDeclaredRefs {
		var errs []error
		// There are multiple pinned versions of the same dependency, we use the latest one.
		sort.Slice(refs, func(i, j int) bool {
			refToCommit := moduleToRefToCommit[moduleFullName]
			iTime, err := refToCommit[refs[i].Ref()].CreateTime()
			if err != nil {
				errs = append(errs, err)
			}
			jTime, err := refToCommit[refs[j].Ref()].CreateTime()
			if err != nil {
				errs = append(errs, err)
			}
			return iTime.After(jTime)
		})
		if len(errs) > 0 {
			return nil, nil, multierr.Combine(errs...)
		}
		depModuleFullNameToResolvedRef[moduleFullName] = refs[0]
	}
	resolvedDepModuleKeys := make([]bufmodule.ModuleKey, 0, len(moduleFullNameToLockKeys))
	for moduleFullName, lockKeys := range moduleFullNameToLockKeys {
		resolvedRef, ok := depModuleFullNameToResolvedRef[moduleFullName]
		if ok && resolvedRef.Ref() != "" {
			// If we have already picked a pinned dependency ref for this dependency,
			// we use that as the lock entry as well.
			resolvedCommit := moduleToRefToCommit[moduleFullName][resolvedRef.Ref()]
			resolvedDepModuleKeys = append(resolvedDepModuleKeys, resolvedCommit.ModuleKey())
			continue
		}
		var errs []error
		// Otherwise, we pick the latest key from the buf.locks we have read.
		sort.Slice(lockKeys, func(i, j int) bool {
			iTime, err := commitIDToCommit[lockKeys[i].CommitID()].CreateTime()
			if err != nil {
				errs = append(errs, err)
			}
			jTime, err := commitIDToCommit[lockKeys[j].CommitID()].CreateTime()
			if err != nil {
				errs = append(errs, err)
			}
			return iTime.After(jTime)
		})
		if len(errs) > 0 {
			return nil, nil, multierr.Combine(errs...)
		}
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

func equivalentLintConfigInV2(lintConfig bufconfig.LintConfig) (bufconfig.LintConfig, error) {
	deprecations, err := buflint.GetRelevantDeprecations(lintConfig.FileVersion())
	if err != nil {
		return nil, err
	}
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
		deprecations,
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
	deprecations, err := bufbreaking.GetRelevantDeprecations(breakingConfig.FileVersion())
	if err != nil {
		return nil, err
	}
	equivalentCheckConfigV2, err := equivalentCheckConfigInV2(
		breakingConfig,
		func(checkConfig bufconfig.CheckConfig) ([]bufcheck.Rule, error) {
			breakingConfig := bufconfig.NewBreakingConfig(
				checkConfig,
				breakingConfig.IgnoreUnstablePackages(),
			)
			return bufbreaking.RulesForConfig(breakingConfig)
		},
		deprecations,
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
	deprecations map[string][]string,
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
	simplyTranslatedCheckConfig, err := bufconfig.NewEnabledCheckConfig(
		bufconfig.FileVersionV2,
		undeprecateSlice(checkConfig.UseIDsAndCategories(), deprecations),
		undeprecateSlice(checkConfig.ExceptIDsAndCategories(), deprecations),
		checkConfig.IgnorePaths(),
		undeprecateMap(checkConfig.IgnoreIDOrCategoryToPaths(), deprecations),
	)
	if err != nil {
		return nil, err
	}
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
	return bufconfig.NewEnabledCheckConfig(
		bufconfig.FileVersionV2,
		append(simplyTranslatedCheckConfig.UseIDsAndCategories(), missingIDs...),
		append(simplyTranslatedCheckConfig.ExceptIDsAndCategories(), extraIDs...),
		simplyTranslatedCheckConfig.IgnorePaths(),
		simplyTranslatedCheckConfig.IgnoreIDOrCategoryToPaths(),
	)
}

// undeprecateSlice transforms the given slice of IDs so that any deprecated
// IDs are replaced with their replacements per the given deprecations.
func undeprecateSlice(ids []string, deprecations map[string][]string) []string {
	newIDs := make([]string, 0, len(ids))
	for _, id := range ids {
		replacements, ok := deprecations[id]
		if ok {
			newIDs = append(newIDs, replacements...)
		} else {
			newIDs = append(newIDs, id)
		}
	}
	return newIDs
}

// undeprecateMap transforms the given map of IDs to values so that any
// deprecated IDs are replaced with their replacements per the given
// deprecations. When there is more than one replacement, all entries
// for the replacements will have the same value.
func undeprecateMap[T any](idMap map[string]T, deprecations map[string][]string) map[string]T {
	newIDs := make(map[string]T, len(idMap))
	for id, val := range idMap {
		replacements, ok := deprecations[id]
		if ok {
			for _, replacement := range replacements {
				newIDs[replacement] = val
			}
		} else {
			newIDs[id] = val
		}
	}
	return newIDs
}
