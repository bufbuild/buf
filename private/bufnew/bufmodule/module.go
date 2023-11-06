package bufmodule

import (
	"context"
	"io/fs"

	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
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

// ModuleToBucket converts the given Module to a storage.ReadBucket.
func ModuleToBucket(module Module) storage.ReadBucket {
	return newModuleBucket(module)
}

// WalkModuleProtoFileInfos is a convenience function that walks just the .proto FileInfos within a Module.
func WalkModuleProtoFileInfos(ctx context.Context, module Module, f func(FileInfo) error) error {
	return module.WalkFileInfos(
		ctx,
		func(fileInfo FileInfo) error {
			if normalpath.Ext(fileInfo.Path()) != ".proto" {
				return nil
			}
			return f(fileInfo)
		},
	)
}

// GetDocFile gets the singular documentation File for the Module, if it exists.
//
// When creating a Module from a Bucket, we check the file paths buf.md, README.md, and README.markdown
// to exist, in that order. The first one to exist is chosen as the documentation File that is considered
// part of the Module, and any others are discarded. This function will return that File that was chosen.
//
// Returns an error with fs.ErrNotExist if no documentation file exists.
func GetModuleDocFile(ctx context.Context, module Module) (File, error) {
	for _, docFilePath := range orderedDocFilePaths {
		if _, err := module.StatFileInfo(ctx, docFilePath); err == nil {
			return module.GetFile(ctx, docFilePath)
		}
	}
	return nil, fs.ErrNotExist
}

// GetModuleLicenseFile gets the license File for the Module, if it exists.
//
// Returns an error with fs.ErrNotExist if the license File does not exist.
func GetModuleLicenseFile(ctx context.Context, module Module) (File, error) {
	return module.GetFile(ctx, licenseFilePath)
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
	if err := module.ModuleReadBucket().WalkFileInfos(
		ctx,
		func(fileInfo FileInfo) (retErr error) {
			file, err := module.ModuleReadBucket().GetFile(ctx, fileInfo.Path())
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
	for _, dep := range module.Deps() {
		digests = append(digests, dep.Digest())
	}
	return bufcas.NewDigestForDigests(digests)
}
