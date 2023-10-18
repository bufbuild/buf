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
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L2517
	maxLenFieldNumberInStringRules = 3
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L2548
	minBytesFieldNumberInStringRules = 4
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L2579
	patternFieldNumberInStringRules = 6
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L2595
	prefixFieldNumberInStringRules = 7
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L2610
	suffixFieldNumberInStringRules = 8
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L2625
	containsFieldNumberInStringRules = 9
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L2844
	wellKnownRegexFieldNumberInStringRules = 24
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L2874
	strictFieldNumberInStringRules = 25
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L2961
	patternFieldNumberInBytesRules = 11
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L2976
	prefixFieldNumberInBytesRules = 5
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L2991
	suffixFieldNumberInBytesRules = 6
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L3006
	containsFieldNumberInBytesRules = 7
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L3164
	notInFieldNumberInEnumRules = 4
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L3183
	minItemsNumberInRepeatedFieldRules = 1
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L3199
	maxItemsNumberInRepeatedFieldRules = 2
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L3235
	itemsFieldNumberInRepeatedRules = 4
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L3249
	minPairsFieldNumberInMapRules = 1
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L3263
	maxPairsFieldNumberInMapRules = 2
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L3281
	keysFieldNumberInMapRules = 4
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L3298
	valuesFieldNumberInMapRules = 5
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L3696
	withInFieldNumberInTimestampRules = 9
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
)

func checkConstraintsForField(
	adder *adder,
	fieldConstraints *validate.FieldConstraints,
	field protosource.Field,
	fullNameToEnum map[string]protosource.Enum,
	fullNameToMessage map[string]protosource.Message,
) error {
	if fieldConstraints == nil {
		return nil
	}
	fieldConstraintsMessage := fieldConstraints.ProtoReflect()
	typeRulesFieldDescriptor := fieldConstraintsMessage.WhichOneof(typeOneofDescriptor)
	if typeRulesFieldDescriptor == nil {
		return nil
	}
	typeRulesFieldNumber := int32(typeRulesFieldDescriptor.Number())
	// checkMapRules and checkRepeatedRules are special cases that call checkConstraintsForField.
	if typeRulesFieldNumber == mapRulesFieldNumber {
		return checkMapRules(adder, fieldConstraints.GetMap(), field, fullNameToEnum, fullNameToMessage)
	}
	if typeRulesFieldNumber == repeatedRulesFieldNumber {
		return checkRepeatedRules(adder, fieldConstraints.GetRepeated(), field, fullNameToEnum, fullNameToMessage)
	}
	checkRulesTypeMatchFieldType(adder, field, typeRulesFieldNumber, string(typeRulesFieldDescriptor.Message().Name()))
	if numberRulesCheckFunc, ok := fieldNumberToCheckNumberRulesFunc[typeRulesFieldNumber]; ok {
		numberRulesMessage := fieldConstraintsMessage.Get(typeRulesFieldDescriptor).Message()
		return numberRulesCheckFunc(adder, typeRulesFieldNumber, numberRulesMessage)
	}
	switch typeRulesFieldNumber {
	case boolRulesFieldNumber:
		// Bool rules only have `const` and does not need checking.
	case stringRulesFieldNumber:
		return checkStringRules(adder, fieldConstraints.GetString_())
	case bytesRulesFieldNumber:
		return checkBytesRules(adder, fieldConstraints.GetBytes())
	case enumRulesFieldNumber:
		return checkEnumRules(adder, fieldConstraints.GetEnum(), field, fullNameToEnum)
	case anyRulesFieldNumber:
		checkAnyRules(adder, fieldConstraints.GetAny())
	case durationRulesFieldNumber:
		return checkDurationRules(adder, fieldConstraints.GetDuration())
	case timestampRulesFieldNumber:
		return checkTimestampRules(adder, fieldConstraints.GetTimestamp())
	}
	return nil
}

func checkRulesTypeMatchFieldType(adder *adder, field protosource.Field, ruleFieldNumber int32, ruleName string) {
	if field.Type() == descriptorpb.FieldDescriptorProto_TYPE_MESSAGE {
		expectedFieldMessageName, ok := fieldNumberToAllowedMessageName[ruleFieldNumber]
		if ok && string(expectedFieldMessageName) == field.TypeName() {
			return
		}
		adder.addForPathf(
			[]int32{ruleFieldNumber},
			"%s should not be defined on a field of type %s",
			ruleName,
			field.TypeName(),
		)
		return
	}
	expectedType, ok := fieldNumberToAllowedProtoType[ruleFieldNumber]
	if !ok || expectedType != field.Type() {
		adder.addForPathf(
			[]int32{ruleFieldNumber},
			"%s should not be defined on a field of type %v",
			ruleName,
			field.Type(),
		)
	}
}

func checkRepeatedRules(
	baseAdder *adder,
	r *validate.RepeatedRules,
	field protosource.Field,
	fullNameToEnum map[string]protosource.Enum,
	fullNameToMessage map[string]protosource.Message,
) error {
	if field.Label() != descriptorpb.FieldDescriptorProto_LABEL_REPEATED || field.IsMap() {
		baseAdder.addForPathf(
			[]int32{repeatedRulesFieldNumber},
			"field is not repeated but has repeated rules",
		)
	}
	if r.GetUnique() && field.Type() == descriptorpb.FieldDescriptorProto_TYPE_MESSAGE {
		baseAdder.addForPathf(
			[]int32{repeatedRulesFieldNumber},
			"unique rule is only allowed for scalar types",
		)
	}
	if r.MinItems != nil && r.MaxItems != nil && *r.MinItems > *r.MaxItems {
		baseAdder.addForPathsf(
			[][]int32{
				{repeatedRulesFieldNumber, maxItemsNumberInRepeatedFieldRules},
				{repeatedRulesFieldNumber, minItemsNumberInRepeatedFieldRules},
			},
			"min_items is greater than max_items",
		)
	}
	itemAdder := &adder{
		field:    baseAdder.field,
		basePath: []int32{repeatedRulesFieldNumber, itemsFieldNumberInRepeatedRules},
		addFunc:  baseAdder.addFunc,
	}
	return checkConstraintsForField(itemAdder, r.Items, field, fullNameToEnum, fullNameToMessage)
}

func checkMapRules(
	baseAdder *adder,
	r *validate.MapRules,
	field protosource.Field,
	fullNameToEnum map[string]protosource.Enum,
	fullNameToMessage map[string]protosource.Message,
) error {
	if !field.IsMap() {
		baseAdder.addForPathf(
			[]int32{mapRulesFieldNumber},
			"field is not a map but has map rules",
		)
	}
	if r.MinPairs != nil && r.MaxPairs != nil && *r.MinPairs > *r.MaxPairs {
		baseAdder.addForPathsf(
			[][]int32{
				{mapRulesFieldNumber, minPairsFieldNumberInMapRules},
				{mapRulesFieldNumber, maxPairsFieldNumberInMapRules},
			},
			"min_pairs is greater than max_pairs",
		)
	}
	mapMessage, ok := fullNameToMessage[field.TypeName()]
	if !ok {
		return fmt.Errorf("unable to find message for %s", field.TypeName())
	}
	if len(mapMessage.Fields()) != 2 {
		return fmt.Errorf("synthetic map message %s does not have enough fields", mapMessage.FullName())
	}
	keyAdder := &adder{
		field:    baseAdder.field,
		basePath: []int32{mapRulesFieldNumber, keysFieldNumberInMapRules},
		addFunc:  baseAdder.addFunc,
	}
	err := checkConstraintsForField(keyAdder, r.Keys, mapMessage.Fields()[0], fullNameToEnum, fullNameToMessage)
	if err != nil {
		return err
	}
	valueAdder := &adder{
		field:    baseAdder.field,
		basePath: []int32{mapRulesFieldNumber, valuesFieldNumberInMapRules},
		addFunc:  baseAdder.addFunc,
	}
	return checkConstraintsForField(valueAdder, r.Values, mapMessage.Fields()[1], fullNameToEnum, fullNameToMessage)
}

func checkStringRules(adder *adder, stringRules *validate.StringRules) error {
	checkConstAndIn(adder, stringRules, stringRulesFieldNumber)
	if err := checkLenRules(adder, stringRules, stringRulesFieldNumber, "len", "min_len", "max_len"); err != nil {
		return err
	}
	if err := checkLenRules(adder, stringRules, stringRulesFieldNumber, "len_bytes", "min_bytes", "max_bytes"); err != nil {
		return err
	}
	if stringRules.MaxLen != nil && stringRules.MaxBytes != nil && *stringRules.MaxBytes < *stringRules.MaxLen {
		// Saying a string has at most 5 bytes and at most 6 runes is the same as saying at most 5 bytes.
		adder.addForPathf(
			[]int32{stringRulesFieldNumber, maxLenFieldNumberInStringRules},
			"max_bytes is less than max_len, making max_len redundant",
		)
	}
	if stringRules.MinLen != nil && stringRules.MinBytes != nil && *stringRules.MinBytes < *stringRules.MinLen {
		// Saying a string has at least 5 bytes and at least 6 runes is the same as saying at least 6 runes.
		adder.addForPathf(
			[]int32{stringRulesFieldNumber, minBytesFieldNumberInStringRules},
			"min_bytes is less than min_len, making min_bytes redundant",
		)
	}
	substringFields := []struct {
		value       *string
		name        string
		fieldNumber int32
	}{
		{value: stringRules.Prefix, name: "prefix", fieldNumber: prefixFieldNumberInStringRules},
		{value: stringRules.Suffix, name: "suffix", fieldNumber: suffixFieldNumberInStringRules},
		{value: stringRules.Contains, name: "contains", fieldNumber: containsFieldNumberInStringRules},
	}
	for _, substringField := range substringFields {
		if substringField.value == nil {
			continue
		}
		substring := *substringField.value
		if runeCount := uint64(utf8.RuneCountInString(substring)); stringRules.MaxLen != nil && runeCount > *stringRules.MaxLen {
			adder.addForPathf(
				[]int32{stringRulesFieldNumber, substringField.fieldNumber},
				"%s has length %d, exceeding max_len",
				substringField.name,
				runeCount,
			)
		}
		if lenBytes := uint64(len(substring)); stringRules.MaxBytes != nil && lenBytes > *stringRules.MaxBytes {
			adder.addForPathf(
				[]int32{stringRulesFieldNumber, substringField.fieldNumber},
				"%s has %d bytes, exceeding max_bytes",
				substringField.name,
				lenBytes,
			)
		}
	}
	wellKnownRegex := stringRules.GetWellKnownRegex()
	nonStrict := stringRules.Strict != nil && !*stringRules.Strict
	switch wellKnownRegex {
	case validate.KnownRegex_KNOWN_REGEX_UNSPECIFIED:
		if nonStrict {
			adder.addForPathf(
				[]int32{stringRulesFieldNumber, strictFieldNumberInStringRules},
				"strict should not be set without well_known_regex",
			)
		}
	case validate.KnownRegex_KNOWN_REGEX_HTTP_HEADER_NAME, validate.KnownRegex_KNOWN_REGEX_HTTP_HEADER_VALUE:
		// TODO: do we care about this check? If the user wants an additional pattern
		// to match, perhaps it shouldn't be treated as a mistake.
		if stringRules.Pattern != nil {
			adder.addForPathsf(
				[][]int32{
					{stringRulesFieldNumber, wellKnownRegexFieldNumberInStringRules},
					{stringRulesFieldNumber, patternFieldNumberInStringRules},
				},
				"regex well_known_regex and regex pattern are incompatible",
			)
		}
	}
	if stringRules.Pattern == nil {
		return nil
	}
	if _, err := regexp.Compile(*stringRules.Pattern); err != nil {
		adder.addForPathf(
			[]int32{stringRulesFieldNumber, patternFieldNumberInStringRules},
			"unable to parse regex pattern %s: %w", *stringRules.Pattern, err,
		)
	}
	return nil
}

func checkBytesRules(adder *adder, bytesRules *validate.BytesRules) error {
	checkConstAndIn(adder, bytesRules, bytesRulesFieldNumber)
	if err := checkLenRules(adder, bytesRules, bytesRulesFieldNumber, "len", "min_len", "max_len"); err != nil {
		return err
	}
	substringFields := []struct {
		value       []byte
		name        string
		fieldNumber int32
	}{
		{value: bytesRules.Prefix, name: "prefix", fieldNumber: prefixFieldNumberInBytesRules},
		{value: bytesRules.Suffix, name: "suffix", fieldNumber: suffixFieldNumberInBytesRules},
		{value: bytesRules.Contains, name: "contains", fieldNumber: containsFieldNumberInBytesRules},
	}
	for _, substringField := range substringFields {
		if bytesRules.MaxLen != nil && uint64(len(substringField.value)) > *bytesRules.MaxLen {
			adder.addForPathf(
				[]int32{bytesRulesFieldNumber, substringField.fieldNumber},
				"%s has length %d, exceeding max_len",
				substringField.name,
				len(substringField.value),
			)
		}
	}
	if bytesRules.Pattern == nil {
		return nil
	}
	if _, err := regexp.Compile(*bytesRules.Pattern); err != nil {
		adder.addForPathf(
			[]int32{bytesRulesFieldNumber, patternFieldNumberInBytesRules},
			"unable to parse regex pattern %s: %w", *bytesRules.Pattern, err,
		)
	}
	return nil
}

func checkEnumRules(
	adder *adder,
	r *validate.EnumRules,
	field protosource.Field,
	fullNameToEnum map[string]protosource.Enum,
) error {
	checkConstAndIn(adder, r, enumRulesFieldNumber)
	if !r.GetDefinedOnly() {
		return nil
	}
	if len(r.In) == 0 && len(r.NotIn) == 0 {
		return nil
	}
	enum := fullNameToEnum[field.TypeName()]
	if enum == nil {
		return fmt.Errorf("unable to resolve enum %s", field.TypeName())
	}
	defined := enum.Values()
	vals := make(map[int]struct{}, len(defined))
	for _, val := range defined {
		vals[val.Number()] = struct{}{}
	}
	for _, notIn := range r.NotIn {
		if _, ok := vals[int(notIn)]; !ok {
			adder.addForPathf(
				[]int32{enumRulesFieldNumber, notInFieldNumberInEnumRules},
				"value (%d) is rejected by defined_only and does not need to be in not_in",
				notIn,
			)
		}
	}
	return nil
}

func checkAnyRules(adder *adder, anyRules *validate.AnyRules) {
	checkConstAndIn(adder, anyRules, anyRulesFieldNumber)
}

func checkDurationRules(adder *adder, r *validate.DurationRules) error {
	return checkNumericRules[durationpb.Duration](
		adder,
		durationRulesFieldNumber,
		r.ProtoReflect(),
		getDurationFromValue,
		compareDuration,
	)
}

func checkTimestampRules(adder *adder, r *validate.TimestampRules) error {
	if err := checkNumericRules[timestamppb.Timestamp](
		adder,
		timestampRulesFieldNumber,
		r.ProtoReflect(),
		getTimestampFromValue,
		compareTimestamp,
	); err != nil {
		return err
	}
	if r.GetLtNow() && r.GetGtNow() {
		adder.addForPathf(
			[]int32{timestampRulesFieldNumber},
			"gt_now and lt_now cannot be used together",
		)
	}
	if r.Within != nil {
		if !r.Within.IsValid() {
			adder.addForPathf(
				[]int32{timestampRulesFieldNumber, withInFieldNumberInTimestampRules},
				"within duration is invalid",
			)
		}
		if r.Within.Seconds <= 0 && r.Within.Nanos <= 0 {
			adder.addForPathf(
				[]int32{timestampRulesFieldNumber},
				"within duration must be positive",
			)
		}
	}
	// TODO: not sure if we really need to check this:
	areNowRulesDefined := r.GetLtNow() || r.GetGtNow() || r.Within != nil
	areAbsoluteRulesDefined := r.GetLt() != nil || r.GetLte() != nil || r.GetGt() != nil || r.GetGte() != nil
	if areNowRulesDefined && areAbsoluteRulesDefined {
		adder.addForPathf(
			[]int32{timestampRulesFieldNumber},
			"rules relative to now cannot be mixed with absolute gt/gte/lt/lte rules",
		)
	}
	return nil
}

func checkConstAndIn(adder *adder, rule proto.Message, ruleNumber int32) {
	var (
		fieldCount       int
		constFieldNumber int32
		inFieldNumber    int32
		isConstSpecified bool
		isInSpecified    bool
	)
	ruleMessage := rule.ProtoReflect()
	ruleMessage.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		fieldCount++
		switch string(fd.Name()) {
		case "const":
			isConstSpecified = true
			constFieldNumber = int32(fd.Number())
		case "in":
			isInSpecified = true
			inFieldNumber = int32(fd.Number())
		}
		return true
	})
	if isConstSpecified && fieldCount > 1 {
		adder.addForPathf(
			[]int32{ruleNumber, constFieldNumber},
			"const should be the only rule when specified",
		)
	}
	if isInSpecified && fieldCount > 1 {
		adder.addForPathf(
			[]int32{ruleNumber, inFieldNumber},
			"in should be the only rule when specified",
		)
	}
}

func checkLenRules(
	adder *adder,
	rules proto.Message,
	ruleFieldNumber int32,
	lenFieldName string,
	minLenFieldName string,
	maxLenFieldName string,
) error {
	var (
		length            *uint64
		lengthFieldNumber int32
		minLen            *uint64
		minLenFieldNumber int32
		maxLen            *uint64
		maxLenFieldNumber int32
		err               error
	)
	rules.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		switch string(fd.Name()) {
		case lenFieldName:
			uint64Value, ok := v.Interface().(uint64)
			if !ok {
				err = fmt.Errorf("unable to cast %v to uint64", v.Interface())
				return false
			}
			length = &uint64Value
			lengthFieldNumber = int32(fd.Number())
		case minLenFieldName:
			uint64Value, ok := v.Interface().(uint64)
			if !ok {
				err = fmt.Errorf("unable to cast %v to uint64", v.Interface())
				return false
			}
			minLen = &uint64Value
			minLenFieldNumber = int32(fd.Number())
		case maxLenFieldName:
			uint64Value, ok := v.Interface().(uint64)
			if !ok {
				err = fmt.Errorf("unable to cast %v to uint64", v.Interface())
				return false
			}
			maxLen = &uint64Value
			maxLenFieldNumber = int32(fd.Number())
		}
		return true
	})
	if err != nil {
		return err
	}
	if length != nil {
		if minLen != nil {
			adder.addForPathf(
				[]int32{ruleFieldNumber, lengthFieldNumber},
				"cannot have both %s and %s on the same field", lenFieldName, minLenFieldName,
			)
		}
		if maxLen != nil {
			adder.addForPathf(
				[]int32{ruleFieldNumber, lengthFieldNumber},
				"cannot have both %s and %s on the same field", lenFieldName, maxLenFieldName,
			)
		}
	}
	if maxLen == nil || minLen == nil {
		return nil
	}
	if *minLen > *maxLen {
		adder.addForPathsf(
			[][]int32{
				{ruleFieldNumber, minLenFieldNumber},
				{ruleFieldNumber, maxLenFieldNumber},
			},
			"%s should be greater than %s", minLenFieldName, maxLenFieldName,
		)
	} else if *minLen == *maxLen {
		adder.addForPathsf(
			[][]int32{
				{ruleFieldNumber, minLenFieldNumber},
				{ruleFieldNumber, maxLenFieldNumber},
			},
			"%s is equal to %s, consider using %s", minLenFieldName, maxLenFieldName, lenFieldName,
		)
	}
	return nil
}
