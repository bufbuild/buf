// Copyright 2020-2022 Buf Technologies, Inc.
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
	"io"
	"strconv"
)

func textPrinter(writer io.Writer, fileAnnotations []FileAnnotation) error {
	return printEachAnnotationOnNewLine(
		writer,
		fileAnnotations,
		printFileAnnotationAsText,
	)
}

func msvsPrinter(writer io.Writer, fileAnnotations []FileAnnotation) error {
	return printEachAnnotationOnNewLine(
		writer,
		fileAnnotations,
		printFileAnnotationAsMSVS,
	)
}

func jsonPrinter(writer io.Writer, fileAnnotations []FileAnnotation) error {
	return printEachAnnotationOnNewLine(
		writer,
		fileAnnotations,
		printFileAnnotationAsJSON,
	)
}

func printFileAnnotationAsText(buffer *bytes.Buffer, f FileAnnotation) error {
	_, _ = buffer.WriteString(f.String())
	return nil
}

func printFileAnnotationAsMSVS(buffer *bytes.Buffer, f FileAnnotation) error {
	// This will work as long as f != (*fileAnnotation)(nil)
	if f == nil {
		return nil
	}
	path := "<input>"
	line := f.StartLine()
	column := f.StartColumn()
	message := f.Message()
	if f.FileInfo() != nil {
		path = f.FileInfo().ExternalPath()
	}
	if line == 0 {
		line = 1
	}
	typeString := f.Type()
	if typeString == "" {
		// should never happen but just in case
		typeString = "FAILURE"
	}
	if message == "" {
		message = f.Type()
		// should never happen but just in case
		if message == "" {
			message = "FAILURE"
		}
	}
	_, _ = buffer.WriteString(path)
	_, _ = buffer.WriteRune('(')
	_, _ = buffer.WriteString(strconv.Itoa(line))
	if column != 0 {
		_, _ = buffer.WriteRune(',')
		_, _ = buffer.WriteString(strconv.Itoa(column))
	}
	_, _ = buffer.WriteString(") : error ")
	_, _ = buffer.WriteString(typeString)
	_, _ = buffer.WriteString(" : ")
	_, _ = buffer.WriteString(message)
	return nil
}

func printFileAnnotationAsJSON(buffer *bytes.Buffer, f FileAnnotation) error {
	data, err := json.Marshal(newExternalFileAnnotation(f))
	if err != nil {
		return err
	}
	_, _ = buffer.Write(data)
	return nil
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

func newExternalFileAnnotation(f FileAnnotation) externalFileAnnotation {
	path := ""
	if f.FileInfo() != nil {
		path = f.FileInfo().ExternalPath()
	}
	return externalFileAnnotation{
		Path:        path,
		StartLine:   f.StartLine(),
		StartColumn: f.StartColumn(),
		EndLine:     f.EndLine(),
		EndColumn:   f.EndColumn(),
		Type:        f.Type(),
		Message:     f.Message(),
	}
}

func printEachAnnotationOnNewLine(
	writer io.Writer,
	fileAnnotations []FileAnnotation,
	fileAnnotationPrinter func(writer *bytes.Buffer, fileAnnotation FileAnnotation) error,
) error {
	buffer := bytes.NewBuffer(nil)
	for _, fileAnnotation := range fileAnnotations {
		buffer.Reset()
		if err := fileAnnotationPrinter(buffer, fileAnnotation); err != nil {
			return err
		}
		_, _ = buffer.WriteString("\n")
		if _, err := writer.Write(buffer.Bytes()); err != nil {
			return err
		}
	}
	return nil
}
