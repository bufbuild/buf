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

package bufmoduleref

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/bufpkg/buflock"
	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"go.uber.org/multierr"
)

const (
	// Main is the default reference used if no other reference is specified.
	Main = "main"
)

// FileInfo contains module file info.
type FileInfo interface {
	// Path is the path of the file relative to the root it is contained within.
	// This will be normalized, validated and never empty,
	// This will be unique within a given Image.
	Path() string
	// ExternalPath returns the path that identifies this file externally.
	//
	// This will be unnormalized.
	// Never empty. Falls back to Path if there is not an external path.
	//
	// Example:
	//	 Assume we had the input path /foo/bar which is a local directory.
	//   Path: one/one.proto
	//   RootDirPath: proto
	//   ExternalPath: /foo/bar/proto/one/one.proto
	ExternalPath() string
	// ModuleIdentity is the module that this file came from.
	//
	// Note this *can* be nil if we did not build from a named module.
	// All code must assume this can be nil.
	// Note that nil checking should work since the backing type is always a pointer.
	ModuleIdentity() ModuleIdentity
	// Commit is the commit for the module that this file came from.
	//
	// This will only be set if ModuleIdentity is set, but may not be set
	// even if ModuleIdentity is set, that is commit is optional information
	// even if we know what module this file came from.
	Commit() string

	isFileInfo()
}

// NewFileInfo returns a new FileInfo.
//
// TODO: we should make moduleIdentity and commit options.
// TODO: we don't validate commit
func NewFileInfo(
	path string,
	externalPath string,
	moduleIdentity ModuleIdentity,
	commit string,
) (FileInfo, error) {
	return newFileInfo(
		path,
		externalPath,
		moduleIdentity,
		commit,
	)
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
// This parses the path in the form remote/owner/repository
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
// It references either a branch, tag, or a commit.
// Note that since commits belong to branches, we can deduce
// the branch from the commit when resolving.
type ModuleReference interface {
	ModuleIdentity

	// Prints either remote/owner/repository:{branch,commit}
	// If the reference is equal to MainBranch, prints remote/owner/repository.
	fmt.Stringer

	// Either branch, tag, or commit
	Reference() string

	isModuleReference()
}

// NewModuleReference returns a new validated ModuleReference.
func NewModuleReference(
	remote string,
	owner string,
	repository string,
	reference string,
) (ModuleReference, error) {
	return newModuleReference(remote, owner, repository, reference)
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
// If a branch, commit, draft, or tag is not provided, the "main" branch is used.
//
// This parses the path in the form remote/owner/repository{:branch,:commit,:draft,:tag}.
func ModuleReferenceForString(path string) (ModuleReference, error) {
	remote, owner, repository, reference, err := parseModuleReferenceComponents(path)
	if err != nil {
		return nil, err
	}
	if reference == "" {
		// Default to the main branch if a ':' separator was not specified.
		reference = Main
	}
	return NewModuleReference(remote, owner, repository, reference)
}

// IsCommitModuleReference returns true if the ModuleReference references a commit.
//
// If false, this means the ModuleReference references a branch or tag.
// Branch and tag disambiguation needs to be done server-side.
func IsCommitModuleReference(moduleReference ModuleReference) bool {
	return IsCommitReference(moduleReference.Reference())
}

// IsCommitReference returns whether the provided reference is a commit.
func IsCommitReference(reference string) bool {
	_, err := uuidutil.FromDashless(reference)
	return err == nil
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
	Commit() string
	Digest() string

	isModulePin()
}

// NewModulePin returns a new validated ModulePin.
func NewModulePin(
	remote string,
	owner string,
	repository string,
	commit string,
	digest string,
) (ModulePin, error) {
	return newModulePin(remote, owner, repository, commit, digest)
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

// ValidateModulePinsConsistentDigests verifies that module pins to the same commit don't change digests.
// This is important to avoid MITM issues, where the module digest stored in a buf.lock file doesn't match
// the module pin returned from the BSR.
// Returns an error that fulfills IsDigestChanged if any valid digest changed from the buf.lock file for
// the same dependency commit.
func ValidateModulePinsConsistentDigests(
	ctx context.Context,
	bucket storage.ReadBucket,
	modulePins []ModulePin,
) error {
	currentConfig, err := buflock.ReadConfig(ctx, bucket)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	if len(currentConfig.Dependencies) == 0 {
		return nil
	}
	currentIdentityAndCommitToDigest := make(map[string]string, len(currentConfig.Dependencies))
	for _, dep := range currentConfig.Dependencies {
		// Ignore dependencies with no digest
		if dep.Digest == "" {
			continue
		}
		// Ignore dependencies with an invalid digest.
		// We want to replace these with a valid digest.
		if _, err := bufcas.ParseDigest(dep.Digest); err != nil {
			continue
		}
		key := fmt.Sprintf("%s/%s/%s:%s", dep.Remote, dep.Owner, dep.Repository, dep.Commit)
		currentIdentityAndCommitToDigest[key] = dep.Digest
	}
	var changedErrors error
	for _, pin := range modulePins {
		if pin.Digest() == "" {
			continue
		}
		if currentDigest, ok := currentIdentityAndCommitToDigest[pin.String()]; ok && currentDigest != pin.Digest() {
			changedErrors = multierr.Append(changedErrors, &digestChangedError{
				currentDigest: currentDigest,
				updatedPin:    pin,
			})
		}
	}
	return changedErrors
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
		a.Reference() == b.Reference()
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
		a.Commit() == b.Commit() &&
		a.Digest() == b.Digest()
}

// DependencyModulePinsForBucket reads the module dependencies from the lock file in the bucket.
func DependencyModulePinsForBucket(
	ctx context.Context,
	readBucket storage.ReadBucket,
) ([]ModulePin, error) {
	lockFile, err := buflock.ReadConfig(ctx, readBucket)
	if err != nil {
		return nil, fmt.Errorf("failed to read lock file: %w", err)
	}
	modulePins := make([]ModulePin, 0, len(lockFile.Dependencies))
	for _, dep := range lockFile.Dependencies {
		modulePin, err := NewModulePin(
			dep.Remote,
			dep.Owner,
			dep.Repository,
			dep.Commit,
			dep.Digest,
		)
		if err != nil {
			return nil, err
		}
		modulePins = append(modulePins, modulePin)
	}
	// just to be safe
	SortModulePins(modulePins)
	if err := ValidateModulePinsUniqueByIdentity(modulePins); err != nil {
		return nil, err
	}
	return modulePins, nil
}

// PutDependencyModulePinsToBucket writes the module dependencies to the write bucket in the form of a lock file.
func PutDependencyModulePinsToBucket(
	ctx context.Context,
	writeBucket storage.WriteBucket,
	modulePins []ModulePin,
) error {
	if err := ValidateModulePinsUniqueByIdentity(modulePins); err != nil {
		return err
	}
	SortModulePins(modulePins)
	lockFile := &buflock.Config{
		Dependencies: make([]buflock.Dependency, 0, len(modulePins)),
	}
	for _, pin := range modulePins {
		lockFile.Dependencies = append(
			lockFile.Dependencies,
			buflock.Dependency{
				Remote:     pin.Remote(),
				Owner:      pin.Owner(),
				Repository: pin.Repository(),
				Commit:     pin.Commit(),
				Digest:     pin.Digest(),
			},
		)
	}
	return buflock.WriteConfig(ctx, writeBucket, lockFile)
}

// SortFileInfos sorts the FileInfos by Path.
//
// This should be treated as the default sorting mechanism.
func SortFileInfos(fileInfos []FileInfo) {
	if len(fileInfos) == 0 {
		return
	}
	sort.Slice(
		fileInfos,
		func(i int, j int) bool {
			return fileInfos[i].Path() < fileInfos[j].Path()
		},
	)
}

// SortFileInfosByExternalPath sorts the FileInfos by ExternalPath.
func SortFileInfosByExternalPath(fileInfos []FileInfo) {
	if len(fileInfos) == 0 {
		return
	}
	sort.Slice(
		fileInfos,
		func(i int, j int) bool {
			return fileInfos[i].ExternalPath() < fileInfos[j].ExternalPath()
		},
	)
}

// SortModuleReferences sorts the ModuleReferences lexicographically by their identity.
func SortModuleReferences(references []ModuleReference) {
	sort.Slice(references, func(i, j int) bool {
		return references[i].IdentityString() < references[j].IdentityString()
	})
}

// SortModulePins sorts the ModulePins.
func SortModulePins(modulePins []ModulePin) {
	sort.Slice(modulePins, func(i, j int) bool {
		return modulePinLess(modulePins[i], modulePins[j])
	})
}

// IsDigestChanged returns true if the error indicates an unexpected digest change.
func IsDigestChanged(err error) bool {
	var errDigestChanged *digestChangedError
	return errors.As(err, &errDigestChanged)
}

// digestChangedError is returned if module pin digests have changed unexpectedly.
type digestChangedError struct {
	// currentDigest is the digest found in the buf.lock file.
	currentDigest string
	// updatedPin is a module pin with a different digest than currentDigest for the same commit.
	updatedPin ModulePin
}

func (e *digestChangedError) Error() string {
	return fmt.Sprintf(
		"module %s commit %q returned an unexpected digest: local buf.lock=%q, remote=%q",
		e.updatedPin.IdentityString(),
		e.updatedPin.Commit(),
		e.currentDigest,
		e.updatedPin.Digest(),
	)
}
