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
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func printAsText(writer io.Writer, fileAnnotations []FileAnnotation) error {
	return printEachAnnotationOnNewLine(
		writer,
		fileAnnotations,
		printFileAnnotationAsText,
	)
}

func printAsMSVS(writer io.Writer, fileAnnotations []FileAnnotation) error {
	return printEachAnnotationOnNewLine(
		writer,
		fileAnnotations,
		printFileAnnotationAsMSVS,
	)
}

func printAsJSON(writer io.Writer, fileAnnotations []FileAnnotation) error {
	return printEachAnnotationOnNewLine(
		writer,
		fileAnnotations,
		printFileAnnotationAsJSON,
	)
}

func printAsJUnit(writer io.Writer, fileAnnotations []FileAnnotation) error {
	encoder := xml.NewEncoder(writer)
	encoder.Indent("", "  ")
	testsuites := xml.StartElement{Name: xml.Name{Local: "testsuites"}}
	err := encoder.EncodeToken(testsuites)
	if err != nil {
		return err
	}
	annotationsByPath := groupAnnotationsByPath(fileAnnotations)
	for _, annotations := range annotationsByPath {
		path := "<input>"
		if fileInfo := annotations[0].FileInfo(); fileInfo != nil {
			path = fileInfo.ExternalPath()
		}
		path = strings.TrimSuffix(path, ".proto")
		testsuite := xml.StartElement{
			Name: xml.Name{Local: "testsuite"},
			Attr: []xml.Attr{
				{Name: xml.Name{Local: "name"}, Value: path},
				{Name: xml.Name{Local: "tests"}, Value: strconv.Itoa(len(annotations))},
				{Name: xml.Name{Local: "failures"}, Value: strconv.Itoa(len(annotations))},
				{Name: xml.Name{Local: "errors"}, Value: "0"},
			},
		}
		if err := encoder.EncodeToken(testsuite); err != nil {
			return err
		}
		for _, annotation := range annotations {
			if err := printFileAnnotationAsJUnit(encoder, annotation); err != nil {
				return err
			}
		}
		if err := encoder.EncodeToken(xml.EndElement{Name: testsuite.Name}); err != nil {
			return err
		}
	}
	if err := encoder.EncodeToken(xml.EndElement{Name: testsuites.Name}); err != nil {
		return err
	}
	if err := encoder.Flush(); err != nil {
		return err
	}
	if _, err := writer.Write([]byte("\n")); err != nil {
		return err
	}
	return nil
}

func printFileAnnotationAsJUnit(encoder *xml.Encoder, annotation FileAnnotation) error {
	testcase := xml.StartElement{Name: xml.Name{Local: "testcase"}}
	name := annotation.Type()
	if annotation.StartColumn() != 0 {
		name += fmt.Sprintf("_%d_%d", annotation.StartLine(), annotation.StartColumn())
	} else if annotation.StartLine() != 0 {
		name += fmt.Sprintf("_%d", annotation.StartLine())
	}
	testcase.Attr = append(testcase.Attr, xml.Attr{Name: xml.Name{Local: "name"}, Value: name})
	if err := encoder.EncodeToken(testcase); err != nil {
		return err
	}
	failure := xml.StartElement{
		Name: xml.Name{Local: "failure"},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "message"}, Value: annotation.String()},
			{Name: xml.Name{Local: "type"}, Value: annotation.Type()},
		},
	}
	if err := encoder.EncodeToken(failure); err != nil {
		return err
	}
	if err := encoder.EncodeToken(xml.EndElement{Name: failure.Name}); err != nil {
		return err
	}
	if err := encoder.EncodeToken(xml.EndElement{Name: testcase.Name}); err != nil {
		return err
	}
	return nil
}

func groupAnnotationsByPath(annotations []FileAnnotation) [][]FileAnnotation {
	pathToIndex := make(map[string]int)
	annotationsByPath := make([][]FileAnnotation, 0)
	for _, annotation := range annotations {
		path := "<input>"
		if fileInfo := annotation.FileInfo(); fileInfo != nil {
			path = fileInfo.ExternalPath()
		}
		index, ok := pathToIndex[path]
		if !ok {
			index = len(annotationsByPath)
			pathToIndex[path] = index
			annotationsByPath = append(annotationsByPath, nil)
		}
		annotationsByPath[index] = append(annotationsByPath[index], annotation)
	}
	return annotationsByPath
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
