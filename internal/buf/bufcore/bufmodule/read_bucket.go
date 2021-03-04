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

package bufmodule

import (
	"context"

	"github.com/bufbuild/buf/internal/pkg/storage"
)

// readBucket implements the ReadBucket interface.
type readBucket struct {
	storage.ReadBucket

	moduleReference ModuleReference
}

func newReadBucket(
	sourceReadBucket storage.ReadBucket,
	moduleReference ModuleReference,
) *readBucket {
	return &readBucket{
		ReadBucket:      sourceReadBucket,
		moduleReference: moduleReference,
	}
}

// StatModuleFile gets info in the object, including info
// specific to the file's module.
func (r *readBucket) StatModuleFile(ctx context.Context, path string) (ObjectInfo, error) {
	objectInfo, err := r.ReadBucket.Stat(ctx, path)
	if err != nil {
		return nil, err
	}
	return newObjectInfo(objectInfo, r.moduleReference), nil
}

// WalkModuleFiles walks the bucket with the prefix, calling f on
// each path. If the prefix doesn't exist, this is a no-op.
func (r *readBucket) WalkModuleFiles(ctx context.Context, path string, f func(ObjectInfo) error) error {
	return r.ReadBucket.Walk(
		ctx,
		path,
		func(objectInfo storage.ObjectInfo) error {
			return f(newObjectInfo(objectInfo, r.moduleReference))
		},
	)
}
