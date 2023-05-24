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
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTree(t *testing.T) {
	t.Parallel()

	/*
		This is generated using the following procedure:
		```sh
		➜ git init
		➜ touch .gitignore a.proto b
		➜ mkdir c && touch c/d.proto
		➜ git add * && git add --chmod +x b
		➜ git commit -m 'initial commit'
		```
		Then simply `git cat-file` the tree at HEAD and encode to base64.
	*/
	bytes, err := base64.StdEncoding.DecodeString("MTAwNjQ0IGEucHJvdG8A5p3im7LR1kNLiymud1rYwuSMU5ExMDA3NTUgYgDmneKbstHWQ0uLKa53WtjC5IxTkTQwMDAwIGMAXEw7X4b6IGAIGHO/LwaXPdE5gys=")
	require.NoError(t, err)
	hash, err := parseHashFromHex("43848150a6f5f6d76eeef6e0f69eb46290eefab6")
	require.NoError(t, err)

	tree, err := parseTree(hash, bytes)

	assert.NoError(t, err)
	assert.Equal(t, tree.Hash(), hash)
	assert.Len(t, tree.Nodes(), 3)
	assert.Equal(t, tree.Nodes()[0].Hash().Hex(), "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391")
	assert.Equal(t, tree.Nodes()[0].Name(), "a.proto")
	assert.Equal(t, tree.Nodes()[0].Mode(), ModeFile)
	assert.Equal(t, tree.Nodes()[1].Hash().Hex(), "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391")
	assert.Equal(t, tree.Nodes()[1].Name(), "b")
	assert.Equal(t, tree.Nodes()[1].Mode(), ModeExe)
	assert.Equal(t, tree.Nodes()[2].Hash().Hex(), "5c4c3b5f86fa2060081873bf2f06973dd139832b")
	assert.Equal(t, tree.Nodes()[2].Name(), "c")
	assert.Equal(t, tree.Nodes()[2].Mode(), ModeDir)
}
