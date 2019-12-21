// Package bufbuild drives the building of Protobuf files.
package bufbuild

import (
	"context"

	"github.com/bufbuild/buf/internal/buf/bufpb"
	"github.com/bufbuild/buf/internal/pkg/analysis"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"go.uber.org/zap"
)

// ProtoRootFilePathResolver resolves root file paths from real file paths.
type ProtoRootFilePathResolver interface {
	// GetRootFilePath returns the root file path for the real file path, if it exists.
	// If it does not exist, the empty string is returned.
	//
	// The input path is normalized and validated, and checked for empty.
	GetRootFilePath(realFilePath string) (string, error)
}

// ProtoRealFilePathResolver resolves real file paths from root file paths.
//
// Real file paths are actual file paths within a bucket,  while root file paths
// are those as referenced within an Image, i.e relatve to the roots.
type ProtoRealFilePathResolver interface {
	// GetRealFilePath returns the real file path for the root file path, if it exists.
	// If it does not exist, the empty string is returned, and linters should fall back
	// to the root file path for output.
	//
	// The input path is normalized and validated, and checked for empty.
	GetRealFilePath(rootFilePath string) (string, error)
}

// ProtoFileSet is a set of .proto files.
type ProtoFileSet interface {
	ProtoRootFilePathResolver
	ProtoRealFilePathResolver

	// Roots returns the proto_paths.
	//
	// There must be no overlap between the roots, ie foo/bar and foo are not allowed.
	// All Protobuf files must be unique relative to the roots, ie if foo and bar
	// are roots, then foo/baz.proto and bar/baz.proto are not allowed.
	//
	// Relative.
	// Normalized and validated.
	// Non-empty.
	// Returns a copy.
	Roots() []string
	// RootFilePaths returns the sorted list of file paths for the .proto files
	// that are relative to the roots.
	//
	// Relative.
	// Normalized and validated.
	// Non-empty.
	// Returns a copy.
	RootFilePaths() []string
	// RealFilePaths returns the list of real file paths for the .proto files.
	// These will be sorted in the same order as RootFilePaths(), that is
	// each index will match the index of the same file in RootFilePaths().
	//
	// Relative.
	// Normalized and validated.
	// Non-empty.
	// Returns a copy.
	RealFilePaths() []string
}

// Handler handles the build functionality.
type Handler interface {
	// Build builds an image for the bucket.
	//
	// If annotations or an error is returned, no image or resolver is returned.
	//
	// Annotations will be relative to the root of the bucket before returning, ie the
	// real file paths that already have the GetRealFilePath from the ProtoFileSet applied.
	Build(
		ctx context.Context,
		bucket storage.ReadBucket,
		protoFileSet ProtoFileSet,
		options BuildOptions,
	) (bufpb.Image, []*analysis.Annotation, error)
	// Files get the files for the bucket by returning a ProtoFileSet.
	Files(
		ctx context.Context,
		bucket storage.ReadBucket,
		options FilesOptions,
	) (ProtoFileSet, error)
}

// BuildOptions are options for Build.
type BuildOptions struct {
	// IncludeImports says to include imports.
	IncludeImports bool
	// IncludeSourceInfo says to include source info.
	IncludeSourceInfo bool
	// CopyToMemory says to copy the bucket to a memory bucket before building.
	//
	// If the bucket is already a memory bucket, this will result in a no-op.
	CopyToMemory bool
}

// FilesOptions are options for Files.
type FilesOptions struct {
	// Roots are the root directories within a bucket to search for Protobuf files.
	// If roots is empty, the default is ["."].
	//
	// There should be no overlap between the roots, ie foo/bar and foo are not allowed.
	// All Protobuf files must be unique relative to the roots, ie if foo and bar
	// are roots, then foo/baz.proto and bar/baz.proto are not allowed.
	//
	// All roots must be relative.
	// All roots will be normalized and validated.
	//
	Roots []string
	// Excludes are the directories within a bucket to exclude.
	//
	// There should be no overlap between the excludes, ie foo/bar and foo are not allowed.
	//
	// All excludes must reside within a root, but none willbe equal to a root.
	// All excludes must be relative.
	// All excludes will be normalized and validated.
	Excludes []string
	// SpecificRealFilePaths are the specific real file paths to get.
	//
	// All paths must be within a root.
	//
	// If SpecificRealFilePaths is empty, this gets all the files under Buf control.
	// If specificRealFilePaths is not empty, this uses these specific files, and Excludes is ignored.
	//
	// All paths must be relative.
	// All paths will be normalized and validated.
	SpecificRealFilePaths []string
	// SpecificRealFilePathsAllowNotExist allows file paths within SpecificRealFilePaths
	// to not exist without returning error.
	SpecificRealFilePathsAllowNotExist bool
}

// NewHandler returns a new Handler.
func NewHandler(logger *zap.Logger) Handler {
	return newHandler(logger)
}

// FixAnnotationFilenames attempts to make all filenames into real file paths.
//
// If the resolver is nil, this does nothing.
func FixAnnotationFilenames(resolver ProtoRealFilePathResolver, annotations []*analysis.Annotation) error {
	if resolver == nil {
		return nil
	}
	if len(annotations) == 0 {
		return nil
	}
	for _, annotation := range annotations {
		if err := fixAnnotationFilename(resolver, annotation); err != nil {
			return err
		}
	}
	return nil
}

func fixAnnotationFilename(resolver ProtoRealFilePathResolver, annotation *analysis.Annotation) error {
	if annotation.Filename == "" {
		return nil
	}
	filePath, err := resolver.GetRealFilePath(annotation.Filename)
	if err != nil {
		return err
	}
	if filePath != "" {
		annotation.Filename = filePath
	}
	return nil
}
