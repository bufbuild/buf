// Copyright 2020-2024 Buf Technologies, Inc.
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
	"sort"
	"strconv"
	"strings"
)

type fileAnnotationSet struct {
	fileAnnotations []FileAnnotation
}

func newFileAnnotationSet(fileAnnotations []FileAnnotation) *fileAnnotationSet {
	if len(fileAnnotations) == 0 {
		return nil
	}
	return &fileAnnotationSet{
		fileAnnotations: deduplicateAndSortFileAnnotations(fileAnnotations),
	}
}

func (f *fileAnnotationSet) FileAnnotations() []FileAnnotation {
	return f.fileAnnotations
}

func (f *fileAnnotationSet) String() string {
	var sb strings.Builder
	for i, fileAnnotation := range f.fileAnnotations {
		_, _ = sb.WriteString(fileAnnotation.String())
		if i != len(f.fileAnnotations)-1 {
			_, _ = sb.WriteRune('\n')
		}
	}
	return sb.String()
}

func (f *fileAnnotationSet) Error() string {
	return f.String()
}

func (*fileAnnotationSet) isFileAnnotationSet() {}

// deduplicateAndSortFileAnnotations deduplicates the FileAnnotations based on their
// string representation and sorts them according to the order specified in SortFileAnnotations.
//
// This function makes a copy of the input FileAnnotations.
func deduplicateAndSortFileAnnotations(fileAnnotations []FileAnnotation) []FileAnnotation {
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
	sortFileAnnotations(deduplicated)
	return deduplicated
}

// sortFileAnnotations sorts the FileAnnotations.
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
func sortFileAnnotations(fileAnnotations []FileAnnotation) {
	sort.Stable(sortFileAnnotationSlice(fileAnnotations))
}

type sortFileAnnotationSlice []FileAnnotation

func (a sortFileAnnotationSlice) Len() int          { return len(a) }
func (a sortFileAnnotationSlice) Swap(i int, j int) { a[i], a[j] = a[j], a[i] }
func (a sortFileAnnotationSlice) Less(i int, j int) bool {
	return fileAnnotationCompareTo(a[i], a[j]) < 0
}

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
