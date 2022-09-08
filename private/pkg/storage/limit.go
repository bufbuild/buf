// Copyright 2020-2022 Buf Technologies, Inc.
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

package storage

import (
	"context"
	"fmt"

	"go.uber.org/atomic"
)

// LimitWriteBucket returns a [WriteBucket] that writes to [writeBucket]
// but stops with an error after [limit] bytes are are written.
//
// The error can be checked using [IsWriteLimitReached].
func LimitWriteBucket(writeBucket WriteBucket, limit int64) WriteBucket {
	return newLimitedWriter(writeBucket, limit)
}

type limitedWriteBucket struct {
	WriteBucket
	currentSize *atomic.Int64
	limit       int64
}

func newLimitedWriter(bucket WriteBucket, limit int64) *limitedWriteBucket {
	return &limitedWriteBucket{
		WriteBucket: bucket,
		currentSize: atomic.NewInt64(0),
		limit:       limit,
	}
}

func (w *limitedWriteBucket) Put(ctx context.Context, path string) (WriteObjectCloser, error) {
	writeObjectCloser, err := w.WriteBucket.Put(ctx, path)
	if err != nil {
		return nil, err
	}
	return newLimitedWriteObjectCloser(writeObjectCloser, w.currentSize, w.limit), nil
}

type limitedWriteObjectCloser struct {
	WriteObjectCloser

	bucketSize *atomic.Int64
	limit      int64
}

func newLimitedWriteObjectCloser(
	writeObjectCloser WriteObjectCloser,
	bucketSize *atomic.Int64,
	limit int64,
) *limitedWriteObjectCloser {
	return &limitedWriteObjectCloser{
		WriteObjectCloser: writeObjectCloser,
		bucketSize:        bucketSize,
		limit:             limit,
	}
}

func (o *limitedWriteObjectCloser) Write(p []byte) (n int, err error) {
	newSize := o.bucketSize.Add(int64(len(p)))
	if newSize > o.limit {
		return 0, fmt.Errorf("limit writer: write limit reached: limit: %d, exceeded by: %d", o.limit, newSize-o.limit)
	}
	return o.WriteObjectCloser.Write(p)
}
