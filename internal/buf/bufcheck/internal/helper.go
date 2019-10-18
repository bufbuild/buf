package internal

import (
	"fmt"

	"github.com/bufbuild/buf/internal/pkg/analysis"
	"github.com/bufbuild/buf/internal/pkg/protodesc"
)

// Helper is a helper for checkers.
type Helper struct {
	id          string
	annotations []*analysis.Annotation
}

// NewHelper returns a new Helper for the given id.
func NewHelper(id string) *Helper {
	return &Helper{
		id: id,
	}
}

// AddAnnotationf adds an annotation with the id as the Type.
//
// If descriptor is nil, no filename information is added.
// If location is nil, no line or column information will be added.
func (h *Helper) AddAnnotationf(
	descriptor protodesc.Descriptor,
	location protodesc.Location,
	format string,
	args ...interface{},
) {
	h.annotations = append(
		h.annotations,
		newAnnotationf(
			h.id,
			descriptor,
			location,
			format,
			args...,
		),
	)
}

// Annotations returns the added annotations.
func (h *Helper) Annotations() []*analysis.Annotation {
	return h.annotations
}

// newAnnotationf adds an annotation with the id as the Type.
//
// If descriptor is nil, no filename information is added.
// If location is nil, no line or column information will be added.
func newAnnotationf(
	id string,
	descriptor protodesc.Descriptor,
	location protodesc.Location,
	format string,
	args ...interface{},
) *analysis.Annotation {
	filename := ""
	if descriptor != nil {
		// this is a root file path
		filename = descriptor.FilePath()
	}
	startLine := 0
	startColumn := 0
	endLine := 0
	endColumn := 0
	if location != nil {
		startLine = location.StartLine()
		startColumn = location.StartColumn()
		endLine = location.EndLine()
		endColumn = location.EndColumn()
	}
	return &analysis.Annotation{
		Filename:    filename,
		StartLine:   startLine,
		StartColumn: startColumn,
		EndLine:     endLine,
		EndColumn:   endColumn,
		Type:        id,
		Message:     fmt.Sprintf(format, args...),
	}
}
