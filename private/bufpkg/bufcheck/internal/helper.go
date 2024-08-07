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

package internal

import (
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufprotosource"
)

// Helper is a helper for rules.
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
	descriptor bufprotosource.Descriptor,
	location bufprotosource.Location,
	format string,
	args ...interface{},
) {
	h.addFileAnnotationf(
		descriptor,
		nil,
		location,
		nil,
		format,
		args...,
	)
}

// AddFileAnnotationWithExtraIgnoreDescriptorsf adds a FileAnnotation with the id as the Type.
//
// extraIgnoreDescriptors are extra desciptors to check for ignores.
//
// If descriptor is nil, no filename information is added.
// If location is nil, no line or column information will be added.
func (h *Helper) AddFileAnnotationWithExtraIgnoreDescriptorsf(
	descriptor bufprotosource.Descriptor,
	extraIgnoreDescriptors []bufprotosource.Descriptor,
	location bufprotosource.Location,
	format string,
	args ...interface{},
) {
	h.addFileAnnotationf(
		descriptor,
		extraIgnoreDescriptors,
		location,
		nil,
		format,
		args...,
	)
}

// AddFileAnnotationWithExtraIgnoreLocationsf adds a FileAnnotation with the id as the Type.
//
// extraIgnoreLocations are extra locations to check for comment ignores.
//
// If descriptor is nil, no filename information is added.
// If location is nil, no line or column information will be added.
func (h *Helper) AddFileAnnotationWithExtraIgnoreLocationsf(
	descriptor bufprotosource.Descriptor,
	location bufprotosource.Location,
	extraIgnoreLocations []bufprotosource.Location,
	format string,
	args ...interface{},
) {
	h.addFileAnnotationf(
		descriptor,
		nil,
		location,
		extraIgnoreLocations,
		format,
		args...,
	)
}

func (h *Helper) addFileAnnotationf(
	descriptor bufprotosource.Descriptor,
	extraIgnoreDescriptors []bufprotosource.Descriptor,
	location bufprotosource.Location,
	extraIgnoreLocations []bufprotosource.Location,
	format string,
	args ...interface{},
) {
	ignoreLocations := []bufprotosource.Location{location}
	var descriptorLocation bufprotosource.Location
	if locationDescriptor, ok := descriptor.(bufprotosource.LocationDescriptor); ok {
		if descriptorLocation = locationDescriptor.Location(); descriptorLocation != location {
			// Include the location of the descriptor itself as a source of ignores.
			ignoreLocations = append(ignoreLocations, descriptorLocation)
		}
	}
	for _, extraIgnoreLocation := range extraIgnoreLocations {
		if extraIgnoreLocation != location && extraIgnoreLocation != descriptorLocation {
			ignoreLocations = append(ignoreLocations, extraIgnoreLocation)
		}
	}
	if h.ignoreFunc != nil && h.ignoreFunc(
		h.id,
		append([]bufprotosource.Descriptor{descriptor}, extraIgnoreDescriptors...),
		ignoreLocations,
	) {
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
	descriptor bufprotosource.Descriptor,
	location bufprotosource.Location,
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
	var fileInfo bufanalysis.FileInfo
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
