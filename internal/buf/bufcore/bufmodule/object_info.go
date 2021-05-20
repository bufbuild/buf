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
	"github.com/bufbuild/buf/internal/pkg/storage"
)

// objectInfo implements the ObjectInfo interface.
type objectInfo struct {
	storage.ObjectInfo

	moduleCommit ModuleCommit
}

// newObjectInfo returns a new ObjectInfo.
func newObjectInfo(
	storageObjectInfo storage.ObjectInfo,
	moduleCommit ModuleCommit,
) *objectInfo {
	return &objectInfo{
		ObjectInfo:   storageObjectInfo,
		moduleCommit: moduleCommit,
	}
}

// ModuleCommit returns this object's ModuleCommit, if any.
func (o *objectInfo) ModuleCommit() ModuleCommit {
	return o.moduleCommit
}
