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
	"path/filepath"

	"github.com/bufbuild/buf/internal/pkg/fetch"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
)

var _ SourceRef = &sourceRef{}

type sourceRef struct {
	bucketRef fetch.BucketRef

	// we only do external paths for dir refs for now so we cache this
	normalizedDirPath string
	// this is also cached
	workdir string
}

func newSourceRef(
	bucketRef fetch.BucketRef,
	workdir string,
) *sourceRef {
	normalizedDirPath := ""
	if dirRef, ok := bucketRef.(fetch.DirRef); ok {
		normalizedDirPath = dirRef.Path()
	}
	return &sourceRef{
		bucketRef:         bucketRef,
		normalizedDirPath: normalizedDirPath,
		workdir:           workdir,
	}
}

func (r *sourceRef) ExternalPathToRelPath(externalPath string) (string, error) {
	// if not a dir ref, we do nothing for now
	// we will likely want to change this
	if r.normalizedDirPath == "" {
		return normalpath.NormalizeAndValidate(externalPath)
	}
	// we now know we have a dir ref

	// if we had a directory input, then we need to make externalPath relative to that directory
	absDirPath, err := filepath.Abs(normalpath.Unnormalize(r.normalizedDirPath))
	if err != nil {
		return "", err
	}
	// we don't actually need to unnormalize externalPath but we do anyways
	absExternalPath, err := filepath.Abs(normalpath.Unnormalize(externalPath))
	if err != nil {
		return "", err
	}
	relPath, err := filepath.Rel(absDirPath, absExternalPath)
	if err != nil {
		return "", err
	}
	return normalpath.NormalizeAndValidate(relPath)
}

func (r *sourceRef) RelPathToExternalPath(relPath string) (string, error) {
	relPath, err := normalpath.NormalizeAndValidate(relPath)
	if err != nil {
		return "", err
	}
	// if not a dir ref, we do nothing for now
	// we will likely want to change this
	if r.normalizedDirPath == "" {
		return normalpath.Unnormalize(relPath), nil
	}
	// we now know we have a dir ref

	// if the dir path is ".", do nothing, we are done
	if r.normalizedDirPath == "." {
		return normalpath.Unnormalize(relPath), nil
	}
	// add the prefix directory
	// Normalize and Join call filepath.Clean
	externalPath := normalpath.Unnormalize(normalpath.Join(r.normalizedDirPath, relPath))
	// if the directory was absolute, we can output absolute paths
	if filepath.IsAbs(normalpath.Unnormalize(r.normalizedDirPath)) {
		return externalPath, nil
	}
	// else, we want to make the path relative to the currently directory
	absExternalPath, err := filepath.Abs(externalPath)
	if err != nil {
		return "", err
	}
	return filepath.Rel(r.workdir, absExternalPath)
}

func (r *sourceRef) fetchRef() fetch.Ref {
	return r.bucketRef
}

func (r *sourceRef) fetchBucketRef() fetch.BucketRef {
	return r.bucketRef
}
