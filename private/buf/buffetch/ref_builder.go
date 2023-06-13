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
	"context"
	"fmt"
	"path/filepath"

	"github.com/bufbuild/buf/private/buf/buffetch/internal"
	"github.com/bufbuild/buf/private/pkg/app"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type refBuilder struct {
	tracer trace.Tracer
}

func newRefBuilder() *refBuilder {
	return &refBuilder{
		tracer: otel.GetTracerProvider().Tracer(tracerName),
	}
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

func (r *refBuilder) GetGitRef(
	ctx context.Context,
	format string,
	path string,
	options ...GetGitRefOption,
) (_ Ref, retErr error) {
	_, span := r.tracer.Start(ctx, "get_git_ref")
	defer span.End()
	defer func() {
		if retErr != nil {
			span.RecordError(retErr)
			span.SetStatus(codes.Error, retErr.Error())
		}
	}()
	if app.IsDevNull(path) {
		return nil, newDevNullNotAllowedError(path, format)
	}
	getGitRefOptions := newGetGitRefOptions()
	for _, option := range options {
		option(getGitRefOptions)
	}
	parsedRef, err := internal.NewGitRef(
		format,
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

func (r *refBuilder) GetModuleRef(
	ctx context.Context,
	format string,
	path string,
) (_ Ref, retErr error) {
	_, span := r.tracer.Start(ctx, "get_module_ref")
	defer span.End()
	defer func() {
		if retErr != nil {
			span.RecordError(retErr)
			span.SetStatus(codes.Error, retErr.Error())
		}
	}()
	if app.IsDevNull(path) {
		return nil, newDevNullNotAllowedError(path, format)
	}
	parsedRef, err := internal.NewModuleRef(format, path)
	if err != nil {
		return nil, err
	}
	return newModuleRef(parsedRef), nil
}

func (r *refBuilder) GetDirRef(
	ctx context.Context,
	format string,
	path string,
) (_ Ref, retErr error) {
	_, span := r.tracer.Start(ctx, "get_dir_ref")
	defer span.End()
	defer func() {
		if retErr != nil {
			span.RecordError(retErr)
			span.SetStatus(codes.Error, retErr.Error())
		}
	}()
	if app.IsDevNull(path) {
		return nil, newDevNullNotAllowedError(path, format)
	}
	parsedRef, err := internal.NewDirRef(format, path)
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

func (r *refBuilder) GetProtoFileRef(
	ctx context.Context,
	format string,
	path string,
	options ...GetProtoFileRefOption,
) (_ Ref, retErr error) {
	_, span := r.tracer.Start(ctx, "get_proto_file_ref")
	defer span.End()
	defer func() {
		if retErr != nil {
			span.RecordError(retErr)
			span.SetStatus(codes.Error, retErr.Error())
		}
	}()
	if app.IsDevNull(path) {
		return nil, newDevNullNotAllowedError(path, format)
	}
	getProtoFileRefOptions := newGetProtoFileRefOptions()
	for _, option := range options {
		option(getProtoFileRefOptions)
	}
	parsedRef, err := internal.NewProtoFileRef(format, path, getProtoFileRefOptions.includePackageFiles)
	if err != nil {
		return nil, err
	}
	return newProtoFileRef(parsedRef), nil
}

type getTarballRefOptions struct {
	compression     string
	stripComponents uint32
	subDirPath      string
}

func newGetTarballRefOptions() *getTarballRefOptions {
	return &getTarballRefOptions{}
}

func (r *refBuilder) GetTarballRef(
	ctx context.Context,
	format string,
	path string,
	options ...GetTarballRefOption,
) (_ Ref, retErr error) {
	_, span := r.tracer.Start(ctx, "get_tarball_ref")
	defer span.End()
	defer func() {
		if retErr != nil {
			span.RecordError(retErr)
			span.SetStatus(codes.Error, retErr.Error())
		}
	}()
	if app.IsDevNull(path) {
		return nil, newDevNullNotAllowedError(path, format)
	}
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
		compressionType, err = internal.ParseCompressionType(compression)
		if err != nil {
			return nil, err
		}
	}
	parsedRef, err := internal.NewArchiveRef(
		format,
		path,
		internal.ArchiveTypeTar,
		compressionType,
		getTarballRefOptions.stripComponents,
		getTarballRefOptions.subDirPath,
	)
	if err != nil {
		return nil, err
	}
	return newSourceRef(parsedRef), nil
}

type getZipArchiveRefOptions struct {
	stripComponents uint32
	subDirPath      string
}

func newGetZipArchiveRefOptions() *getZipArchiveRefOptions {
	return &getZipArchiveRefOptions{}
}

func (r *refBuilder) GetZipArchiveRef(
	ctx context.Context,
	format string,
	path string,
	options ...GetZipArchiveRefOption,
) (_ Ref, retErr error) {
	_, span := r.tracer.Start(ctx, "get_zip_archive_ref")
	defer span.End()
	defer func() {
		if retErr != nil {
			span.RecordError(retErr)
			span.SetStatus(codes.Error, retErr.Error())
		}
	}()
	if app.IsDevNull(path) {
		return nil, newDevNullNotAllowedError(path, format)
	}
	getZipArchiveRefOptions := newGetZipArchiveRefOptions()
	for _, option := range options {
		option(getZipArchiveRefOptions)
	}
	parsedRef, err := internal.NewArchiveRef(
		format,
		path,
		internal.ArchiveTypeZip,
		internal.CompressionTypeNone,
		getZipArchiveRefOptions.stripComponents,
		getZipArchiveRefOptions.subDirPath,
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

func (r *refBuilder) GetJSONImageRef(
	ctx context.Context,
	format string,
	path string,
	options ...GetImageRefOption,
) (_ Ref, retErr error) {
	_, span := r.tracer.Start(ctx, "get_json_image_ref")
	defer span.End()
	defer func() {
		if retErr != nil {
			span.RecordError(retErr)
			span.SetStatus(codes.Error, retErr.Error())
		}
	}()
	if app.IsDevNull(path) {
		return nil, newDevNullNotAllowedError(path, format)
	}
	return r.getImageRef(format, ImageEncodingJSON, path, options...)
}

func (r *refBuilder) GetBinaryImageRef(
	ctx context.Context,
	format string,
	path string,
	options ...GetImageRefOption,
) (_ Ref, retErr error) {
	_, span := r.tracer.Start(ctx, "get_binary_image_ref")
	defer span.End()
	defer func() {
		if retErr != nil {
			span.RecordError(retErr)
			span.SetStatus(codes.Error, retErr.Error())
		}
	}()
	return r.getImageRef(format, ImageEncodingBin, path, options...)
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
	case ".gz":
		compressionType = internal.CompressionTypeGzip
	}
	if compression := getImageRefOptions.compression; compression != "" {
		var err error
		compressionType, err = internal.ParseCompressionType(compression)
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

func newDevNullNotAllowedError(path string, format string) error {
	return fmt.Errorf("%s is not allowed for %s", path, format)
}
