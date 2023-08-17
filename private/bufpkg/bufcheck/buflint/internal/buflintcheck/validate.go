package buflintcheck

import (
	"fmt"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"reflect"
	"regexp"
	"time"
	"unicode/utf8"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"

	"github.com/bufbuild/buf/private/pkg/protosource"
	"google.golang.org/protobuf/types/descriptorpb"
)

type module struct {
	add      addFunc
	field    protosource.Field
	location protosource.Location
}

func newModule(add addFunc, field protosource.Field) *module {
	return &module{
		add:      add,
		field:    field,
		location: field.OptionExtensionLocation(validate.E_Field),
	}
}

func (m *module) checkFieldRules(rules *validate.FieldConstraints) {
	if rules == nil {
		return
	}

	switch r := rules.Type.(type) {
	case *validate.FieldConstraints_Float:
		m.mustType(descriptorpb.FieldDescriptorProto_TYPE_FLOAT, FloatValueWKT)
		m.checkFloat(r.Float)
	case *validate.FieldConstraints_Double:
		m.mustType(descriptorpb.FieldDescriptorProto_TYPE_DOUBLE, DoubleValueWKT)
		m.checkDouble(r.Double)
	case *validate.FieldConstraints_Int32:
		m.mustType(descriptorpb.FieldDescriptorProto_TYPE_INT32, Int32ValueWKT)
		m.checkInt32(r.Int32)
	case *validate.FieldConstraints_Int64:
		m.mustType(descriptorpb.FieldDescriptorProto_TYPE_INT64, Int64ValueWKT)
		m.checkInt64(r.Int64)
	case *validate.FieldConstraints_Uint32:
		m.mustType(descriptorpb.FieldDescriptorProto_TYPE_UINT32, UInt32ValueWKT)
		m.checkUInt32(r.Uint32)
	case *validate.FieldConstraints_Uint64:
		m.mustType(descriptorpb.FieldDescriptorProto_TYPE_UINT64, UInt64ValueWKT)
		m.checkUInt64(r.Uint64)
	case *validate.FieldConstraints_Sint32:
		m.mustType(descriptorpb.FieldDescriptorProto_TYPE_SINT32, UnknownWKT)
		m.checkSInt32(r.Sint32)
	case *validate.FieldConstraints_Sint64:
		m.mustType(descriptorpb.FieldDescriptorProto_TYPE_SINT64, UnknownWKT)
		m.checkSInt64(r.Sint64)
	case *validate.FieldConstraints_Fixed32:
		m.mustType(descriptorpb.FieldDescriptorProto_TYPE_FIXED32, UnknownWKT)
		m.checkFixed32(r.Fixed32)
	case *validate.FieldConstraints_Fixed64:
		m.mustType(descriptorpb.FieldDescriptorProto_TYPE_FIXED64, UnknownWKT)
		m.checkFixed64(r.Fixed64)
	case *validate.FieldConstraints_Sfixed32:
		m.mustType(descriptorpb.FieldDescriptorProto_TYPE_SFIXED32, UnknownWKT)
		m.checkSFixed32(r.Sfixed32)
	case *validate.FieldConstraints_Sfixed64:
		m.mustType(descriptorpb.FieldDescriptorProto_TYPE_SFIXED64, UnknownWKT)
		m.checkSFixed64(r.Sfixed64)
	case *validate.FieldConstraints_Bool:
		m.mustType(descriptorpb.FieldDescriptorProto_TYPE_BOOL, BoolValueWKT)
	case *validate.FieldConstraints_String_:
		m.mustType(descriptorpb.FieldDescriptorProto_TYPE_STRING, StringValueWKT)
		m.checkString(r.String_)
	case *validate.FieldConstraints_Bytes:
		m.mustType(descriptorpb.FieldDescriptorProto_TYPE_BYTES, BytesValueWKT)
		m.checkBytes(r.Bytes)
	case *validate.FieldConstraints_Enum:
		m.mustType(descriptorpb.FieldDescriptorProto_TYPE_ENUM, UnknownWKT)
		m.checkEnum(r.Enum)
	case *validate.FieldConstraints_Repeated:
		// TODO: check type
		m.checkRepeated(r.Repeated)
	case *validate.FieldConstraints_Map:
		// TODO: check type
		m.checkMap(r.Map)
	case *validate.FieldConstraints_Any:
		// TODO: check type
		m.checkAny(r.Any)
	case *validate.FieldConstraints_Duration:
		// TODO: check type
		m.checkDuration(r.Duration)
	case *validate.FieldConstraints_Timestamp:
		// TODO: check type
		m.checkTimestamp(r.Timestamp)
	}
}

func (m *module) mustType(pt descriptorpb.FieldDescriptorProto_Type, wrapper WellKnownType) {
	// TODO: (elliotmjackson) the logic here is a mess
	//m.field.TypeLocation()
	//if emb := m.field.TypeName(); emb != "" && IsWellKnown(emb) && NewWellKnownType(emb) == wrapper {
	//	m.mustType(m.field.Message().Fields()[0].Type(), UnknownWKT)
	//	return
	//}

	// TODO: this is likely caught already
	//if typ, ok := field.(Repeatable); ok {
	//	if !typ.IsRepeated() {
	//		add(field, field.OptionExtensionLocation(validate.E_Field), nil,
	//			"repeated rule should be used for repeated fields")
	//	}
	//}

	expr := m.field.Type() == pt
	m.assert(
		expr,
		"expected rules for ",
		m.field.Type(),
		" but got ",
		pt.String(),
	)
}

func (m *module) checkFloat(r *validate.FloatRules) {
	m.checkNums(len(r.In), len(r.NotIn), r.Const, r.Lt, r.Lte, r.Gt, r.Gte)
}

func (m *module) checkDouble(r *validate.DoubleRules) {
	m.checkNums(len(r.In), len(r.NotIn), r.Const, r.Lt, r.Lte, r.Gt, r.Gte)
}

func (m *module) checkInt32(r *validate.Int32Rules) {
	m.checkNums(len(r.In), len(r.NotIn), r.Const, r.Lt, r.Lte, r.Gt, r.Gte)
}

func (m *module) checkInt64(r *validate.Int64Rules) {
	m.checkNums(len(r.In), len(r.NotIn), r.Const, r.Lt, r.Lte, r.Gt, r.Gte)
}

func (m *module) checkUInt32(r *validate.UInt32Rules) {
	m.checkNums(len(r.In), len(r.NotIn), r.Const, r.Lt, r.Lte, r.Gt, r.Gte)
}

func (m *module) checkUInt64(r *validate.UInt64Rules) {
	m.checkNums(len(r.In), len(r.NotIn), r.Const, r.Lt, r.Lte, r.Gt, r.Gte)
}

func (m *module) checkSInt32(r *validate.SInt32Rules) {
	m.checkNums(len(r.In), len(r.NotIn), r.Const, r.Lt, r.Lte, r.Gt, r.Gte)
}

func (m *module) checkSInt64(r *validate.SInt64Rules) {
	m.checkNums(len(r.In), len(r.NotIn), r.Const, r.Lt, r.Lte, r.Gt, r.Gte)
}

func (m *module) checkFixed32(r *validate.Fixed32Rules) {
	m.checkNums(len(r.In), len(r.NotIn), r.Const, r.Lt, r.Lte, r.Gt, r.Gte)
}

func (m *module) checkFixed64(r *validate.Fixed64Rules) {
	m.checkNums(len(r.In), len(r.NotIn), r.Const, r.Lt, r.Lte, r.Gt, r.Gte)
}

func (m *module) checkSFixed32(r *validate.SFixed32Rules) {
	m.checkNums(len(r.In), len(r.NotIn), r.Const, r.Lt, r.Lte, r.Gt, r.Gte)
}

func (m *module) checkSFixed64(r *validate.SFixed64Rules) {
	m.checkNums(len(r.In), len(r.NotIn), r.Const, r.Lt, r.Lte, r.Gt, r.Gte)
}

func (m *module) checkNums(in, notIn int, ci, lti, ltei, gti, gtei interface{}) {
	m.checkIns(in, notIn)

	c := reflect.ValueOf(ci)
	lt, lte := reflect.ValueOf(lti), reflect.ValueOf(ltei)
	gt, gte := reflect.ValueOf(gti), reflect.ValueOf(gtei)

	m.assert(c.IsNil() ||
		in == 0 && notIn == 0 &&
			lt.IsNil() && lte.IsNil() &&
			gt.IsNil() && gte.IsNil(),
		"`const` can be the only rule on a field",
	)

	m.assert(in == 0 ||
		lt.IsNil() && lte.IsNil() &&
			gt.IsNil() && gte.IsNil(),
		"cannot have both `in` and range constraint rules on the same field",
	)

	m.assert(lt.IsNil() || lte.IsNil(),
		"cannot have both `lt` and `lte` rules on the same field",
	)

	m.assert(gt.IsNil() || gte.IsNil(),
		"cannot have both `gt` and `gte` rules on the same field",
	)

	if !lt.IsNil() {
		m.assert(gt.IsNil() || !reflect.DeepEqual(lti, gti),
			"cannot have equal `gt` and `lt` rules on the same field")
		m.assert(gte.IsNil() || !reflect.DeepEqual(lti, gtei),
			"cannot have equal `gte` and `lt` rules on the same field")
	} else if !lte.IsNil() {
		m.assert(gt.IsNil() || !reflect.DeepEqual(ltei, gti),
			"cannot have equal `gt` and `lte` rules on the same field")
		m.assert(gte.IsNil() || !reflect.DeepEqual(ltei, gtei),
			"use `const` instead of equal `lte` and `gte` rules")
	}
}

func (m *module) checkIns(in, notIn int) {
	m.assert(in == 0 || notIn == 0,
		"cannot have both `in` and `not_in` rules on the same field")
}

func (m *module) assert(expr bool, v ...interface{}) {
	if !expr {
		m.add(m.field, m.location, nil, fmt.Sprint(v...))
	}
}

func (m *module) checkString(r *validate.StringRules) {
	m.checkLen(r.Len, r.MinLen, r.MaxLen)
	m.checkLen(r.LenBytes, r.MinBytes, r.MaxBytes)
	m.checkMinMax(r.MinLen, r.MaxLen)
	m.checkMinMax(r.MinBytes, r.MaxBytes)
	m.checkIns(len(r.In), len(r.NotIn))
	m.checkWellKnownRegex(r.GetWellKnownRegex(), r)
	m.checkPattern(r.Pattern, len(r.In))

	if r.MaxLen != nil {
		max := int(r.GetMaxLen())
		m.assert(utf8.RuneCountInString(r.GetPrefix()) <= max, "`prefix` length exceeds the `max_len`")
		m.assert(utf8.RuneCountInString(r.GetSuffix()) <= max, "`suffix` length exceeds the `max_len`")
		m.assert(utf8.RuneCountInString(r.GetContains()) <= max, "`contains` length exceeds the `max_len`")

		m.assert(r.MaxBytes == nil || r.GetMaxBytes() >= r.GetMaxLen(),
			"`max_len` cannot exceed `max_bytes`")
	}

	if r.MaxBytes != nil {
		max := int(r.GetMaxBytes())
		m.assert(len(r.GetPrefix()) <= max, "`prefix` length exceeds the `max_bytes`")
		m.assert(len(r.GetSuffix()) <= max, "`suffix` length exceeds the `max_bytes`")
		m.assert(len(r.GetContains()) <= max, "`contains` length exceeds the `max_bytes`")
	}
}

func (m *module) checkLen(len, min, max *uint64) {
	if len == nil {
		return
	}

	m.assert(min == nil,
		"cannot have both `len` and `min_len` rules on the same field")

	m.assert(max == nil,
		"cannot have both `len` and `max_len` rules on the same field")
}

func (m *module) checkMinMax(min, max *uint64) {
	if min == nil || max == nil {
		return
	}

	m.assert(*min <= *max,
		"`min` value is greater than `max` value")
}

var (
	unknown         = ""
	httpHeaderName  = "^:?[0-9a-zA-Z!#$%&'*+-.^_|~\x60]+$"
	httpHeaderValue = "^[^\u0000-\u0008\u000A-\u001F\u007F]*$"
	headerString    = "^[^\u0000\u000A\u000D]*$" // For non-strict validation.
)

// Map from well-known regex to a regex pattern.
var regex_map = map[string]*string{
	"UNKNOWN":           &unknown,
	"HTTP_HEADER_NAME":  &httpHeaderName,
	"HTTP_HEADER_VALUE": &httpHeaderValue,
	"HEADER_STRING":     &headerString,
}

func (m *module) checkWellKnownRegex(wk validate.KnownRegex, r *validate.StringRules) {
	if wk != 0 {
		m.assert(r.Pattern == nil, "regex `well_known_regex` and regex `pattern` are incompatible")
		non_strict := r.Strict != nil && !*r.Strict
		if (wk.String() == "HTTP_HEADER_NAME" || wk.String() == "HTTP_HEADER_VALUE") && non_strict {
			// Use non-strict header validation.
			r.Pattern = regex_map["HEADER_STRING"]
		} else {
			r.Pattern = regex_map[wk.String()]
		}
	}
}

func (m *module) checkPattern(p *string, in int) {
	if p != nil {
		m.assert(in == 0, "regex `pattern` and `in` rules are incompatible")
		_, err := regexp.Compile(*p)
		m.assert(err != nil, "unable to parse regex `pattern`")
	}
}

func (m *module) checkEnum(r *validate.EnumRules) {
	m.checkIns(len(r.In), len(r.NotIn))

	if r.GetDefinedOnly() && len(r.In) > 0 {
		typ, ok := m.field.(interface {
			Enum() protosource.Enum
		})

		m.assert(!ok, "unexpected field type (%T)", m.field)

		defined := typ.Enum().Values()
		vals := make(map[int]struct{}, len(defined))

		for _, val := range defined {
			vals[val.Number()] = struct{}{}
		}

		for _, in := range r.In {
			if _, ok = vals[int(in)]; !ok {
				m.assert(!ok, "undefined `in` value (%d) conflicts with `defined_only` rule", in)
			}
		}
	}
}

func (m *module) checkBytes(r *validate.BytesRules) {
	m.checkMinMax(r.MinLen, r.MaxLen)
	m.checkIns(len(r.In), len(r.NotIn))
	m.checkPattern(r.Pattern, len(r.In))

	if r.MaxLen != nil {
		max := int(r.GetMaxLen())
		m.assert(len(r.GetPrefix()) <= max, "`prefix` length exceeds the `max_len`")
		m.assert(len(r.GetSuffix()) <= max, "`suffix` length exceeds the `max_len`")
		m.assert(len(r.GetContains()) <= max, "`contains` length exceeds the `max_len`")
	}
}

func (m *module) checkRepeated(r *validate.RepeatedRules) {
	m.assert(
		m.field.Label() == descriptorpb.FieldDescriptorProto_LABEL_REPEATED,
		"field is not repeated but got repeated rules",
	)

	m.checkMinMax(r.MinItems, r.MaxItems)

	if r.GetUnique() {
		m.assert(m.field.Type() != descriptorpb.FieldDescriptorProto_TYPE_MESSAGE,
			"unique rule is only applicable for scalar types")
	}

	// TODO: this returns an error which is ignored here
	m.checkFieldRules(r.Items)
}

func (m *module) checkMap(r *validate.MapRules) {
	// TODO: determine if field is a map
	isMessage := m.field.Type() == descriptorpb.FieldDescriptorProto_TYPE_MESSAGE
	message := m.field.Message()
	m.assert(
		isMessage && message.IsMapEntry(),
		"field is not a map but got map rules",
	)

	m.checkMinMax(r.MinPairs, r.MaxPairs)

	// TODO: this returns an error which is ignored here
	m.checkFieldRules(r.Keys)
	// TODO: this returns an error which is ignored here
	m.checkFieldRules(r.Values)
}

func (m *module) checkAny(r *validate.AnyRules) {
	m.checkIns(len(r.In), len(r.NotIn))
}

func (m *module) checkDuration(r *validate.DurationRules) {
	m.checkNums(
		len(r.GetIn()),
		len(r.GetNotIn()),
		m.checkDur(r.GetConst()),
		m.checkDur(r.GetLt()),
		m.checkDur(r.GetLte()),
		m.checkDur(r.GetGt()),
		m.checkDur(r.GetGte()))

	for _, v := range r.GetIn() {
		m.assert(v != nil, "cannot have nil values in `in`")
		m.checkDur(v)
	}

	for _, v := range r.GetNotIn() {
		m.assert(v != nil, "cannot have nil values in `not_in`")
		m.checkDur(v)
	}
}

func (m *module) checkDur(d *durationpb.Duration) *time.Duration {
	if d == nil {
		return nil
	}

	dur, err := d.AsDuration(), d.CheckValid()
	m.assert(err == nil, "could not resolve duration")
	return &dur
}

func (m *module) checkTimestamp(r *validate.TimestampRules) {
	m.checkNums(0, 0,
		m.checkTS(r.GetConst()),
		m.checkTS(r.GetLt()),
		m.checkTS(r.GetLte()),
		m.checkTS(r.GetGt()),
		m.checkTS(r.GetGte()))

	m.assert((r.LtNow == nil && r.GtNow == nil) || (r.Lt == nil && r.Lte == nil && r.Gt == nil && r.Gte == nil),
		"`now` rules cannot be mixed with absolute `lt/gt` rules")

	m.assert(r.Within == nil || (r.Lt == nil && r.Lte == nil && r.Gt == nil && r.Gte == nil),
		"`within` rule cannot be used with absolute `lt/gt` rules")

	m.assert(r.LtNow == nil || r.GtNow == nil,
		"both `now` rules cannot be used together")

	dur := m.checkDur(r.Within)
	m.assert(dur == nil || *dur > 0,
		"`within` rule must be positive and non-zero")
}

func (m *module) checkTS(ts *timestamppb.Timestamp) *int64 {
	if ts == nil {
		return nil
	}

	t, err := ts.AsTime(), ts.CheckValid()
	m.assert(err == nil, "could not resolve timestamp")
	return proto.Int64(t.UnixNano())
}
