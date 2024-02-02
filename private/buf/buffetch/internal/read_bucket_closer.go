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

package internal

import (
	"context"

	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
)

var _ ReadBucketCloser = &readBucketCloser{}

type readBucketCloser struct {
	storage.ReadBucketCloser

	subDirPath          string
	pathForExternalPath func(string) (string, error)
}

func newReadBucketCloser(
	storageReadBucketCloser storage.ReadBucketCloser,
	subDirPath string,
	pathForExternalPath func(string) (string, error),
) (*readBucketCloser, error) {
	normalizedSubDirPath, err := normalpath.NormalizeAndValidate(subDirPath)
	if err != nil {
		return nil, err
	}
	return &readBucketCloser{
		ReadBucketCloser:    storageReadBucketCloser,
		subDirPath:          normalizedSubDirPath,
		pathForExternalPath: pathForExternalPath,
	}, nil
}

func newReadBucketCloserForReadWriteBucket(
	readWriteBucket ReadWriteBucket,
) *readBucketCloser {
	return &readBucketCloser{
		ReadBucketCloser:    storage.NopReadBucketCloser(readWriteBucket),
		subDirPath:          readWriteBucket.SubDirPath(),
		pathForExternalPath: readWriteBucket.PathForExternalPath,
	}
}

func (r *readBucketCloser) SubDirPath() string {
	return r.subDirPath
}

func (r *readBucketCloser) PathForExternalPath(externalPath string) (string, error) {
	return r.pathForExternalPath(externalPath)
}

func (r *readBucketCloser) copyToInMemory(ctx context.Context) (*readBucketCloser, error) {
	storageReadBucket, err := storagemem.CopyReadBucket(ctx, r.ReadBucketCloser)
	if err != nil {
		return nil, err
	}
	return &readBucketCloser{
		ReadBucketCloser: compositeStorageReadBucketCloser{
			ReadBucket: storageReadBucket,
			closeFunc:  r.ReadBucketCloser.Close,
		},
		subDirPath:          r.subDirPath,
		pathForExternalPath: r.pathForExternalPath,
	}, nil
}

type compositeStorageReadBucketCloser struct {
	storage.ReadBucket
	closeFunc func() error
}

func (c compositeStorageReadBucketCloser) Close() error {
	if c.closeFunc != nil {
		return c.closeFunc()
	}
	return nil
}
