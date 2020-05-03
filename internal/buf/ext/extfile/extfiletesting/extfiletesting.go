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

package extfiletesting

import (
	"strings"
	"testing"

	filev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/file/v1beta1"
	"github.com/bufbuild/buf/internal/pkg/util/utilproto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NewFileAnnotationNoLocation returns a new FileAnnotation for testing.
//
// This does not set the Message field.
func NewFileAnnotationNoLocation(path string, t string) *filev1beta1.FileAnnotation {
	return &filev1beta1.FileAnnotation{
		Path: path,
		Type: t,
	}
}

// NewFileAnnotation returns a new FileAnnotation for testing.
//
// This does not set the Message field.
func NewFileAnnotation(path string, startLine int, startColumn int, endLine int, endColumn int, t string) *filev1beta1.FileAnnotation {
	return &filev1beta1.FileAnnotation{
		Path:        path,
		StartLine:   uint32(startLine),
		StartColumn: uint32(startColumn),
		EndLine:     uint32(endLine),
		EndColumn:   uint32(endColumn),
		Type:        t,
	}
}

// AssertFileAnnotationsEqual asserts that the annotations are equal minus the Message field.
func AssertFileAnnotationsEqual(t *testing.T, expected []*filev1beta1.FileAnnotation, actual []*filev1beta1.FileAnnotation) {
	s := make([]string, len(actual))
	for i, fileAnnotation := range actual {
		data, err := utilproto.MarshalJSONOrigName(fileAnnotation)
		require.NoError(t, err)
		s[i] = string(data)
	}
	assert.Equal(t, normalizeFileAnnotations(expected), normalizeFileAnnotations(actual), strings.Join(s, "\n"))
}

func normalizeFileAnnotations(fileAnnotations []*filev1beta1.FileAnnotation) []*filev1beta1.FileAnnotation {
	if fileAnnotations == nil {
		return nil
	}
	normalizedFileAnnotations := make([]*filev1beta1.FileAnnotation, len(fileAnnotations))
	for i, a := range fileAnnotations {
		normalizedFileAnnotations[i] = &filev1beta1.FileAnnotation{
			Path:        a.Path,
			StartLine:   a.StartLine,
			StartColumn: a.StartColumn,
			EndLine:     a.EndLine,
			EndColumn:   a.EndColumn,
			Type:        a.Type,
		}
	}
	return normalizedFileAnnotations
}
