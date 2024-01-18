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
	"io"
	"strings"
)

type annotatedTag struct {
	hash    Hash
	commit  Hash
	name    string
	tagger  Ident
	message string
}

func (t *annotatedTag) Hash() Hash {
	return t.hash
}
func (t *annotatedTag) Commit() Hash {
	return t.commit
}
func (t *annotatedTag) Name() string {
	return t.name
}
func (t *annotatedTag) Tagger() Ident {
	return t.tagger
}
func (t *annotatedTag) Message() string {
	return t.message
}

func parseAnnotatedTag(hash Hash, data []byte) (*annotatedTag, error) {
	t := &annotatedTag{
		hash: hash,
	}
	buffer := bytes.NewBuffer(data)
	line, err := buffer.ReadString('\n')
	for err != io.EOF && line != "\n" {
		header, value, _ := strings.Cut(line, " ")
		value = strings.TrimRight(value, "\n")
		switch header {
		case "object":
			if t.commit, err = parseHashFromHex(value); err != nil {
				return nil, err
			}
		case "tagger":
			if t.tagger, err = parseIdent([]byte(value)); err != nil {
				return nil, err
			}
		case "tag":
			t.name = value
		default:
			// We do not parse the remaining headers.
		}
		line, err = buffer.ReadString('\n')
	}
	t.message = buffer.String()
	t.message = strings.TrimRight(t.message, "\n")
	return t, err
}
