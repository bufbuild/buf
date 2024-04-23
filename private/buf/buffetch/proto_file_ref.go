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

var _ ProtoFileRef = &protoFileRef{}

type protoFileRef struct {
	protoFileRef internal.ProtoFileRef
}

func newProtoFileRef(internalProtoFileRef internal.ProtoFileRef) *protoFileRef {
	return &protoFileRef{
		protoFileRef: internalProtoFileRef,
	}
}

func (r *protoFileRef) ProtoFilePath() string {
	return r.protoFileRef.Path()
}

func (r *protoFileRef) IncludePackageFiles() bool {
	return r.protoFileRef.IncludePackageFiles()
}

func (r *protoFileRef) IsDevPath() bool {
	switch r.protoFileRef.FileScheme() {
	case internal.FileSchemeStdio,
		internal.FileSchemeStdin,
		internal.FileSchemeStdout,
		internal.FileSchemeNull:
		return true
	default:
		return false
	}
}

func (r *protoFileRef) internalRef() internal.Ref {
	return r.protoFileRef
}

func (r *protoFileRef) internalBucketRef() internal.BucketRef {
	return r.protoFileRef
}

func (r *protoFileRef) internalProtoFileRef() internal.ProtoFileRef {
	return r.protoFileRef
}

func (*protoFileRef) isSourceOrModuleRef() {}

func (*protoFileRef) isDirOrProtoFileRef() {}
