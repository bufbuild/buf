package extfiletesting

import (
	"encoding/json"
	"strings"
	"testing"

	filev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/file/v1beta1"
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
		data, err := json.Marshal(fileAnnotation)
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
