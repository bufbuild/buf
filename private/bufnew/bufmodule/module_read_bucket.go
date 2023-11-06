package bufmodule

import (
	"context"
	"io/fs"

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
	// IsProtoFilesSelfContained returns true if the .proto files within the Bucket are guaranteed to only
	// import from each other. That is, an Image could be built from this Bucket directly.
	//
	// Note that this property may hold for this Bucket even if IsProtoFilesSelfContained() returns false,
	// but it will never be the case that IsProtoFilesSelfContained() returns true if the bucket is not self-contained.
	// A Bucket created from a ModuleSet will always be self-contained, a Bucket created from a Module
	// will never be self-contained.
	IsProtoFilesSelfContained() bool

	isModuleReadBucket()
}

// ModuleReadBucketToStorageReadBucket converts the given ModuleReadBucket to a storage.ReadBucket.
func ModuleReadBucketToStorageReadBucket(moduleReadBucket ModuleReadBucket) storage.ReadBucket {
	return newStorageReadBucket(moduleReadBucket)
}

// ModuleReadBucketWithOnlyFileTypes returns a new ModuleReadBucket that only contains the given
// FileTypes.
//
// Common use case is to get only the .proto files.
func ModuleReadBucketWithOnlyFileTypes(
	moduleReadBucket ModuleReadBucket,
	fileTypes ...FileType,
) ModuleReadBucket {
	return newFilteredModuleReadBucket(moduleReadBucket, fileTypes)
}

// ModuleReadBucketWithOnlyProtoFiles is a convenience function that returns a new
// ModuleReadBucket that only contains the .proto files.
func ModuleReadBucketWithOnlyProtoFiles(moduleReadBucket ModuleReadBucket) ModuleReadBucket {
	return ModuleReadBucketWithOnlyFileTypes(moduleReadBucket, FileTypeProto)
}

// GetDocFile gets the singular documentation File for the Module, if it exists.
//
// When creating a Module from a Bucket, we check the file paths buf.md, README.md, and README.markdown
// to exist, in that order. The first one to exist is chosen as the documentation File that is considered
// part of the Module, and any others are discarded. This function will return that File that was chosen.
//
// Returns an error with fs.ErrNotExist if no documentation file exists.
func GetDocFile(ctx context.Context, moduleReadBucket ModuleReadBucket) (File, error) {
	for _, docFilePath := range orderedDocFilePaths {
		if _, err := moduleReadBucket.StatFileInfo(ctx, docFilePath); err == nil {
			return moduleReadBucket.GetFile(ctx, docFilePath)
		}
	}
	return nil, fs.ErrNotExist
}

// GetLicenseFile gets the license File for the Module, if it exists.
//
// Returns an error with fs.ErrNotExist if the license File does not exist.
func GetLicenseFile(ctx context.Context, moduleReadBucket ModuleReadBucket) (File, error) {
	return moduleReadBucket.GetFile(ctx, licenseFilePath)
}

// *** PRIVATE ***

type storageReadBucket struct {
	delegate ModuleReadBucket
}

func newStorageReadBucket(delegate ModuleReadBucket) *storageReadBucket {
	return &storageReadBucket{
		delegate: delegate,
	}
}

func (b *storageReadBucket) Get(ctx context.Context, path string) (storage.ReadObjectCloser, error) {
	return b.delegate.GetFile(ctx, path)
}

func (b *storageReadBucket) Stat(ctx context.Context, path string) (storage.ObjectInfo, error) {
	return b.delegate.StatFileInfo(ctx, path)
}

func (b *storageReadBucket) Walk(ctx context.Context, prefix string, f func(storage.ObjectInfo) error) error {
	return b.delegate.WalkFileInfos(
		ctx,
		func(fileInfo FileInfo) error {
			if !normalpath.EqualsOrContainsPath(prefix, fileInfo.Path(), normalpath.Relative) {
				return nil
			}
			return f(fileInfo)
		},
	)
}

type filteredModuleReadBucket struct {
	delegate    ModuleReadBucket
	fileTypeMap map[FileType]struct{}
}

func newFilteredModuleReadBucket(
	delegate ModuleReadBucket,
	fileTypes []FileType,
) *filteredModuleReadBucket {
	// Filter out FileTypes that are not in the delegate.
	delegateFileTypeMap := fileTypeSliceToMap(delegate.FileTypes())
	fileTypeMap := fileTypeSliceToMap(fileTypes)
	for fileType := range fileTypeMap {
		if _, ok := delegateFileTypeMap[fileType]; !ok {
			delete(fileTypeMap, fileType)
		}
	}
	return &filteredModuleReadBucket{
		delegate:    delegate,
		fileTypeMap: fileTypeSliceToMap(fileTypes),
	}
}

func (f *filteredModuleReadBucket) GetFile(ctx context.Context, path string) (File, error) {
	// Stat'ing the filtered bucket, not the delegate.
	if _, err := f.StatFileInfo(ctx, path); err != nil {
		return nil, err
	}
	return f.delegate.GetFile(ctx, path)
}

func (f *filteredModuleReadBucket) StatFileInfo(ctx context.Context, path string) (FileInfo, error) {
	fileInfo, err := f.delegate.StatFileInfo(ctx, path)
	if err != nil {
		return nil, err
	}
	if _, ok := f.fileTypeMap[fileInfo.FileType()]; !ok {
		return nil, &fs.PathError{Op: "stat", Path: path, Err: fs.ErrNotExist}
	}
	return fileInfo, nil
}

func (f *filteredModuleReadBucket) WalkFileInfos(ctx context.Context, fn func(FileInfo) error) error {
	return f.delegate.WalkFileInfos(
		ctx,
		func(fileInfo FileInfo) error {
			if _, ok := f.fileTypeMap[fileInfo.FileType()]; !ok {
				return nil
			}
			return fn(fileInfo)
		},
	)
}

func (f *filteredModuleReadBucket) FileTypes() []FileType {
	return fileTypeMapToSortedSlice(f.fileTypeMap)
}

func (f *filteredModuleReadBucket) IsProtoFilesSelfContained() bool {
	if _, ok := f.fileTypeMap[FileTypeProto]; !ok {
		return false
	}
	return f.delegate.IsProtoFilesSelfContained()
}

func (*filteredModuleReadBucket) isModuleReadBucket() {}
