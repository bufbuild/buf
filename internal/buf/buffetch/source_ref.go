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

package buffetch

import (
	"github.com/bufbuild/buf/internal/buf/bufpath"
	"github.com/bufbuild/buf/internal/pkg/fetch"
)

var _ SourceRef = &sourceRef{}

type sourceRef struct {
	bucketRef       fetch.BucketRef
	dirPathResolver bufpath.PathResolver
}

func newSourceRef(bucketRef fetch.BucketRef) *sourceRef {
	var dirPathResolver bufpath.PathResolver
	if dirRef, ok := bucketRef.(fetch.DirRef); ok {
		dirPathResolver = bufpath.NewDirPathResolver(dirRef.Path())
	}
	return &sourceRef{
		bucketRef:       bucketRef,
		dirPathResolver: dirPathResolver,
	}
}

func (r *sourceRef) PathResolver() bufpath.PathResolver {
	if r.dirPathResolver != nil {
		return r.dirPathResolver
	}
	return bufpath.NopPathResolver
}

func (r *sourceRef) fetchRef() fetch.Ref {
	return r.bucketRef
}

func (r *sourceRef) fetchBucketRef() fetch.BucketRef {
	return r.bucketRef
}
