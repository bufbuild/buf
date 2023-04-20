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

package object

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
)

const (
	// ModeUnknown is a mode's zero value.
	ModeUnknown FileMode = 0
	// ModeFile is a blob that should be written as a plain file.
	ModeFile FileMode = 010_0644
	// ModeExec is a blob that should be written with the executable bit set.
	ModeExe FileMode = 010_0755
	// ModeDir is a tree to be unpacked as a subdirectory in the current
	// directory.
	ModeDir FileMode = 004_0000
	// ModeSymlink is a blob with its content being the path linked to.
	ModeSymlink FileMode = 012_0000
	// ModeSubmodule is a commit that the submodule is checked out at.
	ModeSubmodule FileMode = 016_0000
)

// digestLength is the length, in bytes, of digests in object format SHA1. Each
// entry's digest in a tree object is this fixed length.
const digestLength = 20

// FileMode is how to interpret a tree entry's object. See the Mode* constants
// for how to interpret each mode value.
type FileMode uint32

// Validate returns an error if the value is not a known value.
func (fm FileMode) Validate() error {
	switch fm {
	case ModeFile:
	case ModeExe:
	case ModeDir:
	case ModeSymlink:
	case ModeSubmodule:
	default:
		return fmt.Errorf("unknown file mode: %o", fm)
	}
	return nil
}

// UnmarshalText decodes the octal form of a file mode into one of the valid
// Mode* values.
func (fm *FileMode) UnmarshalText(txt []byte) error {
	mode, err := strconv.ParseUint(string(txt), 8, 32)
	if err != nil {
		return err
	}
	if err := ((FileMode)(mode)).Validate(); err != nil {
		return err
	}
	*fm = (FileMode)(mode)
	return nil
}

// Tree represents a git tree. Trees are a manifest of other git objects,
// including other trees.
type Tree struct {
	Entries []TreeEntry
}

// UnmarshalBinary decodes the binary form of one tree's entry.
func (t *Tree) UnmarshalBinary(data []byte) error {
	t.Entries = nil
	for len(data) > 0 {
		i := bytes.Index(data, []byte{0})
		if i == -1 {
			return errors.New("malformed tree")
		}
		length := i + 1 + digestLength
		ent := TreeEntry{}
		if err := ent.UnmarshalBinary(data[:length]); err != nil {
			return fmt.Errorf("malformed tree: %w", err)
		}
		t.Entries = append(t.Entries, ent)
		data = data[length:]
	}
	return nil
}

// TreeEntry represents a single object describe in a tree. These objects have
// a file mode associated with them, which hints at the type of object located
// at ID is (tree or blob).
type TreeEntry struct {
	Name string
	Mode FileMode
	ID   ID
}

// UnmarshalBinary decodes a tree in binary form. An example binary tree would
// be the standard out of
//
//	$ git cat-file tree main
func (ent *TreeEntry) UnmarshalBinary(data []byte) error {
	modeAndName, hash, found := bytes.Cut(data, []byte{0})
	if !found {
		return errors.New("malformed entry")
	}
	if err := ent.ID.UnmarshalBinary(hash); err != nil {
		return fmt.Errorf("malformed entry: %w", err)
	}
	mode, name, found := bytes.Cut(modeAndName, []byte{' '})
	if !found {
		return errors.New("malformed entry")
	}
	ent.Name = string(name)
	if err := ent.Mode.UnmarshalText(mode); err != nil {
		return err
	}
	return nil
}
