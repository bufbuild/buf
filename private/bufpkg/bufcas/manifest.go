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
	"errors"
	"fmt"
	"sort"
	"strings"

	storagev1beta1 "github.com/bufbuild/buf/private/gen/proto/go/buf/registry/storage/v1beta1"
)

// Maniest is a set FileNodes.
type Manifest interface {
	// fmt.Stringer encodes the Manifest into its canonical form, consisting of
	// an ordered list of paths and their hash digests. Sorted by path.
	//
	// See the documentation on FileNode for how FileNodes are encoded.
	//
	// An example encoded manifest:
	//
	//	shake256:cd22db48cf7c274bbffcb5494a854000cd21b074df7c6edabbd0102c4be8d7623e3931560fcda7acfab286ae1d4f506911daa31f223ee159f59ffce0c7acbbaa  buf.lock
	//	shake256:3b353aa5aacd11015e8577f16e2c4e7a242ce773d8e3a16806795bb94f76e601b0db9bf42d5e1907fda63303e1fa1c65f1c175ecc025a3ef29c3456ad237ad84  buf.md
	//	shake256:7c88a20cf931702d042a4ddee3fde5de84814544411f1c62dbf435b1b81a12a8866a070baabcf8b5a0d31675af361ccb2d93ddada4cdcc11bab7ea3d8d7c4667  buf.yaml
	//  pet/v1/empty_file.proto
	//	shake256:9db25155eafd19b36882cff129daac575baa67ee44d1cb1fd3894342b28c72b83eb21aa595b806e9cb5344759bc8308200c5af98e4329aa83014dde99afa903a  pet/v1/pet.proto
	fmt.Stringer

	// FileNodes returns the set of FileNodes that make up the Manifest.
	//
	// The paths of the given FileNodes are guaranteed to be unique.
	// The iteration order will be the sorted order of the paths.
	FileNodes() []FileNode
	// GetDigest gets the Digest for the given path.
	//
	// Returns nil and true if the path exists, but the file is empty.
	// Returns false if the path does not exist.
	GetDigest(path string) (Digest, bool)

	// Protect against creation of a Manifest outside of this package, as we
	// do very careful validation.
	isManifest()
}

// NewManifest returns a new Manifest for the given path -> Digest map.
//
// FileNodes are deduplicated upon construction, however if two FileNodes
// with the same path have different Digests, an error is returned.
func NewManifest(fileNodes []FileNode) (Manifest, error) {
	return newManifest(fileNodes)
}

// NewManifestForString returns a new Manifest for the given Manifest string.
//
// This reverses Manifest.String().
func NewManifestForString(s string) (Manifest, error) {
	var fileNodes []FileNode
	if s[len(s)-1] != '\n' {
		return nil, errors.New("string for Manifest did not end with newline")
	}
	for i, line := range strings.Split(s, "\n") {
		fileNode, err := NewFileNodeForString(line)
		if err != nil {
			return nil, fmt.Errorf("invalid Manifest at line %d: %w", i, err)
		}
		fileNodes = append(fileNodes, fileNode)
	}
	return NewManifest(fileNodes)
}

// ManifestToBlob converts the string representation of the given Manifest into a Blob.
//
// The Manifest is assumed to be non-nil
func ManifestToBlob(manifest Manifest) (Blob, error) {
	return NewBlobForContent(DigestTypeShake256, strings.NewReader(manifest.String()))
}

// BlobToManifest converts the given Blob representing the string representation of a Manifest into a Manifest.
//
// The Blob is assumed to be non-nil
func BlobToManifest(blob Blob) (Manifest, error) {
	return NewManifestForString(string(blob.Content()))
}

// ManifestToProtoBlob converts the string representation of the given Manifest into a proto Blob.
//
// # The Manifest is assumed to be non-nil
//
// TODO: validate the returned proto Blob.
func ManifestToProtoBlob(manifest Manifest) (*storagev1beta1.Blob, error) {
	blob, err := ManifestToBlob(manifest)
	if err != nil {
		return nil, err
	}
	return BlobToProto(blob)
}

// BlobToManifest converts the given proto Blob representing the string representation of a Manifest into a Manifest.
//
// # The proto Blob is assumed to be non-nil
//
// TODO: validate the input proto Blob.
func ProtoBlobToManifest(protoBlob *storagev1beta1.Blob) (Manifest, error) {
	blob, err := ProtoToBlob(protoBlob)
	if err != nil {
		return nil, err
	}
	return BlobToManifest(blob)
}

// *** PRIVATE ***

type manifest struct {
	// Stores valid paths with nil digests as well
	pathToFileNode        map[string]FileNode
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
	}, nil
}

func (m *manifest) FileNodes() []FileNode {
	return m.sortedUniqueFileNodes
}

func (m *manifest) GetDigest(path string) (Digest, bool) {
	fileNode, ok := m.pathToFileNode[path]
	if !ok {
		return nil, false
	}
	return fileNode.Digest(), true
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
