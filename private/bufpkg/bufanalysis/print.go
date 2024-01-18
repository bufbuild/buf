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

func printAsGithubActions(writer io.Writer, fileAnnotations []FileAnnotation) error {
	return printEachAnnotationOnNewLine(
		writer,
		fileAnnotations,
		printFileAnnotationAsGithubActions,
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
	line := atLeast1(f.StartLine())
	column := atLeast1(f.StartColumn())
	message := f.Message()
	if f.FileInfo() != nil {
		path = f.FileInfo().ExternalPath()
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

func printFileAnnotationAsGithubActions(buffer *bytes.Buffer, f FileAnnotation) error {
	if f == nil {
		return nil
	}
	_, _ = buffer.WriteString("::error ")

	// file= is required for GitHub Actions, however it is possible to not have
	// a path for a FileAnnotation. We still print something, however we need
	// to test what happens in GitHub Actions if no valid path is printed out.
	path := "<input>"
	if f.FileInfo() != nil {
		path = f.FileInfo().ExternalPath()
	}
	_, _ = buffer.WriteString("file=")
	_, _ = buffer.WriteString(path)

	// Everything else is optional.
	if startLine := f.StartLine(); startLine > 0 {
		_, _ = buffer.WriteString(",line=")
		_, _ = buffer.WriteString(strconv.Itoa(startLine))
		// We only print column information if we have line information.
		if startColumn := f.StartColumn(); startColumn > 0 {
			_, _ = buffer.WriteString(",col=")
			_, _ = buffer.WriteString(strconv.Itoa(startColumn))
		}
		// We only do any ending line information if we have starting line information
		if endLine := f.EndLine(); endLine > 0 {
			_, _ = buffer.WriteString(",endLine=")
			_, _ = buffer.WriteString(strconv.Itoa(endLine))
			// We only print column information if we have line information.
			if endColumn := f.EndColumn(); endColumn > 0 {
				// Yes, the spec has "col" for start and "endColumn" for end.
				_, _ = buffer.WriteString(",endColumn=")
				_, _ = buffer.WriteString(strconv.Itoa(endColumn))
			}
		}
	}

	_, _ = buffer.WriteString("::")
	_, _ = buffer.WriteString(f.Message())
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
		StartLine:   atLeast1(f.StartLine()),
		StartColumn: atLeast1(f.StartColumn()),
		EndLine:     atLeast1(f.EndLine()),
		EndColumn:   atLeast1(f.EndColumn()),
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
