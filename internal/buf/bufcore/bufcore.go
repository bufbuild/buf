// Copyright 2020-2021 Buf Technologies, Inc.
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

// Package bufcore contains core types.
package bufcore

import (
	"sort"

	"github.com/bufbuild/buf/internal/pkg/storage"
)

// FileInfo contains protobuf file info.
type FileInfo interface {
	// Path is the path of the file relative to the root it is contained within.
	// This will be normalized, validated and never empty,
	// This will be unique within a given Image.
	Path() string
	// ExternalPath returns the path that identifies this file externally.
	//
	// This will be unnormalized.
	// Never empty. Falls back to Path if there is not an external path.
	//
	// Example:
	//	 Assume we had the input path /foo/bar which is a local directory.

	//   Path: one/one.proto
	//   RootDirPath: proto
	//   ExternalPath: /foo/bar/proto/one/one.proto
	ExternalPath() string
	// IsImport returns true if this file is an import.
	IsImport() bool

	// WithIsImport returns this FileInfo with the given IsImport value.
	WithIsImport(isImport bool) FileInfo

	isFileInfo()
}

// NewFileInfo returns a new FileInfo.
//
// If externalPath is empty, path is used.
func NewFileInfo(path string, externalPath string, isImport bool) (FileInfo, error) {
	return newFileInfo(path, externalPath, isImport)
}

// NewFileInfoForObjectInfo returns a new FileInfo for the storage.ObjectInfo.
//
// The same rules apply to ObjectInfos for paths as FileInfos so we do not need to validate.
func NewFileInfoForObjectInfo(objectInfo storage.ObjectInfo, isImport bool) FileInfo {
	return newFileInfoForObjectInfo(objectInfo, isImport)
}

// SortFileInfos sorts the FileInfos.
func SortFileInfos(fileInfos []FileInfo) {
	sort.Slice(
		fileInfos,
		func(i int, j int) bool {
			return fileInfos[i].Path() < fileInfos[j].Path()
		},
	)
}
