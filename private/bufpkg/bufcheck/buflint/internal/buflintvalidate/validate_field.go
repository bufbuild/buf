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

	"github.com/bufbuild/buf/private/gen/proto/go/buf/validate"
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
	add      func(protosource.Descriptor, protosource.Location, []protosource.Location, string, ...interface{})
	files    []protosource.File
	field    protosource.Field
	location protosource.Location
}

func (m *validateField) CheckFieldRules(rules *validate.FieldConstraints) {
	if rules == nil {
		return
	}
	if wkt := lookupWellKnownType(m.field.TypeName()); wkt.valid() && wkt == anyWKT {
		m.validateNoCustomRulesApplied(rules)
	}
	switch r := rules.Type.(type) {
	case *validate.FieldConstraints_Float:
		m.assertFieldTypeMatches(descriptorpb.FieldDescriptorProto_TYPE_FLOAT, floatValueWKT)
		gt, gte, lt, lte := resolveLimits[
			float32,
			*validate.FloatRules_Gt,
			*validate.FloatRules_Gte,
			*validate.FloatRules_Lt,
			*validate.FloatRules_Lte,
		](r.Float, r.Float.GreaterThan, r.Float.LessThan)
		validateNumberField(m, len(r.Float.In), len(r.Float.NotIn), r.Float.Const, gt, gte, lt, lte)
	case *validate.FieldConstraints_Double:
		m.assertFieldTypeMatches(descriptorpb.FieldDescriptorProto_TYPE_DOUBLE, doubleValueWKT)
		gt, gte, lt, lte := resolveLimits[
			float64,
			*validate.DoubleRules_Gt,
			*validate.DoubleRules_Gte,
			*validate.DoubleRules_Lt,
			*validate.DoubleRules_Lte,
		](r.Double, r.Double.GreaterThan, r.Double.LessThan)
		validateNumberField(m, len(r.Double.In), len(r.Double.NotIn), r.Double.Const, gt, gte, lt, lte)
	case *validate.FieldConstraints_Int32:
		m.assertFieldTypeMatches(descriptorpb.FieldDescriptorProto_TYPE_INT32, int32ValueWKT)
		gt, gte, lt, lte := resolveLimits[
			int32,
			*validate.Int32Rules_Gt,
			*validate.Int32Rules_Gte,
			*validate.Int32Rules_Lt,
			*validate.Int32Rules_Lte,
		](r.Int32, r.Int32.GreaterThan, r.Int32.LessThan)
		validateNumberField(m, len(r.Int32.In), len(r.Int32.NotIn), r.Int32.Const, gt, gte, lt, lte)
	case *validate.FieldConstraints_Int64:
		m.assertFieldTypeMatches(descriptorpb.FieldDescriptorProto_TYPE_INT64, int64ValueWKT)
		gt, gte, lt, lte := resolveLimits[
			int64,
			*validate.Int64Rules_Gt,
			*validate.Int64Rules_Gte,
			*validate.Int64Rules_Lt,
			*validate.Int64Rules_Lte,
		](r.Int64, r.Int64.GreaterThan, r.Int64.LessThan)
		validateNumberField(m, len(r.Int64.In), len(r.Int64.NotIn), r.Int64.Const, gt, gte, lt, lte)
	case *validate.FieldConstraints_Uint32:
		m.assertFieldTypeMatches(descriptorpb.FieldDescriptorProto_TYPE_UINT32, uInt32ValueWKT)
		gt, gte, lt, lte := resolveLimits[
			uint32,
			*validate.UInt32Rules_Gt,
			*validate.UInt32Rules_Gte,
			*validate.UInt32Rules_Lt,
			*validate.UInt32Rules_Lte,
		](r.Uint32, r.Uint32.GreaterThan, r.Uint32.LessThan)
		validateNumberField(m, len(r.Uint32.In), len(r.Uint32.NotIn), r.Uint32.Const, gt, gte, lt, lte)
	case *validate.FieldConstraints_Uint64:
		m.assertFieldTypeMatches(descriptorpb.FieldDescriptorProto_TYPE_UINT64, uInt64ValueWKT)
		gt, gte, lt, lte := resolveLimits[
			uint64,
			*validate.UInt64Rules_Gt,
			*validate.UInt64Rules_Gte,
			*validate.UInt64Rules_Lt,
			*validate.UInt64Rules_Lte,
		](r.Uint64, r.Uint64.GreaterThan, r.Uint64.LessThan)
		validateNumberField(m, len(r.Uint64.In), len(r.Uint64.NotIn), r.Uint64.Const, gt, gte, lt, lte)
	case *validate.FieldConstraints_Sint32:
		m.assertFieldTypeMatches(descriptorpb.FieldDescriptorProto_TYPE_SINT32, unknownWKT)
		gt, gte, lt, lte := resolveLimits[
			int32,
			*validate.SInt32Rules_Gt,
			*validate.SInt32Rules_Gte,
			*validate.SInt32Rules_Lt,
			*validate.SInt32Rules_Lte,
		](r.Sint32, r.Sint32.GreaterThan, r.Sint32.LessThan)
		validateNumberField(m, len(r.Sint32.In), len(r.Sint32.NotIn), r.Sint32.Const, gt, gte, lt, lte)
	case *validate.FieldConstraints_Sint64:
		m.assertFieldTypeMatches(descriptorpb.FieldDescriptorProto_TYPE_SINT64, unknownWKT)
		gt, gte, lt, lte := resolveLimits[
			int64,
			*validate.SInt64Rules_Gt,
			*validate.SInt64Rules_Gte,
			*validate.SInt64Rules_Lt,
			*validate.SInt64Rules_Lte,
		](r.Sint64, r.Sint64.GreaterThan, r.Sint64.LessThan)
		validateNumberField(m, len(r.Sint64.In), len(r.Sint64.NotIn), r.Sint64.Const, gt, gte, lt, lte)
	case *validate.FieldConstraints_Fixed32:
		m.assertFieldTypeMatches(descriptorpb.FieldDescriptorProto_TYPE_FIXED32, unknownWKT)
		gt, gte, lt, lte := resolveLimits[
			uint32,
			*validate.Fixed32Rules_Gt,
			*validate.Fixed32Rules_Gte,
			*validate.Fixed32Rules_Lt,
			*validate.Fixed32Rules_Lte,
		](r.Fixed32, r.Fixed32.GreaterThan, r.Fixed32.LessThan)
		validateNumberField(m, len(r.Fixed32.In), len(r.Fixed32.NotIn), r.Fixed32.Const, gt, gte, lt, lte)
	case *validate.FieldConstraints_Fixed64:
		m.assertFieldTypeMatches(descriptorpb.FieldDescriptorProto_TYPE_FIXED64, unknownWKT)
		gt, gte, lt, lte := resolveLimits[
			uint64,
			*validate.Fixed64Rules_Gt,
			*validate.Fixed64Rules_Gte,
			*validate.Fixed64Rules_Lt,
			*validate.Fixed64Rules_Lte,
		](r.Fixed64, r.Fixed64.GreaterThan, r.Fixed64.LessThan)
		validateNumberField(m, len(r.Fixed64.In), len(r.Fixed64.NotIn), r.Fixed64.Const, gt, gte, lt, lte)
	case *validate.FieldConstraints_Sfixed32:
		m.assertFieldTypeMatches(descriptorpb.FieldDescriptorProto_TYPE_SFIXED32, unknownWKT)
		gt, gte, lt, lte := resolveLimits[
			int32,
			*validate.SFixed32Rules_Gt,
			*validate.SFixed32Rules_Gte,
			*validate.SFixed32Rules_Lt,
			*validate.SFixed32Rules_Lte,
		](r.Sfixed32, r.Sfixed32.GreaterThan, r.Sfixed32.LessThan)
		validateNumberField(m, len(r.Sfixed32.In), len(r.Sfixed32.NotIn), r.Sfixed32.Const, gt, gte, lt, lte)
	case *validate.FieldConstraints_Sfixed64:
		m.assertFieldTypeMatches(descriptorpb.FieldDescriptorProto_TYPE_SFIXED64, unknownWKT)
		gt, gte, lt, lte := resolveLimits[
			int64,
			*validate.SFixed64Rules_Gt,
			*validate.SFixed64Rules_Gte,
			*validate.SFixed64Rules_Lt,
			*validate.SFixed64Rules_Lte,
		](r.Sfixed64, r.Sfixed64.GreaterThan, r.Sfixed64.LessThan)
		validateNumberField(m, len(r.Sfixed64.In), len(r.Sfixed64.NotIn), r.Sfixed64.Const, gt, gte, lt, lte)
	case *validate.FieldConstraints_Bool:
		m.assertFieldTypeMatches(descriptorpb.FieldDescriptorProto_TYPE_BOOL, boolValueWKT)
	case *validate.FieldConstraints_String_:
		m.assertFieldTypeMatches(descriptorpb.FieldDescriptorProto_TYPE_STRING, stringValueWKT)
		m.validateStringField(r.String_)
	case *validate.FieldConstraints_Bytes:
		m.assertFieldTypeMatches(descriptorpb.FieldDescriptorProto_TYPE_BYTES, bytesValueWKT)
		m.validateBytesField(r.Bytes)
	case *validate.FieldConstraints_Enum:
		m.assertFieldTypeMatches(descriptorpb.FieldDescriptorProto_TYPE_ENUM, unknownWKT)
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

func (m *validateField) assertFieldTypeMatches(pt descriptorpb.FieldDescriptorProto_Type, wrapper wellKnownType) {
	if wrapper != unknownWKT {
		if emb := embed(m.field, m.files...); emb != nil {
			if wkt := lookupWellKnownType(emb.Name()); wkt.valid() && wkt == wrapper {
				field := emb.Fields()[0]
				NewValidateField(m.add, m.files, field).assertFieldTypeMatches(field.Type(), unknownWKT)
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

func (m *validateField) checkIns(in, notIn int) {
	m.assertf(in == 0 || notIn == 0,
		"cannot have both in and not_in rules on the same field")
}

func (m *validateField) assertf(expr bool, format string, v ...interface{}) {
	if !expr {
		m.add(m.field, m.location, nil, format, v...)
	}
}

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
		m.assertf(utf8.RuneCountInString(r.GetPrefix()) <= max, "prefix length exceeds the max_len")
		m.assertf(utf8.RuneCountInString(r.GetSuffix()) <= max, "suffix length exceeds the max_len")
		m.assertf(utf8.RuneCountInString(r.GetContains()) <= max, "contains length exceeds the max_len")

		m.assertf(r.MaxBytes == nil || r.GetMaxBytes() >= r.GetMaxLen(),
			"max_len cannot exceed max_bytes")
	}

	if r.MaxBytes != nil {
		max := int(r.GetMaxBytes())
		m.assertf(len(r.GetPrefix()) <= max, "prefix length exceeds the max_bytes")
		m.assertf(len(r.GetSuffix()) <= max, "suffix length exceeds the max_bytes")
		m.assertf(len(r.GetContains()) <= max, "contains length exceeds the max_bytes")
	}
}

func (m *validateField) validateEnumField(r *validate.EnumRules) {
	m.checkIns(len(r.In), len(r.NotIn))

	if r.GetDefinedOnly() && len(r.In) > 0 {
		enum := getEnum(m.field, m.files...)
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
			m.assertf(ok, "undefined in value (%d) conflicts with defined_only rule", in)
		}
	}
}

func (m *validateField) validateBytesField(r *validate.BytesRules) {
	m.checkMinMax(r.MinLen, r.MaxLen)
	m.checkIns(len(r.In), len(r.NotIn))
	m.checkPattern(r.Pattern, len(r.In))

	if r.MaxLen != nil {
		max := int(r.GetMaxLen())
		m.assertf(len(r.GetPrefix()) <= max, "prefix length exceeds the max_len")
		m.assertf(len(r.GetSuffix()) <= max, "suffix length exceeds the max_len")
		m.assertf(len(r.GetContains()) <= max, "contains length exceeds the max_len")
	}
}

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

func (m *validateField) validateMapField(r *validate.MapRules) {
	m.assertf(
		m.field.IsMap(),
		"field is not a map but got map rules",
	)

	m.checkMinMax(r.MinPairs, r.MaxPairs)

	m.CheckFieldRules(r.Keys)
	m.CheckFieldRules(r.Values)
}

func (m *validateField) validateAnyField(r *validate.AnyRules) {
	m.checkIns(len(r.In), len(r.NotIn))
}

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
		m.assertf(v != nil, "cannot have nil values in in")
		m.checkDur(v)
	}

	for _, v := range r.GetNotIn() {
		m.assertf(v != nil, "cannot have nil values in not_in")
		m.checkDur(v)
	}
}

func (m *validateField) validateTimestampField(r *validate.TimestampRules) {
	validateNumberField(m, 0, 0,
		m.checkTS(r.GetConst()),
		m.checkTS(r.GetLt()),
		m.checkTS(r.GetLte()),
		m.checkTS(r.GetGt()),
		m.checkTS(r.GetGte()))

	var gt, gte, lt, lte *timestamppb.Timestamp
	var ltNow, gtNow *bool

	switch r.GreaterThan.(type) {
	case *validate.TimestampRules_Gt:
		n := r.GetGt()
		gt = n
	case *validate.TimestampRules_Gte:
		n := r.GetGte()
		gte = n
	case *validate.TimestampRules_GtNow:
		n := r.GetGtNow()
		gtNow = &n
	}
	switch r.LessThan.(type) {
	case *validate.TimestampRules_Lt:
		n := r.GetLt()
		lt = n
	case *validate.TimestampRules_Lte:
		n := r.GetLte()
		lte = n
	case *validate.TimestampRules_LtNow:
		n := r.GetLtNow()
		ltNow = &n
	}

	m.assertf((ltNow == nil && gtNow == nil) || (lt == nil && lte == nil && gt == nil && gte == nil),
		"now rules cannot be mixed with absolute lt/gt rules")

	m.assertf(r.Within == nil || (lt == nil && lte == nil && gt == nil && gte == nil),
		"within rule cannot be used with absolute lt/gt rules")

	m.assertf(ltNow == nil || gtNow == nil,
		"both now rules cannot be used together")

	dur := m.checkDur(r.Within)
	m.assertf(dur == nil || *dur > 0,
		"within rule must be positive and non-zero")
}

func (m *validateField) checkLen(length, min, max *uint64) {
	if length == nil {
		return
	}

	m.assertf(min == nil,
		"cannot have both len and min_len rules on the same field")

	m.assertf(max == nil,
		"cannot have both len and max_len rules on the same field")
}

func (m *validateField) checkMinMax(min, max *uint64) {
	if min == nil || max == nil {
		return
	}

	m.assertf(*min <= *max,
		"min value is greater than max value")
}

func (m *validateField) checkWellKnownRegex(wk validate.KnownRegex, r *validate.StringRules) {
	if wk != 0 {
		m.assertf(r.Pattern == nil, "regex well_known_regex and regex pattern are incompatible")
		nonStrict := r.Strict != nil && !*r.Strict
		if (wk.String() == "HTTP_HEADER_NAME" || wk.String() == "HTTP_HEADER_VALUE") && nonStrict {
			// Use non-strict header validation.
			r.Pattern = regexMap["HEADER_STRING"]
		} else {
			r.Pattern = regexMap[wk.String()]
		}
	}
}

func (m *validateField) checkPattern(p *string, in int) {
	if p != nil {
		m.assertf(in == 0, "regex pattern and in rules are incompatible")
		_, err := regexp.Compile(*p)
		m.assertf(err != nil, "unable to parse regex pattern")
	}
}

func (m *validateField) checkDur(d *durationpb.Duration) *time.Duration {
	if d == nil {
		return nil
	}

	dur, err := d.AsDuration(), d.CheckValid()
	m.assertf(err == nil, "could not resolve duration")
	return &dur
}

func (m *validateField) checkTS(ts *timestamppb.Timestamp) *int64 {
	if ts == nil {
		return nil
	}

	t, err := ts.AsTime(), ts.CheckValid()
	m.assertf(err == nil, "could not resolve timestamp")
	return proto.Int64(t.UnixNano())
}

func (m *validateField) validateNoCustomRulesApplied(r *validate.FieldConstraints) {
	m.assertf(len(r.GetCel()) == 0, "custom rules are not supported for this field type")
}
