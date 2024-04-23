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
	"bytes"
	"strconv"
)

type fileAnnotation struct {
	fileInfo    FileInfo
	startLine   int
	startColumn int
	endLine     int
	endColumn   int
	typeString  string
	message     string
}

func newFileAnnotation(
	fileInfo FileInfo,
	startLine int,
	startColumn int,
	endLine int,
	endColumn int,
	typeString string,
	message string,
) *fileAnnotation {
	return &fileAnnotation{
		fileInfo:    fileInfo,
		startLine:   startLine,
		startColumn: startColumn,
		endLine:     endLine,
		endColumn:   endColumn,
		typeString:  typeString,
		message:     message,
	}
}

func (f *fileAnnotation) FileInfo() FileInfo {
	return f.fileInfo
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
	line := atLeast1(f.startLine)
	column := atLeast1(f.startColumn)
	message := f.message
	if f.fileInfo != nil {
		path = f.fileInfo.ExternalPath()
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
	_, _ = buffer.WriteString(strconv.Itoa(line))
	_, _ = buffer.WriteRune(':')
	_, _ = buffer.WriteString(strconv.Itoa(column))
	_, _ = buffer.WriteRune(':')
	_, _ = buffer.WriteString(message)
	return buffer.String()
}

func (*fileAnnotation) isFileAnnotation() {}
