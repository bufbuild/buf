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

package bufcas

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Manifest is a set of FileNodes.
type Manifest interface {
	// fmt.Stringer encodes the Manifest into its canonical form, consisting of
	// an sorted list of paths and their digests, sorted by path.
	//
	// See the documentation on FileNode for how FileNodes are encoded.
	//
	// An example encoded manifest:
	//
	//	shake256:cd22db48cf7c274bbffcb5494a854000cd21b074df7c6edabbd0102c4be8d7623e3931560fcda7acfab286ae1d4f506911daa31f223ee159f59ffce0c7acbbaa  buf.lock
	//	shake256:3b353aa5aacd11015e8577f16e2c4e7a242ce773d8e3a16806795bb94f76e601b0db9bf42d5e1907fda63303e1fa1c65f1c175ecc025a3ef29c3456ad237ad84  buf.md
	//	shake256:7c88a20cf931702d042a4ddee3fde5de84814544411f1c62dbf435b1b81a12a8866a070baabcf8b5a0d31675af361ccb2d93ddada4cdcc11bab7ea3d8d7c4667  buf.yaml
	//	shake256:9db25155eafd19b36882cff129daac575baa67ee44d1cb1fd3894342b28c72b83eb21aa595b806e9cb5344759bc8308200c5af98e4329aa83014dde99afa903a  pet/v1/pet.proto
	fmt.Stringer

	// FileNodes returns the set of FileNodes that make up the Manifest.
	//
	// The paths of the given FileNodes are guaranteed to be unique.
	// The iteration order will be the sorted order of the paths.
	FileNodes() []FileNode
	// GetFileNode gets the FileNode for the given path.
	//
	// Returns nil if the path does not exist.
	GetFileNode(path string) FileNode
	// GetDigest gets the Digest for the given path.
	//
	// Returns nil if the path does not exist.
	GetDigest(path string) Digest

	// Protect against creation of a Manifest outside of this package, as we
	// do very careful validation.
	isManifest()
}

// NewManifest returns a new Manifest for the given path -> Digest map.
//
// FileNodes are deduplicated upon construction, however if two FileNodes
// with the same path have different Digests, an error is returned.
func NewManifest(fileNodes []FileNode) (Manifest, error) {
	pathToFileNode, err := getAndValidateManifestPathToFileNode(fileNodes)
	if err != nil {
		return nil, err
	}
	return newManifest(pathToFileNode), nil
}

// ParseManifest parses a Manifest from its string representation.
//
// This reverses Manifest.String().
func ParseManifest(s string) (Manifest, error) {
	var fileNodes []FileNode
	original := s
	if len(s) > 0 {
		if s[len(s)-1] != '\n' {
			return nil, &ParseError{
				typeString: "manifest",
				input:      original,
				err:        errors.New("did not end with newline"),
			}
		}
		s = s[:len(s)-1]
		for i, line := range strings.Split(s, "\n") {
			fileNode, err := ParseFileNode(line)
			if err != nil {
				return nil, &ParseError{
					typeString: "manifest",
					input:      original,
					err:        fmt.Errorf("line %d: %w", i, err),
				}
			}
			fileNodes = append(fileNodes, fileNode)
		}
	}
	// Even if len(s) == 0, we still go through this flow.
	// Just making sure that in the future, we still count an empty manifest as valid.
	// Validation occurs within getAndValidateManifestPathToFileNode, so we pass nil to that.
	pathToFileNode, err := getAndValidateManifestPathToFileNode(fileNodes)
	if err != nil {
		return nil, &ParseError{
			typeString: "manifest",
			input:      original,
			err:        err,
		}
	}
	return newManifest(pathToFileNode), nil
}

// ManifestToBlob converts the string representation of the given Manifest into a Blob.
//
// The Manifest is assumed to be non-nil.
func ManifestToBlob(manifest Manifest) (Blob, error) {
	return NewBlobForContent(strings.NewReader(manifest.String()))
}

// ManifestToDigest converts the string representation of the given Manifest into a Digest.
//
// The Manifest is assumed to be non-nil.
func ManifestToDigest(manifest Manifest) (Digest, error) {
	return NewDigestForContent(strings.NewReader(manifest.String()))
}

// BlobToManifest converts the given Blob representing the string representation of a Manifest into a Manifest.
//
// # The Blob is assumed to be non-nil
//
// This function returns ParseErrors since this is effectively parsing the blob.
func BlobToManifest(blob Blob) (Manifest, error) {
	return ParseManifest(string(blob.Content()))
}

// *** PRIVATE ***

type manifest struct {
	pathToFileNode        map[string]FileNode
	sortedUniqueFileNodes []FileNode
}

// use getAndValidateManifestPathToFileNode to create pathToFileNode.
func newManifest(pathToFileNode map[string]FileNode) *manifest {
	// Just cache ahead of time for now.
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
		pathToFileNode:        pathToFileNode,
		sortedUniqueFileNodes: sortedUniqueFileNodes,
	}
}

func (m *manifest) FileNodes() []FileNode {
	return m.sortedUniqueFileNodes
}

func (m *manifest) GetFileNode(path string) FileNode {
	return m.pathToFileNode[path]
}

func (m *manifest) GetDigest(path string) Digest {
	fileNode := m.GetFileNode(path)
	if fileNode == nil {
		return nil
	}
	return fileNode.Digest()
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

// serves as parameter validation as well.
func getAndValidateManifestPathToFileNode(fileNodes []FileNode) (map[string]FileNode, error) {
	pathToFileNode := make(map[string]FileNode)
	for _, fileNode := range fileNodes {
		if existingFileNode, ok := pathToFileNode[fileNode.Path()]; ok {
			errorMessage := fmt.Sprintf("path %q was duplicated when creating a manifest", fileNode.Path())
			if !DigestEqual(existingFileNode.Digest(), fileNode.Digest()) {
				errorMessage += fmt.Sprintf(
					" and the two path entries had different digests: %q, %q",
					existingFileNode.Digest().String(),
					fileNode.Digest().String(),
				)
			}
			return nil, errors.New(errorMessage)
		} else {
			pathToFileNode[fileNode.Path()] = fileNode
		}
	}
	return pathToFileNode, nil
}
