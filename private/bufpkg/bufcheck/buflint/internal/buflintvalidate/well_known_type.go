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

import "strings"

// WellKnownTypePackage is the proto package name where all Well Known Types
// currently reside.
const WellKnownTypePackage string = "google.protobuf."

// WellKnownType (WKT) encapsulates the Name of a Message from the
// `google.protobuf` package. Most official protoc plugins special case code
// generation on these messages.
type WellKnownType string

// 1-to-1 mapping of the WKT names to WellKnownTypes.
const (
	// UnknownWKT indicates that the type is not a known WKT. This value may be
	// returned erroneously mapping a Name to a WellKnownType or if a WKT is
	// added to the `google.protobuf` package but this library is outdated.
	UnknownWKT WellKnownType = "Unknown"

	AnyWKT         WellKnownType = "Any"
	DurationWKT    WellKnownType = "Duration"
	EmptyWKT       WellKnownType = "Empty"
	StructWKT      WellKnownType = "Struct"
	TimestampWKT   WellKnownType = "Timestamp"
	ValueWKT       WellKnownType = "Value"
	ListValueWKT   WellKnownType = "ListValue"
	DoubleValueWKT WellKnownType = "DoubleValue"
	FloatValueWKT  WellKnownType = "FloatValue"
	Int64ValueWKT  WellKnownType = "Int64Value"
	UInt64ValueWKT WellKnownType = "UInt64Value"
	Int32ValueWKT  WellKnownType = "Int32Value"
	UInt32ValueWKT WellKnownType = "UInt32Value"
	BoolValueWKT   WellKnownType = "BoolValue"
	StringValueWKT WellKnownType = "StringValue"
	BytesValueWKT  WellKnownType = "BytesValue"
)

var lookupTable = map[string]WellKnownType{
	"Any":         AnyWKT,
	"Duration":    DurationWKT,
	"Empty":       EmptyWKT,
	"Struct":      StructWKT,
	"Timestamp":   TimestampWKT,
	"Value":       ValueWKT,
	"ListValue":   ListValueWKT,
	"DoubleValue": DoubleValueWKT,
	"FloatValue":  FloatValueWKT,
	"Int64Value":  Int64ValueWKT,
	"UInt64Value": UInt64ValueWKT,
	"Int32Value":  Int32ValueWKT,
	"UInt32Value": UInt32ValueWKT,
	"BoolValue":   BoolValueWKT,
	"StringValue": StringValueWKT,
	"BytesValue":  BytesValueWKT,
}

// LookupWellKnownType returns the WellKnownType related to the provided Name. If the
// name is not recognized, UnknownWKT is returned.
func LookupWellKnownType(in string) WellKnownType {
	if strings.HasPrefix(in, WellKnownTypePackage) {
		in = strings.TrimPrefix(in, WellKnownTypePackage)
	}
	if wellKnownType, ok := lookupTable[in]; ok {
		return wellKnownType
	}

	return UnknownWKT
}

// Valid returns true if the WellKnownType is recognized by this library.
func (wkt WellKnownType) Valid() bool {
	_, ok := lookupTable[string(wkt)]
	return ok
}
