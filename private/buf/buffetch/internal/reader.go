// Copyright 2020-2024 Buf Technologies, Inc.
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

	moduleEnabled  bool
	moduleReader   bufmodule.ModuleReader
	moduleResolver bufmodule.ModuleResolver
	tracer         trace.Tracer
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
) (ReadBucketCloserWithTerminateFileProvider, error) {
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
		)
	default:
		return nil, fmt.Errorf("unknown BucketRef type: %T", bucketRef)
	}
}

func (r *reader) GetModule(
	ctx context.Context,
	container app.EnvStdinContainer,
	moduleRef ModuleRef,
	_ ...GetModuleOption,
) (bufmodule.Module, error) {
	switch t := moduleRef.(type) {
	case ModuleRef:
		return r.getModule(
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
	terminateFileNames [][]string,
) (_ ReadBucketCloserWithTerminateFileProvider, retErr error) {
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
	terminateFileProvider, err := getTerminateFileProviderForBucket(ctx, readWriteBucket, subDirPath, terminateFileNames)
	if err != nil {
		return nil, err
	}
	var terminateFileDirectoryPath string
	// Get the highest priority file found and use it as the terminate file directory path.
	terminateFiles := terminateFileProvider.GetTerminateFiles()
	if len(terminateFiles) != 0 {
		terminateFileDirectoryPath = terminateFiles[0].Path()
	}
	if terminateFileDirectoryPath != "" {
		relativeSubDirPath, err := normalpath.Rel(terminateFileDirectoryPath, subDirPath)
		if err != nil {
			return nil, err
		}
		readBucketCloser, err := newReadBucketCloser(
			storage.NopReadBucketCloser(storage.MapReadBucket(readWriteBucket, storage.MapOnPrefix(terminateFileDirectoryPath))),
			terminateFileDirectoryPath,
			relativeSubDirPath,
		)
		if err != nil {
			return nil, err
		}
		return newReadBucketCloserWithTerminateFiles(
			readBucketCloser,
			nil,
		), nil
	}
	var readBucket storage.ReadBucket = readWriteBucket
	if subDirPath != "." {
		readBucket = storage.MapReadBucket(readWriteBucket, storage.MapOnPrefix(subDirPath))
	}
	readBucketCloser, err := newReadBucketCloser(
		storage.NopReadBucketCloser(readBucket),
		"",
		"",
	)
	if err != nil {
		return nil, err
	}
	return newReadBucketCloserWithTerminateFiles(
		readBucketCloser,
		nil,
	), nil
}

func (r *reader) getDirBucket(
	ctx context.Context,
	container app.EnvStdinContainer,
	dirRef DirRef,
	terminateFileNames [][]string,
) (ReadBucketCloserWithTerminateFileProvider, error) {
	if !r.localEnabled {
		return nil, NewReadLocalDisabledError()
	}
	terminateFileProvider, err := getTerminateFileProviderForOS(dirRef.Path(), terminateFileNames)
	if err != nil {
		return nil, err
	}
	rootPath, dirRelativePath, err := r.getBucketRootPathAndRelativePath(ctx, container, dirRef.Path(), terminateFileProvider)
	if err != nil {
		return nil, err
	}
	readWriteBucket, err := r.storageosProvider.NewReadWriteBucket(
		rootPath,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return nil, err
	}
	if dirRelativePath != "" {
		// Verify that the subDirPath exists too.
		if _, err := r.storageosProvider.NewReadWriteBucket(
			normalpath.Join(rootPath, dirRelativePath),
			storageos.ReadWriteBucketWithSymlinksIfSupported(),
		); err != nil {
			return nil, err
		}
		readWriteBucketCloser, err := newReadWriteBucketCloser(
			storage.NopReadWriteBucketCloser(readWriteBucket),
			rootPath,
			dirRelativePath,
		)
		if err != nil {
			return nil, err
		}
		return newReadBucketCloserWithTerminateFiles(
			readWriteBucketCloser,
			nil,
		), nil
	}
	readBucketCloser, err := newReadWriteBucketCloser(
		storage.NopReadWriteBucketCloser(readWriteBucket),
		"",
		"",
	)
	if err != nil {
		return nil, err
	}
	return newReadBucketCloserWithTerminateFiles(
		readBucketCloser,
		nil,
	), nil
}

func (r *reader) getProtoFileBucket(
	ctx context.Context,
	container app.EnvStdinContainer,
	protoFileRef ProtoFileRef,
	terminateFileNames [][]string,
) (ReadBucketCloserWithTerminateFileProvider, error) {
	if !r.localEnabled {
		return nil, NewReadLocalDisabledError()
	}
	terminateFileProvider, err := getTerminateFileProviderForOS(normalpath.Dir(protoFileRef.Path()), terminateFileNames)
	if err != nil {
		return nil, err
	}
	rootPath, _, err := r.getBucketRootPathAndRelativePath(ctx, container, normalpath.Dir(protoFileRef.Path()), terminateFileProvider)
	if err != nil {
		return nil, err
	}
	readWriteBucket, err := r.storageosProvider.NewReadWriteBucket(
		rootPath,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return nil, err
	}
	readWriteBucketCloser, err := newReadWriteBucketCloser(
		storage.NopReadWriteBucketCloser(readWriteBucket),
		rootPath,
		"", // For ProtoFileRef, we default to using the working directory
	)
	if err != nil {
		return nil, err
	}
	return newReadBucketCloserWithTerminateFiles(
		readWriteBucketCloser,
		terminateFileProvider,
	), nil
}

// getBucketRootPathAndRelativePath is a helper function that returns the rootPath and relative
// path if available for the readWriteBucket based on the dirRef and protoFileRef.
func (r *reader) getBucketRootPathAndRelativePath(
	ctx context.Context,
	container app.EnvStdinContainer,
	dirPath string,
	terminateFileProvider TerminateFileProvider,
) (string, string, error) {
	// Set the terminateFile to the first terminateFilesPriorities result, since that's the highest
	// priority file.
	var terminateFileDirectoryAbsPath string
	// Get the highest priority file found and use it as the terminate file directory path.
	terminateFiles := terminateFileProvider.GetTerminateFiles()
	if len(terminateFiles) != 0 {
		terminateFileDirectoryAbsPath = terminateFiles[0].Path()
	}
	if terminateFileDirectoryAbsPath != "" {
		// If the terminate file exists, we need to determine the relative path from the
		// terminateFileDirectoryAbsPath to the target DirRef.Path.
		wd, err := osext.Getwd()
		if err != nil {
			return "", "", err
		}
		terminateFileRelativePath, err := normalpath.Rel(wd, terminateFileDirectoryAbsPath)
		if err != nil {
			return "", "", err
		}
		dirAbsPath, err := normalpath.NormalizeAndAbsolute(dirPath)
		if err != nil {
			return "", "", err
		}
		dirRelativePath, err := normalpath.Rel(terminateFileDirectoryAbsPath, dirAbsPath)
		if err != nil {
			return "", "", err
		}
		// It should be impossible for the dirRefRelativePath to be outside of the context
		// diretory, but we validate just to make sure.
		dirRelativePath, err = normalpath.NormalizeAndValidate(dirRelativePath)
		if err != nil {
			return "", "", err
		}
		rootPath := terminateFileRelativePath
		if filepath.IsAbs(normalpath.Unnormalize(dirPath)) {
			// If the input was provided as an absolute path,
			// we preserve it by initializing the workspace
			// bucket with an absolute path.
			rootPath = terminateFileDirectoryAbsPath
		}
		return rootPath, dirRelativePath, nil
	}
	return dirPath, "", nil
}

func (r *reader) getGitBucket(
	ctx context.Context,
	container app.EnvStdinContainer,
	gitRef GitRef,
	terminateFileNames [][]string,
) (_ ReadBucketCloserWithTerminateFileProvider, retErr error) {
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
	terminateFileProvider, err := getTerminateFileProviderForBucket(ctx, readWriteBucket, subDirPath, terminateFileNames)
	if err != nil {
		return nil, err
	}
	var terminateFileDirectoryPath string
	// Get the highest priority file found and use it as the terminate file directory path.
	terminateFiles := terminateFileProvider.GetTerminateFiles()
	if len(terminateFiles) != 0 {
		terminateFileDirectoryPath = terminateFiles[0].Path()
	}
	if terminateFileDirectoryPath != "" {
		relativeSubDirPath, err := normalpath.Rel(terminateFileDirectoryPath, subDirPath)
		if err != nil {
			return nil, err
		}
		readBucketCloser, err := newReadBucketCloser(
			storage.NopReadBucketCloser(storage.MapReadBucket(readWriteBucket, storage.MapOnPrefix(terminateFileDirectoryPath))),
			terminateFileDirectoryPath,
			relativeSubDirPath,
		)
		if err != nil {
			return nil, err
		}
		return newReadBucketCloserWithTerminateFiles(
			readBucketCloser,
			nil,
		), nil
	}
	var readBucket storage.ReadBucket = readWriteBucket
	if subDirPath != "." {
		readBucket = storage.MapReadBucket(readBucket, storage.MapOnPrefix(subDirPath))
	}
	readBucketCloser, err := newReadBucketCloser(
		storage.NopReadBucketCloser(readBucket),
		"",
		"",
	)
	if err != nil {
		return nil, err
	}
	return newReadBucketCloserWithTerminateFiles(
		readBucketCloser,
		nil,
	), nil
}

func (r *reader) getModule(
	ctx context.Context,
	container app.EnvStdinContainer,
	moduleRef ModuleRef,
) (bufmodule.Module, error) {
	if !r.moduleEnabled {
		return nil, NewReadModuleDisabledError()
	}
	if r.moduleReader == nil {
		return nil, errors.New("module reader is nil")
	}
	if r.moduleResolver == nil {
		return nil, errors.New("module resolver is nil")
	}
	modulePin, err := r.moduleResolver.GetModulePin(ctx, moduleRef.ModuleReference())
	if err != nil {
		return nil, err
	}
	return r.moduleReader.GetModule(ctx, modulePin)
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

// getTerminateFileProviderForBucket returns the directory path that contains
// one of the terminateFileNames, starting with the subDirPath and ascending until the root
// of the bucket.
func getTerminateFileProviderForBucket(
	ctx context.Context,
	readBucket storage.ReadBucket,
	subDirPath string,
	terminateFileNames [][]string,
) (TerminateFileProvider, error) {
	terminateFiles := make([]TerminateFile, len(terminateFileNames))
	if len(terminateFileNames) == 0 {
		return newTerminateFileProvider(terminateFiles), nil
	}
	terminateFileDirectoryPath := normalpath.Normalize(subDirPath)
	var foundFiles int
	for {
		foundTerminateFiles, err := terminateFilesInBucket(ctx, readBucket, terminateFileDirectoryPath, terminateFileNames)
		if err != nil {
			return nil, err
		}
		for i, terminateFile := range foundTerminateFiles {
			// We only want to return the first file for a hiearchy, so if a file already exists
			// for a layer of hierarchy, we do not check again.
			if terminateFiles[i] == nil {
				// If a file is found for a specific layer of hierarchy, we return the first file found
				// for that hierarchy.
				if terminateFile != nil {
					terminateFiles[i] = terminateFile
					foundFiles++
				}
			}
		}
		// If we have found a terminate file for each layer, we can return now.
		if foundFiles == len(terminateFileNames) {
			return newTerminateFileProvider(terminateFiles), nil
		}
		parent := normalpath.Dir(terminateFileDirectoryPath)
		if parent == terminateFileDirectoryPath {
			break
		}
		terminateFileDirectoryPath = parent
	}
	// The number of terminate files found is less than the number of layers of terminate files
	// we are accepting, so we must prune the nil values from the initial instantiation of the slice.
	var prunedTerminateFiles []TerminateFile
	for _, terminateFile := range terminateFiles {
		if terminateFile != nil {
			prunedTerminateFiles = append(prunedTerminateFiles, terminateFile)
		}
	}
	return newTerminateFileProvider(prunedTerminateFiles), nil
}

func terminateFilesInBucket(
	ctx context.Context,
	readBucket storage.ReadBucket,
	directoryPath string,
	paths [][]string,
) ([]TerminateFile, error) {
	foundPaths := make([]TerminateFile, len(paths))
	for i := range paths {
		// We need to check for the existence of all the terminate files, so we ascend and prepend
		// the current directory to determine the fully-qualified filepaths.
		//
		// For example:
		//   [][]string{
		//     ["buf.work.yaml", "buf.work"],
		//     ["buf.yaml", "buf.mod"],
		//   }
		// ==>
		//
		//   [][]string{
		//     ["/current/path/buf.work.yaml", "/current/path/buf.work"],
		//     ["/current/path/buf.yaml", "/current/path/buf.mod"],
		//   }
		for _, path := range paths[i] {
			path = normalpath.Join(directoryPath, path)
			exists, err := storage.Exists(ctx, readBucket, path)
			if err != nil {
				return nil, err
			}
			if exists {
				foundPaths[i] = newTerminateFile(normalpath.Base(path), normalpath.Dir(path))
				// We want the first file found for each layer of hierarchy.
				break
			}
		}
	}
	return foundPaths, nil
}

// getTerminateFileProviderForOS returns the directory that contains
// the terminateFileName, starting with the subDirPath and ascending until
// the root of the local filesystem.
func getTerminateFileProviderForOS(
	subDirPath string,
	terminateFileNames [][]string,
) (TerminateFileProvider, error) {
	terminateFiles := make([]TerminateFile, len(terminateFileNames))
	if len(terminateFileNames) == 0 {
		return newTerminateFileProvider(terminateFiles), nil
	}
	fileInfo, err := os.Stat(normalpath.Unnormalize(subDirPath))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &fs.PathError{Op: "stat", Path: subDirPath, Err: fs.ErrNotExist}
		}
		return nil, err
	}
	if !fileInfo.IsDir() {
		return nil, normalpath.NewError(normalpath.Unnormalize(subDirPath), errors.New("not a directory"))
	}
	terminateFileDirectoryPath, err := normalpath.NormalizeAndAbsolute(subDirPath)
	if err != nil {
		return nil, err
	}
	var foundFiles int
	for {
		foundTerminateFiles, err := terminateFilesOnOS(terminateFileDirectoryPath, terminateFileNames)
		if err != nil {
			return nil, err
		}
		// More terminate files were found than layers of hierarchy, return an error. This path
		// should be unreachable.
		if len(foundTerminateFiles) > len(terminateFileNames) {
			return nil, fmt.Errorf("more than one terminate file found per level of prioritization: %v", foundTerminateFiles)
		}
		for i, terminateFile := range foundTerminateFiles {
			// We only want to return the first file for a hiearchy, so if a file already exists
			// for a layer of hierarchy, we do not check again.
			if terminateFiles[i] == nil {
				// If a file is found for a specific layer of hierarchy, we return the first file found
				// for that hierarchy.
				if terminateFile != nil {
					terminateFiles[i] = terminateFile
					foundFiles++
				}
			}
		}
		// If we have found a terminate file for each layer, we can return now.
		if foundFiles == len(terminateFileNames) {
			return newTerminateFileProvider(terminateFiles), nil
		}
		parent := normalpath.Dir(terminateFileDirectoryPath)
		if parent == terminateFileDirectoryPath {
			break
		}
		terminateFileDirectoryPath = parent
	}
	// The number of terminate files found is less than the number of layers of terminate files
	// we are accepting, so we must prune the nil values from the initial instantiation of the slice.
	var prunedTerminateFiles []TerminateFile
	for _, terminateFile := range terminateFiles {
		if terminateFile != nil {
			prunedTerminateFiles = append(prunedTerminateFiles, terminateFile)
		}
	}
	return newTerminateFileProvider(prunedTerminateFiles), nil
}

func terminateFilesOnOS(directoryPath string, paths [][]string) ([]TerminateFile, error) {
	foundPaths := make([]TerminateFile, len(paths))
	for i := range paths {
		// We need to check for the existence of all the terminate files, so we ascend and prepend
		// the current directory to determine the fully-qualified filepaths.
		//
		// For example:
		//   [][]string{
		//     ["buf.work.yaml", "buf.work"],
		//     ["buf.yaml", "buf.mod"],
		//   }
		// ==>
		//
		//   [][]string{
		//     ["/current/path/buf.work.yaml", "/current/path/buf.work"],
		//     ["/current/path/buf.yaml", "/current/path/buf.mod"],
		//   }
		for _, path := range paths[i] {
			path = normalpath.Unnormalize(normalpath.Join(directoryPath, path))
			fileInfo, err := os.Stat(path)
			if err != nil && !os.IsNotExist(err) {
				return nil, err
			}
			if fileInfo != nil && !fileInfo.IsDir() {
				foundPaths[i] = newTerminateFile(normalpath.Base(path), normalpath.Dir(path))
				// We want the first file found for each layer of hierarchy.
				break
			}
		}
	}
	return foundPaths, nil
}

type getFileOptions struct {
	keepFileCompression bool
}

func newGetFileOptions() *getFileOptions {
	return &getFileOptions{}
}

type getBucketOptions struct {
	terminateFileNames [][]string
}

func newGetBucketOptions() *getBucketOptions {
	return &getBucketOptions{}
}

type getModuleOptions struct{}
