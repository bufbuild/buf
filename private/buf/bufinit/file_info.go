// Copyright 2020-2023 Buf Technologies, Inc.
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

package bufinit

import (
	"errors"
	"fmt"
	"sort"

	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/protocompile/ast"
)

type fileInfo struct {
	// Normalized, validated, and never empty or ".".
	Path    string `json:"path,omitempty" yaml:"path,omitempty"`
	Package string `json:"package,omitempty" yaml:"package,omitempty"`
	// Normalized, validated, and each element never empty or ".".
	ImportPaths []string `json:"import_paths,omitempty" yaml:"import_paths,omitempty"`
}

func newFileInfo(fileNode *ast.FileNode) (*fileInfo, error) {
	// Should always be normalized, but defensive programming.
	path, err := normalizeAndValidateProtoFile(fileNode.Name())
	if err != nil {
		return nil, err
	}
	fileInfo := &fileInfo{
		Path: path,
	}
	for _, decl := range fileNode.Decls {
		switch decl := decl.(type) {
		case *ast.ImportNode:
			// Should always be normalized, but defensive programming.
			importPath, err := normalizeAndValidateProtoFile(decl.Name.AsString())
			if err != nil {
				return nil, err
			}
			fileInfo.ImportPaths = append(fileInfo.ImportPaths, importPath)
		case *ast.PackageNode:
			fileInfo.Package = string(decl.Name.AsIdentifier())
		}
	}
	sort.Slice(
		fileInfo.ImportPaths,
		func(i int, j int) bool {
			return fileInfo.ImportPaths[i] < fileInfo.ImportPaths[j]
		},
	)
	return fileInfo, nil
}

func sortFileInfos(fileInfos []*fileInfo) {
	sort.Slice(
		fileInfos,
		func(i int, j int) bool {
			return fileInfos[i].Path < fileInfos[j].Path
		},
	)
}

func normalizeAndValidateProtoFile(path string) (string, error) {
	if path == "" {
		return "", errors.New("empty path")
	}
	if path == "." {
		return "", fmt.Errorf("path cannot be '.'")
	}
	return normalpath.NormalizeAndValidate(path)
}
