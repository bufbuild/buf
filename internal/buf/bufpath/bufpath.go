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

package bufpath

// NopPathResolver is a PathResolver that is a no-op.
//
// This PathResolver will still normalize and validate fields per the requirements.
var NopPathResolver PathResolver = newNopPathResolver()

// RelPathResolver resolves external paths to relative paths.
type RelPathResolver interface {
	// ExternalPathToRelPath takes a path external to the asset and converts it to
	// a path that is relative to the asset.
	//
	// The returned path will be normalized and validated.
	//
	// Example:
	//   Directory: /foo/bar
	//   ExternalPath: /foo/bar/baz/bat.proto
	//   RelPath: baz/bat.proto
	ExternalPathToRelPath(externalPath string) (string, error)
}

// ExternalPathResolver resolves relative paths to external paths.
type ExternalPathResolver interface {
	// RelPathToExternalPath takes a path relative to the asset and converts it
	// to a path that is external to the asset.
	//
	// This path is not necessarily a file path, and should only be used to
	// uniquely identify this file as compared to other assets, and for display
	// to users.
	//
	// The input path will be normalized and validated.
	// The output path will be unnormalized, if it is a file path.
	//
	// Example:
	//   Directory: /foo/bar
	//   RelPath: baz/bat.proto
	//   ExternalPath: /foo/bar/baz/bat.proto
	//
	// Example:
	//   Directory: .
	//   RelPath: baz/bat.proto
	//   ExternalPath: baz/bat.proto
	RelPathToExternalPath(relPath string) (string, error)
}

// PathResolver resolves both external and relative paths.
type PathResolver interface {
	RelPathResolver
	ExternalPathResolver
}

// NewDirPathResolver returns a PathResolver for a directory.
//
// The dirPath will be normalized.
func NewDirPathResolver(dirPath string) PathResolver {
	return newDirPathResolver(dirPath)
}
