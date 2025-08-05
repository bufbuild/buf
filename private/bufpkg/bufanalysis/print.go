// Copyright 2020-2025 Buf Technologies, Inc.
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
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/bufbuild/buf/private/pkg/shake256"
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

func printAsGitLabCodeQuality(writer io.Writer, fileAnnotations []FileAnnotation) error {
	report := make([]*externalGitLabCodeQualityIssue, 0, len(fileAnnotations))
	for _, f := range fileAnnotations {
		if f == nil {
			continue
		}
		gitLabCodeQualityContentIssue, err := newExternalGitLabCodeQualityIssue(f)
		if err != nil {
			return err
		}
		report = append(report, gitLabCodeQualityContentIssue)
	}
	return json.NewEncoder(writer).Encode(report)
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
	if pluginName, policyName := f.PluginName(), f.PolicyName(); pluginName != "" || policyName != "" {
		_, _ = buffer.WriteString(" (")
		if pluginName != "" {
			_, _ = buffer.WriteString(pluginName)
		}
		if pluginName != "" && policyName != "" {
			_, _ = buffer.WriteString(", ")
		}
		if policyName != "" {
			_, _ = buffer.WriteString(policyName)
		}
		_, _ = buffer.WriteRune(')')
	}
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
	if pluginName, policyName := f.PluginName(), f.PolicyName(); pluginName != "" || policyName != "" {
		_, _ = buffer.WriteString(" (")
		if pluginName != "" {
			_, _ = buffer.WriteString(pluginName)
		}
		if pluginName != "" && policyName != "" {
			_, _ = buffer.WriteString(", ")
		}
		if policyName != "" {
			_, _ = buffer.WriteString(policyName)
		}
		_, _ = buffer.WriteRune(')')
	}
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
	Plugin      string `json:"plugin,omitempty" yaml:"plugin,omitempty"`
	Policy      string `json:"policy,omitempty" yaml:"policy,omitempty"`
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
		Plugin:      f.PluginName(),
		Policy:      f.PolicyName(),
	}
}

// externalGitLabCodeQualityIssue represents the GitLab Code Quality Report Issue structure
// expected for the GitLab Code Quality Report output.
//
// https://docs.gitlab.com/ci/testing/code_quality/#code-quality-report-format
// Note that all fields are required for GitLabCodeQualityIssues. If a field is missing,
// for example, the path of a deleted file in breaking changes, no GitLab Code Quality violation
// will be surfaced in the Code Quality Report.
type externalGitLabCodeQualityIssue struct {
	// Description is a human readable description of the code quality violation. This maps
	// to the Message in FileAnnotation.
	Description string `json:"description,omitempty"`
	// CheckName is a human readable name of the check or rule. This maps to the Type in
	// FileAnnotation.
	CheckName string `json:"check_name,omitempty"`
	// Fingerprint is a unique identifier for the specific code quality violation. This maps
	// to a shake256 digest of the JSON FileAnnotation.
	Fingerprint string `json:"fingerprint,omitempty"`
	// Location is a location structure that represents the file location of the code quality
	// violation.
	Location externalGitLabCodeQualityIssueLocation `json:"location,omitempty"`
	// Severity is the line location of the code quality violation. We use "minor" as the default.
	Severity string `json:"severity,omitempty"`
}

type externalGitLabCodeQualityIssueLocation struct {
	// Path is the relative file path of the code quality violation. This maps to the external
	// path of the FileAnnotation.
	Path string `json:"path,omitempty"`
	// Positions is the line location of the code quality violation. This maps to StartLine
	// of the FileAnnotation.
	Positions externalGitLabCodeQualityIssueLocationPositions `json:"positions,omitempty"`
}

type externalGitLabCodeQualityIssueLocationPositions struct {
	Begin externalGitLabCodeQualityIssueLocationPosition `json:"begin,omitempty"`
	End   externalGitLabCodeQualityIssueLocationPosition `json:"end,omitempty"`
}

type externalGitLabCodeQualityIssueLocationPosition struct {
	Line   int `json:"line,omitempty"`
	Column int `json:"column,omitempty"`
}

// newExternalGitLabCodeQualityIssue returns an externalGitLabCodeQualityIssue for the
// specified FileAnnotation.
func newExternalGitLabCodeQualityIssue(f FileAnnotation) (*externalGitLabCodeQualityIssue, error) {
	path := ""
	if f.FileInfo() != nil {
		// GitLab Code Quality Issues strictly require the path to be a relative path based on
		// the repository. This will need to be enforced by the user.
		path = f.FileInfo().ExternalPath()
	}
	gitLabCodeQualityIssue := externalGitLabCodeQualityIssue{
		Description: f.Message(),
		CheckName:   f.Type(),
		Location: externalGitLabCodeQualityIssueLocation{
			Path: path,
			Positions: externalGitLabCodeQualityIssueLocationPositions{
				Begin: externalGitLabCodeQualityIssueLocationPosition{
					Line:   f.StartLine(),
					Column: f.StartColumn(),
				},
				End: externalGitLabCodeQualityIssueLocationPosition{
					Line:   f.EndLine(),
					Column: f.EndColumn(),
				},
			},
		},
		Severity: "minor",
	}
	gitLabCodeQualityIssueContent, err := json.Marshal(gitLabCodeQualityIssue)
	if err != nil {
		return nil, err
	}
	// We use the hash of the GitLab Code Quality issues content, minus the hash itself.
	hash, err := shake256.NewDigestForContent(bytes.NewReader(gitLabCodeQualityIssueContent))
	if err != nil {
		return nil, err
	}
	gitLabCodeQualityIssue.Fingerprint = hex.EncodeToString(hash.Value())
	return &gitLabCodeQualityIssue, nil
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
