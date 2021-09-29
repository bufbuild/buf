package storagemem

import (
	"bytes"
	"fmt"

	"github.com/bufbuild/buf/private/pkg/storage"
)

type writeObjectCloser struct {
	readBucketBuilder *readBucketBuilder
	path              string
	externalPath      string
	buffer            *bytes.Buffer
	closed            bool
}

func newWriteObjectCloser(
	readBucketBuilder *readBucketBuilder,
	path string,
) *writeObjectCloser {
	return &writeObjectCloser{
		readBucketBuilder: readBucketBuilder,
		path:              path,
		buffer:            bytes.NewBuffer(nil),
	}
}

func (w *writeObjectCloser) Write(p []byte) (int, error) {
	if w.closed {
		return 0, storage.ErrClosed
	}
	return w.buffer.Write(p)
}

func (w *writeObjectCloser) SetExternalPath(externalPath string) error {
	if w.externalPath != "" {
		return fmt.Errorf("external path already set: %q", w.externalPath)
	}
	w.externalPath = externalPath
	return nil
}

func (w *writeObjectCloser) Close() error {
	if w.closed {
		return storage.ErrClosed
	}
	w.closed = true
	// overwrites anything existing
	// this is the same behavior as storageos
	w.readBucketBuilder.lock.Lock()
	defer w.readBucketBuilder.lock.Unlock()
	w.readBucketBuilder.pathToImmutableObject[w.path] = newImmutableObject(
		w.path,
		w.externalPath,
		w.buffer.Bytes(),
	)
	return nil
}
