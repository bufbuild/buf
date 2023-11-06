package bufmodule

import (
	"context"

	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
)

// ModuleReadBucket is an object analogous to storage.ReadBucket that supplements ObjectInfos
// and Objects with the data on the Module that supplied them.
//
// ModuleReadBuckets talk in terms of Files and FileInfos. They are easily converted into
// storage.ReadBuckets.
//
// The contents of a ModuleReadBucket are specific to its context. In the context of a Module,
// a ModuleReadBucket will return .proto files, documentation file(s), and license file(s). However,
// in the context of converting a ModuleSet into its corresponding .proto files, a ModuleReadBucket
// will only contain .proto files.
type ModuleReadBucket interface {
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
	// FileTypes returns the possible FileTypes that this Bucket may contain.
	//
	// Note that the Bucket may not contain all of these FileTypes - for example, a Bucket returned
	// from a Module will contain all FileTypes, but may not have FileTypeDoc or FileTypeLicense.
	FileTypes() []FileType
	// IsSelfContained returns true if the .proto files within the Bucket are guaranteed to only
	// import from each other. That is, an Image could be built from this Bucket directly.
	//
	// Note that this property may hold for this Bucket even if IsSelfContained() returns false,
	// but it will never be the case that IsSelfContained() returns true if the bucket is not self-contained.
	// A Bucket created from a ModuleSet will always be self-contained, a Bucket created from a Module
	// will never be self-contained.
	IsSelfContained() bool

	isModuleReadBucket()
}

// *** PRIVATE ***

type storageReadBucket struct {
	moduleReadBucket ModuleReadBucket
}

func newStorageReadBucket(moduleReadBucket ModuleReadBucket) *storageReadBucket {
	return &storageReadBucket{
		moduleReadBucket: moduleReadBucket,
	}
}

func (b *storageReadBucket) Get(ctx context.Context, path string) (storage.ReadObjectCloser, error) {
	return b.moduleReadBucket.GetFile(ctx, path)
}

func (b *storageReadBucket) Stat(ctx context.Context, path string) (storage.ObjectInfo, error) {
	return b.moduleReadBucket.StatFileInfo(ctx, path)
}

func (b *storageReadBucket) Walk(ctx context.Context, prefix string, f func(storage.ObjectInfo) error) error {
	return b.moduleReadBucket.WalkFileInfos(
		ctx,
		func(fileInfo FileInfo) error {
			if !normalpath.EqualsOrContainsPath(prefix, fileInfo.Path(), normalpath.Relative) {
				return nil
			}
			return f(fileInfo)
		},
	)
}
