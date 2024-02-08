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

package bufmodule

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufbreaking/bufbreakingconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/buflint/buflintconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	breakingv1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/breaking/v1"
	lintv1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/lint/v1"
	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/storage"
	"go.uber.org/multierr"
)

const (
	// DefaultDocumentationPath defines the default path to the documentation file, relative to the root of the module.
	DefaultDocumentationPath = "buf.md"
	// LicenseFilePath defines the path to the license file, relative to the root of the module.
	LicenseFilePath = "LICENSE"

	// b3DigestPrefix is the digest prefix for the third version of the digest function.
	//
	// It is used by the CLI cache and intended to eventually replace b1 entirely.
	b3DigestPrefix = "b3"
)

var (
	// AllDocumentationPaths defines all possible paths to the documentation file, relative to the root of the module.
	AllDocumentationPaths = []string{
		DefaultDocumentationPath,
		"README.md",
		"README.markdown",
	}
)

// ModuleFile is a module file.
type ModuleFile interface {
	bufmoduleref.FileInfo
	io.ReadCloser

	isModuleFile()
}

// Module is a Protobuf module.
//
// It contains the files for the sources, and the dependency names.
//
// Terminology:
//
// Targets (Modules and ModuleFileSets):
//
//	Just the files specified to build. This will either be sources, or will be specific files
//	within sources, ie this is a subset of Sources. The difference between Targets and Sources happens
//	when i.e. the --path flag is used.
//
// Sources (Modules and ModuleFileSets):
//
//	The files with no dependencies. This is a superset of Targets and subset of All.
//
// All (ModuleFileSets only):
//
//	All files including dependencies. This is a superset of Sources.
type Module interface {
	// TargetFileInfos gets all FileInfos specified as target files. This is either
	// all the FileInfos belonging to the module, or those specified by ModuleWithTargetPaths().
	//
	// It does not include dependencies.
	//
	// The returned TargetFileInfos are sorted by path.
	TargetFileInfos(ctx context.Context) ([]bufmoduleref.FileInfo, error)
	// SourceFileInfos gets all FileInfos belonging to the module.
	//
	// It does not include dependencies.
	//
	// The returned SourceFileInfos are sorted by path.
	SourceFileInfos(ctx context.Context) ([]bufmoduleref.FileInfo, error)
	// GetModuleFile gets the source file for the given path.
	//
	// Returns fs.ErrNotExist error if the file does not exist.
	GetModuleFile(ctx context.Context, path string) (ModuleFile, error)
	// DeclaredDirectDependencies returns the direct dependencies declared in the configuration file.
	//
	// The returned ModuleReferences are sorted by remote, owner, repository, and reference (if
	// present). The returned ModulePins are unique by remote, owner, repository.
	//
	// This does not include any transitive dependencies, but if the declarations are correct,
	// this should be a subset of the dependencies from DependencyModulePins.
	//
	// TODO: validate that this is a subset? This may mess up construction.
	DeclaredDirectDependencies() []bufmoduleref.ModuleReference
	// DependencyModulePins gets the dependency ModulePins.
	//
	// The returned ModulePins are sorted by remote, owner, repository, branch, commit, and then digest.
	// The returned ModulePins are unique by remote, owner, repository.
	//
	// This includes all transitive dependencies.
	DependencyModulePins() []bufmoduleref.ModulePin
	// Documentation gets the contents of the module documentation file, buf.md and returns the string representation.
	// This may return an empty string if the documentation file does not exist.
	Documentation() string
	// DocumentationPath returns the path to the documentation file for the module.
	// Can be one of `buf.md`, `README.md` or `README.markdown`
	DocumentationPath() string
	// License gets the contents of the module license file, LICENSE and returns the string representation.
	// This may return an empty string if the documentation file does not exist.
	License() string
	// BreakingConfig returns the breaking change check configuration set for the module.
	//
	// This may be nil, since older versions of the module would not have this stored.
	BreakingConfig() *bufbreakingconfig.Config
	// LintConfig returns the lint check configuration set for the module.
	//
	// This may be nil, since older versions of the module would not have this stored.
	LintConfig() *buflintconfig.Config
	// FileSet returns the FileSet for the module.
	//
	// FileSet will be nil if the Module was not constructed with a FileSet.
	//
	// A FileSet's contents contain a lexicographically sorted list of path names along
	// with each path's digest. In addition to the .proto files in the module, it also lists
	// the buf.yaml, LICENSE, buf.md, and buf.lock files (if present).
	FileSet() bufcas.FileSet
	// ModuleIdentity returns the ModuleIdentity for the Module, if it was
	// provided at construction time via ModuleWithModuleIdentity or ModuleWithModuleIdentityAndCommit.
	//
	// Note this *can* be nil if we did not build from a named module.
	// All code must assume this can be nil.
	// nil checking should work since the backing type is always a pointer.
	ModuleIdentity() bufmoduleref.ModuleIdentity
	// Commit returns the commit for the Module, if it was
	// provided at construction time via ModuleWithModuleIdentityAndCommit.

	// Note this can be empty.
	// This will only be set if ModuleIdentity is set. but may not be set
	// even if ModuleIdentity is set, that is commit is optional information
	// even if we know what module this file came from.
	Commit() string
	// WorkspaceDirectory returns the directory within the workspace that this Module was constructed from,
	// if this Module was constructed from a Workspace. If the Module was not constructed from a Workspace,
	// This value will be empty.
	//
	// This is needed for now because we need to determine if the input for ModuleFileSetBuilder is equal
	// to any Modules within the given optional Workspace. This is terrible. Once we have refactored
	// the CLI to have Workspaces as a first-class citizen, where the typical case is a Workspace with
	// a single Module, we will no longer need to do this type of check, and this can be removed.
	WorkspaceDirectory() string

	getSourceReadBucket() storage.ReadBucket
	isModule()
}

// ModuleOption is used to construct Modules.
type ModuleOption func(*module)

// ModuleWithModuleIdentity is used to construct a Module with a ModuleIdentity.
func ModuleWithModuleIdentity(moduleIdentity bufmoduleref.ModuleIdentity) ModuleOption {
	return func(module *module) {
		module.moduleIdentity = moduleIdentity
	}
}

// ModuleWithModuleIdentityAndCommit is used to construct a Module with a ModuleIdentity and commit.
//
// If the moduleIdentity is nil, the commit must be empty, that is it is not valid to have
// a non-empty commit and a nil moduleIdentity.
func ModuleWithModuleIdentityAndCommit(moduleIdentity bufmoduleref.ModuleIdentity, commit string) ModuleOption {
	return func(module *module) {
		module.moduleIdentity = moduleIdentity
		module.commit = commit
	}
}

// ModuleWithWorkspaceDirectory returns a new ModuleOption that sets the workspace directory.
//
// See the comment on Module.WorkspaceDirectory() for more details.
func ModuleWithWorkspaceDirectory(workspaceDirectory string) ModuleOption {
	return func(module *module) {
		module.workspaceDirectory = workspaceDirectory
	}
}

// NewModuleForBucket returns a new Module. It attempts to read dependencies
// from a lock file in the read bucket.
func NewModuleForBucket(
	ctx context.Context,
	readBucket storage.ReadBucket,
	options ...ModuleOption,
) (Module, error) {
	return newModuleForBucket(ctx, readBucket, options...)
}

// NewModuleForProto returns a new Module for the given proto Module.
func NewModuleForProto(
	ctx context.Context,
	protoModule *modulev1alpha1.Module,
	options ...ModuleOption,
) (Module, error) {
	return newModuleForProto(ctx, protoModule, options...)
}

// NewModuleForFileSet returns a new Module given the FileSet.
func NewModuleForFileSet(
	ctx context.Context,
	fileSet bufcas.FileSet,
	options ...ModuleOption,
) (Module, error) {
	return newModuleForFileSet(ctx, fileSet, options...)
}

// ModuleWithTargetPaths returns a new Module that specifies specific file or directory paths to build.
//
// These paths must exist.
// These paths must be relative to the roots.
// These paths will be normalized and validated.
// These paths must be unique when normalized and validated.
// Multiple calls to this option will override previous calls.
//
// Note that this will result in TargetFileInfos containing only these paths, and not
// any imports. Imports, and non-targeted files, are still available via SourceFileInfos.
func ModuleWithTargetPaths(
	module Module,
	targetPaths []string,
	excludePaths []string,
) (Module, error) {
	return newTargetingModule(module, targetPaths, excludePaths, false)
}

// ModuleWithTargetPathsAllowNotExist returns a new Module specifies specific file or directory paths to build,
// but allows the specified paths to not exist.
//
// Note that this will result in TargetFileInfos containing only these paths, and not
// any imports. Imports, and non-targeted files, are still available via SourceFileInfos.
func ModuleWithTargetPathsAllowNotExist(
	module Module,
	targetPaths []string,
	excludePaths []string,
) (Module, error) {
	return newTargetingModule(module, targetPaths, excludePaths, true)
}

// ModuleWithExcludePaths returns a new Module that excludes specific file or directory
// paths to build.
//
// Note that this will result in TargetFileInfos containing only the paths that have not been
// excluded and any imports. Imports are still available via SourceFileInfos.
func ModuleWithExcludePaths(
	module Module,
	excludePaths []string,
) (Module, error) {
	return newTargetingModule(module, nil, excludePaths, false)
}

// ModuleWithExcludePathsAllowNotExist returns a new Module that excludes specific file or
// directory paths to build, but allows the specified paths to not exist.
//
// Note that this will result in TargetFileInfos containing only these paths, and not
// any imports. Imports, and non-targeted files, are still available via SourceFileInfos.
func ModuleWithExcludePathsAllowNotExist(
	module Module,
	excludePaths []string,
) (Module, error) {
	return newTargetingModule(module, nil, excludePaths, true)
}

// ModuleResolver resolves modules.
type ModuleResolver interface {
	// GetModulePin resolves the provided ModuleReference to a ModulePin.
	//
	// Returns an error with fs.ErrNotExist if the named Module does not exist.
	GetModulePin(ctx context.Context, moduleReference bufmoduleref.ModuleReference) (bufmoduleref.ModulePin, error)
}

// NewNopModuleResolver returns a new ModuleResolver that always returns a fs.ErrNotExist error.
func NewNopModuleResolver() ModuleResolver {
	return newNopModuleResolver()
}

// ModuleReader reads resolved modules.
type ModuleReader interface {
	// GetModule gets the Module for the ModulePin.
	//
	// Returns an error with fs.ErrNotExist if the Module does not exist.
	GetModule(ctx context.Context, modulePin bufmoduleref.ModulePin) (Module, error)
}

// NewNopModuleReader returns a new ModuleReader that always returns a fs.ErrNotExist error.
func NewNopModuleReader() ModuleReader {
	return newNopModuleReader()
}

// ModuleFileSet is a Protobuf module file set.
//
// It contains the files for both targets, sources and dependencies.
//
// TODO: we should not have ModuleFileSet inherit from Module, this is confusing
type ModuleFileSet interface {
	// Note that GetModuleFile will pull from All files instead of just Source Files!
	Module
	// AllFileInfos gets all FileInfos associated with the module, including dependencies.
	//
	// The returned FileInfos are sorted by path.
	AllFileInfos(ctx context.Context) ([]bufmoduleref.FileInfo, error)

	isModuleFileSet()
}

// NewModuleFileSet returns a new ModuleFileSet.
func NewModuleFileSet(
	module Module,
	dependencies []Module,
) ModuleFileSet {
	return newModuleFileSet(module, dependencies)
}

// Workspace represents a workspace.
//
// It is guaranteed that all Modules within this workspace have no overlapping file paths.
type Workspace interface {
	// GetModule gets the module identified by the given ModuleIdentity.
	//
	GetModule(moduleIdentity bufmoduleref.ModuleIdentity) (Module, bool)
	// GetModules returns all of the modules found in the workspace.
	GetModules() []Module
}

// NewWorkspace returns a new module workspace.
//
// The Context is not retained, and is only used for validation during construction.
func NewWorkspace(
	ctx context.Context,
	namedModules map[string]Module,
	allModules []Module,
) (Workspace, error) {
	return newWorkspace(
		ctx,
		namedModules,
		allModules,
	)
}

// ModuleToProtoModule converts the Module to a proto Module.
//
// This takes all Sources and puts them in the Module, not just Targets.
func ModuleToProtoModule(ctx context.Context, module Module) (*modulev1alpha1.Module, error) {
	// these are returned sorted, so there is no need to sort
	// the resulting protoModuleFiles afterwards
	sourceFileInfos, err := module.SourceFileInfos(ctx)
	if err != nil {
		return nil, err
	}
	protoModuleFiles := make([]*modulev1alpha1.ModuleFile, len(sourceFileInfos))
	for i, sourceFileInfo := range sourceFileInfos {
		protoModuleFile, err := moduleFileToProto(ctx, module, sourceFileInfo.Path())
		if err != nil {
			return nil, err
		}
		protoModuleFiles[i] = protoModuleFile
	}
	// these are returned sorted, so there is no need to sort
	// the resulting protoModuleNames afterwards
	dependencyModulePins := module.DependencyModulePins()
	protoModulePins := make([]*modulev1alpha1.ModulePin, len(dependencyModulePins))
	for i, dependencyModulePin := range dependencyModulePins {
		protoModulePins[i] = bufmoduleref.NewProtoModulePinForModulePin(dependencyModulePin)
	}
	var protoBreakingConfig *breakingv1.Config
	if module.BreakingConfig() != nil {
		protoBreakingConfig = bufbreakingconfig.ProtoForConfig(module.BreakingConfig())
	}
	var protoLintConfig *lintv1.Config
	if module.LintConfig() != nil {
		protoLintConfig = buflintconfig.ProtoForConfig(module.LintConfig())
	}
	protoModule := &modulev1alpha1.Module{
		Files:             protoModuleFiles,
		Dependencies:      protoModulePins,
		Documentation:     module.Documentation(),
		DocumentationPath: module.DocumentationPath(),
		BreakingConfig:    protoBreakingConfig,
		LintConfig:        protoLintConfig,
		License:           module.License(),
	}
	if err := ValidateProtoModule(protoModule); err != nil {
		return nil, err
	}
	return protoModule, nil
}

// ModuleDigestB3 returns the b3 digest for the Module.
//
// To create the module digest (SHA256):
//  1. For every file in the module (sorted lexicographically by path):
//     a. Add the file path
//     b. Add the file contents
//  2. Add the dependency's module identity and commit ID (sorted lexicographically by commit ID)
//  3. Add the module identity if available.
//  4. Add the module documentation if available.
//  5. Add the module documentation path if available.
//  6. Add the module license if available.
//  7. Add the breaking and lint configurations if available.
//  8. Produce the final digest by URL-base64 encoding the summed bytes and prefixing it with the digest prefix
func ModuleDigestB3(ctx context.Context, module Module) (string, error) {
	hash := sha256.New()
	// We do not want to change the sort order as the rest of the codebase relies on it,
	// but we only want to use commit as part of the sort order, so we make a copy of
	// the slice and sort it by commit
	for _, dependencyModulePin := range copyModulePinsSortedByOnlyCommit(module.DependencyModulePins()) {
		if _, err := hash.Write([]byte(dependencyModulePin.IdentityString() + ":" + dependencyModulePin.Commit())); err != nil {
			return "", err
		}
	}
	sourceFileInfos, err := module.SourceFileInfos(ctx)
	if err != nil {
		return "", err
	}
	for _, sourceFileInfo := range sourceFileInfos {
		if _, err := hash.Write([]byte(sourceFileInfo.Path())); err != nil {
			return "", err
		}
		moduleFile, err := module.GetModuleFile(ctx, sourceFileInfo.Path())
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(hash, moduleFile); err != nil {
			return "", multierr.Append(err, moduleFile.Close())
		}
		if err := moduleFile.Close(); err != nil {
			return "", err
		}
	}
	if moduleIdentity := module.ModuleIdentity(); moduleIdentity != nil {
		if _, err := hash.Write([]byte(moduleIdentity.IdentityString())); err != nil {
			return "", err
		}
	}
	if docs := module.Documentation(); docs != "" {
		if _, err := hash.Write([]byte(docs)); err != nil {
			return "", err
		}
	}
	if docPath := module.DocumentationPath(); docPath != "" && docPath != DefaultDocumentationPath {
		if _, err := hash.Write([]byte(docPath)); err != nil {
			return "", err
		}
	}
	if license := module.License(); license != "" {
		if _, err := hash.Write([]byte(license)); err != nil {
			return "", err
		}
	}
	if breakingConfig := module.BreakingConfig(); breakingConfig != nil {
		breakingConfigBytes, err := bufbreakingconfig.BytesForConfig(breakingConfig)
		if err != nil {
			return "", err
		}
		if _, err := hash.Write(breakingConfigBytes); err != nil {
			return "", err
		}
	}
	if lintConfig := module.LintConfig(); lintConfig != nil {
		lintConfigBytes, err := buflintconfig.BytesForConfig(lintConfig)
		if err != nil {
			return "", err
		}
		if _, err := hash.Write(lintConfigBytes); err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("%s-%s", b3DigestPrefix, base64.URLEncoding.EncodeToString(hash.Sum(nil))), nil
}

// ModuleToBucket writes the given Module to the WriteBucket.
//
// This writes the sources and the buf.lock file.
// This copies external paths if the WriteBucket supports setting of external paths.
func ModuleToBucket(
	ctx context.Context,
	module Module,
	writeBucket storage.WriteBucket,
) error {
	fileInfos, err := module.SourceFileInfos(ctx)
	if err != nil {
		return err
	}
	for _, fileInfo := range fileInfos {
		if err := putModuleFileToBucket(ctx, module, fileInfo.Path(), writeBucket); err != nil {
			return err
		}
	}
	if docs := module.Documentation(); docs != "" {
		moduleDocPath := DefaultDocumentationPath
		if docPath := module.DocumentationPath(); docPath != "" {
			moduleDocPath = docPath
		}
		if err := storage.PutPath(ctx, writeBucket, moduleDocPath, []byte(docs)); err != nil {
			return err
		}
	}
	if license := module.License(); license != "" {
		if err := storage.PutPath(ctx, writeBucket, LicenseFilePath, []byte(license)); err != nil {
			return err
		}
	}
	if err := bufmoduleref.PutDependencyModulePinsToBucket(ctx, writeBucket, module.DependencyModulePins()); err != nil {
		return err
	}
	// This is the default version created by bufconfig getters. The versions should be the
	// same across lint and breaking configs.
	version := bufconfig.V1Version
	var breakingConfigVersion string
	if module.BreakingConfig() != nil {
		breakingConfigVersion = module.BreakingConfig().Version
	}
	var lintConfigVersion string
	if module.LintConfig() != nil {
		lintConfigVersion = module.LintConfig().Version
	}
	// If one of either breaking or lint config is non-nil, then other config will also be non-nil,
	// even if a module does not set both configurations. An empty with the correct version
	// will be set by the configuration getters.
	if breakingConfigVersion != lintConfigVersion {
		return fmt.Errorf("breaking config version %q does not match lint config version %q", breakingConfigVersion, lintConfigVersion)
	}
	if breakingConfigVersion != "" || lintConfigVersion != "" {
		version = breakingConfigVersion
	}
	writeConfigOptions := []bufconfig.WriteConfigOption{
		bufconfig.WriteConfigWithModuleIdentity(module.ModuleIdentity()),
		bufconfig.WriteConfigWithBreakingConfig(module.BreakingConfig()),
		bufconfig.WriteConfigWithLintConfig(module.LintConfig()),
		bufconfig.WriteConfigWithVersion(version),
	}
	return bufconfig.WriteConfig(ctx, writeBucket, writeConfigOptions...)
}

// TargetModuleFilesToBucket writes the target files of the given Module to the WriteBucket.
//
// This does not write the buf.lock file.
// This copies external paths if the WriteBucket supports setting of external paths.
func TargetModuleFilesToBucket(
	ctx context.Context,
	module Module,
	writeBucket storage.WriteBucket,
) error {
	fileInfos, err := module.TargetFileInfos(ctx)
	if err != nil {
		return err
	}
	for _, fileInfo := range fileInfos {
		if err := putModuleFileToBucket(ctx, module, fileInfo.Path(), writeBucket); err != nil {
			return err
		}
	}
	return nil
}
