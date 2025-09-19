// Copyright 2020-2025 Buf Technologies, Inc.
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
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"buf.build/go/protovalidate"
	"github.com/bufbuild/buf/private/bufpkg/bufprotosource"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	// https://buf.build/bufbuild/protovalidate/docs/main:buf.validate#buf.validate.FieldRule
	// These numbers are used for two purposes:
	// 1. Identity which type oneof is specified in a FieldRule. i.e. Is DoubleRules defined or
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
	ignoreFieldNumber         = 27
	// https://buf.build/bufbuild/protovalidate/docs/v0.5.1:buf.validate#buf.validate.StringRules
	minLenFieldNumberInStringRules         = 2
	maxLenFieldNumberInStringRules         = 3
	minBytesFieldNumberInStringRules       = 4
	maxBytesFieldNumberInStringRules       = 5
	patternFieldNumberInStringRules        = 6
	prefixFieldNumberInStringRules         = 7
	suffixFieldNumberInStringRules         = 8
	containsFieldNumberInStringRules       = 9
	notContainsFieldNumberInStringRules    = 23
	wellKnownRegexFieldNumberInStringRules = 24
	strictFieldNumberInStringRules         = 25
	// https://buf.build/bufbuild/protovalidate/docs/v0.5.1:buf.validate#buf.validate.BytesRules
	patternFieldNumberInBytesRules  = 4
	prefixFieldNumberInBytesRules   = 5
	suffixFieldNumberInBytesRules   = 6
	containsFieldNumberInBytesRules = 7
	// https://buf.build/bufbuild/protovalidate/docs/v0.5.1:buf.validate#buf.validate.RepeatedRules
	minItemsFieldNumberInRepeatedFieldRules = 1
	maxItemsFieldNumberInRepeatedFieldRules = 2
	uniqueFieldNumberInRepeatedFieldRules   = 3
	itemsFieldNumberInRepeatedRules         = 4
	// https://buf.build/bufbuild/protovalidate/docs/v0.5.1:buf.validate#buf.validate.MapRules
	minPairsFieldNumberInMapRules = 1
	maxPairsFieldNumberInMapRules = 2
	keysFieldNumberInMapRules     = 4
	valuesFieldNumberInMapRules   = 5
	// https://buf.build/bufbuild/protovalidate/docs/v0.5.1:buf.validate#buf.validate.TimestampRules
	ltNowFieldNumberInTimestampRules  = 7
	gtNowFieldNumberInTimestampRules  = 8
	withInFieldNumberInTimestampRules = 9

	exampleName = "example"
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
	// https://buf.build/bufbuild/protovalidate/docs/v0.5.1:buf.validate#buf.validate.FieldRules
	fieldRulesDescriptor = validate.File_buf_validate_validate_proto.Messages().ByName("FieldRules")
	typeOneofDescriptor  = fieldRulesDescriptor.Oneofs().ByName("type")
)

// checkField validates that protovalidate rules defined for this field are
// valid, not including CEL expressions.
func checkField(
	add func(bufprotosource.Descriptor, bufprotosource.Location, []bufprotosource.Location, string, ...any),
	field bufprotosource.Field,
	extensionTypeResolver protoencoding.Resolver,
) error {
	fieldDescriptor, err := field.AsDescriptor()
	if err != nil {
		return err
	}
	constraints, err := protovalidate.ResolveFieldRules(fieldDescriptor)
	if err != nil {
		return err
	}
	return checkRulesForField(
		&adder{
			field:               field,
			fieldPrettyTypeName: getFieldTypePrettyNameName(fieldDescriptor),
			addFunc:             add,
		},
		constraints,
		fieldDescriptor.ContainingMessage(),
		nil,
		fieldDescriptor,
		fieldDescriptor.Cardinality() == protoreflect.Repeated,
		extensionTypeResolver,
	)
}

func checkRulesForField(
	adder *adder,
	fieldRules *validate.FieldRules,
	// This is needed because recursive calls of this function still need the same
	// containing message. For example, checkRulesForField(.., fieldDescriptor, ...)
	// may call checkRulesForField(..., fieldDescriptor.MapKey(), ...), but the map
	// key should be associated with the same containing message as the field's. Since
	// fieldDescriptor.MapKey().ContainingMessage() is not the same as fieldDescriptor.ContainingMessage(),
	// this needs to be passed around.
	containingMessageDescriptor protoreflect.MessageDescriptor,
	// Only pass a non nil value when the field is a synthetic map key/value field
	parentMapFieldDescriptor protoreflect.FieldDescriptor,
	fieldDescriptor protoreflect.FieldDescriptor,
	expectRepeatedRule bool,
	extensionTypeResolver protoencoding.Resolver,
) error {
	if fieldRules == nil {
		return nil
	}
	if fieldDescriptor.IsExtension() {
		checkRulesForExtension(adder, fieldRules)
	}
	if fieldDescriptor.ContainingOneof() != nil &&
		!protodesc.ToFieldDescriptorProto(fieldDescriptor).GetProto3Optional() &&
		fieldRules.GetRequired() {
		adder.addForPathf(
			[]int32{requiredFieldNumber},
			"Field %q has %s but is in a oneof (%s). Oneof fields must not have %s.",
			adder.fieldName(),
			adder.getFieldRuleName(requiredFieldNumber),
			fieldDescriptor.ContainingOneof().Name(),
			adder.getFieldRuleName(requiredFieldNumber),
		)
	}
	checkFieldFlags(adder, fieldDescriptor, fieldRules)
	if err := checkCELForField(
		adder,
		fieldRules,
		fieldDescriptor,
		!expectRepeatedRule,
	); err != nil {
		return err
	}
	fieldRulesMessage := fieldRules.ProtoReflect()
	typeRulesFieldDescriptor := fieldRulesMessage.WhichOneof(typeOneofDescriptor)
	if typeRulesFieldDescriptor == nil {
		return nil
	}
	typeRulesFieldNumber := int32(typeRulesFieldDescriptor.Number())
	// Map and repeated special cases that contain fieldRules.
	if typeRulesFieldNumber == mapRulesFieldNumber {
		return checkMapRules(adder, fieldRules.GetMap(), fieldDescriptor, containingMessageDescriptor, extensionTypeResolver)
	}
	if typeRulesFieldNumber == repeatedRulesFieldNumber {
		return checkRepeatedRules(adder, fieldRules.GetRepeated(), fieldDescriptor, containingMessageDescriptor, extensionTypeResolver)
	}
	typesMatch := checkRulesTypeMatchFieldType(adder, fieldDescriptor, typeRulesFieldNumber, expectRepeatedRule)
	if !typesMatch {
		return nil
	}
	typeRulesMessage := fieldRulesMessage.Get(typeRulesFieldDescriptor).Message()
	var exampleValues []protoreflect.Value
	var exampleFieldNumber int32
	typeRulesMessage.Range(func(fd protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		if string(fd.Name()) == exampleName {
			exampleFieldNumber = int32(fd.Number())
			// This assumed all *Rules.Example are repeated, otherwise it panics.
			list := value.List()
			for i := range list.Len() {
				exampleValues = append(exampleValues, list.Get(i))
			}
			return false
		}
		return true
	})
	if len(exampleValues) > 0 {
		if err := checkExampleValues(
			adder,
			[]int32{typeRulesFieldNumber, exampleFieldNumber},
			fieldRules,
			typeRulesMessage,
			containingMessageDescriptor,
			parentMapFieldDescriptor,
			fieldDescriptor,
			exampleValues,
			extensionTypeResolver,
		); err != nil {
			return err
		}
	}
	if numberRulesCheckFunc, ok := fieldNumberToCheckNumberRulesFunc[typeRulesFieldNumber]; ok {
		numberRulesMessage := fieldRulesMessage.Get(typeRulesFieldDescriptor).Message()
		return numberRulesCheckFunc(adder, typeRulesFieldNumber, numberRulesMessage)
	}
	switch typeRulesFieldNumber {
	case boolRulesFieldNumber:
		// Bool rules only have `const` and does not need validating.
	case stringRulesFieldNumber:
		return checkStringRules(adder, fieldRules.GetString())
	case bytesRulesFieldNumber:
		return checkBytesRules(adder, fieldRules.GetBytes())
	case enumRulesFieldNumber:
		checkEnumRules(adder, fieldRules.GetEnum())
	case anyRulesFieldNumber:
		checkAnyRules(adder, fieldRules.GetAny())
	case durationRulesFieldNumber:
		return checkDurationRules(adder, fieldRules.GetDuration())
	case timestampRulesFieldNumber:
		return checkTimestampRules(adder, fieldRules.GetTimestamp())
	}
	return nil
}

func checkFieldFlags(
	adder *adder,
	fieldDescriptor protoreflect.FieldDescriptor,
	fieldRules *validate.FieldRules,
) {
	var fieldCount int
	fieldRules.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		fieldCount++
		return true
	})
	if fieldRules.GetIgnore() == validate.Ignore_IGNORE_ALWAYS && fieldCount > 1 {
		adder.addForPathf(
			[]int32{ignoreFieldNumber},
			"Field %q has %s=%v and therefore other rules in %s are not applied and should be removed.",
			adder.fieldName(),
			adder.getFieldRuleName(ignoreFieldNumber),
			validate.Ignore_IGNORE_ALWAYS,
			adder.getFieldRuleName(),
		)
	}
	if fieldRules.GetRequired() && fieldRules.GetIgnore() == validate.Ignore_IGNORE_IF_ZERO_VALUE {
		adder.addForPathsf(
			[][]int32{
				{requiredFieldNumber},
				{ignoreFieldNumber},
			},
			"Field %q has both %s and %s=%v. A field cannot be empty if it is required.",
			adder.fieldName(),
			adder.getFieldRuleName(requiredFieldNumber),
			adder.getFieldRuleName(ignoreFieldNumber),
			validate.Ignore_IGNORE_IF_ZERO_VALUE,
		)
	}
	if fieldRules.GetIgnore() == validate.Ignore_IGNORE_IF_ZERO_VALUE && fieldDescriptor.HasPresence() && !fieldDescriptor.IsExtension() {
		adder.addForPathf(
			[]int32{ignoreFieldNumber},
			"Field %q has %s=%v and tracks presence. This is the same the default and the ignore option can be removed.",
			adder.fieldName(),
			adder.getFieldRuleName(ignoreFieldNumber),
			validate.Ignore_IGNORE_IF_ZERO_VALUE,
			adder.getFieldRuleName(),
		)
	}
}

// Assumes the rule isn't a map rule or repeated rule, but the field could be a
// map or a repeated field.
func checkRulesTypeMatchFieldType(
	adder *adder,
	fieldDescriptor protoreflect.FieldDescriptor,
	ruleFieldNumber int32,
	expectRepeatedRule bool,
) bool {
	if expectRepeatedRule {
		adder.addForPathf(
			[]int32{ruleFieldNumber},
			"Field %q is of type repeated %s but has %s rules.",
			adder.fieldName(),
			adder.fieldPrettyTypeName,
			adder.getFieldRuleName(ruleFieldNumber),
		)
		return false
	}
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
		"Field %q is of type %s but has %s rules.",
		adder.fieldName(),
		adder.fieldPrettyTypeName,
		adder.getFieldRuleName(ruleFieldNumber),
	)
	return false
}

func checkRulesForExtension(
	adder *adder,
	fieldRules *validate.FieldRules,
) {
	if fieldRules.GetRequired() {
		adder.addForPathf(
			[]int32{requiredFieldNumber},
			"Field %q is an extension field and cannot have %s.",
			adder.fieldName(),
			adder.getFieldRuleName(requiredFieldNumber),
		)
	}
	if fieldRules.GetIgnore() == validate.Ignore_IGNORE_IF_ZERO_VALUE {
		adder.addForPathf(
			[]int32{ignoreFieldNumber},
			"Field %q is an extension field and cannot have %s=%v.",
			adder.fieldName(),
			adder.getFieldRuleName(ignoreFieldNumber),
			validate.Ignore_IGNORE_IF_ZERO_VALUE,
		)
	}
}

func checkRepeatedRules(
	baseAdder *adder,
	repeatedRules *validate.RepeatedRules,
	fieldDescriptor protoreflect.FieldDescriptor,
	containingMessageDescriptor protoreflect.MessageDescriptor,
	extensionTypeResolver protoencoding.Resolver,
) error {
	if !fieldDescriptor.IsList() {
		baseAdder.addForPathf(
			[]int32{repeatedRulesFieldNumber},
			"Field %q is not repeated but has %s.",
			baseAdder.fieldName(),
			baseAdder.getFieldRuleName(repeatedRulesFieldNumber),
		)
		return nil
	}
	if repeatedRules.GetUnique() && isFieldDescriptorMessage(fieldDescriptor) {
		if _, isFieldWrapper := wrapperTypeNames[string(fieldDescriptor.Message().FullName())]; !isFieldWrapper {
			baseAdder.addForPathf(
				[]int32{repeatedRulesFieldNumber, uniqueFieldNumberInRepeatedFieldRules},
				"Field %q is of type %s but has %s set to true, which is only allowed for scalar types and wrapper types.",
				baseAdder.fieldName(),
				baseAdder.fieldPrettyTypeName,
				baseAdder.getFieldRuleName(repeatedRulesFieldNumber, uniqueFieldNumberInRepeatedFieldRules),
			)
		}
	}
	if repeatedRules.MinItems != nil && repeatedRules.MaxItems != nil && *repeatedRules.MinItems > *repeatedRules.MaxItems {
		baseAdder.addForPathf(
			[]int32{repeatedRulesFieldNumber, minItemsFieldNumberInRepeatedFieldRules},
			"Field %q has value %d for %s, which must be higher than value %d for %s.",
			baseAdder.fieldName(),
			*repeatedRules.MinItems,
			baseAdder.getFieldRuleName(repeatedRulesFieldNumber, minItemsFieldNumberInRepeatedFieldRules),
			*repeatedRules.MaxItems,
			baseAdder.getFieldRuleName(repeatedRulesFieldNumber, maxItemsFieldNumberInRepeatedFieldRules),
		)
		baseAdder.addForPathf(
			[]int32{repeatedRulesFieldNumber, maxItemsFieldNumberInRepeatedFieldRules},
			"Field %q has value %d for %s, which must be lower than value %d for %s.",
			baseAdder.fieldName(),
			*repeatedRules.MaxItems,
			baseAdder.getFieldRuleName(repeatedRulesFieldNumber, maxItemsFieldNumberInRepeatedFieldRules),
			*repeatedRules.MinItems,
			baseAdder.getFieldRuleName(repeatedRulesFieldNumber, minItemsFieldNumberInRepeatedFieldRules),
		)
	}
	itemAdder := baseAdder.cloneWithNewBasePath(repeatedRulesFieldNumber, itemsFieldNumberInRepeatedRules)
	return checkRulesForField(itemAdder, repeatedRules.Items, containingMessageDescriptor, nil, fieldDescriptor, false, extensionTypeResolver)
}

func checkMapRules(
	baseAdder *adder,
	mapRules *validate.MapRules,
	fieldDescriptor protoreflect.FieldDescriptor,
	containingMessageDescriptor protoreflect.MessageDescriptor,
	extensionTypeResolver protoencoding.Resolver,
) error {
	if !fieldDescriptor.IsMap() {
		baseAdder.addForPathf(
			[]int32{mapRulesFieldNumber},
			"Field %q is not a map but has %s.",
			baseAdder.fieldName(),
			baseAdder.getFieldRuleName(mapRulesFieldNumber),
		)
		return nil
	}
	if mapRules.MinPairs != nil && mapRules.MaxPairs != nil && *mapRules.MinPairs > *mapRules.MaxPairs {
		baseAdder.addForPathf(
			[]int32{mapRulesFieldNumber, minPairsFieldNumberInMapRules},
			"Field %q has value %d for %s, which must be lower than value %d for %s.",
			baseAdder.fieldName(),
			*mapRules.MinPairs,
			baseAdder.getFieldRuleName(mapRulesFieldNumber, minPairsFieldNumberInMapRules),
			*mapRules.MaxPairs,
			baseAdder.getFieldRuleName(mapRulesFieldNumber, maxPairsFieldNumberInMapRules),
		)
		baseAdder.addForPathf(
			[]int32{mapRulesFieldNumber, maxPairsFieldNumberInMapRules},
			"Field %q has value %d for %s, which is lower than value %d for %s.",
			baseAdder.fieldName(),
			*mapRules.MaxPairs,
			baseAdder.getFieldRuleName(mapRulesFieldNumber, maxPairsFieldNumberInMapRules),
			*mapRules.MinPairs,
			baseAdder.getFieldRuleName(mapRulesFieldNumber, minPairsFieldNumberInMapRules),
		)
	}
	keyAdder := baseAdder.cloneWithNewBasePath(mapRulesFieldNumber, keysFieldNumberInMapRules)
	err := checkRulesForField(keyAdder, mapRules.Keys, containingMessageDescriptor, fieldDescriptor, fieldDescriptor.MapKey(), false, extensionTypeResolver)
	if err != nil {
		return err
	}
	valueAdder := baseAdder.cloneWithNewBasePath(mapRulesFieldNumber, valuesFieldNumberInMapRules)
	return checkRulesForField(valueAdder, mapRules.Values, containingMessageDescriptor, fieldDescriptor, fieldDescriptor.MapValue(), false, extensionTypeResolver)
}

func checkStringRules(adder *adder, stringRules *validate.StringRules) error {
	checkConst(adder, stringRules, stringRulesFieldNumber)
	if err := checkLenRules(adder, stringRules, stringRulesFieldNumber, "len", "min_len", "max_len"); err != nil {
		return err
	}
	if err := checkLenRules(adder, stringRules, stringRulesFieldNumber, "len_bytes", "min_bytes", "max_bytes"); err != nil {
		return err
	}
	if stringRules.MinLen != nil && stringRules.MaxBytes != nil && *stringRules.MaxBytes < *stringRules.MinLen {
		adder.addForPathf(
			[]int32{stringRulesFieldNumber, minLenFieldNumberInStringRules},
			"Field %q has value %d for %s, which must be lower than value %d for %s. A string with %d UTF-8 characters has at least %d bytes, which is higher than %d bytes.",
			adder.fieldName(),
			*stringRules.MinLen,
			adder.getFieldRuleName(stringRulesFieldNumber, minLenFieldNumberInStringRules),
			*stringRules.MaxBytes,
			adder.getFieldRuleName(stringRulesFieldNumber, maxBytesFieldNumberInStringRules),
			*stringRules.MinLen,
			*stringRules.MinLen,
			*stringRules.MaxBytes,
		)
		adder.addForPathf(
			[]int32{stringRulesFieldNumber, maxBytesFieldNumberInStringRules},
			"Field %q has value %d for %s, which must be higher than value %d for %s. A string with %d UTF-8 characters has at least %d bytes, which is higher than %d bytes.",
			adder.fieldName(),
			*stringRules.MaxBytes,
			adder.getFieldRuleName(stringRulesFieldNumber, maxBytesFieldNumberInStringRules),
			*stringRules.MinLen,
			adder.getFieldRuleName(stringRulesFieldNumber, minLenFieldNumberInStringRules),
			*stringRules.MinLen,
			*stringRules.MinLen,
			*stringRules.MaxBytes,
		)
	}
	if stringRules.MaxLen != nil && stringRules.MinBytes != nil && *stringRules.MaxLen*4 < *stringRules.MinBytes {
		adder.addForPathf(
			[]int32{stringRulesFieldNumber, minBytesFieldNumberInStringRules},
			"Field %q has value %d for %s but %d for %s. A string with %d UTF-8 characters has at most %d bytes.",
			adder.fieldName(),
			*stringRules.MinBytes,
			adder.getFieldRuleName(stringRulesFieldNumber, minBytesFieldNumberInStringRules),
			*stringRules.MaxLen,
			adder.getFieldRuleName(stringRulesFieldNumber, maxLenFieldNumberInStringRules),
			*stringRules.MaxLen,
			*stringRules.MaxLen*4,
		)
		adder.addForPathf(
			[]int32{stringRulesFieldNumber, maxLenFieldNumberInStringRules},
			"Field %q has value %d for %s but %d for %s. A string with %d UTF-8 characters has at most %d bytes.",
			adder.fieldName(),
			*stringRules.MaxLen,
			adder.getFieldRuleName(stringRulesFieldNumber, maxLenFieldNumberInStringRules),
			*stringRules.MinBytes,
			adder.getFieldRuleName(stringRulesFieldNumber, minBytesFieldNumberInStringRules),
			*stringRules.MaxLen,
			*stringRules.MaxLen*4,
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
				"Field %q has a %s of length %d, exceeding its max_len (%d). It is impossible for a string to contain %q while having less than or equal to %d UTF-8 characters.",
				adder.fieldName(),
				adder.getFieldRuleName(stringRulesFieldNumber, substringFieldNumber),
				runeCount,
				*stringRules.MaxLen,
				substring,
				runeCount,
			)
		}
		if lenBytes := uint64(len(substring)); stringRules.MaxBytes != nil && lenBytes > *stringRules.MaxBytes {
			adder.addForPathf(
				[]int32{stringRulesFieldNumber, substringFieldNumber},
				"Field %q has a %s of %d bytes, exceeding its max_bytes (%d). It is impossible for a string to contain %q while having less than or equal to %d bytes.",
				adder.fieldName(),
				adder.getFieldRuleName(stringRulesFieldNumber, substringFieldNumber),
				lenBytes,
				*stringRules.MaxBytes,
				substring,
				lenBytes,
			)
		}
		if stringRules.NotContains != nil && strings.Contains(substring, *stringRules.NotContains) {
			adder.addForPathf(
				[]int32{stringRulesFieldNumber, substringFieldNumber},
				"Field %q has a %s (%q) containing its not_contains (%q). It is impossible for a string to contain %q without containing %q.",
				adder.fieldName(),
				adder.getFieldRuleName(stringRulesFieldNumber, substringFieldNumber),
				substring,
				*stringRules.NotContains,
				substring,
				*stringRules.NotContains,
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
			"Field %q has %s without %s. %s only applies to %s and is invalid without it.",
			adder.fieldName(),
			adder.getFieldRuleName(stringRulesFieldNumber, strictFieldNumberInStringRules),
			adder.getFieldRuleName(stringRulesFieldNumber, wellKnownRegexFieldNumberInStringRules),
			adder.getFieldRuleName(stringRulesFieldNumber, strictFieldNumberInStringRules),
			adder.getFieldRuleName(stringRulesFieldNumber, wellKnownRegexFieldNumberInStringRules),
		)
	}
	return nil
}

func checkBytesRules(adder *adder, bytesRules *validate.BytesRules) error {
	checkConst(adder, bytesRules, bytesRulesFieldNumber)
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
				"Field %q has a %s of %d bytes, exceeding its max_len (%d). It is impossible to contain %q while having less than or equal to %d bytes.",
				adder.fieldName(),
				adder.getFieldRuleName(bytesRulesFieldNumber, subBytesField.fieldNumber),
				len(subBytesField.value),
				*bytesRules.MaxLen,
				subBytesField.value,
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
	checkConst(adder, enumRules, enumRulesFieldNumber)
}

func checkAnyRules(adder *adder, anyRules *validate.AnyRules) {
	checkConst(adder, anyRules, anyRulesFieldNumber)
}

func checkDurationRules(adder *adder, r *validate.DurationRules) error {
	return checkNumericRules[durationpb.Duration](
		adder,
		durationRulesFieldNumber,
		r.ProtoReflect(),
		getDurationFromValue,
		compareDuration,
		func(d *durationpb.Duration) any { return d },
	)
}

func checkTimestampRules(adder *adder, timestampRules *validate.TimestampRules) error {
	if err := checkNumericRules[timestamppb.Timestamp](
		adder,
		timestampRulesFieldNumber,
		timestampRules.ProtoReflect(),
		getTimestampFromValue,
		compareTimestamp,
		func(t *timestamppb.Timestamp) any { return t },
	); err != nil {
		return err
	}
	if timestampRules.GetLtNow() && timestampRules.GetGtNow() {
		adder.addForPathsf(
			[][]int32{
				{timestampRulesFieldNumber, gtNowFieldNumberInTimestampRules},
				{timestampRulesFieldNumber, ltNowFieldNumberInTimestampRules},
			},
			"Field %q has both %s and %s. A timestamp cannot be both before and after validation time.",
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
				"Field %q must have a positive %s (%v).",
				adder.fieldName(),
				adder.getFieldRuleName(timestampRulesFieldNumber, withInFieldNumberInTimestampRules),
				timestampRules.Within,
			)
		}
	}
	return nil
}

func checkExampleValues(
	adder *adder,
	pathToExampleValues []int32,
	fieldRules *validate.FieldRules,
	typeRulesMessage protoreflect.Message,
	containingMessageDescriptor protoreflect.MessageDescriptor,
	// Not nil only if fieldDescriptor is a synthetic field for map key/value.
	parentMapFieldDescriptor protoreflect.FieldDescriptor,
	fieldDescriptor protoreflect.FieldDescriptor,
	exampleValues []protoreflect.Value,
	extensionTypeResolver protoencoding.Resolver,
) error {
	// A rule on this field may be from a predefined rule from an imported file. In order to
	// set example values on the message for validation and check against all rules on the field,
	// we pass in an extensionTypeResolver that includes imported files and reparse all extensions
	// for the message to ensure that we are able to resolve predefined rules.
	if err := protoencoding.ReparseExtensions(extensionTypeResolver, typeRulesMessage); err != nil {
		return err
	}
	hasRules := len(fieldRules.GetCel()) > 0
	if !hasRules {
		typeRulesMessage.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
			if string(fd.Name()) != exampleName {
				hasRules = true
				return false
			}
			return true
		})
	}
	if !hasRules {
		// Since there are no constraints to check example values against, we already checked
		// if the proper example type has been set on the field, so we can return here.
		return nil
	}
	// For each example value, instantiate a message of its containing message's type
	// and set the field that we are linting to the example value:
	//  containingMessage {
	//    ...
	//    fieldToLint: exampleValue,
	//    ...
	//  }
	// and validate this message instance with protovalidate and filter the structured
	// errors by field name to determine whether this example value fails rules defined
	// on the same field.
	validator, err := protovalidate.New()
	if err != nil {
		return err
	}
	// The shape of field path in a protovalidate.Violation depends on the type of the field descriptor.
	violationFilterFunc := func(violation *validate.Violation) bool {
		return len(violation.GetField().GetElements()) == 1 &&
			violation.GetField().GetElements()[0].GetFieldNumber() == int32(fieldDescriptor.Number()) &&
			violation.GetField().GetElements()[0].GetSubscript() == nil
	}
	switch {
	case fieldDescriptor.IsList():
		// Field path looks like repeated_field[10]
		violationFilterFunc = func(violation *validate.Violation) bool {
			if len(violation.GetField().GetElements()) != 1 ||
				violation.GetField().GetElements()[0].GetFieldNumber() != int32(fieldDescriptor.Number()) ||
				violation.GetField().GetElements()[0].GetSubscript() == nil {
				return false
			}
			_, ok := violation.GetField().GetElements()[0].Subscript.(*validate.FieldPathElement_Index)
			return ok
		}
	case parentMapFieldDescriptor != nil && fieldDescriptor.Name() == "key":
		// Field path looks like map_field["the key value that failed"] and ForKey is set to true.
		violationFilterFunc = func(violation *validate.Violation) bool {
			if !violation.GetForKey() ||
				len(violation.GetField().GetElements()) != 1 ||
				violation.GetField().GetElements()[0].GetFieldNumber() != int32(parentMapFieldDescriptor.Number()) ||
				violation.GetField().GetElements()[0].GetSubscript() == nil {
				return false
			}
			_, ok := violation.GetField().GetElements()[0].Subscript.(*validate.FieldPathElement_Index)
			return !ok
		}
	case parentMapFieldDescriptor != nil && fieldDescriptor.Name() == "value":
		// Field path looks like map_field["the key value that failed"], but ForKey is set to false.
		violationFilterFunc = func(violation *validate.Violation) bool {
			if violation.GetForKey() ||
				len(violation.GetField().GetElements()) != 1 ||
				violation.GetField().GetElements()[0].GetFieldNumber() != int32(parentMapFieldDescriptor.Number()) ||
				violation.GetField().GetElements()[0].GetSubscript() == nil {
				return false
			}
			_, ok := violation.GetField().GetElements()[0].Subscript.(*validate.FieldPathElement_Index)
			return !ok
		}
	}
	for exampleValueIndex, exampleValue := range exampleValues {
		filterFieldDescriptor := fieldDescriptor
		messageToValidate := dynamicpb.NewMessage(containingMessageDescriptor)
		switch {
		case fieldDescriptor.IsList():
			list := messageToValidate.NewField(fieldDescriptor).List()
			list.Append(exampleValue)
			messageToValidate.Set(fieldDescriptor, protoreflect.ValueOfList(list))
		case parentMapFieldDescriptor != nil:
			// In the case of map fields, we need to filter for the parent field descriptor in the
			// Filter function. The violationFilterFunc will filter to approrpriate violation for
			// "key" and "value" example values.
			filterFieldDescriptor = parentMapFieldDescriptor
			mapEntryMessageDescriptor := fieldDescriptor.ContainingMessage()
			// We are being very defensive, because a type mismatch may cause a panic in protoreflect.
			if !mapEntryMessageDescriptor.IsMapEntry() {
				return syserror.Newf("containing message %q is not a map", mapEntryMessageDescriptor.Name())
			}
			if !parentMapFieldDescriptor.IsMap() {
				return syserror.Newf("parent field descriptor %q is passed but is not a map field", parentMapFieldDescriptor.Name())
			}
			if containingMessageDescriptor.Fields().ByName(parentMapFieldDescriptor.Name()) == nil {
				return syserror.Newf("containing message %q does not have field named %q", containingMessageDescriptor.Name(), parentMapFieldDescriptor.Name())
			}
			if parentMapFieldDescriptor.Message().Name() != mapEntryMessageDescriptor.Name() {
				return syserror.Newf("field %q should have parent of type %q but has type %q", fieldDescriptor.Name(), parentMapFieldDescriptor.Message().Name(), containingMessageDescriptor.Name())
			}
			mapEntry := messageToValidate.NewField(parentMapFieldDescriptor).Map()
			switch fieldDescriptor.Name() {
			case "key":
				mapEntry.Set(
					exampleValue.MapKey(),
					dynamicpb.NewMessage(parentMapFieldDescriptor.Message()).NewField(parentMapFieldDescriptor.MapValue()),
				)
			case "value":
				mapEntry.Set(
					dynamicpb.NewMessage(parentMapFieldDescriptor.Message()).NewField(parentMapFieldDescriptor.MapKey()).MapKey(),
					exampleValue,
				)
			default:
				return syserror.Newf("expected key or value as synthetic field name for map entry's field name, got %q", fieldDescriptor.Name())
			}
			messageToValidate.Set(parentMapFieldDescriptor, protoreflect.ValueOfMap(mapEntry))
		case fieldDescriptor.Enum() != nil:
			// We need to handle enum examples in a special way, since enum examples are set as
			// int32, but we need to set it to the enum value to the field.
			// So we cast exampleValue to an int32 and check that cast first before attempting
			// to set it to the message field. This is because messageToValidate.Set will panic
			// if an invalid type is attempted to be set.
			exampleInt32, ok := exampleValue.Interface().(int32)
			if !ok {
				return syserror.Newf("expected enum example value to be int32 for field %q, got %T type instead", fieldDescriptor.FullName(), exampleValue)
			}
			messageToValidate.Set(fieldDescriptor, protoreflect.ValueOf(protoreflect.EnumNumber(exampleInt32)))
		case fieldDescriptor.Message() != nil:
			// We need to handle the case where the field is a wrapper type. We set the value directly base on the wrapper type.
			switch string(fieldDescriptor.Message().FullName()) {
			case string((&wrapperspb.FloatValue{}).ProtoReflect().Descriptor().FullName()):
				exampleValueFloat, ok := exampleValue.Interface().(float32)
				if !ok {
					return syserror.Newf("unexpected type found for float wrapper type %T", exampleValue.Interface())
				}
				messageToValidate.Set(
					fieldDescriptor,
					protoreflect.ValueOf(
						(&wrapperspb.FloatValue{Value: exampleValueFloat}).ProtoReflect(),
					),
				)
			case string((&wrapperspb.DoubleValue{}).ProtoReflect().Descriptor().FullName()):
				exampleValueDouble, ok := exampleValue.Interface().(float64)
				if !ok {
					return syserror.Newf("unexpected type found for double wrapper type %T", exampleValue.Interface())
				}
				messageToValidate.Set(
					fieldDescriptor,
					protoreflect.ValueOf(
						(&wrapperspb.DoubleValue{Value: exampleValueDouble}).ProtoReflect(),
					),
				)
			case string((&wrapperspb.Int32Value{}).ProtoReflect().Descriptor().FullName()):
				exampleValueInt32, ok := exampleValue.Interface().(int32)
				if !ok {
					return syserror.Newf("unexpected type found for int32 wrapper type %T", exampleValue.Interface())
				}
				messageToValidate.Set(
					fieldDescriptor,
					protoreflect.ValueOf(
						(&wrapperspb.Int32Value{Value: exampleValueInt32}).ProtoReflect(),
					),
				)
			case string((&wrapperspb.Int64Value{}).ProtoReflect().Descriptor().FullName()):
				exampleValueInt64, ok := exampleValue.Interface().(int64)
				if !ok {
					return syserror.Newf("unexpected type found for int64 wrapper type %T", exampleValue.Interface())
				}
				messageToValidate.Set(
					fieldDescriptor,
					protoreflect.ValueOf(
						(&wrapperspb.Int64Value{Value: exampleValueInt64}).ProtoReflect(),
					),
				)
			case string((&wrapperspb.UInt32Value{}).ProtoReflect().Descriptor().FullName()):
				exampleValueUInt32, ok := exampleValue.Interface().(uint32)
				if !ok {
					return syserror.Newf("unexpected type found for uint32 wrapper type %T", exampleValue.Interface())
				}
				messageToValidate.Set(
					fieldDescriptor,
					protoreflect.ValueOf(
						(&wrapperspb.UInt32Value{Value: exampleValueUInt32}).ProtoReflect(),
					),
				)
			case string((&wrapperspb.UInt64Value{}).ProtoReflect().Descriptor().FullName()):
				exampleValueUInt64, ok := exampleValue.Interface().(uint64)
				if !ok {
					return syserror.Newf("unexpected type found for uint32 wrapper type %T", exampleValue.Interface())
				}
				messageToValidate.Set(
					fieldDescriptor,
					protoreflect.ValueOf(
						(&wrapperspb.UInt64Value{Value: exampleValueUInt64}).ProtoReflect(),
					),
				)
			case string((&wrapperspb.BoolValue{}).ProtoReflect().Descriptor().FullName()):
				exampleValueBool, ok := exampleValue.Interface().(bool)
				if !ok {
					return syserror.Newf("unexpected type found for bool wrapper type %T", exampleValue.Interface())
				}
				messageToValidate.Set(
					fieldDescriptor,
					protoreflect.ValueOf(
						(&wrapperspb.BoolValue{Value: exampleValueBool}).ProtoReflect(),
					),
				)
			case string((&wrapperspb.StringValue{}).ProtoReflect().Descriptor().FullName()):
				exampleValueString, ok := exampleValue.Interface().(string)
				if !ok {
					return syserror.Newf("unexpected type found for string wrapper type %T", exampleValue.Interface())
				}
				messageToValidate.Set(
					fieldDescriptor,
					protoreflect.ValueOf(
						(&wrapperspb.StringValue{Value: exampleValueString}).ProtoReflect(),
					),
				)
			case string((&wrapperspb.BytesValue{}).ProtoReflect().Descriptor().FullName()):
				exampleValueBytes, ok := exampleValue.Interface().([]byte)
				if !ok {
					return syserror.Newf("unexpected type found for bytes wrapper type %T", exampleValue.Interface())
				}
				messageToValidate.Set(
					fieldDescriptor,
					protoreflect.ValueOf(
						(&wrapperspb.BytesValue{Value: exampleValueBytes}).ProtoReflect(),
					),
				)
			default:
				// In the case where it is not a wrapper type (e.g. google.protobuf.Timestamp), we just set the example
				// value directly.
				messageToValidate.Set(fieldDescriptor, exampleValue)
			}
		default:
			messageToValidate.Set(fieldDescriptor, exampleValue)
		}
		err := validator.Validate(messageToValidate,
			protovalidate.WithFilter(
				protovalidate.FilterFunc(func(
					_ protoreflect.Message,
					d protoreflect.Descriptor,
				) bool {
					return d == filterFieldDescriptor
				}),
			),
		)
		if err == nil {
			continue
		}
		var compilationErr *protovalidate.CompilationError
		if errors.As(err, &compilationErr) {
			// Expression failing to compile meaning some CEL expressions are invalid,
			// which is checked in this rule (PROTOVALIDATE), but not by this function.
			break
		}
		validationErr := &protovalidate.ValidationError{}
		if errors.As(err, &validationErr) {
			for _, violation := range validationErr.Violations {
				if violationFilterFunc(violation.Proto) {
					adder.addForPathf(append(pathToExampleValues, int32(exampleValueIndex)), `"%v" is an example value but does not satisfy rule %q: %s`, exampleValue.Interface(), violation.Proto.GetRuleId(), violation.Proto.GetMessage())
				}
			}
			continue
		}
		return fmt.Errorf("unexpected error from protovalidate: %s", err.Error())
	}
	return nil
}

func checkConst(adder *adder, rule proto.Message, ruleFieldNumber int32) {
	var (
		fieldCount       int
		constFieldNumber int32
		isConstSpecified bool
	)
	ruleMessage := rule.ProtoReflect()
	ruleMessage.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		fieldCount++
		switch string(fd.Name()) {
		case "const":
			isConstSpecified = true
			constFieldNumber = int32(fd.Number())
		}
		return true
	})
	if isConstSpecified && fieldCount > 1 {
		adder.addForPathf(
			[]int32{ruleFieldNumber, constFieldNumber},
			"Field %q has %s, therefore other rules in %s are not applied and should be removed.",
			adder.fieldName(),
			adder.getFieldRuleName(ruleFieldNumber, constFieldNumber),
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
			[]int32{ruleFieldNumber, minLenFieldNumber},
			"Field %q has %s and therefore, %s is redundant and should be removed.",
			adder.fieldName(),
			adder.getFieldRuleName(ruleFieldNumber, lengthFieldNumber),
			adder.getFieldRuleName(ruleFieldNumber, minLenFieldNumber),
		)
	}
	if length != nil && maxLen != nil {
		adder.addForPathf(
			[]int32{ruleFieldNumber, maxLenFieldNumber},
			"Field %q has %s and therefore, %s is redundant and should be removed.",
			adder.fieldName(),
			adder.getFieldRuleName(ruleFieldNumber, lengthFieldNumber),
			adder.getFieldRuleName(ruleFieldNumber, maxLenFieldNumber),
		)
	}
	if maxLen == nil || minLen == nil {
		return nil
	}
	if *minLen > *maxLen {
		adder.addForPathf(
			[]int32{ruleFieldNumber, minLenFieldNumber},
			"Field %q has value %d for %s, which must be lower than value %d for %s.",
			adder.fieldName(),
			*minLen,
			adder.getFieldRuleName(ruleFieldNumber, minLenFieldNumber),
			*maxLen,
			adder.getFieldRuleName(ruleFieldNumber, maxLenFieldNumber),
		)
		adder.addForPathf(
			[]int32{ruleFieldNumber, maxLenFieldNumber},
			"Field %q has value %d for %s, which must be higher than value %d for %s.",
			adder.fieldName(),
			*maxLen,
			adder.getFieldRuleName(ruleFieldNumber, maxLenFieldNumber),
			*minLen,
			adder.getFieldRuleName(ruleFieldNumber, minLenFieldNumber),
		)
	} else if *minLen == *maxLen {
		adder.addForPathsf(
			[][]int32{
				{ruleFieldNumber, minLenFieldNumber},
				{ruleFieldNumber, maxLenFieldNumber},
			},
			"Field %q has equal %s and %s, use %s.const instead.",
			adder.fieldName(),
			adder.getFieldRuleName(ruleFieldNumber, minLenFieldNumber),
			maxLenFieldName,
			adder.getFieldRuleName(ruleFieldNumber),
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
