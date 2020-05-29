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
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/git"
	"github.com/bufbuild/buf/internal/pkg/httpauth"
	"github.com/bufbuild/buf/internal/pkg/instrument"
	"github.com/bufbuild/buf/internal/pkg/ioutilextended"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storagearchive"
	"github.com/bufbuild/buf/internal/pkg/storage/storagemem"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/klauspost/pgzip"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type reader struct {
	logger *zap.Logger

	localEnabled bool
	stdioEnabled bool

	httpEnabled       bool
	httpClient        *http.Client
	httpAuthenticator httpauth.Authenticator

	gitEnabled bool
	gitCloner  git.Cloner
}

func newReader(
	logger *zap.Logger,
	options ...ReaderOption,
) *reader {
	reader := &reader{
		logger: logger,
	}
	for _, option := range options {
		option(reader)
	}
	return reader
}

func (r *reader) GetFile(
	ctx context.Context,
	container app.EnvStdinContainer,
	fileRef FileRef,
	options ...GetFileOption,
) (io.ReadCloser, error) {
	getFileOptions := newGetFileOptions()
	for _, option := range options {
		option(getFileOptions)
	}
	switch t := fileRef.(type) {
	case SingleRef:
		return r.getSingle(
			ctx,
			container,
			t,
			getFileOptions.keepFileCompression,
		)
	case ArchiveRef:
		return r.getArchiveFile(
			ctx,
			container,
			t,
			getFileOptions.keepFileCompression,
		)
	default:
		return nil, fmt.Errorf("unknown FileRef type: %T", fileRef)
	}
}

func (r *reader) GetBucket(
	ctx context.Context,
	container app.EnvStdinContainer,
	bucketRef BucketRef,
	options ...GetBucketOption,
) (storage.ReadBucketCloser, error) {
	getBucketOptions := newGetBucketOptions()
	for _, option := range options {
		option(getBucketOptions)
	}
	switch t := bucketRef.(type) {
	case ArchiveRef:
		return r.getArchiveBucket(
			ctx,
			container,
			t,
			getBucketOptions.transformerOptions,
		)
	case DirRef:
		return r.getDirBucket(
			ctx,
			container,
			t,
			getBucketOptions.transformerOptions,
		)
	case GitRef:
		return r.getGitBucket(
			ctx,
			container,
			t,
			getBucketOptions.transformerOptions,
		)
	default:
		return nil, fmt.Errorf("unknown BucketRef type: %T", bucketRef)
	}
}

func (r *reader) getSingle(
	ctx context.Context,
	container app.EnvStdinContainer,
	singleRef SingleRef,
	keepFileCompression bool,
) (io.ReadCloser, error) {
	readCloser, _, err := r.getFileReadCloserAndSize(ctx, container, singleRef, keepFileCompression)
	return readCloser, err
}

func (r *reader) getArchiveFile(
	ctx context.Context,
	container app.EnvStdinContainer,
	archiveRef ArchiveRef,
	keepFileCompression bool,
) (io.ReadCloser, error) {
	readCloser, _, err := r.getFileReadCloserAndSize(ctx, container, archiveRef, keepFileCompression)
	return readCloser, err
}

func (r *reader) getArchiveBucket(
	ctx context.Context,
	container app.EnvStdinContainer,
	archiveRef ArchiveRef,
	transformerOptions []normalpath.TransformerOption,
) (_ storage.ReadBucketCloser, retErr error) {
	readCloser, size, err := r.getFileReadCloserAndSize(ctx, container, archiveRef, false)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, readCloser.Close())
	}()
	if stripComponents := archiveRef.StripComponents(); stripComponents > 0 {
		transformerOptions = append(
			transformerOptions,
			normalpath.WithStripComponents(stripComponents),
		)
	}
	readWriteBucketCloser := storagemem.NewReadWriteBucketCloser()
	defer func() {
		if retErr != nil {
			retErr = multierr.Append(retErr, readWriteBucketCloser.Close())
		}
	}()
	defer instrument.Start(r.logger, "unarchive").End()
	switch archiveType := archiveRef.ArchiveType(); archiveType {
	case ArchiveTypeTar:
		if err := storagearchive.Untar(
			ctx,
			readCloser,
			readWriteBucketCloser,
			transformerOptions...,
		); err != nil {
			return nil, err
		}
	case ArchiveTypeZip:
		var readerAt io.ReaderAt
		if size < 0 {
			data, err := ioutil.ReadAll(readCloser)
			if err != nil {
				return
			}
			readerAt = bytes.NewReader(data)
			size = int64(len(data))
		} else {
			readerAt, err = ioutilextended.ReaderAtForReader(readCloser)
			if err != nil {
				return nil, err
			}
		}
		if err := storagearchive.Unzip(
			ctx,
			readerAt,
			size,
			readWriteBucketCloser,
			transformerOptions...,
		); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown ArchiveType: %v", archiveType)
	}
	return readWriteBucketCloser, nil
}

func (r *reader) getDirBucket(
	ctx context.Context,
	container app.EnvStdinContainer,
	dirRef DirRef,
	transformerOptions []normalpath.TransformerOption,
) (storage.ReadBucketCloser, error) {
	if !r.localEnabled {
		return nil, newReadLocalDisabledError()
	}
	return storageos.NewReadBucketCloser(dirRef.Path())
}

func (r *reader) getGitBucket(
	ctx context.Context,
	container app.EnvStdinContainer,
	gitRef GitRef,
	transformerOptions []normalpath.TransformerOption,
) (_ storage.ReadBucketCloser, retErr error) {
	if !r.gitEnabled {
		return nil, newReadGitDisabledError()
	}
	gitURL, err := getGitURL(gitRef)
	if err != nil {
		return nil, err
	}
	readWriteBucketCloser := storagemem.NewReadWriteBucketCloser()
	defer func() {
		if retErr != nil {
			retErr = multierr.Append(retErr, readWriteBucketCloser.Close())
		}
	}()
	if err := r.gitCloner.CloneToBucket(
		ctx,
		container,
		gitURL,
		gitRef.GitRefName(),
		readWriteBucketCloser,
		git.CloneToBucketOptions{
			RecurseSubmodules:  gitRef.RecurseSubmodules(),
			TransformerOptions: transformerOptions,
		},
	); err != nil {
		return nil, fmt.Errorf("could not clone %s: %v", gitURL, err)
	}
	return readWriteBucketCloser, nil
}

func (r *reader) getFileReadCloserAndSize(
	ctx context.Context,
	container app.EnvStdinContainer,
	fileRef FileRef,
	keepFileCompression bool,
) (_ io.ReadCloser, _ int64, retErr error) {
	readCloser, size, err := r.getFileReadCloserAndSizePotentiallyCompressed(ctx, container, fileRef)
	if err != nil {
		return nil, -1, err
	}
	defer func() {
		if retErr != nil {
			retErr = multierr.Append(retErr, readCloser.Close())
		}
	}()
	if keepFileCompression {
		return readCloser, size, nil
	}
	switch compressionType := fileRef.CompressionType(); compressionType {
	case CompressionTypeNone:
		return readCloser, size, nil
	case CompressionTypeGzip:
		gzipReader, err := pgzip.NewReader(readCloser)
		if err != nil {
			return nil, -1, err
		}
		return ioutilextended.CompositeReadCloser(gzipReader, readCloser), -1, nil
	default:
		return nil, -1, fmt.Errorf("unknown CompressionType: %v", compressionType)
	}
}

// returns -1 if size unknown
func (r *reader) getFileReadCloserAndSizePotentiallyCompressed(
	ctx context.Context,
	container app.EnvStdinContainer,
	fileRef FileRef,
) (io.ReadCloser, int64, error) {
	switch fileScheme := fileRef.FileScheme(); fileScheme {
	case FileSchemeHTTP:
		if !r.httpEnabled {
			return nil, -1, newReadHTTPDisabledError()
		}
		return r.getFileReadCloserAndSizePotentiallyCompressedHTTP(ctx, container, "http://"+fileRef.Path())
	case FileSchemeHTTPS:
		if !r.httpEnabled {
			return nil, -1, newReadHTTPDisabledError()
		}
		return r.getFileReadCloserAndSizePotentiallyCompressedHTTP(ctx, container, "https://"+fileRef.Path())
	case FileSchemeLocal:
		if !r.localEnabled {
			return nil, -1, newReadLocalDisabledError()
		}
		file, err := os.Open(fileRef.Path())
		if err != nil {
			return nil, -1, err
		}
		fileInfo, err := file.Stat()
		if err != nil {
			return nil, -1, err
		}
		return file, fileInfo.Size(), nil
	case FileSchemeStdio:
		if !r.stdioEnabled {
			return nil, -1, newReadStdioDisabledError()
		}
		return ioutil.NopCloser(container.Stdin()), -1, nil
	case FileSchemeNull:
		return ioutilextended.DiscardReadCloser, 0, nil
	default:
		return nil, -1, fmt.Errorf("unknown FileScheme: %v", fileScheme)
	}
}

// the httpPath must have the scheme attached
func (r *reader) getFileReadCloserAndSizePotentiallyCompressedHTTP(
	ctx context.Context,
	container app.EnvStdinContainer,
	httpPath string,
) (io.ReadCloser, int64, error) {
	request, err := http.NewRequestWithContext(ctx, "GET", httpPath, nil)
	if err != nil {
		return nil, -1, err
	}
	if _, err := r.httpAuthenticator.SetAuth(container, request); err != nil {
		return nil, -1, err
	}
	response, err := r.httpClient.Do(request)
	if err != nil {
		return nil, -1, err
	}
	if response.StatusCode != http.StatusOK {
		err := fmt.Errorf("got HTTP status code %d", response.StatusCode)
		if response.Body != nil {
			return nil, -1, multierr.Append(err, response.Body.Close())
		}
		return nil, -1, err
	}
	// ContentLength is -1 if unknown, which is what we want
	return response.Body, response.ContentLength, nil
}

func getGitURL(gitRef GitRef) (string, error) {
	switch gitScheme := gitRef.GitScheme(); gitScheme {
	case GitSchemeHTTP:
		return "http://" + gitRef.Path(), nil
	case GitSchemeHTTPS:
		return "https://" + gitRef.Path(), nil
	case GitSchemeSSH:
		return "ssh://" + gitRef.Path(), nil
	case GitSchemeLocal:
		absPath, err := filepath.Abs(normalpath.Unnormalize(gitRef.Path()))
		if err != nil {
			return "", err
		}
		return "file://" + absPath, nil
	default:
		return "", fmt.Errorf("unknown GitScheme: %v", gitScheme)
	}
}

type getFileOptions struct {
	keepFileCompression bool
}

func newGetFileOptions() *getFileOptions {
	return &getFileOptions{}
}

type getBucketOptions struct {
	transformerOptions []normalpath.TransformerOption
}

func newGetBucketOptions() *getBucketOptions {
	return &getBucketOptions{}
}
