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
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// TODO: reivisit type validateField
// TODO: consistent lint message tone/language
// TODO: in cel linting, check no google.protobuf.Any is used (if this check makes sense).
// TODO: report at one location or both location
// TODO: rename file names

const (
	// https://buf.build/bufbuild/protovalidate/docs/main:buf.validate#buf.validate.FieldConstraints
	floatRulesFieldNumber     = 1
	doubleRulesFieldNumber    = 2
	int32RulesFieldNumber     = 3
	int64RulesFieldNumber     = 4
	uInt32RulesFieldNumber    = 5
	uInt64RulesFieldNumber    = 6
	sInt32RulesFieldNumber    = 7
	sInt64RulesFieldNumber    = 8
	fixed32RulesFieldNumber   = 9
	fixed64RulesFieldNumber   = 10
	sFixed32RulesFieldNumber  = 11
	sFixed64RulesFieldNumber  = 12
	boolRulesFieldNumber      = 13
	stringRulesFieldNumber    = 14
	bytesRulesFieldNumber     = 15
	enumRulesFieldNumber      = 16
	repeatedRulesFieldNumber  = 18
	mapRulesFieldNumber       = 19
	anyRulesFieldNumber       = 20
	durationRulesFieldNumber  = 21
	timestampRulesFieldNumber = 22
)

var (
	// Some rules can only be defined for fields of a specific primitive type.
	// For example, SFixed64Rules can only be defined on a field of type sfixed64.
	// Some rules can only be defined for fields of a specific message type. For
	// example, TimestampRules can only be defined on fields of type google.protobuf.Timestamp.
	// Others can be defined on either fields of a certain primitive type or fields
	// of a certain message type. For example, Int32Rules can be defined on either
	// a int32 field or a google.protobuf.Int32Value field.
	fieldNumberToAllowedProtoType = map[int32]descriptorpb.FieldDescriptorProto_Type{
		floatRulesFieldNumber:    descriptorpb.FieldDescriptorProto_TYPE_FLOAT,
		doubleRulesFieldNumber:   descriptorpb.FieldDescriptorProto_TYPE_DOUBLE,
		int32RulesFieldNumber:    descriptorpb.FieldDescriptorProto_TYPE_INT32,
		int64RulesFieldNumber:    descriptorpb.FieldDescriptorProto_TYPE_INT64,
		uInt32RulesFieldNumber:   descriptorpb.FieldDescriptorProto_TYPE_UINT32,
		uInt64RulesFieldNumber:   descriptorpb.FieldDescriptorProto_TYPE_UINT64,
		sInt32RulesFieldNumber:   descriptorpb.FieldDescriptorProto_TYPE_SINT32,
		sInt64RulesFieldNumber:   descriptorpb.FieldDescriptorProto_TYPE_SINT64,
		fixed32RulesFieldNumber:  descriptorpb.FieldDescriptorProto_TYPE_FIXED32,
		fixed64RulesFieldNumber:  descriptorpb.FieldDescriptorProto_TYPE_FIXED64,
		sFixed32RulesFieldNumber: descriptorpb.FieldDescriptorProto_TYPE_SFIXED32,
		sFixed64RulesFieldNumber: descriptorpb.FieldDescriptorProto_TYPE_SFIXED64,
		boolRulesFieldNumber:     descriptorpb.FieldDescriptorProto_TYPE_BOOL,
		stringRulesFieldNumber:   descriptorpb.FieldDescriptorProto_TYPE_STRING,
		bytesRulesFieldNumber:    descriptorpb.FieldDescriptorProto_TYPE_BYTES,
		enumRulesFieldNumber:     descriptorpb.FieldDescriptorProto_TYPE_ENUM,
	}
	fieldNumberToAllowedMessageName = map[int32]protoreflect.FullName{
		floatRulesFieldNumber:     (&wrapperspb.FloatValue{}).ProtoReflect().Descriptor().FullName(),
		doubleRulesFieldNumber:    (&wrapperspb.DoubleValue{}).ProtoReflect().Descriptor().FullName(),
		int32RulesFieldNumber:     (&wrapperspb.Int32Value{}).ProtoReflect().Descriptor().FullName(),
		int64RulesFieldNumber:     (&wrapperspb.Int64Value{}).ProtoReflect().Descriptor().FullName(),
		uInt32RulesFieldNumber:    (&wrapperspb.UInt32Value{}).ProtoReflect().Descriptor().FullName(),
		uInt64RulesFieldNumber:    (&wrapperspb.UInt64Value{}).ProtoReflect().Descriptor().FullName(),
		boolRulesFieldNumber:      (&wrapperspb.BoolValue{}).ProtoReflect().Descriptor().FullName(),
		stringRulesFieldNumber:    (&wrapperspb.StringValue{}).ProtoReflect().Descriptor().FullName(),
		bytesRulesFieldNumber:     (&wrapperspb.BytesValue{}).ProtoReflect().Descriptor().FullName(),
		anyRulesFieldNumber:       (&anypb.Any{}).ProtoReflect().Descriptor().FullName(),
		durationRulesFieldNumber:  (&durationpb.Duration{}).ProtoReflect().Descriptor().FullName(),
		timestampRulesFieldNumber: (&timestamppb.Timestamp{}).ProtoReflect().Descriptor().FullName(),
	}
	typeOneofDescriptor = validate.File_buf_validate_validate_proto.Messages().ByName("FieldConstraints").Oneofs().ByName("type")
	// https://buf.build/bufbuild/protovalidate/file/v0.4.3:buf/validate/validate.proto#L2846
	wellKnownHttpHeaderNamePattern = "^:?[0-9a-zA-Z!#$%&'*+-.^_|~\x60]+$"
	// https://buf.build/bufbuild/protovalidate/file/v0.4.3:buf/validate/validate.proto#L2853
	wellKnownHttpHeaderValuePattern = "^[^\u0000-\u0008\u000A-\u001F\u007F]*$"
	// https://buf.build/bufbuild/protovalidate/file/v0.4.3:buf/validate/validate.proto#L2854
	wellKnownHeaderStringPattern = "^[^\u0000\u000A\u000D]*$" // For non-strict validation.
)

// validateField is a validate Field.
type validateField struct {
	add      func(protosource.Descriptor, protosource.Location, []protosource.Location, string, ...interface{})
	files    []protosource.File
	field    protosource.Field
	location protosource.Location
}

func newValidateField(
	add func(protosource.Descriptor, protosource.Location, []protosource.Location, string, ...interface{}),
	files []protosource.File,
	field protosource.Field,
) *validateField {
	return &validateField{
		add:      add,
		files:    files,
		field:    field,
		location: field.OptionExtensionLocation(validate.E_Field),
	}
}

func (m *validateField) CheckConstraintsForField(fieldConstraints *validate.FieldConstraints, field protosource.Field) {
	if fieldConstraints == nil {
		return
	}
	fieldConstraintsMessage := fieldConstraints.ProtoReflect()
	typeField := fieldConstraintsMessage.WhichOneof(typeOneofDescriptor)
	if typeField == nil {
		return
	}
	typeFieldNumber := int32(typeField.Number())
	if typeFieldNumber == mapRulesFieldNumber {
		m.validateMapField(fieldConstraints.GetMap(), field)
		return
	}
	if typeFieldNumber == repeatedRulesFieldNumber {
		m.validateRepeatedField(fieldConstraints.GetRepeated(), field)
		return
	}
	checkTypeMatch(m, field, typeFieldNumber)
	if floatRulesFieldNumber <= typeFieldNumber && typeFieldNumber <= sFixed64RulesFieldNumber {
		numberRulesMessage := fieldConstraintsMessage.Get(typeField).Message()
		validateNumberRulesMessage(m, field, typeFieldNumber, numberRulesMessage)
		return
	}
	switch r := fieldConstraints.Type.(type) {
	case *validate.FieldConstraints_Bool:
		// Bool rules only have `const` and does not need validation.
	case *validate.FieldConstraints_String_:
		m.validateStringField(r.String_)
	case *validate.FieldConstraints_Bytes:
		m.validateBytesField(r.Bytes)
	case *validate.FieldConstraints_Enum:
		m.validateEnumField(r.Enum, field)
	case *validate.FieldConstraints_Any:
		m.validateAnyField(r.Any)
	case *validate.FieldConstraints_Duration:
		m.validateDurationField(r.Duration)
	case *validate.FieldConstraints_Timestamp:
		m.validateTimestampField(r.Timestamp)
	}
}

func checkTypeMatch(validateField *validateField, field protosource.Field, ruleTag int32) {
	if field.Type() == descriptorpb.FieldDescriptorProto_TYPE_MESSAGE {
		expectedFieldMessageName, ok := fieldNumberToAllowedMessageName[ruleTag]
		if ok && string(expectedFieldMessageName) == field.TypeName() {
			return
		}
		validateField.add(
			field,
			validateField.location,
			nil,
			// TODO: instead of `rule tag 1`, say `FloatRules`.
			"rule tag %d should not be defined on a field of type %s",
			ruleTag,
			field.TypeName(),
		)
		return
	}
	expectedType, ok := fieldNumberToAllowedProtoType[ruleTag]
	if !ok {
		// TODO
		return
	}
	if expectedType != field.Type() {
		validateField.add(
			field,
			validateField.location,
			nil,
			// TODO: instead of `rule tag 1`, say `FloatRules`.
			"rule tag %d should not be defined on a field of type %v",
			ruleTag,
			field.Type(),
		)
	}
}

func (m *validateField) checkIns(
	in int,
	notIn int,
) {
	m.assertf(in == 0 || notIn == 0,
		"cannot have both in and not_in rules on the same field")
}

func (m *validateField) assertf(expr bool, format string, v ...interface{}) {
	if !expr {
		m.add(m.field, m.location, nil, format, v...)
	}
}

func (m *validateField) validateStringField(r *validate.StringRules) {
	if r.Len != nil {
		m.assertf(r.MinLen == nil, "cannot have both len and min_len on the same field")
		m.assertf(r.MaxLen == nil, "cannot have both len and max_len on the same field")
	}
	if r.LenBytes != nil {
		m.assertf(r.MinBytes == nil, "cannot have both len_bytes and min_bytes on the same field")
		m.assertf(r.MaxBytes == nil, "cannot have both len_bytes and max_bytes on the same field")
	}
	m.checkMinMax(r.MinLen, "min_len", r.MaxLen, "max_len")
	m.checkMinMax(r.MinBytes, "min_bytes", r.MaxBytes, "max_bytes")
	m.checkIns(len(r.In), len(r.NotIn))
	patternInEffect := r.Pattern
	wellKnownRegex := r.GetWellKnownRegex()
	nonStrict := r.Strict != nil && !*r.Strict
	switch wellKnownRegex {
	case validate.KnownRegex_KNOWN_REGEX_UNSPECIFIED:
		m.assertf(!nonStrict, "cannot specify strict without specifying well_known_regex")
	case validate.KnownRegex_KNOWN_REGEX_HTTP_HEADER_NAME:
		m.assertf(r.Pattern == nil, "regex well_known_regex and regex pattern are incompatible")
		patternInEffect = &wellKnownHttpHeaderNamePattern
		if nonStrict {
			patternInEffect = &wellKnownHeaderStringPattern
		}
	case validate.KnownRegex_KNOWN_REGEX_HTTP_HEADER_VALUE:
		m.assertf(r.Pattern == nil, "regex well_known_regex and regex pattern are incompatible")
		patternInEffect = &wellKnownHttpHeaderValuePattern
		if nonStrict {
			patternInEffect = &wellKnownHeaderStringPattern
		}
	}
	m.checkPattern(patternInEffect, len(r.In))
	if r.MaxLen != nil {
		max := r.GetMaxLen()
		m.assertf(uint64(utf8.RuneCountInString(r.GetPrefix())) <= max, "prefix length exceeds max_len")
		m.assertf(uint64(utf8.RuneCountInString(r.GetSuffix())) <= max, "suffix length exceeds max_len")
		m.assertf(uint64(utf8.RuneCountInString(r.GetContains())) <= max, "contains length exceeds max_len")

		m.assertf(r.MaxBytes == nil || r.GetMaxBytes() >= r.GetMaxLen(), "max_len cannot exceed max_bytes")
	}
	if r.MaxBytes != nil {
		max := r.GetMaxBytes()
		m.assertf(uint64(len(r.GetPrefix())) <= max, "prefix length exceeds the max_bytes")
		m.assertf(uint64(len(r.GetSuffix())) <= max, "suffix length exceeds the max_bytes")
		m.assertf(uint64(len(r.GetContains())) <= max, "contains length exceeds the max_bytes")
	}
}

func (m *validateField) validateEnumField(r *validate.EnumRules, field protosource.Field) {
	m.checkIns(len(r.In), len(r.NotIn))
	if !r.GetDefinedOnly() {
		return
	}
	if len(r.In) == 0 && len(r.NotIn) == 0 {
		return
	}
	enum := getEnum(field, m.files...)
	if enum == nil {
		// TODO: return error
		return
	}
	defined := enum.Values()
	vals := make(map[int]struct{}, len(defined))
	for _, val := range defined {
		vals[val.Number()] = struct{}{}
	}
	if len(r.In) > 0 {
		for _, in := range r.In {
			_, ok := vals[int(in)]
			m.assertf(ok, "undefined in value (%d) conflicts with defined_only rule", in)
		}
	}
	if len(r.NotIn) > 0 {
		for _, notIn := range r.NotIn {
			_, ok := vals[int(notIn)]
			m.assertf(ok, "undefined not_in value (%d) is redundant, as it is already rejected by defined_only")
		}
	}
}

func (m *validateField) validateBytesField(r *validate.BytesRules) {
	if r.Len != nil {
		m.assertf(r.MinLen == nil, "cannot have both len and min_len on the same field")
		m.assertf(r.MaxLen == nil, "cannot have both len and max_len on the same field")
	}
	m.checkMinMax(r.MinLen, "min_len", r.MaxLen, "max_len")
	m.checkIns(len(r.In), len(r.NotIn))
	m.checkPattern(r.Pattern, len(r.In))
	if r.MaxLen != nil {
		max := r.GetMaxLen()
		m.assertf(uint64(len(r.GetPrefix())) <= max, "prefix length exceeds max_len")
		m.assertf(uint64(len(r.GetSuffix())) <= max, "suffix length exceeds max_len")
		m.assertf(uint64(len(r.GetContains())) <= max, "contains length exceeds max_len")
	}
}

func (m *validateField) validateRepeatedField(r *validate.RepeatedRules, field protosource.Field) {
	m.assertf(
		field.Label() == descriptorpb.FieldDescriptorProto_LABEL_REPEATED && !field.IsMap(),
		"field is not repeated but got repeated rules",
	)

	m.checkMinMax(r.MinItems, "min_items", r.MaxItems, "max_items")

	if r.GetUnique() {
		m.assertf(field.Type() != descriptorpb.FieldDescriptorProto_TYPE_MESSAGE,
			"unique rule is only applicable for scalar types")
	}

	m.CheckConstraintsForField(r.Items, field)
}

func (m *validateField) validateMapField(r *validate.MapRules, field protosource.Field) {
	m.assertf(
		field.IsMap(),
		"field is not a map but got map rules",
	)

	m.checkMinMax(r.MinPairs, "min_pairs", r.MaxPairs, "max_pairs")

	mapMessage := embed(field, m.files...)
	// TODO: make sure it has two fields
	m.CheckConstraintsForField(r.Keys, mapMessage.Fields()[0])
	m.CheckConstraintsForField(r.Values, mapMessage.Fields()[1])
}

func (m *validateField) validateAnyField(r *validate.AnyRules) {
	m.checkIns(len(r.In), len(r.NotIn))
}

func (m *validateField) validateDurationField(r *validate.DurationRules) {
	in := make([]time.Duration, 0, len(r.GetIn()))
	for _, duration := range r.GetIn() {
		if duration == nil {
			// TODO: don't use assertf here
			m.assertf(false, "cannot have nil values in in")
			continue
		}
		in = append(in, *m.checkDur(duration))
	}
	notIn := make([]time.Duration, 0, len(r.GetNotIn()))
	for _, duration := range r.GetNotIn() {
		if duration == nil {
			// TODO: don't use asssertf here
			m.assertf(false, "cannot have nil values in in")
			continue
		}
		notIn = append(notIn, *m.checkDur(duration))
	}
	validateCommonNumericRule[time.Duration](
		m,
		durationRulesFieldNumber,
		durationFieldNumberSet,
		&numericCommonRule[time.Duration]{
			constant: m.checkDur(r.GetConst()),
			in:       in,
			notIn:    notIn,
			valueRange: *newNumericRange[time.Duration](
				m.checkDur(r.GetGt()),
				m.checkDur(r.GetGte()),
				m.checkDur(r.GetLt()),
				m.checkDur(r.GetLte()),
			),
		},
	)

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
	validateTimeRule[timestamppb.Timestamp, commonTime](
		m,
		m.field,
		timestampRulesFieldNumber,
		r.ProtoReflect(),
		func(value protoreflect.Value) *commonTime {
			bytes, _ := proto.Marshal(value.Message().Interface())
			t := &timestamppb.Timestamp{}
			proto.Unmarshal(bytes, t)
			m.assertf(t.IsValid(), "invalid timestamp")
			return &commonTime{
				seconds: t.Seconds,
				nanos:   t.Nanos,
			}
		},
		func(ct1, ct2 commonTime) int {
			if ct1.seconds > ct2.seconds {
				return 1
			}
			if ct1.seconds < ct2.seconds {
				return -1
			}
			return int(ct1.nanos - ct2.nanos)
		},
	)

	areNowRulesDefined := r.GetLtNow() || r.GetGtNow()
	areAbsoluteRulesDefined := r.GetLt() != nil || r.GetLte() != nil || r.GetGt() != nil || r.GetGte() != nil

	m.assertf(!areNowRulesDefined || !areAbsoluteRulesDefined, "now rules cannot be mixed with absolute lt/gt rules")
	m.assertf(r.Within == nil || !areAbsoluteRulesDefined, "within rule cannot be used with absolute lt/gt rules")

	// TODO: merge location if possible
	m.assertf(!r.GetLtNow() || !r.GetGtNow(), "gt_now and lt_now cannot be used together")

	dur := m.checkDur(r.Within)
	m.assertf(dur == nil || *dur > 0, "within rule must be positive")
}

func (m *validateField) checkMinMax(
	min *uint64,
	minFieldName string,
	max *uint64,
	maxFieldName string,
) {
	if min == nil || max == nil {
		return
	}

	m.assertf(*min <= *max,
		"%s value is greater than %s value", minFieldName, maxFieldName)
}

func (m *validateField) checkPattern(p *string, in int) {
	if p == nil {
		return
	}
	m.assertf(in == 0, "regex pattern and in rules are incompatible")
	_, err := regexp.Compile(*p)
	m.assertf(err == nil, "unable to parse regex pattern %s: %w", *p, err)
}

func (m *validateField) checkDur(d *durationpb.Duration) *time.Duration {
	if d == nil {
		return nil
	}

	dur, err := d.AsDuration(), d.CheckValid()
	m.assertf(err == nil, "could not resolve duration")
	return &dur
}
