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

package buflintvalidate

import (
	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"github.com/bufbuild/buf/private/bufpkg/bufprotosource"
	"google.golang.org/protobuf/encoding/protowire"
)

// The typical use of adder is calling adder.addForPathf([]int32{int64RulesFieldNumber, someFieldNumber}, "message")
// from checkConstraintsForField (or a function that it calls). Notice that checkConstraintsForField
// is recursive, because it can call checkMapRules and checkRepeatedRules, both of which can
// call checkConstraintsForField.
//
// If checkConstraintsForField is called by checkMapRules, when we add a file annotation, the
// location should be for something like `repeated.items.string.max_len`. We need to search for the
// location by a path like [mapRulesFieldNumber, keysFieldNumber, StringRulesFieldNumber, ...].
//
// If checkConstraintsForField is not in a recursive call, when we add a file annotation, the
// location should be for something like `string.max_len`. We need to search for the location by
// a path like [int64RulesFieldNumber, ...].
//
// However, from checkConstraintsForField's perspective, it doesn't know whether it's in a recursive
// call. It always treats the path like [int64RulesFieldNumber, ...], as opposed to [mapRulesFieldNumber, keysFieldNumber, StringRulesFieldNumber, ...].
// To preserve the first part of the path, [mapRulesFieldNumber, keysFieldNumber], we create a new adder
// with a base path when we recursively call checkConstraintsForField. The new adder will automatically
// prepend the base path whenever it searches for a location. This is manageable because the recursion
// depth is at most 2 -- if checkMapRules or checkRepeatedRules calls checkConstraintsForField,
// this call of checkConstraintsForField won't call checkMapRules or checkRepeatedRules.
type adder struct {
	field               bufprotosource.Field
	fieldPrettyTypeName string
	basePath            []int32
	addFunc             func(bufprotosource.Descriptor, bufprotosource.Location, []bufprotosource.Location, string, ...interface{})
}

func (a *adder) cloneWithNewBasePath(basePath ...int32) *adder {
	return &adder{
		field:               a.field,
		fieldPrettyTypeName: a.fieldPrettyTypeName,
		basePath:            basePath,
		addFunc:             a.addFunc,
	}
}

func (a *adder) addForPathf(path []int32, format string, args ...interface{}) {
	// Copy a.basePath so it won't be modified by append.
	combinedPath := make([]int32, len(a.basePath), len(a.basePath)+len(path))
	copy(combinedPath, a.basePath)
	a.addFunc(
		a.field,
		a.field.OptionExtensionLocation(validate.E_Field, append(combinedPath, path...)...),
		nil,
		format,
		args...,
	)
}

func (a *adder) addForPathsf(paths [][]int32, format string, args ...interface{}) {
	locations := make([]bufprotosource.Location, 0, len(paths))
	for _, path := range paths {
		// Copy a.basePath so it won't be modified by append.
		combinedPath := make([]int32, len(a.basePath), len(a.basePath)+len(path))
		copy(combinedPath, a.basePath)
		locations = append(locations, a.field.OptionExtensionLocation(validate.E_Field, append(combinedPath, path...)...))
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

func (a *adder) fieldName() string {
	return a.field.Name()
}

func (a *adder) getFieldRuleName(path ...int32) string {
	name := "(buf.validate.field)"
	fields := fieldConstraintsDescriptor.Fields()
	combinedPath := path
	if len(a.basePath) > 0 {
		combinedPath = make([]int32, len(a.basePath), len(a.basePath)+len(path))
		copy(combinedPath, a.basePath)
		combinedPath = append(combinedPath, path...)
	}
	for _, fieldNumber := range combinedPath {
		subField := fields.ByNumber(protowire.Number(fieldNumber))
		if subField == nil {
			return name
		}
		name += "."
		name += string(subField.Name())
		subFieldMessage := subField.Message()
		if subFieldMessage == nil {
			return name
		}
		fields = subField.Message().Fields()
	}
	return name
}

func deduplicateLocations(locations []bufprotosource.Location) []bufprotosource.Location {
	type locationFields struct {
		startLine   int
		startColumn int
		endLine     int
		endColumn   int
	}
	exactLocations := map[locationFields]struct{}{}
	uniqueLocations := make([]bufprotosource.Location, 0, len(locations))
	for _, location := range locations {
		var locationValue locationFields
		if location != nil {
			locationValue = locationFields{
				startLine:   location.StartLine(),
				startColumn: location.StartColumn(),
				endLine:     location.EndLine(),
				endColumn:   location.EndColumn(),
			}
		}
		if _, ok := exactLocations[locationValue]; ok {
			continue
		}
		exactLocations[locationValue] = struct{}{}
		uniqueLocations = append(uniqueLocations, location)
	}
	return uniqueLocations
}
