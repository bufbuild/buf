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

package internal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/httpauth"
	"github.com/bufbuild/buf/private/pkg/ioext"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/osext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagearchive"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/klauspost/compress/zstd"
	"github.com/klauspost/pgzip"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type reader struct {
	logger            *zap.Logger
	storageosProvider storageos.Provider

	localEnabled bool
	stdioEnabled bool

	httpEnabled       bool
	httpClient        *http.Client
	httpAuthenticator httpauth.Authenticator

	gitEnabled bool
	gitCloner  git.Cloner

	moduleEnabled     bool
	moduleKeyProvider bufmodule.ModuleKeyProvider
	tracer            trace.Tracer
}

func newReader(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	options ...ReaderOption,
) *reader {
	reader := &reader{
		logger:            logger,
		storageosProvider: storageosProvider,
		tracer:            otel.GetTracerProvider().Tracer("bufbuild/buf"),
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
) (ReadBucketCloser, error) {
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
			getBucketOptions.terminateFileNames,
		)
	case DirRef:
		return r.getDirBucket(
			ctx,
			container,
			t,
			getBucketOptions.terminateFileNames,
		)
	case GitRef:
		return r.getGitBucket(
			ctx,
			container,
			t,
			getBucketOptions.terminateFileNames,
		)
	case ProtoFileRef:
		return r.getProtoFileBucket(
			ctx,
			container,
			t,
			getBucketOptions.terminateFileNames,
			getBucketOptions.protoFileTerminateFileNames,
		)
	default:
		return nil, fmt.Errorf("unknown BucketRef type: %T", bucketRef)
	}
}

func (r *reader) GetModuleKey(
	ctx context.Context,
	container app.EnvStdinContainer,
	moduleRef ModuleRef,
	_ ...GetModuleOption,
) (bufmodule.ModuleKey, error) {
	switch t := moduleRef.(type) {
	case ModuleRef:
		return r.getModuleKey(
			ctx,
			container,
			t,
		)
	default:
		return nil, fmt.Errorf("unknown ModuleRef type: %T", moduleRef)
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
	terminateFileNames []string,
) (_ ReadBucketCloser, retErr error) {
	subDirPath, err := normalpath.NormalizeAndValidate(archiveRef.SubDirPath())
	if err != nil {
		return nil, err
	}
	readCloser, size, err := r.getFileReadCloserAndSize(ctx, container, archiveRef, false)
	if err != nil {
		return nil, err
	}
	readWriteBucket := storagemem.NewReadWriteBucket()
	ctx, span := r.tracer.Start(ctx, "unarchive")
	defer span.End()
	defer func() {
		retErr = multierr.Append(retErr, readCloser.Close())
		if retErr != nil {
			span.RecordError(retErr)
			span.SetStatus(codes.Error, retErr.Error())
		}
	}()
	switch archiveType := archiveRef.ArchiveType(); archiveType {
	case ArchiveTypeTar:
		if err := storagearchive.Untar(
			ctx,
			readCloser,
			readWriteBucket,
			nil,
			archiveRef.StripComponents(),
		); err != nil {
			return nil, err
		}
	case ArchiveTypeZip:
		var readerAt io.ReaderAt
		if size < 0 {
			data, err := io.ReadAll(readCloser)
			if err != nil {
				return nil, err
			}
			readerAt = bytes.NewReader(data)
			size = int64(len(data))
		} else {
			readerAt, err = ioext.ReaderAtForReader(readCloser)
			if err != nil {
				return nil, err
			}
		}
		if err := storagearchive.Unzip(
			ctx,
			readerAt,
			size,
			readWriteBucket,
			nil,
			archiveRef.StripComponents(),
		); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown ArchiveType: %v", archiveType)
	}
	return getReadBucketCloserForBucket(ctx, storage.NopReadBucketCloser(readWriteBucket), subDirPath, terminateFileNames)
}

func (r *reader) getDirBucket(
	ctx context.Context,
	container app.EnvStdinContainer,
	dirRef DirRef,
	terminateFileNames []string,
) (ReadBucketCloser, error) {
	if !r.localEnabled {
		return nil, NewReadLocalDisabledError()
	}
	return getReadBucketCloserForOS(ctx, r.storageosProvider, dirRef.Path(), terminateFileNames)
}

func (r *reader) getProtoFileBucket(
	ctx context.Context,
	container app.EnvStdinContainer,
	protoFileRef ProtoFileRef,
	terminateFileNames []string,
	protoFileTerminateFileNames []string,
) (ReadBucketCloser, error) {
	if !r.localEnabled {
		return nil, NewReadLocalDisabledError()
	}
	return getReadBucketCloserForOSProtoFile(
		ctx,
		r.storageosProvider,
		protoFileRef.Path(),
		terminateFileNames,
		protoFileTerminateFileNames,
	)
}

func (r *reader) getGitBucket(
	ctx context.Context,
	container app.EnvStdinContainer,
	gitRef GitRef,
	terminateFileNames []string,
) (_ ReadBucketCloser, retErr error) {
	if !r.gitEnabled {
		return nil, NewReadGitDisabledError()
	}
	if r.gitCloner == nil {
		return nil, errors.New("git cloner is nil")
	}
	subDirPath, err := normalpath.NormalizeAndValidate(gitRef.SubDirPath())
	if err != nil {
		return nil, err
	}
	gitURL, err := getGitURL(gitRef)
	if err != nil {
		return nil, err
	}
	readWriteBucket := storagemem.NewReadWriteBucket()
	if err := r.gitCloner.CloneToBucket(
		ctx,
		container,
		gitURL,
		gitRef.Depth(),
		readWriteBucket,
		git.CloneToBucketOptions{
			Name:              gitRef.GitName(),
			RecurseSubmodules: gitRef.RecurseSubmodules(),
		},
	); err != nil {
		return nil, fmt.Errorf("could not clone %s: %v", gitURL, err)
	}
	return getReadBucketCloserForBucket(ctx, storage.NopReadBucketCloser(readWriteBucket), subDirPath, terminateFileNames)
}

func (r *reader) getModuleKey(
	ctx context.Context,
	container app.EnvStdinContainer,
	moduleRef ModuleRef,
) (bufmodule.ModuleKey, error) {
	if !r.moduleEnabled {
		return nil, NewReadModuleDisabledError()
	}
	if r.moduleKeyProvider == nil {
		return nil, errors.New("module key provider is nil")
	}
	moduleKeys, err := r.moduleKeyProvider.GetModuleKeysForModuleRefs(ctx, moduleRef.ModuleRef())
	if err != nil {
		return nil, err
	}
	if len(moduleKeys) != 1 {
		return nil, fmt.Errorf("expected 1 ModuleKey, got %d", len(moduleKeys))
	}
	return moduleKeys[0], nil
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
		gzipReadCloser, err := pgzip.NewReader(readCloser)
		if err != nil {
			return nil, -1, err
		}
		return ioext.CompositeReadCloser(
			gzipReadCloser,
			ioext.ChainCloser(
				gzipReadCloser,
				readCloser,
			),
		), -1, nil
	case CompressionTypeZstd:
		zstdDecoder, err := zstd.NewReader(readCloser)
		if err != nil {
			return nil, -1, err
		}
		zstdReadCloser := zstdDecoder.IOReadCloser()
		return ioext.CompositeReadCloser(
			zstdReadCloser,
			ioext.ChainCloser(
				zstdReadCloser,
				readCloser,
			),
		), -1, nil
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
			return nil, -1, NewReadHTTPDisabledError()
		}
		return r.getFileReadCloserAndSizePotentiallyCompressedHTTP(ctx, container, "http://"+fileRef.Path())
	case FileSchemeHTTPS:
		if !r.httpEnabled {
			return nil, -1, NewReadHTTPDisabledError()
		}
		return r.getFileReadCloserAndSizePotentiallyCompressedHTTP(ctx, container, "https://"+fileRef.Path())
	case FileSchemeLocal:
		if !r.localEnabled {
			return nil, -1, NewReadLocalDisabledError()
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
	case FileSchemeStdio, FileSchemeStdin:
		if !r.stdioEnabled {
			return nil, -1, NewReadStdioDisabledError()
		}
		return io.NopCloser(container.Stdin()), -1, nil
	case FileSchemeStdout:
		return nil, -1, errors.New("cannot read from stdout")
	case FileSchemeNull:
		return ioext.DiscardReadCloser, 0, nil
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
	if r.httpClient == nil {
		return nil, 0, errors.New("http client is nil")
	}
	if r.httpAuthenticator == nil {
		return nil, 0, errors.New("http authenticator is nil")
	}
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
	case GitSchemeGit:
		return "git://" + gitRef.Path(), nil
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

// Use for memory buckets i.e. archive and git.
func getReadBucketCloserForBucket(
	ctx context.Context,
	inputBucket storage.ReadBucketCloser,
	inputSubDirPath string,
	terminateFileNames []string,
) (ReadBucketCloser, error) {
	mapPath, subDirPath, _, err := getMapPathAndSubDirPath(ctx, inputBucket, inputSubDirPath, terminateFileNames)
	if err != nil {
		return nil, err
	}
	if mapPath != "." {
		inputBucket = storage.MapReadBucketCloser(
			inputBucket,
			storage.MapOnPrefix(mapPath),
		)
	}
	return newReadBucketCloser(
		inputBucket,
		subDirPath,
		func(externalPath string) (string, error) {
			return normalpath.NormalizeAndValidate(externalPath)
		},
	)
}

// Use for directory-based buckets.
func getReadBucketCloserForOS(
	ctx context.Context,
	storageosProvider storageos.Provider,
	inputDirPath string,
	terminateFileNames []string,
) (ReadBucketCloser, error) {
	inputDirPath = normalpath.Normalize(inputDirPath)
	absInputDirPath, err := normalpath.NormalizeAndAbsolute(inputDirPath)
	if err != nil {
		return nil, err
	}
	osRootBucket, err := storageosProvider.NewReadWriteBucket(
		"/",
		// TODO: is this right? verify in deleted code
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return nil, err
	}
	mapPath, subDirPath, _, err := getMapPathAndSubDirPath(
		ctx,
		osRootBucket,
		// This makes the path relative to the bucket.
		absInputDirPath[1:],
		terminateFileNames,
	)
	if err != nil {
		return nil, err
	}
	// Examples:
	//
	// inputDirPath: path/to/foo
	// terminateFileLocation: path/to
	// returnMapPath: path/to
	// returnSubDirPath: foo
	// Make bucket on: Rel(pwd, returnMapPath)
	//
	// inputDirPath: /users/alice/path/to/foo
	// terminateFileLocation: /users/alice/path/to
	// returnMapPath: /users/alice/path/to
	// returnSubDirPath: foo
	// Make bucket on: os.PathSeparator + returnMapPath (since absolute)
	var bucketPath string
	if filepath.IsAbs(normalpath.Unnormalize(inputDirPath)) {
		bucketPath = string(os.PathSeparator) + mapPath
	} else {
		pwd, err := osext.Getwd()
		if err != nil {
			return nil, err
		}
		pwd = normalpath.Normalize(pwd)
		// Deleting leading os.PathSeparator so we can make mapPath relative.
		bucketPath, err = normalpath.Rel(pwd[1:], mapPath)
		if err != nil {
			return nil, err
		}
	}
	bucket, err := storageosProvider.NewReadWriteBucket(
		bucketPath,
		// TODO: is this right? verify in deleted code
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return nil, err
	}
	return newReadBucketCloser(
		storage.NopReadBucketCloser(bucket),
		subDirPath,
		func(externalPath string) (string, error) {
			absBucketPath, err := filepath.Abs(normalpath.Unnormalize(bucketPath))
			if err != nil {
				return "", err
			}
			// We shouldn't actually need to unnormalize externalPath but we do anyways.
			absExternalPath, err := filepath.Abs(normalpath.Unnormalize(externalPath))
			if err != nil {
				return "", err
			}
			path, err := filepath.Rel(absBucketPath, absExternalPath)
			if err != nil {
				return "", err
			}
			return normalpath.NormalizeAndValidate(path)
		},
	)
}

// Use for ProtoFileRefs.
func getReadBucketCloserForOSProtoFile(
	ctx context.Context,
	storageosProvider storageos.Provider,
	protoFilePath string,
	terminateFileNames []string,
	protoFileTerminateFileNames []string,
) (ReadBucketCloser, error) {
	// First, we figure out which directory we consider to be the module that encapsulates
	// this ProtoFileRef. If we find a buf.yaml or buf.work.yaml, then we use that as the directory. If we
	// do not, we use the current directory as the directory.
	protoFileDirPath := normalpath.Dir(protoFilePath)
	absProtoFileDirPath, err := normalpath.NormalizeAndAbsolute(protoFileDirPath)
	if err != nil {
		return nil, err
	}
	osRootBucket, err := storageosProvider.NewReadWriteBucket(
		"/",
		// TODO: is this right? verify in deleted code
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	// mapPath is the path to the bucket that contains a buf.yaml.
	// subDirPath is the relative path from mapPath to the protoFileDirPath, but we don't use it.
	mapPath, _, foundProtoTerminateFileName, err := getMapPathAndSubDirPath(
		ctx,
		osRootBucket,
		// This makes the path relative to the bucket.
		absProtoFileDirPath[1:],
		append(
			terminateFileNames,
			protoFileTerminateFileNames...,
		),
	)
	if err != nil {
		return nil, err
	}

	var protoTerminateFileDirPath string
	if !foundProtoTerminateFileName {
		// If we did not find a buf.yaml or buf.work.yaml, use the current directory.
		// If the ProtoFileRef path was absolute, use an absolute path, otherwise relative.
		if filepath.IsAbs(normalpath.Unnormalize(protoFileDirPath)) {
			pwd, err := osext.Getwd()
			if err != nil {
				return nil, err
			}
			protoTerminateFileDirPath = normalpath.Normalize(pwd)
		} else {
			protoTerminateFileDirPath = "."
		}
	} else {
		// We found a buf.yaml or buf.work.yaml, use that directory.
		// If we found a buf.yaml or buf.workl.yaml and the ProtoFileRef path is absolute, use an absolute path, otherwise relative.
		if filepath.IsAbs(normalpath.Unnormalize(protoFileDirPath)) {
			protoTerminateFileDirPath = string(os.PathSeparator) + mapPath
		} else {
			pwd, err := osext.Getwd()
			if err != nil {
				return nil, err
			}
			pwd = normalpath.Normalize(pwd)
			// Deleting leading os.PathSeparator so we can make mapPath relative.
			protoTerminateFileDirPath, err = normalpath.Rel(pwd[1:], mapPath)
			if err != nil {
				return nil, err
			}
		}
	}
	// Now, build a workspace bucket based on the module we found (either buf.yaml or current directory)
	// TODO: do we do filtering of bucket in bufwire right now? Need to bring that up here or
	// add as targeting files when constructing a Workspace.
	return getReadBucketCloserForOS(ctx, storageosProvider, protoTerminateFileDirPath, terminateFileNames)
}

// Gets two values:
//
//   - The directory relative to the bucket that the bucket should be mapped onto.
//   - A new subDirPath that matches the inputSubDirPath but for the new ReadBucketCloser.
//
// Examples:
//
// inputSubDirPath: path/to/foo
// terminateFileLocation: path/to
// returnMapPath: path/to
// returnSubDirPath: foo
//
// inputSubDirPath: users/alice/path/to/foo
// terminateFileLocation: users/alice/path/to
// returnMapPath: users/alice/path/to
// returnSubDirPath: foo
//
// inputBucket: path/to/foo
// terminateFileLocation: NONE
// returnMapPath: path/to/foo
// returnSubDirPath: .

// inputSubDirPath: .
// terminateFileLocation: NONE
// returnMapPath: .
// returnSubDirPath: .
func getMapPathAndSubDirPath(
	ctx context.Context,
	inputBucket storage.ReadBucket,
	inputSubDirPath string,
	terminateFileNames []string,
) (mapPath string, subDirPath string, foundTerminateFileName bool, retErr error) {
	inputSubDirPath, err := normalpath.NormalizeAndValidate(inputSubDirPath)
	if err != nil {
		return "", "", false, err
	}
	// The for loops would take care of these base cases, but we don't want to call storage.MapReadBucket
	// unless we have to.
	if len(terminateFileNames) == 0 {
		return inputSubDirPath, ".", false, nil
	}
	for curPath := inputSubDirPath; curPath != "."; curPath = normalpath.Dir(curPath) {
		for _, terminateFileName := range terminateFileNames {
			_, err := inputBucket.Stat(ctx, normalpath.Join(curPath, terminateFileName))
			if err == nil {
				subDirPath, err := normalpath.Rel(curPath, inputSubDirPath)
				if err != nil {
					return "", "", false, err
				}
				return curPath, subDirPath, true, nil
			}
		}
	}
	return inputSubDirPath, ".", false, nil
}

type getFileOptions struct {
	keepFileCompression bool
}

func newGetFileOptions() *getFileOptions {
	return &getFileOptions{}
}

type getBucketOptions struct {
	terminateFileNames          []string
	protoFileTerminateFileNames []string
}

func newGetBucketOptions() *getBucketOptions {
	return &getBucketOptions{}
}

type getModuleOptions struct{}
