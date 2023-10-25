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
	"bytes"
	"fmt"
	"sort"
)

type manifest struct {
	sortedUniqueFileNodes []FileNode
}

func newManifest(fileNodes []FileNode) (*manifest, error) {
	pathToFileNode := make(map[string]FileNode)
	for _, fileNode := range fileNodes {
		if existingFileNode, ok := pathToFileNode[fileNode.Path()]; ok {
			// Handles nil case
			if !DigestEqual(existingFileNode.Digest(), fileNode.Digest()) {
				return nil, fmt.Errorf("path %q had different Digests when constructing FileNode", fileNode.Path())
			}
		} else {
			pathToFileNode[fileNode.Path()] = fileNode
		}
	}
	sortedUniqueFileNodes := make([]FileNode, 0, len(pathToFileNode))
	for _, fileNode := range pathToFileNode {
		sortedUniqueFileNodes = append(sortedUniqueFileNodes, fileNode)
	}
	sort.Slice(
		sortedUniqueFileNodes,
		func(i int, j int) bool {
			return sortedUniqueFileNodes[i].Path() < sortedUniqueFileNodes[j].Path()
		},
	)
	return &manifest{
		sortedUniqueFileNodes: sortedUniqueFileNodes,
	}, nil
}

func (m *manifest) FileNodes() []FileNode {
	return m.sortedUniqueFileNodes
}

func (m *manifest) String() string {
	buffer := bytes.NewBuffer(nil)
	for _, fileNode := range m.sortedUniqueFileNodes {
		_, _ = buffer.WriteString(fileNode.String())
		_, _ = buffer.WriteRune('\n')
	}
	return buffer.String()
}

func (*manifest) isManifest() {}
