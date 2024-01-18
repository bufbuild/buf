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
	"fmt"
	"path/filepath"

	"github.com/bufbuild/buf/private/buf/buffetch/internal"
	"github.com/bufbuild/buf/private/pkg/normalpath"
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

// PathForExternalPath for a proto file ref will only ever have one successful case, which
// is `".", <nil>` and will error on all other paths. The `Path()` of the internal.ProtoFileRef
// will always point to a specific proto file, e.g. `foo/bar/baz.proto`, thus the function
// errors against inputs that are not matching to the proto file ref input.
func (r *protoFileRef) PathForExternalPath(externalPath string) (string, error) {
	externalPathAbs, err := filepath.Abs(normalpath.Unnormalize(externalPath))
	if err != nil {
		return "", err
	}
	internalRefPathAbs, err := filepath.Abs(normalpath.Unnormalize(r.protoFileRef.Path()))
	if err != nil {
		return "", err
	}
	if externalPathAbs != internalRefPathAbs {
		return "", fmt.Errorf(`path provided "%s" does not match ref path "%s"`, externalPath, r.protoFileRef.Path())
	}
	return ".", nil
}

func (r *protoFileRef) IncludePackageFiles() bool {
	return r.protoFileRef.IncludePackageFiles()
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
