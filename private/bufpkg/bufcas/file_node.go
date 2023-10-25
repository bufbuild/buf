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
	"strings"

	storagev1beta1 "github.com/bufbuild/buf/private/gen/proto/go/buf/registry/storage/v1beta1"
	"github.com/bufbuild/buf/private/pkg/normalpath"
)

// FileNode is a path and associated digest.
type FileNode interface {
	// String encodes the FileNode into its canonical form.
	//
	// If the digest is nil, this is simply:
	//
	//   path
	//
	// If the digest is not nil, this is:
	//
	//   digestString[SP][SP]path
	fmt.Stringer

	// Path returns the Path of the file.
	//
	// The path is normalized and non-empty.
	Path() string
	// Digest returns the Digest of the file.
	//
	// The Digest may be nil, in which case the file is empty.
	Digest() Digest

	// Protect against creation of a FileNode outside of this package, as we
	// do very careful validation.
	isFileNode()
}

// NewFileNode returns a new FileNode.
//
// The Digest may be nil.
//
// The path is validated to be normalized and non-empty.
func NewFileNode(path string, digest Digest) (FileNode, error) {
	return newFileNode(path, digest)
}

// NewFileNodeForString returns a new FileNode for the given FileNode string.
//
// This reverses FileNode.String().
func NewFileNodeForString(s string) (FileNode, error) {
	switch split := strings.Split(s, "  "); len(split) {
	case 1:
		return NewFileNode(split[0], nil)
	case 2:
		digest, err := NewDigestForString(split[0])
		if err != nil {
			return nil, err
		}
		return NewFileNode(split[1], digest)
	default:
		return nil, fmt.Errorf("unknown FileNode encoding: %q", s)
	}
}

// FileNodeToProto converts the given FileNode to a proto FileNode.
//
// If the given FileNode is nil, returns nil.
//
// TODO: validate the returned FileNode.
func FileNodeToProto(fileNode FileNode) (*storagev1beta1.FileNode, error) {
	if fileNode == nil {
		return nil, nil
	}
	protoDigest, err := DigestToProto(fileNode.Digest())
	if err != nil {
		return nil, err
	}
	return &storagev1beta1.FileNode{
		Path:   fileNode.Path(),
		Digest: protoDigest,
	}, nil
}

// ProtoToFileNode converts the given proto FileNode to a FileNode.
//
// If the given proto FileNode is nil, returns nil.
//
// The path is validated to be normalized and non-empty.
// TODO: validate the input proto FileNode.
func ProtoToFileNode(protoFileNode *storagev1beta1.FileNode) (FileNode, error) {
	if protoFileNode == nil {
		return nil, nil
	}
	digest, err := ProtoToDigest(protoFileNode.Digest)
	if err != nil {
		return nil, err
	}
	return NewFileNode(protoFileNode.Path, digest)
}

// *** PRIVATE ***

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
