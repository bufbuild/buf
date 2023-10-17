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
	"fmt"
	"regexp"
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

func (m *validateField) checkConstraintsForField(
	adder *adder,
	fieldConstraints *validate.FieldConstraints,
	field protosource.Field,
	fullNameToEnum map[string]protosource.Enum,
	fullNameToMessage map[string]protosource.Message,
) {
	if fieldConstraints == nil {
		return
	}
	fieldConstraintsMessage := fieldConstraints.ProtoReflect()
	typeRulesField := fieldConstraintsMessage.WhichOneof(typeOneofDescriptor)
	if typeRulesField == nil {
		return
	}
	typeRulesFieldNumber := int32(typeRulesField.Number())
	if typeRulesFieldNumber == mapRulesFieldNumber {
		m.validateMapField(adder, fieldConstraints.GetMap(), field, fullNameToEnum, fullNameToMessage)
		return
	}
	if typeRulesFieldNumber == repeatedRulesFieldNumber {
		m.validateRepeatedField(adder, fieldConstraints.GetRepeated(), field, fullNameToEnum, fullNameToMessage)
		return
	}
	checkTypeMatch(adder, field, typeRulesFieldNumber, string(typeRulesField.Message().Name()))
	if floatRulesFieldNumber <= typeRulesFieldNumber && typeRulesFieldNumber <= sFixed64RulesFieldNumber {
		numberRulesMessage := fieldConstraintsMessage.Get(typeRulesField).Message()
		validateNumberRulesMessage(adder, typeRulesFieldNumber, numberRulesMessage)
		return
	}
	switch r := fieldConstraints.Type.(type) {
	case *validate.FieldConstraints_Bool:
		// Bool rules only have `const` and does not need validation.
	case *validate.FieldConstraints_String_:
		m.validateStringField(adder, r.String_)
	case *validate.FieldConstraints_Bytes:
		m.validateBytesField(adder, r.Bytes)
	case *validate.FieldConstraints_Enum:
		validateEnumField(adder, r.Enum, field, fullNameToEnum)
	case *validate.FieldConstraints_Any:
		validateAnyField(adder, r.Any)
	case *validate.FieldConstraints_Duration:
		validateDurationField(adder, r.Duration)
	case *validate.FieldConstraints_Timestamp:
		validateTimestampField(adder, r.Timestamp)
	}
}

func checkTypeMatch(adder *adder, field protosource.Field, ruleFieldNumber int32, ruleName string) {
	if field.Type() == descriptorpb.FieldDescriptorProto_TYPE_MESSAGE {
		expectedFieldMessageName, ok := fieldNumberToAllowedMessageName[ruleFieldNumber]
		if ok && string(expectedFieldMessageName) == field.TypeName() {
			return
		}
		adder.addForPath(
			[]int32{ruleFieldNumber},
			"%s should not be defined on a field of type %s",
			ruleName,
			field.TypeName(),
		)
		return
	}
	expectedType, ok := fieldNumberToAllowedProtoType[ruleFieldNumber]
	if !ok {
		// TODO
		return
	}
	if expectedType != field.Type() {
		adder.addForPath(
			[]int32{ruleFieldNumber},
			"%s should not be defined on a field of type %v",
			ruleName,
			field.Type(),
		)
	}
}

func checkIns(
	adder *adder,
	in int,
	notIn int,
) {
	if in != 0 && notIn != 0 {
		adder.add("cannot have both in and not_in rules on the same field")
	}
}

func (m *validateField) assertf(expr bool, format string, v ...interface{}) {
	if !expr {
		m.add(m.field, m.location, nil, format, v...)
	}
}

func (m *validateField) validateStringField(adder *adder, r *validate.StringRules) {
	if r.Len != nil {
		m.assertf(r.MinLen == nil, "cannot have both len and min_len on the same field")
		m.assertf(r.MaxLen == nil, "cannot have both len and max_len on the same field")
	}
	if r.LenBytes != nil {
		m.assertf(r.MinBytes == nil, "cannot have both len_bytes and min_bytes on the same field")
		m.assertf(r.MaxBytes == nil, "cannot have both len_bytes and max_bytes on the same field")
	}
	checkMinMax(adder, r.MinLen, "min_len", r.MaxLen, "max_len")
	checkMinMax(adder, r.MinBytes, "min_bytes", r.MaxBytes, "max_bytes")
	checkIns(adder, len(r.In), len(r.NotIn))
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
	checkPattern(adder, patternInEffect, len(r.In))
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

func validateEnumField(
	adder *adder,
	r *validate.EnumRules,
	field protosource.Field,
	fullNameToEnum map[string]protosource.Enum,
) {
	checkIns(adder, len(r.In), len(r.NotIn))
	if !r.GetDefinedOnly() {
		return
	}
	if len(r.In) == 0 && len(r.NotIn) == 0 {
		return
	}
	enum := fullNameToEnum[field.TypeName()]
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
			if !ok {
				adder.add("undefined in value (%d) conflicts with defined_only rule", in)
			}
		}
	}
	if len(r.NotIn) > 0 {
		for _, notIn := range r.NotIn {
			_, ok := vals[int(notIn)]
			if !ok {
				adder.add("undefined not_in value (%d) is redundant, as it is already rejected by defined_only")
			}
		}
	}
}

func (m *validateField) validateBytesField(adder *adder, r *validate.BytesRules) {
	if r.Len != nil {
		m.assertf(r.MinLen == nil, "cannot have both len and min_len on the same field")
		m.assertf(r.MaxLen == nil, "cannot have both len and max_len on the same field")
	}
	checkMinMax(adder, r.MinLen, "min_len", r.MaxLen, "max_len")
	checkIns(adder, len(r.In), len(r.NotIn))
	checkPattern(adder, r.Pattern, len(r.In))
	if r.MaxLen != nil {
		max := r.GetMaxLen()
		m.assertf(uint64(len(r.GetPrefix())) <= max, "prefix length exceeds max_len")
		m.assertf(uint64(len(r.GetSuffix())) <= max, "suffix length exceeds max_len")
		m.assertf(uint64(len(r.GetContains())) <= max, "contains length exceeds max_len")
	}
}

func (m *validateField) validateRepeatedField(
	adder *adder,
	r *validate.RepeatedRules,
	field protosource.Field,
	fullNameToEnum map[string]protosource.Enum,
	fullNameToMessage map[string]protosource.Message,
) {
	if field.Label() != descriptorpb.FieldDescriptorProto_LABEL_REPEATED || field.IsMap() {
		adder.add("field is not repeated but got repeated rules")
	}
	checkMinMax(adder, r.MinItems, "min_items", r.MaxItems, "max_items")
	if r.GetUnique() && field.Type() == descriptorpb.FieldDescriptorProto_TYPE_MESSAGE {
		adder.add("unique rule is only applicable for scalar types")
	}
	m.checkConstraintsForField(adder, r.Items, field, fullNameToEnum, fullNameToMessage)
}

func (m *validateField) validateMapField(
	adder *adder,
	r *validate.MapRules,
	field protosource.Field,
	fullNameToEnum map[string]protosource.Enum,
	fullNameToMessage map[string]protosource.Message,
) {
	if !field.IsMap() {
		adder.add("field is not a map but got map rules")
	}
	checkMinMax(adder, r.MinPairs, "min_pairs", r.MaxPairs, "max_pairs")
	// TODO: error if not found
	mapMessage := fullNameToMessage[field.TypeName()]
	// TODO: make sure it has two fields
	m.checkConstraintsForField(adder, r.Keys, mapMessage.Fields()[0], fullNameToEnum, fullNameToMessage)
	m.checkConstraintsForField(adder, r.Values, mapMessage.Fields()[1], fullNameToEnum, fullNameToMessage)
}

func validateAnyField(adder *adder, r *validate.AnyRules) {
	checkIns(adder, len(r.In), len(r.NotIn))
}

func validateDurationField(adder *adder, r *validate.DurationRules) {
	validateNumericRule[copiableTime](
		adder,
		durationRulesFieldNumber,
		r.ProtoReflect(),
		func(value protoreflect.Value) (*copiableTime, string) {
			// TODO: what if this errors?
			bytes, _ := proto.Marshal(value.Message().Interface())
			duration := &durationpb.Duration{}
			proto.Unmarshal(bytes, duration)
			if !duration.IsValid() {
				return nil, fmt.Sprintf("%v is an invalid duration", duration)
			}
			return &copiableTime{
				seconds: duration.Seconds,
				nanos:   duration.Nanos,
			}, ""
		},
		compareTime,
	)
}

func validateTimestampField(adder *adder, r *validate.TimestampRules) {
	validateNumericRule[copiableTime](
		adder,
		timestampRulesFieldNumber,
		r.ProtoReflect(),
		func(value protoreflect.Value) (*copiableTime, string) {
			// TODO: what if this errors?
			bytes, _ := proto.Marshal(value.Message().Interface())
			timestamp := &timestamppb.Timestamp{}
			proto.Unmarshal(bytes, timestamp)
			if !timestamp.IsValid() {
				return nil, fmt.Sprintf("%v is not a valid timestamp", timestamp)
			}
			return &copiableTime{
				seconds: timestamp.Seconds,
				nanos:   timestamp.Nanos,
			}, ""
		},
		compareTime,
	)
	if r.Within != nil {
		if !r.Within.IsValid() {
			adder.addForPath(
				// TODO: append within field number
				[]int32{timestampRulesFieldNumber},
				"within duration is invalid",
			)
		}
		if r.Within.Seconds <= 0 && r.Within.Nanos <= 0 {
			adder.addForPath(
				[]int32{timestampRulesFieldNumber},
				"within duration must be positive",
			)
		}
	}
	areNowRulesDefined := r.GetLtNow() || r.GetGtNow()
	areAbsoluteRulesDefined := r.GetLt() != nil || r.GetLte() != nil || r.GetGt() != nil || r.GetGte() != nil
	if areNowRulesDefined && areAbsoluteRulesDefined {
		adder.add("now rules cannot be mixed with absolute lt/gt rules")
	}
	if r.Within != nil && areAbsoluteRulesDefined {
		adder.add("within rule cannot be used with absolute lt/gt rules")
	}
	// TODO: merge location if possible
	if r.GetLtNow() && r.GetGtNow() {
		adder.add("gt_now and lt_now cannot be used together")
	}
}

func checkMinMax(
	adder *adder,
	min *uint64,
	minFieldName string,
	max *uint64,
	maxFieldName string,
) {
	if min != nil && max != nil && *min > *max {
		adder.add("%s value is greater than %s value", minFieldName, maxFieldName)
	}
}

// TODO: update func signature
func checkPattern(adder *adder, p *string, in int) {
	if p == nil {
		return
	}
	if in != 0 {
		adder.add("regex pattern and in rules are incompatible")
	}
	_, err := regexp.Compile(*p)
	if err != nil {
		adder.add("unable to parse regex pattern %s: %w", *p, err)
	}
}
