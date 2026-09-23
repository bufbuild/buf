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

const (
	// defaultContext is the default context window, matching `diff -u` and git.
	defaultContext = 3
	// noNewlineMarker records that the line above it was not newline terminated.
	noNewlineMarker = "\\ No newline at end of file\n"
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
//
// The script is then normalized similar to GNU diff and git normalize theirs.
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

// PrintOption is an option for Print.
type PrintOption func(*printOptions)

// PrintWithContext sets the number of carried over lines kept either side of a
// change. The default is 3, as for diff -u and git. A negative value is treated
// as zero.
//
// Mutually exclusive with PrintWithFullContext - the last one wins.
func PrintWithContext(context int) PrintOption {
	return func(printOptions *printOptions) {
		printOptions.context = max(context, 0)
		printOptions.fullContext = false
	}
}

// PrintWithFullContext keeps every line of the original sequence rather than a
// window around each change.
//
// Mutually exclusive with PrintWithContext - the last one wins.
//
// The result is not a valid unified diff: the lines before the first change
// precede every hunk header, so patch and git apply reject it. In exchange the
// whole original sequence can be recovered from the output, which a caller
// that re-slices the context itself needs.
func PrintWithFullContext() PrintOption {
	return func(printOptions *printOptions) {
		printOptions.fullContext = true
	}
}

type printOptions struct {
	context     int
	fullContext bool
}

// Print prints the edits in the unified diff format without the header.
//
// Ref: https://www.gnu.org/software/diffutils/manual/html_node/Detailed-Unified.html
func Print(from, to [][]byte, edits []Edit, options ...PrintOption) ([]byte, error) {
	resolved := &printOptions{context: defaultContext, fullContext: false}
	for _, option := range options {
		option(resolved)
	}
	lines, err := diffLines(from, to, edits)
	if err != nil {
		return nil, err
	}
	context := min(resolved.context, len(lines))
	if resolved.fullContext {
		return emitFullContext(lines, context), nil
	}
	return emitHunks(lines, context), nil
}

// printLine is one line of the diff body. A zero EditKind is a line carried
// over from both sequences. The line is the caller's slice, so a final line
// that is not newline terminated is still missing its terminator here.
type printLine struct {
	EditKind EditKind
	line     []byte
}

// emitHunks writes a unified diff carrying at most context carried over lines
// either side of each change. Changes separated by at most twice the context
// share a hunk, which is where diff -u and git stop merging.
func emitHunks(lines []printLine, context int) []byte {
	var buffer bytes.Buffer
	oldLine, newLine := 1, 1
	emitted := 0
	for index := 0; index < len(lines); {
		if lines[index].EditKind == 0 {
			index++
			continue
		}
		// Take every change that is close enough to share this hunk.
		end := index
		for {
			for end < len(lines) && lines[end].EditKind != 0 {
				end++
			}
			carried := end
			for carried < len(lines) && lines[carried].EditKind == 0 {
				carried++
			}
			if carried >= len(lines) || carried-end > 2*context {
				break
			}
			end = carried
		}
		start := max(index-context, emitted)
		stop := min(end+context, len(lines))
		skippedOld, skippedNew := countLines(lines[emitted:start])
		oldLine += skippedOld
		newLine += skippedNew
		region := lines[start:stop]
		oldCount, newCount := countLines(region)
		buffer.Write(hunkHeader(oldLine, oldCount, newLine, newCount))
		writeLines(&buffer, region)
		oldLine += oldCount
		newLine += newCount
		emitted = stop
		index = stop
	}
	return buffer.Bytes()
}

// diffLines applies the edit script to produce the body of the diff, one entry
// per line, with no hunk headers.
func diffLines(from, to [][]byte, edits []Edit) ([]printLine, error) {
	lines := make([]printLine, 0, len(from)+len(edits))
	fromIndex := 0
	for _, edit := range edits {
		if err := validateEdit(edit, len(from), len(to)); err != nil {
			return nil, err
		}
		for fromIndex < edit.FromPosition {
			lines = append(lines, printLine{line: from[fromIndex]})
			fromIndex++
		}
		switch edit.Kind {
		case EditKindDelete:
			lines = append(lines, printLine{
				EditKind: EditKindDelete,
				line:     from[edit.FromPosition],
			})
			fromIndex++
		case EditKindInsert:
			lines = append(lines, printLine{
				EditKind: EditKindInsert,
				line:     to[edit.ToPosition],
			})
		}
	}
	for fromIndex < len(from) {
		lines = append(lines, printLine{line: from[fromIndex]})
		fromIndex++
	}
	return lines, nil
}

func validateEdit(edit Edit, fromLines, toLines int) error {
	switch edit.Kind {
	case EditKindDelete:
		if edit.FromPosition < 0 || edit.FromPosition >= fromLines {
			return fmt.Errorf(
				"delete position %d out of range for %d lines",
				edit.FromPosition,
				fromLines,
			)
		}
	case EditKindInsert:
		if edit.FromPosition < 0 || edit.FromPosition > fromLines {
			return fmt.Errorf(
				"insert position %d out of range for %d lines",
				edit.FromPosition,
				fromLines,
			)
		}
		if edit.ToPosition < 0 || edit.ToPosition >= toLines {
			return fmt.Errorf(
				"insert position %d out of range for %d lines",
				edit.ToPosition,
				toLines,
			)
		}
	default:
		return errors.New("unknown edit kind")
	}
	return nil
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
		oldCount, newCount := countLines(region)
		if containsChange(region) {
			buffer.Write(hunkHeader(oldLine, oldCount, newLine, newCount))
		}
		writeLines(&buffer, region)
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
		if !isNewlineTerminated(line.line) {
			size += 1 + len(noNewlineMarker)
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
		if !isNewlineTerminated(line.line) {
			// Only a sequence's final line can lack its terminator. Supply it
			// and record that it was missing.
			buffer.WriteByte('\n')
			buffer.WriteString(noNewlineMarker)
		}
	}
}

func isNewlineTerminated(line []byte) bool {
	return len(line) > 0 && line[len(line)-1] == '\n'
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
		fromRun := newChangeRun(from, fromChanged, fromIndex)
		toRun := newChangeRun(to, toChanged, toIndex)
		if fromRun.isEmpty() && toRun.isEmpty() {
			// A pair of carried over lines.
			fromIndex++
			toIndex++
			continue
		}
		// A block is bounded by a pair of carried over lines or by the ends of
		// both sequences, so one side's bound speaks for both.
		for fromRun.start > 0 && fromRun.canSlideUp() && toRun.canSlideUp() {
			fromRun.slideUp()
			toRun.slideUp()
		}
		for fromRun.end < len(from) && fromRun.canSlideDown() && toRun.canSlideDown() {
			fromRun.slideDown()
			toRun.slideDown()
		}
		fromIndex, toIndex = fromRun.end, toRun.end
	}
	return editsFromChanged(fromChanged, toChanged, len(edits))
}

// changeRun is one side of a change block: the changed lines [start, end) of
// one sequence. A block's deleted lines form its run in the original sequence
// and its inserted lines its run in the new one. Either run may be empty, but
// not both.
type changeRun struct {
	lines   [][]byte
	changed []bool
	start   int
	end     int
}

func newChangeRun(lines [][]byte, changed []bool, start int) changeRun {
	end := start
	for end < len(changed) && changed[end] {
		end++
	}
	return changeRun{lines: lines, changed: changed, start: start, end: end}
}

func (r *changeRun) isEmpty() bool {
	return r.start == r.end
}

// canSlideUp reports whether the run can move one line earlier and describe
// the same edit: the line joining its front must equal the one leaving its
// back, so the carried over lines are unchanged. The caller ensures there is a
// line before the run.
func (r *changeRun) canSlideUp() bool {
	return r.isEmpty() || bytes.Equal(r.lines[r.start-1], r.lines[r.end-1])
}

// slideUp moves the run one line earlier, absorbing any run it meets.
func (r *changeRun) slideUp() {
	if !r.isEmpty() {
		r.changed[r.start-1], r.changed[r.end-1] = true, false
	}
	r.start--
	r.end--
	for r.start > 0 && r.changed[r.start-1] {
		r.start--
	}
}

// canSlideDown reports whether the run can move one line later and describe
// the same edit: the line leaving its front must equal the one joining its
// back. The caller ensures there is a line after the run.
func (r *changeRun) canSlideDown() bool {
	return r.isEmpty() || bytes.Equal(r.lines[r.start], r.lines[r.end])
}

// slideDown moves the run one line later, absorbing any run it meets.
func (r *changeRun) slideDown() {
	if !r.isEmpty() {
		r.changed[r.start], r.changed[r.end] = false, true
	}
	r.start++
	r.end++
	for r.end < len(r.changed) && r.changed[r.end] {
		r.end++
	}
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
