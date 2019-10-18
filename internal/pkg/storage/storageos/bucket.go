package storageos

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"

	"github.com/bufbuild/buf/internal/pkg/errs"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storagepath"
)

type bucket struct {
	rootPath string
	closed   bool
}

func newBucket(rootPath string) (*bucket, error) {
	rootPath = storagepath.Unnormalize(rootPath)
	fileInfo, err := os.Stat(rootPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, storage.NewErrNotExist(rootPath)
		}
		return nil, err
	}
	if !fileInfo.IsDir() {
		return nil, newErrNotDir(rootPath)
	}
	// allow anything with OS buckets including absolute paths
	// and jumping context
	rootPath = storagepath.Normalize(rootPath)
	return &bucket{
		rootPath: rootPath,
	}, nil
}

func (b *bucket) Type() string {
	return BucketType
}

func (b *bucket) Get(ctx context.Context, path string) (storage.ReadObject, error) {
	path, err := storagepath.NormalizeAndValidate(path)
	if err != nil {
		return nil, err
	}
	if path == "." {
		return nil, errors.New("cannot get root")
	}
	path = storagepath.Unnormalize(storagepath.Join(b.rootPath, path))
	if b.closed {
		return nil, storage.ErrClosed
	}
	// this is potentially introducing two calls to a file
	// instead of one, ie we do both Stat and Open as opposed
	// to just Open
	// we do this to make sure we are only reading regular files
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, storage.NewErrNotExist(path)
		}
		return nil, err
	}
	if !fileInfo.Mode().IsRegular() {
		// making this a user error as any access means this was generally requested
		// by the user, since we only call the function for Walk on regular files
		return nil, errs.NewUserErrorf("%q is not a regular file", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	if fileInfo.Size() > int64(math.MaxUint32) {
		return nil, fmt.Errorf("file too large: %d", fileInfo.Size())
	}
	return newReadObject(file, uint32(fileInfo.Size())), nil
}

func (b *bucket) Stat(ctx context.Context, path string) (storage.ObjectInfo, error) {
	path, err := storagepath.NormalizeAndValidate(path)
	if err != nil {
		return storage.ObjectInfo{}, err
	}
	if path == "." {
		return storage.ObjectInfo{}, errors.New("cannot check root")
	}
	path = storagepath.Unnormalize(storagepath.Join(b.rootPath, path))
	if b.closed {
		return storage.ObjectInfo{}, storage.ErrClosed
	}
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return storage.ObjectInfo{}, storage.NewErrNotExist(path)
		}
		return storage.ObjectInfo{}, err
	}
	if !fileInfo.Mode().IsRegular() {
		// should this be an error, or just return false?
		// probably an error as we should not be trying to access
		// files that are not regular, however in walk we just
		// filter non-regular files
		// making this a user error as any access means this was generally requested
		// by the user, since we only call the function for Walk on regular files
		return storage.ObjectInfo{}, errs.NewUserErrorf("%q is not a regular file", path)
	}
	return storage.ObjectInfo{
		Size: uint32(fileInfo.Size()),
	}, nil
}

func (b *bucket) Walk(ctx context.Context, prefix string, f func(string) error) error {
	prefix, err := storagepath.NormalizeAndValidate(prefix)
	if err != nil {
		return err
	}
	prefix = storagepath.Unnormalize(storagepath.Join(b.rootPath, prefix))
	if b.closed {
		return storage.ErrClosed
	}
	fileCount := 0
	// Walk does not follow symlinks
	return filepath.Walk(
		prefix,
		func(path string, fileInfo os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			fileCount++
			select {
			case <-ctx.Done():
				err := ctx.Err()
				if err == context.DeadlineExceeded {
					return errs.NewUserErrorf("timed out after walking %d files", fileCount)
				}
				return err
			default:
			}
			if fileInfo.Mode().IsRegular() {
				rel, err := storagepath.Rel(b.rootPath, storagepath.Normalize(path))
				if err != nil {
					return err
				}
				// just in case
				rel, err = storagepath.NormalizeAndValidate(rel)
				if err != nil {
					return err
				}
				if err := f(rel); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

func (b *bucket) Put(ctx context.Context, path string, size uint32) (storage.WriteObject, error) {
	path, err := storagepath.NormalizeAndValidate(path)
	if err != nil {
		return nil, err
	}
	if path == "." {
		return nil, errors.New("cannot put root")
	}
	path = storagepath.Unnormalize(storagepath.Join(b.rootPath, path))
	if b.closed {
		return nil, storage.ErrClosed
	}
	dir := storagepath.Unnormalize(storagepath.Dir(storagepath.Normalize(path)))
	fileInfo, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	} else if !fileInfo.IsDir() {
		return nil, newErrNotDir(dir)
	}
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	return newWriteObject(file, size), nil
}

func (b *bucket) Close() error {
	if b.closed {
		return storage.ErrClosed
	}
	b.closed = true
	return nil
}

type readObject struct {
	file *os.File
	size uint32
}

func newReadObject(file *os.File, size uint32) *readObject {
	return &readObject{
		file: file,
		size: size,
	}
}

func (r *readObject) Read(p []byte) (int, error) {
	return r.file.Read(p)
}

func (r *readObject) Close() error {
	return r.file.Close()
}

func (r *readObject) Size() uint32 {
	return r.size
}

type writeObject struct {
	file    *os.File
	size    uint32
	written int
}

func newWriteObject(file *os.File, size uint32) *writeObject {
	return &writeObject{
		file: file,
		size: size,
	}
}

func (r *writeObject) Write(p []byte) (int, error) {
	if uint32(r.written+len(p)) > r.size {
		return 0, io.EOF
	}
	n, err := r.file.Write(p)
	r.written += n
	return n, err
}

func (r *writeObject) Close() error {
	err := r.file.Close()
	if uint32(r.written) != r.size {
		return storage.ErrIncompleteWrite
	}
	return err
}

func (r *writeObject) Size() uint32 {
	return r.size
}

// newErrNotDir returns a new PathError for a path not being a directory.
func newErrNotDir(path string) *storage.PathError {
	return &storage.PathError{
		Path: path,
		Err:  errNotDir,
	}
}
