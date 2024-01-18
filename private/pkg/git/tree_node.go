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

package git

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
)

type treeNode struct {
	name string
	mode ObjectMode
	hash Hash
}

func parseTreeNode(data []byte) (*treeNode, error) {
	/*
		data is in the format
			<mode><space><name>\0<hash>
	*/
	modeAndName, hash, found := bytes.Cut(data, []byte{0})
	if !found {
		return nil, errors.New("parse tree node")
	}
	parsedHash, err := newHashFromBytes(hash)
	if err != nil {
		return nil, fmt.Errorf("parse tree node hash: %w", err)
	}
	mode, name, found := bytes.Cut(modeAndName, []byte{' '})
	if !found {
		return nil, errors.New("parse tree node")
	}
	parsedFileMode, err := parseObjectMode(mode)
	if err != nil {
		return nil, fmt.Errorf("parse tree node object mode: %w", err)
	}
	return &treeNode{
		hash: parsedHash,
		name: string(name),
		mode: parsedFileMode,
	}, nil
}

func (e *treeNode) Name() string {
	return e.name
}

func (e *treeNode) Mode() ObjectMode {
	return e.mode
}

func (e *treeNode) Hash() Hash {
	return e.hash
}

// decodes the octal form of a object mode into one of the valid Mode* values.
func parseObjectMode(data []byte) (ObjectMode, error) {
	mode, err := strconv.ParseUint(string(data), 8, 32)
	if err != nil {
		return 0, err
	}
	switch ObjectMode(mode) {
	case ModeFile:
	case ModeExe:
	case ModeDir:
	case ModeSymlink:
	case ModeSubmodule:
	default:
		return 0, fmt.Errorf("unknown object mode: %o", mode)
	}
	return ObjectMode(mode), nil
}
