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
	"sort"
	"strings"
	"time"

	"github.com/bufbuild/buf/internal/buf/bufcore"
	modulev1alpha1 "github.com/bufbuild/buf/internal/gen/proto/go/buf/alpha/module/v1alpha1"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/uuidutil"
	"go.uber.org/multierr"
)

const (
	// LockFilePath defines the path to the lock file, relative to the root of the module.
	LockFilePath = "buf.lock"
	// MainBranch is the name of the branch created for every repository.
	// This is the default branch used if no branch or commit is specified.
	MainBranch = "main"

	b1DigestPrefix = "b1"
)

// ModuleFile is a module file.
type ModuleFile interface {
	bufcore.FileInfo
	io.ReadCloser

	isModuleFile()
}

// ModuleOwner is a module owner.
//
// It just contains remote, owner.
//
// This is shared by ModuleIdentity.
type ModuleOwner interface {
	Remote() string
	Owner() string

	isModuleOwner()
}

// NewModuleOwner returns a new ModuleOwner.
func NewModuleOwner(
	remote string,
	owner string,
) (ModuleOwner, error) {
	return newModuleOwner(remote, owner)
}

// ModuleOwnerForString returns a new ModuleOwner for the given string.
//
// This parses the path in the form remote/owner.
func ModuleOwnerForString(path string) (ModuleOwner, error) {
	slashSplit := strings.Split(path, "/")
	if len(slashSplit) != 2 {
		return nil, newInvalidModuleOwnerStringError(path)
	}
	remote := strings.TrimSpace(slashSplit[0])
	if remote == "" {
		return nil, newInvalidModuleIdentityStringError(path)
	}
	owner := strings.TrimSpace(slashSplit[1])
	if owner == "" {
		return nil, newInvalidModuleIdentityStringError(path)
	}
	return NewModuleOwner(remote, owner)
}

// ModuleIdentity is a module identity.
//
// It just contains remote, owner, repository.
//
// This is shared by ModuleReference and ModulePin.
type ModuleIdentity interface {
	ModuleOwner

	Repository() string

	// IdentityString is the string remote/owner/repository.
	IdentityString() string

	isModuleIdentity()
}

// NewModuleIdentity returns a new ModuleIdentity.
func NewModuleIdentity(
	remote string,
	owner string,
	repository string,
) (ModuleIdentity, error) {
	return newModuleIdentity(remote, owner, repository)
}

// ModuleIdentityForString returns a new ModuleIdentity for the given string.
//
// This parses the path in the form remote/owner/repository:{branch,commit}.
//
// TODO: we may want to add a special error if we detect / or @ as this may be a common mistake.
func ModuleIdentityForString(path string) (ModuleIdentity, error) {
	remote, owner, repository, err := parseModuleIdentityComponents(path)
	if err != nil {
		return nil, err
	}
	return NewModuleIdentity(remote, owner, repository)
}

// ModuleReference is a module reference.
//
// It references either a branch, or a commit.
// Only one of Branch and Commit will be set.
// Note that since commits belong to branches, we can deduce
// the branch from the commit when resolving.
type ModuleReference interface {
	ModuleIdentity

	// Prints either remote/owner/repository:{branch,commit}
	fmt.Stringer

	// only one of these will be set
	Branch() string
	// only one of these will be set
	Commit() string

	isModuleReference()
}

// NewBranchModuleReference returns a new validated ModuleReference for a branch.
func NewBranchModuleReference(
	remote string,
	owner string,
	repository string,
	branch string,
) (ModuleReference, error) {
	return newModuleReference(remote, owner, repository, branch, "")
}

// NewCommitModuleReference returns a new validated ModuleReference for a commit.
func NewCommitModuleReference(
	remote string,
	owner string,
	repository string,
	commit string,
) (ModuleReference, error) {
	return newModuleReference(remote, owner, repository, "", commit)
}

// NewModuleReferenceForProto returns a new ModuleReference for the given proto ModuleReference.
func NewModuleReferenceForProto(protoModuleReference *modulev1alpha1.ModuleReference) (ModuleReference, error) {
	return newModuleReferenceForProto(protoModuleReference)
}

// NewModuleReferencesForProtos maps the Protobuf equivalent into the internal representation.
func NewModuleReferencesForProtos(protoModuleReferences ...*modulev1alpha1.ModuleReference) ([]ModuleReference, error) {
	if len(protoModuleReferences) == 0 {
		return nil, nil
	}
	moduleReferences := make([]ModuleReference, len(protoModuleReferences))
	for i, protoModuleReference := range protoModuleReferences {
		moduleReference, err := NewModuleReferenceForProto(protoModuleReference)
		if err != nil {
			return nil, err
		}
		moduleReferences[i] = moduleReference
	}
	return moduleReferences, nil
}

// NewProtoModuleReferenceForModuleReference returns a new proto ModuleReference for the given ModuleReference.
func NewProtoModuleReferenceForModuleReference(moduleReference ModuleReference) *modulev1alpha1.ModuleReference {
	return newProtoModuleReferenceForModuleReference(moduleReference)
}

// NewProtoModuleReferencesForModuleReferences maps the given module references into the protobuf representation.
func NewProtoModuleReferencesForModuleReferences(moduleReferences ...ModuleReference) []*modulev1alpha1.ModuleReference {
	if len(moduleReferences) == 0 {
		return nil
	}
	protoModuleReferences := make([]*modulev1alpha1.ModuleReference, len(moduleReferences))
	for i, moduleReference := range moduleReferences {
		protoModuleReferences[i] = NewProtoModuleReferenceForModuleReference(moduleReference)
	}
	return protoModuleReferences
}

// ModuleReferenceForString returns a new ModuleReference for the given string.
// If a branch or commit is not provided, the "main" branch is used.
//
// This parses the path in the form remote/owner/repository{:branch,:commit}.
func ModuleReferenceForString(path string, options ...ModuleReferenceForStringOption) (ModuleReference, error) {
	moduleReferenceForStringOptions := newModuleReferenceForStringOptions()
	for _, option := range options {
		option(moduleReferenceForStringOptions)
	}
	remote, owner, repository, ref, err := parseModuleReferenceComponents(path)
	if err != nil {
		return nil, err
	}
	if ref == "" && moduleReferenceForStringOptions.requireBranch {
		return nil, fmt.Errorf("a branch is required in module reference path %q", path)
	}
	if ref == "" {
		// Default to the main branch if a ':' separator was not specified.
		return NewBranchModuleReference(remote, owner, repository, MainBranch)
	}
	if _, err := uuidutil.FromDashless(ref); err == nil {
		return NewCommitModuleReference(remote, owner, repository, ref)
	}
	return NewBranchModuleReference(remote, owner, repository, ref)
}

// ModuleReferenceForStringOption is an option for ModuleReferenceForString.
type ModuleReferenceForStringOption func(*moduleReferenceForStringOptions)

// ModuleReferenceForStringRequireBranch returns a new ModuleReferenceForStringOption that
// requires that a branch was specified in the module reference path.
//
// The default is to use MainBranch if a branch is not specified.
func ModuleReferenceForStringRequireBranch() ModuleReferenceForStringOption {
	return func(moduleReferenceForStringOptions *moduleReferenceForStringOptions) {
		moduleReferenceForStringOptions.requireBranch = true
	}
}

// BranchModuleReferenceForString returns a new ModuleReference for the given string.
// If a branch is not provided, the "main" branch is used.
//
// This parses the path in the form remote/owner/repository:branch.
// If a commit is provided, an error is returned.
func BranchModuleReferenceForString(path string, options ...BranchModuleReferenceForStringOption) (ModuleReference, error) {
	branchModuleReferenceForStringOptions := newBranchModuleReferenceForStringOptions()
	for _, option := range options {
		option(branchModuleReferenceForStringOptions)
	}
	remote, owner, repository, ref, err := parseModuleReferenceComponents(path)
	if err != nil {
		return nil, err
	}
	if ref == "" && branchModuleReferenceForStringOptions.requireBranch {
		return nil, fmt.Errorf("a branch is required in module reference path %q", path)
	}
	if ref == "" {
		// Default to the main branch if a ':' separator was not specified.
		return NewBranchModuleReference(remote, owner, repository, MainBranch)
	}
	return NewBranchModuleReference(remote, owner, repository, ref)
}

// BranchModuleReferenceForStringOption is an option for BranchModuleReferenceForString.
type BranchModuleReferenceForStringOption func(*branchModuleReferenceForStringOptions)

// BranchModuleReferenceForStringRequireBranch returns a new BranchModuleReferenceForStringOption that
// requires that a branch was specified in the module reference path.
//
// The default is to use MainBranch if a branch is not specified.
func BranchModuleReferenceForStringRequireBranch() BranchModuleReferenceForStringOption {
	return func(branchModuleReferenceForStringOptions *branchModuleReferenceForStringOptions) {
		branchModuleReferenceForStringOptions.requireBranch = true
	}
}

// CommitModuleReferenceForString returns a new ModuleReference for the given string.
//
// This parses the path in the form remote/owner/repository:commit.
// If a commit is not provided, an error is returned.
func CommitModuleReferenceForString(path string) (ModuleReference, error) {
	remote, owner, repository, ref, err := parseModuleReferenceComponents(path)
	if err != nil {
		return nil, err
	}
	if ref == "" {
		return nil, newInvalidModuleReferenceStringError(path)
	}
	return NewCommitModuleReference(remote, owner, repository, ref)
}

// ModulePin is a module pin.
//
// It references a specific point in time of a Module.
//
// Note that a commit does this itself, but we want all this information.
// This is what is stored in a buf.lock file.
type ModulePin interface {
	ModuleIdentity

	// Prints remote/owner/repository:commit, which matches ModuleReference
	fmt.Stringer

	// all of these will be set
	Branch() string
	Commit() string
	Digest() string
	CreateTime() time.Time

	isModulePin()
}

// NewModulePin returns a new validated ModulePin.
func NewModulePin(
	remote string,
	owner string,
	repository string,
	branch string,
	commit string,
	digest string,
	createTime time.Time,
) (ModulePin, error) {
	return newModulePin(remote, owner, repository, branch, commit, digest, createTime)
}

// NewModulePinForProto returns a new ModulePin for the given proto ModulePin.
func NewModulePinForProto(protoModulePin *modulev1alpha1.ModulePin) (ModulePin, error) {
	return newModulePinForProto(protoModulePin)
}

// NewModulePinsForProtos maps the Protobuf equivalent into the internal representation.
func NewModulePinsForProtos(protoModulePins ...*modulev1alpha1.ModulePin) ([]ModulePin, error) {
	if len(protoModulePins) == 0 {
		return nil, nil
	}
	modulePins := make([]ModulePin, len(protoModulePins))
	for i, protoModulePin := range protoModulePins {
		modulePin, err := NewModulePinForProto(protoModulePin)
		if err != nil {
			return nil, err
		}
		modulePins[i] = modulePin
	}
	return modulePins, nil
}

// NewProtoModulePinForModulePin returns a new proto ModulePin for the given ModulePin.
func NewProtoModulePinForModulePin(modulePin ModulePin) *modulev1alpha1.ModulePin {
	return newProtoModulePinForModulePin(modulePin)
}

// NewProtoModulePinsForModulePins maps the given module pins into the protobuf representation.
func NewProtoModulePinsForModulePins(modulePins ...ModulePin) []*modulev1alpha1.ModulePin {
	if len(modulePins) == 0 {
		return nil
	}
	protoModulePins := make([]*modulev1alpha1.ModulePin, len(modulePins))
	for i, modulePin := range modulePins {
		protoModulePins[i] = NewProtoModulePinForModulePin(modulePin)
	}
	return protoModulePins
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
	TargetFileInfos(ctx context.Context) ([]bufcore.FileInfo, error)
	// SourceFileInfos gets all FileInfos belonging to the module.
	//
	// It does not include dependencies.
	//
	// The returned SourceFileInfos are sorted by path.
	SourceFileInfos(ctx context.Context) ([]bufcore.FileInfo, error)
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
	DependencyModulePins() []ModulePin

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

// NewModuleForBucketWithDependencyModulePins explicitly specifies the dependencies
// that should be used when creating the Module. The module names must be resolved
// and unique.
func NewModuleForBucketWithDependencyModulePins(
	ctx context.Context,
	readBucket storage.ReadBucket,
	dependencyModulePins []ModulePin,
) (Module, error) {
	return newModuleForBucketWithDependencyModulePins(ctx, readBucket, dependencyModulePins)
}

// NewModuleForProto returns a new Module for the given proto Module.
func NewModuleForProto(
	ctx context.Context,
	protoModule *modulev1alpha1.Module,
) (Module, error) {
	return newModuleForProto(ctx, protoModule)
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
func ModuleWithTargetPaths(module Module, targetPaths []string) (Module, error) {
	return newTargetingModule(module, targetPaths, false)
}

// ModuleWithTargetPathsAllowNotExist returns a new Module specified specific file or directory paths to build,
// but allows the specified paths to not exist.
//
// Note that this will result in TargetFileInfos containing only these paths, and not
// any imports. Imports, and non-targeted files, are still available via SourceFileInfos.
func ModuleWithTargetPathsAllowNotExist(module Module, targetPaths []string) (Module, error) {
	return newTargetingModule(module, targetPaths, true)
}

// ModuleResolver resolves modules.
type ModuleResolver interface {
	// GetModulePin resolves the provided ModuleReference to a ModulePin.
	//
	// Returns an error that fufills storage.IsNotExist if the named Module does not exist.
	GetModulePin(ctx context.Context, moduleReference ModuleReference) (ModulePin, error)
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
	GetModule(ctx context.Context, modulePin ModulePin) (Module, error)
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
		protoModulePins[i] = NewProtoModulePinForModulePin(dependencyModulePin)
	}
	protoModule := &modulev1alpha1.Module{
		Files:        protoModuleFiles,
		Dependencies: protoModulePins,
	}
	if err := ValidateProtoModule(protoModule); err != nil {
		return nil, err
	}
	return protoModule, nil
}

// ModuleDigest returns the b1 digest for the Module.
//
// To create the module digest (SHA256):
// 	1. For every file in the module (sorted lexicographically by path):
// 		a. Add the file path
//		b. Add the file contents
// 	2. Add the dependency hashes (sorted lexicographically by the string representation)
//	3. Produce the final digest by URL-base64 encoding the summed bytes and prefixing it with the digest prefix
func ModuleDigest(ctx context.Context, module Module) (string, error) {
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
	return fmt.Sprintf("%s-%s", b1DigestPrefix, base64.URLEncoding.EncodeToString(hash.Sum(nil))), nil
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
	return putDependencyModulePinsToBucket(ctx, writeBucket, module.DependencyModulePins())
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

// ValidateModuleReferencesUniqueByIdentity returns an error if the module references contain any duplicates.
//
// This only checks remote, owner, repository.
func ValidateModuleReferencesUniqueByIdentity(moduleReferences []ModuleReference) error {
	seenModuleReferences := make(map[string]struct{})
	for _, moduleReference := range moduleReferences {
		moduleIdentityString := moduleReference.IdentityString()
		if _, ok := seenModuleReferences[moduleIdentityString]; ok {
			return fmt.Errorf("module %s appeared twice", moduleIdentityString)
		}
		seenModuleReferences[moduleIdentityString] = struct{}{}
	}
	return nil
}

// ValidateModulePinsUniqueByIdentity returns an error if the module pins contain any duplicates.
//
// This only checks remote, owner, repository.
func ValidateModulePinsUniqueByIdentity(modulePins []ModulePin) error {
	seenModulePins := make(map[string]struct{})
	for _, modulePin := range modulePins {
		moduleIdentityString := modulePin.IdentityString()
		if _, ok := seenModulePins[moduleIdentityString]; ok {
			return fmt.Errorf("module %s appeared twice", moduleIdentityString)
		}
		seenModulePins[moduleIdentityString] = struct{}{}
	}
	return nil
}

// ModuleReferenceEqual returns true if a equals b.
func ModuleReferenceEqual(a ModuleReference, b ModuleReference) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	if a == nil {
		return true
	}
	return a.Remote() == b.Remote() &&
		a.Owner() == b.Owner() &&
		a.Repository() == b.Repository() &&
		a.Branch() == b.Branch() &&
		a.Commit() == b.Commit()
}

// ModulePinEqual returns true if a equals b.
func ModulePinEqual(a ModulePin, b ModulePin) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	if a == nil {
		return true
	}
	return a.Remote() == b.Remote() &&
		a.Owner() == b.Owner() &&
		a.Repository() == b.Repository() &&
		a.Branch() == b.Branch() &&
		a.Commit() == b.Commit() &&
		a.Digest() == b.Digest() &&
		a.CreateTime().Equal(b.CreateTime())
}

// PutModuleDependencyModulePinsToBucket writes the module dependencies to the write bucket in the form of a lock file.
func PutModuleDependencyModulePinsToBucket(ctx context.Context, writeBucket storage.WriteBucket, module Module) error {
	// we know module dependency module pins are sorted and unique
	return putDependencyModulePinsToBucket(ctx, writeBucket, module.DependencyModulePins())
}

// SortModulePins sorts the ModulePins.
func SortModulePins(modulePins []ModulePin) {
	sort.Slice(modulePins, func(i, j int) bool {
		return modulePinLess(modulePins[i], modulePins[j])
	})
}

// parseModuleReferenceComponents parses and returns the remote, owner, repository,
// and ref (branch or commit) from the given path.
func parseModuleReferenceComponents(path string) (remote string, owner string, repository string, ref string, err error) {
	remote, owner, rest, err := parseModuleIdentityComponents(path)
	if err != nil {
		return "", "", "", "", newInvalidModuleReferenceStringError(path)
	}
	restSplit := strings.Split(rest, ":")
	repository = strings.TrimSpace(restSplit[0])
	if len(restSplit) == 1 {
		return remote, owner, repository, "", nil
	}
	if len(restSplit) == 2 {
		ref := strings.TrimSpace(restSplit[1])
		if ref == "" {
			return "", "", "", "", newInvalidModuleReferenceStringError(path)
		}
		return remote, owner, repository, ref, nil
	}
	return "", "", "", "", newInvalidModuleReferenceStringError(path)
}

func parseModuleIdentityComponents(path string) (remote string, owner string, repository string, err error) {
	slashSplit := strings.Split(path, "/")
	if len(slashSplit) != 3 {
		return "", "", "", newInvalidModuleIdentityStringError(path)
	}
	remote = strings.TrimSpace(slashSplit[0])
	if remote == "" {
		return "", "", "", newInvalidModuleIdentityStringError(path)
	}
	owner = strings.TrimSpace(slashSplit[1])
	if owner == "" {
		return "", "", "", newInvalidModuleIdentityStringError(path)
	}
	repository = strings.TrimSpace(slashSplit[2])
	if repository == "" {
		return "", "", "", newInvalidModuleIdentityStringError(path)
	}
	return remote, owner, repository, nil
}

type moduleReferenceForStringOptions struct {
	requireBranch bool
}

func newModuleReferenceForStringOptions() *moduleReferenceForStringOptions {
	return &moduleReferenceForStringOptions{}
}

type branchModuleReferenceForStringOptions struct {
	requireBranch bool
}

func newBranchModuleReferenceForStringOptions() *branchModuleReferenceForStringOptions {
	return &branchModuleReferenceForStringOptions{}
}
