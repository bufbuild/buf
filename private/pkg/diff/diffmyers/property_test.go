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

package diffmyers

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDiffAppliesCleanly(t *testing.T) {
	t.Parallel()
	forEachSequencePair(t, func(t *testing.T, from, to string) {
		fromLines, toLines := splitLines(from), splitLines(to)
		edits := Diff(fromLines, toLines)
		require.Equal(
			t,
			to,
			applyEdits(fromLines, toLines, edits),
			"applying the edit script did not produce the new sequence\nfrom: %q\nto:   %q",
			from,
			to,
		)
	})
}

func TestDiffIsMinimal(t *testing.T) {
	t.Parallel()
	forEachSequencePair(t, func(t *testing.T, from, to string) {
		fromLines, toLines := splitLines(from), splitLines(to)
		edits := Diff(fromLines, toLines)
		require.Len(
			t,
			edits,
			editDistance(fromLines, toLines),
			"edit script is not minimal\nfrom: %q\nto:   %q",
			from,
			to,
		)
	})
}

func TestDiffIsCompacted(t *testing.T) {
	t.Parallel()
	forEachSequencePair(t, func(t *testing.T, from, to string) {
		fromLines, toLines := splitLines(from), splitLines(to)
		edits := Diff(fromLines, toLines)
		require.Equal(
			t,
			edits,
			compactChangeBlocks(fromLines, toLines, edits),
			"compacting the edit script again changed it\nfrom: %q\nto:   %q",
			from,
			to,
		)
	})
}

// forEachSequencePair calls check with every ordered pair of sequences in
// each corpus.
func forEachSequencePair(t *testing.T, check func(t *testing.T, from, to string)) {
	t.Helper()
	for _, corpus := range []struct {
		name     string
		alphabet string
		maxLines int
	}{
		{name: "two-symbols", alphabet: "ab", maxLines: 7},
		{name: "three-symbols", alphabet: "abc", maxLines: 4},
	} {
		t.Run(corpus.name, func(t *testing.T) {
			t.Parallel()
			sequences := allSequences(corpus.alphabet, corpus.maxLines)
			for _, from := range sequences {
				for _, to := range sequences {
					check(t, from, to)
				}
			}
		})
	}
}

// allSequences returns every newline-terminated sequence over alphabet with at
// most maxLines lines, shortest first.
func allSequences(alphabet string, maxLines int) []string {
	sequences := []string{""}
	prefixes := []string{""}
	for range maxLines {
		var next []string
		for _, prefix := range prefixes {
			for _, symbol := range []byte(alphabet) {
				var builder strings.Builder
				builder.WriteString(prefix)
				builder.WriteByte(symbol)
				builder.WriteByte('\n')
				next = append(next, builder.String())
			}
		}
		sequences = append(sequences, next...)
		prefixes = next
	}
	return sequences
}

func splitLines(s string) [][]byte {
	lines := bytes.SplitAfter([]byte(s), []byte("\n"))
	if len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// applyEdits applies the specified edits, returning the result as a string.
func applyEdits(from, to [][]byte, edits []Edit) string {
	var buffer bytes.Buffer
	fromIndex := 0
	for _, edit := range edits {
		for fromIndex < edit.FromPosition {
			buffer.Write(from[fromIndex])
			fromIndex++
		}
		switch edit.Kind {
		case EditKindDelete:
			fromIndex++
		case EditKindInsert:
			buffer.Write(to[edit.ToPosition])
		}
	}
	for _, line := range from[fromIndex:] {
		buffer.Write(line)
	}
	return buffer.String()
}

// editDistance returns the minimum number of insertions and deletions that turn
// from into to.
func editDistance(from, to [][]byte) int {
	previous := make([]int, len(to)+1)
	current := make([]int, len(to)+1)
	for toIndex := range previous {
		previous[toIndex] = toIndex
	}
	for fromIndex := 1; fromIndex <= len(from); fromIndex++ {
		current[0] = fromIndex
		for toIndex := 1; toIndex <= len(to); toIndex++ {
			if bytes.Equal(from[fromIndex-1], to[toIndex-1]) {
				current[toIndex] = previous[toIndex-1]
			} else {
				current[toIndex] = min(previous[toIndex], current[toIndex-1]) + 1
			}
		}
		previous, current = current, previous
	}
	return previous[len(to)]
}
