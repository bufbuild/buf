// Package storagegit implements git utilities.
//
// This uses https://github.com/src-d/go-git.
package storagegit

import (
	"context"
	"io"
	"math"
	"runtime"
	"sync"

	"github.com/bufbuild/buf/internal/pkg/errs"
	"github.com/bufbuild/buf/internal/pkg/logutil"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storagepath"
	"go.uber.org/zap"
	"gopkg.in/src-d/go-billy.v4"
	"gopkg.in/src-d/go-billy.v4/memfs"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/storage/memory"
)

// Clone clones the url into the bucket.
//
// This is roughly equivalent to git clone --branch gitBranch --single-branch --depth 1 gitUrl.
// Only regular files are added to the bucket.
//
// Branch is required.
//
// This really needs more testing and cleanup
// Only use for local CLI checking
func Clone(
	ctx context.Context,
	logger *zap.Logger,
	gitURL string,
	gitBranch string,
	bucket storage.Bucket,
	options ...storagepath.TransformerOption,
) error {
	defer logutil.Defer(logger, "git_clone")()

	if gitBranch == "" {
		// we detect this outside of this function so this is a system error
		return errs.NewInternal("gitBranch is empty")
	}
	filesystem := memfs.New()
	if _, err := git.CloneContext(
		ctx,
		memory.NewStorage(),
		filesystem,
		&git.CloneOptions{
			URL:           gitURL,
			ReferenceName: plumbing.NewBranchReferenceName(gitBranch),
			SingleBranch:  true,
			Depth:         1,
		},
	); err != nil {
		return err
	}
	return copyBillyFilesystemToBucket(ctx, logger, filesystem, bucket, options...)
}

func copyBillyFilesystemToBucket(
	ctx context.Context,
	logger *zap.Logger,
	filesystem billy.Filesystem,
	bucket storage.Bucket,
	options ...storagepath.TransformerOption,
) error {
	defer logutil.Defer(logger, "git_clone_copy")()

	transformer := storagepath.NewTransformer(options...)
	semaphoreC := make(chan struct{}, runtime.NumCPU())
	var retErr error
	var wg sync.WaitGroup
	var lock sync.Mutex
	if walkErr := walkBillyFilesystemDir(
		ctx,
		filesystem,
		func(regularFilePath string, regularFileSize uint32) error {
			if regularFilePath == "" || regularFilePath[0] != '/' {
				return errs.NewInternalf("invalid regularFilePath: %q", regularFilePath)
			}
			// just to make sure
			path, err := storagepath.NormalizeAndValidate(regularFilePath[1:])
			if err != nil {
				return err
			}
			path, ok := transformer.Transform(path)
			if !ok {
				return nil
			}
			wg.Add(1)
			semaphoreC <- struct{}{}
			go func() {
				err := copyBillyPath(ctx, filesystem, bucket, regularFilePath, regularFileSize, path)
				lock.Lock()
				retErr = errs.Append(retErr, err)
				lock.Unlock()
				<-semaphoreC
				wg.Done()
			}()
			return nil
		},
		"/",
	); walkErr != nil {
		return walkErr
	}
	wg.Wait()
	return retErr
}

func copyBillyPath(
	ctx context.Context,
	from billy.Filesystem,
	to storage.Bucket,
	fromPath string,
	fromSize uint32,
	toPath string,
) error {
	file, err := from.Open(fromPath)
	if err != nil {
		return err
	}
	writeObject, err := to.Put(ctx, toPath, fromSize)
	if err != nil {
		return errs.Append(err, file.Close())
	}
	_, err = io.Copy(writeObject, file)
	return errs.Append(err, errs.Append(writeObject.Close(), file.Close()))
}

func walkBillyFilesystemDir(
	ctx context.Context,
	filesystem billy.Filesystem,
	// regularFilePath will be the billy filesystem path
	f func(regularFilePath string, regularFileSize uint32) error,
	dirPath string,
) error {
	if dirPath == "" || dirPath[0] != '/' {
		return errs.NewInternalf("invalid dirPath: %q", dirPath)
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	fileInfos, err := filesystem.ReadDir(dirPath)
	if err != nil {
		return err
	}
	for _, fileInfo := range fileInfos {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		name := fileInfo.Name()
		if name == "" || name[0] == '/' {
			return errs.NewInternalf("invalid name: %q", name)
		}
		if fileInfo.Mode().IsRegular() {
			size := fileInfo.Size()
			if size > math.MaxUint32 {
				return errs.NewInternalf("size %d is greater than uint32", size)
			}
			// TODO: check to make sure normalization matches up with billy package
			if err := f(storagepath.Join(dirPath, name), uint32(size)); err != nil {
				return err
			}
		}
		if fileInfo.Mode().IsDir() {
			// TODO: check to make sure normalization matches up with billy package
			if err := walkBillyFilesystemDir(ctx, filesystem, f, storagepath.Join(dirPath, name)); err != nil {
				return err
			}
		}
	}
	return nil
}
