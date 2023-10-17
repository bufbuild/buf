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

func (a *adder) add(format string, args ...interface{}) {
	a.addFunc(
		a.field,
		a.field.OptionExtensionLocation(validate.E_Field),
		nil,
		format,
		args...,
	)
}

func (a *adder) addForPath(path []int32, format string, args ...interface{}) {
	a.addFunc(
		a.field,
		a.field.OptionExtensionLocation(validate.E_Field, path...),
		nil,
		format,
		args...,
	)
}

func (a *adder) addForPaths(path []int32, additionalPaths [][]int32, format string, args ...interface{}) {
	locations := make([]protosource.Location, 0, len(additionalPaths)+1)
	locations = append(locations, a.field.OptionExtensionLocation(validate.E_Field, path...))
	for _, additionalPath := range additionalPaths {
		locations = append(locations, a.field.OptionExtensionLocation(validate.E_Field, additionalPath...))
	}
	// different paths can have the same location
	locations = deduplicateLocations(locations)
	if len(locations) > 0 {
		a.addFunc(
			a.field,
			locations[0],
			locations[1:],
			format,
			args...,
		)
	}
}

func deduplicateLocations(locations []protosource.Location) []protosource.Location {
	exactLocations := map[int]map[int]map[int]map[int]struct{}{}
	uniqueLocations := make([]protosource.Location, 0, len(locations))
	for _, location := range locations {
		if _, ok := exactLocations[location.StartLine()][location.StartColumn()][location.EndLine()][location.EndColumn()]; ok {
			continue
		}
		exactLocations[location.StartLine()][location.StartColumn()][location.EndLine()][location.EndColumn()] = struct{}{}
		uniqueLocations = append(uniqueLocations, location)
	}
	return uniqueLocations
}
