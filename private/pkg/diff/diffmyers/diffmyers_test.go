// Copyright 2020-2026 Buf Technologies, Inc.
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
	"os"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/pkg/diff/diffmyers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const writeGoldenFiles = false

func TestDiff(t *testing.T) {
	t.Parallel()
	t.Run("delete-and-insert", func(t *testing.T) {
		t.Parallel()
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
		testPrint(t, from, to, edits, "delete-and-insert")
	})
	t.Run("insert-one", func(t *testing.T) {
		t.Parallel()
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
		testPrint(t, from, to, edits, "insert")
	})
	t.Run("delete-one", func(t *testing.T) {
		t.Parallel()
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
		testPrint(t, from, to, edits, "delete")
	})
	t.Run("create-file", func(t *testing.T) {
		t.Parallel()
		const from = ""
		const to = "Hello, world!\n"
		edits := diffmyers.Diff(
			splitLines(from),
			splitLines(to),
		)
		assert.Equal(t, edits, []diffmyers.Edit{
			{
				Kind:         diffmyers.EditKindInsert,
				FromPosition: 0,
				ToPosition:   0,
			},
		})
		testPrint(t, from, to, edits, "create")
	})
	t.Run("remove", func(t *testing.T) {
		t.Parallel()
		const from = "Hello, world!\n"
		const to = ""
		edits := diffmyers.Diff(
			splitLines(from),
			splitLines(to),
		)
		assert.Equal(t, edits, []diffmyers.Edit{
			{
				Kind:         diffmyers.EditKindDelete,
				FromPosition: 0,
			},
		})
		testPrint(t, from, to, edits, "remove")
	})
	t.Run("equal", func(t *testing.T) {
		t.Parallel()
		const from = "Hello, world!\n"
		const to = "Hello, world!\n"
		edits := diffmyers.Diff(
			splitLines(from),
			splitLines(to),
		)
		assert.Len(t, edits, 0)
		testPrint(t, from, to, edits, "equal")
	})
	// The example from https://www.gnu.org/software/diffutils/manual/html_node/Sample-diff-Input.html
	t.Run("lao-tzu", func(t *testing.T) {
		t.Parallel()
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
  they have different names.
`
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
The door of all subtleties!
`
		edits := diffmyers.Diff(
			splitLines(lao),
			splitLines(tzu),
		)
		assert.Equal(t,
			[]diffmyers.Edit{
				{
					Kind: diffmyers.EditKindDelete,
				},
				{
					Kind:         diffmyers.EditKindDelete,
					FromPosition: 1,
				},
				{
					Kind:         diffmyers.EditKindDelete,
					FromPosition: 3,
				},
				{
					Kind:         diffmyers.EditKindInsert,
					FromPosition: 4,
					ToPosition:   1,
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
			edits,
		)
		testPrint(t, lao, tzu, edits, "lao-tzu")
	})

	// The raw Myers script for these replacements places an insertion before a
	// deletion, and for the second case interleaves them. GNU diff and git
	// always emit a change block as deletions followed by insertions.
	t.Run("replace-one-line-with-two", func(t *testing.T) {
		t.Parallel()
		const from = "a\n"
		const to = "b\nb\n"
		edits := diffmyers.Diff(
			splitLines(from),
			splitLines(to),
		)
		assert.Equal(t, []diffmyers.Edit{
			{
				Kind: diffmyers.EditKindDelete,
			},
			{
				Kind:         diffmyers.EditKindInsert,
				FromPosition: 1,
			},
			{
				Kind:         diffmyers.EditKindInsert,
				FromPosition: 1,
				ToPosition:   1,
			},
		}, edits)
		testPrint(t, from, to, edits, "replace-one-line-with-two")
	})

	t.Run("replace-one-line-with-four", func(t *testing.T) {
		t.Parallel()
		const from = "a\n"
		const to = `b
b
b
b
`
		edits := diffmyers.Diff(
			splitLines(from),
			splitLines(to),
		)
		assert.Equal(t, []diffmyers.Edit{
			{
				Kind: diffmyers.EditKindDelete,
			},
			{
				Kind:         diffmyers.EditKindInsert,
				FromPosition: 1,
			},
			{
				Kind:         diffmyers.EditKindInsert,
				FromPosition: 1,
				ToPosition:   1,
			},
			{
				Kind:         diffmyers.EditKindInsert,
				FromPosition: 1,
				ToPosition:   2,
			},
			{
				Kind:         diffmyers.EditKindInsert,
				FromPosition: 1,
				ToPosition:   3,
			},
		}, edits)
		testPrint(t, from, to, edits, "replace-one-line-with-four")
	})

	// Without compaction the deletion run below is split by a line that could
	// have been part of it, producing "-b b -b -b" instead of " b -b -b -b".
	t.Run("coalesce-deletion-run", func(t *testing.T) {
		t.Parallel()
		const from = `a
b
b
b
b
a
d
b
`
		const to = `c
c
a
b
d
`
		edits := diffmyers.Diff(
			splitLines(from),
			splitLines(to),
		)
		testPrint(t, from, to, edits, "coalesce-deletion-run")
	})

	// Without compaction an appended block is attributed to the closing brace
	// of the preceding block, so the diff reads as "+}" followed by the new
	// message and stops before the final brace.
	t.Run("appended-block-boundary", func(t *testing.T) {
		t.Parallel()
		const from = `message A {
  int32 x = 1;
}
`
		const to = `message A {
  int32 x = 1;
}

message B {
  int32 y = 1;
}
`
		edits := diffmyers.Diff(
			splitLines(from),
			splitLines(to),
		)
		testPrint(t, from, to, edits, "appended-block-boundary")
	})

	// Two changes four unchanged lines apart belong to one hunk, as they would
	// under diff -U3 and git. Seven lines apart they do not.
	t.Run("merge-nearby-changes", func(t *testing.T) {
		t.Parallel()
		const from = `a
m
m
m
m
z
`
		const to = `A
m
m
m
m
Z
`
		edits := diffmyers.Diff(
			splitLines(from),
			splitLines(to),
		)
		testPrint(t, from, to, edits, "merge-nearby-changes")
	})

	t.Run("split-distant-changes", func(t *testing.T) {
		t.Parallel()
		const from = `a
m
m
m
m
m
m
m
z
`
		const to = `A
m
m
m
m
m
m
m
Z
`
		edits := diffmyers.Diff(
			splitLines(from),
			splitLines(to),
		)
		testPrint(t, from, to, edits, "split-distant-changes")
	})

	// A sequence whose final line is not newline terminated is recorded with
	// the same marker GNU diff and git use. Without it the two lines below are
	// indistinguishable in the output.
	t.Run("no-newline-at-end-of-from", func(t *testing.T) {
		t.Parallel()
		const from = "a\nb"
		const to = "a\nb\n"
		edits := diffmyers.Diff(
			splitLines(from),
			splitLines(to),
		)
		testPrint(t, from, to, edits, "no-newline-at-end-of-from")
	})

	t.Run("no-newline-at-end-of-to", func(t *testing.T) {
		t.Parallel()
		const from = "a\nb\n"
		const to = "a\nb"
		edits := diffmyers.Diff(
			splitLines(from),
			splitLines(to),
		)
		testPrint(t, from, to, edits, "no-newline-at-end-of-to")
	})

	// These need the block to move earlier and absorb the one it meets;
	// sliding later alone leaves the run split by a carried over line.
	t.Run("merge-deletion-run-backwards", func(t *testing.T) {
		t.Parallel()
		const from = `a
b
b
`
		const to = "b\n"
		edits := diffmyers.Diff(
			splitLines(from),
			splitLines(to),
		)
		testPrint(t, from, to, edits, "merge-deletion-run-backwards")
	})

	t.Run("merge-insertion-run-backwards", func(t *testing.T) {
		t.Parallel()
		const from = "a\n"
		const to = `b
a
a
`
		edits := diffmyers.Diff(
			splitLines(from),
			splitLines(to),
		)
		testPrint(t, from, to, edits, "merge-insertion-run-backwards")
	})

	// An empty range is numbered with the line before it, so this insertion
	// after old line 8 is recorded as -8,0 rather than -9,0.
	t.Run("insert-away-from-other-changes", func(t *testing.T) {
		t.Parallel()
		const from = `l1
l2
l3
l4
l5
l6
l7
l8
l9
l10
`
		const to = `l1
l2
l3
l4
l5
l6
l7
l8
NEW
l9
l10
`
		edits := diffmyers.Diff(
			splitLines(from),
			splitLines(to),
		)
		testPrint(t, from, to, edits, "insert-away-from-other-changes")
	})

	// Full context keeps every line, so the whole original sequence can be
	// recovered from the output.
	t.Run("full-context", func(t *testing.T) {
		t.Parallel()
		const from = `a
m
m
m
m
m
m
m
z
`
		const to = `A
m
m
m
m
m
m
m
Z
`
		edits := diffmyers.Diff(
			splitLines(from),
			splitLines(to),
		)
		testPrint(t, from, to, edits, "full-context", diffmyers.PrintWithFullContext())
	})

	// A narrower window keeps fewer carried over lines and splits hunks sooner.
	t.Run("context-width-one", func(t *testing.T) {
		t.Parallel()
		const from = `a
m
m
m
m
z
`
		const to = `A
m
m
m
m
Z
`
		edits := diffmyers.Diff(
			splitLines(from),
			splitLines(to),
		)
		testPrint(t, from, to, edits, "context-width-one", diffmyers.PrintWithContext(1))
	})

	// The marker belongs to the line it follows, so a final line outside the
	// window takes it with it rather than widening the hunk.
	t.Run("no-newline-outside-context", func(t *testing.T) {
		t.Parallel()
		const from = `a
b
c
d
e
f
g
h
i
j`
		const to = `a
b
c
d
e
F
g
h
i
j`
		edits := diffmyers.Diff(
			splitLines(from),
			splitLines(to),
		)
		testPrint(t, from, to, edits, "no-newline-outside-context")
	})

	t.Run("first-line-prefix", func(t *testing.T) {
		t.Parallel()
		from := `syntax = "proto3";

package test;

message Foo {
  string field1 = 1;
  string field2 = 2;
  string field3 = 3;
  string field4 = 4;
  string field5 = 5;
}
`
		to := `syntax = "proto3";

package test;

message Foo {
  string field1 = 1;
  string field2 = 2;
  string field3 = 3;
  string field4 = 4;
  int32 field5 = 5;
}
`
		expectedFirstLineOfOutput := " syntax = \"proto3\";"
		fromLines := splitLines(from)
		toLines := splitLines(to)
		edits := diffmyers.Diff(
			fromLines,
			toLines,
		)
		diff, err := diffmyers.Print(
			fromLines,
			toLines,
			edits,
			diffmyers.PrintWithFullContext(),
		)
		require.NoError(t, err)
		before, _, _ := bytes.Cut(diff, []byte("\n"))

		firstLine := before
		actualFirstLine := string(firstLine)
		require.Equal(t, expectedFirstLineOfOutput, actualFirstLine,
			"First line of diff output should match expected format (single space prefix, no double space)")
		testPrint(t, from, to, edits, "first-line-prefix", diffmyers.PrintWithFullContext())
	})
}

func TestPrintDoesNotModifyInput(t *testing.T) {
	t.Parallel()
	// The final line is deliberately not newline terminated, which is the case
	// Print has to normalize.
	from := [][]byte{[]byte("Hello, world!\n"), []byte("Goodbye, world!")}
	to := [][]byte{[]byte("Hello, world!\n")}
	before := make([][]byte, len(from))
	for i, line := range from {
		before[i] = bytes.Clone(line)
	}
	_, err := diffmyers.Print(from, to, diffmyers.Diff(from, to))
	require.NoError(t, err)
	assert.Equal(t, before, from, "Print must not modify the sequences it is given")
}

func TestPrintEmptyLine(t *testing.T) {
	t.Parallel()
	// An empty final line is not produced by splitLines, but Print is exported
	// and must not panic on one. Both sequences below hold the same bytes, so
	// the reported deletion and missing newline are artifacts of that input.
	// This pins the behavior rather than endorsing it.
	from := [][]byte{[]byte("Hello, world!\n"), {}}
	to := [][]byte{[]byte("Hello, world!\n")}
	diff, err := diffmyers.Print(from, to, diffmyers.Diff(from, to))
	require.NoError(t, err)
	assert.Equal(
		t,
		`@@ -1,2 +1,1 @@
 Hello, world!
-
\ No newline at end of file
`,
		string(diff),
	)
}

func TestPrintContextBounds(t *testing.T) {
	t.Parallel()
	const from = `a
b
c
d
e
f
g
`
	const to = `a
b
c
X
e
f
g
`
	edits := diffmyers.Diff(splitLines(from), splitLines(to))
	zero, err := diffmyers.Print(
		splitLines(from),
		splitLines(to),
		edits,
		diffmyers.PrintWithContext(0),
	)
	require.NoError(t, err)
	assert.Equal(t, `@@ -4,1 +4,1 @@
-d
+X
`, string(zero))
	// A negative window is treated as zero rather than slicing out of range.
	negative, err := diffmyers.Print(
		splitLines(from),
		splitLines(to),
		edits,
		diffmyers.PrintWithContext(-1),
	)
	require.NoError(t, err)
	assert.Equal(t, string(zero), string(negative))
}

func testPrint(
	t *testing.T,
	from, to string,
	edits []diffmyers.Edit,
	golden string,
	options ...diffmyers.PrintOption,
) {
	t.Run("print", func(t *testing.T) {
		diff, err := diffmyers.Print(
			splitLines(from),
			splitLines(to),
			edits,
			options...,
		)
		require.NoError(t, err)
		goldenFilePath := filepath.Join("testdata", golden)
		if writeGoldenFiles {
			require.NoError(t, os.WriteFile(goldenFilePath, diff, 0600))
		}
		diffGolden, err := os.ReadFile(goldenFilePath)
		require.NoError(t, err)
		assert.Equal(t, string(diff), string(diffGolden))
	})
}

func splitLines(s string) [][]byte {
	lines := bytes.SplitAfter([]byte(s), []byte("\n"))
	if len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}
	return lines
}
