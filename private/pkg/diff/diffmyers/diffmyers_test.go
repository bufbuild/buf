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

package diffmyers_test

import (
	"bytes"
	"testing"

	"github.com/bufbuild/buf/private/pkg/diff/diffmyers"
	"github.com/stretchr/testify/assert"
)

func TestDiff(t *testing.T) {
	t.Run("delete-and-insert", func(t *testing.T) {
		const from = "Hello, world!\n"
		const to = "Goodbye, world!\n"
		edits := diffmyers.Diff(
			splitLines(from),
			splitLines(to),
		)
		assert.Equal(t, edits, []diffmyers.Edit{
			{
				Kind: diffmyers.EditKindDelete,
			},
			{
				Kind:         diffmyers.EditKindInsert,
				FromPosition: 1,
			},
		})
	})
	t.Run("insert", func(t *testing.T) {
		const from = "Hello, world!\n"
		const to = "Hello, world!\nGoodbye, world!\n"
		edits := diffmyers.Diff(
			splitLines(from),
			splitLines(to),
		)
		assert.Equal(t, edits, []diffmyers.Edit{
			{
				Kind:         diffmyers.EditKindInsert,
				FromPosition: 1,
				ToPosition:   1,
			},
		})
	})
	t.Run("delete", func(t *testing.T) {
		const from = "Hello, world!\nGoodbye, world!\n"
		const to = "Hello, world!\n"
		edits := diffmyers.Diff(
			splitLines(from),
			splitLines(to),
		)
		assert.Equal(t, edits, []diffmyers.Edit{
			{
				Kind:         diffmyers.EditKindDelete,
				FromPosition: 1,
			},
		})
	})
	t.Run("equal", func(t *testing.T) {
		const from = "Hello, world!\n"
		const to = "Hello, world!\n"
		edits := diffmyers.Diff(
			splitLines(from),
			splitLines(to),
		)
		assert.Len(t, edits, 0)
	})
	// The example from https://www.gnu.org/software/diffutils/manual/html_node/Sample-diff-Input.html
	t.Run("lao-tzu", func(t *testing.T) {
		const lao = `The Way that can be told of is not the eternal Way;
The name that can be named is not the eternal name.
The Nameless is the origin of Heaven and Earth;
The Named is the mother of all things.
Therefore let there always be non-being,
  so we may see their subtlety,
And let there always be being,
  so we may see their outcome.
The two are the same,
But after they are produced,
  they have different names.`
		const tzu = `The Nameless is the origin of Heaven and Earth;
The named is the mother of all things.

Therefore let there always be non-being,
  so we may see their subtlety,
And let there always be being,
  so we may see their outcome.
The two are the same,
But after they are produced,
  they have different names.
They both may be called deep and profound.
Deeper and more profound,
The door of all subtleties!`
		edits := diffmyers.Diff(
			splitLines(lao),
			splitLines(tzu),
		)
		assert.Equal(t, edits,
			[]diffmyers.Edit{
				{
					Kind: diffmyers.EditKindDelete,
				},
				{
					Kind:         diffmyers.EditKindDelete,
					FromPosition: 1,
				},
				{
					Kind:         diffmyers.EditKindInsert,
					FromPosition: 3,
					ToPosition:   1,
				},
				{
					Kind:         diffmyers.EditKindDelete,
					FromPosition: 3,
				},
				{
					Kind:         diffmyers.EditKindInsert,
					FromPosition: 4,
					ToPosition:   2,
				},
				{
					Kind:         diffmyers.EditKindInsert,
					FromPosition: 11,
					ToPosition:   10,
				},
				{
					Kind:         diffmyers.EditKindInsert,
					FromPosition: 11,
					ToPosition:   11,
				},
				{
					Kind:         diffmyers.EditKindInsert,
					FromPosition: 11,
					ToPosition:   12,
				},
			},
		)
	})
}

func splitLines(s string) [][]byte {
	lines := bytes.SplitAfter([]byte(s), []byte("\n"))
	if len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}
	return lines
}
