package storagemem

import (
	"bytes"

	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem/internal"
	"github.com/bufbuild/buf/private/pkg/storage/storageutil"
)

type readObjectCloser struct {
	storageutil.ObjectInfo

	reader *bytes.Reader
	closed bool
}

func newReadObjectCloser(immutableObject *internal.ImmutableObject) *readObjectCloser {
	return &readObjectCloser{
		ObjectInfo: immutableObject.ObjectInfo,
		reader:     bytes.NewReader(immutableObject.Data()),
	}
}

func (r *readObjectCloser) Read(p []byte) (int, error) {
	if r.closed {
		return 0, storage.ErrClosed
	}
	return r.reader.Read(p)
}

func (r *readObjectCloser) Close() error {
	if r.closed {
		return storage.ErrClosed
	}
	r.closed = true
	return nil
}
