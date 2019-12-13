// Package bufbuild drives the building of Protobuf files.
//
// This package has two directory concepts used as input: roots and excludes.
//
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
// Excludes are the directories within a bucket to exclude.
//
// There should be no overlap between the excludes, ie foo/bar and foo are not allowed.
//
// All excludes must reside within a root, but none willbe equal to a root.
// All excludes must be relative.
// All excludes will be normalized and validated.
package bufbuild

import (
	"context"

	"github.com/bufbuild/buf/internal/buf/buferrs"
	"github.com/bufbuild/buf/internal/buf/bufpb"
	"github.com/bufbuild/buf/internal/pkg/analysis"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"go.uber.org/zap"
)

// ErrFilePathUnknown is the error returned by GetRealFilePath and GetRootFilePath if
// the input path does not have a corresponding output path.
var ErrFilePathUnknown = buferrs.NewSystemError("real file path unknown")

// ProtoFilePathResolver transforms input file paths to output file paths.
type ProtoFilePathResolver interface {
	// GetFilePath returns the file path for the input file path, if it exists.
	// If it does not exist, linters should fall back to the input file path for output.
	// If it does not exist, ErrFilePathUnknown is returned.
	//
	// The input path is normalized and validated, and checked for empty.
	GetFilePath(inputFilePath string) (string, error)
}

// Handler handles the main build functionality.
type Handler interface {
	// BuildImage builds an image for the bucket.
	//
	// If specificRealFilePaths is empty, this builds all the files under Buf control.
	// If specificRealFilePaths is not empty, this uses these specific files.
	// specificRealFilePaths may include files that do not exist; this will be checked
	// prior to running the build per the documention for Provider.
	//
	// If annotations or an error is returned, no image or resolver is returned.
	// Annotations will be relative to the root of the bucket before returning, ie the
	// real file paths that already have the resolver applied.
	BuildImage(
		ctx context.Context,
		bucket storage.ReadBucket,
		roots []string,
		excludes []string,
		specificRealFilePaths []string,
		specificRealFilePathsAllowNotExist bool,
		includeImports bool,
		includeSourceInfo bool,
	) (bufpb.Image, ProtoFilePathResolver, []*analysis.Annotation, error)

	// ListFiles lists the files for the bucket and config.
	//
	// File paths will be relative to the root of the bucket before returning, ie the
	// real file paths that already have the resolver applied.
	// File paths are sorted.
	ListFiles(
		ctx context.Context,
		bucket storage.ReadBucket,
		roots []string,
		excludes []string,
	) ([]string, error)
}

// NewHandler returns a new Handler.
func NewHandler(
	logger *zap.Logger,
	buildProvider Provider,
	buildRunner Runner,
) Handler {
	return newHandler(
		logger,
		buildProvider,
		buildRunner,
	)
}

// ProtoFileRootPathResolver resolves root file paths from real file paths.
type ProtoFileRootPathResolver interface {
	// GetRootFilePath returns the root file path for the real file path, if it exists.
	// If it does not exist, ErrFilePathUnknown is returned.
	//
	// The input path is normalized and validated, and checked for empty.
	GetRootFilePath(realFilePath string) (string, error)
}

// ProtoFileRealPathResolver resolves real file paths from root file paths.
type ProtoFileRealPathResolver interface {
	// GetRealFilePath returns the real file path for the root file path, if it exists.
	// If it does not exist, linters should fall back to the root file path for output.
	// If it does not exist, ErrFilePathUnknown is returned.
	//
	// The input path is normalized and validated, and checked for empty.
	GetRealFilePath(rootFilePath string) (string, error)
}

// ProtoFileSet is a set of .proto files.
type ProtoFileSet interface {
	// GetFilePath returns the value of GetRealFilePath.
	ProtoFilePathResolver
	ProtoFileRootPathResolver
	ProtoFileRealPathResolver

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

// Provider is a provider.
type Provider interface {
	// GetProtoFileSetForBucket gets the set for the bucket and config.
	//
	GetProtoFileSetForBucket(
		ctx context.Context,
		bucket storage.ReadBucket,
		roots []string,
		excludes []string,
	) (ProtoFileSet, error)
	// GetSetForRealFilePaths gets the set for the real file paths and config.
	//
	// File paths will be validated to make sure they are within a root,
	// unique relative to roots, and that they exist. If allowNotExist
	// is true, files that do not exist will be filtered. This is useful
	// for i.e. --limit-to-input-files.
	GetProtoFileSetForRealFilePaths(
		ctx context.Context,
		bucket storage.ReadBucket,
		roots []string,
		realFilePaths []string,
		allowNotExist bool,
	) (ProtoFileSet, error)
}

// NewProvider returns a new Provider.
func NewProvider(logger *zap.Logger) Provider {
	return newProvider(logger)
}

// Runner runs compilations.
type Runner interface {
	// Run runs compilation.
	//
	// If an error is returned, it is a system error.
	// Only one of Image and annotations will be returned.
	//
	// Annotations will be sorted, but Filenames will not have the roots as a prefix, instead
	// they will be relative to the roots. This should be fixed for linter outputs if image
	// mode is not used.
	Run(
		ctx context.Context,
		bucket storage.ReadBucket,
		protoFileSet ProtoFileSet,
		includeImports bool,
		includeSourceInfo bool,
	) (bufpb.Image, []*analysis.Annotation, error)
}

// NewRunner returns a new Runner.
func NewRunner(logger *zap.Logger) Runner {
	return newRunner(logger)
}

// FixAnnotationFilenames attempts to make all filenames into real file paths.
//
// If the resolver is nil, this does nothing.
func FixAnnotationFilenames(resolver ProtoFilePathResolver, annotations []*analysis.Annotation) error {
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

func fixAnnotationFilename(resolver ProtoFilePathResolver, annotation *analysis.Annotation) error {
	if annotation.Filename == "" {
		return nil
	}
	filePath, err := resolver.GetFilePath(annotation.Filename)
	if err != nil {
		if err == ErrFilePathUnknown {
			return nil
		}
		return err
	}
	annotation.Filename = filePath
	return nil
}
