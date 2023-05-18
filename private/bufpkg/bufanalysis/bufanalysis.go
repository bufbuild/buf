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

package bufanalysis

import (
	"crypto/sha256"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

const (
	// FormatText is the text format for FileAnnotations.
	FormatText Format = iota + 1
	// FormatJSON is the JSON format for FileAnnotations.
	FormatJSON
	// FormatMSVS is the MSVS format for FileAnnotations.
	FormatMSVS
	// FormatJUnit is the JUnit format for FileAnnotations.
	FormatJUnit
	// FormatGithubActions is the Github Actions format for FileAnnotations.
	//
	// See https://docs.github.com/en/actions/using-workflows/workflow-commands-for-github-actions#setting-an-error-message.
	FormatGithubActions
)

var (
	// AllFormatStrings is all format strings without aliases.
	//
	// Sorted in the order we want to display them.
	AllFormatStrings = []string{
		"text",
		"json",
		"msvs",
		"junit",
		"github-actions",
	}
	// AllFormatStringsWithAliases is all format strings with aliases.
	//
	// Sorted in the order we want to display them.
	AllFormatStringsWithAliases = []string{
		"text",
		"gcc",
		"json",
		"msvs",
		"junit",
		"github-actions",
	}

	stringToFormat = map[string]Format{
		"text": FormatText,
		// alias for text
		"gcc":            FormatText,
		"json":           FormatJSON,
		"msvs":           FormatMSVS,
		"junit":          FormatJUnit,
		"github-actions": FormatGithubActions,
	}
	formatToString = map[Format]string{
		FormatText:          "text",
		FormatJSON:          "json",
		FormatMSVS:          "msvs",
		FormatJUnit:         "junit",
		FormatGithubActions: "github-actions",
	}
)

// Format is a FileAnnotation format.
type Format int

// String implements fmt.Stringer.
func (f Format) String() string {
	s, ok := formatToString[f]
	if !ok {
		return strconv.Itoa(int(f))
	}
	return s
}

// ParseFormat parses the Format.
//
// The empty strings defaults to FormatText.
func ParseFormat(s string) (Format, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return FormatText, nil
	}
	f, ok := stringToFormat[s]
	if ok {
		return f, nil
	}
	return 0, fmt.Errorf("unknown format: %q", s)
}

// FileInfo is a minimal FileInfo interface.
type FileInfo interface {
	Path() string
	ExternalPath() string
}

// FileAnnotation is a file annotation.
type FileAnnotation interface {
	// Stringer returns the string representation of this annotation.
	fmt.Stringer

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
//	ExternalPath
//	StartLine
//	StartColumn
//	Type
//	Message
//	EndLine
//	EndColumn
func SortFileAnnotations(fileAnnotations []FileAnnotation) {
	sort.Stable(sortFileAnnotations(fileAnnotations))
}

// DeduplicateAndSortFileAnnotations deduplicates the FileAnnotations based on their
// string representation and sorts them according to the order specified in SortFileAnnotations.
func DeduplicateAndSortFileAnnotations(fileAnnotations []FileAnnotation) []FileAnnotation {
	deduplicated := make([]FileAnnotation, 0, len(fileAnnotations))
	seen := make(map[string]struct{}, len(fileAnnotations))
	for _, fileAnnotation := range fileAnnotations {
		key := hash(fileAnnotation)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		deduplicated = append(deduplicated, fileAnnotation)
	}
	SortFileAnnotations(deduplicated)
	return deduplicated
}

// PrintFileAnnotations prints the file annotations separated by newlines.
func PrintFileAnnotations(writer io.Writer, fileAnnotations []FileAnnotation, formatString string) error {
	format, err := ParseFormat(formatString)
	if err != nil {
		return err
	}

	switch format {
	case FormatText:
		return printAsText(writer, fileAnnotations)
	case FormatJSON:
		return printAsJSON(writer, fileAnnotations)
	case FormatMSVS:
		return printAsMSVS(writer, fileAnnotations)
	case FormatJUnit:
		return printAsJUnit(writer, fileAnnotations)
	case FormatGithubActions:
		return printAsGithubActions(writer, fileAnnotations)
	default:
		return fmt.Errorf("unknown FileAnnotation Format: %v", format)
	}
}

// hash returns a hash value that uniquely identifies the given FileAnnotation.
func hash(fileAnnotation FileAnnotation) string {
	path := ""
	if fileInfo := fileAnnotation.FileInfo(); fileInfo != nil {
		path = fileInfo.ExternalPath()
	}
	hash := sha256.New()
	_, _ = hash.Write([]byte(path))
	_, _ = hash.Write([]byte(strconv.Itoa(fileAnnotation.StartLine())))
	_, _ = hash.Write([]byte(strconv.Itoa(fileAnnotation.StartColumn())))
	_, _ = hash.Write([]byte(strconv.Itoa(fileAnnotation.EndLine())))
	_, _ = hash.Write([]byte(strconv.Itoa(fileAnnotation.EndColumn())))
	_, _ = hash.Write([]byte(fileAnnotation.Type()))
	_, _ = hash.Write([]byte(fileAnnotation.Message()))
	return string(hash.Sum(nil))
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
