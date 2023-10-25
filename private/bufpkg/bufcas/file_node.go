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

package bufcas

import (
	"errors"
	"fmt"

	"github.com/bufbuild/buf/private/pkg/normalpath"
)

type fileNode struct {
	path   string
	digest Digest
}

func newFileNode(path string, digest Digest) (*fileNode, error) {
	if path == "" {
		return nil, errors.New("path was empty when constructing a FileNode")
	}
	normalizedPath, err := normalpath.NormalizeAndValidate(path)
	if err != nil {
		return nil, fmt.Errorf("path %q was not valid when constructing a FileNode: %w", path, err)
	}
	if path != normalizedPath {
		return nil, fmt.Errorf("path %q was not equal to normalized path %q when constructing a FileNode", path, normalizedPath)
	}
	return &fileNode{
		path:   path,
		digest: digest,
	}, nil
}

func (f *fileNode) Path() string {
	return f.path
}

func (f *fileNode) Digest() Digest {
	return f.digest
}

func (f *fileNode) String() string {
	if f.digest == nil {
		return f.path
	}
	return f.digest.String() + "  " + f.path
}

func (*fileNode) isFileNode() {}
