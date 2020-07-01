// Copyright 2020 Buf Technologies, Inc.
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

package storagemem

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/storage"
)

type readBucketBuilder struct {
	pathToData         map[string][]byte
	pathToExternalPath map[string]string
	lock               sync.Mutex
}

func newReadBucketBuilder() *readBucketBuilder {
	return &readBucketBuilder{
		pathToData:         make(map[string][]byte),
		pathToExternalPath: make(map[string]string),
	}
}

func (b *readBucketBuilder) Put(ctx context.Context, path string, size uint32) (storage.WriteObjectCloser, error) {
	path, err := normalpath.NormalizeAndValidate(path)
	if err != nil {
		return nil, err
	}
	if path == "." {
		return nil, errors.New("cannot put root")
	}
	return newWriteObjectCloser(b, path, size), nil
}

func (*readBucketBuilder) SetExternalPathSupported() bool {
	return true
}

func (b *readBucketBuilder) ToReadBucket(options ...ReadBucketOption) (storage.ReadBucket, error) {
	return newReadBucket(b.pathToData, append(options, withPathToExternalPath(b.pathToExternalPath))...)
}

type writeObjectCloser struct {
	readBucketBuilder    *readBucketBuilder
	path                 string
	size                 uint32
	buffer               *bytes.Buffer
	explicitExternalPath string
	written              int
	closed               bool
}

func newWriteObjectCloser(
	readBucketBuilder *readBucketBuilder,
	path string,
	size uint32,
) *writeObjectCloser {
	return &writeObjectCloser{
		readBucketBuilder: readBucketBuilder,
		path:              path,
		size:              size,
		buffer:            bytes.NewBuffer(nil),
	}
}

func (w *writeObjectCloser) Write(p []byte) (int, error) {
	if w.closed {
		return 0, storage.ErrClosed
	}
	if uint32(w.written+len(p)) > w.size {
		return 0, io.EOF
	}
	n, err := w.buffer.Write(p)
	w.written += n
	return n, err
}

func (w *writeObjectCloser) SetExternalPath(externalPath string) error {
	if w.explicitExternalPath != "" {
		// just to make sure
		return fmt.Errorf("external path already set: %q", w.explicitExternalPath)
	}
	w.explicitExternalPath = externalPath
	return nil
}

func (w *writeObjectCloser) Close() error {
	if w.closed {
		return storage.ErrClosed
	}
	w.closed = true
	if uint32(w.written) != w.size {
		return storage.ErrIncompleteWrite
	}
	// overwrites anything existing
	// this is the same behavior as storageos
	w.readBucketBuilder.lock.Lock()
	w.readBucketBuilder.pathToData[w.path] = w.buffer.Bytes()
	if w.explicitExternalPath != "" {
		w.readBucketBuilder.pathToExternalPath[w.path] = w.explicitExternalPath
	} else {
		delete(w.readBucketBuilder.pathToExternalPath, w.path)
	}
	w.readBucketBuilder.lock.Unlock()
	return nil
}

func withPathToExternalPath(pathToExternalPath map[string]string) ReadBucketOption {
	return func(readBucketOptions *readBucketOptions) {
		readBucketOptions.pathToExternalPath = pathToExternalPath
	}
}
