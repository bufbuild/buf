// Copyright 2020-2021 Buf Technologies, Inc.
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
	"io"

	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem/internal"
	"github.com/bufbuild/buf/private/pkg/storage/storageutil"
	"github.com/klauspost/compress/zstd"
)

type readObjectCloser struct {
	storageutil.ObjectInfo

	reader io.Reader
	closer func()
	closed bool
}

func newReadObjectCloser(bucket *bucket, immutableObject *internal.ImmutableObject) (*readObjectCloser, error) {
	var reader io.Reader = bytes.NewReader(immutableObject.Data())
	var closer func()
	if bucket.compression {
		decoder, err := zstd.NewReader(reader)
		if err != nil {
			return nil, err
		}
		reader = decoder
		closer = decoder.Close
	}
	return &readObjectCloser{
		ObjectInfo: immutableObject.ObjectInfo,
		reader:     reader,
		closer:     closer,
	}, nil
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
	if r.closer != nil {
		r.closer()
	}
	return nil
}
