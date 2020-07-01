// Copyright 2020 Buf Technologies, Inc.
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

	"github.com/bufbuild/buf/internal/buf/bufanalysis"
	"github.com/bufbuild/buf/internal/buf/bufcore"
	"github.com/bufbuild/buf/internal/buf/bufsrc"
)

// Helper is a helper for checkers.
type Helper struct {
	id              string
	ignoreFunc      IgnoreFunc
	fileAnnotations []bufanalysis.FileAnnotation
}

// NewHelper returns a new Helper for the given id.
func NewHelper(id string, ignoreFunc IgnoreFunc) *Helper {
	return &Helper{
		id:         id,
		ignoreFunc: ignoreFunc,
	}
}

// AddFileAnnotationf adds a FileAnnotation with the id as the Type.
//
// If descriptor is nil, no filename information is added.
// If location is nil, no line or column information will be added.
func (h *Helper) AddFileAnnotationf(
	descriptor bufsrc.Descriptor,
	location bufsrc.Location,
	format string,
	args ...interface{},
) {
	if h.ignoreFunc != nil && h.ignoreFunc(h.id, descriptor, location) {
		return
	}
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
func (h *Helper) FileAnnotations() []bufanalysis.FileAnnotation {
	return h.fileAnnotations
}

// newFileAnnotationf adds a FileAnnotation with the id as the Type.
//
// If descriptor is nil, no filename information is added.
// If location is nil, no line or column information will be added.
func newFileAnnotationf(
	id string,
	descriptor bufsrc.Descriptor,
	location bufsrc.Location,
	format string,
	args ...interface{},
) bufanalysis.FileAnnotation {
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
	var fileInfo bufcore.FileInfo
	if descriptor != nil {
		fileInfo = descriptor.File()
	}
	return bufanalysis.NewFileAnnotation(
		fileInfo,
		startLine,
		startColumn,
		endLine,
		endColumn,
		id,
		fmt.Sprintf(format, args...),
	)
}
