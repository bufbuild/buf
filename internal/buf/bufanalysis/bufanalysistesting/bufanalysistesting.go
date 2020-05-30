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

package bufanalysistesting

import (
	"fmt"
	"testing"

	"github.com/bufbuild/buf/internal/buf/bufanalysis"
	"github.com/bufbuild/buf/internal/buf/bufimage"
	"github.com/bufbuild/buf/internal/buf/bufpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FileAnnotationFunc is a function that creates a new FileAnnotation with a resolver.
type FileAnnotationFunc func(*testing.T, bufpath.ExternalPathResolver) bufanalysis.FileAnnotation

// NewFileAnnotationNoLocationOrPathFunc returns a new FileAnnotationFunc with no location or FileRef.
func NewFileAnnotationNoLocationOrPathFunc(typeString string) FileAnnotationFunc {
	return NewFileAnnotationNoLocationFunc(
		"",
		"",
		typeString,
	)
}

// NewFileAnnotationNoLocationFunc returns a new FileAnnotationFunc with no location.
//
// If rootRelFilePath == "", FileRef will be nil.
func NewFileAnnotationNoLocationFunc(
	rootRelFilePath string,
	rootDirPath string,
	typeString string,
) FileAnnotationFunc {
	return NewFileAnnotationFunc(
		rootRelFilePath,
		rootDirPath,
		0,
		0,
		0,
		0,
		typeString,
	)
}

// NewFileAnnotationFunc returns a new FileAnnotationFunc.
//
// If rootRelFilePath == "", FileRef will be nil.
func NewFileAnnotationFunc(
	rootRelFilePath string,
	rootDirPath string,
	startLine int,
	startColumn int,
	endLine int,
	endColumn int,
	typeString string,
) FileAnnotationFunc {
	return func(
		t *testing.T,
		externalPathResolver bufpath.ExternalPathResolver,
	) bufanalysis.FileAnnotation {
		var fileRef bufimage.FileRef
		var err error
		if rootRelFilePath != "" {
			fileRef, err = bufimage.NewFileRef(
				rootRelFilePath,
				rootDirPath,
				externalPathResolver,
			)
			require.NoError(t, err)
		}
		return bufanalysis.NewFileAnnotation(
			fileRef,
			startLine,
			startColumn,
			endLine,
			endColumn,
			typeString,
			"",
		)
	}
}

// FileAnnotations returns the FileAnnotations.
func FileAnnotations(
	t *testing.T,
	fileAnnotationFuncs []FileAnnotationFunc,
	externalPathResolver bufpath.ExternalPathResolver,
) []bufanalysis.FileAnnotation {
	fileAnnotations := make([]bufanalysis.FileAnnotation, len(fileAnnotationFuncs))
	for i, fileAnnotationFunc := range fileAnnotationFuncs {
		fileAnnotations[i] = fileAnnotationFunc(t, externalPathResolver)
	}
	return fileAnnotations
}

// AssertFileAnnotationsEqual asserts that the annotations are equal minus the message.
func AssertFileAnnotationsEqual(
	t *testing.T,
	expected []bufanalysis.FileAnnotation,
	actual []bufanalysis.FileAnnotation,
) {
	expected = normalizeFileAnnotations(t, expected)
	actual = normalizeFileAnnotations(t, actual)
	assert.Equal(t, len(expected), len(actual), fmt.Sprint(actual))
	if len(expected) == len(actual) {
		for i, a := range actual {
			e := expected[i]
			expectedFileRef := e.FileRef()
			actualFileRef := a.FileRef()
			assert.Equal(t, expectedFileRef, actualFileRef)
			assert.Equal(t, e, a)
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
		fileRef := a.FileRef()
		var err error
		if fileRef != nil {
			fileRef, err = bufimage.NewDirectFileRef(
				fileRef.RootRelFilePath(),
				fileRef.RootDirPath(),
				fileRef.ExternalFilePath(),
			)
			require.NoError(t, err)
		}
		normalizedFileAnnotations[i] = bufanalysis.NewFileAnnotation(
			fileRef,
			a.StartLine(),
			a.StartColumn(),
			a.EndLine(),
			a.EndColumn(),
			a.Type(),
			"",
		)
	}
	return normalizedFileAnnotations
}
