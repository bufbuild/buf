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

package storagemem

import (
	"bytes"
	"fmt"

	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem/internal"
)

type writeObjectCloser struct {
	bucket       *bucket
	path         string
	externalPath string
	localPath    string
	buffer       *bytes.Buffer
	closed       bool
}

func newWriteObjectCloser(
	bucket *bucket,
	path string,
) *writeObjectCloser {
	return &writeObjectCloser{
		bucket: bucket,
		path:   path,
		buffer: bytes.NewBuffer(nil),
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

func (w *writeObjectCloser) SetLocalPath(localPath string) error {
	if w.localPath != "" {
		return fmt.Errorf("local path already set: %q", w.localPath)
	}
	w.localPath = localPath
	return nil
}

func (w *writeObjectCloser) Close() error {
	if w.closed {
		return storage.ErrClosed
	}
	w.closed = true
	// overwrites anything existing
	// this is the same behavior as storageos
	w.bucket.lock.Lock()
	defer w.bucket.lock.Unlock()
	// Note that if there is an existing reader for an object of the same path,
	// that reader will continue to read the original file, but we accept this
	// as no less consistent than os mechanics.
	w.bucket.pathToImmutableObject[w.path] = internal.NewImmutableObject(
		w.path,
		w.externalPath,
		w.localPath,
		w.buffer.Bytes(),
	)
	return nil
}
