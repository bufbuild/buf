package buflintcheck

import (
	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"reflect"
	"regexp"
	"time"
	"unicode/utf8"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bufbuild/buf/private/pkg/protosource"
	"google.golang.org/protobuf/types/descriptorpb"
)

var (
	unknown         = ""
	httpHeaderName  = "^:?[0-9a-zA-Z!#$%&'*+-.^_|~\x60]+$"
	httpHeaderValue = "^[^\u0000-\u0008\u000A-\u001F\u007F]*$"
	headerString    = "^[^\u0000\u000A\u000D]*$" // For non-strict validation.
	// Map from well-known regex to a regex pattern.
	regex_map = map[string]*string{
		"UNKNOWN":           &unknown,
		"HTTP_HEADER_NAME":  &httpHeaderName,
		"HTTP_HEADER_VALUE": &httpHeaderValue,
		"HEADER_STRING":     &headerString,
	}
)

// validateModule is a validate module.
type validateModule struct {
	add   addFunc
	files []protosource.File
}

func newValidateModule(files []protosource.File, add addFunc) *validateModule {
	return &validateModule{
		add:   add,
		files: files,
	}
}

// validateField is a validate Field.
type validateField struct {
	module   *validateModule
	field    protosource.Field
	location protosource.Location
}

// newValidateField returns a new validate validateField.
func (m *validateModule) newValidateField(field protosource.Field) *validateField {
	return &validateField{
		module:   m,
		field:    field,
		location: field.OptionExtensionLocation(validate.E_Field),
	}
}

// newValidateField returns a new validate validateField.
func newValidateField(module *validateModule, field protosource.Field) *validateField {
	return &validateField{
		module:   module,
		field:    field,
		location: field.OptionExtensionLocation(validate.E_Field),
	}
}

// checkFieldRules checks the rules for the field.
func (m *validateField) checkFieldRules(rules *validate.FieldConstraints) {
	if rules == nil {
		return
	}
	switch r := rules.Type.(type) {
	case *validate.FieldConstraints_Float:
		m.mustType(descriptorpb.FieldDescriptorProto_TYPE_FLOAT, FloatValueWKT)
		checkNums(m, len(r.Float.In), len(r.Float.NotIn), r.Float.Const, r.Float.Lt, r.Float.Lte, r.Float.Gt, r.Float.Gte)
	case *validate.FieldConstraints_Double:
		m.mustType(descriptorpb.FieldDescriptorProto_TYPE_DOUBLE, DoubleValueWKT)
		checkNums(m, len(r.Double.In), len(r.Double.NotIn), r.Double.Const, r.Double.Lt, r.Double.Lte, r.Double.Gt, r.Double.Gte)
	case *validate.FieldConstraints_Int32:
		m.mustType(descriptorpb.FieldDescriptorProto_TYPE_INT32, Int32ValueWKT)
		checkNums(m, len(r.Int32.In), len(r.Int32.NotIn), r.Int32.Const, r.Int32.Lt, r.Int32.Lte, r.Int32.Gt, r.Int32.Gte)
	case *validate.FieldConstraints_Int64:
		m.mustType(descriptorpb.FieldDescriptorProto_TYPE_INT64, Int64ValueWKT)
		checkNums(m, len(r.Int64.In), len(r.Int64.NotIn), r.Int64.Const, r.Int64.Lt, r.Int64.Lte, r.Int64.Gt, r.Int64.Gte)
	case *validate.FieldConstraints_Uint32:
		m.mustType(descriptorpb.FieldDescriptorProto_TYPE_UINT32, UInt32ValueWKT)
		checkNums(m, len(r.Uint32.In), len(r.Uint32.NotIn), r.Uint32.Const, r.Uint32.Lt, r.Uint32.Lte, r.Uint32.Gt, r.Uint32.Gte)
	case *validate.FieldConstraints_Uint64:
		m.mustType(descriptorpb.FieldDescriptorProto_TYPE_UINT64, UInt64ValueWKT)
		checkNums(m, len(r.Uint64.In), len(r.Uint64.NotIn), r.Uint64.Const, r.Uint64.Lt, r.Uint64.Lte, r.Uint64.Gt, r.Uint64.Gte)
	case *validate.FieldConstraints_Sint32:
		m.mustType(descriptorpb.FieldDescriptorProto_TYPE_SINT32, UnknownWKT)
		checkNums(m, len(r.Sint32.In), len(r.Sint32.NotIn), r.Sint32.Const, r.Sint32.Lt, r.Sint32.Lte, r.Sint32.Gt, r.Sint32.Gte)
	case *validate.FieldConstraints_Sint64:
		m.mustType(descriptorpb.FieldDescriptorProto_TYPE_SINT64, UnknownWKT)
		checkNums(m, len(r.Sint64.In), len(r.Sint64.NotIn), r.Sint64.Const, r.Sint64.Lt, r.Sint64.Lte, r.Sint64.Gt, r.Sint64.Gte)
	case *validate.FieldConstraints_Fixed32:
		m.mustType(descriptorpb.FieldDescriptorProto_TYPE_FIXED32, UnknownWKT)
		checkNums(m, len(r.Fixed32.In), len(r.Fixed32.NotIn), r.Fixed32.Const, r.Fixed32.Lt, r.Fixed32.Lte, r.Fixed32.Gt, r.Fixed32.Gte)
	case *validate.FieldConstraints_Fixed64:
		m.mustType(descriptorpb.FieldDescriptorProto_TYPE_FIXED64, UnknownWKT)
		checkNums(m, len(r.Fixed64.In), len(r.Fixed64.NotIn), r.Fixed64.Const, r.Fixed64.Lt, r.Fixed64.Lte, r.Fixed64.Gt, r.Fixed64.Gte)
	case *validate.FieldConstraints_Sfixed32:
		m.mustType(descriptorpb.FieldDescriptorProto_TYPE_SFIXED32, UnknownWKT)
		checkNums(m, len(r.Sfixed32.In), len(r.Sfixed32.NotIn), r.Sfixed32.Const, r.Sfixed32.Lt, r.Sfixed32.Lte, r.Sfixed32.Gt, r.Sfixed32.Gte)
	case *validate.FieldConstraints_Sfixed64:
		m.mustType(descriptorpb.FieldDescriptorProto_TYPE_SFIXED64, UnknownWKT)
		checkNums(m, len(r.Sfixed64.In), len(r.Sfixed64.NotIn), r.Sfixed64.Const, r.Sfixed64.Lt, r.Sfixed64.Lte, r.Sfixed64.Gt, r.Sfixed64.Gte)
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
		m.checkRepeated(r.Repeated)
	case *validate.FieldConstraints_Map:
		m.checkMap(r.Map)
	case *validate.FieldConstraints_Any:
		m.noCustomRules(rules)
		m.checkAny(r.Any)
	case *validate.FieldConstraints_Duration:
		m.checkDuration(r.Duration)
	case *validate.FieldConstraints_Timestamp:
		m.checkTimestamp(r.Timestamp)
	}
}

// mustType asserts that the field type is the same as the given type.
func (m *validateField) mustType(pt descriptorpb.FieldDescriptorProto_Type, wrapper WellKnownType) {
	if wrapper != UnknownWKT {
		if emb := m.field.Embed(m.module.files...); emb != nil {
			if wkt := LookupWKT(emb.Name()); wkt.Valid() && wkt == wrapper {
				field := emb.Fields()[0]
				newValidateField(m.module, field).mustType(field.Type(), UnknownWKT)
				return
			}
		}
	}

	expr := m.field.Type() == pt
	m.assert(
		expr,
		"expected rules for %s but got %s",
		m.field.Type(),
		pt.String(),
	)
}

// checkNums asserts that the given numbers are valid.
func checkNums[T any](m *validateField, in, notIn int, ci, lti, ltei, gti, gtei T) {
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

// checkIns asserts that the given `in` and `not_in` rules are valid.
func (m *validateField) checkIns(in, notIn int) {
	m.assert(in == 0 || notIn == 0,
		"cannot have both `in` and `not_in` rules on the same field")
}

// assert asserts that the given expression is true and adds an error if not.
func (m *validateField) assert(expr bool, format string, v ...interface{}) {
	if !expr {
		m.module.add(m.field, m.location, nil, format, v...)
	}
}

// checkString asserts that the given string rules are valid.
func (m *validateField) checkString(r *validate.StringRules) {
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

// checkEnum asserts that the given enum rules are valid.
func (m *validateField) checkEnum(r *validate.EnumRules) {
	m.checkIns(len(r.In), len(r.NotIn))

	if r.GetDefinedOnly() && len(r.In) > 0 {
		// TODO: this is not working
		typ, ok := m.field.(interface {
			Enum() protosource.Enum
		})
		if !ok {
			m.assert(!ok, "unexpected field type (%T)", m.field)
			return
		}

		enum := typ.Enum()
		if enum == nil {
			return
		}
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

// checkBytes asserts that the given bytes rules are valid.
func (m *validateField) checkBytes(r *validate.BytesRules) {
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

// checkRepeated validates the repeated rules.
func (m *validateField) checkRepeated(r *validate.RepeatedRules) {
	m.assert(
		m.field.Label() == descriptorpb.FieldDescriptorProto_LABEL_REPEATED && !m.field.IsMap(),
		"field is not repeated but got repeated rules",
	)

	m.checkMinMax(r.MinItems, r.MaxItems)

	if r.GetUnique() {
		m.assert(m.field.Type() != descriptorpb.FieldDescriptorProto_TYPE_MESSAGE,
			"unique rule is only applicable for scalar types")
	}

	m.checkFieldRules(r.Items)
}

// checkMap validates the map rules.
func (m *validateField) checkMap(r *validate.MapRules) {
	m.assert(
		m.field.IsMap(),
		"field is not a map but got map rules",
	)

	m.checkMinMax(r.MinPairs, r.MaxPairs)

	m.checkFieldRules(r.Keys)
	m.checkFieldRules(r.Values)
}

// checkAny validates the any rules.
func (m *validateField) checkAny(r *validate.AnyRules) {
	m.checkIns(len(r.In), len(r.NotIn))
}

// checkDuration validates the duration rules.
func (m *validateField) checkDuration(r *validate.DurationRules) {
	checkNums(m,
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

// checkTimestamp validates the timestamp rules.
func (m *validateField) checkTimestamp(r *validate.TimestampRules) {
	checkNums(m, 0, 0,
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

// checkLen checks that the `len` rule is not used with `min_len` or `max_len`
func (m *validateField) checkLen(len, min, max *uint64) {
	if len == nil {
		return
	}

	m.assert(min == nil,
		"cannot have both `len` and `min_len` rules on the same field")

	m.assert(max == nil,
		"cannot have both `len` and `max_len` rules on the same field")
}

// checkMinMax checks that the `min` and `max` rules are used correctly
func (m *validateField) checkMinMax(min, max *uint64) {
	if min == nil || max == nil {
		return
	}

	m.assert(*min <= *max,
		"`min` value is greater than `max` value")
}

// checkWellKnownRegex checks that the `well_known_regex` rule is used correctly
func (m *validateField) checkWellKnownRegex(wk validate.KnownRegex, r *validate.StringRules) {
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

// checkPattern checks that the `pattern` rule is used correctly
func (m *validateField) checkPattern(p *string, in int) {
	if p != nil {
		m.assert(in == 0, "regex `pattern` and `in` rules are incompatible")
		_, err := regexp.Compile(*p)
		m.assert(err != nil, "unable to parse regex `pattern`")
	}
}

// checkDur checks that the `duration` rule is used correctly
func (m *validateField) checkDur(d *durationpb.Duration) *time.Duration {
	if d == nil {
		return nil
	}

	dur, err := d.AsDuration(), d.CheckValid()
	m.assert(err == nil, "could not resolve duration")
	return &dur
}

// checkTS checks that the `timestamp` rule is used correctly
func (m *validateField) checkTS(ts *timestamppb.Timestamp) *int64 {
	if ts == nil {
		return nil
	}

	t, err := ts.AsTime(), ts.CheckValid()
	m.assert(err == nil, "could not resolve timestamp")
	return proto.Int64(t.UnixNano())
}

// noCustomRules asserts that the given custom rules are not used.
func (m *validateField) noCustomRules(r *validate.FieldConstraints) {
	m.assert(len(r.GetCel()) == 0, "custom rules are not supported for this field type")
}
