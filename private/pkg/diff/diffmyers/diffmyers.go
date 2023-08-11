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

package diffmyers

import (
	"bytes"
	"errors"
	"fmt"
)

// EditKind is the kind of edit.
type EditKind int

const (
	// EditKindDelete is a delete.
	EditKindDelete EditKind = iota + 1
	// EditKindInsert is an insert.
	EditKindInsert
)

// Edit is an delete or insert operation.
type Edit struct {
	// Kind is the kind of edit. It is either an insert or a delete.
	Kind EditKind
	// FromPosition is the line to edit in the original sequence.
	FromPosition int
	// ToPosition is the line in the new sequence. It is only valid for
	// inserts.
	ToPosition int
}

// Diff does a diff. It returns a [[]Edit] which when applied to the original
// sequence will result in the new sequence.
//
// The algorithm is based on the paper "An O(ND) Difference Algorithm and Its
// Variations" by Eugene W. Myers. The paper is available at https://citeseerx.ist.psu.edu/doc/10.1.1.4.6927.
//
// It implements the linear space refinement of the algorithm described in section 4b. This is the
// same algorithm used by git.
func Diff(from, to [][]byte) []Edit {
	return shortestEdits(from, to, 0, 0)
}

// Print prints the edits in the unified diff format without the header.
//
// Ref: https://www.gnu.org/software/diffutils/manual/html_node/Detailed-Unified.html
func Print(from, to [][]byte, edits []Edit) ([]byte, error) {
	const contextThreshold = 2
	type printLine struct {
		EditKind EditKind
		line     []byte
		hunk     bool
	}
	// If the last line of from is not a newline append one.
	if len(from) > 0 && from[len(from)-1] != nil {
		last := from[len(from)-1]
		if last[len(last)-1] != '\n' {
			from[len(from)-1] = append(last, '\n')
		}
	}
	// We preallocate the slice to avoid reallocations.
	//
	// Each edit is either a delete or an insert so the total number of lines
	// in the diff is the number of edits plus the number of lines in the
	// original sequence. The worst case for the hunk headers are
	// as many edits.
	out := make([]*printLine, 0, len(from)+2*len(edits))
	var fromIndex, toIndex, bufferSize int
	for i := 0; i < len(edits); i++ {
		// Remember the start of the hunk. We add 1 to the indexes because
		// we want to print the line number and they start at 1.
		hunkOldStart := fromIndex + 1
		hunkNewStart := toIndex + 1
		// Reserve the space for the hunk header.
		hunk := &printLine{hunk: true}
		out = append(out, hunk)
		var (
			insertCount, deleteCount int
			printHunk                bool
		)
		// Print the lines in the edit.
		for j := i; j < len(edits); j++ {
			// Print the lines before the edit.
			var advance int
			for _, line := range from[fromIndex:edits[i].FromPosition] {
				out = append(out, &printLine{line: line})
				bufferSize += len(line) + 1
				advance++
			}
			// Advance the indexes.
			toIndex += advance
			fromIndex += advance
			insertCount += advance
			deleteCount += advance
			if advance > contextThreshold {
				i--
				break
			}
			printHunk = true
			switch edits[j].Kind {
			case EditKindDelete:
				deleteCount++
				fromIndex++
				out = append(out, &printLine{
					EditKind: EditKindDelete,
					line:     from[edits[j].FromPosition],
				})
			case EditKindInsert:
				insertCount++
				toIndex++
				out = append(out, &printLine{
					EditKind: EditKindInsert,
					line:     to[edits[j].ToPosition],
				})
			default:
				return nil, errors.New("unknown edit kind")
			}
			bufferSize += len(out[len(out)-1].line) + 1
			i++
		}
		if printHunk {
			// Print the hunk header.
			hunk.line = []byte(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", hunkOldStart, deleteCount, hunkNewStart, insertCount))
			bufferSize += len(hunk.line) + 1
		}
	}
	// Print the lines after the last edit.
	for _, line := range from[fromIndex:] {
		out = append(out, &printLine{line: line})
		bufferSize += len(line) + 1
	}
	var buffer bytes.Buffer
	buffer.Grow(bufferSize)
	for _, line := range out {
		if line.hunk && len(line.line) > 0 {
			buffer.Write(line.line)
			continue
		}
		switch line.EditKind {
		case EditKindDelete:
			buffer.WriteByte('-')
		case EditKindInsert:
			buffer.WriteByte('+')
		default:
			buffer.WriteByte(' ')
		}
		buffer.Write(line.line)
	}
	return buffer.Bytes(), nil
}

func shortestEdits(from, to [][]byte, fromOffset, toOffset int) []Edit {
	n, m := len(from), len(to)
	if m == 0 { // We've reached the end of the 'to' sequence. So delete the rest of the 'from' sequence.
		edits := make([]Edit, len(from))
		for i := range from {
			edits[i] = Edit{
				Kind:         EditKindDelete,
				FromPosition: fromOffset + i,
			}
		}
		return edits
	}
	if n == 0 { // We've reached the end of the 'from' sequence. So insert the rest of the 'to' sequence.
		edits := make([]Edit, len(to))
		for i := range to {
			edits[i] = Edit{
				Kind:         EditKindInsert,
				FromPosition: fromOffset,
				ToPosition:   toOffset + i,
			}
		}
		return edits
	}
	d, x, y, u, v := findMiddleSnake(from, to)
	if d > 1 || x != u && y != v {
		return append(shortestEdits(from[:x], to[:y], fromOffset, toOffset), shortestEdits(from[u:], to[v:], fromOffset+u, toOffset+v)...)
	}
	if m > n {
		return shortestEdits(nil, to[n:m], fromOffset+n, toOffset+n)
	}
	if m < n {
		return shortestEdits(from[m:n], nil, fromOffset+m, toOffset+m)
	}
	return nil
}

// returns the length, starting and ending points of the middle snake.
//
// This is based on the pseudo code in page 11. This deliberately deviates from
// the style of using descriptive variables names to ease comparison with the
// pseudo code and variable names in the paper.
func findMiddleSnake(from, to [][]byte) (d int, x int, y int, u int, v int) {
	n, m := len(from), len(to)
	maxD := ceiledHalf(n + m)
	// We need to allocate 2*maxD+1 because k can go from -maxD to maxD.
	// Wherever we access them we just offset by maxD.
	vf := make([]int, 2*maxD+1)
	vb := make([]int, 2*maxD+1)
	for i := 0; i < len(vf); i++ {
		vf[i] = -1
		vb[i] = -1
	}
	vf[1+maxD] = 0
	vb[1+maxD] = 0
	delta := n - m
	for d := 0; d <= maxD; d++ {
		for k := -d; k <= d; k += 2 { // Forward snake
			var x int
			// We prefer deletions over insertions.
			if k == -d || (k != d && vf[k-1+maxD] < vf[k+1+maxD]) {
				x = vf[k+1+maxD]
			} else {
				x = vf[k-1+maxD] + 1
			}
			y := x - k
			// Initial point
			xi := x
			yi := y
			// Move diagonally as far as possible.
			for x < n && y < m && bytes.Equal(from[x], to[y]) {
				x++
				y++
			}
			vf[k+maxD] = x
			if (delta&1 == 1) && -(k-delta) >= -(d-1) && -(k-delta) <= (d-1) && vb[(-(k-delta))+maxD] != -1 {
				if x+vb[(-(k-delta))+maxD] >= n {
					return 2*d - 1, xi, yi, x, y
				}
			}
		}
		for k := -d; k <= d; k += 2 { // Backward snake
			var x int
			if k == -d || (k != d && vb[k-1+maxD] < vb[k+1+maxD]) {
				x = vb[k+1+maxD]
			} else {
				x = vb[k-1+maxD] + 1
			}
			y := x - k
			xi := x
			yi := y
			for x < n && y < m && bytes.Equal(from[n-x-1], to[m-y-1]) {
				x++
				y++
			}
			vb[k+maxD] = x
			if (delta&1 == 0) && -(k-delta) >= -d && -(k-delta) <= d && vf[(-(k-delta))+maxD] != -1 {
				if x+vf[(-(k-delta))+maxD] >= n {
					return 2 * d, n - x, m - y, n - xi, m - yi
				}
			}
		}
	}
	return -1, -1, -1, -1, -1
}

func ceiledHalf(n int) int {
	if n%2 == 0 {
		return n / 2
	}
	return n/2 + 1
}
