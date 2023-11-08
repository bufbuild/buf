package bufmodule

import (
	"context"
	"errors"
	"fmt"
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

	// DepModules returns the dependency list for this specific module.
	//
	// This list is pruned - only Modules that this Module actually depends on via import statements
	// within its .proto files will be returned.
	//
	// Dependencies with the same ModuleFullName will always have the same commits and digests.
	DepModules(ctx context.Context) ([]Module, error)

	addPotentialDepModules(...Module)
	isModule()
}

// *** PRIVATE ***

// module

type module struct {
	ModuleReadBucket

	moduleFullName ModuleFullName
	commitID       string

	getDigest     func() (bufcas.Digest, error)
	getDepModules func() ([]Module, error)

	potentialDepModules []Module
}

// must set ModuleReadBucket after constructor via setModuleReadBucket
func newModule(
	ctx context.Context,
	moduleFullName ModuleFullName,
	commitID string,
) *module {
	module := &module{
		moduleFullName: moduleFullName,
		commitID:       commitID,
	}
	module.getDigest = sync.OnceValues(
		func() (bufcas.Digest, error) {
			return moduleDigestB5(ctx, module)
		},
	)
	module.getDepModules = sync.OnceValues(
		func() ([]Module, error) {
			return getActualDepModules(ctx, module, module.potentialDepModules)
		},
	)
	return module
}

func (m *module) ModuleFullName() ModuleFullName {
	return m.moduleFullName
}

func (m *module) CommitID() string {
	return m.commitID
}

func (m *module) Digest() (bufcas.Digest, error) {
	return m.getDigest()
}

func (m *module) DepModules(ctx context.Context) ([]Module, error) {
	return m.getDepModules()
}

func (m *module) addPotentialDepModules(depModules ...Module) {
	m.potentialDepModules = append(m.potentialDepModules, depModules...)
}

func (m *module) setModuleReadBucket(moduleReadBucket ModuleReadBucket) {
	m.ModuleReadBucket = moduleReadBucket
}

func (*module) isModuleInfo() {}
func (*module) isModule()     {}

// lazyModule

type lazyModule struct {
	ModuleInfo

	getModuleAndDigest func() (Module, bufcas.Digest, error)
	getDepModules      func() ([]Module, error)

	potentialDepModules []Module
}

func newLazyModule(
	ctx context.Context,
	moduleInfo ModuleInfo,
	getModuleFunc func() (Module, error),
) Module {
	lazyModule := &lazyModule{
		ModuleInfo: moduleInfo,
		getModuleAndDigest: onceThreeValues(
			func() (Module, bufcas.Digest, error) {
				module, err := getModuleFunc()
				if err != nil {
					return nil, nil, err
				}
				expectedDigest, err := moduleInfo.Digest()
				if err != nil {
					return nil, nil, err
				}
				actualDigest, err := module.Digest()
				if err != nil {
					return nil, nil, err
				}
				if !bufcas.DigestEqual(expectedDigest, actualDigest) {
					return nil, nil, fmt.Errorf("expected digest %v, got %v", expectedDigest, actualDigest)
				}
				return module, actualDigest, nil
			},
		),
	}
	lazyModule.getDepModules = sync.OnceValues(
		func() ([]Module, error) {
			module, _, err := lazyModule.getModuleAndDigest()
			if err != nil {
				return nil, err
			}
			potentialDepModules, err := module.DepModules(ctx)
			if err != nil {
				return nil, err
			}
			// Prefer declared dependencies first, as these are not ready from remote.
			return getActualDepModules(ctx, lazyModule, append(lazyModule.potentialDepModules, potentialDepModules...))
		},
	)
	return lazyModule
}

func (m *lazyModule) Digest() (bufcas.Digest, error) {
	_, digest, err := m.getModuleAndDigest()
	return digest, err
}

func (m *lazyModule) GetFile(ctx context.Context, path string) (File, error) {
	module, _, err := m.getModuleAndDigest()
	if err != nil {
		return nil, err
	}
	return module.GetFile(ctx, path)
}

func (m *lazyModule) StatFileInfo(ctx context.Context, path string) (FileInfo, error) {
	module, _, err := m.getModuleAndDigest()
	if err != nil {
		return nil, err
	}
	return module.StatFileInfo(ctx, path)
}

func (m *lazyModule) WalkFileInfos(ctx context.Context, f func(FileInfo) error) error {
	module, _, err := m.getModuleAndDigest()
	if err != nil {
		return err
	}
	return module.WalkFileInfos(ctx, f)
}

func (m *lazyModule) DepModules(ctx context.Context) ([]Module, error) {
	return m.getDepModules()
}

func (m *lazyModule) addPotentialDepModules(depModules ...Module) {
	m.potentialDepModules = append(m.potentialDepModules, depModules...)
}

func (*lazyModule) isModuleReadBucket() {}
func (*lazyModule) isModule()           {}

// moduleDigestB5 computes a b5 Digest for the given Module.
//
// A Module Digest is a composite Digest of all Module Files, and all Module dependencies.
//
// All Files are added to a bufcas.Manifest, which is then turned into a bufcas.Blob.
// The Digest of the Blob, along with all Digests of the dependencies, are then sorted,
// and then digested themselves as content.
//
// Note that the name of the Module and any of its dependencies has no effect on the Digest.
func moduleDigestB5(ctx context.Context, module Module) (bufcas.Digest, error) {
	fileDigest, err := moduleReadBucketDigestB5(ctx, module)
	if err != nil {
		return nil, err
	}
	depModules, err := module.DepModules(ctx)
	if err != nil {
		return nil, err
	}
	digests := []bufcas.Digest{fileDigest}
	for _, depModule := range depModules {
		digest, err := depModule.Digest()
		if err != nil {
			return nil, err
		}
		digests = append(digests, digest)
	}

	// NewDigestForDigests deals with sorting.
	return bufcas.NewDigestForDigests(digests)
}

// getActualDepModules gets the actual dependencies for the Module  from the potential dependency list.
//
// TODO: go through imports, figure out which dep modules contain those imports, return just that list
// Make sure to memoize file -> imports mapping, and pass it around the ModuleBuilder.
func getActualDepModules(
	ctx context.Context,
	moduleReadBucket ModuleReadBucket,
	potentialDepModules []Module,
) ([]Module, error) {
	potentialDepModules, err := getUniqueModulesWithEarlierPreferred(ctx, potentialDepModules)
	if err != nil {
		return nil, err
	}
	return nil, errors.New("TODO")
}

// uniqueModulesWithEarlierPreferred deduplicates the Module list with the earlier modules being preferred.
//
// Callers should put modules built from local sources earlier than Modules built from remote sources.
//
// Duplication determined based ModuleFullName and on Digest, that is if a Module has an equal
// ModuleFullName, or an equal Digest, it is considered a duplicate.
//
// We want to account for Modules with the same name but different digests, that is a dep in a workspace
// that has the same name as something in a buf.lock file, we prefer the local dep in the workspace.
func getUniqueModulesWithEarlierPreferred(ctx context.Context, modules []Module) ([]Module, error) {
	alreadySeenModuleFullNameStrings := make(map[string]struct{})
	alreadySeenDigestStrings := make(map[string]struct{})
	uniqueModules := make([]Module, 0, len(modules))
	for _, module := range modules {
		var moduleFullNameString string
		if moduleFullName := module.ModuleFullName(); moduleFullName != nil {
			moduleFullNameString = moduleFullName.String()
		}
		digest, err := module.Digest()
		if err != nil {
			return nil, err
		}
		digestString := digest.String()

		var alreadySeenModuleByName bool
		if moduleFullNameString != "" {
			_, alreadySeenModuleByName = alreadySeenModuleFullNameStrings[moduleFullNameString]
		}
		_, alreadySeenModulebyDigest := alreadySeenDigestStrings[digestString]

		alreadySeenModuleFullNameStrings[moduleFullNameString] = struct{}{}
		alreadySeenDigestStrings[digestString] = struct{}{}

		if !alreadySeenModuleByName && !alreadySeenModulebyDigest {
			uniqueModules = append(uniqueModules, module)
		}
	}
	return nil, errors.New("TODO")
}

// onceThreeValues returns a function that invokes f only once and returns the values
// returned by f. The returned function may be called concurrently.
//
// If f panics, the returned function will panic with the same value on every call.
//
// This is copied from sync.OnceValues and extended to for three values.
func onceThreeValues[T1, T2, T3 any](f func() (T1, T2, T3)) func() (T1, T2, T3) {
	var (
		once  sync.Once
		valid bool
		p     any
		r1    T1
		r2    T2
		r3    T3
	)
	g := func() {
		defer func() {
			p = recover()
			if !valid {
				panic(p)
			}
		}()
		r1, r2, r3 = f()
		valid = true
	}
	return func() (T1, T2, T3) {
		once.Do(g)
		if !valid {
			panic(p)
		}
		return r1, r2, r3
	}
}
