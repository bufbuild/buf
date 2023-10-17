// Copyright 2020-2023 Buf Technologies, Inc.
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

package buflintvalidate

import (
	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"github.com/bufbuild/buf/private/pkg/protosource"
)

type adder struct {
	field   protosource.Field
	addFunc func(protosource.Descriptor, protosource.Location, []protosource.Location, string, ...interface{})
}

func (a *adder) addf(format string, args ...interface{}) {
	a.addFunc(
		a.field,
		a.field.OptionExtensionLocation(validate.E_Field),
		nil,
		format,
		args...,
	)
}

func (a *adder) addForPathf(path []int32, format string, args ...interface{}) {
	a.addFunc(
		a.field,
		a.field.OptionExtensionLocation(validate.E_Field, path...),
		nil,
		format,
		args...,
	)
}

func (a *adder) addForPathsf(paths [][]int32, format string, args ...interface{}) {
	locations := make([]protosource.Location, 0, len(paths))
	for _, path := range paths {
		locations = append(locations, a.field.OptionExtensionLocation(validate.E_Field, path...))
	}
	// different paths can have the same location
	locations = deduplicateLocations(locations)
	for _, location := range locations {
		a.addFunc(
			a.field,
			location,
			nil,
			format,
			args...,
		)
	}
}

func deduplicateLocations(locations []protosource.Location) []protosource.Location {
	type locationFields struct {
		startLine   int
		startColumn int
		endLine     int
		endColumn   int
	}
	exactLocations := map[locationFields]struct{}{}
	uniqueLocations := make([]protosource.Location, 0, len(locations))
	for _, location := range locations {
		locationFields := locationFields{
			startLine:   location.StartLine(),
			startColumn: location.StartColumn(),
			endLine:     location.EndLine(),
			endColumn:   location.EndColumn(),
		}
		if _, ok := exactLocations[locationFields]; ok {
			continue
		}
		exactLocations[locationFields] = struct{}{}
		uniqueLocations = append(uniqueLocations, location)
	}
	return uniqueLocations
}
