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
const wellKnownTypePackage string = "google.protobuf."

// wellKnownType (WKT) encapsulates the Name of a Message from the
// `google.protobuf` package. Most official protoc plugins special case code
// generation on these messages.
type wellKnownType string

// 1-to-1 mapping of the WKT names to WellKnownTypes.
const (
	// unknownWKT indicates that the type is not a known WKT. This value may be
	// returned erroneously mapping a Name to a wellKnownType or if a WKT is
	// added to the `google.protobuf` package but this library is outdated.
	unknownWKT wellKnownType = "Unknown"

	anyWKT         wellKnownType = "Any"
	durationWKT    wellKnownType = "Duration"
	emptyWKT       wellKnownType = "Empty"
	structWKT      wellKnownType = "Struct"
	timestampWKT   wellKnownType = "Timestamp"
	valueWKT       wellKnownType = "Value"
	listValueWKT   wellKnownType = "ListValue"
	doubleValueWKT wellKnownType = "DoubleValue"
	floatValueWKT  wellKnownType = "FloatValue"
	int64ValueWKT  wellKnownType = "Int64Value"
	uInt64ValueWKT wellKnownType = "UInt64Value"
	int32ValueWKT  wellKnownType = "Int32Value"
	uInt32ValueWKT wellKnownType = "UInt32Value"
	boolValueWKT   wellKnownType = "BoolValue"
	stringValueWKT wellKnownType = "StringValue"
	bytesValueWKT  wellKnownType = "BytesValue"
)

var stringToWellKnownType = map[string]wellKnownType{
	"Any":         anyWKT,
	"Duration":    durationWKT,
	"Empty":       emptyWKT,
	"Struct":      structWKT,
	"Timestamp":   timestampWKT,
	"Value":       valueWKT,
	"ListValue":   listValueWKT,
	"DoubleValue": doubleValueWKT,
	"FloatValue":  floatValueWKT,
	"Int64Value":  int64ValueWKT,
	"UInt64Value": uInt64ValueWKT,
	"Int32Value":  int32ValueWKT,
	"UInt32Value": uInt32ValueWKT,
	"BoolValue":   boolValueWKT,
	"StringValue": stringValueWKT,
	"BytesValue":  bytesValueWKT,
}

func lookupWellKnownType(in string) wellKnownType {
	in = strings.TrimPrefix(in, wellKnownTypePackage)
	if wellKnownType, ok := stringToWellKnownType[in]; ok {
		return wellKnownType
	}

	return unknownWKT
}

func (wkt wellKnownType) valid() bool {
	_, ok := stringToWellKnownType[string(wkt)]
	return ok
}
