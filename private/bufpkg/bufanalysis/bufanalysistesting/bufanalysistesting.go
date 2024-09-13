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

package bufanalysistesting

import (
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/stretchr/testify/assert"
)

// NewFileAnnotationNoLocationOrPath returns a new FileAnnotation with no location or FileInfo.
func NewFileAnnotationNoLocationOrPath(
	t *testing.T,
	typeString string,
) bufanalysis.FileAnnotation {
	return NewFileAnnotation(
		t,
		"",
		0,
		0,
		0,
		0,
		typeString,
	)
}

// NewFileAnnotationNoLocation returns a new FileAnnotation with no location.
//
// fileInfo can be nil.
func NewFileAnnotationNoLocation(
	t *testing.T,
	path string,
	typeString string,
) bufanalysis.FileAnnotation {
	return NewFileAnnotation(
		t,
		path,
		1,
		1,
		1,
		1,
		typeString,
	)
}

// NewFileAnnotation returns a new FileAnnotation.
func NewFileAnnotation(
	t *testing.T,
	path string,
	startLine int,
	startColumn int,
	endLine int,
	endColumn int,
	typeString string,
) bufanalysis.FileAnnotation {
	return newFileAnnotation(
		t,
		path,
		startLine,
		startColumn,
		endLine,
		endColumn,
		typeString,
		"",
		"",
	)
}

func newFileAnnotation(
	t *testing.T,
	path string,
	startLine int,
	startColumn int,
	endLine int,
	endColumn int,
	typeString string,
	message string,
	pluginName string,
) bufanalysis.FileAnnotation {
	var fileInfo bufanalysis.FileInfo
	if path != "" {
		fileInfo = newFileInfo(path)
	}
	return bufanalysis.NewFileAnnotation(
		fileInfo,
		startLine,
		startColumn,
		endLine,
		endColumn,
		typeString,
		message,
		pluginName,
	)
}

// AssertFileAnnotationsEqual asserts that the annotations are equal minus the message.
func AssertFileAnnotationsEqual(
	t *testing.T,
	expected []bufanalysis.FileAnnotation,
	actual []bufanalysis.FileAnnotation,
) {
	expected = normalizeFileAnnotations(t, expected)
	actual = normalizeFileAnnotations(t, actual)
	if !assert.Equal(
		t,
		expected,
		actual,
	) {
		t.Log("If actuals are correct, change expectations to the following:")
		for _, annotation := range actual {
			var path string
			if fileInfo := annotation.FileInfo(); fileInfo != nil {
				path = fileInfo.Path()
			}
			if annotation.StartLine() == 0 && annotation.StartColumn() == 0 &&
				annotation.EndLine() == 0 && annotation.EndColumn() == 0 {
				if path == "" {
					t.Logf("    bufanalysistesting.NewFileAnnotationNoLocationOrPath(t, %q),",
						annotation.Type(),
					)
				} else {
					t.Logf("    bufanalysistesting.NewFileAnnotationNoLocation(t, %q, %q),",
						path,
						annotation.Type(),
					)
				}
			} else {
				t.Logf("    bufanalysistesting.NewFileAnnotation(t, %q, %d, %d, %d, %d, %q),",
					path,
					annotation.StartLine(),
					annotation.StartColumn(),
					annotation.EndLine(),
					annotation.EndColumn(),
					annotation.Type(),
				)
			}
		}
	}
}

func normalizeFileAnnotations(
	t *testing.T,
	fileAnnotations []bufanalysis.FileAnnotation,
) []bufanalysis.FileAnnotation {
	if fileAnnotations == nil {
		return nil
	}
	normalizedFileAnnotations := make([]bufanalysis.FileAnnotation, len(fileAnnotations))
	for i, a := range fileAnnotations {
		fileInfo := a.FileInfo()
		if fileInfo != nil {
			fileInfo = newFileInfo(fileInfo.Path())
		}
		normalizedFileAnnotations[i] = bufanalysis.NewFileAnnotation(
			fileInfo,
			a.StartLine(),
			a.StartColumn(),
			a.EndLine(),
			a.EndColumn(),
			a.Type(),
			"",
			"",
		)
	}
	return normalizedFileAnnotations
}

type fileInfo struct {
	path string
}

func newFileInfo(path string) *fileInfo {
	return &fileInfo{
		path: path,
	}
}

func (f *fileInfo) Path() string {
	return f.path
}

func (f *fileInfo) ExternalPath() string {
	return f.path
}
