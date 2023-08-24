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
	"regexp"
	"time"
	"unicode/utf8"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"github.com/bufbuild/buf/private/pkg/protosource"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	unknown         = ""
	httpHeaderName  = "^:?[0-9a-zA-Z!#$%&'*+-.^_|~\x60]+$"
	httpHeaderValue = "^[^\u0000-\u0008\u000A-\u001F\u007F]*$"
	headerString    = "^[^\u0000\u000A\u000D]*$" // For non-strict validation.
	// Map from well-known regex to a regex pattern.
	regexMap = map[string]*string{
		"UNKNOWN":           &unknown,
		"HTTP_HEADER_NAME":  &httpHeaderName,
		"HTTP_HEADER_VALUE": &httpHeaderValue,
		"HEADER_STRING":     &headerString,
	}
)

// validateField is a validate Field.
type validateField struct {
	module   *validateModule
	field    protosource.Field
	location protosource.Location
}

// newValidateField returns a new validate validateField.
func newValidateField(module *validateModule, field protosource.Field) *validateField {
	return &validateField{
		module:   module,
		field:    field,
		location: field.OptionExtensionLocation(validate.E_Field),
	}
}

// CheckFieldRules checks the rules for the field.
func (m *validateField) CheckFieldRules(rules *validate.FieldConstraints) {
	if rules == nil {
		return
	}
	if wkt := LookupWellKnownType(m.field.TypeName()); wkt.Valid() && wkt == AnyWKT {
		m.validateNoCustomRulesApplied(rules)
	}
	switch r := rules.Type.(type) {
	case *validate.FieldConstraints_Float:
		m.assertFieldTypeMatches(descriptorpb.FieldDescriptorProto_TYPE_FLOAT, FloatValueWKT)
		validateNumberField(m, len(r.Float.In), len(r.Float.NotIn), r.Float.Const, r.Float.Lt, r.Float.Lte, r.Float.Gt, r.Float.Gte)
	case *validate.FieldConstraints_Double:
		m.assertFieldTypeMatches(descriptorpb.FieldDescriptorProto_TYPE_DOUBLE, DoubleValueWKT)
		validateNumberField(m, len(r.Double.In), len(r.Double.NotIn), r.Double.Const, r.Double.Lt, r.Double.Lte, r.Double.Gt, r.Double.Gte)
	case *validate.FieldConstraints_Int32:
		m.assertFieldTypeMatches(descriptorpb.FieldDescriptorProto_TYPE_INT32, Int32ValueWKT)
		validateNumberField(m, len(r.Int32.In), len(r.Int32.NotIn), r.Int32.Const, r.Int32.Lt, r.Int32.Lte, r.Int32.Gt, r.Int32.Gte)
	case *validate.FieldConstraints_Int64:
		m.assertFieldTypeMatches(descriptorpb.FieldDescriptorProto_TYPE_INT64, Int64ValueWKT)
		validateNumberField(m, len(r.Int64.In), len(r.Int64.NotIn), r.Int64.Const, r.Int64.Lt, r.Int64.Lte, r.Int64.Gt, r.Int64.Gte)
	case *validate.FieldConstraints_Uint32:
		m.assertFieldTypeMatches(descriptorpb.FieldDescriptorProto_TYPE_UINT32, UInt32ValueWKT)
		validateNumberField(m, len(r.Uint32.In), len(r.Uint32.NotIn), r.Uint32.Const, r.Uint32.Lt, r.Uint32.Lte, r.Uint32.Gt, r.Uint32.Gte)
	case *validate.FieldConstraints_Uint64:
		m.assertFieldTypeMatches(descriptorpb.FieldDescriptorProto_TYPE_UINT64, UInt64ValueWKT)
		validateNumberField(m, len(r.Uint64.In), len(r.Uint64.NotIn), r.Uint64.Const, r.Uint64.Lt, r.Uint64.Lte, r.Uint64.Gt, r.Uint64.Gte)
	case *validate.FieldConstraints_Sint32:
		m.assertFieldTypeMatches(descriptorpb.FieldDescriptorProto_TYPE_SINT32, UnknownWKT)
		validateNumberField(m, len(r.Sint32.In), len(r.Sint32.NotIn), r.Sint32.Const, r.Sint32.Lt, r.Sint32.Lte, r.Sint32.Gt, r.Sint32.Gte)
	case *validate.FieldConstraints_Sint64:
		m.assertFieldTypeMatches(descriptorpb.FieldDescriptorProto_TYPE_SINT64, UnknownWKT)
		validateNumberField(m, len(r.Sint64.In), len(r.Sint64.NotIn), r.Sint64.Const, r.Sint64.Lt, r.Sint64.Lte, r.Sint64.Gt, r.Sint64.Gte)
	case *validate.FieldConstraints_Fixed32:
		m.assertFieldTypeMatches(descriptorpb.FieldDescriptorProto_TYPE_FIXED32, UnknownWKT)
		validateNumberField(m, len(r.Fixed32.In), len(r.Fixed32.NotIn), r.Fixed32.Const, r.Fixed32.Lt, r.Fixed32.Lte, r.Fixed32.Gt, r.Fixed32.Gte)
	case *validate.FieldConstraints_Fixed64:
		m.assertFieldTypeMatches(descriptorpb.FieldDescriptorProto_TYPE_FIXED64, UnknownWKT)
		validateNumberField(m, len(r.Fixed64.In), len(r.Fixed64.NotIn), r.Fixed64.Const, r.Fixed64.Lt, r.Fixed64.Lte, r.Fixed64.Gt, r.Fixed64.Gte)
	case *validate.FieldConstraints_Sfixed32:
		m.assertFieldTypeMatches(descriptorpb.FieldDescriptorProto_TYPE_SFIXED32, UnknownWKT)
		validateNumberField(m, len(r.Sfixed32.In), len(r.Sfixed32.NotIn), r.Sfixed32.Const, r.Sfixed32.Lt, r.Sfixed32.Lte, r.Sfixed32.Gt, r.Sfixed32.Gte)
	case *validate.FieldConstraints_Sfixed64:
		m.assertFieldTypeMatches(descriptorpb.FieldDescriptorProto_TYPE_SFIXED64, UnknownWKT)
		validateNumberField(m, len(r.Sfixed64.In), len(r.Sfixed64.NotIn), r.Sfixed64.Const, r.Sfixed64.Lt, r.Sfixed64.Lte, r.Sfixed64.Gt, r.Sfixed64.Gte)
	case *validate.FieldConstraints_Bool:
		m.assertFieldTypeMatches(descriptorpb.FieldDescriptorProto_TYPE_BOOL, BoolValueWKT)
	case *validate.FieldConstraints_String_:
		m.assertFieldTypeMatches(descriptorpb.FieldDescriptorProto_TYPE_STRING, StringValueWKT)
		m.validateStringField(r.String_)
	case *validate.FieldConstraints_Bytes:
		m.assertFieldTypeMatches(descriptorpb.FieldDescriptorProto_TYPE_BYTES, BytesValueWKT)
		m.validateBytesField(r.Bytes)
	case *validate.FieldConstraints_Enum:
		m.assertFieldTypeMatches(descriptorpb.FieldDescriptorProto_TYPE_ENUM, UnknownWKT)
		m.validateEnumField(r.Enum)
	case *validate.FieldConstraints_Repeated:
		m.validateRepeatedField(r.Repeated)
	case *validate.FieldConstraints_Map:
		m.validateMapField(r.Map)
	case *validate.FieldConstraints_Any:
		m.validateAnyField(r.Any)
	case *validate.FieldConstraints_Duration:
		m.validateDurationField(r.Duration)
	case *validate.FieldConstraints_Timestamp:
		m.validateTimestampField(r.Timestamp)
	}
}

// assertFieldTypeMatches asserts that the field type is the same as the given type.
func (m *validateField) assertFieldTypeMatches(pt descriptorpb.FieldDescriptorProto_Type, wrapper WellKnownType) {
	if wrapper != UnknownWKT {
		if emb := m.field.Embed(m.module.files...); emb != nil {
			if wkt := LookupWellKnownType(emb.Name()); wkt.Valid() && wkt == wrapper {
				field := emb.Fields()[0]
				newValidateField(m.module, field).assertFieldTypeMatches(field.Type(), UnknownWKT)
				return
			}
		}
	}

	expr := m.field.Type() == pt
	m.assertf(
		expr,
		"expected rules for %s but got %s",
		m.field.Type(),
		pt.String(),
	)
}

// checkIns asserts that the given `in` and `not_in` rules are valid.
func (m *validateField) checkIns(in, notIn int) {
	m.assertf(in == 0 || notIn == 0,
		"cannot have both `in` and `not_in` rules on the same field")
}

// assertf asserts that the given expression is true and adds an error if not.
func (m *validateField) assertf(expr bool, format string, v ...interface{}) {
	if !expr {
		m.module.add(m.field, m.location, nil, format, v...)
	}
}

// validateStringField asserts that the given string rules are valid.
func (m *validateField) validateStringField(r *validate.StringRules) {
	m.checkLen(r.Len, r.MinLen, r.MaxLen)
	m.checkLen(r.LenBytes, r.MinBytes, r.MaxBytes)
	m.checkMinMax(r.MinLen, r.MaxLen)
	m.checkMinMax(r.MinBytes, r.MaxBytes)
	m.checkIns(len(r.In), len(r.NotIn))
	m.checkWellKnownRegex(r.GetWellKnownRegex(), r)
	m.checkPattern(r.Pattern, len(r.In))

	if r.MaxLen != nil {
		max := int(r.GetMaxLen())
		m.assertf(utf8.RuneCountInString(r.GetPrefix()) <= max, "`prefix` length exceeds the `max_len`")
		m.assertf(utf8.RuneCountInString(r.GetSuffix()) <= max, "`suffix` length exceeds the `max_len`")
		m.assertf(utf8.RuneCountInString(r.GetContains()) <= max, "`contains` length exceeds the `max_len`")

		m.assertf(r.MaxBytes == nil || r.GetMaxBytes() >= r.GetMaxLen(),
			"`max_len` cannot exceed `max_bytes`")
	}

	if r.MaxBytes != nil {
		max := int(r.GetMaxBytes())
		m.assertf(len(r.GetPrefix()) <= max, "`prefix` length exceeds the `max_bytes`")
		m.assertf(len(r.GetSuffix()) <= max, "`suffix` length exceeds the `max_bytes`")
		m.assertf(len(r.GetContains()) <= max, "`contains` length exceeds the `max_bytes`")
	}
}

// validateEnumField asserts that the given enum rules are valid.
func (m *validateField) validateEnumField(r *validate.EnumRules) {
	m.checkIns(len(r.In), len(r.NotIn))

	if r.GetDefinedOnly() && len(r.In) > 0 {
		enum := m.field.Enum(m.module.files...)
		if enum == nil {
			return
		}
		defined := enum.Values()
		vals := make(map[int]struct{}, len(defined))

		for _, val := range defined {
			vals[val.Number()] = struct{}{}
		}

		for _, in := range r.In {
			_, ok := vals[int(in)]
			m.assertf(ok, "undefined `in` value (%d) conflicts with `defined_only` rule", in)
		}
	}
}

// validateBytesField asserts that the given bytes rules are valid.
func (m *validateField) validateBytesField(r *validate.BytesRules) {
	m.checkMinMax(r.MinLen, r.MaxLen)
	m.checkIns(len(r.In), len(r.NotIn))
	m.checkPattern(r.Pattern, len(r.In))

	if r.MaxLen != nil {
		max := int(r.GetMaxLen())
		m.assertf(len(r.GetPrefix()) <= max, "`prefix` length exceeds the `max_len`")
		m.assertf(len(r.GetSuffix()) <= max, "`suffix` length exceeds the `max_len`")
		m.assertf(len(r.GetContains()) <= max, "`contains` length exceeds the `max_len`")
	}
}

// validateRepeatedField validates the repeated rules.
func (m *validateField) validateRepeatedField(r *validate.RepeatedRules) {
	m.assertf(
		m.field.Label() == descriptorpb.FieldDescriptorProto_LABEL_REPEATED && !m.field.IsMap(),
		"field is not repeated but got repeated rules",
	)

	m.checkMinMax(r.MinItems, r.MaxItems)

	if r.GetUnique() {
		m.assertf(m.field.Type() != descriptorpb.FieldDescriptorProto_TYPE_MESSAGE,
			"unique rule is only applicable for scalar types")
	}

	m.CheckFieldRules(r.Items)
}

// validateMapField validates the map rules.
func (m *validateField) validateMapField(r *validate.MapRules) {
	m.assertf(
		m.field.IsMap(),
		"field is not a map but got map rules",
	)

	m.checkMinMax(r.MinPairs, r.MaxPairs)

	m.CheckFieldRules(r.Keys)
	m.CheckFieldRules(r.Values)
}

// validateAnyField validates the any rules.
func (m *validateField) validateAnyField(r *validate.AnyRules) {
	m.checkIns(len(r.In), len(r.NotIn))
}

// validateDurationField validates the duration rules.
func (m *validateField) validateDurationField(r *validate.DurationRules) {
	validateNumberField(m,
		len(r.GetIn()),
		len(r.GetNotIn()),
		m.checkDur(r.GetConst()),
		m.checkDur(r.GetLt()),
		m.checkDur(r.GetLte()),
		m.checkDur(r.GetGt()),
		m.checkDur(r.GetGte()))

	for _, v := range r.GetIn() {
		m.assertf(v != nil, "cannot have nil values in `in`")
		m.checkDur(v)
	}

	for _, v := range r.GetNotIn() {
		m.assertf(v != nil, "cannot have nil values in `not_in`")
		m.checkDur(v)
	}
}

// validateTimestampField validates the timestamp rules.
func (m *validateField) validateTimestampField(r *validate.TimestampRules) {
	validateNumberField(m, 0, 0,
		m.checkTS(r.GetConst()),
		m.checkTS(r.GetLt()),
		m.checkTS(r.GetLte()),
		m.checkTS(r.GetGt()),
		m.checkTS(r.GetGte()))

	m.assertf((r.LtNow == nil && r.GtNow == nil) || (r.Lt == nil && r.Lte == nil && r.Gt == nil && r.Gte == nil),
		"`now` rules cannot be mixed with absolute `lt/gt` rules")

	m.assertf(r.Within == nil || (r.Lt == nil && r.Lte == nil && r.Gt == nil && r.Gte == nil),
		"`within` rule cannot be used with absolute `lt/gt` rules")

	m.assertf(r.LtNow == nil || r.GtNow == nil,
		"both `now` rules cannot be used together")

	dur := m.checkDur(r.Within)
	m.assertf(dur == nil || *dur > 0,
		"`within` rule must be positive and non-zero")
}

// checkLen checks that the `len` rule is not used with `min_len` or `max_len`
func (m *validateField) checkLen(length, min, max *uint64) {
	if length == nil {
		return
	}

	m.assertf(min == nil,
		"cannot have both `len` and `min_len` rules on the same field")

	m.assertf(max == nil,
		"cannot have both `len` and `max_len` rules on the same field")
}

// checkMinMax checks that the `min` and `max` rules are used correctly
func (m *validateField) checkMinMax(min, max *uint64) {
	if min == nil || max == nil {
		return
	}

	m.assertf(*min <= *max,
		"`min` value is greater than `max` value")
}

// checkWellKnownRegex checks that the `well_known_regex` rule is used correctly
func (m *validateField) checkWellKnownRegex(wk validate.KnownRegex, r *validate.StringRules) {
	if wk != 0 {
		m.assertf(r.Pattern == nil, "regex `well_known_regex` and regex `pattern` are incompatible")
		nonStrict := r.Strict != nil && !*r.Strict
		if (wk.String() == "HTTP_HEADER_NAME" || wk.String() == "HTTP_HEADER_VALUE") && nonStrict {
			// Use non-strict header validation.
			r.Pattern = regexMap["HEADER_STRING"]
		} else {
			r.Pattern = regexMap[wk.String()]
		}
	}
}

// checkPattern checks that the `pattern` rule is used correctly
func (m *validateField) checkPattern(p *string, in int) {
	if p != nil {
		m.assertf(in == 0, "regex `pattern` and `in` rules are incompatible")
		_, err := regexp.Compile(*p)
		m.assertf(err != nil, "unable to parse regex `pattern`")
	}
}

// checkDur checks that the `duration` rule is used correctly
func (m *validateField) checkDur(d *durationpb.Duration) *time.Duration {
	if d == nil {
		return nil
	}

	dur, err := d.AsDuration(), d.CheckValid()
	m.assertf(err == nil, "could not resolve duration")
	return &dur
}

// checkTS checks that the `timestamp` rule is used correctly
func (m *validateField) checkTS(ts *timestamppb.Timestamp) *int64 {
	if ts == nil {
		return nil
	}

	t, err := ts.AsTime(), ts.CheckValid()
	m.assertf(err == nil, "could not resolve timestamp")
	return proto.Int64(t.UnixNano())
}

// validateNoCustomRulesApplied asserts that the given custom rules are not used.
func (m *validateField) validateNoCustomRulesApplied(r *validate.FieldConstraints) {
	m.assertf(len(r.GetCel()) == 0, "custom rules are not supported for this field type")
}
