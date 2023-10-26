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
	"strings"
	"unicode/utf8"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"github.com/bufbuild/buf/private/pkg/protosource"
	"github.com/bufbuild/protovalidate-go/resolver"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	// https://buf.build/bufbuild/protovalidate/docs/main:buf.validate#buf.validate.FieldConstraints
	// These numbers are used for two purposes:
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
	skippedFieldNumber        = 24
	requiredFieldNumber       = 25
	ignoreEmptyFieldNumber    = 26
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L2501
	minLenFieldNumberInStringRules = 2
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L2517
	maxLenFieldNumberInStringRules = 3
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L2548
	minBytesFieldNumberInStringRules = 4
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L2563
	maxBytesFieldNumberInStringRules = 5
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L2579
	patternFieldNumberInStringRules = 6
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L2595
	prefixFieldNumberInStringRules = 7
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L2610
	suffixFieldNumberInStringRules = 8
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L2625
	containsFieldNumberInStringRules = 9
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L2640
	notContainsFieldNumberInStringRules = 23
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L2844
	wellKnownRegexFieldNumberInStringRules = 24
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L2874
	strictFieldNumberInStringRules = 25
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L2961
	patternFieldNumberInBytesRules = 4
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L2976
	prefixFieldNumberInBytesRules = 5
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L2991
	suffixFieldNumberInBytesRules = 6
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L3006
	containsFieldNumberInBytesRules = 7
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L3183
	minItemsFieldNumberInRepeatedFieldRules = 1
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L3199
	maxItemsFieldNumberInRepeatedFieldRules = 2
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L3214
	uniqueFieldNumberInRepeatedFieldRules = 3
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
	// https://buf.build/bufbuild/protovalidate/file/main:buf/validate/validate.proto#L3566
	ltNowFieldNumberInTimestampRules = 7
	// https://buf.build/bufbuild/protovalidate/file/main:buf/validate/validate.proto#L3682
	gtNowFieldNumberInTimestampRules = 8
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L3696
	withInFieldNumberInTimestampRules = 9
)

var (
	fieldNumberToAllowedScalarType = map[int32]protoreflect.Kind{
		floatRulesFieldNumber:    protoreflect.FloatKind,
		doubleRulesFieldNumber:   protoreflect.DoubleKind,
		int32RulesFieldNumber:    protoreflect.Int32Kind,
		int64RulesFieldNumber:    protoreflect.Int64Kind,
		uInt32RulesFieldNumber:   protoreflect.Uint32Kind,
		uInt64RulesFieldNumber:   protoreflect.Uint64Kind,
		sInt32RulesFieldNumber:   protoreflect.Sint32Kind,
		sInt64RulesFieldNumber:   protoreflect.Sint64Kind,
		fixed32RulesFieldNumber:  protoreflect.Fixed32Kind,
		fixed64RulesFieldNumber:  protoreflect.Fixed64Kind,
		sFixed32RulesFieldNumber: protoreflect.Sfixed32Kind,
		sFixed64RulesFieldNumber: protoreflect.Sfixed64Kind,
		boolRulesFieldNumber:     protoreflect.BoolKind,
		stringRulesFieldNumber:   protoreflect.StringKind,
		bytesRulesFieldNumber:    protoreflect.BytesKind,
		enumRulesFieldNumber:     protoreflect.EnumKind,
	}
	fieldNumberToAllowedMessageName = map[int32]string{
		floatRulesFieldNumber:     string((&wrapperspb.FloatValue{}).ProtoReflect().Descriptor().FullName()),
		doubleRulesFieldNumber:    string((&wrapperspb.DoubleValue{}).ProtoReflect().Descriptor().FullName()),
		int32RulesFieldNumber:     string((&wrapperspb.Int32Value{}).ProtoReflect().Descriptor().FullName()),
		int64RulesFieldNumber:     string((&wrapperspb.Int64Value{}).ProtoReflect().Descriptor().FullName()),
		uInt32RulesFieldNumber:    string((&wrapperspb.UInt32Value{}).ProtoReflect().Descriptor().FullName()),
		uInt64RulesFieldNumber:    string((&wrapperspb.UInt64Value{}).ProtoReflect().Descriptor().FullName()),
		boolRulesFieldNumber:      string((&wrapperspb.BoolValue{}).ProtoReflect().Descriptor().FullName()),
		stringRulesFieldNumber:    string((&wrapperspb.StringValue{}).ProtoReflect().Descriptor().FullName()),
		bytesRulesFieldNumber:     string((&wrapperspb.BytesValue{}).ProtoReflect().Descriptor().FullName()),
		anyRulesFieldNumber:       string((&anypb.Any{}).ProtoReflect().Descriptor().FullName()),
		durationRulesFieldNumber:  string((&durationpb.Duration{}).ProtoReflect().Descriptor().FullName()),
		timestampRulesFieldNumber: string((&timestamppb.Timestamp{}).ProtoReflect().Descriptor().FullName()),
	}
	wrapperTypeNames = map[string]struct{}{
		string((&wrapperspb.FloatValue{}).ProtoReflect().Descriptor().FullName()):  {},
		string((&wrapperspb.DoubleValue{}).ProtoReflect().Descriptor().FullName()): {},
		string((&wrapperspb.Int32Value{}).ProtoReflect().Descriptor().FullName()):  {},
		string((&wrapperspb.Int64Value{}).ProtoReflect().Descriptor().FullName()):  {},
		string((&wrapperspb.UInt32Value{}).ProtoReflect().Descriptor().FullName()): {},
		string((&wrapperspb.UInt64Value{}).ProtoReflect().Descriptor().FullName()): {},
		string((&wrapperspb.BoolValue{}).ProtoReflect().Descriptor().FullName()):   {},
		string((&wrapperspb.StringValue{}).ProtoReflect().Descriptor().FullName()): {},
		string((&wrapperspb.BytesValue{}).ProtoReflect().Descriptor().FullName()):  {},
	}
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L117
	fieldConstraintsDescriptor = validate.File_buf_validate_validate_proto.Messages().ByName("FieldConstraints")
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L169
	typeOneofDescriptor = validate.File_buf_validate_validate_proto.Messages().ByName("FieldConstraints").Oneofs().ByName("type")
)

// checkForField validates that protovalidate rules defined for this field are
// valid, not including CEL expressions.
func checkForField(
	add func(protosource.Descriptor, protosource.Location, []protosource.Location, string, ...interface{}),
	descriptorResolver protodesc.Resolver,
	field protosource.Field,
) error {
	fieldDescriptor, err := getReflectFieldDescriptor(descriptorResolver, field)
	if err != nil {
		return err
	}
	constraints := resolver.DefaultResolver{}.ResolveFieldConstraints(fieldDescriptor)
	return checkConstraintsForField(
		&adder{
			field:               field,
			fieldPrettyTypeName: getFieldTypePrettyNameName(fieldDescriptor),
			addFunc:             add,
		},
		constraints,
		fieldDescriptor,
	)
}

func checkConstraintsForField(
	adder *adder,
	fieldConstraints *validate.FieldConstraints,
	fieldDescriptor protoreflect.FieldDescriptor,
) error {
	if fieldConstraints == nil {
		return nil
	}
	if fieldDescriptor.IsExtension() {
		checkConstraintsForExtension(adder, fieldConstraints)
	}
	checkFieldFlags(adder, fieldConstraints)
	if err := checkCELForField(
		adder,
		fieldConstraints,
		fieldDescriptor,
	); err != nil {
		return err
	}
	fieldConstraintsMessage := fieldConstraints.ProtoReflect()
	typeRulesFieldDescriptor := fieldConstraintsMessage.WhichOneof(typeOneofDescriptor)
	if typeRulesFieldDescriptor == nil {
		return nil
	}
	typeRulesFieldNumber := int32(typeRulesFieldDescriptor.Number())
	// Map and repeated special cases that contain fieldConstraints.
	if typeRulesFieldNumber == mapRulesFieldNumber {
		return checkMapRules(adder, fieldConstraints.GetMap(), fieldDescriptor)
	}
	if typeRulesFieldNumber == repeatedRulesFieldNumber {
		return checkRepeatedRules(adder, fieldConstraints.GetRepeated(), fieldDescriptor)
	}
	typesMatch := checkRulesTypeMatchFieldType(adder, fieldDescriptor, typeRulesFieldNumber)
	if !typesMatch {
		return nil
	}
	if numberRulesCheckFunc, ok := fieldNumberToCheckNumberRulesFunc[typeRulesFieldNumber]; ok {
		numberRulesMessage := fieldConstraintsMessage.Get(typeRulesFieldDescriptor).Message()
		return numberRulesCheckFunc(adder, typeRulesFieldNumber, numberRulesMessage)
	}
	switch typeRulesFieldNumber {
	case boolRulesFieldNumber:
		// Bool rules only have `const` and does not need validating.
	case stringRulesFieldNumber:
		return checkStringRules(adder, fieldConstraints.GetString_())
	case bytesRulesFieldNumber:
		return checkBytesRules(adder, fieldConstraints.GetBytes())
	case enumRulesFieldNumber:
		checkEnumRules(adder, fieldConstraints.GetEnum())
	case anyRulesFieldNumber:
		checkAnyRules(adder, fieldConstraints.GetAny())
	case durationRulesFieldNumber:
		return checkDurationRules(adder, fieldConstraints.GetDuration())
	case timestampRulesFieldNumber:
		return checkTimestampRules(adder, fieldConstraints.GetTimestamp())
	}
	return nil
}

func checkFieldFlags(
	adder *adder,
	fieldConstraints *validate.FieldConstraints,
) {
	var fieldCount int
	fieldConstraints.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		fieldCount++
		return true
	})
	if fieldConstraints.GetSkipped() && fieldCount > 1 {
		adder.addForPathf(
			[]int32{skippedFieldNumber},
			"Field %q has %s and does need other rules in %s.",
			adder.fieldName(),
			adder.getFieldRuleName(skippedFieldNumber),
			adder.getFieldRuleName(),
		)
	}
	if fieldConstraints.GetRequired() && fieldConstraints.GetIgnoreEmpty() {
		adder.addForPathsf(
			[][]int32{
				{requiredFieldNumber},
				{ignoreEmptyFieldNumber},
			},
			"Field %q has both %s and %s.",
			adder.fieldName(),
			adder.getFieldRuleName(requiredFieldNumber),
			adder.getFieldRuleName(ignoreEmptyFieldNumber),
		)
	}
}

// Assumes the rule isn't a map rule or repeated rule, but the field could be a
// map or a repeated field.
func checkRulesTypeMatchFieldType(
	adder *adder,
	fieldDescriptor protoreflect.FieldDescriptor,
	ruleFieldNumber int32,
) bool {
	if expectedScalarType, ok := fieldNumberToAllowedScalarType[ruleFieldNumber]; ok &&
		expectedScalarType == fieldDescriptor.Kind() {
		return true
	}
	if expectedFieldMessageName, ok := fieldNumberToAllowedMessageName[ruleFieldNumber]; ok &&
		isFieldDescriptorMessage(fieldDescriptor) && string(fieldDescriptor.Message().FullName()) == expectedFieldMessageName {
		return true
	}
	adder.addForPathf(
		[]int32{ruleFieldNumber},
		"Field %q has type %s but has %s rules.",
		adder.fieldName(),
		adder.fieldPrettyTypeName,
		adder.getFieldRuleName(ruleFieldNumber),
	)
	return false
}

func checkConstraintsForExtension(
	adder *adder,
	fieldConstraints *validate.FieldConstraints,
) {
	if fieldConstraints.GetRequired() {
		adder.addForPathf(
			[]int32{requiredFieldNumber},
			"Field %q is an extension field and cannot have %s.",
			adder.fieldName(),
			adder.getFieldRuleName(requiredFieldNumber),
		)
	}
	if fieldConstraints.GetIgnoreEmpty() {
		adder.addForPathf(
			[]int32{ignoreEmptyFieldNumber},
			"Field %q is an extension field and cannot have %s.",
			adder.fieldName(),
			adder.getFieldRuleName(ignoreEmptyFieldNumber),
		)
	}
}

func checkRepeatedRules(
	baseAdder *adder,
	repeatedRules *validate.RepeatedRules,
	fieldDescriptor protoreflect.FieldDescriptor,
) error {
	if !fieldDescriptor.IsList() {
		baseAdder.addForPathf(
			[]int32{repeatedRulesFieldNumber},
			"Field %q is not a repeated field but has %s.",
			baseAdder.fieldName(),
			baseAdder.getFieldRuleName(repeatedRulesFieldNumber),
		)
		return nil
	}
	if repeatedRules.GetUnique() && isFieldDescriptorMessage(fieldDescriptor) {
		if _, isFieldWrapper := wrapperTypeNames[string(fieldDescriptor.Message().FullName())]; !isFieldWrapper {
			baseAdder.addForPathf(
				[]int32{repeatedRulesFieldNumber, uniqueFieldNumberInRepeatedFieldRules},
				"Field %q does not have a scalar type or a wrapper type but has %s set to true.",
				baseAdder.fieldName(),
				baseAdder.getFieldRuleName(repeatedRulesFieldNumber, uniqueFieldNumberInRepeatedFieldRules),
			)
		}
	}
	if repeatedRules.MinItems != nil && repeatedRules.MaxItems != nil && *repeatedRules.MinItems > *repeatedRules.MaxItems {
		baseAdder.addForPathsf(
			[][]int32{
				{repeatedRulesFieldNumber, maxItemsFieldNumberInRepeatedFieldRules},
				{repeatedRulesFieldNumber, minItemsFieldNumberInRepeatedFieldRules},
			},
			"Field %q has a %s (%d) higher than its %s (%d).",
			baseAdder.fieldName(),
			baseAdder.getFieldRuleName(repeatedRulesFieldNumber, minItemsFieldNumberInRepeatedFieldRules),
			*repeatedRules.MinItems,
			baseAdder.getFieldRuleName(repeatedRulesFieldNumber, maxItemsFieldNumberInRepeatedFieldRules),
			*repeatedRules.MaxItems,
		)
	}
	itemAdder := baseAdder.cloneWithNewBasePath(repeatedRulesFieldNumber, itemsFieldNumberInRepeatedRules)
	return checkConstraintsForField(itemAdder, repeatedRules.Items, fieldDescriptor)
}

func checkMapRules(
	baseAdder *adder,
	mapRules *validate.MapRules,
	fieldDescriptor protoreflect.FieldDescriptor,
) error {
	if !fieldDescriptor.IsMap() {
		baseAdder.addForPathf(
			[]int32{mapRulesFieldNumber},
			"Field %q is not a map but has %s.",
			baseAdder.fieldName(),
			baseAdder.getFieldRuleName(mapRulesFieldNumber),
		)
	}
	if mapRules.MinPairs != nil && mapRules.MaxPairs != nil && *mapRules.MinPairs > *mapRules.MaxPairs {
		baseAdder.addForPathsf(
			[][]int32{
				{mapRulesFieldNumber, minPairsFieldNumberInMapRules},
				{mapRulesFieldNumber, maxPairsFieldNumberInMapRules},
			},
			"Field %q has a %s higher than its %s.",
			baseAdder.fieldName(),
			baseAdder.getFieldRuleName(mapRulesFieldNumber, minPairsFieldNumberInMapRules),
			baseAdder.getFieldRuleName(mapRulesFieldNumber, maxPairsFieldNumberInMapRules),
		)
	}
	keyAdder := baseAdder.cloneWithNewBasePath(mapRulesFieldNumber, keysFieldNumberInMapRules)
	err := checkConstraintsForField(keyAdder, mapRules.Keys, fieldDescriptor.MapKey())
	if err != nil {
		return err
	}
	valueAdder := baseAdder.cloneWithNewBasePath(mapRulesFieldNumber, valuesFieldNumberInMapRules)
	return checkConstraintsForField(valueAdder, mapRules.Values, fieldDescriptor.MapValue())
}

func checkStringRules(adder *adder, stringRules *validate.StringRules) error {
	checkConstAndIn(adder, stringRules, stringRulesFieldNumber)
	if err := checkLenRules(adder, stringRules, stringRulesFieldNumber, "len", "min_len", "max_len"); err != nil {
		return err
	}
	if err := checkLenRules(adder, stringRules, stringRulesFieldNumber, "len_bytes", "min_bytes", "max_bytes"); err != nil {
		return err
	}
	if stringRules.MinLen != nil && stringRules.MaxBytes != nil && *stringRules.MaxBytes < *stringRules.MinLen {
		adder.addForPathsf(
			[][]int32{
				{stringRulesFieldNumber, minLenFieldNumberInStringRules},
				{stringRulesFieldNumber, maxBytesFieldNumberInStringRules},
			},
			"Field %q has a %s (%d) higher than its %s (%d).",
			adder.fieldName(),
			adder.getFieldRuleName(stringRulesFieldNumber, maxBytesFieldNumberInStringRules),
			*stringRules.MaxBytes,
			adder.getFieldRuleName(stringRulesFieldNumber, minLenFieldNumberInStringRules),
			*stringRules.MinLen,
		)
	}
	if stringRules.MaxLen != nil && stringRules.MinBytes != nil && *stringRules.MaxLen*4 < *stringRules.MinBytes {
		adder.addForPathsf(
			[][]int32{
				{stringRulesFieldNumber, minBytesFieldNumberInStringRules},
				{stringRulesFieldNumber, maxLenFieldNumberInStringRules},
			},
			"Field %q has a %s (%d) higher than 4 times its %s (%d), the maximum number of bytes a string of length %d can have.",
			adder.fieldName(),
			adder.getFieldRuleName(stringRulesFieldNumber, maxLenFieldNumberInStringRules),
			*stringRules.MaxLen,
			adder.getFieldRuleName(stringRulesFieldNumber, minBytesFieldNumberInStringRules),
			*stringRules.MinBytes,
			*stringRules.MaxLen,
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
		substringFieldNumber := substringField.fieldNumber
		if runeCount := uint64(utf8.RuneCountInString(substring)); stringRules.MaxLen != nil && runeCount > *stringRules.MaxLen {
			adder.addForPathf(
				[]int32{stringRulesFieldNumber, substringFieldNumber},
				"Field %q has a %s of length %d, exceeding its max_len (%d).",
				adder.fieldName(),
				adder.getFieldRuleName(stringRulesFieldNumber, substringFieldNumber),
				runeCount,
				*stringRules.MaxLen,
			)
		}
		if lenBytes := uint64(len(substring)); stringRules.MaxBytes != nil && lenBytes > *stringRules.MaxBytes {
			adder.addForPathf(
				[]int32{stringRulesFieldNumber, substringFieldNumber},
				"Field %q has a %s of %d bytes, exceeding its max_bytes (%d).",
				adder.fieldName(),
				adder.getFieldRuleName(stringRulesFieldNumber, substringFieldNumber),
				lenBytes,
				*stringRules.MaxBytes,
			)
		}
		if stringRules.NotContains != nil && strings.Contains(substring, *stringRules.NotContains) {
			adder.addForPathf(
				[]int32{stringRulesFieldNumber, substringFieldNumber},
				"Field %q has a %s (%q) containing its not_contains (%q).",
				adder.fieldName(),
				adder.getFieldRuleName(stringRulesFieldNumber, substringFieldNumber),
				substring,
				*stringRules.NotContains,
			)
		}
		if stringRules.NotContains != nil && strings.Contains(*stringRules.NotContains, substring) {
			adder.addForPathf(
				[]int32{stringRulesFieldNumber, substringFieldNumber},
				"Field %q has a %s (%q) containing its %s (%q).",
				adder.fieldName(),
				adder.getFieldRuleName(stringRulesFieldNumber, notContainsFieldNumberInStringRules),
				*stringRules.NotContains,
				substringField.name,
				substring,
			)
		}
	}
	if stringRules.Pattern != nil {
		if _, err := regexp.Compile(*stringRules.Pattern); err != nil {
			adder.addForPathf(
				[]int32{stringRulesFieldNumber, patternFieldNumberInStringRules},
				"Field %q has a %s that fails to compile: %s.",
				adder.fieldName(),
				adder.getFieldRuleName(stringRulesFieldNumber, patternFieldNumberInStringRules),
				err.Error(),
			)
		}
	}
	nonStrict := stringRules.Strict != nil && !*stringRules.Strict
	if stringRules.GetWellKnownRegex() == validate.KnownRegex_KNOWN_REGEX_UNSPECIFIED && nonStrict {
		adder.addForPathf(
			[]int32{stringRulesFieldNumber, strictFieldNumberInStringRules},
			"Field %q has %s without %s.",
			adder.fieldName(),
			adder.getFieldRuleName(stringRulesFieldNumber, strictFieldNumberInStringRules),
			adder.getFieldRuleName(stringRulesFieldNumber, wellKnownRegexFieldNumberInStringRules),
		)
	}
	return nil
}

func checkBytesRules(adder *adder, bytesRules *validate.BytesRules) error {
	checkConstAndIn(adder, bytesRules, bytesRulesFieldNumber)
	if err := checkLenRules(adder, bytesRules, bytesRulesFieldNumber, "len", "min_len", "max_len"); err != nil {
		return err
	}
	subBytesFields := []struct {
		value       []byte
		name        string
		fieldNumber int32
	}{
		{value: bytesRules.Prefix, name: "prefix", fieldNumber: prefixFieldNumberInBytesRules},
		{value: bytesRules.Suffix, name: "suffix", fieldNumber: suffixFieldNumberInBytesRules},
		{value: bytesRules.Contains, name: "contains", fieldNumber: containsFieldNumberInBytesRules},
	}
	for _, subBytesField := range subBytesFields {
		if bytesRules.MaxLen != nil && uint64(len(subBytesField.value)) > *bytesRules.MaxLen {
			adder.addForPathf(
				[]int32{bytesRulesFieldNumber, subBytesField.fieldNumber},
				"Field %q has a %s of %d bytes, exceeding its max_len (%d).",
				adder.fieldName(),
				adder.getFieldRuleName(bytesRulesFieldNumber, subBytesField.fieldNumber),
				len(subBytesField.value),
				*bytesRules.MaxLen,
			)
		}
	}
	if bytesRules.Pattern != nil {
		if _, err := regexp.Compile(*bytesRules.Pattern); err != nil {
			adder.addForPathf(
				[]int32{bytesRulesFieldNumber, patternFieldNumberInBytesRules},
				"Field %q has a %s that fails to compile: %s.",
				adder.fieldName(),
				adder.getFieldRuleName(bytesRulesFieldNumber, patternFieldNumberInBytesRules),
				err.Error(),
			)
		}
	}
	return nil
}

func checkEnumRules(
	adder *adder,
	enumRules *validate.EnumRules,
) {
	checkConstAndIn(adder, enumRules, enumRulesFieldNumber)
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
		func(d *durationpb.Duration) interface{} { return d },
	)
}

func checkTimestampRules(adder *adder, timestampRules *validate.TimestampRules) error {
	if err := checkNumericRules[timestamppb.Timestamp](
		adder,
		timestampRulesFieldNumber,
		timestampRules.ProtoReflect(),
		getTimestampFromValue,
		compareTimestamp,
		func(t *timestamppb.Timestamp) interface{} { return t },
	); err != nil {
		return err
	}
	if timestampRules.GetLtNow() && timestampRules.GetGtNow() {
		adder.addForPathsf(
			[][]int32{
				{timestampRulesFieldNumber, gtNowFieldNumberInTimestampRules},
				{timestampRulesFieldNumber, ltNowFieldNumberInTimestampRules},
			},
			"Field %q has both %s and %s.",
			adder.fieldName(),
			adder.getFieldRuleName(timestampRulesFieldNumber, gtNowFieldNumberInTimestampRules),
			adder.getFieldRuleName(timestampRulesFieldNumber, ltNowFieldNumberInTimestampRules),
		)
	}
	if timestampRules.Within != nil {
		if durationErrString := checkDuration(timestampRules.Within); durationErrString != "" {
			adder.addForPathf(
				[]int32{timestampRulesFieldNumber, withInFieldNumberInTimestampRules},
				"Field %q has an invalid %s: %s.",
				adder.fieldName(),
				adder.getFieldRuleName(timestampRulesFieldNumber, withInFieldNumberInTimestampRules),
				durationErrString,
			)
		} else if timestampRules.Within.Seconds <= 0 && timestampRules.Within.Nanos <= 0 {
			adder.addForPathf(
				[]int32{timestampRulesFieldNumber, withInFieldNumberInTimestampRules},
				"Field %q has a negative %s (%v).",
				adder.fieldName(),
				adder.getFieldRuleName(timestampRulesFieldNumber, withInFieldNumberInTimestampRules),
				timestampRules.Within,
			)
		}
	}
	return nil
}

func checkConstAndIn(adder *adder, rule proto.Message, ruleFieldNumber int32) {
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
			[]int32{ruleFieldNumber, constFieldNumber},
			"Field %q has %s and does not need other rules in %s.",
			adder.fieldName(),
			adder.getFieldRuleName(ruleFieldNumber, constFieldNumber),
			adder.getFieldRuleName(ruleFieldNumber),
		)
	}
	if isInSpecified && fieldCount > 1 {
		adder.addForPathf(
			[]int32{ruleFieldNumber, inFieldNumber},
			"Field %q has %s and does not need other rules in %s.",
			adder.fieldName(),
			adder.getFieldRuleName(ruleFieldNumber, inFieldNumber),
			adder.getFieldRuleName(ruleFieldNumber),
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
			lengthValue, ok := v.Interface().(uint64)
			if !ok {
				err = fmt.Errorf("%v is not an uint64", v.Interface())
				return false
			}
			length = &lengthValue
			lengthFieldNumber = int32(fd.Number())
		case minLenFieldName:
			lengthValue, ok := v.Interface().(uint64)
			if !ok {
				err = fmt.Errorf("%v is not an uint64", v.Interface())
				return false
			}
			minLen = &lengthValue
			minLenFieldNumber = int32(fd.Number())
		case maxLenFieldName:
			lengthValue, ok := v.Interface().(uint64)
			if !ok {
				err = fmt.Errorf("%v is not an uint64", v.Interface())
				return false
			}
			maxLen = &lengthValue
			maxLenFieldNumber = int32(fd.Number())
		}
		return true
	})
	if err != nil {
		return err
	}
	if length != nil && minLen != nil {
		adder.addForPathf(
			[]int32{ruleFieldNumber, lengthFieldNumber},
			"Field %q has both %s and %s.",
			adder.fieldName(),
			adder.getFieldRuleName(ruleFieldNumber, lengthFieldNumber),
			adder.getFieldRuleName(ruleFieldNumber, minLenFieldNumber),
		)
	}
	if length != nil && maxLen != nil {
		adder.addForPathf(
			[]int32{ruleFieldNumber, lengthFieldNumber},
			"Field %q has both %s and %s.",
			adder.fieldName(),
			adder.getFieldRuleName(ruleFieldNumber, lengthFieldNumber),
			adder.getFieldRuleName(ruleFieldNumber, maxLenFieldNumber),
		)
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
			"Field %q has a %s (%d) higher than its %s (%d).",
			adder.fieldName(),
			adder.getFieldRuleName(ruleFieldNumber, minLenFieldNumber),
			*minLen,
			adder.getFieldRuleName(ruleFieldNumber, maxLenFieldNumber),
			*maxLen,
		)
	} else if *minLen == *maxLen {
		adder.addForPathsf(
			[][]int32{
				{ruleFieldNumber, minLenFieldNumber},
				{ruleFieldNumber, maxLenFieldNumber},
			},
			"Field %q has equal %s and %s, use const instead.",
			adder.fieldName(),
			adder.getFieldRuleName(ruleFieldNumber, minLenFieldNumber),
			maxLenFieldName,
		)
	}
	return nil
}

func getFieldTypePrettyNameName(fieldDescriptor protoreflect.FieldDescriptor) string {
	if !isFieldDescriptorMessage(fieldDescriptor) {
		return fieldDescriptor.Kind().String()
	}
	if fieldDescriptor.IsMap() {
		return fmt.Sprintf(
			"map<%s, %s>",
			getFieldTypePrettyNameName(fieldDescriptor.MapKey()),
			getFieldTypePrettyNameName(fieldDescriptor.MapValue()),
		)
	}
	return string(fieldDescriptor.Message().FullName())
}

func isFieldDescriptorMessage(fieldDescriptor protoreflect.FieldDescriptor) bool {
	return fieldDescriptor.Kind() == protoreflect.MessageKind || fieldDescriptor.Kind() == protoreflect.GroupKind
}
