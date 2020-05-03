// Copyright 2020 Buf Technologies Inc.
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

import "github.com/bufbuild/buf/internal/pkg/storage"

// NewObjectInfo returns a new ObjectInfo.
func NewObjectInfo(size uint32) storage.ObjectInfo {
	return objectInfo{size: size}
}

type objectInfo struct {
	size uint32
}

func (o objectInfo) Size() uint32 {
	return o.size
}

// NewBucketInfo returns a new BucketInfo.
func NewBucketInfo(inMemory bool) storage.BucketInfo {
	return bucketInfo{inMemory: inMemory}
}

type bucketInfo struct {
	inMemory bool
}

func (b bucketInfo) InMemory() bool {
	return b.inMemory
}
