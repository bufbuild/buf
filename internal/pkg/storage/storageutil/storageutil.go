// Package storageutil implements storage utilities.
package storageutil

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"io"
	"io/ioutil"
	"runtime"
	"sync"

	"github.com/bufbuild/buf/internal/pkg/errs"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storagepath"
)

// Copy copies the bucket at from to the bucket at to for the given prefix.
//
// Copies done concurrently.
//
// Paths from the source bucket will be transformed before being added to the destination bucket.
// Returns the number of files copied.
func Copy(
	ctx context.Context,
	from storage.ReadBucket,
	to storage.Bucket,
	prefix string,
	options ...storagepath.TransformerOption,
) (int, error) {
	transformer := storagepath.NewTransformer(options...)
	semaphoreC := make(chan struct{}, runtime.NumCPU())
	var retErr error
	var count int
	var wg sync.WaitGroup
	var lock sync.Mutex
	if walkErr := from.Walk(
		ctx,
		prefix,
		func(path string) error {
			newPath, ok := transformer.Transform(path)
			if !ok {
				return nil
			}
			wg.Add(1)
			semaphoreC <- struct{}{}
			go func() {
				err := copyPath(ctx, from, to, path, newPath)
				lock.Lock()
				if err != nil {
					retErr = errs.Append(retErr, err)
				} else {
					count++
				}
				lock.Unlock()
				<-semaphoreC
				wg.Done()
			}()
			return nil
		},
	); walkErr != nil {
		return count, walkErr
	}
	wg.Wait()
	return count, retErr
}

// CopyPaths copies the paths from the bucket at from to the bucket at to, if they exist.
//
// Paths will be normalized within this function.
// Copies done concurrently.
//
// Returns the number of files copied.
func CopyPaths(
	ctx context.Context,
	from storage.ReadBucket,
	to storage.Bucket,
	paths ...string,
) (int, error) {
	semaphoreC := make(chan struct{}, runtime.NumCPU())
	var retErr error
	var count int
	var wg sync.WaitGroup
	var lock sync.Mutex
	for _, path := range paths {
		path := path
		wg.Add(1)
		semaphoreC <- struct{}{}
		go func() {
			err := copyPath(ctx, from, to, path, path)
			lock.Lock()
			if err != nil {
				if !storage.IsNotExist(err) {
					retErr = errs.Append(retErr, err)
				}
			} else {
				count++
			}
			lock.Unlock()
			<-semaphoreC
			wg.Done()
		}()
	}
	wg.Wait()
	return count, retErr
}

// returns storage.ErrNotExist if fromPath does not exist
func copyPath(
	ctx context.Context,
	from storage.ReadBucket,
	to storage.Bucket,
	fromPath string,
	toPath string,
) error {
	readObject, err := from.Get(ctx, fromPath)
	if err != nil {
		return err
	}
	writeObject, err := to.Put(ctx, toPath, readObject.Size())
	if err != nil {
		return errs.Append(err, readObject.Close())
	}
	_, err = io.Copy(writeObject, readObject)
	return errs.Append(err, errs.Append(writeObject.Close(), readObject.Close()))
}

// Untar untars the given tar archive from the reader into the bucket.
//
// Only regular files are added to the bucket.
//
// Paths from the tar archive will be transformed before adding to the bucket.
func Untar(
	ctx context.Context,
	reader io.Reader,
	bucket storage.Bucket,
	options ...storagepath.TransformerOption,
) error {
	return untar(ctx, reader, bucket, false, options...)
}

// Untargz untars the given targz archive from the reader into the bucket.
//
// Only regular files are added to the bucket.
//
// Paths from the targz archive will be transformed before adding to the bucket.
func Untargz(
	ctx context.Context,
	reader io.Reader,
	bucket storage.Bucket,
	options ...storagepath.TransformerOption,
) error {
	return untar(ctx, reader, bucket, true, options...)
}

func untar(
	ctx context.Context,
	reader io.Reader,
	bucket storage.Bucket,
	gzipped bool,
	options ...storagepath.TransformerOption,
) error {
	transformer := storagepath.NewTransformer(options...)
	var err error
	if gzipped {
		reader, err = gzip.NewReader(reader)
		if err != nil {
			return err
		}
	}
	tarReader := tar.NewReader(reader)
	fileCount := 0
	for tarHeader, err := tarReader.Next(); err != io.EOF; tarHeader, err = tarReader.Next() {
		if err != nil {
			return err
		}
		fileCount++
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if err == context.DeadlineExceeded {
				return errs.NewDeadlineExceededf("timed out after untarring %d files", fileCount)
			}
			return err
		default:
		}
		path, err := storagepath.NormalizeAndValidate(tarHeader.Name)
		if err != nil {
			return err
		}
		if path == "." {
			continue
		}
		path, ok := transformer.Transform(path)
		if !ok {
			continue
		}
		if tarHeader.FileInfo().Mode().IsRegular() {
			writeObject, err := bucket.Put(ctx, path, uint32(tarHeader.Size))
			if err != nil {
				return err
			}
			_, writeErr := io.Copy(writeObject, tarReader)
			if err := writeObject.Close(); err != nil {
				return err
			}
			if writeErr != nil {
				return writeErr
			}
		}
	}
	return nil
}

// Tar tars the given bucket to the writer.
//
// Only regular files are added to the writer.
// All files are written as 0644.
//
// Paths from the bucket will be transformed before adding to the writer.
func Tar(
	ctx context.Context,
	writer io.Writer,
	bucket storage.Bucket,
	prefix string,
	options ...storagepath.TransformerOption,
) error {
	return doTar(ctx, writer, bucket, prefix, false, options...)
}

// Targz tars and gzips the given bucket to the writer.
//
// Only regular files are added to the writer.
// All files are written as 0644.
//
// Paths from the bucket will be transformed before adding to the writer.
func Targz(
	ctx context.Context,
	writer io.Writer,
	bucket storage.Bucket,
	prefix string,
	options ...storagepath.TransformerOption,
) error {
	return doTar(ctx, writer, bucket, prefix, true, options...)
}

func doTar(
	ctx context.Context,
	writer io.Writer,
	bucket storage.Bucket,
	prefix string,
	gzipped bool,
	options ...storagepath.TransformerOption,
) (retErr error) {
	transformer := storagepath.NewTransformer(options...)
	if gzipped {
		gzipWriter := gzip.NewWriter(writer)
		defer func() {
			retErr = errs.Append(retErr, gzipWriter.Close())
		}()
		writer = gzipWriter
	}
	tarWriter := tar.NewWriter(writer)
	defer func() {
		retErr = errs.Append(retErr, tarWriter.Close())
	}()
	return bucket.Walk(
		ctx,
		prefix,
		func(path string) error {
			newPath, ok := transformer.Transform(path)
			if !ok {
				return nil
			}
			readObject, err := bucket.Get(ctx, path)
			if err != nil {
				return err
			}
			if err := tarWriter.WriteHeader(
				&tar.Header{
					Typeflag: tar.TypeReg,
					Name:     newPath,
					Size:     int64(readObject.Size()),
					// If we ever use this outside of testing, we will want to do something about this
					Mode: 0644,
				},
			); err != nil {
				return errs.Append(err, readObject.Close())
			}
			_, err = io.Copy(tarWriter, readObject)
			return errs.Append(err, readObject.Close())
		},
	)
}

// ReadPath is analogous to ioutil.ReadFile.
//
// Returns an error that fufills storage.IsNotExist if the path does not exist.
func ReadPath(ctx context.Context, bucket storage.ReadBucket, path string) (_ []byte, retErr error) {
	readObject, err := bucket.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := readObject.Close(); err != nil && retErr == nil {
			retErr = err
		}
	}()
	return ioutil.ReadAll(readObject)
}
