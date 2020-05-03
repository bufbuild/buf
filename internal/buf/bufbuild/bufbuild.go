// Copyright 2020 Buf Technologies Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package bufbuild drives the building of Protobuf files.
package bufbuild

import (
	"context"
	"runtime"

	"github.com/bufbuild/buf/internal/buf/ext/extfile"
	filev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/file/v1beta1"
	imagev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/image/v1beta1"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"go.uber.org/zap"
)

const (
	// DefaultCopyToMemoryFileThreshold is the default copy to memory threshold.
	DefaultCopyToMemoryFileThreshold = 16
)

var (
	// DefaultParallelism is the default parallelism.
	DefaultParallelism = runtime.NumCPU()
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

	// Size returns the size of the set.
	//
	// This is equal to len(RootFilePaths()) and len(RealFilePaths()).
	Size() int
}

// Handler handles the build functionality.
type Handler interface {
	// Build builds an image for the bucket.
	//
	// If FileAnnotations or an error is returned, no image or resolver is returned.
	//
	// FileAnnotations will be relative to the root of the bucket before returning, ie the
	// real file paths that already have the GetRealFilePath from the ProtoFileSet applied.
	Build(
		ctx context.Context,
		readBucket storage.ReadBucket,
		protoFileSet ProtoFileSet,
		options BuildOptions,
	) (*imagev1beta1.Image, []*filev1beta1.FileAnnotation, error)
	// GetProtoFileSet get the files for the entire bucket.
	GetProtoFileSet(
		ctx context.Context,
		readBucket storage.ReadBucket,
		options GetProtoFileSetOptions,
	) (ProtoFileSet, error)
	// GetProtoFileSetForFiles gets the specific files within the bucket.
	GetProtoFileSetForFiles(
		ctx context.Context,
		readBucket storage.ReadBucket,
		realFilePaths []string,
		options GetProtoFileSetForFilesOptions,
	) (ProtoFileSet, error)
}

// BuildOptions are options for Build.
type BuildOptions struct {
	// IncludeImports says to include imports.
	IncludeImports bool
	// IncludeSourceInfo says to include source info.
	IncludeSourceInfo bool
}

// GetProtoFileSetOptions are options for Files.
type GetProtoFileSetOptions struct {
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
}

// GetProtoFileSetForFilesOptions are options for GetProtoFileSetForFiles.
type GetProtoFileSetForFilesOptions struct {
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
	// AllowNotExist allows file paths within realFilePaths to not exist
	// without returning error.
	AllowNotExist bool
}

// NewHandler returns a new Handler.
func NewHandler(logger *zap.Logger, options ...HandlerOption) Handler {
	return newHandler(logger, options...)
}

// HandlerOption is an option for a new Handler.
type HandlerOption func(*handler)

// HandlerWithParallelism says how many threads to compile with in parallel.
//
// Serial compilation is performed if this value is <=1.
// The default is to use DefaultParallelism.
func HandlerWithParallelism(parallelism int) HandlerOption {
	return func(handler *handler) {
		handler.parallelism = parallelism
	}
}

// HandlerWithCopyToMemoryFileThreshold says to copy files to memory before compilation if
// at least this many files are present.
//
// If this value is <=0, files are never copied.
// The default is to use DefaultCopyToMemoryFileThreshold.
func HandlerWithCopyToMemoryFileThreshold(copyToMemoryFileThreshold int) HandlerOption {
	return func(handler *handler) {
		handler.copyToMemoryFileThreshold = copyToMemoryFileThreshold
	}
}

// FixFileAnnotationPaths attempts to make all paths into real file paths.
//
// If the resolver is nil, this does nothing.
func FixFileAnnotationPaths(resolver ProtoRealFilePathResolver, fileAnnotations ...*filev1beta1.FileAnnotation) error {
	if resolver == nil {
		return nil
	}
	return extfile.ResolveFileAnnotationPaths(resolver.GetRealFilePath, fileAnnotations...)
}
