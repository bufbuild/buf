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
	"path/filepath"

	"github.com/bufbuild/buf/private/buf/buftarget"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
)

var _ ReadWriteBucket = &readWriteBucket{}

type readWriteBucket struct {
	storage.ReadWriteBucket

	subDirPath          string
	pathForExternalPath func(string) (string, error)
}

func newReadWriteBucket(
	storageReadWriteBucket storage.ReadWriteBucket,
	bucketTargeting buftarget.BucketTargeting,
) (*readWriteBucket, error) {
	// TODO(doria): document this
	normalizedSubDirPath, err := normalpath.NormalizeAndValidate(bucketTargeting.InputPath())
	if err != nil {
		return nil, err
	}
	return &readWriteBucket{
		ReadWriteBucket: storageReadWriteBucket,
		subDirPath:      normalizedSubDirPath,
		pathForExternalPath: func(externalPath string) (string, error) {
			absBucketPath, err := filepath.Abs(normalpath.Unnormalize(bucketTargeting.ControllingWorkspacePath()))
			if err != nil {
				return "", err
			}
			absExternalPath, err := filepath.Abs(normalpath.Unnormalize(externalPath))
			if err != nil {
				return "", err
			}
			path, err := filepath.Rel(absBucketPath, absExternalPath)
			if err != nil {
				return "", err
			}
			return normalpath.NormalizeAndValidate(path)
		},
	}, nil
}

func (r *readWriteBucket) SubDirPath() string {
	return r.subDirPath
}

func (r *readWriteBucket) PathForExternalPath(externalPath string) (string, error) {
	return r.pathForExternalPath(externalPath)
}
