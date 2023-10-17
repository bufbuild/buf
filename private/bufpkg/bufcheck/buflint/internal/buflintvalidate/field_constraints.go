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
	"unicode/utf8"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"github.com/bufbuild/buf/private/pkg/protosource"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// TODO: consistent lint message tone/language
// TODO: in cel linting, check no google.protobuf.Any is used (if this check makes sense).
// TODO: report at one location or both location
// TODO: rename file names

const (
	// https://buf.build/bufbuild/protovalidate/docs/main:buf.validate#buf.validate.FieldConstraints
	// These numbers are passed for two purposes:
	// 1. Identity which type oneof is specified in a FieldConstraints. i.e. Is DoubleRules defined or
	// StringRules defined?
	// 2. Use it to construct a path to pass it to OptionExtensionLocation to get a more precise location.
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
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L169
	typeOneofDescriptor = validate.File_buf_validate_validate_proto.Messages().ByName("FieldConstraints").Oneofs().ByName("type")
	// https://buf.build/bufbuild/protovalidate/file/v0.4.3:buf/validate/validate.proto#L2846
	wellKnownHttpHeaderNamePattern = "^:?[0-9a-zA-Z!#$%&'*+-.^_|~\x60]+$"
	// https://buf.build/bufbuild/protovalidate/file/v0.4.3:buf/validate/validate.proto#L2853
	wellKnownHttpHeaderValuePattern = "^[^\u0000-\u0008\u000A-\u001F\u007F]*$"
	// https://buf.build/bufbuild/protovalidate/file/v0.4.3:buf/validate/validate.proto#L2854
	wellKnownHeaderStringPattern = "^[^\u0000\u000A\u000D]*$" // For non-strict validation.
)

func checkConstraintsForField(
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
		validateMapField(adder, fieldConstraints.GetMap(), field, fullNameToEnum, fullNameToMessage)
		return
	}
	if typeRulesFieldNumber == repeatedRulesFieldNumber {
		validateRepeatedField(adder, fieldConstraints.GetRepeated(), field, fullNameToEnum, fullNameToMessage)
		return
	}
	checkRulesTypeMatchFieldType(adder, field, typeRulesFieldNumber, string(typeRulesField.Message().Name()))
	if numberRulesValidateFunc, ok := numberRulesFieldNumberToValidateFunc[typeRulesFieldNumber]; ok {
		numberRulesMessage := fieldConstraintsMessage.Get(typeRulesField).Message()
		numberRulesValidateFunc(adder, typeRulesFieldNumber, numberRulesMessage)
		return
	}
	switch typeRules := fieldConstraints.Type.(type) {
	case *validate.FieldConstraints_Bool:
		// Bool rules only have `const` and does not need validation.
	case *validate.FieldConstraints_String_:
		validateStringField(adder, typeRules.String_)
	case *validate.FieldConstraints_Bytes:
		validateBytesField(adder, typeRules.Bytes)
	case *validate.FieldConstraints_Enum:
		validateEnumField(adder, typeRules.Enum, field, fullNameToEnum)
	case *validate.FieldConstraints_Any:
		validateAnyField(adder, typeRules.Any)
	case *validate.FieldConstraints_Duration:
		validateDurationField(adder, typeRules.Duration)
	case *validate.FieldConstraints_Timestamp:
		validateTimestampField(adder, typeRules.Timestamp)
	}
}

func checkRulesTypeMatchFieldType(adder *adder, field protosource.Field, ruleFieldNumber int32, ruleName string) {
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

func checkInAndNotIn(
	adder *adder,
	in int,
	notIn int,
) {
	if in != 0 && notIn != 0 {
		adder.add("cannot have both in and not_in rules on the same field")
	}
}

func checkLenRules(
	adder *adder,
	len *uint64,
	lenFieldName string,
	minLen *uint64,
	minLenFieldName string,
	maxLen *uint64,
	maxLenFieldName string,
) {
	if len != nil {
		if minLen != nil {
			adder.add("cannot have both %s and %s on the same field", lenFieldName, minLenFieldName)
		}
		if maxLen != nil {
			adder.add("cannot have both %s and %s on the same field", lenFieldName, maxLenFieldName)
		}
	}
	if maxLen != nil && minLen != nil {
		if *minLen > *maxLen {
			adder.add("%s should be greater than %s", minLenFieldName, maxLenFieldName)
		}
		if *minLen == *maxLen {
			adder.add("%s is equal to %s, consider using %s", minLenFieldName, maxLenFieldName, lenFieldName)
		}
	}
}

func validateStringField(adder *adder, r *validate.StringRules) {
	checkInAndNotIn(adder, len(r.In), len(r.NotIn))
	checkLenRules(adder, r.Len, "len", r.MinLen, "min_len", r.MaxLen, "max_len")
	checkLenRules(adder, r.LenBytes, "len_bytes", r.MinBytes, "min_bytes", r.MaxBytes, "max_bytes")
	if r.MaxLen != nil && r.MaxBytes != nil && *r.MaxBytes < *r.MaxLen {
		adder.add("max_bytes is less than max_len, making max_len redundant")
	}
	if r.MinLen != nil && r.MinBytes != nil && *r.MinBytes < *r.MinLen {
		adder.add("min_bytes is less than min_len, making min_bytes redundant")
	}
	substringFields := []struct {
		value *string
		name  string
	}{
		{value: r.Prefix, name: "prefix"},
		{value: r.Suffix, name: "suffix"},
		{value: r.Contains, name: "containts"},
	}
	for _, substringField := range substringFields {
		if substringField.value == nil {
			continue
		}
		substring := *substringField.value
		if r.MaxLen != nil && uint64(utf8.RuneCountInString(substring)) > *r.MaxLen {
			adder.addForPath(
				[]int32{stringRulesFieldNumber},
				"%s has length %d, exceeding max_len",
				substringField.name,
				utf8.RuneCountInString(substring),
			)
		}
		if r.MaxBytes != nil && uint64(len(substring)) > *r.MaxBytes {
			adder.addForPath(
				[]int32{stringRulesFieldNumber},
				"%s has %d bytes, exceeding max_bytes",
				substringField.name,
				len(substring),
			)
		}
	}
	patternInEffect := r.Pattern
	wellKnownRegex := r.GetWellKnownRegex()
	nonStrict := r.Strict != nil && !*r.Strict
	switch wellKnownRegex {
	case validate.KnownRegex_KNOWN_REGEX_UNSPECIFIED:
		if nonStrict {
			adder.add("strict should not be set without well_known_regex")
		}
	case validate.KnownRegex_KNOWN_REGEX_HTTP_HEADER_NAME:
		if r.Pattern != nil {
			adder.add("regex well_known_regex and regex pattern are incompatible")
		}
		patternInEffect = &wellKnownHttpHeaderNamePattern
		if nonStrict {
			patternInEffect = &wellKnownHeaderStringPattern
		}
	case validate.KnownRegex_KNOWN_REGEX_HTTP_HEADER_VALUE:
		if r.Pattern != nil {
			adder.add("regex well_known_regex and regex pattern are incompatible")
		}
		patternInEffect = &wellKnownHttpHeaderValuePattern
		if nonStrict {
			patternInEffect = &wellKnownHeaderStringPattern
		}
	}
	checkPattern(adder, patternInEffect, len(r.In))
}

func validateBytesField(adder *adder, r *validate.BytesRules) {
	checkInAndNotIn(adder, len(r.In), len(r.NotIn))
	checkLenRules(adder, r.Len, "len", r.MinLen, "min_len", r.MaxLen, "max_len")
	substringFields := []struct {
		value []byte
		name  string
	}{
		{value: r.Prefix, name: "prefix"},
		{value: r.Suffix, name: "suffix"},
		{value: r.Contains, name: "containts"},
	}
	for _, substringField := range substringFields {
		if r.MaxLen != nil && uint64(len(substringField.value)) > *r.MaxLen {
			adder.addForPath(
				[]int32{bytesRulesFieldNumber},
				"%s has length %d, exceeding max_len",
				substringField.name,
				len(substringField.value),
			)
		}
	}
	checkPattern(adder, r.Pattern, len(r.In))
}

func validateEnumField(
	adder *adder,
	r *validate.EnumRules,
	field protosource.Field,
	fullNameToEnum map[string]protosource.Enum,
) {
	checkInAndNotIn(adder, len(r.In), len(r.NotIn))
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

func validateRepeatedField(
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
	checkConstraintsForField(adder, r.Items, field, fullNameToEnum, fullNameToMessage)
}

func validateMapField(
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
	checkConstraintsForField(adder, r.Keys, mapMessage.Fields()[0], fullNameToEnum, fullNameToMessage)
	checkConstraintsForField(adder, r.Values, mapMessage.Fields()[1], fullNameToEnum, fullNameToMessage)
}

func validateAnyField(adder *adder, r *validate.AnyRules) {
	checkInAndNotIn(adder, len(r.In), len(r.NotIn))
}

func validateDurationField(adder *adder, r *validate.DurationRules) {
	validateNumericRule[durationpb.Duration](
		adder,
		durationRulesFieldNumber,
		r.ProtoReflect(),
		getDurationFromValue,
		compareDuration,
	)
}

func validateTimestampField(adder *adder, r *validate.TimestampRules) {
	validateNumericRule[timestamppb.Timestamp](
		adder,
		timestampRulesFieldNumber,
		r.ProtoReflect(),
		getTimestampFromValue,
		compareTimestamp,
	)
	if r.GetLtNow() && r.GetGtNow() {
		adder.addForPath(
			[]int32{timestampRulesFieldNumber},
			"gt_now and lt_now cannot be used together",
		)
	}
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
	// TODO: not sure if we really need to validate the following:
	areNowRulesDefined := r.GetLtNow() || r.GetGtNow()
	areAbsoluteRulesDefined := r.GetLt() != nil || r.GetLte() != nil || r.GetGt() != nil || r.GetGte() != nil
	if areNowRulesDefined && areAbsoluteRulesDefined {
		adder.add("now rules cannot be mixed with absolute lt/gt rules")
	}
	if r.Within != nil && areAbsoluteRulesDefined {
		adder.add("within rule cannot be used with absolute lt/gt rules")
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
// TODO: remove in.
func checkPattern(adder *adder, pattern *string, in int) {
	if pattern == nil {
		return
	}
	if in != 0 {
		adder.add("regex pattern and in rules are incompatible")
	}
	_, err := regexp.Compile(*pattern)
	if err != nil {
		adder.add("unable to parse regex pattern %s: %w", *pattern, err)
	}
}
