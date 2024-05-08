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

package buffetch

import (
	"github.com/bufbuild/buf/private/buf/buffetch/internal"
)

var _ DirRef = &dirRef{}

type dirRef struct {
	iDirRef internal.DirRef
}

func newDirRef(iDirRef internal.DirRef) *dirRef {
	return &dirRef{
		iDirRef: iDirRef,
	}
}

func (r *dirRef) DirPath() string {
	return r.iDirRef.Path()
}

func (r *dirRef) internalRef() internal.Ref {
	return r.iDirRef
}

func (r *dirRef) internalBucketRef() internal.BucketRef {
	return r.iDirRef
}

func (r *dirRef) internalDirRef() internal.DirRef {
	return r.iDirRef
}

func (*dirRef) isSourceOrModuleRef() {}

func (*dirRef) isDirOrProtoFileRef() {}
