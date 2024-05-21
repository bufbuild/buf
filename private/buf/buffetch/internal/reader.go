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

	"github.com/bufbuild/buf/private/buf/buftarget"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/httpauth"
	"github.com/bufbuild/buf/private/pkg/ioext"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/osext"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagearchive"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/syserror"
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
	options ...GetReadBucketCloserOption,
) (retReadBucketCloser ReadBucketCloser, _ buftarget.BucketTargeting, retErr error) {
	getReadBucketCloserOptions := newGetReadBucketCloserOptions()
	for _, option := range options {
		option(getReadBucketCloserOptions)
	}
	if getReadBucketCloserOptions.copyToInMemory {
		defer func() {
			if retReadBucketCloser != nil {
				castReadBucketCloser, ok := retReadBucketCloser.(*readBucketCloser)
				if !ok {
					retErr = multierr.Append(
						retErr,
						syserror.Newf("expected *readBucketCloser but got %T", retReadBucketCloser),
					)
					return
				}
				var err error
				retReadBucketCloser, err = castReadBucketCloser.copyToInMemory(ctx)
				retErr = multierr.Append(retErr, err)
			}
		}()
	}
	switch t := bucketRef.(type) {
	case ArchiveRef:
		return r.getArchiveBucket(
			ctx,
			container,
			t,
			getReadBucketCloserOptions.targetPaths,
			getReadBucketCloserOptions.targetExcludePaths,
			getReadBucketCloserOptions.terminateFunc,
		)
	case DirRef:
		readWriteBucket, bucketTargeting, err := r.getDirBucket(
			ctx,
			container,
			t,
			getReadBucketCloserOptions.targetPaths,
			getReadBucketCloserOptions.targetExcludePaths,
			getReadBucketCloserOptions.terminateFunc,
		)
		if err != nil {
			return nil, nil, err
		}
		return newReadBucketCloserForReadWriteBucket(readWriteBucket), bucketTargeting, nil
	case GitRef:
		return r.getGitBucket(
			ctx,
			container,
			t,
			getReadBucketCloserOptions.targetPaths,
			getReadBucketCloserOptions.targetExcludePaths,
			getReadBucketCloserOptions.terminateFunc,
		)
	case ProtoFileRef:
		return r.getProtoFileBucket(
			ctx,
			container,
			t,
			getReadBucketCloserOptions.terminateFunc,
		)
	default:
		return nil, nil, fmt.Errorf("unknown BucketRef type: %T", bucketRef)
	}
}

func (r *reader) GetReadWriteBucket(
	ctx context.Context,
	container app.EnvStdinContainer,
	dirRef DirRef,
	options ...GetReadWriteBucketOption,
) (ReadWriteBucket, buftarget.BucketTargeting, error) {
	getReadWriteBucketOptions := newGetReadWriteBucketOptions()
	for _, option := range options {
		option(getReadWriteBucketOptions)
	}
	return r.getDirBucket(
		ctx,
		container,
		dirRef,
		getReadWriteBucketOptions.targetPaths,
		getReadWriteBucketOptions.targetExcludePaths,
		getReadWriteBucketOptions.terminateFunc,
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
	targetPaths []string,
	targetExcludePaths []string,
	terminateFunc buftarget.TerminateFunc,
) (ReadBucketCloser, buftarget.BucketTargeting, error) {
	readCloser, size, err := r.getFileReadCloserAndSize(ctx, container, archiveRef, false)
	if err != nil {
		return nil, nil, err
	}
	readWriteBucket := storagemem.NewReadWriteBucket()
	switch archiveType := archiveRef.ArchiveType(); archiveType {
	case ArchiveTypeTar:
		if err := storagearchive.Untar(
			ctx,
			readCloser,
			readWriteBucket,
			storagearchive.UntarWithStripComponentCount(
				archiveRef.StripComponents(),
			),
		); err != nil {
			return nil, nil, err
		}
	case ArchiveTypeZip:
		var readerAt io.ReaderAt
		if size < 0 {
			data, err := io.ReadAll(readCloser)
			if err != nil {
				return nil, nil, err
			}
			readerAt = bytes.NewReader(data)
			size = int64(len(data))
		} else {
			readerAt, err = ioext.ReaderAtForReader(readCloser)
			if err != nil {
				return nil, nil, err
			}
		}
		if err := storagearchive.Unzip(
			ctx,
			readerAt,
			size,
			readWriteBucket,
			storagearchive.UnzipWithStripComponentCount(
				archiveRef.StripComponents(),
			),
		); err != nil {
			return nil, nil, err
		}
	default:
		return nil, nil, fmt.Errorf("unknown ArchiveType: %v", archiveType)
	}
	return getReadBucketCloserForBucket(
		ctx,
		r.logger,
		storage.NopReadBucketCloser(readWriteBucket),
		archiveRef.SubDirPath(),
		targetPaths,
		targetExcludePaths,
		terminateFunc,
	)
}

func (r *reader) getDirBucket(
	ctx context.Context,
	container app.EnvStdinContainer,
	dirRef DirRef,
	targetPaths []string,
	targetExcludePaths []string,
	terminateFunc buftarget.TerminateFunc,
) (ReadWriteBucket, buftarget.BucketTargeting, error) {
	if !r.localEnabled {
		return nil, nil, NewReadLocalDisabledError()
	}
	return getReadWriteBucketForOS(
		ctx,
		r.logger,
		r.storageosProvider,
		dirRef.Path(),
		targetPaths,
		targetExcludePaths,
		terminateFunc,
	)
}

func (r *reader) getProtoFileBucket(
	ctx context.Context,
	container app.EnvStdinContainer,
	protoFileRef ProtoFileRef,
	terminateFunc buftarget.TerminateFunc,
) (ReadBucketCloser, buftarget.BucketTargeting, error) {
	if !r.localEnabled {
		return nil, nil, NewReadLocalDisabledError()
	}
	return getReadBucketCloserForOSProtoFile(
		ctx,
		r.logger,
		r.storageosProvider,
		protoFileRef.Path(),
		terminateFunc,
	)
}

func (r *reader) getGitBucket(
	ctx context.Context,
	container app.EnvStdinContainer,
	gitRef GitRef,
	targetPaths []string,
	targetExcludePaths []string,
	terminateFunc buftarget.TerminateFunc,
) (ReadBucketCloser, buftarget.BucketTargeting, error) {
	if !r.gitEnabled {
		return nil, nil, NewReadGitDisabledError()
	}
	if r.gitCloner == nil {
		return nil, nil, errors.New("git cloner is nil")
	}
	gitURL, err := getGitURL(gitRef)
	if err != nil {
		return nil, nil, err
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
		return nil, nil, fmt.Errorf("could not clone %s: %v", gitURL, err)
	}
	return getReadBucketCloserForBucket(
		ctx,
		r.logger,
		storage.NopReadBucketCloser(readWriteBucket),
		gitRef.SubDirPath(),
		targetPaths,
		targetExcludePaths,
		terminateFunc,
	)
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
	moduleKeys, err := r.moduleKeyProvider.GetModuleKeysForModuleRefs(
		ctx,
		[]bufmodule.ModuleRef{moduleRef.ModuleRef()},
		bufmodule.DigestTypeB5,
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
	targetPaths []string,
	targetExcludePaths []string,
	terminateFunc buftarget.TerminateFunc,
) (ReadBucketCloser, buftarget.BucketTargeting, error) {
	if err := validatePaths(inputSubDirPath, targetPaths, targetExcludePaths); err != nil {
		return nil, nil, err
	}
	// For archive and git refs, target paths and target exclude paths are expected to be
	// mapped to the inputSubDirPath rather than the execution context.
	// This affects buftarget when checking and mapping paths against the controlling
	// workspace, so we need to ensure that all paths are properly mapped.
	targetPaths, targetExcludePaths = mapTargetPathsAndTargetExcludePathsForArchiveAndGitRefs(
		inputSubDirPath,
		targetPaths,
		targetExcludePaths,
	)
	bucketTargeting, err := buftarget.NewBucketTargeting(
		ctx,
		logger,
		inputBucket,
		inputSubDirPath,
		targetPaths,
		targetExcludePaths,
		terminateFunc,
	)
	if err != nil {
		return nil, nil, err
	}
	// If a controlling workspace is found, then we map the bucket on the controlling
	// workspace path. We only need to remap the bucket in the case where a controlling
	// workspace is found. In the case where no controlling workspace is found, bufworkspace
	// will handle the SubDirPath through workspace targeting given the bucket and BucketTargeting.
	//
	// We return the same bucket targeting information, since BucketTargeting
	// maps the target paths and target exclude paths to the controlling workspace path when
	// one is found.
	bucketPath := "."
	if bucketTargeting.ControllingWorkspace() != nil && bucketTargeting.ControllingWorkspace().Path() != "." {
		bucketPath = bucketTargeting.ControllingWorkspace().Path()
		inputBucket = storage.MapReadBucketCloser(
			inputBucket,
			storage.MapOnPrefix(bucketPath),
		)
	}
	logger.Debug(
		"buffetch creating new bucket",
		zap.String("bucketPath", bucketPath),
		zap.Strings("targetPaths", bucketTargeting.TargetPaths()),
	)
	readBucketCloser := newReadBucketCloser(inputBucket, bucketTargeting)
	return readBucketCloser, bucketTargeting, nil
}

// Use for directory-based buckets.
func getReadWriteBucketForOS(
	ctx context.Context,
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	inputDirPath string,
	targetPaths []string,
	targetExcludePaths []string,
	terminateFunc buftarget.TerminateFunc,
) (ReadWriteBucket, buftarget.BucketTargeting, error) {
	fsRoot, fsRootInputSubDirPath, err := fsRootAndFSRelPathForPath(inputDirPath)
	if err != nil {
		return nil, nil, err
	}
	fsRootTargetPaths := make([]string, len(targetPaths))
	for i, targetPath := range targetPaths {
		_, fsRootTargetPath, err := fsRootAndFSRelPathForPath(targetPath)
		if err != nil {
			return nil, nil, err
		}
		fsRootTargetPaths[i] = fsRootTargetPath
	}
	fsRootTargetExcludePaths := make([]string, len(targetExcludePaths))
	for i, targetExcludePath := range targetExcludePaths {
		_, fsRootTargetExcludePath, err := fsRootAndFSRelPathForPath(targetExcludePath)
		if err != nil {
			return nil, nil, err
		}
		fsRootTargetExcludePaths[i] = fsRootTargetExcludePath
	}
	osRootBucket, err := storageosProvider.NewReadWriteBucket(
		fsRoot,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return nil, nil, err
	}
	osRootBucketTargeting, err := buftarget.NewBucketTargeting(
		ctx,
		logger,
		osRootBucket,
		fsRootInputSubDirPath,
		fsRootTargetPaths,
		fsRootTargetExcludePaths,
		terminateFunc,
	)
	if err != nil {
		return nil, nil, attemptToFixOSRootBucketPathErrors(fsRoot, err)
	}
	// osRootBucketTargeting returns the information on whether or not a controlling
	// workspace was found based on the inputDirPath.
	//
	// *** CONTROLLING WOKRSPACE FOUND ***
	//
	// In the case where a controlling workspace is found, we want to create a new bucket
	// for the controlling workspace.
	// If the inputDirPath is an absolute path, we want to use an absolute path:
	//
	//    bucketPath := fsRoot + controllingWorkspace.Path()
	//
	// If the inputDirPath is a relative path, we want to use a relative path between the
	// current working directory (pwd) and controlling workspace.
	//
	//    bucketPath := Rel(Rel(fsRoot, pwd), controllingWorkspace.Path())
	//
	// We do not need to remap the input dir, target paths, and target exclude paths
	// returned by buftarget.BucketTargeting, because they are already relative to the
	// controlling workpsace.
	//
	// *** CONTROLLING WOKRSPACE NOT FOUND ***
	//
	// In the case where a controlling workpsace is not found, we have three outcomes for
	// creating a new bucket.
	// If the inputDirPath is an absolute path, we want to use an absolute path to the input
	// path:
	//
	//    bucketPath := Abs(inputDirPath)
	//
	// If the inputDirPath is a relative path, there are two possible outcomes: the input
	// dir is within the context of the working directory or is outside of the context of
	// the working directory.
	//
	// In the case where the input dir, is within the context of the working directory, we
	// use pwd:
	//
	//    bucketPath := Rel(fsRoot,pwd)
	//
	// In the case where the input dir is outside the context of the working directory, we
	// use the input dir relative to the pwd:
	//
	//    bucketPath := Rel(Rel(fsRoot,pwd), Rel(fsRoot, inputDirPath))
	//
	// For all cases where no controlling workspace was found, we need to remap the input
	// path, target paths, and target exclude paths to the root of the new bucket.
	var bucketPath string
	var inputDir string
	bucketTargetPaths := make([]string, len(osRootBucketTargeting.TargetPaths()))
	bucketTargetExcludePaths := make([]string, len(osRootBucketTargeting.TargetExcludePaths()))
	if controllingWorkspace := osRootBucketTargeting.ControllingWorkspace(); controllingWorkspace != nil {
		if filepath.IsAbs(normalpath.Unnormalize(inputDirPath)) {
			bucketPath = normalpath.Join(fsRoot, osRootBucketTargeting.ControllingWorkspace().Path())
		} else {
			// Relative input dir
			pwdFSRelPath, err := getPWDFSRelPath()
			if err != nil {
				return nil, nil, err
			}
			bucketPath, err = normalpath.Rel(pwdFSRelPath, osRootBucketTargeting.ControllingWorkspace().Path())
			if err != nil {
				return nil, nil, err
			}
		}
		inputDir = osRootBucketTargeting.SubDirPath()
		bucketTargetPaths = osRootBucketTargeting.TargetPaths()
		bucketTargetExcludePaths = osRootBucketTargeting.TargetExcludePaths()
	} else {
		// No controlling workspace found
		if filepath.IsAbs(normalpath.Unnormalize(inputDirPath)) {
			bucketPath = inputDirPath
		} else {
			// Relative input dir
			pwdFSRelPath, err := getPWDFSRelPath()
			if err != nil {
				return nil, nil, err
			}
			if filepath.IsLocal(normalpath.Unnormalize(inputDirPath)) {
				// Use current working directory
				bucketPath = "."
			} else {
				// Relative input dir outside of working directory context
				bucketPath, err = normalpath.Rel(pwdFSRelPath, fsRootInputSubDirPath)
				if err != nil {
					return nil, nil, err
				}
			}
		}
		// Map input dir, target paths, and target exclude paths to the new bucket path.
		_, bucketPathFSRelPath, err := fsRootAndFSRelPathForPath(bucketPath)
		if err != nil {
			return nil, nil, err
		}
		inputDir, err = normalpath.Rel(bucketPathFSRelPath, osRootBucketTargeting.SubDirPath())
		if err != nil {
			return nil, nil, err
		}
		for i, targetPath := range osRootBucketTargeting.TargetPaths() {
			bucketTargetPaths[i], err = normalpath.Rel(bucketPathFSRelPath, targetPath)
			if err != nil {
				return nil, nil, err
			}
		}
		for i, targetExcludePath := range osRootBucketTargeting.TargetExcludePaths() {
			bucketTargetExcludePaths[i], err = normalpath.Rel(bucketPathFSRelPath, targetExcludePath)
			if err != nil {
				return nil, nil, err
			}
		}
	}
	bucket, err := storageosProvider.NewReadWriteBucket(
		bucketPath,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return nil, nil, err
	}
	bucketTargeting, err := buftarget.NewBucketTargeting(
		ctx,
		logger,
		bucket,
		inputDir,
		bucketTargetPaths,
		bucketTargetExcludePaths,
		terminateFunc,
	)
	if err != nil {
		return nil, nil, err
	}
	readWriteBucket := newReadWriteBucket(bucket, bucketPath, bucketTargeting)
	return readWriteBucket, bucketTargeting, nil
}

// Use for ProtoFileRefs.
func getReadBucketCloserForOSProtoFile(
	ctx context.Context,
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	protoFilePath string,
	terminateFunc buftarget.TerminateFunc,
) (ReadBucketCloser, buftarget.BucketTargeting, error) {
	// For proto file refs, we treat the input directory as the directory of
	// the file and the file as a target path.
	// No other target paths and target exclude paths are supported with
	// proto file refs.
	protoFileDir := normalpath.Dir(protoFilePath)
	readWriteBucket, bucketTargeting, err := getReadWriteBucketForOS(
		ctx,
		logger,
		storageosProvider,
		protoFileDir,
		[]string{protoFilePath},
		nil, // no target exclude paths are supported for proto file refs
		terminateFunc,
	)
	if err != nil {
		return nil, nil, err
	}
	return newReadBucketCloserForReadWriteBucket(readWriteBucket), bucketTargeting, nil
}

// getPWDFSRelPath is a helper function that gets the relative path of the current working
// directory to the FS root.
func getPWDFSRelPath() (string, error) {
	pwd, err := osext.Getwd()
	if err != nil {
		return "", err
	}
	_, pwdFSRelPath, err := fsRootAndFSRelPathForPath(pwd)
	if err != nil {
		return "", err
	}
	return pwdFSRelPath, nil
}

// fsRootAndFSRelPathForPath is a helper function that takes a path and returns the FS
// root and relative path to the FS root.
func fsRootAndFSRelPathForPath(path string) (string, string, error) {
	absPath, err := normalpath.NormalizeAndAbsolute(path)
	if err != nil {
		return "", "", err
	}
	// Split the absolute path into components to get the FS root
	absPathComponents := normalpath.Components(absPath)
	fsRoot := absPathComponents[0]
	fsRelPath, err := normalpath.Rel(fsRoot, absPath)
	if err != nil {
		return "", "", err
	}
	return fsRoot, fsRelPath, nil
}

// We attempt to fix up paths we get back to better printing to the user.
// Without this, we'll get things like "stat: Users/foo/path/to/input: does not exist"
// based on our usage of osRootBucket and absProtoFileDirPath above. While we won't
// break any contracts printing these out, this is confusing to the user, so this is
// our attempt to fix that.
//
// This is going to take away other intermediate errors unfortunately.
func attemptToFixOSRootBucketPathErrors(fsRoot string, err error) error {
	var pathError *fs.PathError
	if errors.As(err, &pathError) {
		pwd, err := osext.Getwd()
		if err != nil {
			return err
		}
		pwd = normalpath.Normalize(pwd)
		if normalpath.EqualsOrContainsPath(pwd, normalpath.Join(fsRoot, pathError.Path), normalpath.Absolute) {
			relPath, err := normalpath.Rel(pwd, normalpath.Join(fsRoot, pathError.Path))
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

func validatePaths(
	inputSubDirPath string,
	targetPaths []string,
	targetExcludePaths []string,
) error {
	if _, err := normalpath.NormalizeAndValidate(inputSubDirPath); err != nil {
		return err
	}
	if _, err := slicesext.MapError(
		targetPaths,
		normalpath.NormalizeAndValidate,
	); err != nil {
		return err
	}
	if _, err := slicesext.MapError(
		targetPaths,
		normalpath.NormalizeAndValidate,
	); err != nil {
		return err
	}
	return nil
}

func mapTargetPathsAndTargetExcludePathsForArchiveAndGitRefs(
	inputSubDirPath string,
	targetPaths []string,
	targetExcludePaths []string,
) ([]string, []string) {
	// No need to remap
	if inputSubDirPath == "." {
		return targetPaths, targetExcludePaths
	}
	return slicesext.Map(
			targetPaths,
			func(targetPath string) string {
				return normalpath.Join(inputSubDirPath, targetPath)
			},
		),
		slicesext.Map(
			targetExcludePaths,
			func(targetExcludePath string) string {
				return normalpath.Join(inputSubDirPath, targetExcludePath)
			},
		)
}

type getFileOptions struct {
	keepFileCompression bool
}

func newGetFileOptions() *getFileOptions {
	return &getFileOptions{}
}

type getReadBucketCloserOptions struct {
	terminateFunc      buftarget.TerminateFunc
	copyToInMemory     bool
	targetPaths        []string
	targetExcludePaths []string
}

func newGetReadBucketCloserOptions() *getReadBucketCloserOptions {
	return &getReadBucketCloserOptions{}
}

type getReadWriteBucketOptions struct {
	terminateFunc      buftarget.TerminateFunc
	targetPaths        []string
	targetExcludePaths []string
}

func newGetReadWriteBucketOptions() *getReadWriteBucketOptions {
	return &getReadWriteBucketOptions{}
}

type getModuleOptions struct{}
