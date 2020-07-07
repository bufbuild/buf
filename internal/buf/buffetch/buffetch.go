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

package buffetch

import (
	"context"
	"io"
	"net/http"

	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/fetch"
	"github.com/bufbuild/buf/internal/pkg/git"
	"github.com/bufbuild/buf/internal/pkg/httpauth"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"go.uber.org/zap"
)

const (
	// ImageEncodingBin is the binary image encoding.
	ImageEncodingBin ImageEncoding = iota + 1
	// ImageEncodingJSON is the JSON image encoding.
	ImageEncodingJSON
)

var (
	// ImageFormatsString is the string representation of all image formats.
	//
	// This does not include deprecated formats.
	ImageFormatsString = formatsToString(imageFormatsNotDeprecated)
	// SourceFormatsString is the string representation of all source formats.
	//
	// This does not include deprecated formats.
	SourceFormatsString = formatsToString(sourceFormatsNotDeprecated)
	// AllFormatsString is the string representation of all formats.
	//
	// This does not include deprecated formats.
	AllFormatsString = formatsToString(allFormatsNotDeprecated)
)

// ImageEncoding is the encoding of the image.
type ImageEncoding int

// PathResolver resolves external paths to paths.
type PathResolver interface {
	// PathForExternalPath takes a path external to the asset and converts it to
	// a path that is relative to the asset.
	//
	// The returned path will be normalized and validated.
	//
	// Example:
	//   Directory: /foo/bar
	//   ExternalPath: /foo/bar/baz/bat.proto
	//   Path: baz/bat.proto
	//
	// Example:
	//   Directory: .
	//   ExternalPath: baz/bat.proto
	//   Path: baz/bat.proto
	PathForExternalPath(externalPath string) (string, error)
}

// Ref is an image file or source bucket reference.
type Ref interface {
	PathResolver

	fetchRef() fetch.Ref
}

// ImageRef is an image file reference.
type ImageRef interface {
	Ref
	ImageEncoding() ImageEncoding
	IsNull() bool
	fetchFileRef() fetch.FileRef
}

// SourceRef is a source bucket reference.
type SourceRef interface {
	Ref
	fetchBucketRef() fetch.BucketRef
}

// ImageRefParser is an image ref parser for Buf.
type ImageRefParser interface {
	// GetImageRef gets the reference for the image file.
	GetImageRef(ctx context.Context, value string) (ImageRef, error)
}

// SourceRefParser is a source ref parser for Buf.
type SourceRefParser interface {
	// GetSourceRef gets the reference for the source file.
	GetSourceRef(ctx context.Context, value string) (SourceRef, error)
}

// RefParser is a ref parser for Buf.
type RefParser interface {
	ImageRefParser
	SourceRefParser

	// GetRef gets the reference for the image file or source bucket.
	GetRef(ctx context.Context, value string) (Ref, error)
}

// NewRefParser returns a new RefParser.
func NewRefParser(logger *zap.Logger) RefParser {
	return newRefParser(logger)
}

// NewImageRefParser returns a new RefParser for images only.
//
// This defaults to binary.
func NewImageRefParser(logger *zap.Logger) ImageRefParser {
	return newImageRefParser(logger)
}

// Reader is a reader for Buf.
type Reader interface {
	// GetImageFile gets the image file.
	//
	// The returned file will be uncompressed.
	GetImageFile(
		ctx context.Context,
		container app.EnvStdinContainer,
		imageRef ImageRef,
	) (io.ReadCloser, error)
	// GetSource gets the source bucket.
	//
	// The returned bucket will only have .proto and configuration files.
	GetSourceBucket(
		ctx context.Context,
		container app.EnvStdinContainer,
		sourceRef SourceRef,
	) (storage.ReadBucketCloser, error)
}

// NewReader returns a new Reader.
func NewReader(
	logger *zap.Logger,
	httpClient *http.Client,
	httpAuthenticator httpauth.Authenticator,
	gitCloner git.Cloner,
) Reader {
	return newReader(
		logger,
		httpClient,
		httpAuthenticator,
		gitCloner,
	)
}

// Writer is a writer for Buf.
type Writer interface {
	// PutImageFile puts the image file.
	PutImageFile(
		ctx context.Context,
		container app.EnvStdoutContainer,
		imageRef ImageRef,
	) (io.WriteCloser, error)
}

// NewWriter returns a new Writer.
func NewWriter(
	logger *zap.Logger,
) Writer {
	return newWriter(
		logger,
	)
}
