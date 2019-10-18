// Package analysistesting implements testing functionality for Annotations.
package analysistesting

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/bufbuild/buf/internal/pkg/analysis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NewAnnotationNoLocation returns a new Annotation for testing.
//
// This does not set the Message field.
func NewAnnotationNoLocation(filename string, t string) *analysis.Annotation {
	return &analysis.Annotation{
		Filename: filename,
		Type:     t,
	}
}

// NewAnnotation returns a new Annotation for testing.
//
// This does not set the Message field.
func NewAnnotation(filename string, startLine int, startColumn int, endLine int, endColumn int, t string) *analysis.Annotation {
	return &analysis.Annotation{
		Filename:    filename,
		StartLine:   startLine,
		StartColumn: startColumn,
		EndLine:     endLine,
		EndColumn:   endColumn,
		Type:        t,
	}
}

// AssertAnnotationsEqual asserts that the annotations are equal minus the Message field.
func AssertAnnotationsEqual(t *testing.T, expected []*analysis.Annotation, actual []*analysis.Annotation) {
	s := make([]string, len(actual))
	for i, annotation := range actual {
		data, err := json.Marshal(annotation)
		require.NoError(t, err)
		s[i] = string(data)
	}
	assert.Equal(t, normalizeAnnotations(expected), normalizeAnnotations(actual), strings.Join(s, "\n"))
}

func normalizeAnnotations(annotations []*analysis.Annotation) []*analysis.Annotation {
	if annotations == nil {
		return nil
	}
	normalizedAnnotations := make([]*analysis.Annotation, len(annotations))
	for i, a := range annotations {
		normalizedAnnotations[i] = &analysis.Annotation{
			Filename:    a.Filename,
			StartLine:   a.StartLine,
			StartColumn: a.StartColumn,
			EndLine:     a.EndLine,
			EndColumn:   a.EndColumn,
			Type:        a.Type,
		}
	}
	return normalizedAnnotations
}
