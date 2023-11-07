package bufmodule

import (
	"context"
	"sync"

	"github.com/bufbuild/buf/private/bufpkg/bufcas"
)

// Module presents a BSR module.
type Module interface {
	// ModuleInfo contains a Module's optional ModuleFullName, optional commit ID, and Digest.
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

	// ModuleDeps returns the dependency list for this specific module.
	//
	// This list is pruned - only Modules that this Module actually depends on via import statements
	// within its .proto files will be returned.
	//
	// Colocated modules will always have the same commits and digests for a given dependency.
	ModuleDeps(ctx context.Context) ([]ModuleDep, error)

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
	moduleDeps, err := module.ModuleDeps(ctx)
	if err != nil {
		return nil, err
	}
	digests := []bufcas.Digest{fileDigest}
	for _, moduleDep := range moduleDeps {
		digest, err := moduleDep.Digest(ctx)
		if err != nil {
			return nil, err
		}
		digests = append(digests, digest)
	}

	// NewDigestForDigests deals with sorting.
	return bufcas.NewDigestForDigests(digests)
}

// *** PRIVATE ***

// module

type module struct {
	ModuleInfo
	ModuleReadBucket

	moduleDeps []ModuleDep
}

func newModule(
	moduleInfo ModuleInfo,
	moduleReadBucket ModuleReadBucket,
	moduleDeps []ModuleDep,
) *module {
	return &module{
		ModuleInfo:       moduleInfo,
		ModuleReadBucket: moduleReadBucket,
		moduleDeps:       moduleDeps,
	}
}

func (m *module) ModuleDeps(context.Context) ([]ModuleDep, error) {
	return m.moduleDeps, nil
}

func (*module) isModule() {}

// lazyModule

type lazyModule struct {
	ModuleInfo

	getModule func() (Module, error)
}

func newLazyModule(
	moduleInfo ModuleInfo,
	getModule func() (Module, error),
) Module {
	return &lazyModule{
		ModuleInfo: moduleInfo,
		getModule:  sync.OnceValues(getModule),
	}
}

func (m *lazyModule) GetFile(ctx context.Context, path string) (File, error) {
	module, err := m.getModule()
	if err != nil {
		return nil, err
	}
	return module.GetFile(ctx, path)
}

func (m *lazyModule) StatFileInfo(ctx context.Context, path string) (FileInfo, error) {
	module, err := m.getModule()
	if err != nil {
		return nil, err
	}
	return module.StatFileInfo(ctx, path)
}

func (m *lazyModule) WalkFileInfos(ctx context.Context, f func(FileInfo) error) error {
	module, err := m.getModule()
	if err != nil {
		return err
	}
	return module.WalkFileInfos(ctx, f)
}

func (m *lazyModule) ModuleDeps(ctx context.Context) ([]ModuleDep, error) {
	module, err := m.getModule()
	if err != nil {
		return nil, err
	}
	return module.ModuleDeps(ctx)
}

func (*lazyModule) isModuleReadBucket() {}
func (*lazyModule) isModule()           {}
