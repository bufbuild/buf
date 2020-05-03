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

package internal

import (
	"fmt"

	filev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/file/v1beta1"
	"github.com/bufbuild/buf/internal/pkg/protodesc"
)

// Helper is a helper for checkers.
type Helper struct {
	id              string
	fileAnnotations []*filev1beta1.FileAnnotation
}

// NewHelper returns a new Helper for the given id.
func NewHelper(id string) *Helper {
	return &Helper{
		id: id,
	}
}

// AddFileAnnotationf adds a FileAnnotation with the id as the Type.
//
// If descriptor is nil, no filename information is added.
// If location is nil, no line or column information will be added.
func (h *Helper) AddFileAnnotationf(
	descriptor protodesc.Descriptor,
	location protodesc.Location,
	format string,
	args ...interface{},
) {
	h.fileAnnotations = append(
		h.fileAnnotations,
		newFileAnnotationf(
			h.id,
			descriptor,
			location,
			format,
			args...,
		),
	)
}

// FileAnnotations returns the added FileAnnotations.
func (h *Helper) FileAnnotations() []*filev1beta1.FileAnnotation {
	return h.fileAnnotations
}

// newFileAnnotationf adds a FileAnnotation with the id as the Type.
//
// If descriptor is nil, no filename information is added.
// If location is nil, no line or column information will be added.
func newFileAnnotationf(
	id string,
	descriptor protodesc.Descriptor,
	location protodesc.Location,
	format string,
	args ...interface{},
) *filev1beta1.FileAnnotation {
	path := ""
	if descriptor != nil {
		// this is a root file path
		path = descriptor.FilePath()
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
	return &filev1beta1.FileAnnotation{
		Path:        path,
		StartLine:   uint32(startLine),
		StartColumn: uint32(startColumn),
		EndLine:     uint32(endLine),
		EndColumn:   uint32(endColumn),
		Type:        id,
		Message:     fmt.Sprintf(format, args...),
	}
}
