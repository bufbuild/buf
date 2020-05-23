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
	ImageFormatsString = formatsToString(imageFormats)
	// SourceFormatsString is the string representation of all source formats.
	SourceFormatsString = formatsToString(sourceFormats)
	// AllFormatsString is the string representation of all formats.
	AllFormatsString = formatsToString(allFormats)
)

// ImageEncoding is the encoding of the image.
type ImageEncoding int

// PathResolver resolves paths within assets.
type PathResolver interface {
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

// RefParser is an  ref parser for Buf.
type RefParser interface {
	// GetRef gets the reference for the image file or source bucket.
	GetRef(
		ctx context.Context,
		value string,
	) (Ref, error)
	// GetImageRef gets the reference for the image file.
	GetImageRef(
		ctx context.Context,
		value string,
	) (ImageRef, error)
	// GetSourceRef gets the reference for the source bucket.
	GetSourceRef(
		ctx context.Context,
		value string,
	) (SourceRef, error)
}

// NewRefParser returns a new RefParser.
func NewRefParser(
	logger *zap.Logger,
) RefParser {
	return newRefParser(
		logger,
	)
}

// Reader is a reader for Buf.
type Reader interface {
	// GetImage gets the image file.
	//
	// The returned file will be uncompressed.
	GetImage(
		ctx context.Context,
		container app.EnvStdinContainer,
		imageRef ImageRef,
	) (io.ReadCloser, error)
	// GetSource gets the source bucket.
	//
	// The returned bucket will only have .proto and configuration files.
	GetSource(
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
	// PutImage puts the image file.
	PutImage(
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
