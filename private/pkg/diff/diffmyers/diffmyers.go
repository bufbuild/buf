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
	// A sub-problem never needs more diagonals than the whole problem, so one
	// pair of arrays serves the entire recursion.
	maxD := ceiledHalf(len(from) + len(to))
	search := &snakeSearch{
		forward:  make([]int, 2*maxD+1),
		backward: make([]int, 2*maxD+1),
	}
	return compactChangeBlocks(from, to, search.shortestEdits(from, to, 0, 0))
}

// snakeSearch holds the furthest reaching path arrays that findMiddleSnake
// works in, so that the recursion allocates once rather than at every level.
type snakeSearch struct {
	forward  []int
	backward []int
}

// noNewlineMarker records that the line above it was not newline terminated in
// the sequence it came from. It is not a line of either sequence, so it is not
// counted in the hunk header.
const noNewlineMarker = "\\ No newline at end of file\n"

// defaultContext is the number of carried over lines kept around a change,
// matching diff -u and git.
const defaultContext = 3

// printLine is one line of the diff body. A zero EditKind is a line carried
// over from both sequences.
type printLine struct {
	EditKind  EditKind
	line      []byte
	noNewline bool
}

// Print prints the edits in the unified diff format without the header.
//
// Ref: https://www.gnu.org/software/diffutils/manual/html_node/Detailed-Unified.html
func Print(from, to [][]byte, edits []Edit) ([]byte, error) {
	lines, err := diffLines(from, to, edits)
	if err != nil {
		return nil, err
	}
	return emitFullContext(lines, defaultContext), nil
}

// diffLines applies the edit script to produce the body of the diff, one entry
// per line, with no hunk headers.
func diffLines(from, to [][]byte, edits []Edit) ([]printLine, error) {
	// A sequence's final line may not be newline terminated. Supply the
	// terminator when the line is read, rather than writing it back into the
	// caller's slice, and report the fact so that it can be recorded in the
	// output.
	lineAt := func(lines [][]byte, index int) ([]byte, bool) {
		line := lines[index]
		if index != len(lines)-1 || (len(line) > 0 && line[len(line)-1] == '\n') {
			return line, false
		}
		return append(bytes.Clone(line), '\n'), true
	}
	lines := make([]printLine, 0, len(from)+len(edits))
	fromIndex := 0
	for _, edit := range edits {
		for fromIndex < edit.FromPosition {
			line, noNewline := lineAt(from, fromIndex)
			lines = append(lines, printLine{line: line, noNewline: noNewline})
			fromIndex++
		}
		switch edit.Kind {
		case EditKindDelete:
			line, noNewline := lineAt(from, edit.FromPosition)
			lines = append(lines, printLine{
				EditKind:  EditKindDelete,
				line:      line,
				noNewline: noNewline,
			})
			fromIndex++
		case EditKindInsert:
			line, noNewline := lineAt(to, edit.ToPosition)
			lines = append(lines, printLine{
				EditKind:  EditKindInsert,
				line:      line,
				noNewline: noNewline,
			})
		default:
			return nil, errors.New("unknown edit kind")
		}
	}
	for fromIndex < len(from) {
		line, noNewline := lineAt(from, fromIndex)
		lines = append(lines, printLine{line: line, noNewline: noNewline})
		fromIndex++
	}
	return lines, nil
}

// emitFullContext writes every line of the diff, placing a hunk header before
// each run of changes. Hunks are split by a run of carried over lines longer
// than twice the context, which is where diff -u and git stop merging.
//
// The result carries the whole original sequence, so a consumer can rebuild it
// from the output, but for the same reason any lines before the first change
// precede every header and it is not a valid unified diff.
func emitFullContext(lines []printLine, context int) []byte {
	var buffer bytes.Buffer
	buffer.Grow(bufferSizeFor(lines))
	oldLine, newLine := 1, 1
	for index := 0; index < len(lines); {
		end := hunkEnd(lines, index, context)
		region := lines[index:end]
		if containsChange(region) {
			oldCount, newCount := countLines(region)
			buffer.Write(hunkHeader(oldLine, oldCount, newLine, newCount))
		}
		writeLines(&buffer, region)
		oldCount, newCount := countLines(region)
		oldLine += oldCount
		newLine += newCount
		index = end
	}
	return buffer.Bytes()
}

// hunkEnd returns the index just past the hunk that starts at index. A hunk
// takes carried over lines and changes in turn and ends once it has taken a
// run of carried over lines longer than twice the context.
func hunkEnd(lines []printLine, index, context int) int {
	for {
		carried := index
		for carried < len(lines) && lines[carried].EditKind == 0 {
			carried++
		}
		run := carried - index
		index = carried
		if run > 2*context || index >= len(lines) {
			return index
		}
		for index < len(lines) && lines[index].EditKind != 0 {
			index++
		}
		if index >= len(lines) {
			return index
		}
	}
}

func containsChange(lines []printLine) bool {
	for _, line := range lines {
		if line.EditKind != 0 {
			return true
		}
	}
	return false
}

func countLines(lines []printLine) (int, int) {
	var oldCount, newCount int
	for _, line := range lines {
		switch line.EditKind {
		case EditKindDelete:
			oldCount++
		case EditKindInsert:
			newCount++
		default:
			oldCount++
			newCount++
		}
	}
	return oldCount, newCount
}

func bufferSizeFor(lines []printLine) int {
	size := 0
	for _, line := range lines {
		size += len(line.line) + 1
		if line.noNewline {
			size += len(noNewlineMarker)
		}
	}
	return size
}

func writeLines(buffer *bytes.Buffer, lines []printLine) {
	for _, line := range lines {
		switch line.EditKind {
		case EditKindDelete:
			buffer.WriteByte('-')
		case EditKindInsert:
			buffer.WriteByte('+')
		default:
			buffer.WriteByte(' ')
		}
		buffer.Write(line.line)
		if line.noNewline {
			buffer.WriteString(noNewlineMarker)
		}
	}
}

// changeBlock is a run of deleted lines in the original sequence together with
// the run of inserted lines that replaces them. Either run may be empty, but
// not both.
type changeBlock struct {
	fromStart, fromEnd int
	toStart, toEnd     int
}

// compactChangeBlocks shifts every change block as far as it can go and
// rewrites the script so that each block's deletions precede its insertions.
//
// The Myers bisection places a block wherever the middle snake falls, which is
// decided by content elsewhere in the file, so one replacement lands
// differently in different parts of a file and a run gets split by a line that
// could have been part of it. GNU diff and git both normalize this first.
//
// A block moves as early as it can go, absorbing any block it meets, and then
// as late as the merged block can go. Absorbing is the part that matters: two
// runs separated by one carried over line are one run, and only once joined
// does where they belong have an answer.
//
// git also scores candidates by indentation (XDF_INDENT_HEURISTIC) rather than
// always taking the last. That is not implemented here.
func compactChangeBlocks(from, to [][]byte, edits []Edit) []Edit {
	if len(edits) == 0 {
		return edits
	}
	// A line is either carried over or it is not, so the whole script is
	// captured by one flag per line. Shifting a block is then a matter of
	// moving flags, which keeps the two sequences in step by construction.
	fromChanged := make([]bool, len(from))
	toChanged := make([]bool, len(to))
	for _, edit := range edits {
		switch edit.Kind {
		case EditKindDelete:
			fromChanged[edit.FromPosition] = true
		case EditKindInsert:
			toChanged[edit.ToPosition] = true
		}
	}
	fromIndex, toIndex := 0, 0
	for fromIndex < len(from) || toIndex < len(to) {
		if !(fromIndex < len(from) && fromChanged[fromIndex]) &&
			!(toIndex < len(to) && toChanged[toIndex]) {
			fromIndex++
			toIndex++
			continue
		}
		block := changeBlock{
			fromStart: fromIndex,
			fromEnd:   fromIndex,
			toStart:   toIndex,
			toEnd:     toIndex,
		}
		for block.fromEnd < len(from) && fromChanged[block.fromEnd] {
			block.fromEnd++
		}
		for block.toEnd < len(to) && toChanged[block.toEnd] {
			block.toEnd++
		}
		for block.slideUp(from, to, fromChanged, toChanged) {
		}
		for block.slideDown(from, to, fromChanged, toChanged) {
		}
		fromIndex, toIndex = block.fromEnd, block.toEnd
	}
	return editsFromChanged(fromChanged, toChanged, len(edits))
}

// slideUp moves the block one line earlier if that describes the same edit,
// reporting whether it moved, absorbing any block the move makes adjacent.
//
// The move is safe when each non-empty side's last line equals the line just
// before its run: the line joining the front is then the one leaving the back.
func (b *changeBlock) slideUp(from, to [][]byte, fromChanged, toChanged []bool) bool {
	// There has to be a carried over line on both sides to move past.
	if b.fromStart <= 0 || fromChanged[b.fromStart-1] {
		return false
	}
	if b.toStart <= 0 || toChanged[b.toStart-1] {
		return false
	}
	if b.fromEnd > b.fromStart && !bytes.Equal(from[b.fromStart-1], from[b.fromEnd-1]) {
		return false
	}
	if b.toEnd > b.toStart && !bytes.Equal(to[b.toStart-1], to[b.toEnd-1]) {
		return false
	}
	if b.fromEnd > b.fromStart {
		fromChanged[b.fromStart-1], fromChanged[b.fromEnd-1] = true, false
	}
	if b.toEnd > b.toStart {
		toChanged[b.toStart-1], toChanged[b.toEnd-1] = true, false
	}
	b.fromStart--
	b.fromEnd--
	b.toStart--
	b.toEnd--
	for b.fromStart > 0 && fromChanged[b.fromStart-1] {
		b.fromStart--
	}
	for b.toStart > 0 && toChanged[b.toStart-1] {
		b.toStart--
	}
	return true
}

// slideDown moves the block one line later if that describes the same edit,
// reporting whether it moved. Any block the move makes adjacent is absorbed.
//
// The block can move past the following pair of carried over lines when each
// non-empty side's first line equals the line just after that side's run: the
// line leaving the front of the run is then identical to the one joining the
// back, so the sequences are unchanged.
func (b *changeBlock) slideDown(from, to [][]byte, fromChanged, toChanged []bool) bool {
	// There has to be a carried over line on both sides to move past.
	if b.fromEnd >= len(from) || fromChanged[b.fromEnd] {
		return false
	}
	if b.toEnd >= len(to) || toChanged[b.toEnd] {
		return false
	}
	if b.fromEnd > b.fromStart && !bytes.Equal(from[b.fromStart], from[b.fromEnd]) {
		return false
	}
	if b.toEnd > b.toStart && !bytes.Equal(to[b.toStart], to[b.toEnd]) {
		return false
	}
	if b.fromEnd > b.fromStart {
		fromChanged[b.fromStart], fromChanged[b.fromEnd] = false, true
	}
	if b.toEnd > b.toStart {
		toChanged[b.toStart], toChanged[b.toEnd] = false, true
	}
	b.fromStart++
	b.fromEnd++
	b.toStart++
	b.toEnd++
	for b.fromEnd < len(from) && fromChanged[b.fromEnd] {
		b.fromEnd++
	}
	for b.toEnd < len(to) && toChanged[b.toEnd] {
		b.toEnd++
	}
	return true
}

// editsFromChanged rebuilds an edit script from per line flags. Each block's
// deletions are emitted before its insertions, which is the shape GNU diff and
// git always produce.
func editsFromChanged(fromChanged, toChanged []bool, size int) []Edit {
	edits := make([]Edit, 0, size)
	fromIndex, toIndex := 0, 0
	for fromIndex < len(fromChanged) || toIndex < len(toChanged) {
		blockStart := len(edits)
		for fromIndex < len(fromChanged) && fromChanged[fromIndex] {
			edits = append(edits, Edit{
				Kind:         EditKindDelete,
				FromPosition: fromIndex,
			})
			fromIndex++
		}
		for toIndex < len(toChanged) && toChanged[toIndex] {
			edits = append(edits, Edit{
				Kind:         EditKindInsert,
				FromPosition: fromIndex,
				ToPosition:   toIndex,
			})
			toIndex++
		}
		if len(edits) == blockStart {
			// A pair of carried over lines.
			fromIndex++
			toIndex++
		}
	}
	return edits
}

// hunkHeader formats a hunk header. An empty range is numbered with the line
// before it, which is zero when the range starts the file. GNU patch reads
// that number, and placing an insertion after the following line instead puts
// it one line too late.
func hunkHeader(oldStart, oldCount, newStart, newCount int) []byte {
	if oldCount == 0 {
		oldStart--
	}
	if newCount == 0 {
		newStart--
	}
	return fmt.Appendf(nil, "@@ -%d,%d +%d,%d @@\n", oldStart, oldCount, newStart, newCount)
}

func (s *snakeSearch) shortestEdits(from, to [][]byte, fromOffset, toOffset int) []Edit {
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
	d, x, y, u, v := s.findMiddleSnake(from, to)
	if d > 1 || x != u && y != v {
		return append(s.shortestEdits(from[:x], to[:y], fromOffset, toOffset), s.shortestEdits(from[u:], to[v:], fromOffset+u, toOffset+v)...)
	}
	if m > n {
		return s.shortestEdits(nil, to[n:m], fromOffset+n, toOffset+n)
	}
	if m < n {
		return s.shortestEdits(from[m:n], nil, fromOffset+m, toOffset+m)
	}
	return nil
}

// returns the length, starting and ending points of the middle snake.
//
// This is based on the pseudo code in page 11. This deliberately deviates from
// the style of using descriptive variables names to ease comparison with the
// pseudo code and variable names in the paper.
func (s *snakeSearch) findMiddleSnake(from, to [][]byte) (d int, x int, y int, u int, v int) {
	n, m := len(from), len(to)
	maxD := ceiledHalf(n + m)
	// k goes from -maxD to maxD, so 2*maxD+1 entries are needed and every
	// access is offset by maxD. The arrays carry values from the previous
	// sub-problem and must be reset.
	vf := s.forward[:2*maxD+1]
	vb := s.backward[:2*maxD+1]
	for i := range vf {
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
