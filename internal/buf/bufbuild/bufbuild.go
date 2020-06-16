// Copyright 2020 Buf Technologies, Inc.
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

package bufbuild

import (
	"context"

	"github.com/bufbuild/buf/internal/buf/bufanalysis"
	"github.com/bufbuild/buf/internal/buf/bufimage"
	"github.com/bufbuild/buf/internal/buf/bufpath"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"go.uber.org/zap"
)

// Builder builds Protobuf files into Images.
type Builder interface {
	// Build runs compilation.
	//
	// The FileRefs are assumed to have been created by a FileRefProvider, that is
	// they are unique relative to the roots.
	//
	// If an error is returned, it is a system error.
	// Only one of Image and FileAnnotations will be returned.
	//
	// FileAnnotations will use external file paths.
	Build(
		ctx context.Context,
		readBucket storage.ReadBucket,
		externalPathResolver bufpath.ExternalPathResolver,
		fileRefs []bufimage.FileRef,
		options ...BuildOption,
	) (bufimage.Image, []bufanalysis.FileAnnotation, error)
}

// NewBuilder returns a new Builder.
func NewBuilder(logger *zap.Logger, options ...BuilderOption) Builder {
	return newBuilder(logger, options...)
}

// FileRefProvider provides bufimage.FileRefs.
type FileRefProvider interface {
	// GetAllFileRefs gets all the FileRefs for the bucket.
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
	//
	// FileRefs will be unique by root relative file path.
	// FileRefs will be sorted by root relative file path.
	GetAllFileRefs(
		ctx context.Context,
		readBucket storage.ReadBucket,
		externalPathResolver bufpath.ExternalPathResolver,
		roots []string,
		excludes []string,
	) ([]bufimage.FileRef, error)
	// GetFileRefsForExternalFilePaths gets the FileRefs for the specific files within the bucket.
	//
	// The externalFilePaths will be resolved to file paths relative to the bucket via the PathResolver.
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
	// FileRefs will be unique by root relative file path.
	// FileRefs will be sorted by root relative file path.
	GetFileRefsForExternalFilePaths(
		ctx context.Context,
		readBucket storage.ReadBucket,
		pathResolver bufpath.PathResolver,
		roots []string,
		externalFilePaths []string,
		options ...GetFileRefsForExternalFilePathsOption,
	) ([]bufimage.FileRef, error)
}

// NewFileRefProvider returns a new FileRefProvider.
func NewFileRefProvider(logger *zap.Logger) FileRefProvider {
	return newFileRefProvider(logger)
}

// BuilderOption is an option for a new Builder.
type BuilderOption func(*builder)

// BuildOption is an option for Build.
type BuildOption func(*buildOptions)

// WithExcludeSourceCodeInfo returns a BuildOption that excludes sourceCodeInfo.
func WithExcludeSourceCodeInfo() BuildOption {
	return func(buildOptions *buildOptions) {
		buildOptions.excludeSourceCodeInfo = true
	}
}

// GetFileRefsForExternalFilePathsOption is an option for GetFileRefsForExternalFilePaths.
type GetFileRefsForExternalFilePathsOption func(*getFileRefsForExternalFilePathsOptions)

// WithAllowNotExist says that a given external file path may not exist without error.
func WithAllowNotExist() GetFileRefsForExternalFilePathsOption {
	return func(getFileRefsForExternalFilePathsOptions *getFileRefsForExternalFilePathsOptions) {
		getFileRefsForExternalFilePathsOptions.allowNotExist = true
	}
}

// ExternalConfig is an external config.
type ExternalConfig struct {
	Roots    []string `json:"roots,omitempty" yaml:"roots,omitempty"`
	Excludes []string `json:"excludes,omitempty" yaml:"excludes,omitempty"`
}
