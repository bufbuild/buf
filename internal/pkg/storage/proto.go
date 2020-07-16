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

package storage

import (
	"context"
	"fmt"
	"sort"
	"sync"

	storagev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/storage/v1beta1"
	"github.com/bufbuild/buf/internal/pkg/thread"
)

// ToProtoFileSet copies the ReadBucket to the FileSet.
//
// Copies done concurrently.
// Returns a validated FileSet.
func ToProtoFileSet(
	ctx context.Context,
	readBucket ReadBucket,
) (*storagev1beta1.FileSet, error) {
	paths, err := AllPaths(ctx, readBucket, "")
	if err != nil {
		return nil, err
	}
	files := make([]*storagev1beta1.File, 0, len(paths))
	var lock sync.Mutex
	jobs := make([]func() error, len(paths))
	for i, path := range paths {
		path := path
		jobs[i] = func() error {
			data, err := ReadPath(ctx, readBucket, path)
			if err != nil {
				return err
			}
			file := &storagev1beta1.File{
				Path:    path,
				Content: data,
			}
			lock.Lock()
			files = append(files, file)
			lock.Unlock()
			return nil
		}
	}
	if err := thread.Parallelize(jobs...); err != nil {
		return nil, err
	}
	// we know that the paths are unique already since this is a property of Buckets
	sort.Slice(
		files,
		func(i int, j int) bool {
			return files[i].GetPath() < files[j].GetPath()
		},
	)
	fileSet := &storagev1beta1.FileSet{
		Files: files,
	}
	if err := fileSet.Validate(); err != nil {
		return nil, err
	}
	return fileSet, nil
}

// FromProtoFileSet copies the FileSet to the WriteBucket.
func FromProtoFileSet(
	ctx context.Context,
	fileSet *storagev1beta1.FileSet,
	writeBucket WriteBucket,
) error {
	if err := fileSet.Validate(); err != nil {
		return err
	}
	pathMap := make(map[string]struct{}, len(fileSet.Files))
	for _, file := range fileSet.Files {
		if _, ok := pathMap[file.GetPath()]; ok {
			return fmt.Errorf("duplicate path in FileSet: %s", file.GetPath())
		}
		pathMap[file.GetPath()] = struct{}{}
	}
	jobs := make([]func() error, len(fileSet.Files))
	for i, file := range fileSet.Files {
		file := file
		jobs[i] = func() error {
			return PutPath(ctx, writeBucket, file.GetPath(), file.GetContent())
		}
	}
	return thread.Parallelize(jobs...)
}
