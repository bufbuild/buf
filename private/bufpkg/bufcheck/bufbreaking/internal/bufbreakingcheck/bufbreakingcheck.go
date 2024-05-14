// Copyright 2020-2024 Buf Technologies, Inc.
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

// Package bufbreakingcheck impelements the check functions.
//
// These are used by bufbreakingbuild to create RuleBuilders.
package bufbreakingcheck

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufprotosource"
	"github.com/bufbuild/buf/private/gen/proto/go/google/protobuf"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/protocompile/protoutil"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

const (
	featuresFieldName = "features"

	featureNameUTF8Validation     = "utf8_validation"
	featureNameJSONFormat         = "json_format"
	cppFeatureNameStringType      = "string_type"
	javaFeatureNameUTF8Validation = "utf8_validation"
)

// CheckEnumNoDelete is a check function.
var CheckEnumNoDelete = newFilePairCheckFunc(checkEnumNoDelete)

func checkEnumNoDelete(add addFunc, corpus *corpus, previousFile bufprotosource.File, file bufprotosource.File) error {
	previousNestedNameToEnum, err := bufprotosource.NestedNameToEnum(previousFile)
	if err != nil {
		return err
	}
	nestedNameToEnum, err := bufprotosource.NestedNameToEnum(file)
	if err != nil {
		return err
	}
	for previousNestedName := range previousNestedNameToEnum {
		if _, ok := nestedNameToEnum[previousNestedName]; !ok {
			// TODO: search for enum in other files and return that the enum was moved?
			descriptor, location, err := getDescriptorAndLocationForDeletedElement(file, previousNestedName)
			if err != nil {
				return err
			}
			add(descriptor, nil, location, `Previously present enum %q was deleted from file.`, previousNestedName)
		}
	}
	return nil
}

// CheckEnumSameJSONFormat is a check function.
var CheckEnumSameJSONFormat = newEnumPairCheckFunc(checkEnumSameJSONFormat)

func checkEnumSameJSONFormat(
	add addFunc,
	_ *corpus,
	previousEnum bufprotosource.Enum,
	enum bufprotosource.Enum,
) error {
	previousDescriptor, err := previousEnum.AsDescriptor()
	if err != nil {
		return err
	}
	descriptor, err := enum.AsDescriptor()
	if err != nil {
		return err
	}
	featureField, err := findFeatureField(featureNameJSONFormat, protoreflect.EnumKind)
	if err != nil {
		return err
	}
	val, err := protoutil.ResolveFeature(previousDescriptor, featureField)
	if err != nil {
		return fmt.Errorf("unable to resolve value of %s feature: %w", featureField.Name(), err)
	}
	previousJSONFormat := descriptorpb.FeatureSet_JsonFormat(val.Enum())
	val, err = protoutil.ResolveFeature(descriptor, featureField)
	if err != nil {
		return fmt.Errorf("unable to resolve value of %s feature: %w", featureField.Name(), err)
	}
	jsonFormat := descriptorpb.FeatureSet_JsonFormat(val.Enum())
	if previousJSONFormat == descriptorpb.FeatureSet_ALLOW && jsonFormat != descriptorpb.FeatureSet_ALLOW {
		add(
			enum,
			nil,
			withBackupLocation(enum.Features().JSONFormatLocation(), enum.Location()),
			`Enum %q JSON format support changed from %v to %v.`,
			enum.Name(),
			previousJSONFormat,
			jsonFormat,
		)
	}
	return nil
}

// CheckEnumSameType is a check function.
var CheckEnumSameType = newEnumPairCheckFunc(checkEnumSameType)

func checkEnumSameType(add addFunc, _ *corpus, previousEnum bufprotosource.Enum, enum bufprotosource.Enum) error {
	previousDescriptor, err := previousEnum.AsDescriptor()
	if err != nil {
		return err
	}
	descriptor, err := enum.AsDescriptor()
	if err != nil {
		return err
	}
	if previousDescriptor.IsClosed() != descriptor.IsClosed() {
		previousState, currentState := "closed", "open"
		if descriptor.IsClosed() {
			previousState, currentState = currentState, previousState
		}
		add(
			enum,
			nil,
			withBackupLocation(enum.Features().EnumTypeLocation(), enum.Location()),
			`Enum %q changed from %s to %s.`,
			enum.Name(),
			previousState,
			currentState,
		)
	}
	return nil
}

// CheckEnumValueNoDelete is a check function.
var CheckEnumValueNoDelete = newEnumPairCheckFunc(checkEnumValueNoDelete)

func checkEnumValueNoDelete(add addFunc, corpus *corpus, previousEnum bufprotosource.Enum, enum bufprotosource.Enum) error {
	return checkEnumValueNoDeleteWithRules(add, previousEnum, enum, false, false)
}

// CheckEnumValueNoDeleteUnlessNumberReserved is a check function.
var CheckEnumValueNoDeleteUnlessNumberReserved = newEnumPairCheckFunc(checkEnumValueNoDeleteUnlessNumberReserved)

func checkEnumValueNoDeleteUnlessNumberReserved(add addFunc, corpus *corpus, previousEnum bufprotosource.Enum, enum bufprotosource.Enum) error {
	return checkEnumValueNoDeleteWithRules(add, previousEnum, enum, true, false)
}

// CheckEnumValueNoDeleteUnlessNameReserved is a check function.
var CheckEnumValueNoDeleteUnlessNameReserved = newEnumPairCheckFunc(checkEnumValueNoDeleteUnlessNameReserved)

func checkEnumValueNoDeleteUnlessNameReserved(add addFunc, corpus *corpus, previousEnum bufprotosource.Enum, enum bufprotosource.Enum) error {
	return checkEnumValueNoDeleteWithRules(add, previousEnum, enum, false, true)
}

func checkEnumValueNoDeleteWithRules(add addFunc, previousEnum bufprotosource.Enum, enum bufprotosource.Enum, allowIfNumberReserved bool, allowIfNameReserved bool) error {
	previousNumberToNameToEnumValue, err := bufprotosource.NumberToNameToEnumValue(previousEnum)
	if err != nil {
		return err
	}
	numberToNameToEnumValue, err := bufprotosource.NumberToNameToEnumValue(enum)
	if err != nil {
		return err
	}
	for previousNumber, previousNameToEnumValue := range previousNumberToNameToEnumValue {
		if _, ok := numberToNameToEnumValue[previousNumber]; !ok {
			if !isDeletedEnumValueAllowedWithRules(previousNumber, previousNameToEnumValue, enum, allowIfNumberReserved, allowIfNameReserved) {
				suffix := ""
				if allowIfNumberReserved && allowIfNameReserved {
					return errors.New("both allowIfNumberReserved and allowIfNameReserved set")
				}
				if allowIfNumberReserved {
					suffix = fmt.Sprintf(` without reserving the number "%d"`, previousNumber)
				}
				if allowIfNameReserved {
					nameSuffix := ""
					if len(previousNameToEnumValue) > 1 {
						nameSuffix = "s"
					}
					suffix = fmt.Sprintf(` without reserving the name%s %s`, nameSuffix, stringutil.JoinSliceQuoted(getSortedEnumValueNames(previousNameToEnumValue), ", "))
				}
				add(enum, nil, enum.Location(), `Previously present enum value "%d" on enum %q was deleted%s.`, previousNumber, enum.Name(), suffix)
			}
		}
	}
	return nil
}

func isDeletedEnumValueAllowedWithRules(previousNumber int, previousNameToEnumValue map[string]bufprotosource.EnumValue, enum bufprotosource.Enum, allowIfNumberReserved bool, allowIfNameReserved bool) bool {
	if allowIfNumberReserved {
		return bufprotosource.NumberInReservedRanges(previousNumber, enum.ReservedTagRanges()...)
	}
	if allowIfNameReserved {
		// if true for all names, then ok
		for previousName := range previousNameToEnumValue {
			if !bufprotosource.NameInReservedNames(previousName, enum.ReservedNames()...) {
				return false
			}
		}
		return true
	}
	return false
}

// CheckEnumValueSameName is a check function.
var CheckEnumValueSameName = newEnumValuePairCheckFunc(checkEnumValueSameName)

func checkEnumValueSameName(add addFunc, corpus *corpus, previousNameToEnumValue map[string]bufprotosource.EnumValue, nameToEnumValue map[string]bufprotosource.EnumValue) error {
	previousNames := getSortedEnumValueNames(previousNameToEnumValue)
	names := getSortedEnumValueNames(nameToEnumValue)
	// all current names for this number need to be in the previous set
	// ie if you now have FOO=2, BAR=2, you need to have had FOO=2, BAR=2 previously
	// FOO=2, BAR=2, BAZ=2 now would pass
	// FOO=2, BAR=2, BAZ=2 previously would fail
	if !slicesext.ElementsContained(names, previousNames) {
		previousNamesString := stringutil.JoinSliceQuoted(previousNames, ", ")
		namesString := stringutil.JoinSliceQuoted(names, ", ")
		nameSuffix := ""
		if len(previousNames) > 1 && len(names) > 1 {
			nameSuffix = "s"
		}
		for _, enumValue := range nameToEnumValue {
			add(enumValue, nil, enumValue.NumberLocation(), `Enum value "%d" on enum %q changed name%s from %s to %s.`, enumValue.Number(), enumValue.Enum().Name(), nameSuffix, previousNamesString, namesString)
		}
	}
	return nil
}

// CheckExtensionMessageNoDelete is a check function.
var CheckExtensionMessageNoDelete = newMessagePairCheckFunc(checkExtensionMessageNoDelete)

func checkExtensionMessageNoDelete(add addFunc, corpus *corpus, previousMessage bufprotosource.Message, message bufprotosource.Message) error {
	return checkTagRanges(add, "extension", message, previousMessage.ExtensionRanges(), message.ExtensionRanges())
}

// CheckExtensionNoDelete is a check function.
var CheckExtensionNoDelete = newFilePairCheckFunc(checkExtensionNoDelete)

func checkExtensionNoDelete(add addFunc, corpus *corpus, previousFile bufprotosource.File, file bufprotosource.File) error {
	previousNestedNameToExtension, err := bufprotosource.NestedNameToExtension(previousFile)
	if err != nil {
		return err
	}
	nestedNameToExtension, err := bufprotosource.NestedNameToExtension(file)
	if err != nil {
		return err
	}
	for previousNestedName := range previousNestedNameToExtension {
		if _, ok := nestedNameToExtension[previousNestedName]; !ok {
			descriptor, location, err := getDescriptorAndLocationForDeletedElement(file, previousNestedName)
			if err != nil {
				return err
			}
			add(descriptor, nil, location, `Previously present extension %q was deleted from file.`, previousNestedName)
		}
	}
	return nil
}

// CheckFieldNoDelete is a check function.
var CheckFieldNoDelete = newMessagePairCheckFunc(checkFieldNoDelete)

func checkFieldNoDelete(add addFunc, corpus *corpus, previousMessage bufprotosource.Message, message bufprotosource.Message) error {
	return checkFieldNoDeleteWithRules(add, previousMessage, message, false, false)
}

// CheckFieldNoDeleteUnlessNumberReserved is a check function.
var CheckFieldNoDeleteUnlessNumberReserved = newMessagePairCheckFunc(checkFieldNoDeleteUnlessNumberReserved)

func checkFieldNoDeleteUnlessNumberReserved(add addFunc, corpus *corpus, previousMessage bufprotosource.Message, message bufprotosource.Message) error {
	return checkFieldNoDeleteWithRules(add, previousMessage, message, true, false)
}

// CheckFieldNoDeleteUnlessNameReserved is a check function.
var CheckFieldNoDeleteUnlessNameReserved = newMessagePairCheckFunc(checkFieldNoDeleteUnlessNameReserved)

func checkFieldNoDeleteUnlessNameReserved(add addFunc, corpus *corpus, previousMessage bufprotosource.Message, message bufprotosource.Message) error {
	return checkFieldNoDeleteWithRules(add, previousMessage, message, false, true)
}

func checkFieldNoDeleteWithRules(add addFunc, previousMessage bufprotosource.Message, message bufprotosource.Message, allowIfNumberReserved bool, allowIfNameReserved bool) error {
	previousNumberToField, err := bufprotosource.NumberToMessageField(previousMessage)
	if err != nil {
		return err
	}
	numberToField, err := bufprotosource.NumberToMessageField(message)
	if err != nil {
		return err
	}
	for previousNumber, previousField := range previousNumberToField {
		if _, ok := numberToField[previousNumber]; !ok {
			if !isDeletedFieldAllowedWithRules(previousField, message, allowIfNumberReserved, allowIfNameReserved) {
				suffix := ""
				if allowIfNumberReserved && allowIfNameReserved {
					return errors.New("both allowIfNumberReserved and allowIfNameReserved set")
				}
				if allowIfNumberReserved {
					suffix = fmt.Sprintf(` without reserving the number "%d"`, previousField.Number())
				}
				if allowIfNameReserved {
					suffix = fmt.Sprintf(` without reserving the name %q`, previousField.Name())
				}
				description := fieldDescription(previousField)
				// Description will start with capital letter; lower-case it
				// to better fit in this message.
				description = strings.ToLower(description[:1]) + description[1:]
				add(
					message,
					nil,
					message.Location(),
					`Previously present %s was deleted%s.`,
					description,
					suffix)
			}
		}
	}
	return nil
}

func isDeletedFieldAllowedWithRules(previousField bufprotosource.Field, message bufprotosource.Message, allowIfNumberReserved bool, allowIfNameReserved bool) bool {
	return (allowIfNumberReserved && bufprotosource.NumberInReservedRanges(previousField.Number(), message.ReservedTagRanges()...)) ||
		(allowIfNameReserved && bufprotosource.NameInReservedNames(previousField.Name(), message.ReservedNames()...))
}

// CheckFieldSameCardinality is a check function.
var CheckFieldSameCardinality = newFieldDescriptorPairCheckFunc(checkFieldSameCardinality)

func checkFieldSameCardinality(
	add addFunc,
	_ *corpus,
	_ bufprotosource.Field,
	previousDescriptor protoreflect.FieldDescriptor,
	field bufprotosource.Field,
	descriptor protoreflect.FieldDescriptor,
) error {
	if previousDescriptor.ContainingMessage().IsMapEntry() && descriptor.ContainingMessage().IsMapEntry() {
		// Map entries are generated so nothing to do here. They
		// usually would be safe to check anyway, but it's possible
		// that a map entry field "appears" to inherit field presence
		// from a file default or file syntax, but they don't actually
		// behave differently whether they report implicit vs explicit
		// presence. So just skip the check.
		return nil
	}

	previousCardinality := getCardinality(previousDescriptor)
	currentCardinality := getCardinality(descriptor)
	if previousCardinality != currentCardinality {
		add(field, nil, field.Location(),
			`%s changed cardinality from %q to %q.`,
			fieldDescription(field),
			previousCardinality,
			currentCardinality,
		)
	}
	return nil
}

// CheckFieldSameCppStringType is a check function.
var CheckFieldSameCppStringType = newFieldDescriptorPairCheckFunc(checkFieldSameCppStringType)

func checkFieldSameCppStringType(
	add addFunc,
	corpus *corpus,
	previousField bufprotosource.Field,
	previousDescriptor protoreflect.FieldDescriptor,
	field bufprotosource.Field,
	descriptor protoreflect.FieldDescriptor,
) error {
	if previousDescriptor.ContainingMessage().IsMapEntry() && descriptor.ContainingMessage().IsMapEntry() {
		// Map entries, even with string or bytes keys or values,
		// don't allow configuring the string type.
		return nil
	}
	if (previousDescriptor.Kind() != protoreflect.StringKind && previousDescriptor.Kind() != protoreflect.BytesKind) ||
		(descriptor.Kind() != protoreflect.StringKind && descriptor.Kind() != protoreflect.BytesKind) {
		// this check only applies to string/bytes fields
		return nil
	}
	previousStringType, previousIsStringPiece, err := fieldCppStringType(previousField, previousDescriptor)
	if err != nil {
		return err
	}
	stringType, isStringPiece, err := fieldCppStringType(field, descriptor)
	if err != nil {
		return err
	}
	if (previousStringType != stringType || previousIsStringPiece != isStringPiece) &&
		// it is NOT breaking to move from string_piece -> string
		!(previousIsStringPiece && stringType == protobuf.CppFeatures_STRING) {
		var previousType, currentType fmt.Stringer
		if previousIsStringPiece {
			previousType = descriptorpb.FieldOptions_STRING_PIECE
		} else {
			previousType = previousStringType
		}
		if isStringPiece {
			currentType = descriptorpb.FieldOptions_STRING_PIECE
		} else {
			currentType = stringType
		}
		add(
			field,
			nil,
			withBackupLocation(field.CTypeLocation(), fieldCppStringTypeLocation(field), field.Location()),
			`%s changed C++ string type from %q to %q.`,
			fieldDescription(field),
			previousType,
			currentType,
		)
	}
	return nil
}

// CheckFieldSameJavaUTF8Validation is a check function.
var CheckFieldSameJavaUTF8Validation = newFieldDescriptorPairCheckFunc(checkFieldSameJavaUTF8Validation)

func checkFieldSameJavaUTF8Validation(
	add addFunc,
	corpus *corpus,
	previousField bufprotosource.Field,
	previousDescriptor protoreflect.FieldDescriptor,
	field bufprotosource.Field,
	descriptor protoreflect.FieldDescriptor,
) error {
	if previousDescriptor.Kind() != protoreflect.StringKind || descriptor.Kind() != protoreflect.StringKind {
		// this check only applies to string fields
		return nil
	}
	previousValidation, err := fieldJavaUTF8Validation(previousDescriptor)
	if err != nil {
		return err
	}
	validation, err := fieldJavaUTF8Validation(descriptor)
	if err != nil {
		return err
	}
	if previousValidation != validation {
		add(
			field,
			nil,
			withBackupLocation(field.File().JavaStringCheckUtf8Location(), fieldJavaUTF8ValidationLocation(field), field.Location()),
			`%s changed Java string UTF8 validation from %q to %q.`,
			fieldDescription(field),
			previousValidation,
			validation,
		)
	}
	return nil
}

// CheckFieldSameDefault is a check function.
var CheckFieldSameDefault = newFieldDescriptorPairCheckFunc(checkFieldSameDefault)

func checkFieldSameDefault(
	add addFunc,
	corpus *corpus,
	previousField bufprotosource.Field,
	previousDescriptor protoreflect.FieldDescriptor,
	field bufprotosource.Field,
	descriptor protoreflect.FieldDescriptor,
) error {
	if !canHaveDefault(previousDescriptor) || !canHaveDefault(descriptor) {
		return nil
	}
	previousDefault := getDefault(previousDescriptor)
	currentDefault := getDefault(descriptor)
	if previousDefault.isZero() && currentDefault.isZero() {
		// no defaults to check
		return nil
	}
	if !defaultsEqual(previousDefault, currentDefault) {
		add(
			field,
			nil,
			withBackupLocation(field.DefaultLocation(), field.Location()),
			`% changed default value from %v to %v.`,
			fieldDescription(field),
			previousDefault.printable,
			currentDefault.printable,
		)
	}
	return nil
}

// CheckFieldSameJSONName is a check function.
var CheckFieldSameJSONName = newFieldPairCheckFunc(checkFieldSameJSONName)

func checkFieldSameJSONName(add addFunc, corpus *corpus, previousField bufprotosource.Field, field bufprotosource.Field) error {
	if previousField.Extendee() != "" {
		// JSON name can't be set explicitly for extensions
		return nil
	}
	if previousField.JSONName() != field.JSONName() {
		add(field, nil, withBackupLocation(field.JSONNameLocation(), field.Location()),
			`%s changed option "json_name" from %q to %q.`,
			fieldDescription(field),
			previousField.JSONName(), field.JSONName())
	}
	return nil
}

// CheckFieldSameJSType is a check function.
var CheckFieldSameJSType = newFieldPairCheckFunc(checkFieldSameJSType)

func checkFieldSameJSType(add addFunc, corpus *corpus, previousField bufprotosource.Field, field bufprotosource.Field) error {
	if !is64bitInteger(previousField.Type()) || !is64bitInteger(field.Type()) {
		// this check only applies to 64-bit integer fields
		return nil
	}
	if previousField.JSType() != field.JSType() {
		add(field, nil, withBackupLocation(field.JSTypeLocation(), field.Location()),
			`%s changed option "jstype" from %q to %q.`,
			fieldDescription(field),
			previousField.JSType().String(), field.JSType().String())
	}
	return nil
}

// CheckFieldSameName is a check function.
var CheckFieldSameName = newFieldPairCheckFunc(checkFieldSameName)

func checkFieldSameName(add addFunc, corpus *corpus, previousField bufprotosource.Field, field bufprotosource.Field) error {
	var previousName, name string
	if previousField.Extendee() != "" {
		previousName = previousField.FullName()
		name = field.FullName()
	} else {
		previousName = previousField.Name()
		name = field.Name()
	}
	if previousName != name {
		add(field, nil, field.NameLocation(),
			`%s changed name from %q to %q.`,
			fieldDescriptionWithName(field, ""), // don't include name in description
			previousName, name)
	}
	return nil
}

// CheckFieldSameOneof is a check function.
var CheckFieldSameOneof = newFieldPairCheckFunc(checkFieldSameOneof)

func checkFieldSameOneof(add addFunc, corpus *corpus, previousField bufprotosource.Field, field bufprotosource.Field) error {
	if previousField.Extendee() != "" {
		// extensions can't be defined inside oneofs
		return nil
	}
	previousOneof := previousField.Oneof()
	if previousOneof != nil {
		previousOneofDescriptor, err := previousOneof.AsDescriptor()
		if err != nil {
			return err
		}
		if previousOneofDescriptor.IsSynthetic() {
			// Not considering synthetic oneofs since those are really
			// just strange byproducts of how "explicit presence" is
			// modeled in proto3 syntax. We will separately detect this
			// kind of change via field presence check.
			previousOneof = nil
		}
	}
	oneof := field.Oneof()
	if oneof != nil {
		oneofDescriptor, err := oneof.AsDescriptor()
		if err != nil {
			return err
		}
		if oneofDescriptor.IsSynthetic() {
			// Same remark as above.
			oneof = nil
		}
	}

	previousInsideOneof := previousOneof != nil
	insideOneof := oneof != nil
	if !previousInsideOneof && !insideOneof {
		return nil
	}
	if previousInsideOneof && insideOneof {
		if previousOneof.Name() != oneof.Name() {
			add(field, nil, field.Location(),
				`%sq moved from oneof %q to oneof %q.`,
				fieldDescription(field),
				previousOneof.Name(), oneof.Name())
		}
		return nil
	}

	previous := "inside"
	current := "outside"
	if insideOneof {
		previous = "outside"
		current = "inside"
	}
	add(field, nil, field.Location(),
		`%s moved from %s to %s a oneof.`,
		fieldDescription(field),
		previous, current)
	return nil
}

// CheckFieldSameType is a check function.
var CheckFieldSameType = newFieldDescriptorPairCheckFunc(checkFieldSameType)

func checkFieldSameType(
	add addFunc,
	_ *corpus,
	previousField bufprotosource.Field,
	previousDescriptor protoreflect.FieldDescriptor,
	field bufprotosource.Field,
	descriptor protoreflect.FieldDescriptor,
) error {
	// We use descriptor.Kind(), instead of field.Type(), because it also includes
	// a check of resolved features in Editions files so it can distinguish between
	// normal (length-prefixed) and delimited (aka "group" encoded) messages, which
	// are not compatible.
	if previousDescriptor.Kind() != descriptor.Kind() {
		addFieldChangedType(add, previousField, previousDescriptor, field, descriptor)
		return nil
	}

	switch field.Type() {
	case descriptorpb.FieldDescriptorProto_TYPE_ENUM,
		descriptorpb.FieldDescriptorProto_TYPE_GROUP,
		descriptorpb.FieldDescriptorProto_TYPE_MESSAGE:
		if previousField.TypeName() != field.TypeName() {
			addEnumGroupMessageFieldChangedTypeName(add, previousField, field)
		}
	}
	return nil
}

// CheckFieldSameUTF8Validation is a check function.
var CheckFieldSameUTF8Validation = newFieldDescriptorPairCheckFunc(checkFieldSameUTF8Validation)

func checkFieldSameUTF8Validation(
	add addFunc,
	_ *corpus,
	_ bufprotosource.Field,
	previousDescriptor protoreflect.FieldDescriptor,
	field bufprotosource.Field,
	descriptor protoreflect.FieldDescriptor,
) error {
	if previousDescriptor.Kind() != protoreflect.StringKind || descriptor.Kind() != protoreflect.StringKind {
		return nil
	}
	featureField, err := findFeatureField(featureNameUTF8Validation, protoreflect.EnumKind)
	if err != nil {
		return err
	}
	val, err := protoutil.ResolveFeature(previousDescriptor, featureField)
	if err != nil {
		return fmt.Errorf("unable to resolve value of %s feature: %w", featureField.Name(), err)
	}
	previousUTF8Validation := descriptorpb.FeatureSet_Utf8Validation(val.Enum())
	val, err = protoutil.ResolveFeature(descriptor, featureField)
	if err != nil {
		return fmt.Errorf("unable to resolve value of %s feature: %w", featureField.Name(), err)
	}
	utf8Validation := descriptorpb.FeatureSet_Utf8Validation(val.Enum())
	if previousUTF8Validation != utf8Validation {
		add(
			field,
			nil,
			withBackupLocation(field.Features().UTF8ValidationLocation(), field.Location()),
			`%s changed UTF8 validation from %v to %v.`,
			fieldDescription(field),
			previousUTF8Validation,
			utf8Validation,
		)
	}
	return nil
}

// CheckFieldWireCompatibleCardinality is a check function.
var CheckFieldWireCompatibleCardinality = newFieldDescriptorPairCheckFunc(checkFieldWireCompatibleCardinality)

func checkFieldWireCompatibleCardinality(
	add addFunc,
	_ *corpus,
	_ bufprotosource.Field,
	previousDescriptor protoreflect.FieldDescriptor,
	field bufprotosource.Field,
	descriptor protoreflect.FieldDescriptor,
) error {
	if previousDescriptor.ContainingMessage().IsMapEntry() && descriptor.ContainingMessage().IsMapEntry() {
		// Map entries are generated so nothing to do here. They
		// usually would be safe to check anyway, but it's possible
		// that a map entry field "appears" to inherit field presence
		// from a file default or file syntax, but they don't actually
		// behave differently whether they report implicit vs explicit
		// presence. So just skip the check.
		return nil
	}

	previousCardinality := getCardinality(previousDescriptor)
	currentCardinality := getCardinality(descriptor)
	if cardinalityToWireCompatiblityGroup[previousCardinality] != cardinalityToWireCompatiblityGroup[currentCardinality] {
		add(field, nil, field.Location(),
			`%s changed cardinality from %q to %q.`,
			fieldDescription(field),
			previousCardinality,
			currentCardinality,
		)
	}
	return nil
}

// CheckFieldWireCompatibleType is a check function.
var CheckFieldWireCompatibleType = newFieldDescriptorPairCheckFunc(checkFieldWireCompatibleType)

func checkFieldWireCompatibleType(
	add addFunc,
	corpus *corpus,
	previousField bufprotosource.Field,
	previousDescriptor protoreflect.FieldDescriptor,
	field bufprotosource.Field,
	descriptor protoreflect.FieldDescriptor,
) error {
	// We use descriptor.Kind(), instead of field.Type(), because it also includes
	// a check of resolved features in Editions files so it can distinguish between
	// normal (length-prefixed) and delimited (aka "group" encoded) messages, which
	// are not compatible.
	previousWireCompatibilityGroup, ok := fieldKindToWireCompatiblityGroup[previousDescriptor.Kind()]
	if !ok {
		return fmt.Errorf("unknown FieldDescriptorProtoType: %v", previousDescriptor.Kind())
	}
	wireCompatibilityGroup, ok := fieldKindToWireCompatiblityGroup[descriptor.Kind()]
	if !ok {
		return fmt.Errorf("unknown FieldDescriptorProtoType: %v", descriptor.Kind())
	}
	if previousWireCompatibilityGroup != wireCompatibilityGroup {
		extraMessages := []string{
			"See https://developers.google.com/protocol-buffers/docs/proto3#updating for wire compatibility rules.",
		}
		switch {
		case previousDescriptor.Kind() == protoreflect.StringKind && descriptor.Kind() == protoreflect.BytesKind:
			// It is OK to evolve from string to bytes
			return nil
		case previousDescriptor.Kind() == protoreflect.BytesKind && descriptor.Kind() == protoreflect.StringKind:
			extraMessages = append(
				extraMessages,
				"Note that while string and bytes are compatible if the data is valid UTF-8, there is no way to enforce that a bytes field is UTF-8, so these fields may be incompatible.",
			)
		}
		addFieldChangedType(add, previousField, previousDescriptor, field, descriptor, extraMessages...)
		return nil
	}
	switch field.Type() {
	case descriptorpb.FieldDescriptorProto_TYPE_ENUM:
		if previousField.TypeName() != field.TypeName() {
			return checkEnumWireCompatibleForField(add, corpus, previousField, field)
		}
	case descriptorpb.FieldDescriptorProto_TYPE_GROUP,
		descriptorpb.FieldDescriptorProto_TYPE_MESSAGE:
		if previousField.TypeName() != field.TypeName() {
			addEnumGroupMessageFieldChangedTypeName(add, previousField, field)
			return nil
		}
	}
	return nil
}

// CheckFieldWireJSONCompatibleCardinality is a check function.
var CheckFieldWireJSONCompatibleCardinality = newFieldDescriptorPairCheckFunc(checkFieldWireJSONCompatibleCardinality)

func checkFieldWireJSONCompatibleCardinality(
	add addFunc,
	_ *corpus,
	_ bufprotosource.Field,
	previousDescriptor protoreflect.FieldDescriptor,
	field bufprotosource.Field,
	descriptor protoreflect.FieldDescriptor,
) error {
	if previousDescriptor.ContainingMessage().IsMapEntry() && descriptor.ContainingMessage().IsMapEntry() {
		// Map entries are generated so nothing to do here. They
		// usually would be safe to check anyway, but it's possible
		// that a map entry field "appears" to inherit field presence
		// from a file default or file syntax, but they don't actually
		// behave differently whether they report implicit vs explicit
		// presence. So just skip the check.
		return nil
	}

	previousCardinality := getCardinality(previousDescriptor)
	currentCardinality := getCardinality(descriptor)
	if cardinalityToWireJSONCompatiblityGroup[previousCardinality] != cardinalityToWireJSONCompatiblityGroup[currentCardinality] {
		add(field, nil, field.Location(),
			`%s changed cardinality from %q to %q.`,
			fieldDescription(field),
			previousCardinality,
			currentCardinality,
		)
	}
	return nil
}

// CheckFieldWireJSONCompatibleType is a check function.
var CheckFieldWireJSONCompatibleType = newFieldDescriptorPairCheckFunc(checkFieldWireJSONCompatibleType)

func checkFieldWireJSONCompatibleType(
	add addFunc,
	corpus *corpus,
	previousField bufprotosource.Field,
	previousDescriptor protoreflect.FieldDescriptor,
	field bufprotosource.Field,
	descriptor protoreflect.FieldDescriptor,
) error {
	// We use descriptor.Kind(), instead of field.Type(), because it also includes
	// a check of resolved features in Editions files so it can distinguish between
	// normal (length-prefixed) and delimited (aka "group" encoded) messages, which
	// are not compatible.
	previousWireJSONCompatibilityGroup, ok := fieldKindToWireJSONCompatiblityGroup[previousDescriptor.Kind()]
	if !ok {
		return fmt.Errorf("unknown FieldDescriptorProtoType: %v", previousDescriptor.Kind())
	}
	wireJSONCompatibilityGroup, ok := fieldKindToWireJSONCompatiblityGroup[descriptor.Kind()]
	if !ok {
		return fmt.Errorf("unknown FieldDescriptorProtoType: %v", descriptor.Kind())
	}
	if previousWireJSONCompatibilityGroup != wireJSONCompatibilityGroup {
		addFieldChangedType(
			add,
			previousField,
			previousDescriptor,
			field,
			descriptor,
			"See https://developers.google.com/protocol-buffers/docs/proto3#updating for wire compatibility rules and https://developers.google.com/protocol-buffers/docs/proto3#json for JSON compatibility rules.",
		)
		return nil
	}
	switch descriptor.Kind() {
	case protoreflect.EnumKind:
		if previousField.TypeName() != field.TypeName() {
			return checkEnumWireCompatibleForField(add, corpus, previousField, field)
		}
	case protoreflect.GroupKind, protoreflect.MessageKind:
		if previousField.TypeName() != field.TypeName() {
			addEnumGroupMessageFieldChangedTypeName(add, previousField, field)
			return nil
		}
	}
	return nil
}

func checkEnumWireCompatibleForField(add addFunc, corpus *corpus, previousField bufprotosource.Field, field bufprotosource.Field) error {
	previousEnum, err := getEnumByFullName(
		corpus.previousFiles,
		strings.TrimPrefix(previousField.TypeName(), "."),
	)
	if err != nil {
		return err
	}
	enum, err := getEnumByFullName(
		corpus.files,
		strings.TrimPrefix(field.TypeName(), "."),
	)
	if err != nil {
		return err
	}
	if previousEnum.Name() != enum.Name() {
		// If the short names are not equal, we say that this is a different enum.
		addEnumGroupMessageFieldChangedTypeName(add, previousField, field)
		return nil
	}
	isSubset, err := bufprotosource.EnumIsSubset(enum, previousEnum)
	if err != nil {
		return err
	}
	if !isSubset {
		// If the previous enum is not a subset of the new enum, we say that
		// this is a different enum.
		// We allow subsets so that enum values can be added within the
		// same change.
		addEnumGroupMessageFieldChangedTypeName(add, previousField, field)
		return nil
	}
	return nil
}

func addFieldChangedType(
	add addFunc,
	previousField bufprotosource.Field,
	previousDescriptor protoreflect.FieldDescriptor,
	field bufprotosource.Field,
	descriptor protoreflect.FieldDescriptor,
	extraMessages ...string,
) {
	combinedExtraMessage := ""
	if len(extraMessages) > 0 {
		// protect against mistakenly added empty extra messages
		if joined := strings.TrimSpace(strings.Join(extraMessages, " ")); joined != "" {
			combinedExtraMessage = " " + joined
		}
	}
	var fieldLocation bufprotosource.Location
	switch descriptor.Kind() {
	case protoreflect.MessageKind, protoreflect.EnumKind, protoreflect.GroupKind:
		fieldLocation = field.TypeNameLocation()
	default:
		fieldLocation = field.TypeLocation()
	}
	add(
		field,
		nil,
		fieldLocation,
		`%s changed type from %q to %q.%s`,
		fieldDescription(field),
		fieldDescriptorTypePrettyString(previousDescriptor),
		fieldDescriptorTypePrettyString(descriptor),
		combinedExtraMessage,
	)
}

func addEnumGroupMessageFieldChangedTypeName(add addFunc, previousField bufprotosource.Field, field bufprotosource.Field) {
	add(
		field,
		nil,
		field.TypeNameLocation(),
		`%s changed type from %q to %q.`,
		fieldDescription(field),
		strings.TrimPrefix(previousField.TypeName(), "."),
		strings.TrimPrefix(field.TypeName(), "."),
	)
}

// CheckFileNoDelete is a check function.
var CheckFileNoDelete = newFilesCheckFunc(checkFileNoDelete)

func checkFileNoDelete(add addFunc, corpus *corpus) error {
	previousFilePathToFile, err := bufprotosource.FilePathToFile(corpus.previousFiles...)
	if err != nil {
		return err
	}
	filePathToFile, err := bufprotosource.FilePathToFile(corpus.files...)
	if err != nil {
		return err
	}
	for previousFilePath, previousFile := range previousFilePathToFile {
		if _, ok := filePathToFile[previousFilePath]; !ok {
			// Add previous descriptor to check for ignores. This will mean that if
			// we have ignore_unstable_packages set, this file will cause the ignore
			// to happen.
			add(nil, []bufprotosource.Descriptor{previousFile}, nil, `Previously present file %q was deleted.`, previousFilePath)
		}
	}
	return nil
}

// CheckFileSameCsharpNamespace is a check function.
var CheckFileSameCsharpNamespace = newFilePairCheckFunc(checkFileSameCsharpNamespace)

func checkFileSameCsharpNamespace(add addFunc, corpus *corpus, previousFile bufprotosource.File, file bufprotosource.File) error {
	return checkFileSameValue(add, previousFile.CsharpNamespace(), file.CsharpNamespace(), file, file.CsharpNamespaceLocation(), `option "csharp_namespace"`)
}

// CheckFileSameGoPackage is a check function.
var CheckFileSameGoPackage = newFilePairCheckFunc(checkFileSameGoPackage)

func checkFileSameGoPackage(add addFunc, corpus *corpus, previousFile bufprotosource.File, file bufprotosource.File) error {
	return checkFileSameValue(add, previousFile.GoPackage(), file.GoPackage(), file, file.GoPackageLocation(), `option "go_package"`)
}

// CheckFileSameJavaMultipleFiles is a check function.
var CheckFileSameJavaMultipleFiles = newFilePairCheckFunc(checkFileSameJavaMultipleFiles)

func checkFileSameJavaMultipleFiles(add addFunc, corpus *corpus, previousFile bufprotosource.File, file bufprotosource.File) error {
	return checkFileSameValue(add, strconv.FormatBool(previousFile.JavaMultipleFiles()), strconv.FormatBool(file.JavaMultipleFiles()), file, file.JavaMultipleFilesLocation(), `option "java_multiple_files"`)
}

// CheckFileSameJavaOuterClassname is a check function.
var CheckFileSameJavaOuterClassname = newFilePairCheckFunc(checkFileSameJavaOuterClassname)

func checkFileSameJavaOuterClassname(add addFunc, corpus *corpus, previousFile bufprotosource.File, file bufprotosource.File) error {
	return checkFileSameValue(add, previousFile.JavaOuterClassname(), file.JavaOuterClassname(), file, file.JavaOuterClassnameLocation(), `option "java_outer_classname"`)
}

// CheckFileSameJavaPackage is a check function.
var CheckFileSameJavaPackage = newFilePairCheckFunc(checkFileSameJavaPackage)

func checkFileSameJavaPackage(add addFunc, corpus *corpus, previousFile bufprotosource.File, file bufprotosource.File) error {
	return checkFileSameValue(add, previousFile.JavaPackage(), file.JavaPackage(), file, file.JavaPackageLocation(), `option "java_package"`)
}

// CheckFileSameObjcClassPrefix is a check function.
var CheckFileSameObjcClassPrefix = newFilePairCheckFunc(checkFileSameObjcClassPrefix)

func checkFileSameObjcClassPrefix(add addFunc, corpus *corpus, previousFile bufprotosource.File, file bufprotosource.File) error {
	return checkFileSameValue(add, previousFile.ObjcClassPrefix(), file.ObjcClassPrefix(), file, file.ObjcClassPrefixLocation(), `option "objc_class_prefix"`)
}

// CheckFileSamePackage is a check function.
var CheckFileSamePackage = newFilePairCheckFunc(checkFileSamePackage)

func checkFileSamePackage(add addFunc, corpus *corpus, previousFile bufprotosource.File, file bufprotosource.File) error {
	return checkFileSameValue(add, previousFile.Package(), file.Package(), file, file.PackageLocation(), `package`)
}

// CheckFileSamePhpClassPrefix is a check function.
var CheckFileSamePhpClassPrefix = newFilePairCheckFunc(checkFileSamePhpClassPrefix)

func checkFileSamePhpClassPrefix(add addFunc, corpus *corpus, previousFile bufprotosource.File, file bufprotosource.File) error {
	return checkFileSameValue(add, previousFile.PhpClassPrefix(), file.PhpClassPrefix(), file, file.PhpClassPrefixLocation(), `option "php_class_prefix"`)
}

// CheckFileSamePhpNamespace is a check function.
var CheckFileSamePhpNamespace = newFilePairCheckFunc(checkFileSamePhpNamespace)

func checkFileSamePhpNamespace(add addFunc, corpus *corpus, previousFile bufprotosource.File, file bufprotosource.File) error {
	return checkFileSameValue(add, previousFile.PhpNamespace(), file.PhpNamespace(), file, file.PhpNamespaceLocation(), `option "php_namespace"`)
}

// CheckFileSamePhpMetadataNamespace is a check function.
var CheckFileSamePhpMetadataNamespace = newFilePairCheckFunc(checkFileSamePhpMetadataNamespace)

func checkFileSamePhpMetadataNamespace(add addFunc, corpus *corpus, previousFile bufprotosource.File, file bufprotosource.File) error {
	return checkFileSameValue(add, previousFile.PhpMetadataNamespace(), file.PhpMetadataNamespace(), file, file.PhpMetadataNamespaceLocation(), `option "php_metadata_namespace"`)
}

// CheckFileSameRubyPackage is a check function.
var CheckFileSameRubyPackage = newFilePairCheckFunc(checkFileSameRubyPackage)

func checkFileSameRubyPackage(add addFunc, corpus *corpus, previousFile bufprotosource.File, file bufprotosource.File) error {
	return checkFileSameValue(add, previousFile.RubyPackage(), file.RubyPackage(), file, file.RubyPackageLocation(), `option "ruby_package"`)
}

// CheckFileSameSwiftPrefix is a check function.
var CheckFileSameSwiftPrefix = newFilePairCheckFunc(checkFileSameSwiftPrefix)

func checkFileSameSwiftPrefix(add addFunc, corpus *corpus, previousFile bufprotosource.File, file bufprotosource.File) error {
	return checkFileSameValue(add, previousFile.SwiftPrefix(), file.SwiftPrefix(), file, file.SwiftPrefixLocation(), `option "swift_prefix"`)
}

// CheckFileSameOptimizeFor is a check function.
var CheckFileSameOptimizeFor = newFilePairCheckFunc(checkFileSameOptimizeFor)

func checkFileSameOptimizeFor(add addFunc, corpus *corpus, previousFile bufprotosource.File, file bufprotosource.File) error {
	return checkFileSameValue(add, previousFile.OptimizeFor().String(), file.OptimizeFor().String(), file, file.OptimizeForLocation(), `option "optimize_for"`)
}

// CheckFileSameCcGenericServices is a check function.
var CheckFileSameCcGenericServices = newFilePairCheckFunc(checkFileSameCcGenericServices)

func checkFileSameCcGenericServices(add addFunc, corpus *corpus, previousFile bufprotosource.File, file bufprotosource.File) error {
	return checkFileSameValue(add, strconv.FormatBool(previousFile.CcGenericServices()), strconv.FormatBool(file.CcGenericServices()), file, file.CcGenericServicesLocation(), `option "cc_generic_services"`)
}

// CheckFileSameJavaGenericServices is a check function.
var CheckFileSameJavaGenericServices = newFilePairCheckFunc(checkFileSameJavaGenericServices)

func checkFileSameJavaGenericServices(add addFunc, corpus *corpus, previousFile bufprotosource.File, file bufprotosource.File) error {
	return checkFileSameValue(add, strconv.FormatBool(previousFile.JavaGenericServices()), strconv.FormatBool(file.JavaGenericServices()), file, file.JavaGenericServicesLocation(), `option "java_generic_services"`)
}

// CheckFileSamePyGenericServices is a check function.
var CheckFileSamePyGenericServices = newFilePairCheckFunc(checkFileSamePyGenericServices)

func checkFileSamePyGenericServices(add addFunc, corpus *corpus, previousFile bufprotosource.File, file bufprotosource.File) error {
	return checkFileSameValue(add, strconv.FormatBool(previousFile.PyGenericServices()), strconv.FormatBool(file.PyGenericServices()), file, file.PyGenericServicesLocation(), `option "py_generic_services"`)
}

// CheckFileSameCcEnableArenas is a check function.
var CheckFileSameCcEnableArenas = newFilePairCheckFunc(checkFileSameCcEnableArenas)

func checkFileSameCcEnableArenas(add addFunc, corpus *corpus, previousFile bufprotosource.File, file bufprotosource.File) error {
	return checkFileSameValue(add, strconv.FormatBool(previousFile.CcEnableArenas()), strconv.FormatBool(file.CcEnableArenas()), file, file.CcEnableArenasLocation(), `option "cc_enable_arenas"`)
}

// CheckFileSameSyntax is a check function.
var CheckFileSameSyntax = newFilePairCheckFunc(checkFileSameSyntax)

func checkFileSameSyntax(add addFunc, corpus *corpus, previousFile bufprotosource.File, file bufprotosource.File) error {
	previousSyntax := previousFile.Syntax()
	if previousSyntax == bufprotosource.SyntaxUnspecified {
		previousSyntax = bufprotosource.SyntaxProto2
	}
	syntax := file.Syntax()
	if syntax == bufprotosource.SyntaxUnspecified {
		syntax = bufprotosource.SyntaxProto2
	}
	return checkFileSameValue(add, previousSyntax.String(), syntax.String(), file, file.SyntaxLocation(), `syntax`)
}

func checkFileSameValue(add addFunc, previousValue interface{}, value interface{}, file bufprotosource.File, location bufprotosource.Location, name string) error {
	if previousValue != value {
		add(file, nil, location, `File %s changed from %q to %q.`, name, previousValue, value)
	}
	return nil
}

// CheckMessageNoDelete is a check function.
var CheckMessageNoDelete = newFilePairCheckFunc(checkMessageNoDelete)

func checkMessageNoDelete(add addFunc, corpus *corpus, previousFile bufprotosource.File, file bufprotosource.File) error {
	previousNestedNameToMessage, err := bufprotosource.NestedNameToMessage(previousFile)
	if err != nil {
		return err
	}
	nestedNameToMessage, err := bufprotosource.NestedNameToMessage(file)
	if err != nil {
		return err
	}
	for previousNestedName := range previousNestedNameToMessage {
		if _, ok := nestedNameToMessage[previousNestedName]; !ok {
			descriptor, location := getDescriptorAndLocationForDeletedMessage(file, nestedNameToMessage, previousNestedName)
			add(descriptor, nil, location, `Previously present message %q was deleted from file.`, previousNestedName)
		}
	}
	return nil
}

// CheckMessageNoRemoveStandardDescriptorAccessor is a check function.
var CheckMessageNoRemoveStandardDescriptorAccessor = newMessagePairCheckFunc(checkMessageNoRemoveStandardDescriptorAccessor)

func checkMessageNoRemoveStandardDescriptorAccessor(add addFunc, corpus *corpus, previousMessage bufprotosource.Message, message bufprotosource.Message) error {
	previous := strconv.FormatBool(previousMessage.NoStandardDescriptorAccessor())
	current := strconv.FormatBool(message.NoStandardDescriptorAccessor())
	if previous == "false" && current == "true" {
		add(message, nil, message.NoStandardDescriptorAccessorLocation(), `Message option "no_standard_descriptor_accessor" changed from %q to %q.`, previous, current)
	}
	return nil
}

// CheckMessageSameJSONFormat is a check function.
var CheckMessageSameJSONFormat = newMessagePairCheckFunc(checkMessageSameJSONFormat)

func checkMessageSameJSONFormat(
	add addFunc,
	_ *corpus,
	previousMessage bufprotosource.Message,
	message bufprotosource.Message,
) error {
	previousDescriptor, err := previousMessage.AsDescriptor()
	if err != nil {
		return err
	}
	descriptor, err := message.AsDescriptor()
	if err != nil {
		return err
	}
	featureField, err := findFeatureField(featureNameJSONFormat, protoreflect.EnumKind)
	if err != nil {
		return err
	}
	val, err := protoutil.ResolveFeature(previousDescriptor, featureField)
	if err != nil {
		return fmt.Errorf("unable to resolve value of %s feature: %w", featureField.Name(), err)
	}
	previousJSONFormat := descriptorpb.FeatureSet_JsonFormat(val.Enum())
	val, err = protoutil.ResolveFeature(descriptor, featureField)
	if err != nil {
		return fmt.Errorf("unable to resolve value of %s feature: %w", featureField.Name(), err)
	}
	jsonFormat := descriptorpb.FeatureSet_JsonFormat(val.Enum())
	if previousJSONFormat == descriptorpb.FeatureSet_ALLOW && jsonFormat != descriptorpb.FeatureSet_ALLOW {
		add(
			message,
			nil,
			withBackupLocation(message.Features().JSONFormatLocation(), message.Location()),
			`Message %q JSON format support changed from %v to %v.`,
			message.Name(),
			previousJSONFormat,
			jsonFormat,
		)
	}
	return nil
}

// CheckMessageSameMessageSetWireFormat is a check function.
var CheckMessageSameMessageSetWireFormat = newMessagePairCheckFunc(checkMessageSameMessageSetWireFormat)

func checkMessageSameMessageSetWireFormat(add addFunc, corpus *corpus, previousMessage bufprotosource.Message, message bufprotosource.Message) error {
	previous := strconv.FormatBool(previousMessage.MessageSetWireFormat())
	current := strconv.FormatBool(message.MessageSetWireFormat())
	if previous != current {
		add(message, nil, message.MessageSetWireFormatLocation(), `Message option "message_set_wire_format" changed from %q to %q.`, previous, current)
	}
	return nil
}

// CheckMessageSameRequiredFields is a check function.
var CheckMessageSameRequiredFields = newMessagePairCheckFunc(checkMessageSameRequiredFields)

func checkMessageSameRequiredFields(add addFunc, corpus *corpus, previousMessage bufprotosource.Message, message bufprotosource.Message) error {
	previousNumberToRequiredField, err := bufprotosource.NumberToMessageFieldForLabel(
		previousMessage,
		descriptorpb.FieldDescriptorProto_LABEL_REQUIRED,
	)
	if err != nil {
		return err
	}
	numberToRequiredField, err := bufprotosource.NumberToMessageFieldForLabel(
		message,
		descriptorpb.FieldDescriptorProto_LABEL_REQUIRED,
	)
	if err != nil {
		return err
	}
	for previousNumber := range previousNumberToRequiredField {
		if _, ok := numberToRequiredField[previousNumber]; !ok {
			// we attach the error to the message as the field no longer exists
			add(message, nil, message.Location(), `Message %q had required field "%d" deleted. Required fields must always be sent, so if one side does not know about the required field, this will result in a breakage.`, previousMessage.Name(), previousNumber)
		}
	}
	for number, requiredField := range numberToRequiredField {
		if _, ok := previousNumberToRequiredField[number]; !ok {
			// we attach the error to the added required field
			add(message, nil, requiredField.Location(), `Message %q had required field "%d" added. Required fields must always be sent, so if one side does not know about the required field, this will result in a breakage.`, message.Name(), number)
		}
	}
	return nil
}

// CheckOneofNoDelete is a check function.
var CheckOneofNoDelete = newMessagePairCheckFunc(checkOneofNoDelete)

func checkOneofNoDelete(add addFunc, corpus *corpus, previousMessage bufprotosource.Message, message bufprotosource.Message) error {
	previousNameToOneof, err := bufprotosource.NameToMessageOneof(previousMessage)
	if err != nil {
		return err
	}
	nameToOneof, err := bufprotosource.NameToMessageOneof(message)
	if err != nil {
		return err
	}
	for previousName, previousOneof := range previousNameToOneof {
		if _, ok := nameToOneof[previousName]; !ok {
			previousOneofDescriptor, err := previousOneof.AsDescriptor()
			if err != nil {
				return err
			}
			if previousOneofDescriptor.IsSynthetic() {
				// Not considering synthetic oneofs since those are really
				// just strange byproducts of how "explicit presence" is
				// modeled in proto3 syntax. We will separately detect this
				// kind of change via field presence check.
				continue
			}
			add(message, nil, message.Location(), `Previously present oneof %q on message %q was deleted.`, previousName, message.Name())
		}
	}
	return nil
}

// CheckPackageEnumNoDelete is a check function.
var CheckPackageEnumNoDelete = newFilesCheckFunc(checkPackageEnumNoDelete)

func checkPackageEnumNoDelete(add addFunc, corpus *corpus) error {
	previousPackageToNestedNameToEnum, err := bufprotosource.PackageToNestedNameToEnum(corpus.previousFiles...)
	if err != nil {
		return err
	}
	packageToNestedNameToEnum, err := bufprotosource.PackageToNestedNameToEnum(corpus.files...)
	if err != nil {
		return err
	}
	// caching across loops
	var filePathToFile map[string]bufprotosource.File
	for previousPackage, previousNestedNameToEnum := range previousPackageToNestedNameToEnum {
		if nestedNameToEnum, ok := packageToNestedNameToEnum[previousPackage]; ok {
			for previousNestedName, previousEnum := range previousNestedNameToEnum {
				if _, ok := nestedNameToEnum[previousNestedName]; !ok {
					// if cache not populated, populate it
					if filePathToFile == nil {
						filePathToFile, err = bufprotosource.FilePathToFile(corpus.files...)
						if err != nil {
							return err
						}
					}
					// Check if the file still exists.
					file, ok := filePathToFile[previousEnum.File().Path()]
					if ok {
						// File exists, try to get a location to attach the error to.
						descriptor, location, err := getDescriptorAndLocationForDeletedElement(file, previousNestedName)
						if err != nil {
							return err
						}
						add(descriptor, nil, location, `Previously present enum %q was deleted from package %q.`, previousNestedName, previousPackage)
					} else {
						// File does not exist, we don't know where the enum was deleted from.
						// Add the previous enum to check for ignores. This means that if
						// ignore_unstable_packages is set, this will be triggered if the
						// previous enum was in an unstable package.
						add(nil, []bufprotosource.Descriptor{previousEnum}, nil, `Previously present enum %q was deleted from package %q.`, previousNestedName, previousPackage)
					}
				}
			}
		}
	}
	return nil
}

// CheckPackageExtensionNoDelete is a check function.
var CheckPackageExtensionNoDelete = newFilesCheckFunc(checkPackageExtensionNoDelete)

func checkPackageExtensionNoDelete(add addFunc, corpus *corpus) error {
	previousPackageToNestedNameToExtension, err := bufprotosource.PackageToNestedNameToExtension(corpus.previousFiles...)
	if err != nil {
		return err
	}
	packageToNestedNameToExtension, err := bufprotosource.PackageToNestedNameToExtension(corpus.files...)
	if err != nil {
		return err
	}
	// caching across loops
	var filePathToFile map[string]bufprotosource.File
	for previousPackage, previousNestedNameToExtension := range previousPackageToNestedNameToExtension {
		if nestedNameToExtension, ok := packageToNestedNameToExtension[previousPackage]; ok {
			for previousNestedName, previousExtension := range previousNestedNameToExtension {
				if _, ok := nestedNameToExtension[previousNestedName]; !ok {
					// if cache not populated, populate it
					if filePathToFile == nil {
						filePathToFile, err = bufprotosource.FilePathToFile(corpus.files...)
						if err != nil {
							return err
						}
					}
					// Check if the file still exists.
					file, ok := filePathToFile[previousExtension.File().Path()]
					if ok {
						// File exists, try to get a location to attach the error to.
						descriptor, location, err := getDescriptorAndLocationForDeletedElement(file, previousNestedName)
						if err != nil {
							return err
						}
						add(descriptor, nil, location, `Previously present extension %q was deleted from package %q.`, previousNestedName, previousPackage)
					} else {
						// File does not exist, we don't know where the enum was deleted from.
						// Add the previous enum to check for ignores. This means that if
						// ignore_unstable_packages is set, this will be triggered if the
						// previous enum was in an unstable package.
						add(nil, []bufprotosource.Descriptor{previousExtension}, nil, `Previously present extension %q was deleted from package %q.`, previousNestedName, previousPackage)
					}
				}
			}
		}
	}
	return nil
}

// CheckPackageMessageNoDelete is a check function.
var CheckPackageMessageNoDelete = newFilesCheckFunc(checkPackageMessageNoDelete)

func checkPackageMessageNoDelete(add addFunc, corpus *corpus) error {
	previousPackageToNestedNameToMessage, err := bufprotosource.PackageToNestedNameToMessage(corpus.previousFiles...)
	if err != nil {
		return err
	}
	packageToNestedNameToMessage, err := bufprotosource.PackageToNestedNameToMessage(corpus.files...)
	if err != nil {
		return err
	}
	// caching across loops
	var filePathToFile map[string]bufprotosource.File
	for previousPackage, previousNestedNameToMessage := range previousPackageToNestedNameToMessage {
		if nestedNameToMessage, ok := packageToNestedNameToMessage[previousPackage]; ok {
			for previousNestedName, previousMessage := range previousNestedNameToMessage {
				if _, ok := nestedNameToMessage[previousNestedName]; !ok {
					// if cache not populated, populate it
					if filePathToFile == nil {
						filePathToFile, err = bufprotosource.FilePathToFile(corpus.files...)
						if err != nil {
							return err
						}
					}
					// Check if the file still exists.
					file, ok := filePathToFile[previousMessage.File().Path()]
					if ok {
						// File exists, try to get a location to attach the error to.
						descriptor, location := getDescriptorAndLocationForDeletedMessage(file, nestedNameToMessage, previousNestedName)
						add(descriptor, nil, location, `Previously present message %q was deleted from package %q.`, previousNestedName, previousPackage)
					} else {
						// File does not exist, we don't know where the message was deleted from.
						// Add the previous message to check for ignores. This means that if
						// ignore_unstable_packages is set, this will be triggered if the
						// previous message was in an unstable package.
						add(nil, []bufprotosource.Descriptor{previousMessage}, nil, `Previously present message %q was deleted from package %q.`, previousNestedName, previousPackage)
					}
				}
			}
		}
	}
	return nil
}

// CheckPackageNoDelete is a check function.
var CheckPackageNoDelete = newFilesCheckFunc(checkPackageNoDelete)

func checkPackageNoDelete(add addFunc, corpus *corpus) error {
	previousPackageToFiles, err := bufprotosource.PackageToFiles(corpus.previousFiles...)
	if err != nil {
		return err
	}
	packageToFiles, err := bufprotosource.PackageToFiles(corpus.files...)
	if err != nil {
		return err
	}
	for previousPackage, previousFiles := range previousPackageToFiles {
		if _, ok := packageToFiles[previousPackage]; !ok {
			// Add previous descriptors in the same package as other descriptors to check
			// for ignores. This will mean that if we have ignore_unstable_packages set,
			// any one of these files will cause the ignore to happen. Note that we
			// could probably just attach a single file, but we do this in case we
			// have other ways to ignore in the future.
			previousDescriptors := make([]bufprotosource.Descriptor, len(previousFiles))
			for i, previousFile := range previousFiles {
				previousDescriptors[i] = previousFile
			}
			add(nil, previousDescriptors, nil, `Previously present package %q was deleted.`, previousPackage)
		}
	}
	return nil
}

// CheckPackageServiceNoDelete is a check function.
var CheckPackageServiceNoDelete = newFilesCheckFunc(checkPackageServiceNoDelete)

func checkPackageServiceNoDelete(add addFunc, corpus *corpus) error {
	previousPackageToNameToService, err := bufprotosource.PackageToNameToService(corpus.previousFiles...)
	if err != nil {
		return err
	}
	packageToNameToService, err := bufprotosource.PackageToNameToService(corpus.files...)
	if err != nil {
		return err
	}
	// caching across loops
	var filePathToFile map[string]bufprotosource.File
	for previousPackage, previousNameToService := range previousPackageToNameToService {
		if nameToService, ok := packageToNameToService[previousPackage]; ok {
			for previousName, previousService := range previousNameToService {
				if _, ok := nameToService[previousName]; !ok {
					// if cache not populated, populate it
					if filePathToFile == nil {
						filePathToFile, err = bufprotosource.FilePathToFile(corpus.files...)
						if err != nil {
							return err
						}
					}
					// Check if the file still exists.
					file, ok := filePathToFile[previousService.File().Path()]
					if ok {
						// File exists.
						add(file, nil, nil, `Previously present service %q was deleted from package %q.`, previousName, previousPackage)
					} else {
						// File does not exist, we don't know where the service was deleted from.
						// Add the previous service to check for ignores. This means that if
						// ignore_unstable_packages is set, this will be triggered if the
						// previous service was in an unstable package.
						// TODO: find the service and print that this moved?
						add(nil, []bufprotosource.Descriptor{previousService}, nil, `Previously present service %q was deleted from package %q.`, previousName, previousPackage)
					}
				}
			}
		}
	}
	return nil
}

// CheckReservedEnumNoDelete is a check function.
var CheckReservedEnumNoDelete = newEnumPairCheckFunc(checkReservedEnumNoDelete)

func checkReservedEnumNoDelete(add addFunc, corpus *corpus, previousEnum bufprotosource.Enum, enum bufprotosource.Enum) error {
	if err := checkTagRanges(add, "reserved", enum, previousEnum.ReservedEnumRanges(), enum.ReservedEnumRanges()); err != nil {
		return err
	}
	previousValueToReservedName := bufprotosource.ValueToReservedName(previousEnum)
	valueToReservedName := bufprotosource.ValueToReservedName(enum)
	for previousValue := range previousValueToReservedName {
		if _, ok := valueToReservedName[previousValue]; !ok {
			add(enum, nil, enum.Location(), `Previously present reserved name %q on enum %q was deleted.`, previousValue, enum.Name())
		}
	}
	return nil
}

// CheckReservedMessageNoDelete is a check function.
var CheckReservedMessageNoDelete = newMessagePairCheckFunc(checkReservedMessageNoDelete)

func checkReservedMessageNoDelete(add addFunc, corpus *corpus, previousMessage bufprotosource.Message, message bufprotosource.Message) error {
	if err := checkTagRanges(add, "reserved", message, previousMessage.ReservedMessageRanges(), message.ReservedMessageRanges()); err != nil {
		return err
	}
	previousValueToReservedName := bufprotosource.ValueToReservedName(previousMessage)
	valueToReservedName := bufprotosource.ValueToReservedName(message)
	for previousValue := range previousValueToReservedName {
		if _, ok := valueToReservedName[previousValue]; !ok {
			add(message, nil, message.Location(), `Previously present reserved name %q on message %q was deleted.`, previousValue, message.Name())
		}
	}
	return nil
}

// CheckRPCNoDelete is a check function.
var CheckRPCNoDelete = newServicePairCheckFunc(checkRPCNoDelete)

func checkRPCNoDelete(add addFunc, corpus *corpus, previousService bufprotosource.Service, service bufprotosource.Service) error {
	previousNameToMethod, err := bufprotosource.NameToMethod(previousService)
	if err != nil {
		return err
	}
	nameToMethod, err := bufprotosource.NameToMethod(service)
	if err != nil {
		return err
	}
	for previousName := range previousNameToMethod {
		if _, ok := nameToMethod[previousName]; !ok {
			add(service, nil, service.Location(), `Previously present RPC %q on service %q was deleted.`, previousName, service.Name())
		}
	}
	return nil
}

// CheckRPCSameClientStreaming is a check function.
var CheckRPCSameClientStreaming = newMethodPairCheckFunc(checkRPCSameClientStreaming)

func checkRPCSameClientStreaming(add addFunc, corpus *corpus, previousMethod bufprotosource.Method, method bufprotosource.Method) error {
	if previousMethod.ClientStreaming() != method.ClientStreaming() {
		previous := "streaming"
		current := "unary"
		if method.ClientStreaming() {
			previous = "unary"
			current = "streaming"
		}
		add(method, nil, method.Location(), `RPC %q on service %q changed from client %s to client %s.`, method.Name(), method.Service().Name(), previous, current)
	}
	return nil
}

// CheckRPCSameIdempotencyLevel is a check function.
var CheckRPCSameIdempotencyLevel = newMethodPairCheckFunc(checkRPCSameIdempotencyLevel)

func checkRPCSameIdempotencyLevel(add addFunc, corpus *corpus, previousMethod bufprotosource.Method, method bufprotosource.Method) error {
	previous := previousMethod.IdempotencyLevel()
	current := method.IdempotencyLevel()
	if previous != current {
		add(method, nil, method.IdempotencyLevelLocation(), `RPC %q on service %q changed option "idempotency_level" from %q to %q.`, method.Name(), method.Service().Name(), previous.String(), current.String())
	}
	return nil
}

// CheckRPCSameRequestType is a check function.
var CheckRPCSameRequestType = newMethodPairCheckFunc(checkRPCSameRequestType)

func checkRPCSameRequestType(add addFunc, corpus *corpus, previousMethod bufprotosource.Method, method bufprotosource.Method) error {
	if previousMethod.InputTypeName() != method.InputTypeName() {
		add(method, nil, method.InputTypeLocation(), `RPC %q on service %q changed request type from %q to %q.`, method.Name(), method.Service().Name(), previousMethod.InputTypeName(), method.InputTypeName())
	}
	return nil
}

// CheckRPCSameResponseType is a check function.
var CheckRPCSameResponseType = newMethodPairCheckFunc(checkRPCSameResponseType)

func checkRPCSameResponseType(add addFunc, corpus *corpus, previousMethod bufprotosource.Method, method bufprotosource.Method) error {
	if previousMethod.OutputTypeName() != method.OutputTypeName() {
		add(method, nil, method.OutputTypeLocation(), `RPC %q on service %q changed response type from %q to %q.`, method.Name(), method.Service().Name(), previousMethod.OutputTypeName(), method.OutputTypeName())
	}
	return nil
}

// CheckRPCSameServerStreaming is a check function.
var CheckRPCSameServerStreaming = newMethodPairCheckFunc(checkRPCSameServerStreaming)

func checkRPCSameServerStreaming(add addFunc, corpus *corpus, previousMethod bufprotosource.Method, method bufprotosource.Method) error {
	if previousMethod.ServerStreaming() != method.ServerStreaming() {
		previous := "streaming"
		current := "unary"
		if method.ServerStreaming() {
			previous = "unary"
			current = "streaming"
		}
		add(method, nil, method.Location(), `RPC %q on service %q changed from server %s to server %s.`, method.Name(), method.Service().Name(), previous, current)
	}
	return nil
}

// CheckServiceNoDelete is a check function.
var CheckServiceNoDelete = newFilePairCheckFunc(checkServiceNoDelete)

func checkServiceNoDelete(add addFunc, corpus *corpus, previousFile bufprotosource.File, file bufprotosource.File) error {
	previousNameToService, err := bufprotosource.NameToService(previousFile)
	if err != nil {
		return err
	}
	nameToService, err := bufprotosource.NameToService(file)
	if err != nil {
		return err
	}
	for previousName := range previousNameToService {
		if _, ok := nameToService[previousName]; !ok {
			add(file, nil, nil, `Previously present service %q was deleted from file.`, previousName)
		}
	}
	return nil
}
