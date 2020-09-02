// Copyright 2020 Buf Technologies, Inc.
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
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/bufbuild/buf/internal/buf/bufcore"
	modulev1 "github.com/bufbuild/buf/internal/gen/proto/go/buf/module/v1"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"go.uber.org/multierr"
)

const b1DigestPrefix = "b1"

// LockFilePath defines the path to the lock file, relative to the root of the module.
const LockFilePath = "buf.lock"

// ErrNoTargetFiles is the error returned if there are no target files found.
var ErrNoTargetFiles = errors.New("no .proto target files found")

// NewNoDigestError returns a new error indicating that a module did not have
// a digest where required.
func NewNoDigestError(moduleName ModuleName) error {
	return &errNoDigest{
		moduleName: moduleName,
	}
}

// IsNoDigestError returns whether the error provided, or
// any error wrapped by that error, is a NoDigest error.
func IsNoDigestError(err error) bool {
	return errors.Is(err, &errNoDigest{})
}

// ModuleFile is a file within a Root.
type ModuleFile interface {
	bufcore.FileInfo
	io.ReadCloser

	isModuleFile()
}

// ModuleName is a module name.
type ModuleName interface {
	fmt.Stringer

	// Required.
	Server() string
	// Required.
	Owner() string
	// Required.
	Repository() string
	// Required.
	Version() string
	// Optional.
	Digest() string

	isModuleName()
}

// NewModuleName returns a new validated ModuleName.
func NewModuleName(
	server string,
	owner string,
	repository string,
	version string,
	digest string,
) (ModuleName, error) {
	return newModuleName(server, owner, repository, version, digest)
}

// NewModuleNameForProto returns a new ModuleName for the given proto ModuleName.
func NewModuleNameForProto(protoModuleName *modulev1.ModuleName) (ModuleName, error) {
	return newModuleNameForProto(protoModuleName)
}

// NewModuleNamesForProtos maps the Protobuf equivalent into the internal represenation.
func NewModuleNamesForProtos(moduleNames ...*modulev1.ModuleName) ([]ModuleName, error) {
	if len(moduleNames) == 0 {
		return nil, nil
	}
	result := make([]ModuleName, len(moduleNames))
	for i, proto := range moduleNames {
		moduleName, err := NewModuleNameForProto(proto)
		if err != nil {
			return nil, err
		}
		result[i] = moduleName
	}
	return result, nil
}

// NewProtoModuleNameForModuleName returns a new proto ModuleName for the given ModuleName.
func NewProtoModuleNameForModuleName(moduleName ModuleName) (*modulev1.ModuleName, error) {
	return newProtoModuleNameForModuleName(moduleName)
}

// NewProtoModuleNamesForModuleNames maps the given module names into the protobuf representation.
func NewProtoModuleNamesForModuleNames(moduleNames ...ModuleName) ([]*modulev1.ModuleName, error) {
	if len(moduleNames) == 0 {
		return nil, nil
	}
	result := make([]*modulev1.ModuleName, len(moduleNames))
	for i, proto := range moduleNames {
		moduleName, err := NewProtoModuleNameForModuleName(proto)
		if err != nil {
			return nil, err
		}
		result[i] = moduleName
	}
	return result, nil
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
//   when i.e. the --file flag is used.
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
	TargetFileInfos(ctx context.Context) ([]bufcore.FileInfo, error)
	// SourceFileInfos gets all FileInfos belonging to the module.
	//
	// It does not include dependencies.
	//
	// The returned SourceFileInfos are sorted by path.
	SourceFileInfos(ctx context.Context) ([]bufcore.FileInfo, error)
	// GetFile gets the source file for the given path.
	//
	// Returns storage.IsNotExist error if the file does not exist.
	GetFile(ctx context.Context, path string) (ModuleFile, error)
	// Dependencies gets the dependency ModuleNames. These are always resolved.
	//
	// The returned ModuleNames are sorted by server, owner, repository, version, and then digest.
	Dependencies() []ModuleName

	getSourceReadBucket() storage.ReadBucket
	isModule()
}

// NewModuleForBucket returns a new Module. It attempts reads dependencies
// from a lock file in the read bucket.
func NewModuleForBucket(
	ctx context.Context,
	readBucket storage.ReadBucket,
) (Module, error) {
	return newModuleForBucket(ctx, readBucket)
}

// NewModuleForBucketWithDependencies explicitly specifies the dependencies that should
// be used when creating the Module. The module names must be resolved and unique.
func NewModuleForBucketWithDependencies(
	ctx context.Context,
	readBucket storage.ReadBucket,
	dependencies []ModuleName,
) (Module, error) {
	return newModuleForBucketWithDependencies(ctx, readBucket, dependencies)
}

// NewModuleForProto returns a new Module for the given proto Module.
func NewModuleForProto(
	ctx context.Context,
	protoModule *modulev1.Module,
) (Module, error) {
	return newModuleForProto(ctx, protoModule)
}

// ModuleWithTargetPaths returns a new Module that specifies specific file paths to build.
//
// These paths must exist.
// These paths must be relative to the roots.
// These paths will be normalized and validated.
// These paths must be unique when normalized and validated.
// Multiple calls to this option will override previous calls.
func ModuleWithTargetPaths(module Module, targetPaths []string) (Module, error) {
	return newTargetingModule(module, targetPaths, false)
}

// ModuleWithTargetPathsAllowNotExist returns a new Module specified specifie file paths to build,
// but allows the specified paths to not exist.
func ModuleWithTargetPathsAllowNotExist(module Module, targetPaths []string) (Module, error) {
	return newTargetingModule(module, targetPaths, true)
}

// ModuleReader reads modules.
type ModuleReader interface {
	// GetModule gets the named Module.
	//
	// The given ModuleName must be resolved, i.e. it must have a digest.
	//
	// Returns an error that fufills storage.IsNotExist if the named Module does not exist.
	GetModule(ctx context.Context, moduleName ModuleName) (Module, error)
}

// ModuleWriter writes modules.
type ModuleWriter interface {
	// PutModule puts the named Module.
	//
	// The given ModuleName must be unresolved, i.e. it must not have a digest.
	//
	// Returns a new resolved ModuleName with the digest.
	PutModule(ctx context.Context, moduleName ModuleName, module Module) (ModuleName, error)
}

// ModuleReadWriter is a ModuleReader and ModuleWriter
type ModuleReadWriter interface {
	ModuleReader
	ModuleWriter
}

// NewNopModuleReader returns a new ModuleReader that always returns a storage.IsNotExist error.
func NewNopModuleReader() ModuleReader {
	return newNopModuleReader()
}

// NewNopModuleWriter returns a new ModuleWriter that does not store the Module, instead just
// calculates the digest and returns a ModuleName with this digest.
func NewNopModuleWriter() ModuleWriter {
	return newNopModuleWriter()
}

// NewNopModuleReadWriter returns a new ModuleReadWriter that combines a no-op ModuleReader
// and a no-op ModuleWriter.
func NewNopModuleReadWriter() ModuleReadWriter {
	return newNopModuleReadWriter()
}

// ModuleFileSet is a Protobuf module file set.
//
// It contains the files for both targets, sources and dependencies.
type ModuleFileSet interface {
	// Note that GetFile will pull from All files instead of just Source Files!
	Module
	// AllFileInfos gets all FileInfos associated with the module, including depedencies.
	//
	// The returned FileInfos are sorted by path.
	AllFileInfos(ctx context.Context) ([]bufcore.FileInfo, error)

	isModuleFileSet()
}

// NewModuleFileSet returns a new ModuleFileSet.
func NewModuleFileSet(
	module Module,
	dependencies []Module,
) ModuleFileSet {
	return newModuleFileSet(module, dependencies)
}

// ModuleNameForString returns a new ModuleName for the given string.
//
// This parses the path in the form server/owner/repository/version[:digest]
func ModuleNameForString(path string) (ModuleName, error) {
	slashSplit := strings.Split(path, "/")
	if len(slashSplit) != 4 {
		return nil, newInvalidModuleNameStringError(path, "module name is not in the form server/owner/repository/version")
	}
	server := strings.TrimSpace(slashSplit[0])
	if server == "" {
		return nil, newInvalidModuleNameStringError(path, "server name is empty")
	}
	owner := strings.TrimSpace(slashSplit[1])
	if owner == "" {
		return nil, newInvalidModuleNameStringError(path, "owner name is empty")
	}
	repository := strings.TrimSpace(slashSplit[2])
	if repository == "" {
		return nil, newInvalidModuleNameStringError(path, "repository name is empty")
	}
	versionSplit := strings.Split(slashSplit[3], ":")
	var version string
	var digest string
	switch len(versionSplit) {
	case 1:
		version = strings.TrimSpace(versionSplit[0])
	case 2:
		version = strings.TrimSpace(versionSplit[0])
		digest = strings.TrimSpace(versionSplit[1])
	default:
		return nil, newInvalidModuleNameStringError(path, "invalid version with digest")
	}
	if version == "" {
		return nil, newInvalidModuleNameStringError(path, "version name is empty")
	}
	return NewModuleName(
		server,
		owner,
		repository,
		version,
		digest,
	)
}

// ModuleToProtoModule converts the Module to a proto Module.
//
// This takes all Sources and puts them in the Module, not just Targets.
func ModuleToProtoModule(ctx context.Context, module Module) (*modulev1.Module, error) {
	// these are returned sorted, so there is no need to sort
	// the resulting protoModuleFiles afterwards
	sourceFileInfos, err := module.SourceFileInfos(ctx)
	if err != nil {
		return nil, err
	}
	protoModuleFiles := make([]*modulev1.ModuleFile, len(sourceFileInfos))
	for i, sourceFileInfo := range sourceFileInfos {
		protoModuleFile := &modulev1.ModuleFile{
			Path: sourceFileInfo.Path(),
		}
		moduleFile, err := module.GetFile(ctx, sourceFileInfo.Path())
		if err != nil {
			return nil, err
		}
		protoModuleFile.Content, err = ioutil.ReadAll(moduleFile)
		if err != nil {
			return nil, multierr.Append(err, moduleFile.Close())
		}
		if err := moduleFile.Close(); err != nil {
			return nil, err
		}
		protoModuleFiles[i] = protoModuleFile
	}
	// these are returned sorted, so there is no need to sort
	// the resulting protoModuleNames afterwards
	dependencies := module.Dependencies()
	protoModuleNames := make([]*modulev1.ModuleName, len(dependencies))
	for i, dependency := range dependencies {
		protoModuleName := &modulev1.ModuleName{
			Server:     dependency.Server(),
			Owner:      dependency.Owner(),
			Repository: dependency.Repository(),
			Version:    dependency.Version(),
			Digest:     dependency.Digest(),
		}
		protoModuleNames[i] = protoModuleName
	}
	protoModule := &modulev1.Module{
		Files:        protoModuleFiles,
		Dependencies: protoModuleNames,
	}
	if err := validateProtoModule(protoModule); err != nil {
		return nil, err
	}
	return protoModule, nil
}

// ModuleDigestB1 returns the b1 digest for the module and module name.
//
// The digest on ModuleName must be unset.
// We might want an UnresolvedModuleName, need to see how this plays out.
// To create the module digest (SHA256):
// 	1. Add the string representation of the module version
// 	2. Add the dependency hashes (sorted lexicographically by the string representation)
// 	3. For every file in the module (sorted lexicographically by path):
// 		1. Add the file path
//		2. Add the file contents
//	4. Produce the final digest by URL-base64 encoding the summed bytes and prefixing it with the digest prefix
func ModuleDigestB1(
	ctx context.Context,
	moduleVersion string,
	module Module,
) (string, error) {
	hash := sha256.New()
	// Version must be part of the digest, since we require digests
	// to be unique per-repository.
	//
	// We do not include the server, owner, or repository here
	// as we want the ability to change the repository name or
	// change the repository owner without affecting the digest.
	if _, err := hash.Write([]byte(moduleVersion)); err != nil {
		return "", err
	}
	for _, dependency := range module.Dependencies() {
		if dependency.Digest() == "" {
			return "", NewNoDigestError(dependency)
		}
		// We include each of these individually as opposed to using String
		// so that if the String representation changes, we still get the same digest.
		//
		// Note that this does mean that changing a repository name or owner
		// will result in a different digest, this is something we may
		// want to revisit.
		if _, err := hash.Write([]byte(dependency.Server())); err != nil {
			return "", err
		}
		if _, err := hash.Write([]byte(dependency.Owner())); err != nil {
			return "", err
		}
		if _, err := hash.Write([]byte(dependency.Repository())); err != nil {
			return "", err
		}
		if _, err := hash.Write([]byte(dependency.Version())); err != nil {
			return "", err
		}
		if _, err := hash.Write([]byte(dependency.Digest())); err != nil {
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
		moduleFile, err := module.GetFile(ctx, sourceFileInfo.Path())
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
	return fmt.Sprintf("%s-%s", b1DigestPrefix, base64.URLEncoding.EncodeToString(hash.Sum(nil))), nil
}

// ModuleToBucket writes the given Module to the WriteBucket.
//
// This writes the sources and the buf.lock file.
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
		content, err := ReadModuleFile(ctx, module, fileInfo.Path())
		if err != nil {
			return err
		}
		if err := storage.PutPath(ctx, writeBucket, fileInfo.Path(), content); err != nil {
			return err
		}
	}
	// Create a lock file
	return putDependencies(ctx, writeBucket, module.Dependencies())
}

// ReadModuleFile reads the content for the .proto file at the given path in the Module.
//
// Returns an error that fufills storage.IsNotExist if the path does not exist.
func ReadModuleFile(
	ctx context.Context,
	module Module,
	path string,
) (_ []byte, retErr error) {
	moduleFile, err := module.GetFile(ctx, path)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, moduleFile.Close())
	}()
	return ioutil.ReadAll(moduleFile)
}

// DeduplicateModuleNames returns a deduplicated slice of module names
// by selecting the first occurrence of a module name based on the modules
// representation without a digest.
func DeduplicateModuleNames(moduleNames []ModuleName) []ModuleName {
	deduplicated := make([]ModuleName, 0, len(moduleNames))
	seenModuleNames := make(map[string]struct{})
	for _, moduleName := range moduleNames {
		moduleIdentity := moduleNameIdentity(moduleName)
		if _, ok := seenModuleNames[moduleIdentity]; ok {
			continue
		}
		seenModuleNames[moduleIdentity] = struct{}{}
		deduplicated = append(deduplicated, moduleName)
	}
	// It's important that we sort after we've deduplicated (and not before),
	// so that the first ModuleNames provided are prioritized over the ones
	// that follow.
	sortModuleNames(deduplicated)
	return deduplicated
}

// ValidateModuleNamesUnique returns an error if the module names contain
// any duplicates.
func ValidateModuleNamesUnique(moduleNames []ModuleName) error {
	seenModuleNames := make(map[string]struct{})
	for _, moduleName := range moduleNames {
		moduleIdentity := moduleNameIdentity(moduleName)
		if _, ok := seenModuleNames[moduleIdentity]; ok {
			return fmt.Errorf("module %s appeared twice", moduleIdentity)
		}
		seenModuleNames[moduleIdentity] = struct{}{}
	}
	return nil
}

// ResolvedModuleNameForModule returns a new validated ModuleName that uses the values
// from the given ModuleName and the digest from the Module.
//
// The given ModuleName must not already have a digest.
//
// This is just a convenience function.
func ResolvedModuleNameForModule(ctx context.Context, moduleName ModuleName, module Module) (ModuleName, error) {
	if moduleName.Digest() != "" {
		return nil, fmt.Errorf("module name to ResolvedModuleNameForModule already has a digest: %s", moduleName.String())
	}
	digest, err := ModuleDigestB1(ctx, moduleName.Version(), module)
	if err != nil {
		return nil, err
	}
	return NewModuleName(
		moduleName.Server(),
		moduleName.Owner(),
		moduleName.Repository(),
		moduleName.Version(),
		digest,
	)
}

// UnresolvedModuleName returns the ModuleName without a digest.
//
// This is just a convenience function.
func UnresolvedModuleName(moduleName ModuleName) (ModuleName, error) {
	if moduleName.Digest() == "" {
		return nil, fmt.Errorf("moduleName is already unresolved: %q", moduleName.String())
	}
	return NewModuleName(
		moduleName.Server(),
		moduleName.Owner(),
		moduleName.Repository(),
		moduleName.Version(),
		"",
	)
}

// ValidateModuleDigest validates that the Module matches the digest on ModuleName.
//
// The given ModuleName must have a digest.
//
// This is just a convenience function.
func ValidateModuleDigest(ctx context.Context, moduleName ModuleName, module Module) error {
	unresolvedModuleName, err := UnresolvedModuleName(moduleName)
	if err != nil {
		return err
	}
	digest, err := ModuleDigestB1(ctx, unresolvedModuleName.Version(), module)
	if err != nil {
		return err
	}
	if digest != moduleName.Digest() {
		return fmt.Errorf("mismatched module digest for %s: %s %s", unresolvedModuleName.String(), moduleName.Digest(), digest)
	}
	return nil
}

// ModuleNameEqual returns true if a equals b.
func ModuleNameEqual(a ModuleName, b ModuleName) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	if a == nil {
		return true
	}
	return a.Server() == b.Server() &&
		a.Owner() == b.Owner() &&
		a.Repository() == b.Repository() &&
		a.Version() == b.Version() &&
		a.Digest() == b.Digest()
}

// WriteModuleDependenciesToBucket writes the module dependencies to the write bucket in the form of a lock file.
func WriteModuleDependenciesToBucket(ctx context.Context, writeBucket storage.WriteBucket, module Module) error {
	return putDependencies(ctx, writeBucket, module.Dependencies())
}
