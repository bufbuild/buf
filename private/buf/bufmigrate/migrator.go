// Copyright 2020-2025 Buf Technologies, Inc.
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
	"log/slog"
	"slices"
	"sort"
	"strings"

	"buf.build/go/bufplugin/check"
	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufcheckserver"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/google/uuid"
)

type migrator struct {
	logger            *slog.Logger
	moduleKeyProvider bufmodule.ModuleKeyProvider
	commitProvider    bufmodule.CommitProvider
}

func newMigrator(
	logger *slog.Logger,
	moduleKeyProvider bufmodule.ModuleKeyProvider,
	commitProvider bufmodule.CommitProvider,
) *migrator {
	return &migrator{
		logger:            logger,
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
		m.logger.Debug(fmt.Sprintf("workspace directory paths:\n%s", strings.Join(workspaceDirPaths, "\n")))
	}
	if len(moduleDirPaths) > 0 {
		m.logger.Debug(fmt.Sprintf("module directory paths:\n%s", strings.Join(moduleDirPaths, "\n")))
	}
	if len(bufGenYAMLFilePaths) > 0 {
		m.logger.Debug(fmt.Sprintf("buf.gen.yaml file paths:\n%s", strings.Join(bufGenYAMLFilePaths, "\n")))
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
	workspaceDirPaths, err := xslices.MapError(workspaceDirPaths, normalpath.NormalizeAndValidate)
	if err != nil {
		return nil, err
	}
	moduleDirPaths, err = xslices.MapError(moduleDirPaths, normalpath.NormalizeAndValidate)
	if err != nil {
		return nil, err
	}
	// This does mean that buf.gen.yamls need to be under the directory this is run at, but this is OK.
	bufGenYAMLFilePaths, err = xslices.MapError(bufGenYAMLFilePaths, normalpath.NormalizeAndValidate)
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
	for _, path := range xslices.MapKeysToSortedSlice(migrateBuilder.pathsToDelete) {
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
			retErr = errors.Join(retErr, writeObjectCloser.Close())
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
				retErr = errors.Join(retErr, writeObjectCloser.Close())
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
			retErr = errors.Join(retErr, writeObjectCloser.Close())
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
	depModuleToDeclaredRefs := make(map[string][]bufparse.Ref)
	for _, declaredRef := range migrateBuilder.configuredDepModuleRefs {
		moduleFullName := declaredRef.FullName().String()
		// If a declared dependency also shows up in the workspace, it's not a dependency.
		if _, ok := migrateBuilder.moduleFullNameStringToParentPath[moduleFullName]; ok {
			continue
		}
		depModuleToDeclaredRefs[moduleFullName] = append(depModuleToDeclaredRefs[moduleFullName], declaredRef)
	}
	// module full name --> the list of lock entries that are this module.
	depModuleToLockEntries := make(map[string][]bufmodule.ModuleKey)
	for _, lockEntry := range migrateBuilder.depModuleKeys {
		moduleFullName := lockEntry.FullName().String()
		// If a declared dependency also shows up in the workspace, it's not a dependency.
		//
		// We are only removing lock entries that are in the workspace. A lock entry
		// could be for an indirect dependency not listed in deps in any buf.yaml.
		if _, ok := migrateBuilder.moduleFullNameStringToParentPath[moduleFullName]; ok {
			continue
		}
		depModuleToLockEntries[moduleFullName] = append(depModuleToLockEntries[moduleFullName], lockEntry)
	}
	// This will be set to false if the duplicate dependencies cannot be resolved locally.
	areDependenciesResolved := true
	for depModule, declaredRefs := range depModuleToDeclaredRefs {
		refStringToRef := make(map[string]bufparse.Ref)
		for _, ref := range declaredRefs {
			// Add ref even if ref.Ref() is empty. Therefore, xslices.ToValuesMap is not used.
			refStringToRef[ref.Ref()] = ref
		}
		// If there are both buf.build/foo/bar and buf.build/foo/bar:some_ref, the former will
		// not be used.
		if len(refStringToRef) > 1 {
			delete(refStringToRef, "")
		}
		depModuleToDeclaredRefs[depModule] = xslices.MapValuesToSlice(refStringToRef)
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
		depModuleToLockEntries[depModule] = xslices.MapValuesToSlice(commitIDToKey)
		if len(commitIDToKey) > 1 {
			areDependenciesResolved = false
		}
	}
	if areDependenciesResolved {
		resolvedDeclaredRefs := make([]bufparse.Ref, 0, len(depModuleToDeclaredRefs))
		for _, depModuleRefs := range depModuleToDeclaredRefs {
			// depModuleRefs is guaranteed to have length 1, because areDependenciesResolved is true.
			resolvedDeclaredRefs = append(resolvedDeclaredRefs, depModuleRefs...)
		}
		bufYAML, err := bufconfig.NewBufYAMLFile(
			bufconfig.FileVersionV2,
			migrateBuilder.moduleConfigs,
			// TODO: If we ever need to migrate from a v2 to v3, we will need to handle PluginConfigs and PolicyConfigs
			nil,
			nil,
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
				nil, // Plugins are not supported in v1.
				nil, // Policies are not supported in v1.
				nil, // Policy PluginKeys are not supported in v1.
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
		// TODO: If we ever need to migrate from a v2 to v3, we will need to handle PluginConfigs and PolicyConfigs
		nil,
		nil,
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
			nil, // Plugins are not supported in v1.
			nil, // Policies are not supported in v1.
			nil, // Policy PluginKeys are not supported in v1.
		)
		if err != nil {
			return nil, nil, err
		}
	}
	return bufYAML, bufLock, nil
}

func (m *migrator) getModuleToRefToCommit(
	ctx context.Context,
	moduleRefs []bufparse.Ref,
) (map[string]map[string]bufmodule.Commit, error) {
	// The module refs that are collected by migrateBuilder is across all modules being
	// migrated, so there may be duplicates. ModuleKeyProvider errors on duplicate module
	// refs because it is expensive to make multiple calls to resolve the same module ref,
	// so we deduplicate the module refs we are passing here.
	moduleRefs = xslices.DeduplicateAny(
		moduleRefs,
		func(moduleRef bufparse.Ref) string { return moduleRef.String() },
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

		moduleFullName := moduleRef.FullName()
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
	moduleKeys = xslices.DeduplicateAny(
		moduleKeys,
		func(moduleKey bufmodule.ModuleKey) string { return moduleKey.String() },
	)
	commits, err := m.commitProvider.GetCommitsForModuleKeys(ctx, moduleKeys)
	if err != nil {
		return nil, err
	}
	commitIDToCommit := make(map[uuid.UUID]bufmodule.Commit, len(commits))
	for _, commit := range commits {
		// We don't know if these are unique, so we do not use xslices.ToUniqueValuesMapError.
		commitIDToCommit[commit.ModuleKey().CommitID()] = commit
	}
	return commitIDToCommit, nil
}

func (m *migrator) upgradeModuleKeysToB5(
	ctx context.Context,
	moduleKeys []bufmodule.ModuleKey,
) ([]bufmodule.ModuleKey, error) {
	moduleKeys = slices.Clone(moduleKeys)

	b4IndexedModuleKeys, err := xslices.FilterError(
		xslices.ToIndexed(moduleKeys),
		func(indexedModuleKey xslices.Indexed[bufmodule.ModuleKey]) (bool, error) {
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

	commitKeys, err := xslices.MapError(
		b4IndexedModuleKeys,
		func(indexedModuleKey xslices.Indexed[bufmodule.ModuleKey]) (bufmodule.CommitKey, error) {
			return bufmodule.NewCommitKey(
				indexedModuleKey.Value.FullName().Registry(),
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
			existingModuleKey.FullName(),
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
	moduleFullNameToDeclaredRefs map[string][]bufparse.Ref,
	moduleFullNameToLockKeys map[string][]bufmodule.ModuleKey,
) ([]bufparse.Ref, []bufmodule.ModuleKey, error) {
	depFullNameToResolvedRef := make(map[string]bufparse.Ref)
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
			return nil, nil, errors.Join(errs...)
		}
		depFullNameToResolvedRef[moduleFullName] = refs[0]
	}
	resolvedDepModuleKeys := make([]bufmodule.ModuleKey, 0, len(moduleFullNameToLockKeys))
	for moduleFullName, lockKeys := range moduleFullNameToLockKeys {
		resolvedRef, ok := depFullNameToResolvedRef[moduleFullName]
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
			return nil, nil, errors.Join(errs...)
		}
		resolvedDepModuleKeys = append(resolvedDepModuleKeys, lockKeys[0])
	}
	resolvedDeclaredDependencies := xslices.MapValuesToSlice(depFullNameToResolvedRef)
	// Sort the resolved dependencies for deterministic results.
	sort.Slice(resolvedDeclaredDependencies, func(i, j int) bool {
		return resolvedDeclaredDependencies[i].FullName().String() < resolvedDeclaredDependencies[j].FullName().String()
	})
	sort.Slice(resolvedDepModuleKeys, func(i, j int) bool {
		return resolvedDepModuleKeys[i].FullName().String() < resolvedDepModuleKeys[j].FullName().String()
	})
	return resolvedDeclaredDependencies, resolvedDepModuleKeys, nil
}

func equivalentLintConfigInV2(
	ctx context.Context,
	logger *slog.Logger,
	lintConfig bufconfig.LintConfig,
) (bufconfig.LintConfig, error) {
	equivalentCheckConfigV2, err := equivalentCheckConfigInV2(
		ctx,
		logger,
		check.RuleTypeLint,
		lintConfig,
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

func equivalentBreakingConfigInV2(
	ctx context.Context,
	logger *slog.Logger,
	breakingConfig bufconfig.BreakingConfig,
) (bufconfig.BreakingConfig, error) {
	equivalentCheckConfigV2, err := equivalentCheckConfigInV2(
		ctx,
		logger,
		check.RuleTypeBreaking,
		breakingConfig,
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
	ctx context.Context,
	logger *slog.Logger,
	ruleType check.RuleType,
	checkConfig bufconfig.CheckConfig,
) (bufconfig.CheckConfig, error) {
	// No need for custom lint/breaking plugins since there's no plugins to migrate from <=v1.
	// TODO: If we ever need v3, then we will have to deal with this.
	client, err := bufcheck.NewClient(logger)
	if err != nil {
		return nil, err
	}
	expectedRules, err := client.ConfiguredRules(ctx, ruleType, checkConfig)
	if err != nil {
		return nil, err
	}
	deprecations, err := bufcheck.GetDeprecatedIDToReplacementIDs(expectedRules)
	if err != nil {
		return nil, err
	}
	expectedRules = xslices.Filter(expectedRules, func(rule bufcheck.Rule) bool { return !rule.Deprecated() })
	expectedIDs := xslices.Map(
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
		checkConfig.DisableBuiltin(),
	)
	if err != nil {
		return nil, err
	}
	simplyTranslatedRules, err := client.ConfiguredRules(ctx, ruleType, simplyTranslatedCheckConfig)
	if err != nil {
		return nil, err
	}
	simplyTranslatedIDs := xslices.Map(
		simplyTranslatedRules,
		func(rule bufcheck.Rule) string {
			return rule.ID()
		},
	)
	if slices.Equal(expectedIDs, simplyTranslatedIDs) {
		// If the simple translation is equivalent to before, use it.
		return simplyTranslatedCheckConfig, nil
	}
	// Otherwise, find what's missing and what's extra.
	expectedIDsMap := xslices.ToStructMap(expectedIDs)
	simplyTranslatedIDsMap := xslices.ToStructMap(simplyTranslatedIDs)
	missingIDs := xslices.Filter(
		expectedIDs,
		func(expectedID string) bool {
			_, ok := simplyTranslatedIDsMap[expectedID]
			return !ok
		},
	)
	extraIDs := xslices.Filter(
		simplyTranslatedIDs,
		func(simplyTranslatedID string) bool {
			_, ok := expectedIDsMap[simplyTranslatedID]
			return !ok
		},
	)
	// Filter remaining rules to match the V2 rule set. Any other rules are ignored.
	// Theres no additional rules from plugins as plugins didn't exist before v2.
	validV2IDsMap := make(map[string]struct{})
	for _, ruleSpec := range bufcheckserver.V2Spec.Rules {
		if ruleSpec.Type == ruleType {
			validV2IDsMap[ruleSpec.ID] = struct{}{}
		}
	}
	for _, categorySpec := range bufcheckserver.V2Spec.Categories {
		validV2IDsMap[categorySpec.ID] = struct{}{}
	}
	useIDsAndCategories := xslices.Filter(
		append(simplyTranslatedCheckConfig.UseIDsAndCategories(), missingIDs...),
		func(ruleID string) bool {
			_, ok := validV2IDsMap[ruleID]
			return ok
		},
	)
	exceptIDsAndCategories := xslices.Filter(
		append(simplyTranslatedCheckConfig.ExceptIDsAndCategories(), extraIDs...),
		func(ruleID string) bool {
			_, ok := validV2IDsMap[ruleID]
			return ok
		},
	)
	return bufconfig.NewEnabledCheckConfig(
		bufconfig.FileVersionV2,
		useIDsAndCategories,
		exceptIDsAndCategories,
		simplyTranslatedCheckConfig.IgnorePaths(),
		simplyTranslatedCheckConfig.IgnoreIDOrCategoryToPaths(),
		simplyTranslatedCheckConfig.DisableBuiltin(),
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
