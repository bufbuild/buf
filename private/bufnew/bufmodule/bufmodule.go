package bufmodule

import (
	"context"
	"errors"
	"io"
	"io/fs"

	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"go.uber.org/multierr"
)

const (
	// licenseFilePath is the path of the license file within a Module.
	licenseFilePath = "LICENSE"
)

var (
	// ErrNotExist is the error returned if a File retrieved does not exist.
	//
	// Ese errors.Is(err,
	ErrNotExist = errors.New("file does not exist")

	// orderedDocFilePaths are the potential documentation file paths for a Module.
	//
	// When creating a Module from a Bucket, we check the file paths buf.md, README.md, and README.markdown
	// to exist, in that order. The first one to exist is chosen as the documentation file that is considered
	// part of the Module, and any others are discarded.
	orderedDocFilePaths = []string{
		"buf.md",
		"README.md",
		"README.markdown",
	}
)

// ModuleFullName represents the full name of the Module, including its remote, owner, and name.
type ModuleFullName interface {
	Remote() string
	Owner() string
	Name() string

	isModuleFullName()
}

// ModuleRef is an unresolved reference to a Module.
//
// It can refer to the latest released commit, a different commit, a branch, a tag, or a VCS commit.
type ModuleRef interface {
	// ModuleFullName returns the full name of the Module.
	ModuleFullName() ModuleFullName
	// Ref returns the reference within the Module.
	//
	//   If Ref is empty, this refers to the latest released Commit on the Module.
	//   If Ref is a commit ID, this refers to this commit.
	//   If Ref is a tag ID or name, this refers to the commit associated with the tag.
	//   If Ref is a VCS commit ID or hash, this refers to the commit associated with the VCS commit.
	//   If Ref is a branch ID or name, this refers to the latest commit on the branch.
	//     If there is a conflict between names across resources (for example, there is a
	//     branch and tag with the same name), the following order of precedence is applied:
	//       - commit
	//       - VCS commit
	//       - tag
	//       - branch
	Ref() string

	isModuleReference()
}

// ModulePin is a specific Module and commit, along with its associated Digest.
type ModulePin interface {
	// ModuleFullName returns the full name of the Module.
	ModuleFullName() ModuleFullName
	// CommitID returns the commit ID.
	//
	// This can be used as a Commit ID within the BSR.
	CommitID() string
	// Digest returns the Digest of the Module at the specific commit.
	//
	// ModuleDigestB5 is currently used to calculate Digests.
	Digest() bufcas.Digest

	isModulePin()
}

// ModuleSet is a set of Modules and their associated dependencies.
//
// Within the CLI, this is the set of Modules that comprises a workspace.
// With buf.yaml v2, we have a common set of dependencies for a workspace, however
// we do not in v1. We denote this via IsDepsOnModule.
type ModuleSet interface {
	// Modules returns the Modules within this ModuleSet.
	Modules() []Module
	// Deps returns the common set of dependencies if IsDepsOnModule is true.
	//
	// No dependency within Deps will reference a Module within Modules.
	Deps() []ModulePin
	// IsDepsOnModule returns true if the Deps (if any) are represented on the ModuleSet, and not on the Module.
	//
	// With buf.yaml v2, this will return true.
	// With buf.yaml v1, this will return false.
	//
	// If this is true, use Deps() on the Module to get the Module Deps.
	// Note that if this is true, workspace push is not supported.
	IsDepsOnModule() bool

	isModuleSet()
}

// FileInfo is the file info for a Module file.
//
// It comprises the typical storage.ObjectInfo, along with a pointer back to the Module.
// This allows callers to figure out i.e. the Module ID, FullName, Commit, as well as any other
// data it may need.
type FileInfo interface {
	storage.ObjectInfo

	// Module returns the Module that contains this file.
	Module() Module

	isFileInfo()
}

// File is a file within a Module.
type File interface {
	FileInfo
	io.ReadCloser

	isFile()
}

type Module interface {
	// GetFile gets the File within the Module as specified by the path.
	//
	// Returns an error with fs.ErrNotExist if the path is not part of the Module.
	GetFile(ctx context.Context, path string) (File, error)
	// StatFileInfo gets the FileInfo for the File within the Module as specified by the path.
	//
	// Returns an error with fs.ErrNotExist if the path is not part of the Module.
	StatFileInfo(ctx context.Context, path string) (FileInfo, error)
	// WalkFileInfos walks all Files in the Module, passing the FileInfo to a specified function.
	//
	// This will walk the .proto files, documentation file(s), and license files(s). This package
	// currently exposes functionality to walk just the .proto files, and get the singular
	// documentation and license files, via WalkProtoFileInfos, GetDocFile, and GetLicenseFile.
	//
	// GetDocFile and GetLicenseFile may change in the future if other paths are accepted for
	// documentation or licenses, or if we allow multiple documentation or license files to
	// exist within a Module (currently, only one of each is allowed).
	WalkFileInfos(ctx context.Context, f func(FileInfo) error) error
	// ModuleSet returns the ModuleSet that encompasses this Module.
	//
	// Even in the case of a single Module, all Modules will have an encompassing ModuleSet,
	// including ith buf.yaml v1 with no corresponding buf.work.yaml.
	ModuleSet() ModuleSet
	// Deps returns the Dependency list for this specific module.
	//
	// TODO: define what happens here if IsDepsOnModule is false - does this proxy to ModuleSet.Deps()? If so, you need to redefine the function.
	// TODO: You CANNOT base the ModuleDigestB5 on this. This does not contain the digests for workspace deps.
	// Perhaps add a ModuleSetDeps() []Module?
	Deps() []ModulePin

	Ref() string
	FullName() ModuleFullName
	CommitID() string

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
	for _, dep := range module.Deps() {
		digests = append(digests, dep.Digest())
	}
	return bufcas.NewDigestForDigests(digests)
}

// *** PRIVATE ***

type moduleBucket struct {
	module Module
}

func newModuleBucket(module Module) *moduleBucket {
	return &moduleBucket{
		module: module,
	}
}

func (b *moduleBucket) Get(ctx context.Context, path string) (storage.ReadObjectCloser, error) {
	return b.module.GetFile(ctx, path)
}

func (b *moduleBucket) Stat(ctx context.Context, path string) (storage.ObjectInfo, error) {
	return b.module.StatFileInfo(ctx, path)
}

func (b *moduleBucket) Walk(ctx context.Context, prefix string, f func(storage.ObjectInfo) error) error {
	return b.module.WalkFileInfos(
		ctx,
		func(fileInfo FileInfo) error {
			if !normalpath.EqualsOrContainsPath(prefix, fileInfo.Path(), normalpath.Relative) {
				return nil
			}
			return f(fileInfo)
		},
	)
}
