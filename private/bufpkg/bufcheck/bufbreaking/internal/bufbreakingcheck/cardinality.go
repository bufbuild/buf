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

package bufbreakingcheck

import (
	"strconv"

	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	cardinalityOptionalExplicitPresence cardinality = iota + 1
	cardinalityOptionalImplicitPresence
	cardinalityRequired
	cardinalityRepeated
	cardinalityMap
)

var (
	cardinalityToWireCompatiblityGroup = map[cardinality]int{
		cardinalityOptionalExplicitPresence: 1,
		cardinalityOptionalImplicitPresence: 1,
		cardinalityRequired:                 2,
		cardinalityRepeated:                 3,
		cardinalityMap:                      3,
	}

	cardinalityToWireJSONCompatiblityGroup = map[cardinality]int{
		cardinalityOptionalExplicitPresence: 1,
		cardinalityOptionalImplicitPresence: 1,
		cardinalityRequired:                 2,
		cardinalityRepeated:                 3,
		cardinalityMap:                      4, // maps and repeated use different JSON format
	}
)

type cardinality int

func (c cardinality) String() string {
	switch c {
	case cardinalityOptionalExplicitPresence:
		return "optional with explicit presence"
	case cardinalityOptionalImplicitPresence:
		return "optional with implicit presence"
	case cardinalityRequired:
		return "required"
	case cardinalityRepeated:
		return "repeated"
	case cardinalityMap:
		return "map"
	default:
		return strconv.Itoa(int(c))
	}
}

func getCardinality(field protoreflect.FieldDescriptor) cardinality {
	switch {
	case field.IsList():
		return cardinalityRepeated
	case field.IsMap():
		return cardinalityMap
	case field.Cardinality() == protoreflect.Required:
		return cardinalityRequired
	case field.HasPresence():
		return cardinalityOptionalExplicitPresence
	default:
		return cardinalityOptionalImplicitPresence
	}
}
