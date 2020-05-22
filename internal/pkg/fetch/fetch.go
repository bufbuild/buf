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

package fetch

import (
	"context"
	"io"
	"net/http"

	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/app/apphttp"
	"github.com/bufbuild/buf/internal/pkg/git"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"go.uber.org/zap"
)

const (
	// FileSchemeHTTP is the http file scheme.
	FileSchemeHTTP FileScheme = iota + 1
	// FileSchemeHTTPS is the https file scheme.
	FileSchemeHTTPS
	// FileSchemeLocal is the local file scheme.
	FileSchemeLocal
	// FileSchemeStdio is the stdio file scheme.
	FileSchemeStdio
	// FileSchemeNull is the null file scheme.
	FileSchemeNull

	// GitSchemeHTTP is the http git scheme.
	GitSchemeHTTP GitScheme = iota + 1
	// GitSchemeHTTPS is the https git scheme.
	GitSchemeHTTPS
	// GitSchemeLocal is the local git scheme.
	GitSchemeLocal
	// GitSchemeSSH is the ssh git scheme.
	GitSchemeSSH

	// ArchiveTypeTar is a tar archive.
	ArchiveTypeTar ArchiveType = iota + 1

	// CompressionTypeNone is no compression.
	CompressionTypeNone CompressionType = iota + 1
	// CompressionTypeGzip is gzip compression.
	CompressionTypeGzip
)

// FileScheme is a file scheme.
type FileScheme int

// GitScheme is a git scheme.
type GitScheme int

// ArchiveType is a archive type.
type ArchiveType int

// CompressionType is a compression type.
type CompressionType int

// Ref is a reference.
type Ref interface {
	Format() string
	// Path is the path to.
	//
	// This will be the non-empty path minus the scheme for http and https files.
	// This will be the non-empty normalized file path for local files.
	// This will be empty for stdio and null files.
	// This will be the non-empty normalized directory path for directories.
	// This will be the non-empty path minus the scheme for http, https, and ssh git repositories.
	// This will be the non-empty normalized directory path for local git repositories.
	Path() string
	ref()
}

// FileRef is a file reference.
type FileRef interface {
	Ref
	FileScheme() FileScheme
	CompressionType() CompressionType
	fileRef()
}

// BucketRef is a bucket reference.
type BucketRef interface {
	Ref
	bucketRef()
}

// SingleRef is a non-archive file reference.
type SingleRef interface {
	FileRef
	singleRef()
}

// ArchiveRef is an archive reference.
//
// An ArchiveRef is a special type of reference that can be either a FileRef or a BucketRef.
type ArchiveRef interface {
	FileRef
	BucketRef
	ArchiveType() ArchiveType
	StripComponents() uint32
	archiveRef()
}

// DirRef is a local directory reference.
type DirRef interface {
	BucketRef
	dirRef()
}

// GitRef is a git reference.
type GitRef interface {
	BucketRef
	GitScheme() GitScheme
	GitRefName() git.RefName
	RecurseSubmodules() bool
	gitRef()
}

// RefParser provies references.
type RefParser interface {
	// GetRef gets the Ref for the value.
	//
	// The returned Ref will be either a SingleRef, ArchiveRef, DirRef, or GitRef.
	//
	// The options should be used to validate that you are getting one of the correct formats.
	GetRef(ctx context.Context, value string, options ...GetRefOption) (Ref, error)
}

// NewRefParser returns a new RefParser.
func NewRefParser(logger *zap.Logger, options ...RefParserOption) RefParser {
	return newRefParser(logger, options...)
}

// Reader is a reader.
type Reader interface {
	// GetFile gets the file.
	// SingleRefs and ArchiveRefs will result in decompressed files unless KeepFileCompression is set.
	GetFile(
		ctx context.Context,
		container app.EnvStdinContainer,
		fileRef FileRef,
		options ...GetFileOption,
	) (io.ReadCloser, error)
	// GetBucket gets the bucket .
	GetBucket(
		ctx context.Context,
		container app.EnvStdinContainer,
		bucketRef BucketRef,
		options ...GetBucketOption,
	) (storage.ReadBucketCloser, error)
}

// NewReader returns a new Reader.
func NewReader(
	logger *zap.Logger,
	httpClient *http.Client,
	httpAuthenticator apphttp.Authenticator,
	gitCloner git.Cloner,
	options ...ReaderOption,
) Reader {
	return newReader(
		logger,
		httpClient,
		httpAuthenticator,
		gitCloner,
		options...,
	)
}

// Writer is a writer.
type Writer interface {
	// PutFile puts the file .
	PutFile(
		ctx context.Context,
		container app.EnvStdoutContainer,
		fileRef FileRef,
		options ...PutFileOption,
	) (io.WriteCloser, error)
}

// NewWriter returns a new Writer.
func NewWriter(
	logger *zap.Logger,
	options ...WriterOption,
) Writer {
	return newWriter(
		logger,
		options...,
	)
}

// RefParserOption is an RefParser option.
type RefParserOption func(*refParser)

// WithFormatParser attaches the given format parser.
//
// If format is not manually specified, the RefParser will use this format parser
// with the raw path, that is not normalized.
func WithFormatParser(formatParser func(string) (string, error)) RefParserOption {
	return func(refParser *refParser) {
		refParser.formatParser = formatParser
	}
}

// WithSingleFormat attaches the given format as a single format.
//
// It is up to the user to not incorrectly attached a format twice.
func WithSingleFormat(format string, options ...SingleFormatOption) RefParserOption {
	return func(refParser *refParser) {
		format = normalizeFormat(format)
		if format == "" {
			return
		}
		singleFormatInfo := newSingleFormatInfo()
		for _, option := range options {
			option(singleFormatInfo)
		}
		refParser.singleFormatToInfo[format] = singleFormatInfo
	}
}

// WithArchiveFormat attaches the given format as an archive format.
//
// It is up to the user to not incorrectly attached a format twice.
func WithArchiveFormat(format string, archiveType ArchiveType, options ...ArchiveFormatOption) RefParserOption {
	return func(refParser *refParser) {
		format = normalizeFormat(format)
		if format == "" {
			return
		}
		archiveFormatInfo := newArchiveFormatInfo(archiveType)
		for _, option := range options {
			option(archiveFormatInfo)
		}
		refParser.archiveFormatToInfo[format] = archiveFormatInfo
	}
}

// WithDirFormat attaches the given format as a dir format.
//
// It is up to the user to not incorrectly attached a format twice.
func WithDirFormat(format string, options ...DirFormatOption) RefParserOption {
	return func(refParser *refParser) {
		format = normalizeFormat(format)
		if format == "" {
			return
		}
		dirFormatInfo := newDirFormatInfo()
		for _, option := range options {
			option(dirFormatInfo)
		}
		refParser.dirFormatToInfo[format] = dirFormatInfo
	}
}

// WithGitFormat attaches the given format as a git format.
//
// It is up to the user to not incorrectly attached a format twice.
func WithGitFormat(format string, options ...GitFormatOption) RefParserOption {
	return func(refParser *refParser) {
		format = normalizeFormat(format)
		if format == "" {
			return
		}
		gitFormatInfo := newGitFormatInfo()
		for _, option := range options {
			option(gitFormatInfo)
		}
		refParser.gitFormatToInfo[format] = gitFormatInfo
	}
}

// SingleFormatOption is a single format option.
type SingleFormatOption func(*singleFormatInfo)

// WithSingleDefaultCompressionType sets the default compression type.
func WithSingleDefaultCompressionType(defaultCompressionType CompressionType) SingleFormatOption {
	return func(singleFormatInfo *singleFormatInfo) {
		singleFormatInfo.defaultCompressionType = defaultCompressionType
	}
}

// ArchiveFormatOption is a archive format option.
type ArchiveFormatOption func(*archiveFormatInfo)

// WithArchiveDefaultCompressionType sets the default compression type.
func WithArchiveDefaultCompressionType(defaultCompressionType CompressionType) ArchiveFormatOption {
	return func(archiveFormatInfo *archiveFormatInfo) {
		archiveFormatInfo.defaultCompressionType = defaultCompressionType
	}
}

// DirFormatOption is a dir format option.
type DirFormatOption func(*dirFormatInfo)

// GitFormatOption is a git format option.
type GitFormatOption func(*gitFormatInfo)

// ReaderOption is an Reader option.
type ReaderOption func(*reader)

// WriterOption is an Writer option.
type WriterOption func(*writer)

// GetRefOption is a GetRef option
type GetRefOption func(*getRefOptions)

// WithAllowedFormats limits the allowed formats to the given formats.
func WithAllowedFormats(formats ...string) GetRefOption {
	return func(getRefOptions *getRefOptions) {
		for _, format := range formats {
			getRefOptions.allowedFormats[normalizeFormat(format)] = struct{}{}
		}
	}
}

// GetFileOption is a GetFile option
type GetFileOption func(*getFileOptions)

// WithGetFileKeepFileCompression says to return s compressed.
func WithGetFileKeepFileCompression() GetFileOption {
	return func(getFileOptions *getFileOptions) {
		getFileOptions.keepFileCompression = true
	}
}

// GetBucketOption is a GetBucket option
type GetBucketOption func(*getBucketOptions)

// WithGetBucketExt is equivalent to normalpath.WithExt.
func WithGetBucketExt(ext string) GetBucketOption {
	return func(getBucketOptions *getBucketOptions) {
		getBucketOptions.transformerOptions = append(
			getBucketOptions.transformerOptions,
			normalpath.WithExt(ext),
		)
	}
}

// WithGetBucketExactPath is equivalent to normalpath.WithExactPath.
func WithGetBucketExactPath(exactPath string) GetBucketOption {
	return func(getBucketOptions *getBucketOptions) {
		getBucketOptions.transformerOptions = append(
			getBucketOptions.transformerOptions,
			normalpath.WithExactPath(exactPath),
		)
	}
}

// PutFileOption is a PutFile option
type PutFileOption func(*putFileOptions)

// WithPutFileNoFileCompression says to put s uncompressed.
func WithPutFileNoFileCompression() PutFileOption {
	return func(putFileOptions *putFileOptions) {
		putFileOptions.noFileCompression = true
	}
}
