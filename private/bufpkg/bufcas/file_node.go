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
	"errors"
	"fmt"
	"strings"

	"github.com/bufbuild/buf/private/pkg/normalpath"
)

// FileNode is a path and associated digest.
type FileNode interface {
	// String encodes the FileNode into its canonical form:
	//
	//   digestString[SP][SP]path
	fmt.Stringer

	// Path returns the Path of the file.
	//
	// The path is normalized and non-empty.
	Path() string
	// Digest returns the Digest of the file.
	//
	// The Digest is always non-nil.
	Digest() Digest

	// Protect against creation of a FileNode outside of this package, as we
	// do very careful validation.
	isFileNode()
}

// NewFileNode returns a new FileNode.
//
// The path is validated to be normalized and non-empty.
// The digest is validated to be non-nil.
func NewFileNode(path string, digest Digest) (FileNode, error) {
	if err := validateFileNodeParameters(path, digest); err != nil {
		return nil, err
	}
	return newFileNode(path, digest), nil
}

// ParseFileNode parses the FileNode from its string representation.
//
// The string representation is "digestString[SP][SP]path".
//
// This reverses FileNode.String().
func ParseFileNode(s string) (FileNode, error) {
	split := strings.Split(s, "  ")
	if len(split) != 2 {
		return nil, &ParseError{
			typeString: "file node",
			input:      s,
			err:        errors.New(`must in the form "digest[SP][SP]path"`),
		}
	}
	digest, err := ParseDigest(split[0])
	if err != nil {
		return nil, &ParseError{
			typeString: "file node",
			input:      s,
			err:        err,
		}
	}
	path := split[1]
	if err := validateFileNodeParameters(path, digest); err != nil {
		return nil, &ParseError{
			typeString: "file node",
			input:      s,
			err:        err,
		}
	}
	return newFileNode(path, digest), nil
}

// *** PRIVATE ***

type fileNode struct {
	path   string
	digest Digest
}

// validation should occur outside of this function.
func newFileNode(path string, digest Digest) *fileNode {
	return &fileNode{
		path:   path,
		digest: digest,
	}
}

func (f *fileNode) Path() string {
	return f.path
}

func (f *fileNode) Digest() Digest {
	return f.digest
}

func (f *fileNode) String() string {
	return f.digest.String() + "  " + f.path
}

func (*fileNode) isFileNode() {}

func validateFileNodeParameters(path string, digest Digest) error {
	if path == "" {
		return errors.New("path was empty")
	}
	normalizedPath, err := normalpath.NormalizeAndValidate(path)
	if err != nil {
		return fmt.Errorf("path %q was not valid: %w", path, err)
	}
	if path != normalizedPath {
		return fmt.Errorf("path %q was not equal to normalized path %q", path, normalizedPath)
	}
	if digest == nil {
		return errors.New("no digest specified")
	}
	return nil
}
