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

package buffetch

import (
	"path/filepath"

	"github.com/bufbuild/buf/private/buf/buffetch/internal"
	"github.com/bufbuild/buf/private/pkg/normalpath"
)

var _ ProtoFileRef = &protoFileRef{}

type protoFileRef struct {
	protoFileRef internal.ProtoFileRef
	dirPath      string
}

func newProtoFileRef(internalProtoFileRef internal.ProtoFileRef) *protoFileRef {
	return &protoFileRef{
		protoFileRef: internalProtoFileRef,
		dirPath:      internalProtoFileRef.Path(),
	}
}

func (r *protoFileRef) PathForExternalPath(externalPath string) (string, error) {
	if r.dirPath == "" {
		return normalpath.NormalizeAndValidate(externalPath)
	}
	absDirPath, err := filepath.Abs(normalpath.Unnormalize(r.dirPath))
	if err != nil {
		return "", err
	}
	// we don't actually need to unnormalize externalPath but we do anyways
	absExternalPath, err := filepath.Abs(normalpath.Unnormalize(externalPath))
	if err != nil {
		return "", err
	}
	path, err := filepath.Rel(absDirPath, absExternalPath)
	if err != nil {
		return "", err
	}
	return normalpath.NormalizeAndValidate(path)
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
