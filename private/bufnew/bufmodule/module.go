package bufmodule

import (
	"context"

	"github.com/bufbuild/buf/private/bufpkg/bufcas"
)

// Module presents a BSR module.
type Module interface {
	// ModuleInfo contains a Module's ModuleSetID, optional ModuleFullName, and optional commit ID.
	ModuleInfo
	// ModuleReadBucket allows for reading of a Module's files.
	//
	// A Module consists of .proto files, documentation file(s), and license file(s). All of these
	// are accessible via the functions on ModuleReadBucket. As such, the FileTypes() function will
	// return FileTypeProto, FileTypeDoc, FileTypeLicense.
	//
	// This bucket is not self-contained - it requires the files from dependencies to be so. As such,
	// IsProtoFilesSelfContained() returns false.
	//
	// This package currently exposes functionality to walk just the .proto files, and get the singular
	// documentation and license files, via WalkProtoFileInfos, GetDocFile, and GetLicenseFile.
	//
	// GetDocFile and GetLicenseFile may change in the future if other paths are accepted for
	// documentation or licenses, or if we allow multiple documentation or license files to
	// exist within a Module (currently, only one of each is allowed).
	ModuleReadBucket

	// ExternalDependencyModulePins returns the dependency list for this specific module that are not within the ModuleSet.
	//
	// This list is pruned - only Module that this Module actually depends on via import statements in its
	// files will be returned.
	//
	// Modules within the same ModuleSet will always have the same commits and digests for a given dependency, that is
	// no two Modules will have a different commit for the same dependency.
	//ExternalDependencyModulePins(ctx context.Context) ([]ModulePin, error)
	// ModuleSetDependencyModules returns the dependency list for Modules within the encompassing ModuleSet that this
	// Module depends on.
	//
	// This list is pruned - only Modules that this Module actually depends on via import statements
	// in its files will be returned.
	//ModuleSetDependencyModules(ctx context.Context) ([]Module, error)

	isModule()
}

// ModuleDigestB5 computes a b5 Digest for the given Module.
//
// A Module Digest is a composite Digest of all Module Files, and all Module dependencies.
//
// All Files are added to a bufcas.Manifest, which is then turned into a bufcas.Blob.
// The Digest of the Blob, along with all Digests of the dependencies, are then sorted,
// and then digested themselves as content.
//
// Note that the name of the Module and any of its dependencies has no effect on the Digest.
func ModuleDigestB5(ctx context.Context, module Module) (bufcas.Digest, error) {
	fileDigest, err := moduleReadBucketDigestB5(ctx, module)
	if err != nil {
		return nil, err
	}
	digests := []bufcas.Digest{fileDigest}
	// TODO: THIS IS WRONG. This doesn't include workspace deps. Rework this, see comment above.
	//for _, dep := range module.Deps() {
	//digests = append(digests, dep.Digest())
	//}
	return bufcas.NewDigestForDigests(digests)
}

// *** PRIVATE ***

type module struct{}
