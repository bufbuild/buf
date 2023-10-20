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
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"github.com/bufbuild/buf/private/pkg/protosource"
	"github.com/bufbuild/buf/private/pkg/stringutil"
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
	// https://buf.build/bufbuild/protovalidate/file/main:buf/validate/validate.proto#L2669
	notInFieldNumberInStringRules = 11
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
	// https://buf.build/bufbuild/protovalidate/file/main:buf/validate/validate.proto#L3037
	notInFieldNumberInBytesRules = 9
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L3164
	notInFieldNumberInEnumRules = 4
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
	// https://buf.build/bufbuild/protovalidate/file/v0.4.4:buf/validate/validate.proto#L169
	typeOneofDescriptor = validate.File_buf_validate_validate_proto.Messages().ByName("FieldConstraints").Oneofs().ByName("type")
)

// validateRulesForSingleField validates that protovalidate rules defined for this field are
// valid, not including CEL expressions.
func validateRulesForSingleField(
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
	fieldConstraintsMessage := fieldConstraints.ProtoReflect()
	typeRulesFieldDescriptor := fieldConstraintsMessage.WhichOneof(typeOneofDescriptor)
	if typeRulesFieldDescriptor == nil {
		return nil
	}
	typeRulesFieldNumber := int32(typeRulesFieldDescriptor.Number())
	// checkMapRules and checkRepeatedRules are special cases that call checkConstraintsForField.
	if typeRulesFieldNumber == mapRulesFieldNumber {
		return checkMapRules(adder, fieldConstraints.GetMap(), fieldDescriptor)
	}
	if typeRulesFieldNumber == repeatedRulesFieldNumber {
		return checkRepeatedRules(adder, fieldConstraints.GetRepeated(), fieldDescriptor)
	}
	typeCheck := checkRulesTypeMatchFieldType(adder, fieldDescriptor, typeRulesFieldNumber)
	if !typeCheck {
		return nil
	}
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
		return checkEnumRules(adder, fieldConstraints.GetEnum(), fieldDescriptor.Enum().Values())
	case anyRulesFieldNumber:
		checkAnyRules(adder, fieldConstraints.GetAny())
	case durationRulesFieldNumber:
		return checkDurationRules(adder, fieldConstraints.GetDuration())
	case timestampRulesFieldNumber:
		return checkTimestampRules(adder, fieldConstraints.GetTimestamp())
	}
	return nil
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
		"Field %q is a %s but has %s rules.",
		adder.fieldName(),
		adder.fieldPrettyTypeName,
		adder.getFieldRuleName(ruleFieldNumber),
	)
	return false
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
				"Field %q is not a scalar type or a wrapper type but has %s set to true.",
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
			"Field %q has a %s higher than %s.",
			baseAdder.fieldName(),
			baseAdder.getFieldRuleName(repeatedRulesFieldNumber, minItemsFieldNumberInRepeatedFieldRules),
			baseAdder.getFieldRuleName(repeatedRulesFieldNumber, maxItemsFieldNumberInRepeatedFieldRules),
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
			"Field %q has a %s higher than %s.",
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
	var regex *regexp.Regexp
	var err error
	if stringRules.Pattern != nil {
		regex, err = regexp.Compile(*stringRules.Pattern)
		if err != nil {
			adder.addForPathf(
				[]int32{stringRulesFieldNumber, patternFieldNumberInStringRules},
				"unable to parse regex pattern %s: %w", *stringRules.Pattern, err,
			)
		}
	}
	for i, bannedValue := range stringRules.GetNotIn() {
		var rejectingRules []string
		if stringRules.Len != nil && uint64(utf8.RuneCountInString(bannedValue)) != *stringRules.Len {
			rejectingRules = append(rejectingRules, "len")
		}
		if stringRules.MaxLen != nil && uint64(utf8.RuneCountInString(bannedValue)) > *stringRules.MaxLen {
			rejectingRules = append(rejectingRules, "max_len")
		}
		if stringRules.MinLen != nil && uint64(utf8.RuneCountInString(bannedValue)) < *stringRules.MinLen {
			rejectingRules = append(rejectingRules, "min_len")
		}
		if stringRules.LenBytes != nil && uint64(len(bannedValue)) != *stringRules.LenBytes {
			rejectingRules = append(rejectingRules, "len")
		}
		if stringRules.MaxBytes != nil && uint64(len(bannedValue)) > *stringRules.MaxBytes {
			rejectingRules = append(rejectingRules, "max_bytes")
		}
		if stringRules.MinBytes != nil && uint64(len(bannedValue)) < *stringRules.MinBytes {
			rejectingRules = append(rejectingRules, "min_bytes")
		}
		if stringRules.Prefix != nil && !strings.HasPrefix(bannedValue, *stringRules.Prefix) {
			rejectingRules = append(rejectingRules, "prefix")
		}
		if stringRules.Suffix != nil && !strings.HasSuffix(bannedValue, *stringRules.Suffix) {
			rejectingRules = append(rejectingRules, "suffix")
		}
		if stringRules.Contains != nil && !strings.Contains(bannedValue, *stringRules.Contains) {
			rejectingRules = append(rejectingRules, "contains")
		}
		if stringRules.NotContains != nil && strings.Contains(bannedValue, *stringRules.NotContains) {
			rejectingRules = append(rejectingRules, "not_contains")
		}
		if regex != nil && !regex.MatchString(bannedValue) {
			rejectingRules = append(rejectingRules, "pattern")
		}
		if len(rejectingRules) > 0 {
			adder.addForPathf(
				[]int32{stringRulesFieldNumber, notInFieldNumberInStringRules, int32(i)},
				"%s is already rejected by %s and does not need to be in not_in",
				bannedValue,
				stringutil.SliceToHumanString(rejectingRules),
			)
		}
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
	var regex *regexp.Regexp
	var err error
	if bytesRules.Pattern != nil {
		regex, err = regexp.Compile(*bytesRules.Pattern)
		if err != nil {
			adder.addForPathf(
				[]int32{bytesRulesFieldNumber, patternFieldNumberInBytesRules},
				"unable to parse regex pattern %s: %w", *bytesRules.Pattern, err,
			)
		}
	}
	for i, bannedValue := range bytesRules.GetNotIn() {
		var rejectingRules []string
		if bytesRules.Len != nil && uint64(len(bannedValue)) != *bytesRules.Len {
			rejectingRules = append(rejectingRules, "len")
		}
		if bytesRules.MaxLen != nil && uint64(len(bannedValue)) > *bytesRules.MaxLen {
			rejectingRules = append(rejectingRules, "max_bytes")
		}
		if bytesRules.MinLen != nil && uint64(len(bannedValue)) < *bytesRules.MinLen {
			rejectingRules = append(rejectingRules, "min_bytes")
		}
		if !bytes.HasPrefix(bannedValue, bytesRules.Prefix) {
			rejectingRules = append(rejectingRules, "prefix")
		}
		if !bytes.HasSuffix(bannedValue, bytesRules.Suffix) {
			rejectingRules = append(rejectingRules, "suffi")
		}
		if !bytes.Contains(bannedValue, bytesRules.Contains) {
			rejectingRules = append(rejectingRules, "contains")
		}
		if regex != nil && !regex.Match(bannedValue) {
			rejectingRules = append(rejectingRules, "pattern")
		}
		if len(rejectingRules) > 0 {
			adder.addForPathf(
				[]int32{bytesRulesFieldNumber, notInFieldNumberInBytesRules, int32(i)},
				"%s is already rejected by %s and does not need to be in not_in",
				bannedValue,
				stringutil.SliceToHumanString(rejectingRules),
			)
		}
	}
	return nil
}

func checkEnumRules(
	adder *adder,
	enumRules *validate.EnumRules,
	enumValueDescriptors protoreflect.EnumValueDescriptors,
) error {
	checkConstAndIn(adder, enumRules, enumRulesFieldNumber)
	if !enumRules.GetDefinedOnly() {
		return nil
	}
	if len(enumRules.In) == 0 && len(enumRules.NotIn) == 0 {
		return nil
	}
	for _, notIn := range enumRules.NotIn {
		if enumValueDescriptors.ByNumber(protoreflect.EnumNumber(notIn)) == nil {
			adder.addForPathf(
				[]int32{enumRulesFieldNumber, notInFieldNumberInEnumRules},
				"value %d is rejected by defined_only and does not need to be in not_in",
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
		adder.addForPathf(
			[]int32{timestampRulesFieldNumber},
			"gt_now and lt_now cannot be used together",
		)
	}
	if timestampRules.Within != nil {
		if !timestampRules.Within.IsValid() {
			adder.addForPathf(
				[]int32{timestampRulesFieldNumber, withInFieldNumberInTimestampRules},
				"within duration is invalid",
			)
		}
		if timestampRules.Within.Seconds <= 0 && timestampRules.Within.Nanos <= 0 {
			adder.addForPathf(
				[]int32{timestampRulesFieldNumber},
				"within duration must be positive",
			)
		}
	}
	// TODO: not sure if we really need to check this:
	areNowRulesDefined := timestampRules.GetLtNow() || timestampRules.GetGtNow() || timestampRules.Within != nil
	areAbsoluteRulesDefined := timestampRules.GetLt() != nil || timestampRules.GetLte() != nil || timestampRules.GetGt() != nil || timestampRules.GetGte() != nil
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
