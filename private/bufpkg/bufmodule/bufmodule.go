// Copyright 2020-2021 Buf Technologies, Inc.
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

	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufbreaking/bufbreakingconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/buflint/buflintconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/storage"
	"go.uber.org/multierr"
)

const (
	// DocumentationFilePath defines the path to the documentation file, relative to the root of the module.
	DocumentationFilePath = "buf.md"

	// b1DigestPrefix is the digest prefix for the first version of the digest function.
	//
	// This is used in lockfiles, and stored in the BSR.
	// It is intended to be eventually removed.
	b1DigestPrefix = "b1"

	// b3DigestPrefix is the digest prefix for the third version of the digest function.
	//
	// It is used by the CLI cache and intended to eventually replace b1 entirely.
	b3DigestPrefix = "b3"
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
//   Just the files specified to build. This will either be sources, or will be specific files
//   within sources, ie this is a subset of Sources. The difference between Targets and Sources happens
//   when i.e. the --path flag is used.
// Sources (Modules and ModuleFileSets):
//   The files with no dependencies. This is a superset of Targets and subset of All.
// All (ModuleFileSets only):
//   All files including dependencies. This is a superset of Sources.
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
	// Returns storage.IsNotExist error if the file does not exist.
	GetModuleFile(ctx context.Context, path string) (ModuleFile, error)
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
	// BreakingConfig returns the breaking change check configuration set for the module.
	//
	// This may be nil, since older versions of the module would not have this stored.
	BreakingConfig() *bufbreakingconfig.Config
	// LintConfig returns the lint check configuration set for the module.
	//
	// This may be nil, since older versions of the module would not have this stored.
	LintConfig() *buflintconfig.Config

	getSourceReadBucket() storage.ReadBucket
	// Note this *can* be nil if we did not build from a named module.
	// All code must assume this can be nil.
	// nil checking should work since the backing type is always a pointer.
	//
	// TODO: We can remove the getModuleReference method on the if we fetch
	// FileInfos from the Module and plumb in the ModuleReference here.
	//
	// This approach assumes that all of the FileInfos returned
	// from SourceFileInfos will have their ModuleReference
	// set to the same value, which can be validated.
	getModuleIdentity() bufmoduleref.ModuleIdentity
	// Note this can be empty.
	getCommit() string
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
func ModuleWithModuleIdentityAndCommit(moduleIdentity bufmoduleref.ModuleIdentity, commit string) ModuleOption {
	return func(module *module) {
		module.moduleIdentity = moduleIdentity
		module.commit = commit
	}
}

// NewModuleForBucket returns a new Module. It attempts reads dependencies
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
	// Returns an error that fufills storage.IsNotExist if the named Module does not exist.
	GetModulePin(ctx context.Context, moduleReference bufmoduleref.ModuleReference) (bufmoduleref.ModulePin, error)
}

// NewNopModuleResolver returns a new ModuleResolver that always returns a storage.IsNotExist error.
func NewNopModuleResolver() ModuleResolver {
	return newNopModuleResolver()
}

// ModuleReader reads resolved modules.
type ModuleReader interface {
	// GetModule gets the Module for the ModulePin.
	//
	// Returns an error that fufills storage.IsNotExist if the Module does not exist.
	GetModule(ctx context.Context, modulePin bufmoduleref.ModulePin) (Module, error)
}

// NewNopModuleReader returns a new ModuleReader that always returns a storage.IsNotExist error.
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

// Workspace represents a module workspace.
type Workspace interface {
	// GetModule gets the module identified by the given ModuleIdentity.
	GetModule(moduleIdentity bufmoduleref.ModuleIdentity) (Module, bool)
	// GetModules returns all of the modules found in the workspace.
	GetModules() []Module
}

// NewWorkspace returns a new module workspace.
func NewWorkspace(
	namedModules map[string]Module,
	allModules []Module,
) Workspace {
	return newWorkspace(
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
	protoModule := &modulev1alpha1.Module{
		Files:         protoModuleFiles,
		Dependencies:  protoModulePins,
		Documentation: module.Documentation(),
	}
	if err := ValidateProtoModule(protoModule); err != nil {
		return nil, err
	}
	return protoModule, nil
}

// ModuleDigestB1 returns the b1 digest for the Module.
//
// To create the module digest (SHA256):
// 	1. For every file in the module (sorted lexicographically by path):
// 		a. Add the file path
//		b. Add the file contents
// 	2. Add the dependency hashes (sorted lexicographically by the string representation)
//	3. Produce the final digest by URL-base64 encoding the summed bytes and prefixing it with the digest prefix
func ModuleDigestB1(ctx context.Context, module Module) (string, error) {
	hash := sha256.New()
	// DependencyModulePins returns these sorted
	for _, dependencyModulePin := range module.DependencyModulePins() {
		// We include each of these individually as opposed to using String
		// so that if the String representation changes, we still get the same digest.
		//
		// Note that this does mean that changing a repository name or owner
		// will result in a different digest, this is something we may
		// want to revisit.
		if _, err := hash.Write([]byte(dependencyModulePin.Remote())); err != nil {
			return "", err
		}
		if _, err := hash.Write([]byte(dependencyModulePin.Owner())); err != nil {
			return "", err
		}
		if _, err := hash.Write([]byte(dependencyModulePin.Repository())); err != nil {
			return "", err
		}
		if _, err := hash.Write([]byte(dependencyModulePin.Digest())); err != nil {
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
	if docs := module.Documentation(); docs != "" {
		if _, err := hash.Write([]byte(docs)); err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("%s-%s", b1DigestPrefix, base64.URLEncoding.EncodeToString(hash.Sum(nil))), nil
}

// ModuleDigestB3 returns the b3 digest for the Module.
//
// To create the module digest (SHA256):
//  1. For every file in the module (sorted lexicographically by path):
//      a. Add the file path
//      b. Add the file contents
//  2. Add the dependency commits (sorted lexicographically by commit)
//  3. Add the module documentation if available.
//  4. Add the breaking and lint configurations if available.
//  5. Produce the final digest by URL-base64 encoding the summed bytes and prefixing it with the digest prefix
func ModuleDigestB3(ctx context.Context, module Module) (string, error) {
	hash := sha256.New()
	// We do not want to change the sort order as the rest of the codebase relies on it,
	// but we only want to use commit as part of the sort order, so we make a copy of
	// the slice and sort it by commit
	for _, dependencyModulePin := range copyModulePinsSortedByOnlyCommit(module.DependencyModulePins()) {
		if _, err := hash.Write([]byte(dependencyModulePin.Commit())); err != nil {
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
	if docs := module.Documentation(); docs != "" {
		if _, err := hash.Write([]byte(docs)); err != nil {
			return "", err
		}
	}
	if breakingConfig := module.BreakingConfig(); breakingConfig != nil {
		breakingConfigBytes, err := bufbreakingconfig.NewBreakingConfigToBytes(breakingConfig)
		if err != nil {
			return "", err
		}
		if _, err := hash.Write(breakingConfigBytes); err != nil {
			return "", err
		}
	}
	if lintConfig := module.LintConfig(); lintConfig != nil {
		lintConfigBytes, err := buflintconfig.NewLintConfigToBytes(lintConfig)
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
		if err := storage.PutPath(ctx, writeBucket, DocumentationFilePath, []byte(docs)); err != nil {
			return err
		}
	}
	return bufmoduleref.PutDependencyModulePinsToBucket(ctx, writeBucket, module.DependencyModulePins())
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
