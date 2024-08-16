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

package bufcheckserverhandle

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/bufbuild/buf/private/buf/bufcheck/internal/bufcheckserver/internal/bufcheckserverutil"
	"github.com/bufbuild/buf/private/bufpkg/bufprotosource"
	"github.com/bufbuild/buf/private/gen/proto/go/google/protobuf"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/protocompile/protoutil"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// HandleBreakingEnumNoDelete is a check function.
var HandleBreakingEnumNoDelete = bufcheckserverutil.NewBreakingFilePairRuleHandler(handleBreakingEnumNoDelete)

func handleBreakingEnumNoDelete(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	previousFile bufprotosource.File,
	file bufprotosource.File,
) error {
	previousNestedNameToEnum, err := bufprotosource.NestedNameToEnum(previousFile)
	if err != nil {
		return err
	}
	nestedNameToEnum, err := bufprotosource.NestedNameToEnum(file)
	if err != nil {
		return err
	}
	for previousNestedName, previousEnum := range previousNestedNameToEnum {
		if _, ok := nestedNameToEnum[previousNestedName]; !ok {
			// TODO: search for enum in other files and return that the enum was moved?
			_, location, err := getDescriptorAndLocationForDeletedElement(file, previousNestedName)
			if err != nil {
				return err
			}
			responseWriter.AddProtosourceAnnotation(
				location,
				previousEnum.Location(),
				`Previously present enum %q was deleted from file.`,
				previousNestedName,
			)
		}
	}
	return nil
}

// HandleBreakingExtensionNoDelete is a check function.
var HandleBreakingExtensionNoDelete = bufcheckserverutil.NewBreakingFilePairRuleHandler(handleBreakingExtensionNoDelete)

func handleBreakingExtensionNoDelete(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	previousFile bufprotosource.File,
	file bufprotosource.File,
) error {
	previousNestedNameToExtension, err := bufprotosource.NestedNameToExtension(previousFile)
	if err != nil {
		return err
	}
	nestedNameToExtension, err := bufprotosource.NestedNameToExtension(file)
	if err != nil {
		return err
	}
	for previousNestedName, previousExtension := range previousNestedNameToExtension {
		if _, ok := nestedNameToExtension[previousNestedName]; !ok {
			_, location, err := getDescriptorAndLocationForDeletedElement(file, previousNestedName)
			if err != nil {
				return err
			}
			responseWriter.AddProtosourceAnnotation(
				location,
				previousExtension.Location(),
				`Previously present extension %q was deleted from file.`,
				previousNestedName,
			)
		}
	}
	return nil
}

// HandleBreakingFileDelete is a check function.
var HandleBreakingFileNoDelete = bufcheckserverutil.NewRuleHandler(handleBreakingFileNoDelete)

func handleBreakingFileNoDelete(
	_ context.Context,
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
) error {
	previousFilePathToFile, err := bufprotosource.FilePathToFile(request.AgainstProtosourceFiles()...)
	if err != nil {
		return err
	}
	filePathToFile, err := bufprotosource.FilePathToFile(request.ProtosourceFiles()...)
	if err != nil {
		return err
	}
	for previousFilePath := range previousFilePathToFile {
		if _, ok := filePathToFile[previousFilePath]; !ok {
			responseWriter.AddProtosourceAnnotation(
				nil,
				nil, // TODO: File does not have a Location, make sure that client handles the ignore checks
				`Previously present file %q was deleted.`,
				previousFilePath,
			)
		}
	}
	return nil
}

// HandleBreakingMessageNoDelete is a check function.
var HandleBreakingMessageNoDelete = bufcheckserverutil.NewBreakingFilePairRuleHandler(handleBreakingMessageNoDelete)

func handleBreakingMessageNoDelete(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	previousFile bufprotosource.File,
	file bufprotosource.File,
) error {
	previousNestedNameToMessage, err := bufprotosource.NestedNameToMessage(previousFile)
	if err != nil {
		return err
	}
	nestedNameToMessage, err := bufprotosource.NestedNameToMessage(file)
	if err != nil {
		return err
	}
	for previousNestedName, previousMessage := range previousNestedNameToMessage {
		if _, ok := nestedNameToMessage[previousNestedName]; !ok {
			_, location := getDescriptorAndLocationForDeletedMessage(file, nestedNameToMessage, previousNestedName)
			responseWriter.AddProtosourceAnnotation(
				location,
				previousMessage.Location(),
				`Previously present message %q was deleted from file.`,
				previousNestedName,
			)
		}
	}
	return nil
}

// HandleBreakingServiceNoDelete is a check function.
var HandleBreakingServiceNoDelete = bufcheckserverutil.NewBreakingFilePairRuleHandler(handleBreakingServiceNoDelete)

func handleBreakingServiceNoDelete(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	previousFile bufprotosource.File,
	file bufprotosource.File,
) error {
	previousNameToService, err := bufprotosource.NameToService(previousFile)
	if err != nil {
		return err
	}
	nameToService, err := bufprotosource.NameToService(file)
	if err != nil {
		return err
	}
	for previousName, previousService := range previousNameToService {
		if _, ok := nameToService[previousName]; !ok {
			responseWriter.AddProtosourceAnnotation(
				nil,
				previousService.Location(),
				`Previously present service %q was deleted from file.`,
				previousName,
			)
		}
	}
	return nil
}

// HandleBreakingEnumSameType is a check function.
var HandleBreakingEnumSameType = bufcheckserverutil.NewBreakingEnumPairRuleHandler(handleBreakingEnumSameType)

func handleBreakingEnumSameType(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
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
	if previousDescriptor.IsClosed() != descriptor.IsClosed() {
		previousState, currentState := "closed", "open"
		if descriptor.IsClosed() {
			previousState, currentState = currentState, previousState
		}
		responseWriter.AddProtosourceAnnotation(
			withBackupLocation(enum.Features().EnumTypeLocation(), enum.Location()),
			withBackupLocation(previousEnum.Features().EnumTypeLocation(), previousEnum.Location()),
			`Enum %q changed from %s to %s.`,
			enum.Name(),
			previousState,
			currentState,
		)
	}
	return nil
}

// HandleBreakingEnumValueNoDelete is a check function.
var HandleBreakingEnumValueNoDelete = bufcheckserverutil.NewBreakingEnumPairRuleHandler(handleBreakingEnumValueNoDelete)

func handleBreakingEnumValueNoDelete(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	previousEnum bufprotosource.Enum,
	enum bufprotosource.Enum,
) error {
	return checkEnumValueNoDeleteWithRules(
		responseWriter,
		previousEnum,
		enum,
		false,
		false,
	)
}

func checkEnumValueNoDeleteWithRules(
	responseWriter bufcheckserverutil.ResponseWriter,
	previousEnum bufprotosource.Enum,
	enum bufprotosource.Enum,
	allowIfNumberReserved bool,
	allowIfNameReserved bool,
) error {
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
			if !isDeletedEnumValueAllowedWithRules(
				previousNumber,
				previousNameToEnumValue,
				enum,
				allowIfNumberReserved,
				allowIfNameReserved,
			) {
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
				responseWriter.AddProtosourceAnnotation(
					enum.Location(),
					previousEnum.Location(),
					`Previously present enum value "%d" on enum %q was deleted%s.`,
					previousNumber,
					enum.Name(),
					suffix,
				)
			}
		}
	}
	return nil
}

func isDeletedEnumValueAllowedWithRules(
	previousNumber int,
	previousNameToEnumValue map[string]bufprotosource.EnumValue,
	enum bufprotosource.Enum,
	allowIfNumberReserved bool,
	allowIfNameReserved bool,
) bool {
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

// HandleBreakingExtensionMessageNoDelete is a check function.
var HandleBreakingExtensionMessageNoDelete = bufcheckserverutil.NewBreakingMessagePairRuleHandler(handleBreakingExtensionMessageNoDelete)

func handleBreakingExtensionMessageNoDelete(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	previousMessage bufprotosource.Message,
	message bufprotosource.Message,
) error {
	return checkTagRanges(
		responseWriter,
		"extension",
		message,
		previousMessage,
		previousMessage.ExtensionRanges(),
		message.ExtensionRanges(),
	)
}

// HandleBreakingFieldNoDelete is a check function.
var HandleBreakingFieldNoDelete = bufcheckserverutil.NewBreakingMessagePairRuleHandler(handleBreakingFieldNoDelete)

func handleBreakingFieldNoDelete(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	previousMessage bufprotosource.Message,
	message bufprotosource.Message,
) error {
	return checkFieldNoDeleteWithRules(
		responseWriter,
		previousMessage,
		message,
		false,
		false,
	)
}

func checkFieldNoDeleteWithRules(
	responseWriter bufcheckserverutil.ResponseWriter,
	previousMessage bufprotosource.Message,
	message bufprotosource.Message,
	allowIfNumberReserved bool,
	allowIfNameReserved bool,
) error {
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
				responseWriter.AddProtosourceAnnotation(
					message.Location(),
					previousMessage.Location(),
					`Previously present %s was deleted%s.`,
					description,
					suffix,
				)
			}
		}
	}
	return nil
}

func isDeletedFieldAllowedWithRules(
	previousField bufprotosource.Field,
	message bufprotosource.Message,
	allowIfNumberReserved bool,
	allowIfNameReserved bool,
) bool {
	return (allowIfNumberReserved && bufprotosource.NumberInReservedRanges(previousField.Number(), message.ReservedTagRanges()...)) ||
		(allowIfNameReserved && bufprotosource.NameInReservedNames(previousField.Name(), message.ReservedNames()...))
}

// HandleBreakingFieldSameCardinality is a check function.
var HandleBreakingFieldSameCardinality = bufcheckserverutil.NewBreakingFieldPairRuleHandler(handleBreakingFieldSameCardinality)

func handleBreakingFieldSameCardinality(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	previousField bufprotosource.Field,
	field bufprotosource.Field,
) error {
	previousDescriptor, err := previousField.AsDescriptor()
	if err != nil {
		return err
	}
	descriptor, err := field.AsDescriptor()
	if err != nil {
		return err
	}
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
		responseWriter.AddProtosourceAnnotation(
			field.Location(),
			previousField.Location(),
			`%s changed cardinality from %q to %q.`,
			fieldDescription(field),
			previousCardinality,
			currentCardinality,
		)
	}
	return nil
}

// HandleBreakingFieldSameCppStringType is a check function.
var HandleBreakingFieldSameCppStringType = bufcheckserverutil.NewBreakingFieldPairRuleHandler(handleBreakingFieldSameCppStringType)

func handleBreakingFieldSameCppStringType(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	previousField bufprotosource.Field,
	field bufprotosource.Field,
) error {
	previousDescriptor, err := previousField.AsDescriptor()
	if err != nil {
		return err
	}
	descriptor, err := field.AsDescriptor()
	if err != nil {
		return err
	}
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
		responseWriter.AddProtosourceAnnotation(
			withBackupLocation(field.CTypeLocation(), fieldCppStringTypeLocation(field), field.Location()),
			withBackupLocation(previousField.CTypeLocation(), fieldCppStringTypeLocation(previousField), previousField.Location()),
			`%s changed C++ string type from %q to %q.`,
			fieldDescription(field),
			previousType,
			currentType,
		)
	}
	return nil
}

// HandleBreakingFieldSameJavaUTF8Validation is a check function.
var HandleBreakingFieldSameJavaUTF8Validation = bufcheckserverutil.NewBreakingFieldPairRuleHandler(handleBreakingFieldSameJavaUTF8Validation)

func handleBreakingFieldSameJavaUTF8Validation(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	previousField bufprotosource.Field,
	field bufprotosource.Field,
) error {
	previousDescriptor, err := previousField.AsDescriptor()
	if err != nil {
		return err
	}
	descriptor, err := field.AsDescriptor()
	if err != nil {
		return err
	}
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
		responseWriter.AddProtosourceAnnotation(
			withBackupLocation(field.File().JavaStringCheckUtf8Location(), fieldJavaUTF8ValidationLocation(field), field.Location()),
			withBackupLocation(previousField.File().JavaStringCheckUtf8Location(), fieldJavaUTF8ValidationLocation(previousField), previousField.Location()),
			`%s changed Java string UTF8 validation from %q to %q.`,
			fieldDescription(field),
			previousValidation,
			validation,
		)
	}
	return nil
}

// HandleBreakingFieldSameJSType is a check function.
var HandleBreakingFieldSameJSType = bufcheckserverutil.NewBreakingFieldPairRuleHandler(handleBreakingFieldSameJSType)

func handleBreakingFieldSameJSType(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	previousField bufprotosource.Field,
	field bufprotosource.Field,
) error {
	if !is64bitInteger(previousField.Type()) || !is64bitInteger(field.Type()) {
		// this check only applies to 64-bit integer fields
		return nil
	}
	if previousField.JSType() != field.JSType() {
		responseWriter.AddProtosourceAnnotation(
			withBackupLocation(field.JSTypeLocation(), field.Location()),
			withBackupLocation(previousField.JSTypeLocation(), previousField.Location()),
			`%s changed option "jstype" from %q to %q.`,
			fieldDescription(field),
			previousField.JSType().String(), field.JSType().String())
	}
	return nil
}

// HandleBreakingFieldSameType is a check function.
var HandleBreakingFieldSameType = bufcheckserverutil.NewBreakingFieldPairRuleHandler(handleBreakingFieldSameType)

func handleBreakingFieldSameType(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	previousField bufprotosource.Field,
	field bufprotosource.Field,
) error {
	previousDescriptor, err := previousField.AsDescriptor()
	if err != nil {
		return err
	}
	descriptor, err := field.AsDescriptor()
	if err != nil {
		return err
	}
	// We use descriptor.Kind(), instead of field.Type(), because it also includes
	// a check of resolved features in Editions files so it can distinguish between
	// normal (length-prefixed) and delimited (aka "group" encoded) messages, which
	// are not compatible.
	if previousDescriptor.Kind() != descriptor.Kind() {
		addFieldChangedType(
			responseWriter,
			previousField,
			previousDescriptor,
			field,
			descriptor,
		)
		return nil
	}

	switch field.Type() {
	case descriptorpb.FieldDescriptorProto_TYPE_ENUM,
		descriptorpb.FieldDescriptorProto_TYPE_GROUP,
		descriptorpb.FieldDescriptorProto_TYPE_MESSAGE:
		if previousField.TypeName() != field.TypeName() {
			addEnumGroupMessageFieldChangedTypeName(responseWriter, previousField, field)
		}
	}
	return nil
}

func addFieldChangedType(
	responseWriter bufcheckserverutil.ResponseWriter,
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
	var previousFieldLocation bufprotosource.Location
	switch previousDescriptor.Kind() {
	case protoreflect.MessageKind, protoreflect.EnumKind, protoreflect.GroupKind:
		previousFieldLocation = previousField.TypeNameLocation()
	default:
		previousFieldLocation = previousField.TypeLocation()
	}
	responseWriter.AddProtosourceAnnotation(
		fieldLocation,
		previousFieldLocation,
		`%s changed type from %q to %q.%s`,
		fieldDescription(field),
		fieldDescriptorTypePrettyString(previousDescriptor),
		fieldDescriptorTypePrettyString(descriptor),
		combinedExtraMessage,
	)
}

func addEnumGroupMessageFieldChangedTypeName(
	responseWriter bufcheckserverutil.ResponseWriter,
	previousField bufprotosource.Field,
	field bufprotosource.Field,
) {
	responseWriter.AddProtosourceAnnotation(
		field.TypeNameLocation(),
		previousField.TypeNameLocation(),
		`%s changed type from %q to %q.`,
		fieldDescription(field),
		strings.TrimPrefix(previousField.TypeName(), "."),
		strings.TrimPrefix(field.TypeName(), "."),
	)
}

// HandleBreakingFieldSameUTF8Validation is a check function.
var HandleBreakingFieldSameUTF8Validation = bufcheckserverutil.NewBreakingFieldPairRuleHandler(handleBreakingFieldSameUTF8Validation)

func handleBreakingFieldSameUTF8Validation(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	previousField bufprotosource.Field,
	field bufprotosource.Field,
) error {
	previousDescriptor, err := previousField.AsDescriptor()
	if err != nil {
		return err
	}
	descriptor, err := field.AsDescriptor()
	if err != nil {
		return err
	}
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
		responseWriter.AddProtosourceAnnotation(
			withBackupLocation(field.Features().UTF8ValidationLocation(), field.Location()),
			withBackupLocation(previousField.Features().UTF8ValidationLocation(), previousField.Location()),
			`%s changed UTF8 validation from %v to %v.`,
			fieldDescription(field),
			previousUTF8Validation,
			utf8Validation,
		)
	}
	return nil
}

// HandleBreakingFileSameCcEnableArenas is a check function.
var HandleBreakingFileSameCcEnableArenas = bufcheckserverutil.NewBreakingFilePairRuleHandler(handleBreakingFileSameCcEnableArenas)

func handleBreakingFileSameCcEnableArenas(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	previousFile bufprotosource.File,
	file bufprotosource.File,
) error {
	return checkFileSameValue(
		responseWriter,
		strconv.FormatBool(previousFile.CcEnableArenas()),
		strconv.FormatBool(file.CcEnableArenas()),
		file,
		file.CcEnableArenasLocation(),
		previousFile.CcEnableArenasLocation(),
		`option "cc_enable_arenas"`,
	)
}

// HandleBreakingFileSameCcGenericServices is a check function.
var HandleBreakingFileSameCcGenericServices = bufcheckserverutil.NewBreakingFilePairRuleHandler(handleBreakingFileSameCcGenericServices)

func handleBreakingFileSameCcGenericServices(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	previousFile bufprotosource.File,
	file bufprotosource.File,
) error {
	return checkFileSameValue(
		responseWriter,
		strconv.FormatBool(previousFile.CcGenericServices()),
		strconv.FormatBool(file.CcGenericServices()),
		file,
		file.CcGenericServicesLocation(),
		previousFile.CcGenericServicesLocation(),
		`option "cc_generic_services"`,
	)
}

// HandleBreakingFileSameCsharpNamespace is a check function.
var HandleBreakingFileSameCsharpNamespace = bufcheckserverutil.NewBreakingFilePairRuleHandler(handleBreakingFileSameCsharpNamespace)

func handleBreakingFileSameCsharpNamespace(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	previousFile bufprotosource.File,
	file bufprotosource.File,
) error {
	return checkFileSameValue(
		responseWriter,
		previousFile.CsharpNamespace(),
		file.CsharpNamespace(),
		file,
		file.CsharpNamespaceLocation(),
		previousFile.CsharpNamespaceLocation(),
		`option "csharp_namespace"`,
	)
}

// HandleBreakingFileSameGoPackage is a check function.
var HandleBreakingFileSameGoPackage = bufcheckserverutil.NewBreakingFilePairRuleHandler(handleBreakingFileSameGoPackage)

func handleBreakingFileSameGoPackage(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	previousFile bufprotosource.File,
	file bufprotosource.File,
) error {
	return checkFileSameValue(
		responseWriter,
		previousFile.GoPackage(),
		file.GoPackage(),
		file,
		file.GoPackageLocation(),
		previousFile.GoPackageLocation(),
		`option "go_package"`,
	)
}

// HandleBreakingFileSameJavaGenericServices is a check function.
var HandleBreakingFileSameJavaGenericServices = bufcheckserverutil.NewBreakingFilePairRuleHandler(handleBreakingFileSameJavaGenericServices)

func handleBreakingFileSameJavaGenericServices(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	previousFile bufprotosource.File,
	file bufprotosource.File,
) error {
	return checkFileSameValue(
		responseWriter,
		strconv.FormatBool(previousFile.JavaGenericServices()),
		strconv.FormatBool(file.JavaGenericServices()),
		file,
		file.JavaGenericServicesLocation(),
		previousFile.JavaGenericServicesLocation(),
		`option "java_generic_services"`,
	)
}

// HandleBreakingFileSameJavaMultipleFiles is a check function.
var HandleBreakingFileSameJavaMultipleFiles = bufcheckserverutil.NewBreakingFilePairRuleHandler(handleBreakingFileSameJavaMultipleFiles)

func handleBreakingFileSameJavaMultipleFiles(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	previousFile bufprotosource.File,
	file bufprotosource.File,
) error {
	return checkFileSameValue(
		responseWriter,
		strconv.FormatBool(previousFile.JavaMultipleFiles()),
		strconv.FormatBool(file.JavaMultipleFiles()),
		file,
		file.JavaMultipleFilesLocation(),
		previousFile.JavaMultipleFilesLocation(),
		`option "java_multiple_files"`,
	)
}

// HandleBreakingFileSameJavaOuterClassname is a check function.
var HandleBreakingFileSameJavaOuterClassname = bufcheckserverutil.NewBreakingFilePairRuleHandler(handleBreakingFileSameJavaOuterClassname)

func handleBreakingFileSameJavaOuterClassname(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	previousFile bufprotosource.File,
	file bufprotosource.File,
) error {
	return checkFileSameValue(
		responseWriter,
		previousFile.JavaOuterClassname(),
		file.JavaOuterClassname(),
		file,
		file.JavaOuterClassnameLocation(),
		previousFile.JavaOuterClassnameLocation(),
		`option "java_outer_classname"`,
	)
}

// HandleBreakingFileSameJavaPackage is a check function.
var HandleBreakingFileSameJavaPackage = bufcheckserverutil.NewBreakingFilePairRuleHandler(handleBreakingFileSameJavaPackage)

func handleBreakingFileSameJavaPackage(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	previousFile bufprotosource.File,
	file bufprotosource.File,
) error {
	return checkFileSameValue(
		responseWriter,
		previousFile.JavaPackage(),
		file.JavaPackage(),
		file,
		file.JavaPackageLocation(),
		previousFile.JavaPackageLocation(),
		`option "java_package"`,
	)
}

// HandleBreakingfileSameObjcClassPrefix is a check function.
var HandleBreakingFileSameObjcClassPrefix = bufcheckserverutil.NewBreakingFilePairRuleHandler(handleBreakingFileSameObjcClassPrefix)

func handleBreakingFileSameObjcClassPrefix(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	previousFile bufprotosource.File,
	file bufprotosource.File,
) error {
	return checkFileSameValue(
		responseWriter,
		previousFile.ObjcClassPrefix(),
		file.ObjcClassPrefix(),
		file,
		file.ObjcClassPrefixLocation(),
		previousFile.ObjcClassPrefixLocation(),
		`option "objc_class_prefix"`,
	)
}

// HandleBreakingFileSameOptimizeFor is a check function.
var HandleBreakingFileSameOptimizeFor = bufcheckserverutil.NewBreakingFilePairRuleHandler(handleBreakingFileSameOptimizeFor)

func handleBreakingFileSameOptimizeFor(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	previousFile bufprotosource.File,
	file bufprotosource.File,
) error {
	return checkFileSameValue(
		responseWriter,
		previousFile.OptimizeFor().String(),
		file.OptimizeFor().String(), file,
		file.OptimizeForLocation(),
		previousFile.OptimizeForLocation(),
		`option "optimize_for"`,
	)
}

// HandleBreakingFileSamePhpClassPrefix is a check function.
var HandleBreakingFileSamePhpClassPrefix = bufcheckserverutil.NewBreakingFilePairRuleHandler(handleBreakingFileSamePhpClassPrefix)

func handleBreakingFileSamePhpClassPrefix(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	previousFile bufprotosource.File,
	file bufprotosource.File,
) error {
	return checkFileSameValue(
		responseWriter,
		previousFile.PhpClassPrefix(),
		file.PhpClassPrefix(),
		file,
		file.PhpClassPrefixLocation(),
		previousFile.PhpClassPrefixLocation(),
		`option "php_class_prefix"`,
	)
}

// HandleBreakingFileSamePhpMetadataNamespace is a check function.
var HandleBreakingFileSamePhpMetadataNamespace = bufcheckserverutil.NewBreakingFilePairRuleHandler(handleBreakingFileSamePhpMetadataNamespace)

func handleBreakingFileSamePhpMetadataNamespace(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	previousFile bufprotosource.File,
	file bufprotosource.File,
) error {
	return checkFileSameValue(
		responseWriter,
		previousFile.PhpMetadataNamespace(),
		file.PhpMetadataNamespace(),
		file,
		file.PhpMetadataNamespaceLocation(),
		previousFile.PhpMetadataNamespaceLocation(),
		`option "php_metadata_namespace"`,
	)
}

// HandleBreakingFileSamePhpNamespace is a check function.
var HandleBreakingFileSamePhpNamespace = bufcheckserverutil.NewBreakingFilePairRuleHandler(handleBreakingFileSamePhpNamespace)

func handleBreakingFileSamePhpNamespace(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	previousFile bufprotosource.File,
	file bufprotosource.File,
) error {
	return checkFileSameValue(
		responseWriter,
		previousFile.PhpNamespace(),
		file.PhpNamespace(),
		file,
		file.PhpNamespaceLocation(),
		previousFile.PhpNamespaceLocation(),
		`option "php_namespace"`,
	)
}

// HandleBreakingFileSamePyGenericServices is a check function.
var HandleBreakingFileSamePyGenericServices = bufcheckserverutil.NewBreakingFilePairRuleHandler(handleBreakingFileSamePyGenericServices)

func handleBreakingFileSamePyGenericServices(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	previousFile bufprotosource.File,
	file bufprotosource.File,
) error {
	return checkFileSameValue(
		responseWriter,
		strconv.FormatBool(previousFile.PyGenericServices()),
		strconv.FormatBool(file.PyGenericServices()),
		file,
		file.PyGenericServicesLocation(),
		previousFile.PyGenericServicesLocation(),
		`option "py_generic_services"`,
	)
}

// HandleBreakingFileSameRubyPackage is a check function.
var HandleBreakingFileSameRubyPackage = bufcheckserverutil.NewBreakingFilePairRuleHandler(handleBreakingFileSameRubyPackage)

func handleBreakingFileSameRubyPackage(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	previousFile bufprotosource.File,
	file bufprotosource.File,
) error {
	return checkFileSameValue(
		responseWriter,
		previousFile.RubyPackage(),
		file.RubyPackage(),
		file,
		file.RubyPackageLocation(),
		previousFile.RubyPackageLocation(),
		`option "ruby_package"`,
	)
}

func checkFileSameValue(
	responseWriter bufcheckserverutil.ResponseWriter,
	previousValue interface{},
	value interface{},
	file bufprotosource.File,
	location bufprotosource.Location,
	previousLocation bufprotosource.Location,
	name string,
) error {
	if previousValue != value {
		responseWriter.AddProtosourceAnnotation(
			location,
			previousLocation,
			`File %s changed from %q to %q.`,
			name,
			previousValue,
			value,
		)
	}
	return nil
}
