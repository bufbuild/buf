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

package bufprotosource

import "google.golang.org/protobuf/reflect/protoreflect"

type enum struct {
	namedDescriptor
	optionExtensionDescriptor

	values                             []EnumValue
	allowAlias                         bool
	deprecatedLegacyJSONFieldConflicts bool
	deprecated                         bool
	allowAliasPath                     []int32
	reservedEnumRanges                 []EnumRange
	reservedNames                      []ReservedName
	parent                             Message
}

func newEnum(
	namedDescriptor namedDescriptor,
	optionExtensionDescriptor optionExtensionDescriptor,
	allowAlias bool,
	deprecatedLegacyJSONFieldConflicts bool,
	deprecated bool,
	allowAliasPath []int32,
	parent Message,
) *enum {
	return &enum{
		namedDescriptor:                    namedDescriptor,
		optionExtensionDescriptor:          optionExtensionDescriptor,
		allowAlias:                         allowAlias,
		deprecatedLegacyJSONFieldConflicts: deprecatedLegacyJSONFieldConflicts,
		deprecated:                         deprecated,
		allowAliasPath:                     allowAliasPath,
		parent:                             parent,
	}
}

func (e *enum) Values() []EnumValue {
	return e.values
}

func (e *enum) AllowAlias() bool {
	return e.allowAlias
}

func (e *enum) DeprecatedLegacyJSONFieldConflicts() bool {
	return e.deprecatedLegacyJSONFieldConflicts
}

func (e *enum) Deprecated() bool {
	return e.deprecated
}

func (e *enum) AllowAliasLocation() Location {
	return e.getLocation(e.allowAliasPath)
}

func (e *enum) ReservedEnumRanges() []EnumRange {
	return e.reservedEnumRanges
}

func (e *enum) ReservedTagRanges() []TagRange {
	tagRanges := make([]TagRange, len(e.reservedEnumRanges))
	for i, reservedEnumRange := range e.reservedEnumRanges {
		tagRanges[i] = reservedEnumRange
	}
	return tagRanges
}

func (e *enum) ReservedNames() []ReservedName {
	return e.reservedNames
}

func (e *enum) Parent() Message {
	return e.parent
}

func (e *enum) AsDescriptor() (protoreflect.EnumDescriptor, error) {
	return asDescriptor[protoreflect.EnumDescriptor](&e.descriptor, e.FullName(), "an enum")
}

func (e *enum) addValue(value EnumValue) {
	e.values = append(e.values, value)
}

func (e *enum) addReservedEnumRange(reservedEnumRange EnumRange) {
	e.reservedEnumRanges = append(e.reservedEnumRanges, reservedEnumRange)
}

func (e *enum) addReservedName(reservedName ReservedName) {
	e.reservedNames = append(e.reservedNames, reservedName)
}
