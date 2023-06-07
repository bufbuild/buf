// Copyright 2020-2023 Buf Technologies, Inc.
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

package buffetch

import (
	"path/filepath"

	"github.com/bufbuild/buf/private/buf/buffetch/internal"
)

type refBuilder struct{}

func newRefBuilder() *refBuilder {
	return &refBuilder{}
}

type getGitRefOptions struct {
	branch            string
	tag               string
	ref               string
	depth             uint32
	recurseSubmodules bool
	subDirPath        string
}

func newGetGitRefOptions() *getGitRefOptions {
	return &getGitRefOptions{}
}

func (r *refBuilder) GetGitRef(path string, options ...GetGitRefOption) (Ref, error) {
	getGitRefOptions := newGetGitRefOptions()
	for _, option := range options {
		option(getGitRefOptions)
	}
	parsedRef, err := internal.NewGitRef(
		formatGit,
		path,
		getGitRefOptions.branch,
		getGitRefOptions.tag,
		getGitRefOptions.ref,
		getGitRefOptions.depth,
		getGitRefOptions.recurseSubmodules,
		getGitRefOptions.subDirPath,
	)
	if err != nil {
		return nil, err
	}
	return newSourceRef(parsedRef), nil
}

func (r *refBuilder) GetModuleRef(path string) (Ref, error) {
	parsedRef, err := internal.NewModuleRef(formatMod, path)
	if err != nil {
		return nil, err
	}
	return newModuleRef(parsedRef), nil
}

func (r *refBuilder) GetDirRef(path string) (Ref, error) {
	parsedRef, err := internal.NewDirRef(formatDir, path)
	if err != nil {
		return nil, err
	}
	return newSourceRef(parsedRef), nil
}

type getProtoFileRefOptions struct {
	includePackageFiles bool
}

func newGetProtoFileRefOptions() *getProtoFileRefOptions {
	return &getProtoFileRefOptions{}
}

func (r *refBuilder) GetProtoFileRef(path string, options ...GetProtoFileRefOption) Ref {
	getProtoFileRefOptions := newGetProtoFileRefOptions()
	for _, option := range options {
		option(getProtoFileRefOptions)
	}
	return newProtoFileRef(internal.NewProtoFileRef(formatProtoFile, path, getProtoFileRefOptions.includePackageFiles))
}

type getTarballRefOptions struct {
	compression     string
	stripComponents uint32
	subDir          string
}

func newGetTarballRefOptions() *getTarballRefOptions {
	return &getTarballRefOptions{}
}

func (r *refBuilder) GetTarballRef(path string, options ...GetTarballRefOption) (Ref, error) {
	getTarballRefOptions := newGetTarballRefOptions()
	for _, option := range options {
		option(getTarballRefOptions)
	}
	compressionType := internal.CompressionTypeNone
	switch ext := filepath.Ext(path); ext {
	case ".zst":
		compressionType = internal.CompressionTypeZstd
	case ".tgz", ".gz":
		compressionType = internal.CompressionTypeGzip
	}
	if compression := getTarballRefOptions.compression; compression != "" {
		var err error
		compressionType, err = internal.NewCompressionType(compression)
		if err != nil {
			return nil, err
		}
	}
	parsedRef, err := internal.NewArchiveRef(
		formatTar,
		path,
		internal.ArchiveTypeTar,
		compressionType,
		getTarballRefOptions.stripComponents,
		getTarballRefOptions.subDir,
	)
	if err != nil {
		return nil, err
	}
	return newSourceRef(parsedRef), nil
}

type getZipArchiveRefOptions struct {
	stripComponents uint32
	subDir          string
}

func newGetZipArchiveRefOptions() *getZipArchiveRefOptions {
	return &getZipArchiveRefOptions{}
}

func (r *refBuilder) GetZipArchiveRef(path string, options ...GetZipArchiveRefOption) (Ref, error) {
	getZipArchiveRefOptions := newGetZipArchiveRefOptions()
	for _, option := range options {
		option(getZipArchiveRefOptions)
	}
	parsedRef, err := internal.NewArchiveRef(
		formatZip,
		path,
		internal.ArchiveTypeZip,
		internal.CompressionTypeNone,
		getZipArchiveRefOptions.stripComponents,
		getZipArchiveRefOptions.subDir,
	)
	if err != nil {
		return nil, err
	}
	return newSourceRef(parsedRef), nil
}

type getImageRefOptions struct {
	compression string
}

func newGetImageRefOptions() *getImageRefOptions {
	return &getImageRefOptions{}
}

func (r *refBuilder) GetJSONImageRef(path string, options ...GetImageRefOption) (Ref, error) {
	return r.getImageRef(formatJSON, ImageEncodingJSON, path, options...)
}

func (r *refBuilder) GetBinaryImageRef(path string, options ...GetImageRefOption) (Ref, error) {
	return r.getImageRef(formatBin, ImageEncodingBin, path, options...)
}

func (r *refBuilder) getImageRef(format string, encoding ImageEncoding, path string, options ...GetImageRefOption) (Ref, error) {
	getImageRefOptions := newGetImageRefOptions()
	for _, option := range options {
		option(getImageRefOptions)
	}
	compressionType := internal.CompressionTypeNone
	switch ext := filepath.Ext(path); ext {
	case ".zst":
		compressionType = internal.CompressionTypeZstd
	case ".tgz", ".gz":
		compressionType = internal.CompressionTypeGzip
	}
	if compression := getImageRefOptions.compression; compression != "" {
		var err error
		compressionType, err = internal.NewCompressionType(compression)
		if err != nil {
			return nil, err
		}
	}
	parsedRef, err := internal.NewSingleRef(format, path, compressionType)
	if err != nil {
		return nil, err
	}
	return newImageRef(parsedRef, encoding), nil
}
