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

package bufimage

import (
	"github.com/bufbuild/buf/internal/buf/bufpath"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
)

var _ FileRef = &fileRef{}

type fileRef struct {
	rootRelFilePath  string
	rootDirPath      string
	externalFilePath string
}

// externalPathResolver may be nil
func newFileRef(
	rootRelFilePath string,
	rootDirPath string,
	externalPathResolver bufpath.ExternalPathResolver,
) (*fileRef, error) {
	if rootDirPath == "" {
		rootDirPath = "."
	}
	if err := validateRootRelFilePath(rootRelFilePath); err != nil {
		return nil, err
	}
	if err := validateRootDirPath(rootDirPath); err != nil {
		return nil, err
	}
	externalFilePath, err := externalPathResolver.RelPathToExternalPath(
		normalpath.Join(
			rootDirPath,
			rootRelFilePath,
		),
	)
	if err != nil {
		return nil, err
	}
	return newFileRefNoValidate(
		rootRelFilePath,
		rootDirPath,
		externalFilePath,
	), nil
}

func newDirectFileRef(
	rootRelFilePath string,
	rootDirPath string,
	externalFilePath string,
) (*fileRef, error) {
	if rootDirPath == "" {
		rootDirPath = "."
	}
	if err := validateRootRelFilePath(rootRelFilePath); err != nil {
		return nil, err
	}
	if err := validateRootDirPath(rootDirPath); err != nil {
		return nil, err
	}
	return newFileRefNoValidate(
		rootRelFilePath,
		rootDirPath,
		externalFilePath,
	), nil
}

func newFileRefNoValidate(
	rootRelFilePath string,
	rootDirPath string,
	externalFilePath string,
) *fileRef {
	return &fileRef{
		rootRelFilePath:  rootRelFilePath,
		rootDirPath:      rootDirPath,
		externalFilePath: externalFilePath,
	}
}

func (f *fileRef) RootRelFilePath() string {
	return f.rootRelFilePath
}

func (f *fileRef) RootDirPath() string {
	return f.rootDirPath
}

func (f *fileRef) ExternalFilePath() string {
	return f.externalFilePath
}

func (*fileRef) isFileRef() {}
