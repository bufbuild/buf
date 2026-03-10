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

// Package diff implements diffing.
//
// Should primarily be used for testing.
package diff

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"

	zdiff "znkr.io/diff"
	"znkr.io/diff/textdiff"
)

// Diff does a diff.
//
// Returns nil if no diff.
func Diff(
	ctx context.Context,
	b1 []byte,
	b2 []byte,
	filename1 string,
	filename2 string,
	options ...DiffOption,
) ([]byte, error) {
	diffOptions := newDiffOptions()
	for _, option := range options {
		option(diffOptions)
	}
	return doDiff(b1, b2, filename1, filename2, diffOptions.suppressCommands)
}

// DiffOption is an option for Diff.
type DiffOption func(*diffOptions)

// DiffWithSuppressCommands returns a new DiffOption that suppresses printing of commands.
func DiffWithSuppressCommands() DiffOption {
	return func(diffOptions *diffOptions) {
		diffOptions.suppressCommands = true
	}
}

// DiffWithSuppressTimestamps returns a new DiffOption that suppresses printing of timestamps.
func DiffWithSuppressTimestamps() DiffOption {
	return func(*diffOptions) {}
}

func doDiff(
	b1 []byte,
	b2 []byte,
	filename1 string,
	filename2 string,
	suppressCommands bool,
) ([]byte, error) {
	if bytes.Equal(b1, b2) {
		return nil, nil
	}
	hunks := textdiff.Hunks(b1, b2)
	if len(hunks) == 0 {
		return nil, nil
	}
	// Always print filepath with slash separator.
	filename1 = filepath.ToSlash(filename1)
	filename2 = filepath.ToSlash(filename2)
	if filename1 == filename2 {
		filename1 = filename1 + ".orig"
	}
	var buf bytes.Buffer
	if !suppressCommands {
		fmt.Fprintf(&buf, "diff -u %s %s\n", filename1, filename2)
	}
	fmt.Fprintf(&buf, "--- %s\n", filename1)
	fmt.Fprintf(&buf, "+++ %s\n", filename2)
	for _, h := range hunks {
		fmt.Fprintf(&buf, "@@ -%s +%s @@\n",
			hunkRange(h.LineNoX, h.EndLineNoX),
			hunkRange(h.LineNoY, h.EndLineNoY),
		)
		for _, e := range h.Edits {
			switch e.Op {
			case zdiff.Match:
				fmt.Fprintf(&buf, " %s", e.Line)
			case zdiff.Delete:
				fmt.Fprintf(&buf, "-%s", e.Line)
			case zdiff.Insert:
				fmt.Fprintf(&buf, "+%s", e.Line)
			}
		}
	}
	return buf.Bytes(), nil
}

// hunkRange formats a hunk range for a unified diff header, matching GNU diff -u output:
// - count 0: "l,0" using the 0-based line number (line before which the change occurs)
// - count 1: "l" using 1-based line number, count omitted
// - count N: "l,N" using 1-based line number
func hunkRange(lineNo, endLine int) string {
	count := endLine - lineNo
	switch count {
	case 0:
		return fmt.Sprintf("%d,0", lineNo)
	case 1:
		return fmt.Sprintf("%d", lineNo+1)
	default:
		return fmt.Sprintf("%d,%d", lineNo+1, count)
	}
}

type diffOptions struct {
	suppressCommands bool
}

func newDiffOptions() *diffOptions {
	return &diffOptions{}
}
