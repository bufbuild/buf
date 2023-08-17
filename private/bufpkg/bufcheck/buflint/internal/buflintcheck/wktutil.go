package buflintcheck

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

var wktLookup = map[string]WellKnownType{
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

func NewWellKnownType(m string) WellKnownType {
	if strings.HasPrefix(m, WellKnownTypePackage) {
		return LookupWKT(strings.TrimPrefix(m, WellKnownTypePackage))
	}
	return UnknownWKT
}

func IsWellKnown(m string) bool {
	return NewWellKnownType(m).Valid()
}

// LookupWKT returns the WellKnownType related to the provided Name. If the
// name is not recognized, UnknownWKT is returned.
func LookupWKT(n string) WellKnownType {
	if wkt, ok := wktLookup[n]; ok {
		return wkt
	}

	return UnknownWKT
}

// Valid returns true if the WellKnownType is recognized by this library.
func (wkt WellKnownType) Valid() bool {
	_, ok := wktLookup[string(wkt)]
	return ok
}
