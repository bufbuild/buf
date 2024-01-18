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

package storage

import (
	"context"

	"go.uber.org/atomic"
)

// LimitWriteBucket returns a [WriteBucket] that writes to [writeBucket]
// but stops with an error after [limit] bytes are written.
//
// The error can be checked using [IsWriteLimitReached].
//
// A negative [limit] is same as 0 limit.
func LimitWriteBucket(writeBucket WriteBucket, limit int) WriteBucket {
	if limit < 0 {
		limit = 0
	}
	return newLimitedWriteBucket(writeBucket, int64(limit))
}

type limitedWriteBucket struct {
	WriteBucket
	currentSize *atomic.Int64
	limit       int64
}

func newLimitedWriteBucket(bucket WriteBucket, limit int64) *limitedWriteBucket {
	return &limitedWriteBucket{
		WriteBucket: bucket,
		currentSize: atomic.NewInt64(0),
		limit:       limit,
	}
}

func (w *limitedWriteBucket) Put(ctx context.Context, path string, opts ...PutOption) (WriteObjectCloser, error) {
	writeObjectCloser, err := w.WriteBucket.Put(ctx, path, opts...)
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

func (o *limitedWriteObjectCloser) Write(p []byte) (int, error) {
	writeSize := int64(len(p))
	newBucketSize := o.bucketSize.Add(writeSize)
	if newBucketSize > o.limit {
		o.bucketSize.Sub(writeSize)
		return 0, &errWriteLimitReached{
			Limit:       o.limit,
			ExceedingBy: newBucketSize - o.limit,
		}
	}
	writtenSize, err := o.WriteObjectCloser.Write(p)
	if int64(writtenSize) < writeSize {
		o.bucketSize.Sub(writeSize - int64(writtenSize))
	}
	return writtenSize, err
}
