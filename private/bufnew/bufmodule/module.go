package bufmodule

import (
	"context"

	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"go.uber.org/multierr"
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
	// IsSelfContained() returns false.
	//
	// This package currently exposes functionality to walk just the .proto files, and get the singular
	// documentation and license files, via WalkProtoFileInfos, GetDocFile, and GetLicenseFile.
	//
	// GetDocFile and GetLicenseFile may change in the future if other paths are accepted for
	// documentation or licenses, or if we allow multiple documentation or license files to
	// exist within a Module (currently, only one of each is allowed).
	ModuleReadBucket

	// ModuleSet returns the ModuleSet that encompasses this Module.
	//
	// Even in the case of a single Module, all Modules will have an encompassing ModuleSet,
	// including ith buf.yaml v1 with no corresponding buf.work.yaml.
	ModuleSet() ModuleSet
	// ExternalDependencyModulePins returns the dependency list for this specific module that are not within the ModuleSet.
	//
	// This list is pruned - only Module that this Module actually depends on via import statements in its
	// files will be returned.
	//
	// Modules within the same ModuleSet will always have the same commits and digests for a given dependency, that is
	// no two Modules will have a different commit for the same dependency.
	ExternalDependencyModulePins(ctx context.Context) ([]ModulePin, error)
	// ModuleSetDependencyModules returns the dependency list for Modules within the encompassing ModuleSet that this
	// Module depends on.
	//
	// This list is pruned - only Modules that this Module actually depends on via import statements
	// in its files will be returned.
	ModuleSetDependencyModules(ctx context.Context) ([]Module, error)

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
	var fileNodes []bufcas.FileNode
	if err := module.WalkFileInfos(
		ctx,
		func(fileInfo FileInfo) (retErr error) {
			file, err := module.GetFile(ctx, fileInfo.Path())
			if err != nil {
				return err
			}
			defer func() {
				retErr = multierr.Append(retErr, file.Close())
			}()
			digest, err := bufcas.NewDigestForContent(file)
			if err != nil {
				return err
			}
			fileNode, err := bufcas.NewFileNode(fileInfo.Path(), digest)
			if err != nil {
				return err
			}
			fileNodes = append(fileNodes, fileNode)
			return nil
		},
	); err != nil {
		return nil, err
	}
	manifest, err := bufcas.NewManifest(fileNodes)
	if err != nil {
		return nil, err
	}
	manifestBlob, err := bufcas.ManifestToBlob(manifest)
	if err != nil {
		return nil, err
	}
	digests := []bufcas.Digest{manifestBlob.Digest()}
	// TODO: THIS IS WRONG. This doesn't include workspace deps. Rework this, see comment above.
	//for _, dep := range module.Deps() {
	//digests = append(digests, dep.Digest())
	//}
	return bufcas.NewDigestForDigests(digests)
}

// *** PRIVATE ***

type module struct{}
