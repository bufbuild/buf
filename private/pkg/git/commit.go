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
	"io"
	"strings"
)

type commit struct {
	hash      Hash
	tree      Hash
	parents   []Hash
	author    Ident
	committer Ident
	message   string
}

func (c *commit) Hash() Hash {
	return c.hash
}
func (c *commit) Tree() Hash {
	return c.tree
}
func (c *commit) Parents() []Hash {
	return c.parents
}
func (c *commit) Author() Ident {
	return c.author
}
func (c *commit) Committer() Ident {
	return c.committer
}
func (c *commit) Message() string {
	return c.message
}
func (c *commit) String() string {
	return c.author.Timestamp().String() + " " + c.hash.String()
}

func parseCommit(hash Hash, data []byte) (*commit, error) {
	c := &commit{
		hash: hash,
	}
	buffer := bytes.NewBuffer(data)
	line, err := buffer.ReadString('\n')
	for err != io.EOF && line != "\n" {
		header, value, _ := strings.Cut(line, " ")
		value = strings.TrimRight(value, "\n")
		switch header {
		case "tree":
			if c.tree != nil {
				return nil, errors.New("too many tree headers")
			}
			if c.tree, err = parseHashFromHex(value); err != nil {
				return nil, err
			}
		case "parent":
			if parent, err := parseHashFromHex(value); err != nil {
				return nil, err
			} else {
				c.parents = append(c.parents, parent)
			}
		case "author":
			if c.author, err = parseIdent([]byte(value)); err != nil {
				return nil, err
			}
		case "committer":
			if c.committer, err = parseIdent([]byte(value)); err != nil {
				return nil, err
			}
		default:
			// We do not parse the remaining headers.
		}
		line, err = buffer.ReadString('\n')
	}
	c.message = buffer.String()
	c.message = strings.TrimRight(c.message, "\n")
	return c, err
}
