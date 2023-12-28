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
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
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
}

func newReader(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	options ...ReaderOption,
) *reader {
	reader := &reader{
		logger:            logger,
		storageosProvider: storageosProvider,
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

func (r *reader) GetReadBucketCloser(
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
			getBucketOptions.terminateFunc,
		)
	case DirRef:
		readWriteBucket, err := r.getDirBucket(
			ctx,
			container,
			t,
			getBucketOptions.terminateFunc,
		)
		if err != nil {
			return nil, err
		}
		return newReadBucketCloserForReadWriteBucket(readWriteBucket), nil
	case GitRef:
		return r.getGitBucket(
			ctx,
			container,
			t,
			getBucketOptions.terminateFunc,
		)
	case ProtoFileRef:
		return r.getProtoFileBucket(
			ctx,
			container,
			t,
			getBucketOptions.terminateFunc,
			getBucketOptions.protoFileTerminateFunc,
		)
	default:
		return nil, fmt.Errorf("unknown BucketRef type: %T", bucketRef)
	}
}

func (r *reader) GetReadWriteBucket(
	ctx context.Context,
	container app.EnvStdinContainer,
	dirRef DirRef,
	options ...GetBucketOption,
) (ReadWriteBucket, error) {
	getBucketOptions := newGetBucketOptions()
	for _, option := range options {
		option(getBucketOptions)
	}
	return r.getDirBucket(
		ctx,
		container,
		dirRef,
		getBucketOptions.terminateFunc,
	)
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
	terminateFunc TerminateFunc,
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
	return getReadBucketCloserForBucket(ctx, r.logger, storage.NopReadBucketCloser(readWriteBucket), subDirPath, terminateFunc)
}

func (r *reader) getDirBucket(
	ctx context.Context,
	container app.EnvStdinContainer,
	dirRef DirRef,
	terminateFunc TerminateFunc,
) (ReadWriteBucket, error) {
	if !r.localEnabled {
		return nil, NewReadLocalDisabledError()
	}
	return getReadWriteBucketForOS(ctx, r.logger, r.storageosProvider, dirRef.Path(), terminateFunc)
}

func (r *reader) getProtoFileBucket(
	ctx context.Context,
	container app.EnvStdinContainer,
	protoFileRef ProtoFileRef,
	terminateFunc TerminateFunc,
	protoFileTerminateFunc TerminateFunc,
) (ReadBucketCloser, error) {
	if !r.localEnabled {
		return nil, NewReadLocalDisabledError()
	}
	return getReadBucketCloserForOSProtoFile(
		ctx,
		r.logger,
		r.storageosProvider,
		protoFileRef.Path(),
		terminateFunc,
		protoFileTerminateFunc,
	)
}

func (r *reader) getGitBucket(
	ctx context.Context,
	container app.EnvStdinContainer,
	gitRef GitRef,
	terminateFunc TerminateFunc,
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
	return getReadBucketCloserForBucket(ctx, r.logger, storage.NopReadBucketCloser(readWriteBucket), subDirPath, terminateFunc)
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
	moduleKeys, err := bufmodule.GetModuleKeysForModuleRefs(
		ctx,
		r.moduleKeyProvider,
		moduleRef.ModuleRef(),
	)
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
	logger *zap.Logger,
	inputBucket storage.ReadBucketCloser,
	inputSubDirPath string,
	terminateFunc TerminateFunc,
) (ReadBucketCloser, error) {
	mapPath, subDirPath, _, err := getMapPathAndSubDirPath(ctx, logger, inputBucket, inputSubDirPath, terminateFunc)
	if err != nil {
		return nil, err
	}
	if mapPath != "." {
		inputBucket = storage.MapReadBucketCloser(
			inputBucket,
			storage.MapOnPrefix(mapPath),
		)
	}
	logger.Debug(
		"buffetch creating new bucket",
		zap.String("inputSubDirPath", inputSubDirPath),
		zap.String("mapPath", mapPath),
		zap.String("subDirPath", subDirPath),
	)
	return newReadBucketCloser(
		inputBucket,
		subDirPath,
		// This turns paths that were done relative to the root of the input into paths
		// that are now relative to the mapped bucket.
		//
		// This happens if you do i.e. .git#subdir=foo/bar --path foo/bar/baz.proto
		// We need to turn the path into baz.proto
		func(externalPath string) (string, error) {
			if filepath.IsAbs(externalPath) {
				return "", fmt.Errorf("%s: absolute paths cannot be used for this input type", externalPath)
			}
			if !normalpath.EqualsOrContainsPath(mapPath, externalPath, normalpath.Relative) {
				return "", fmt.Errorf("path %q from input does not contain path %q", mapPath, externalPath)
			}
			relPath, err := normalpath.Rel(mapPath, externalPath)
			if err != nil {
				return "", err
			}
			return normalpath.NormalizeAndValidate(relPath)
		},
	)
}

// Use for directory-based buckets.
func getReadWriteBucketForOS(
	ctx context.Context,
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	inputDirPath string,
	terminateFunc TerminateFunc,
) (ReadWriteBucket, error) {
	inputDirPath = normalpath.Normalize(inputDirPath)
	absInputDirPath, err := normalpath.NormalizeAndAbsolute(inputDirPath)
	if err != nil {
		return nil, err
	}
	osRootBucket, err := storageosProvider.NewReadWriteBucket(
		string(os.PathSeparator),
		// TODO: is this right? verify in deleted code
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return nil, err
	}
	mapPath, subDirPath, _, err := getMapPathAndSubDirPath(
		ctx,
		logger,
		osRootBucket,
		// This makes the path relative to the bucket.
		absInputDirPath[1:],
		terminateFunc,
	)
	if err != nil {
		return nil, attemptToFixOSRootBucketPathErrors(err)
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
		bucketPath = normalpath.Join("/", mapPath)
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
	logger.Debug(
		"creating new OS bucket for controlling workspace",
		zap.String("inputDirPath", inputDirPath),
		zap.String("workspacePath", bucketPath),
		zap.String("subDirPath", subDirPath),
	)
	return newReadWriteBucket(
		bucket,
		subDirPath,
		// This function turns paths into paths relative to the bucket.
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
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	protoFilePath string,
	terminateFunc TerminateFunc,
	protoFileTerminateFunc TerminateFunc,
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
	if err != nil {
		return nil, err
	}
	// mapPath is the path to the bucket that contains a buf.yaml.
	// subDirPath is the relative path from mapPath to the protoFileDirPath, but we don't use it.
	mapPath, _, terminate, err := getMapPathAndSubDirPath(
		ctx,
		logger,
		osRootBucket,
		// This makes the path relative to the bucket.
		absProtoFileDirPath[1:],
		protoFileTerminateFunc,
	)
	if err != nil {
		return nil, attemptToFixOSRootBucketPathErrors(err)
	}

	var protoTerminateFileDirPath string
	if !terminate {
		// If we did not find a buf.yaml or buf.work.yaml, use the current directory.
		// If the ProtoFileRef path was absolute, use an absolute path, otherwise relative.
		//
		// However, if the current directory does not contain the .proto file, we cannot use it,
		// as we need the bucket to encapsulate the .proto file. In this case, we fall back
		// to using the absolute directory of the .proto file. We need to do this because
		// PathForExternalPath (defined in getReadWriteBucketForOS) needs to make sure that
		// a given path can be made relative to the bucket, and be normalized and validated.
		if filepath.IsAbs(normalpath.Unnormalize(protoFileDirPath)) {
			pwd, err := osext.Getwd()
			if err != nil {
				return nil, err
			}
			protoTerminateFileDirPath = normalpath.Normalize(pwd)
		} else {
			protoTerminateFileDirPath = "."
		}
		absProtoTerminateFileDirPath, err := normalpath.NormalizeAndAbsolute(protoTerminateFileDirPath)
		if err != nil {
			return nil, err
		}
		if !normalpath.EqualsOrContainsPath(absProtoFileDirPath, absProtoTerminateFileDirPath, normalpath.Absolute) {
			logger.Debug(
				"did not find enclosing module or workspace for proto file ref and pwd does not encapsulate proto file",
				zap.String("protoFilePath", protoFilePath),
				zap.String("defaultingToAbsProtoFileDirPath", absProtoFileDirPath),
			)
			protoTerminateFileDirPath = absProtoFileDirPath
		} else {
			logger.Debug(
				"did not find enclosing module or workspace for proto file ref",
				zap.String("protoFilePath", protoFilePath),
				zap.String("defaultingToPwd", protoTerminateFileDirPath),
			)
		}
	} else {
		// We found a buf.yaml or buf.work.yaml, use that directory.
		// If we found a buf.yaml or buf.work.yaml and the ProtoFileRef path is absolute, use an absolute path, otherwise relative.
		if filepath.IsAbs(normalpath.Unnormalize(protoFileDirPath)) {
			protoTerminateFileDirPath = normalpath.Join("/", mapPath)
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
		logger.Debug(
			"found enclosing module or workspace for proto file ref",
			zap.String("protoFilePath", protoFilePath),
			zap.String("enclosingDirPath", protoTerminateFileDirPath),
		)
	}
	// Now, build a workspace bucket based on the directory we found.
	// If the directory is a module directory, we'll get the enclosing workspace.
	// If the directory is a workspace directory, this will effectively be a no-op.
	readWriteBucket, err := getReadWriteBucketForOS(ctx, logger, storageosProvider, protoTerminateFileDirPath, terminateFunc)
	if err != nil {
		return nil, err
	}
	return newReadBucketCloserForReadWriteBucket(readWriteBucket), nil
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
	logger *zap.Logger,
	inputBucket storage.ReadBucket,
	inputSubDirPath string,
	terminateFunc TerminateFunc,
) (mapPath string, subDirPath string, terminate bool, retErr error) {
	inputSubDirPath, err := normalpath.NormalizeAndValidate(inputSubDirPath)
	if err != nil {
		return "", "", false, err
	}
	// The for loops would take care of this base case, but we don't want
	// to call storage.MapReadBucket unless we have to.
	if terminateFunc == nil {
		return inputSubDirPath, ".", false, nil
	}
	// We can't do this in a traditional loop like this:
	//
	// for curDirPath := inputSubDirPath; curDirPath != "."; curDirPath = normalpath.Dir(curDirPath) {
	//
	// If we do that, then we don't run terminateFunc for ".", which we want to so that we get
	// the correct value for the terminate bool.
	//
	// Instead, we effectively do a do-while loop.
	curDirPath := inputSubDirPath
	for {
		terminate, err := terminateFunc(ctx, inputBucket, curDirPath, inputSubDirPath)
		if err != nil {
			return "", "", false, err
		}
		if terminate {
			logger.Debug(
				"buffetch termination found",
				zap.String("curDirPath", curDirPath),
				zap.String("inputSubDirPath", inputSubDirPath),
			)
			subDirPath, err := normalpath.Rel(curDirPath, inputSubDirPath)
			if err != nil {
				return "", "", false, err
			}
			return curDirPath, subDirPath, true, nil
		}
		if curDirPath == "." {
			// Do this instead. This makes this loop effectively a do-while loop.
			break
		}
		curDirPath = normalpath.Dir(curDirPath)
	}
	logger.Debug(
		"buffetch no termination found",
		zap.String("inputSubDirPath", inputSubDirPath),
	)
	return inputSubDirPath, ".", false, nil
}

// We attempt to fix up paths we get back to better printing to the user.
// Without this, we'll get things like "stat: Users/foo/path/to/input: does not exist"
// based on our usage of osRootBucket and absProtoFileDirPath above. While we won't
// break any contracts printing these out, this is confusing to the user, so this is
// our attempt to fix that.
//
// This is going to take away other intermediate errors unfortunately.
func attemptToFixOSRootBucketPathErrors(err error) error {
	var pathError *fs.PathError
	if errors.As(err, &pathError) {
		pwd, err := osext.Getwd()
		if err != nil {
			return err
		}
		pwd = normalpath.Normalize(pwd)
		if normalpath.EqualsOrContainsPath(pwd, normalpath.Join("/", pathError.Path), normalpath.Absolute) {
			relPath, err := normalpath.Rel(pwd, normalpath.Join("/", pathError.Path))
			// Just ignore if this errors and do nothing.
			if err == nil {
				// Making a copy just to be super-safe.
				return &fs.PathError{
					Op:   pathError.Op,
					Path: relPath,
					Err:  pathError.Err,
				}
			}
		}
	}
	return err
}

type getFileOptions struct {
	keepFileCompression bool
}

func newGetFileOptions() *getFileOptions {
	return &getFileOptions{}
}

type getBucketOptions struct {
	terminateFunc          TerminateFunc
	protoFileTerminateFunc TerminateFunc
}

func newGetBucketOptions() *getBucketOptions {
	return &getBucketOptions{}
}

type getModuleOptions struct{}
