package storagemem

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"

	"github.com/bufbuild/buf/internal/pkg/bytepool"
	"github.com/bufbuild/buf/internal/pkg/errs"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storagepath"
	"go.uber.org/multierr"
)

type bucket struct {
	segList            *bytepool.SegList
	pathToBytesWrapper map[string]*bytesWrapper
	closed             bool
	lock               sync.RWMutex
}

func newBucket(segList *bytepool.SegList) *bucket {
	return &bucket{
		segList:            segList,
		pathToBytesWrapper: make(map[string]*bytesWrapper),
	}
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
	b.lock.RLock()
	defer b.lock.RUnlock()
	if b.closed {
		return nil, storage.ErrClosed
	}
	bytesWrapper, ok := b.pathToBytesWrapper[path]
	if !ok {
		return nil, storage.NewErrNotExist(path)
	}
	size, err := bytesWrapper.Len()
	if err != nil {
		return nil, err
	}
	return newReadObject(bytesWrapper, uint32(size)), nil
}

func (b *bucket) Stat(ctx context.Context, path string) (storage.ObjectInfo, error) {
	path, err := storagepath.NormalizeAndValidate(path)
	if err != nil {
		return storage.ObjectInfo{}, err
	}
	if path == "." {
		return storage.ObjectInfo{}, errors.New("cannot check root")
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	if b.closed {
		return storage.ObjectInfo{}, storage.ErrClosed
	}
	bytesWrapper, ok := b.pathToBytesWrapper[path]
	if !ok {
		return storage.ObjectInfo{}, storage.NewErrNotExist(path)
	}
	size, err := bytesWrapper.Len()
	if err != nil {
		return storage.ObjectInfo{}, err
	}
	return storage.ObjectInfo{
		Size: uint32(size),
	}, nil
}

func (b *bucket) Walk(ctx context.Context, prefix string, f func(string) error) error {
	prefix, err := storagepath.NormalizeAndValidate(prefix)
	if err != nil {
		return err
	}
	// without this, "internal/buf/proto" would call f for "internal/buf/protocompile"
	if prefix != "." {
		prefix = prefix + "/"
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	if b.closed {
		return storage.ErrClosed
	}
	fileCount := 0
	for path := range b.pathToBytesWrapper {
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
		if prefix == "." || strings.HasPrefix(path, prefix) {
			// only normalized and validated paths can be put into the map
			if err := f(path); err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *bucket) Put(ctx context.Context, path string, size uint32) (storage.WriteObject, error) {
	path, err := storagepath.NormalizeAndValidate(path)
	if err != nil {
		return nil, err
	}
	if path == "." {
		return nil, errors.New("cannot put root")
	}
	b.lock.Lock()
	defer b.lock.Unlock()
	if b.closed {
		return nil, storage.ErrClosed
	}
	bytesWrapper, ok := b.pathToBytesWrapper[path]
	if ok {
		// this has a recycled marker so that if we have outstanding
		// readers or writers, they will fail
		if err := bytesWrapper.Recycle(); err != nil {
			return nil, err
		}
		// just in case
		delete(b.pathToBytesWrapper, path)
	}
	bytesWrapper = newBytesWrapper(b.segList.Get(size))
	b.pathToBytesWrapper[path] = bytesWrapper
	return newWriteObject(bytesWrapper, size), nil
}

func (b *bucket) Close() error {
	b.lock.Lock()
	defer b.lock.Unlock()
	if b.closed {
		return storage.ErrClosed
	}
	var err error
	for _, bytesWrapper := range b.pathToBytesWrapper {
		// this has a recycled marker so that if we have outstanding
		// readers or writers, they will fail
		err = multierr.Append(err, bytesWrapper.Recycle())
	}
	// just in case we don't protect against close somewhere
	b.pathToBytesWrapper = make(map[string]*bytesWrapper)
	b.closed = true
	return err
}

type readObject struct {
	bytesWrapper *bytesWrapper
	size         uint32
	read         int
	closed       bool
	lock         sync.Mutex
}

func newReadObject(bytesWrapper *bytesWrapper, size uint32) *readObject {
	return &readObject{
		bytesWrapper: bytesWrapper,
		size:         size,
	}
}

func (r *readObject) Read(p []byte) (int, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.closed {
		return 0, storage.ErrClosed
	}
	if uint32(r.read) >= r.size {
		return 0, io.EOF
	}
	max := r.size - uint32(r.read)
	if max < uint32(len(p)) {
		p = p[:max]
	}
	n, err := r.bytesWrapper.CopyTo(p, r.read)
	r.read += n
	if uint32(r.read) >= r.size {
		err = io.EOF
	}
	return n, err
}

func (r *readObject) Close() error {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.closed {
		return storage.ErrClosed
	}
	r.closed = true
	return nil
}

func (r *readObject) Size() uint32 {
	return r.size
}

type writeObject struct {
	bytesWrapper *bytesWrapper
	size         uint32
	written      int
	closed       bool
	lock         sync.Mutex
}

func newWriteObject(bytesWrapper *bytesWrapper, size uint32) *writeObject {
	return &writeObject{
		bytesWrapper: bytesWrapper,
		size:         size,
	}
}

func (r *writeObject) Write(p []byte) (int, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.closed {
		return 0, storage.ErrClosed
	}
	if uint32(r.written+len(p)) > r.size {
		return 0, io.EOF
	}
	n, err := r.bytesWrapper.CopyFrom(p, r.written)
	r.written += n
	return n, err
}

func (r *writeObject) Close() error {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.closed {
		return storage.ErrClosed
	}
	r.closed = true
	if uint32(r.written) != r.size {
		return storage.ErrIncompleteWrite
	}
	return nil
}

func (r *writeObject) Size() uint32 {
	return r.size
}

type bytesWrapper struct {
	bytes *bytepool.Bytes
	// protect against outstanding readers or writers
	// if we overwrite a file
	recycled bool
	lock     sync.RWMutex
}

func newBytesWrapper(bytes *bytepool.Bytes) *bytesWrapper {
	return &bytesWrapper{
		bytes: bytes,
	}
}

func (b *bytesWrapper) CopyFrom(from []byte, offset int) (int, error) {
	b.lock.Lock()
	defer b.lock.Unlock()
	if b.recycled {
		return 0, storage.ErrClosed
	}
	// can happen if size was 0 to segList.Get
	if b.bytes == nil {
		return 0, io.EOF
	}
	return b.bytes.CopyFrom(from, offset)
}

func (b *bytesWrapper) CopyTo(to []byte, offset int) (int, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()
	if b.recycled {
		return 0, storage.ErrClosed
	}
	// can happen if size was 0 to segList.Get
	if b.bytes == nil {
		return 0, io.EOF
	}
	return b.bytes.CopyTo(to, offset)
}

func (b *bytesWrapper) Len() (int, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()
	if b.recycled {
		return 0, storage.ErrClosed
	}
	// can happen if size was 0 to segList.Get
	if b.bytes == nil {
		return 0, nil
	}
	return b.bytes.Len(), nil
}

func (b *bytesWrapper) Recycle() error {
	b.lock.Lock()
	defer b.lock.Unlock()
	if b.recycled {
		return storage.ErrClosed
	}
	if b.bytes != nil {
		b.bytes.Recycle()
	}
	b.recycled = true
	return nil
}
