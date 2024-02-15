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
) (retReadBucketCloser ReadBucketCloser, retErr error) {
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
			getReadBucketCloserOptions.terminateFunc,
		)
	case DirRef:
		readWriteBucket, err := r.getDirBucket(
			ctx,
			container,
			t,
			getReadBucketCloserOptions.terminateFunc,
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
			getReadBucketCloserOptions.terminateFunc,
		)
	case ProtoFileRef:
		return r.getProtoFileBucket(
			ctx,
			container,
			t,
			// getReadBucketCloserOptions.terminateFunc,
			getReadBucketCloserOptions.protoFileTerminateFunc,
		)
	default:
		return nil, fmt.Errorf("unknown BucketRef type: %T", bucketRef)
	}
}

func (r *reader) GetReadWriteBucket(
	ctx context.Context,
	container app.EnvStdinContainer,
	dirRef DirRef,
	options ...GetReadWriteBucketOption,
) (ReadWriteBucket, error) {
	getReadWriteBucketOptions := newGetReadWriteBucketOptions()
	for _, option := range options {
		option(getReadWriteBucketOptions)
	}
	return r.getDirBucket(
		ctx,
		container,
		dirRef,
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
	terminateFunc buftarget.TerminateFunc,
) (_ ReadBucketCloser, retErr error) {
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
			storagearchive.UntarWithStripComponentCount(
				archiveRef.StripComponents(),
			),
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
			storagearchive.UnzipWithStripComponentCount(
				archiveRef.StripComponents(),
			),
		); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown ArchiveType: %v", archiveType)
	}
	return getReadBucketCloserForBucket(ctx, r.logger, storage.NopReadBucketCloser(readWriteBucket), archiveRef.SubDirPath(), terminateFunc)
}

func (r *reader) getDirBucket(
	ctx context.Context,
	container app.EnvStdinContainer,
	dirRef DirRef,
	terminateFunc buftarget.TerminateFunc,
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
	terminateFunc buftarget.TerminateFunc,
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
	)
}

func (r *reader) getGitBucket(
	ctx context.Context,
	container app.EnvStdinContainer,
	gitRef GitRef,
	terminateFunc buftarget.TerminateFunc,
) (_ ReadBucketCloser, retErr error) {
	if !r.gitEnabled {
		return nil, NewReadGitDisabledError()
	}
	if r.gitCloner == nil {
		return nil, errors.New("git cloner is nil")
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
	return getReadBucketCloserForBucket(ctx, r.logger, storage.NopReadBucketCloser(readWriteBucket), gitRef.SubDirPath(), terminateFunc)
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
		bufmodule.DigestTypeB4,
		// TODO: Switch back when b5 is ready.
		//bufmodule.DigestTypeB5,
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
	terminateFunc buftarget.TerminateFunc,
) (ReadBucketCloser, error) {
	// TODO(doria): delete later, keeping some notes as we work.
	// The point of doing this is to remap the bucket based on a controlling workspace if found
	// and then also remapping the `SubDirPath` against this.
	bucketTargeting, err := buftarget.NewBucketTargeting(
		ctx,
		logger,
		inputBucket,
		inputSubDirPath,
		nil, // TODO(doria): we should plumb paths down here
		nil, // TODO(doria): we should plumb paths down here
		terminateFunc,
	)
	if err != nil {
		return nil, err
	}
	if bucketTargeting.ControllingWorkspacePath() != "." {
		inputBucket = storage.MapReadBucketCloser(
			inputBucket,
			storage.MapOnPrefix(bucketTargeting.ControllingWorkspacePath()),
		)
	}
	logger.Debug(
		"buffetch creating new bucket",
		zap.String("controllingWorkspacePath", bucketTargeting.ControllingWorkspacePath()),
		zap.Strings("targetPaths", bucketTargeting.TargetPaths()),
	)
	return newReadBucketCloser(
		inputBucket,
		bucketTargeting,
	)
}

// Use for directory-based buckets.
func getReadWriteBucketForOS(
	ctx context.Context,
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	inputDirPath string,
	terminateFunc buftarget.TerminateFunc,
) (ReadWriteBucket, error) {
	fsRoot, inputSubDirPath, err := fsRootAndFSRelPathForPath(inputDirPath)
	if err != nil {
		return nil, err
	}
	osRootBucket, err := storageosProvider.NewReadWriteBucket(
		fsRoot,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return nil, err
	}
	osRootBucketTargeting, err := buftarget.NewBucketTargeting(
		ctx,
		logger,
		osRootBucket,
		inputSubDirPath,
		nil, // TODO(doria): we should plumb paths down here
		nil, // TODO(doria): we should plumb paths down here
		terminateFunc,
	)
	if err != nil {
		return nil, attemptToFixOSRootBucketPathErrors(fsRoot, err)
	}
	// TODO(doria): I'd like to completely kill and refactor this if possible.
	// We now know where the workspace is relative to the FS root.
	// If the input path is provided as an absolute path, we create a new bucket for the
	// controlling workspace using an absolute path.
	// Otherwise we current working directory (pwd) and create the bucket using a relative
	// path to that.
	//
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
	// Make bucket on: FS root + returnMapPath (since absolute)
	var bucketPath string
	var inputPath string
	if filepath.IsAbs(normalpath.Unnormalize(inputDirPath)) {
		var err error
		bucketPath = normalpath.Join(fsRoot, osRootBucketTargeting.ControllingWorkspacePath())
		inputPath, err = normalpath.Rel(bucketPath, normalpath.Join(fsRoot, inputSubDirPath))
		if err != nil {
			return nil, err
		}
	} else {
		pwd, err := osext.Getwd()
		if err != nil {
			return nil, err
		}
		_, pwdFSRelPath, err := fsRootAndFSRelPathForPath(pwd)
		if err != nil {
			return nil, err
		}
		bucketPath, err = normalpath.Rel(pwdFSRelPath, osRootBucketTargeting.ControllingWorkspacePath())
		if err != nil {
			return nil, err
		}
		inputPath, err = normalpath.Rel(osRootBucketTargeting.ControllingWorkspacePath(), inputSubDirPath)
		if err != nil {
			return nil, err
		}
	}
	// Now that we've mapped the workspace against the FS root, we recreate the bucket with
	// at the sub dir path.
	// First we get the absolute path of the controlling workspace, which based on the OS root
	// bucket targeting is the FS root joined with the controlling workspace path.
	// And we use it to make a bucket.
	bucket, err := storageosProvider.NewReadWriteBucket(
		bucketPath,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return nil, err
	}
	bucketTargeting, err := buftarget.NewBucketTargeting(
		ctx,
		logger,
		bucket,
		inputPath,
		nil, // TODO(doria): we should plumb paths down here
		nil, // TODO(doria): we should plumb paths down here
		terminateFunc,
	)
	if err != nil {
		return nil, err
	}
	return newReadWriteBucket(
		bucket,
		bucketTargeting,
	)
}

// Use for ProtoFileRefs.
func getReadBucketCloserForOSProtoFile(
	ctx context.Context,
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	protoFilePath string,
	terminateFunc buftarget.TerminateFunc,
) (ReadBucketCloser, error) {
	protoFileDir := normalpath.Dir(protoFilePath)
	readWriteBucket, err := getReadWriteBucketForOS(
		ctx,
		logger,
		storageosProvider,
		protoFileDir,
		terminateFunc,
	)
	if err != nil {
		return nil, err
	}
	return newReadBucketCloserForReadWriteBucket(readWriteBucket), nil
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

type getFileOptions struct {
	keepFileCompression bool
}

func newGetFileOptions() *getFileOptions {
	return &getFileOptions{}
}

type getReadBucketCloserOptions struct {
	terminateFunc          buftarget.TerminateFunc
	protoFileTerminateFunc buftarget.TerminateFunc
	copyToInMemory         bool
}

func newGetReadBucketCloserOptions() *getReadBucketCloserOptions {
	return &getReadBucketCloserOptions{}
}

type getReadWriteBucketOptions struct {
	terminateFunc buftarget.TerminateFunc
}

func newGetReadWriteBucketOptions() *getReadWriteBucketOptions {
	return &getReadWriteBucketOptions{}
}

type getModuleOptions struct{}
