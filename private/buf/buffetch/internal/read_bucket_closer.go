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
	"fmt"
	"path/filepath"

	"github.com/bufbuild/buf/private/buf/buftarget"
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
	bucketTargeting buftarget.BucketTargeting,
) (*readBucketCloser, error) {
	normalizedSubDirPath, err := normalpath.NormalizeAndValidate(bucketTargeting.InputPath())
	if err != nil {
		return nil, err
	}
	return &readBucketCloser{
		ReadBucketCloser: storageReadBucketCloser,
		subDirPath:       normalizedSubDirPath,
		// This turns paths that were done relative to the root of the input into paths
		// that are now relative to the mapped bucket.
		//
		// This happens if you do i.e. .git#subdir=foo/bar --path foo/bar/baz.proto
		// We need to turn the path into baz.proto
		pathForExternalPath: func(externalPath string) (string, error) {
			if filepath.IsAbs(externalPath) {
				return "", fmt.Errorf("%s: absolute paths cannot be used for this input type", externalPath)
			}
			if !normalpath.EqualsOrContainsPath(bucketTargeting.ControllingWorkspacePath(), externalPath, normalpath.Relative) {
				return "", fmt.Errorf("path %q from input does not contain path %q", bucketTargeting.ControllingWorkspacePath(), externalPath)
			}
			relPath, err := normalpath.Rel(bucketTargeting.ControllingWorkspacePath(), externalPath)
			if err != nil {
				return "", err
			}
			return normalpath.NormalizeAndValidate(relPath)
		},
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
