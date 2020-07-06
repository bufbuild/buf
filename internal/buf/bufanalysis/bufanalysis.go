// Copyright 2020 Buf Technologies, Inc.
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

package bufanalysis

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/bufbuild/buf/internal/pkg/stringjson"
)

// FileInfo is a minimal FileInfo interface.
type FileInfo interface {
	Path() string
	ExternalPath() string
}

// FileAnnotation is a file annotation.
type FileAnnotation interface {
	fmt.Stringer
	json.Marshaler
	// FileInfo is the FileInfo for this annotation.
	//
	// This may be nil.
	FileInfo() FileInfo

	// StartLine is the starting line.
	//
	// If the starting line is not known, this will be 0.
	StartLine() int
	// StartColumn is the starting column.
	//
	// If the starting column is not known, this will be 0.
	StartColumn() int
	// EndLine is the ending line.
	//
	// If the ending line is not known, this will be 0.
	// If the ending line is the same as the starting line, this will be explicitly
	// set to the same value as start_line.
	EndLine() int
	// EndColumn is the ending column.
	//
	// If the ending column is not known, this will be 0.
	// If the ending column is the same as the starting column, this will be explicitly
	// set to the same value as start_column.
	EndColumn() int
	// Type is the type of annotation, typically an ID representing a failure type.
	Type() string
	// Message is the message of the annotation.
	Message() string
}

// NewFileAnnotation returns a new FileAnnotation.
func NewFileAnnotation(
	fileInfo FileInfo,
	startLine int,
	startColumn int,
	endLine int,
	endColumn int,
	typeString string,
	message string,
) FileAnnotation {
	return newFileAnnotation(
		fileInfo,
		startLine,
		startColumn,
		endLine,
		endColumn,
		typeString,
		message,
	)
}

// SortFileAnnotations sorts the FileAnnotations.
//
// The order of sorting is:
//
//   ExternalPath
//   StartLine
//   StartColumn
//   Type
//   Message
//   EndLine
//   EndColumn
func SortFileAnnotations(fileAnnotations []FileAnnotation) {
	sort.Stable(sortFileAnnotations(fileAnnotations))
}

// PrintFileAnnotations prints the file annotations separated by newlines.
func PrintFileAnnotations(writer io.Writer, fileAnnotations []FileAnnotation, asJSON bool) error {
	for _, fileAnnotation := range fileAnnotations {
		if err := stringjson.Println(writer, fileAnnotation, asJSON); err != nil {
			return err
		}
	}
	return nil
}

type sortFileAnnotations []FileAnnotation

func (a sortFileAnnotations) Len() int               { return len(a) }
func (a sortFileAnnotations) Swap(i int, j int)      { a[i], a[j] = a[j], a[i] }
func (a sortFileAnnotations) Less(i int, j int) bool { return fileAnnotationCompareTo(a[i], a[j]) < 0 }

// fileAnnotationCompareTo returns a value less than 0 if a < b, a value
// greater than 0 if a > b, and 0 if a == b.
func fileAnnotationCompareTo(a FileAnnotation, b FileAnnotation) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil && b != nil {
		return -1
	}
	if a != nil && b == nil {
		return 1
	}
	aFileInfo := a.FileInfo()
	bFileInfo := b.FileInfo()
	if aFileInfo == nil && bFileInfo != nil {
		return -1
	}
	if aFileInfo != nil && bFileInfo == nil {
		return 1
	}
	if aFileInfo != nil && bFileInfo != nil {
		if aFileInfo.ExternalPath() < bFileInfo.ExternalPath() {
			return -1
		}
		if aFileInfo.ExternalPath() > bFileInfo.ExternalPath() {
			return 1
		}
	}
	if a.StartLine() < b.StartLine() {
		return -1
	}
	if a.StartLine() > b.StartLine() {
		return 1
	}
	if a.StartColumn() < b.StartColumn() {
		return -1
	}
	if a.StartColumn() > b.StartColumn() {
		return 1
	}
	if a.Type() < b.Type() {
		return -1
	}
	if a.Type() > b.Type() {
		return 1
	}
	if a.Message() < b.Message() {
		return -1
	}
	if a.Message() > b.Message() {
		return 1
	}
	if a.EndLine() < b.EndLine() {
		return -1
	}
	if a.EndLine() > b.EndLine() {
		return 1
	}
	if a.EndColumn() < b.EndColumn() {
		return -1
	}
	if a.EndColumn() > b.EndColumn() {
		return 1
	}
	return 0
}
