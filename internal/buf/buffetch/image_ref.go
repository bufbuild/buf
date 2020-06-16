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

package buffetch

import (
	"github.com/bufbuild/buf/internal/buf/bufpath"
	"github.com/bufbuild/buf/internal/pkg/fetch"
)

var _ ImageRef = &imageRef{}

type imageRef struct {
	fileRef       fetch.FileRef
	imageEncoding ImageEncoding
}

func newImageRef(
	fileRef fetch.FileRef,
	imageEncoding ImageEncoding,
) *imageRef {
	return &imageRef{
		fileRef:       fileRef,
		imageEncoding: imageEncoding,
	}
}

func (*imageRef) PathResolver() bufpath.PathResolver {
	return bufpath.NopPathResolver
}

func (r *imageRef) ImageEncoding() ImageEncoding {
	return r.imageEncoding
}

func (r *imageRef) IsNull() bool {
	return r.fileRef.FileScheme() == fetch.FileSchemeNull
}

func (r *imageRef) fetchRef() fetch.Ref {
	return r.fileRef
}

func (r *imageRef) fetchFileRef() fetch.FileRef {
	return r.fileRef
}
