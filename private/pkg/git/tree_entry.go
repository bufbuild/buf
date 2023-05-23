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

package git

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
)

type treeEntry struct {
	name string
	mode FileMode
	hash Hash
}

func (e *treeEntry) Name() string {
	return e.name
}

func (e *treeEntry) Mode() FileMode {
	return e.mode
}

func (e *treeEntry) Hash() Hash {
	return e.hash
}

func parseTreeEntry(data []byte) (*treeEntry, error) {
	modeAndName, hash, found := bytes.Cut(data, []byte{0})
	if !found {
		return nil, errors.New("malformed entry")
	}
	parsedHash, err := newHashFromBytes(hash)
	if err != nil {
		return nil, fmt.Errorf("malformed git tree entry: %w", err)
	}
	mode, name, found := bytes.Cut(modeAndName, []byte{' '})
	if !found {
		return nil, errors.New("malformed entry")
	}
	parsedFileMode, err := parseFileMode(mode)
	if err != nil {
		return nil, fmt.Errorf("malformed git tree entry: %w", err)
	}
	return &treeEntry{
		hash: parsedHash,
		name: string(name),
		mode: parsedFileMode,
	}, nil
}

// decodes the octal form of a file mode into one of the valid Mode* values.
func parseFileMode(data []byte) (FileMode, error) {
	mode, err := strconv.ParseUint(string(data), 8, 32)
	if err != nil {
		return 0, err
	}
	switch FileMode(mode) {
	case ModeFile:
	case ModeExe:
	case ModeDir:
	case ModeSymlink:
	case ModeSubmodule:
	default:
		return 0, fmt.Errorf("unknown file mode: %o", mode)
	}
	return FileMode(mode), nil
}
