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

package bufpath

import (
	"os"
	"path/filepath"

	"github.com/bufbuild/buf/internal/pkg/normalpath"
)

type dirPathResolver struct {
	normalizedDirPath string
}

func newDirPathResolver(dirPath string) *dirPathResolver {
	return &dirPathResolver{
		normalizedDirPath: normalpath.Normalize(dirPath),
	}
}

func (r *dirPathResolver) ExternalPathToRelPath(externalPath string) (string, error) {
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

func (r *dirPathResolver) RelPathToExternalPath(relPath string) (string, error) {
	relPath, err := normalpath.NormalizeAndValidate(relPath)
	if err != nil {
		return "", err
	}
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
	// else, we want to make the path relative to the current directory
	absExternalPath, err := filepath.Abs(externalPath)
	if err != nil {
		return "", err
	}
	// TODO: cache this?
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Rel(pwd, absExternalPath)
}
