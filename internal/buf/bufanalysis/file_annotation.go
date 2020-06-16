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
	"bytes"
	"encoding/json"
	"strconv"

	"github.com/bufbuild/buf/internal/buf/bufimage"
)

type fileAnnotation struct {
	fileRef     bufimage.FileRef
	startLine   int
	startColumn int
	endLine     int
	endColumn   int
	typeString  string
	message     string
}

func newFileAnnotation(
	fileRef bufimage.FileRef,
	startLine int,
	startColumn int,
	endLine int,
	endColumn int,
	typeString string,
	message string,
) *fileAnnotation {
	return &fileAnnotation{
		fileRef:     fileRef,
		startLine:   startLine,
		startColumn: startColumn,
		endLine:     endLine,
		endColumn:   endColumn,
		typeString:  typeString,
		message:     message,
	}
}

func (f *fileAnnotation) FileRef() bufimage.FileRef {
	return f.fileRef
}

func (f *fileAnnotation) StartLine() int {
	return f.startLine
}

func (f *fileAnnotation) StartColumn() int {
	return f.startColumn
}

func (f *fileAnnotation) EndLine() int {
	return f.endLine
}

func (f *fileAnnotation) EndColumn() int {
	return f.endColumn
}

func (f *fileAnnotation) Type() string {
	return f.typeString
}

func (f *fileAnnotation) Message() string {
	return f.message
}

func (f *fileAnnotation) String() string {
	if f == nil {
		return ""
	}
	path := "<input>"
	line := f.startLine
	column := f.startColumn
	message := f.message
	if f.fileRef != nil {
		path = f.fileRef.ExternalFilePath()
	}
	if line == 0 {
		line = 1
	}
	if column == 0 {
		column = 1
	}
	if message == "" {
		message = f.typeString
		// should never happen but just in case
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

func (f *fileAnnotation) MarshalJSON() ([]byte, error) {
	if f == nil {
		return nil, nil
	}
	return json.Marshal(f.toExternalFileAnnotation())
}

func (f *fileAnnotation) toExternalFileAnnotation() externalFileAnnotation {
	path := ""
	if f.fileRef != nil {
		path = f.fileRef.ExternalFilePath()
	}
	return externalFileAnnotation{
		Path:        path,
		StartLine:   f.startLine,
		StartColumn: f.startColumn,
		EndLine:     f.endLine,
		EndColumn:   f.endColumn,
		Type:        f.typeString,
		Message:     f.message,
	}
}

type externalFileAnnotation struct {
	Path        string `json:"path,omitempty" yaml:"path,omitempty"`
	StartLine   int    `json:"start_line,omitempty" yaml:"start_line,omitempty"`
	StartColumn int    `json:"start_column,omitempty" yaml:"start_column,omitempty"`
	EndLine     int    `json:"end_line,omitempty" yaml:"end_line,omitempty"`
	EndColumn   int    `json:"end_column,omitempty" yaml:"end_column,omitempty"`
	Type        string `json:"type,omitempty" yaml:"type,omitempty"`
	Message     string `json:"message,omitempty" yaml:"message,omitempty"`
}
