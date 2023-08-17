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

func checkFieldRules(add addFunc, field protosource.Field, rules *validate.FieldConstraints) error {
	return newModule(add, field).checkFieldRules(rules)
}

func (m *module) checkFieldRules(rules *validate.FieldConstraints) error {
	if rules == nil {
		return nil
	}

	switch r := rules.Type.(type) {
	case *validate.FieldConstraints_Float:
		mustType(m.add, m.field, descriptorpb.FieldDescriptorProto_TYPE_FLOAT, FloatValueWKT)
		checkFloat(m.add, m.field, r.Float)
	case *validate.FieldConstraints_Double:
		mustType(m.add, m.field, descriptorpb.FieldDescriptorProto_TYPE_DOUBLE, DoubleValueWKT)
		checkDouble(m.add, m.field, r.Double)
	case *validate.FieldConstraints_Int32:
		mustType(m.add, m.field, descriptorpb.FieldDescriptorProto_TYPE_INT32, Int32ValueWKT)
		checkInt32(m.add, m.field, r.Int32)
	case *validate.FieldConstraints_Int64:
		mustType(m.add, m.field, descriptorpb.FieldDescriptorProto_TYPE_INT64, Int64ValueWKT)
		checkInt64(m.add, m.field, r.Int64)
	case *validate.FieldConstraints_Uint32:
		mustType(m.add, m.field, descriptorpb.FieldDescriptorProto_TYPE_UINT32, UInt32ValueWKT)
		checkUInt32(m.add, m.field, r.Uint32)
	case *validate.FieldConstraints_Uint64:
		mustType(m.add, m.field, descriptorpb.FieldDescriptorProto_TYPE_UINT64, UInt64ValueWKT)
		checkUInt64(m.add, m.field, r.Uint64)
	case *validate.FieldConstraints_Sint32:
		mustType(m.add, m.field, descriptorpb.FieldDescriptorProto_TYPE_SINT32, UnknownWKT)
		checkSInt32(m.add, m.field, r.Sint32)
	case *validate.FieldConstraints_Sint64:
		mustType(m.add, m.field, descriptorpb.FieldDescriptorProto_TYPE_SINT64, UnknownWKT)
		checkSInt64(m.add, m.field, r.Sint64)
	case *validate.FieldConstraints_Fixed32:
		mustType(m.add, m.field, descriptorpb.FieldDescriptorProto_TYPE_FIXED32, UnknownWKT)
		checkFixed32(m.add, m.field, r.Fixed32)
	case *validate.FieldConstraints_Fixed64:
		mustType(m.add, m.field, descriptorpb.FieldDescriptorProto_TYPE_FIXED64, UnknownWKT)
		checkFixed64(m.add, m.field, r.Fixed64)
	case *validate.FieldConstraints_Sfixed32:
		mustType(m.add, m.field, descriptorpb.FieldDescriptorProto_TYPE_SFIXED32, UnknownWKT)
		checkSFixed32(m.add, m.field, r.Sfixed32)
	case *validate.FieldConstraints_Sfixed64:
		mustType(m.add, m.field, descriptorpb.FieldDescriptorProto_TYPE_SFIXED64, UnknownWKT)
		checkSFixed64(m.add, m.field, r.Sfixed64)
	case *validate.FieldConstraints_Bool:
		mustType(m.add, m.field, descriptorpb.FieldDescriptorProto_TYPE_BOOL, BoolValueWKT)
	case *validate.FieldConstraints_String_:
		mustType(m.add, m.field, descriptorpb.FieldDescriptorProto_TYPE_STRING, StringValueWKT)
		checkString(m.add, m.field, r.String_)
	case *validate.FieldConstraints_Bytes:
		mustType(m.add, m.field, descriptorpb.FieldDescriptorProto_TYPE_BYTES, BytesValueWKT)
		checkBytes(m.add, m.field, r.Bytes)
	case *validate.FieldConstraints_Enum:
		mustType(m.add, m.field, descriptorpb.FieldDescriptorProto_TYPE_ENUM, UnknownWKT)
		checkEnum(m.add, m.field, r.Enum)
	case *validate.FieldConstraints_Repeated:
		// TODO: check type
		checkRepeated(m.add, m.field, r.Repeated)
	case *validate.FieldConstraints_Map:
		// TODO: check type
		checkMap(m.add, m.field, r.Map)
	case *validate.FieldConstraints_Any:
		// TODO: check type
		checkAny(m.add, m.field, r.Any)
	case *validate.FieldConstraints_Duration:
		// TODO: check type
		checkDuration(m.add, m.field, r.Duration)
	case *validate.FieldConstraints_Timestamp:
		// TODO: check type
		checkTimestamp(m.add, m.field, r.Timestamp)
	case nil: // noop
	default:
		// TODO: (elliotmjackson) consider this case, it might be an error
		return nil
	}
	return nil
}

func mustType(add addFunc, field protosource.Field, pt descriptorpb.FieldDescriptorProto_Type, wrapper WellKnownType) {
	// TODO: (elliotmjackson) the logic here is a mess
	if emb := field.Message(); emb != nil && IsWellKnown(emb) && NewWellKnownType(emb) == wrapper {
		mustType(add, field, emb.Fields()[0].Type(), UnknownWKT)
		return
	}

	// TODO: this is likely caught already
	//if typ, ok := field.(Repeatable); ok {
	//	if !typ.IsRepeated() {
	//		add(field, field.OptionExtensionLocation(validate.E_Field), nil,
	//			"repeated rule should be used for repeated fields")
	//	}
	//}

	if field.Type() != pt {
		add(field, field.OptionExtensionLocation(validate.E_Field), nil,
			"expected rules for %s but got %s", field.Type(), pt.String(),
		)
	}
}

func checkFloat(add addFunc, field protosource.Field, r *validate.FloatRules) {
	checkNums(add, field, len(r.In), len(r.NotIn), r.Const, r.Lt, r.Lte, r.Gt, r.Gte)
}

func checkDouble(add addFunc, field protosource.Field, r *validate.DoubleRules) {
	checkNums(add, field, len(r.In), len(r.NotIn), r.Const, r.Lt, r.Lte, r.Gt, r.Gte)
}

func checkInt32(add addFunc, field protosource.Field, r *validate.Int32Rules) {
	checkNums(add, field, len(r.In), len(r.NotIn), r.Const, r.Lt, r.Lte, r.Gt, r.Gte)
}

func checkInt64(add addFunc, field protosource.Field, r *validate.Int64Rules) {
	checkNums(add, field, len(r.In), len(r.NotIn), r.Const, r.Lt, r.Lte, r.Gt, r.Gte)
}

func checkUInt32(add addFunc, field protosource.Field, r *validate.UInt32Rules) {
	checkNums(add, field, len(r.In), len(r.NotIn), r.Const, r.Lt, r.Lte, r.Gt, r.Gte)
}

func checkUInt64(add addFunc, field protosource.Field, r *validate.UInt64Rules) {
	checkNums(add, field, len(r.In), len(r.NotIn), r.Const, r.Lt, r.Lte, r.Gt, r.Gte)
}

func checkSInt32(add addFunc, field protosource.Field, r *validate.SInt32Rules) {
	checkNums(add, field, len(r.In), len(r.NotIn), r.Const, r.Lt, r.Lte, r.Gt, r.Gte)
}

func checkSInt64(add addFunc, field protosource.Field, r *validate.SInt64Rules) {
	checkNums(add, field, len(r.In), len(r.NotIn), r.Const, r.Lt, r.Lte, r.Gt, r.Gte)
}

func checkFixed32(add addFunc, field protosource.Field, r *validate.Fixed32Rules) {
	checkNums(add, field, len(r.In), len(r.NotIn), r.Const, r.Lt, r.Lte, r.Gt, r.Gte)
}

func checkFixed64(add addFunc, field protosource.Field, r *validate.Fixed64Rules) {
	checkNums(add, field, len(r.In), len(r.NotIn), r.Const, r.Lt, r.Lte, r.Gt, r.Gte)
}

func checkSFixed32(add addFunc, field protosource.Field, r *validate.SFixed32Rules) {
	checkNums(add, field, len(r.In), len(r.NotIn), r.Const, r.Lt, r.Lte, r.Gt, r.Gte)
}

func checkSFixed64(add addFunc, field protosource.Field, r *validate.SFixed64Rules) {
	checkNums(add, field, len(r.In), len(r.NotIn), r.Const, r.Lt, r.Lte, r.Gt, r.Gte)
}

func checkNums(add addFunc, field protosource.Field, in, notIn int, ci, lti, ltei, gti, gtei interface{}) {
	checkIns(add, field, in, notIn)

	c := reflect.ValueOf(ci)
	lt, lte := reflect.ValueOf(lti), reflect.ValueOf(ltei)
	gt, gte := reflect.ValueOf(gti), reflect.ValueOf(gtei)

	assert(add, field,
		c.IsNil() ||
			in == 0 && notIn == 0 &&
				lt.IsNil() && lte.IsNil() &&
				gt.IsNil() && gte.IsNil(),
		"`const` can be the only rule on a field",
	)

	assert(add, field,
		in == 0 ||
			lt.IsNil() && lte.IsNil() &&
				gt.IsNil() && gte.IsNil(),
		"cannot have both `in` and range constraint rules on the same field",
	)

	assert(add, field,
		lt.IsNil() || lte.IsNil(),
		"cannot have both `lt` and `lte` rules on the same field",
	)

	assert(add, field,
		gt.IsNil() || gte.IsNil(),
		"cannot have both `gt` and `gte` rules on the same field",
	)

	if !lt.IsNil() {
		assert(add, field, gt.IsNil() || !reflect.DeepEqual(lti, gti),
			"cannot have equal `gt` and `lt` rules on the same field")
		assert(add, field, gte.IsNil() || !reflect.DeepEqual(lti, gtei),
			"cannot have equal `gte` and `lt` rules on the same field")
	} else if !lte.IsNil() {
		assert(add, field, gt.IsNil() || !reflect.DeepEqual(ltei, gti),
			"cannot have equal `gt` and `lte` rules on the same field")
		assert(add, field, gte.IsNil() || !reflect.DeepEqual(ltei, gtei),
			"use `const` instead of equal `lte` and `gte` rules")
	}
}

func checkIns(add addFunc, field protosource.Field, in, notIn int) {
	assert(add, field,
		in == 0 || notIn == 0,
		"cannot have both `in` and `not_in` rules on the same field")
}

func assert(add addFunc, field protosource.Field, expr bool, v ...interface{}) {
	if !expr {
		add(field, field.OptionExtensionLocation(validate.E_Field), nil, fmt.Sprint(v...))
	}
}

func checkString(add addFunc, field protosource.Field, r *validate.StringRules) {
	checkLen(add, field, r.Len, r.MinLen, r.MaxLen)
	checkLen(add, field, r.LenBytes, r.MinBytes, r.MaxBytes)
	checkMinMax(add, field, r.MinLen, r.MaxLen)
	checkMinMax(add, field, r.MinBytes, r.MaxBytes)
	checkIns(add, field, len(r.In), len(r.NotIn))
	checkWellKnownRegex(add, field, r.GetWellKnownRegex(), r)
	checkPattern(add, field, r.Pattern, len(r.In))

	if r.MaxLen != nil {
		max := int(r.GetMaxLen())
		assert(add, field, utf8.RuneCountInString(r.GetPrefix()) <= max, "`prefix` length exceeds the `max_len`")
		assert(add, field, utf8.RuneCountInString(r.GetSuffix()) <= max, "`suffix` length exceeds the `max_len`")
		assert(add, field, utf8.RuneCountInString(r.GetContains()) <= max, "`contains` length exceeds the `max_len`")

		assert(add, field,
			r.MaxBytes == nil || r.GetMaxBytes() >= r.GetMaxLen(),
			"`max_len` cannot exceed `max_bytes`")
	}

	if r.MaxBytes != nil {
		max := int(r.GetMaxBytes())
		assert(add, field, len(r.GetPrefix()) <= max, "`prefix` length exceeds the `max_bytes`")
		assert(add, field, len(r.GetSuffix()) <= max, "`suffix` length exceeds the `max_bytes`")
		assert(add, field, len(r.GetContains()) <= max, "`contains` length exceeds the `max_bytes`")
	}
}

func checkLen(add addFunc, field protosource.Field, len, min, max *uint64) {
	if len == nil {
		return
	}

	assert(add, field,
		min == nil,
		"cannot have both `len` and `min_len` rules on the same field")

	assert(add, field,
		max == nil,
		"cannot have both `len` and `max_len` rules on the same field")
}

func checkMinMax(add addFunc, field protosource.Field, min, max *uint64) {
	if min == nil || max == nil {
		return
	}

	assert(add, field,
		*min <= *max,
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

func checkWellKnownRegex(add addFunc, field protosource.Field, wk validate.KnownRegex, r *validate.StringRules) {
	if wk != 0 {
		assert(add, field,
			r.Pattern == nil, "regex `well_known_regex` and regex `pattern` are incompatible")
		non_strict := r.Strict != nil && !*r.Strict
		if (wk.String() == "HTTP_HEADER_NAME" || wk.String() == "HTTP_HEADER_VALUE") && non_strict {
			// Use non-strict header validation.
			r.Pattern = regex_map["HEADER_STRING"]
		} else {
			r.Pattern = regex_map[wk.String()]
		}
	}
}

func checkPattern(add addFunc, field protosource.Field, p *string, in int) {
	if p != nil {
		assert(add, field,
			in == 0, "regex `pattern` and `in` rules are incompatible")
		_, err := regexp.Compile(*p)
		assert(add, field,
			err != nil, "unable to parse regex `pattern`")
	}
}

func checkEnum(add addFunc, field protosource.Field, r *validate.EnumRules) {
	checkIns(add, field, len(r.In), len(r.NotIn))

	if r.GetDefinedOnly() && len(r.In) > 0 {
		typ, ok := field.(interface {
			Enum() protosource.Enum
		})

		assert(add, field, !ok, "unexpected field type (%T)", field)

		defined := typ.Enum().Values()
		vals := make(map[int]struct{}, len(defined))

		for _, val := range defined {
			vals[val.Number()] = struct{}{}
		}

		for _, in := range r.In {
			if _, ok = vals[int(in)]; !ok {
				assert(add, field, !ok, "undefined `in` value (%d) conflicts with `defined_only` rule", in)
			}
		}
	}
}

func checkBytes(add addFunc, field protosource.Field, r *validate.BytesRules) {
	checkMinMax(add, field, r.MinLen, r.MaxLen)
	checkIns(add, field, len(r.In), len(r.NotIn))
	checkPattern(add, field, r.Pattern, len(r.In))

	if r.MaxLen != nil {
		max := int(r.GetMaxLen())
		assert(add, field, len(r.GetPrefix()) <= max, "`prefix` length exceeds the `max_len`")
		assert(add, field, len(r.GetSuffix()) <= max, "`suffix` length exceeds the `max_len`")
		assert(add, field, len(r.GetContains()) <= max, "`contains` length exceeds the `max_len`")
	}
}

func checkRepeated(add addFunc, field protosource.Field, r *validate.RepeatedRules) {
	assert(
		add,
		field,
		field.Label() == descriptorpb.FieldDescriptorProto_LABEL_REPEATED,
		"field is not repeated but got repeated rules",
	)

	checkMinMax(add, field, r.MinItems, r.MaxItems)

	if r.GetUnique() {
		assert(add, field,
			field.Type() != descriptorpb.FieldDescriptorProto_TYPE_MESSAGE,
			"unique rule is only applicable for scalar types")
	}

	// TODO: this returns an error which is ignored here
	checkFieldRules(add, field, r.Items)
}

func checkMap(add addFunc, field protosource.Field, r *validate.MapRules) {
	// TODO: determine if field is a map
	isMessage := field.Type() == descriptorpb.FieldDescriptorProto_TYPE_MESSAGE
	message := field.Message()
	assert(
		add,
		field,
		isMessage && message.IsMapEntry(),
		"field is not a map but got map rules",
	)

	checkMinMax(add, field, r.MinPairs, r.MaxPairs)

	// TODO: this returns an error which is ignored here
	checkFieldRules(add, field, r.Keys)
	// TODO: this returns an error which is ignored here
	checkFieldRules(add, field, r.Values)
}

func checkAny(add addFunc, field protosource.Field, r *validate.AnyRules) {
	checkIns(add, field, len(r.In), len(r.NotIn))
}

func checkDuration(add addFunc, field protosource.Field, r *validate.DurationRules) {
	checkNums(add, field,
		len(r.GetIn()),
		len(r.GetNotIn()),
		checkDur(add, field, r.GetConst()),
		checkDur(add, field, r.GetLt()),
		checkDur(add, field, r.GetLte()),
		checkDur(add, field, r.GetGt()),
		checkDur(add, field, r.GetGte()))

	for _, v := range r.GetIn() {
		assert(add, field, v != nil, "cannot have nil values in `in`")
		checkDur(add, field, v)
	}

	for _, v := range r.GetNotIn() {
		assert(add, field, v != nil, "cannot have nil values in `not_in`")
		checkDur(add, field, v)
	}
}

func checkDur(add addFunc, field protosource.Field, d *durationpb.Duration) *time.Duration {
	if d == nil {
		return nil
	}

	dur, err := d.AsDuration(), d.CheckValid()
	assert(add, field, err == nil, "could not resolve duration")
	return &dur
}

func checkTimestamp(add addFunc, field protosource.Field, r *validate.TimestampRules) {
	checkNums(add, field, 0, 0,
		checkTS(add, field, r.GetConst()),
		checkTS(add, field, r.GetLt()),
		checkTS(add, field, r.GetLte()),
		checkTS(add, field, r.GetGt()),
		checkTS(add, field, r.GetGte()))

	assert(add, field,
		(r.LtNow == nil && r.GtNow == nil) || (r.Lt == nil && r.Lte == nil && r.Gt == nil && r.Gte == nil),
		"`now` rules cannot be mixed with absolute `lt/gt` rules")

	assert(add, field,
		r.Within == nil || (r.Lt == nil && r.Lte == nil && r.Gt == nil && r.Gte == nil),
		"`within` rule cannot be used with absolute `lt/gt` rules")

	assert(add, field,
		r.LtNow == nil || r.GtNow == nil,
		"both `now` rules cannot be used together")

	dur := checkDur(add, field, r.Within)
	assert(add, field,
		dur == nil || *dur > 0,
		"`within` rule must be positive and non-zero")
}

func checkTS(add addFunc, field protosource.Field, ts *timestamppb.Timestamp) *int64 {
	if ts == nil {
		return nil
	}

	t, err := ts.AsTime(), ts.CheckValid()
	assert(add, field, err == nil, "could not resolve timestamp")
	return proto.Int64(t.UnixNano())
}
