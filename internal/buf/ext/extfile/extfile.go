// Copyright 2020 Buf Technologies Inc.
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

package extfile

import (
	"bytes"
	"io"
	"sort"
	"strconv"

	filev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/file/v1beta1"
	"github.com/bufbuild/buf/internal/pkg/proto/protoencoding"
)

// FileAnnotationToString returns the basic string representation of the FileAnnotation.
func FileAnnotationToString(fileAnnotation *filev1beta1.FileAnnotation) string {
	path := fileAnnotation.GetPath()
	line := fileAnnotation.GetStartLine()
	column := fileAnnotation.GetStartColumn()
	message := fileAnnotation.GetMessage()
	if path == "" {
		path = "<input>"
	}
	if line == 0 {
		line = 1
	}
	if column == 0 {
		column = 1
	}
	// should never happen but just in case
	if message == "" {
		message = fileAnnotation.Type
		if message == "" {
			message = "FAILURE"
		}
	}
	buffer := bytes.NewBuffer(nil)
	_, _ = buffer.WriteString(path)
	_, _ = buffer.WriteRune(':')
	_, _ = buffer.WriteString(strconv.Itoa(int(line)))
	_, _ = buffer.WriteRune(':')
	_, _ = buffer.WriteString(strconv.Itoa(int(column)))
	_, _ = buffer.WriteRune(':')
	_, _ = buffer.WriteString(message)
	return buffer.String()
}

// SortFileAnnotations sorts the FileAnnotations.
//
// The order of sorting is:
//
//   Path
//   StartLine
//   StartColumn
//   Type
//   Message
//   EndLine
//   EndColumn
func SortFileAnnotations(fileAnnotations []*filev1beta1.FileAnnotation) {
	sort.Stable(sortFileAnnotations(fileAnnotations))
}

type sortFileAnnotations []*filev1beta1.FileAnnotation

func (a sortFileAnnotations) Len() int          { return len(a) }
func (a sortFileAnnotations) Swap(i int, j int) { a[i], a[j] = a[j], a[i] }
func (a sortFileAnnotations) Less(i int, j int) bool {
	if a[i] == nil && a[j] == nil {
		return false
	}
	if a[i] == nil && a[j] != nil {
		return true
	}
	if a[i] != nil && a[j] == nil {
		return false
	}
	if a[i].Path < a[j].Path {
		return true
	}
	if a[i].Path > a[j].Path {
		return false
	}
	if a[i].StartLine < a[j].StartLine {
		return true
	}
	if a[i].StartLine > a[j].StartLine {
		return false
	}
	if a[i].StartColumn < a[j].StartColumn {
		return true
	}
	if a[i].StartColumn > a[j].StartColumn {
		return false
	}
	if a[i].Type < a[j].Type {
		return true
	}
	if a[i].Type > a[j].Type {
		return false
	}
	if a[i].Message < a[j].Message {
		return true
	}
	if a[i].Message > a[j].Message {
		return false
	}
	if a[i].EndLine < a[j].EndLine {
		return true
	}
	if a[i].EndLine > a[j].EndLine {
		return false
	}
	if a[i].EndColumn < a[j].EndColumn {
		return true
	}
	return false
}

// PrintFileAnnotations prints the FileAnnotations to the Writer.
//
// If asJSON is specified, the FileAnnotations are marshalled as JSON.
func PrintFileAnnotations(writer io.Writer, fileAnnotations []*filev1beta1.FileAnnotation, asJSON bool) error {
	if len(fileAnnotations) == 0 {
		return nil
	}
	values := make([][]byte, 0, len(fileAnnotations))
	if asJSON {
		// we do not use extensions so we do not need fileDescriptorProtos
		marshaler := protoencoding.NewJSONMarshalerUseProtoNames(nil)
		for _, fileAnnotation := range fileAnnotations {
			data, err := marshaler.Marshal(fileAnnotation)
			if err != nil {
				return err
			}
			values = append(values, data)
		}
	} else {
		for _, fileAnnotation := range fileAnnotations {
			values = append(values, []byte(FileAnnotationToString(fileAnnotation)))
		}
	}
	for _, value := range values {
		if _, err := writer.Write(value); err != nil {
			return err
		}
		if _, err := writer.Write([]byte{'\n'}); err != nil {
			return err
		}
	}
	return nil
}

// ResolveFileAnnotationPaths attempts to resolve file paths using the given resolver function.
//
// If the resolver is nil, this does nothing.
// If the resolver function returns an empty string for a given path, no modifications are made.
func ResolveFileAnnotationPaths(resolver func(string) (string, error), fileAnnotations ...*filev1beta1.FileAnnotation) error {
	if resolver == nil {
		return nil
	}
	if len(fileAnnotations) == 0 {
		return nil
	}
	for _, fileAnnotation := range fileAnnotations {
		if err := resolveFileAnnotationPath(resolver, fileAnnotation); err != nil {
			return err
		}
	}
	return nil
}

func resolveFileAnnotationPath(resolver func(string) (string, error), fileAnnotation *filev1beta1.FileAnnotation) error {
	if fileAnnotation.Path == "" {
		return nil
	}
	filePath, err := resolver(fileAnnotation.Path)
	if err != nil {
		return err
	}
	if filePath != "" {
		fileAnnotation.Path = filePath
	}
	return nil
}
