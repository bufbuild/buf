package internal

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/bufbuild/buf/internal/pkg/protodesc"
	"github.com/bufbuild/buf/internal/pkg/util/utilstring"
)

// CheckEnumNoDelete is a check function.
var CheckEnumNoDelete = newFilePairCheckFunc(checkEnumNoDelete)

func checkEnumNoDelete(add addFunc, previousFile protodesc.File, file protodesc.File) error {
	previousNestedNameToEnum, err := protodesc.NestedNameToEnum(previousFile)
	if err != nil {
		return err
	}
	nestedNameToEnum, err := protodesc.NestedNameToEnum(file)
	if err != nil {
		return err
	}
	for previousNestedName := range previousNestedNameToEnum {
		if _, ok := nestedNameToEnum[previousNestedName]; !ok {
			// TODO: search for enum in other files and return that the enum was moved?
			descriptor, location, err := getDescriptorAndLocationForDeletedEnum(file, previousNestedName)
			if err != nil {
				return err
			}
			add(descriptor, location, `Previously present enum %q was deleted from file.`, previousNestedName)
		}
	}
	return nil
}

// CheckEnumValueNoDelete is a check function.
var CheckEnumValueNoDelete = newEnumPairCheckFunc(checkEnumValueNoDelete)

func checkEnumValueNoDelete(add addFunc, previousEnum protodesc.Enum, enum protodesc.Enum) error {
	return checkEnumValueNoDeleteWithRules(add, previousEnum, enum, false, false)
}

// CheckEnumValueNoDeleteUnlessNumberReserved is a check function.
var CheckEnumValueNoDeleteUnlessNumberReserved = newEnumPairCheckFunc(checkEnumValueNoDeleteUnlessNumberReserved)

func checkEnumValueNoDeleteUnlessNumberReserved(add addFunc, previousEnum protodesc.Enum, enum protodesc.Enum) error {
	return checkEnumValueNoDeleteWithRules(add, previousEnum, enum, true, false)
}

// CheckEnumValueNoDeleteUnlessNameReserved is a check function.
var CheckEnumValueNoDeleteUnlessNameReserved = newEnumPairCheckFunc(checkEnumValueNoDeleteUnlessNameReserved)

func checkEnumValueNoDeleteUnlessNameReserved(add addFunc, previousEnum protodesc.Enum, enum protodesc.Enum) error {
	return checkEnumValueNoDeleteWithRules(add, previousEnum, enum, false, true)
}

func checkEnumValueNoDeleteWithRules(add addFunc, previousEnum protodesc.Enum, enum protodesc.Enum, allowIfNumberReserved bool, allowIfNameReserved bool) error {
	previousNumberToNameToEnumValue, err := protodesc.NumberToNameToEnumValue(previousEnum)
	if err != nil {
		return err
	}
	numberToNameToEnumValue, err := protodesc.NumberToNameToEnumValue(enum)
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
					suffix = fmt.Sprintf(` without reserving the name%s %s`, nameSuffix, utilstring.JoinSliceQuoted(getSortedEnumValueNames(previousNameToEnumValue), ", "))
				}
				add(enum, enum.Location(), `Previously present enum value "%d" on enum %q was deleted%s.`, previousNumber, enum.Name(), suffix)
			}
		}
	}
	return nil
}

func isDeletedEnumValueAllowedWithRules(previousNumber int, previousNameToEnumValue map[string]protodesc.EnumValue, enum protodesc.Enum, allowIfNumberReserved bool, allowIfNameReserved bool) bool {
	if allowIfNumberReserved {
		return protodesc.NumberInReservedRanges(previousNumber, enum.ReservedRanges()...)
	}
	if allowIfNameReserved {
		// if true for all names, then ok
		for previousName := range previousNameToEnumValue {
			if !protodesc.NameInReservedNames(previousName, enum.ReservedNames()...) {
				return false
			}
		}
		return true
	}
	return false
}

// CheckEnumValueSameName is a check function.
var CheckEnumValueSameName = newEnumValuePairCheckFunc(checkEnumValueSameName)

func checkEnumValueSameName(add addFunc, previousNameToEnumValue map[string]protodesc.EnumValue, nameToEnumValue map[string]protodesc.EnumValue) error {
	previousNames := getSortedEnumValueNames(previousNameToEnumValue)
	names := getSortedEnumValueNames(nameToEnumValue)
	if !utilstring.SliceElementsEqual(previousNames, names) {
		previousNamesString := utilstring.JoinSliceQuoted(previousNames, ", ")
		namesString := utilstring.JoinSliceQuoted(names, ", ")
		nameSuffix := ""
		if len(previousNames) > 1 && len(names) > 1 {
			nameSuffix = "s"
		}
		for _, enumValue := range nameToEnumValue {
			add(enumValue, enumValue.NumberLocation(), `Enum value "%d" on enum %q changed name%s from %s to %s.`, enumValue.Number(), enumValue.Enum().Name(), nameSuffix, previousNamesString, namesString)
		}
	}
	return nil
}

// CheckExtensionMessageNoDelete is a check function.
var CheckExtensionMessageNoDelete = newMessagePairCheckFunc(checkExtensionMessageNoDelete)

func checkExtensionMessageNoDelete(add addFunc, previousMessage protodesc.Message, message protodesc.Message) error {
	previousStringToExtensionRange := protodesc.StringToExtensionRange(previousMessage)
	stringToExtensionRange := protodesc.StringToExtensionRange(message)
	for previousString := range previousStringToExtensionRange {
		if _, ok := stringToExtensionRange[previousString]; !ok {
			add(message, message.Location(), `Previously present extension range %q on message %q was deleted.`, previousString, message.Name())
		}
	}
	return nil
}

// CheckFieldNoDelete is a check function.
var CheckFieldNoDelete = newMessagePairCheckFunc(checkFieldNoDelete)

func checkFieldNoDelete(add addFunc, previousMessage protodesc.Message, message protodesc.Message) error {
	return checkFieldNoDeleteWithRules(add, previousMessage, message, false, false)
}

// CheckFieldNoDeleteUnlessNumberReserved is a check function.
var CheckFieldNoDeleteUnlessNumberReserved = newMessagePairCheckFunc(checkFieldNoDeleteUnlessNumberReserved)

func checkFieldNoDeleteUnlessNumberReserved(add addFunc, previousMessage protodesc.Message, message protodesc.Message) error {
	return checkFieldNoDeleteWithRules(add, previousMessage, message, true, false)
}

// CheckFieldNoDeleteUnlessNameReserved is a check function.
var CheckFieldNoDeleteUnlessNameReserved = newMessagePairCheckFunc(checkFieldNoDeleteUnlessNameReserved)

func checkFieldNoDeleteUnlessNameReserved(add addFunc, previousMessage protodesc.Message, message protodesc.Message) error {
	return checkFieldNoDeleteWithRules(add, previousMessage, message, false, true)
}

func checkFieldNoDeleteWithRules(add addFunc, previousMessage protodesc.Message, message protodesc.Message, allowIfNumberReserved bool, allowIfNameReserved bool) error {
	previousNumberToField, err := protodesc.NumberToMessageField(previousMessage)
	if err != nil {
		return err
	}
	numberToField, err := protodesc.NumberToMessageField(message)
	if err != nil {
		return err
	}
	for previousNumber, previousField := range previousNumberToField {
		if _, ok := numberToField[previousNumber]; !ok {
			if !isDeletedFieldAllowedWithRules(previousField, message, allowIfNumberReserved, allowIfNameReserved) {
				// otherwise prints as hex
				previousNumberString := strconv.FormatInt(int64(previousNumber), 10)
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
				add(message, message.Location(), `Previously present field %q with name %q on message %q was deleted%s.`, previousNumberString, previousField.Name(), message.Name(), suffix)
			}
		}
	}
	return nil
}

func isDeletedFieldAllowedWithRules(previousField protodesc.Field, message protodesc.Message, allowIfNumberReserved bool, allowIfNameReserved bool) bool {
	return (allowIfNumberReserved && protodesc.NumberInReservedRanges(previousField.Number(), message.ReservedRanges()...)) ||
		(allowIfNameReserved && protodesc.NameInReservedNames(previousField.Name(), message.ReservedNames()...))
}

// CheckFieldSameCType is a check function.
var CheckFieldSameCType = newFieldPairCheckFunc(checkFieldSameCType)

func checkFieldSameCType(add addFunc, previousField protodesc.Field, field protodesc.Field) error {
	if previousField.CType() != field.CType() {
		// otherwise prints as hex
		numberString := strconv.FormatInt(int64(field.Number()), 10)
		add(field, withBackupLocation(field.CTypeLocation(), field.Location()), `Field %q with name %q on message %q changed option "ctype" from %q to %q.`, numberString, field.Name(), field.Message().Name(), previousField.CType().String(), field.CType().String())
	}
	return nil
}

// CheckFieldSameJSONName is a check function.
var CheckFieldSameJSONName = newFieldPairCheckFunc(checkFieldSameJSONName)

func checkFieldSameJSONName(add addFunc, previousField protodesc.Field, field protodesc.Field) error {
	if previousField.JSONName() != field.JSONName() {
		// otherwise prints as hex
		numberString := strconv.FormatInt(int64(field.Number()), 10)
		add(field, withBackupLocation(field.JSONNameLocation(), field.Location()), `Field %q with name %q on message %q changed option "json_name" from %q to %q.`, numberString, field.Name(), field.Message().Name(), previousField.JSONName(), field.JSONName())
	}
	return nil
}

// CheckFieldSameJSType is a check function.
var CheckFieldSameJSType = newFieldPairCheckFunc(checkFieldSameJSType)

func checkFieldSameJSType(add addFunc, previousField protodesc.Field, field protodesc.Field) error {
	if previousField.JSType() != field.JSType() {
		// otherwise prints as hex
		numberString := strconv.FormatInt(int64(field.Number()), 10)
		add(field, withBackupLocation(field.JSTypeLocation(), field.Location()), `Field %q with name %q on message %q changed option "jstype" from %q to %q.`, numberString, field.Name(), field.Message().Name(), previousField.JSType().String(), field.JSType().String())
	}
	return nil
}

// CheckFieldSameLabel is a check function.
var CheckFieldSameLabel = newFieldPairCheckFunc(checkFieldSameLabel)

func checkFieldSameLabel(add addFunc, previousField protodesc.Field, field protodesc.Field) error {
	if previousField.Label() != field.Label() {
		// otherwise prints as hex
		numberString := strconv.FormatInt(int64(field.Number()), 10)
		// TODO: specific label location
		add(field, field.Location(), `Field %q on message %q changed label from %q to %q.`, numberString, field.Message().Name(), previousField.Label().String(), field.Label().String())
	}
	return nil
}

// CheckFieldSameName is a check function.
var CheckFieldSameName = newFieldPairCheckFunc(checkFieldSameName)

func checkFieldSameName(add addFunc, previousField protodesc.Field, field protodesc.Field) error {
	if previousField.Name() != field.Name() {
		// otherwise prints as hex
		numberString := strconv.FormatInt(int64(field.Number()), 10)
		add(field, field.NameLocation(), `Field %q on message %q changed name from %q to %q.`, numberString, field.Message().Name(), previousField.Name(), field.Name())
	}
	return nil
}

// CheckFieldSameOneof is a check function.
var CheckFieldSameOneof = newFieldPairCheckFunc(checkFieldSameOneof)

func checkFieldSameOneof(add addFunc, previousField protodesc.Field, field protodesc.Field) error {
	previousOneof, err := protodesc.FieldOneof(previousField)
	if err != nil {
		return err
	}
	oneof, err := protodesc.FieldOneof(field)
	if err != nil {
		return err
	}

	previousInsideOneof := previousOneof != nil
	insideOneof := oneof != nil
	if !previousInsideOneof && !insideOneof {
		return nil
	}
	if previousInsideOneof && insideOneof {
		if previousOneof.Name() != oneof.Name() {
			// otherwise prints as hex
			numberString := strconv.FormatInt(int64(field.Number()), 10)
			add(field, field.Location(), `Field %q on message %q moved from oneof %q to oneof %q.`, numberString, field.Message().Name(), previousOneof.Name(), oneof.Name())
		}
		return nil
	}

	previous := "inside"
	current := "outside"
	if insideOneof {
		previous = "outside"
		current = "inside"
	}
	// otherwise prints as hex
	numberString := strconv.FormatInt(int64(field.Number()), 10)
	add(field, field.Location(), `Field %q on message %q moved from %s to %s a oneof.`, numberString, field.Message().Name(), previous, current)
	return nil
}

// CheckFieldSameType is a check function.
var CheckFieldSameType = newFieldPairCheckFunc(checkFieldSameType)

// TODO: locations not working for map entries
// TODO: weird output for map entries:
//
// breaking_field_same_type/1.proto:1:1:Field "2" on message "SixEntry" changed type from ".a.One" to ".a.Two".
// breaking_field_same_type/1.proto:1:1:Field "2" on message "SixEntry" changed type from ".a.One" to ".a.Two".
// breaking_field_same_type/1.proto:1:1:Field "2" on message "SixEntry" changed type from ".a.One" to ".a.Two".
// breaking_field_same_type/1.proto:8:3:Field "1" on message "Two" changed type from "int32" to "int64".
// breaking_field_same_type/1.proto:9:3:Field "2" on message "Two" changed type from ".a.One" to ".a.Two".
// breaking_field_same_type/1.proto:11:3:Field "4" on message "Two" changed type from ".a.One" to ".a.Two".
// breaking_field_same_type/1.proto:12:3:Field "5" on message "Two" changed type from ".a.Two.FiveEntry" to ".a.Two".
// breaking_field_same_type/1.proto:19:7:Field "1" on message "Five" changed type from "int32" to "int64".
// breaking_field_same_type/1.proto:20:7:Field "2" on message "Five" changed type from ".a.One" to ".a.Two".
// breaking_field_same_type/1.proto:22:7:Field "4" on message "Five" changed type from ".a.One" to ".a.Two".
// breaking_field_same_type/1.proto:23:7:Field "5" on message "Five" changed type from ".a.Three.Four.Five.FiveEntry" to ".a.Two".
// breaking_field_same_type/1.proto:36:5:Field "1" on message "Seven" changed type from "int32" to "int64".
// breaking_field_same_type/1.proto:37:5:Field "2" on message "Seven" changed type from ".a.One" to ".a.Two".
// breaking_field_same_type/1.proto:39:5:Field "4" on message "Seven" changed type from ".a.One" to ".a.Two".
// breaking_field_same_type/1.proto:40:5:Field "5" on message "Seven" changed type from ".a.Three.Seven.FiveEntry" to ".a.Two".
// breaking_field_same_type/2.proto:64:5:Field "1" on message "Nine" changed type from "int32" to "int64".
// breaking_field_same_type/2.proto:65:5:Field "2" on message "Nine" changed type from ".a.One" to ".a.Nine".
func checkFieldSameType(add addFunc, previousField protodesc.Field, field protodesc.Field) error {
	if previousField.Type() != field.Type() {
		// otherwise prints as hex
		previousNumberString := strconv.FormatInt(int64(previousField.Number()), 10)
		add(field, field.TypeLocation(), `Field %q on message %q changed type from %q to %q.`, previousNumberString, field.Message().Name(), previousField.Type().String(), field.Type().String())
		return nil
	}

	switch field.Type() {
	case protodesc.FieldDescriptorProtoTypeEnum, protodesc.FieldDescriptorProtoTypeGroup, protodesc.FieldDescriptorProtoTypeMessage:
		// otherwise prints as hex
		numberString := strconv.FormatInt(int64(previousField.Number()), 10)
		if previousField.TypeName() != field.TypeName() {
			add(
				field,
				field.TypeNameLocation(),
				`Field %q on message %q changed type from %q to %q.`,
				numberString,
				field.Message().Name(),
				strings.TrimPrefix(previousField.TypeName(), "."),
				strings.TrimPrefix(field.TypeName(), "."),
			)
		}
	}
	return nil
}

// CheckFileNoDelete is a check function.
var CheckFileNoDelete = newFilesCheckFunc(checkFileNoDelete)

func checkFileNoDelete(add addFunc, previousFiles []protodesc.File, files []protodesc.File) error {
	previousFilePathToFile, err := protodesc.FilePathToFile(previousFiles...)
	if err != nil {
		return err
	}
	filePathToFile, err := protodesc.FilePathToFile(files...)
	if err != nil {
		return err
	}
	for previousFilePath := range previousFilePathToFile {
		if _, ok := filePathToFile[previousFilePath]; !ok {
			add(nil, nil, `Previously present file %q was deleted.`, previousFilePath)
		}
	}
	return nil
}

// CheckFileSameCsharpNamespace is a check function.
var CheckFileSameCsharpNamespace = newFilePairCheckFunc(checkFileSameCsharpNamespace)

func checkFileSameCsharpNamespace(add addFunc, previousFile protodesc.File, file protodesc.File) error {
	return checkFileSameValue(add, previousFile.CsharpNamespace(), file.CsharpNamespace(), file, file.CsharpNamespaceLocation(), `option "csharp_namespace"`)
}

// CheckFileSameGoPackage is a check function.
var CheckFileSameGoPackage = newFilePairCheckFunc(checkFileSameGoPackage)

func checkFileSameGoPackage(add addFunc, previousFile protodesc.File, file protodesc.File) error {
	return checkFileSameValue(add, previousFile.GoPackage(), file.GoPackage(), file, file.GoPackageLocation(), `option "go_package"`)
}

// CheckFileSameJavaMultipleFiles is a check function.
var CheckFileSameJavaMultipleFiles = newFilePairCheckFunc(checkFileSameJavaMultipleFiles)

func checkFileSameJavaMultipleFiles(add addFunc, previousFile protodesc.File, file protodesc.File) error {
	return checkFileSameValue(add, strconv.FormatBool(previousFile.JavaMultipleFiles()), strconv.FormatBool(file.JavaMultipleFiles()), file, file.JavaMultipleFilesLocation(), `option "java_multiple_files"`)
}

// CheckFileSameJavaOuterClassname is a check function.
var CheckFileSameJavaOuterClassname = newFilePairCheckFunc(checkFileSameJavaOuterClassname)

func checkFileSameJavaOuterClassname(add addFunc, previousFile protodesc.File, file protodesc.File) error {
	return checkFileSameValue(add, previousFile.JavaOuterClassname(), file.JavaOuterClassname(), file, file.JavaOuterClassnameLocation(), `option "java_outer_classname"`)
}

// CheckFileSameJavaPackage is a check function.
var CheckFileSameJavaPackage = newFilePairCheckFunc(checkFileSameJavaPackage)

func checkFileSameJavaPackage(add addFunc, previousFile protodesc.File, file protodesc.File) error {
	return checkFileSameValue(add, previousFile.JavaPackage(), file.JavaPackage(), file, file.JavaPackageLocation(), `option "java_package"`)
}

// CheckFileSameJavaStringCheckUtf8 is a check function.
var CheckFileSameJavaStringCheckUtf8 = newFilePairCheckFunc(checkFileSameJavaStringCheckUtf8)

func checkFileSameJavaStringCheckUtf8(add addFunc, previousFile protodesc.File, file protodesc.File) error {
	return checkFileSameValue(add, strconv.FormatBool(previousFile.JavaStringCheckUtf8()), strconv.FormatBool(file.JavaStringCheckUtf8()), file, file.JavaStringCheckUtf8Location(), `option "java_string_check_utf8"`)
}

// CheckFileSameObjcClassPrefix is a check function.
var CheckFileSameObjcClassPrefix = newFilePairCheckFunc(checkFileSameObjcClassPrefix)

func checkFileSameObjcClassPrefix(add addFunc, previousFile protodesc.File, file protodesc.File) error {
	return checkFileSameValue(add, previousFile.ObjcClassPrefix(), file.ObjcClassPrefix(), file, file.ObjcClassPrefixLocation(), `option "objc_class_prefix"`)
}

// CheckFileSamePackage is a check function.
var CheckFileSamePackage = newFilePairCheckFunc(checkFileSamePackage)

func checkFileSamePackage(add addFunc, previousFile protodesc.File, file protodesc.File) error {
	return checkFileSameValue(add, previousFile.Package(), file.Package(), file, file.PackageLocation(), `package`)
}

// CheckFileSamePhpClassPrefix is a check function.
var CheckFileSamePhpClassPrefix = newFilePairCheckFunc(checkFileSamePhpClassPrefix)

func checkFileSamePhpClassPrefix(add addFunc, previousFile protodesc.File, file protodesc.File) error {
	return checkFileSameValue(add, previousFile.PhpClassPrefix(), file.PhpClassPrefix(), file, file.PhpClassPrefixLocation(), `option "php_class_prefix"`)
}

// CheckFileSamePhpNamespace is a check function.
var CheckFileSamePhpNamespace = newFilePairCheckFunc(checkFileSamePhpNamespace)

func checkFileSamePhpNamespace(add addFunc, previousFile protodesc.File, file protodesc.File) error {
	return checkFileSameValue(add, previousFile.PhpNamespace(), file.PhpNamespace(), file, file.PhpNamespaceLocation(), `option "php_namespace"`)
}

// CheckFileSamePhpMetadataNamespace is a check function.
var CheckFileSamePhpMetadataNamespace = newFilePairCheckFunc(checkFileSamePhpMetadataNamespace)

func checkFileSamePhpMetadataNamespace(add addFunc, previousFile protodesc.File, file protodesc.File) error {
	return checkFileSameValue(add, previousFile.PhpMetadataNamespace(), file.PhpMetadataNamespace(), file, file.PhpMetadataNamespaceLocation(), `option "php_metadata_namespace"`)
}

// CheckFileSameRubyPackage is a check function.
var CheckFileSameRubyPackage = newFilePairCheckFunc(checkFileSameRubyPackage)

func checkFileSameRubyPackage(add addFunc, previousFile protodesc.File, file protodesc.File) error {
	return checkFileSameValue(add, previousFile.RubyPackage(), file.RubyPackage(), file, file.RubyPackageLocation(), `option "ruby_package"`)
}

// CheckFileSameSwiftPrefix is a check function.
var CheckFileSameSwiftPrefix = newFilePairCheckFunc(checkFileSameSwiftPrefix)

func checkFileSameSwiftPrefix(add addFunc, previousFile protodesc.File, file protodesc.File) error {
	return checkFileSameValue(add, previousFile.SwiftPrefix(), file.SwiftPrefix(), file, file.SwiftPrefixLocation(), `option "swift_prefix"`)
}

// CheckFileSameOptimizeFor is a check function.
var CheckFileSameOptimizeFor = newFilePairCheckFunc(checkFileSameOptimizeFor)

func checkFileSameOptimizeFor(add addFunc, previousFile protodesc.File, file protodesc.File) error {
	return checkFileSameValue(add, previousFile.OptimizeFor().String(), file.OptimizeFor().String(), file, file.OptimizeForLocation(), `option "optimize_for"`)
}

// CheckFileSameCcGenericServices is a check function.
var CheckFileSameCcGenericServices = newFilePairCheckFunc(checkFileSameCcGenericServices)

func checkFileSameCcGenericServices(add addFunc, previousFile protodesc.File, file protodesc.File) error {
	return checkFileSameValue(add, strconv.FormatBool(previousFile.CcGenericServices()), strconv.FormatBool(file.CcGenericServices()), file, file.CcGenericServicesLocation(), `option "cc_generic_services"`)
}

// CheckFileSameJavaGenericServices is a check function.
var CheckFileSameJavaGenericServices = newFilePairCheckFunc(checkFileSameJavaGenericServices)

func checkFileSameJavaGenericServices(add addFunc, previousFile protodesc.File, file protodesc.File) error {
	return checkFileSameValue(add, strconv.FormatBool(previousFile.JavaGenericServices()), strconv.FormatBool(file.JavaGenericServices()), file, file.JavaGenericServicesLocation(), `option "java_generic_services"`)
}

// CheckFileSamePyGenericServices is a check function.
var CheckFileSamePyGenericServices = newFilePairCheckFunc(checkFileSamePyGenericServices)

func checkFileSamePyGenericServices(add addFunc, previousFile protodesc.File, file protodesc.File) error {
	return checkFileSameValue(add, strconv.FormatBool(previousFile.PyGenericServices()), strconv.FormatBool(file.PyGenericServices()), file, file.PyGenericServicesLocation(), `option "py_generic_services"`)
}

// CheckFileSamePhpGenericServices is a check function.
var CheckFileSamePhpGenericServices = newFilePairCheckFunc(checkFileSamePhpGenericServices)

func checkFileSamePhpGenericServices(add addFunc, previousFile protodesc.File, file protodesc.File) error {
	return checkFileSameValue(add, strconv.FormatBool(previousFile.PhpGenericServices()), strconv.FormatBool(file.PhpGenericServices()), file, file.PhpGenericServicesLocation(), `option "php_generic_services"`)
}

// CheckFileSameCcEnableArenas is a check function.
var CheckFileSameCcEnableArenas = newFilePairCheckFunc(checkFileSameCcEnableArenas)

func checkFileSameCcEnableArenas(add addFunc, previousFile protodesc.File, file protodesc.File) error {
	return checkFileSameValue(add, strconv.FormatBool(previousFile.CcEnableArenas()), strconv.FormatBool(file.CcEnableArenas()), file, file.CcEnableArenasLocation(), `option "cc_enable_arenas"`)
}

// CheckFileSameSyntax is a check function.
var CheckFileSameSyntax = newFilePairCheckFunc(checkFileSameSyntax)

func checkFileSameSyntax(add addFunc, previousFile protodesc.File, file protodesc.File) error {
	return checkFileSameValue(add, previousFile.Syntax().String(), file.Syntax().String(), file, file.SyntaxLocation(), `syntax`)
}

func checkFileSameValue(add addFunc, previousValue interface{}, value interface{}, file protodesc.File, location protodesc.Location, name string) error {
	if previousValue != value {
		add(file, location, `File %s changed from %q to %q.`, name, previousValue, value)
	}
	return nil
}

// CheckMessageNoDelete is a check function.
var CheckMessageNoDelete = newFilePairCheckFunc(checkMessageNoDelete)

func checkMessageNoDelete(add addFunc, previousFile protodesc.File, file protodesc.File) error {
	previousNestedNameToMessage, err := protodesc.NestedNameToMessage(previousFile)
	if err != nil {
		return err
	}
	nestedNameToMessage, err := protodesc.NestedNameToMessage(file)
	if err != nil {
		return err
	}
	for previousNestedName := range previousNestedNameToMessage {
		if _, ok := nestedNameToMessage[previousNestedName]; !ok {
			descriptor, location := getDescriptorAndLocationForDeletedMessage(file, nestedNameToMessage, previousNestedName)
			add(descriptor, location, `Previously present message %q was deleted from file.`, previousNestedName)
		}
	}
	return nil
}

// CheckMessageNoRemoveStandardDescriptorAccessor is a check function.
var CheckMessageNoRemoveStandardDescriptorAccessor = newMessagePairCheckFunc(checkMessageNoRemoveStandardDescriptorAccessor)

func checkMessageNoRemoveStandardDescriptorAccessor(add addFunc, previousMessage protodesc.Message, message protodesc.Message) error {
	previous := strconv.FormatBool(previousMessage.NoStandardDescriptorAccessor())
	current := strconv.FormatBool(message.NoStandardDescriptorAccessor())
	if previous == "false" && current == "true" {
		add(message, message.NoStandardDescriptorAccessorLocation(), `Message option "no_standard_descriptor_accessor" changed from %q to %q.`, previous, current)
	}
	return nil
}

// CheckMessageSameMessageSetWireFormat is a check function.
var CheckMessageSameMessageSetWireFormat = newMessagePairCheckFunc(checkMessageSameMessageSetWireFormat)

func checkMessageSameMessageSetWireFormat(add addFunc, previousMessage protodesc.Message, message protodesc.Message) error {
	previous := strconv.FormatBool(previousMessage.MessageSetWireFormat())
	current := strconv.FormatBool(message.MessageSetWireFormat())
	if previous != current {
		add(message, message.MessageSetWireFormatLocation(), `Message option "message_set_wire_format" changed from %q to %q.`, previous, current)
	}
	return nil
}

// CheckOneofNoDelete is a check function.
var CheckOneofNoDelete = newMessagePairCheckFunc(checkOneofNoDelete)

func checkOneofNoDelete(add addFunc, previousMessage protodesc.Message, message protodesc.Message) error {
	previousNameToOneof, err := protodesc.NameToMessageOneof(previousMessage)
	if err != nil {
		return err
	}
	nameToOneof, err := protodesc.NameToMessageOneof(message)
	if err != nil {
		return err
	}
	for previousName := range previousNameToOneof {
		if _, ok := nameToOneof[previousName]; !ok {
			add(message, message.Location(), `Previously present oneof %q on message %q was deleted.`, previousName, message.Name())
		}
	}
	return nil
}

// CheckPackageEnumNoDelete is a check function.
var CheckPackageEnumNoDelete = newFilesCheckFunc(checkPackageEnumNoDelete)

func checkPackageEnumNoDelete(add addFunc, previousFiles []protodesc.File, files []protodesc.File) error {
	previousPackageToNestedNameToEnum, err := protodesc.PackageToNestedNameToEnum(previousFiles...)
	if err != nil {
		return err
	}
	packageToNestedNameToEnum, err := protodesc.PackageToNestedNameToEnum(files...)
	if err != nil {
		return err
	}
	// caching across loops
	var filePathToFile map[string]protodesc.File
	for previousPackage, previousNestedNameToEnum := range previousPackageToNestedNameToEnum {
		if nestedNameToEnum, ok := packageToNestedNameToEnum[previousPackage]; ok {
			for previousNestedName, previousEnum := range previousNestedNameToEnum {
				if _, ok := nestedNameToEnum[previousNestedName]; !ok {
					// if cache not populated, populate it
					if filePathToFile == nil {
						filePathToFile, err = protodesc.FilePathToFile(files...)
						if err != nil {
							return err
						}
					}
					// check if the file still exists
					file, ok := filePathToFile[previousEnum.FilePath()]
					if ok {
						// file exists, try to get a location to attach the error to
						descriptor, location, err := getDescriptorAndLocationForDeletedEnum(file, previousNestedName)
						if err != nil {
							return err
						}
						add(descriptor, location, `Previously present enum %q was deleted from package %q.`, previousNestedName, previousPackage)
					} else {
						// file does not exist, we don't know where the enum was deleted from
						add(nil, nil, `Previously present enum %q was deleted from package %q.`, previousNestedName, previousPackage)
					}
				}
			}
		}
	}
	return nil
}

// CheckPackageMessageNoDelete is a check function.
var CheckPackageMessageNoDelete = newFilesCheckFunc(checkPackageMessageNoDelete)

func checkPackageMessageNoDelete(add addFunc, previousFiles []protodesc.File, files []protodesc.File) error {
	previousPackageToNestedNameToMessage, err := protodesc.PackageToNestedNameToMessage(previousFiles...)
	if err != nil {
		return err
	}
	packageToNestedNameToMessage, err := protodesc.PackageToNestedNameToMessage(files...)
	if err != nil {
		return err
	}
	// caching across loops
	var filePathToFile map[string]protodesc.File
	for previousPackage, previousNestedNameToMessage := range previousPackageToNestedNameToMessage {
		if nestedNameToMessage, ok := packageToNestedNameToMessage[previousPackage]; ok {
			for previousNestedName, previousMessage := range previousNestedNameToMessage {
				if _, ok := nestedNameToMessage[previousNestedName]; !ok {
					// if cache not populated, populate it
					if filePathToFile == nil {
						filePathToFile, err = protodesc.FilePathToFile(files...)
						if err != nil {
							return err
						}
					}
					// check if the file still exists
					file, ok := filePathToFile[previousMessage.FilePath()]
					if ok {
						// file exists, try to get a location to attach the error to
						descriptor, location := getDescriptorAndLocationForDeletedMessage(file, nestedNameToMessage, previousNestedName)
						add(descriptor, location, `Previously present message %q was deleted from package %q.`, previousNestedName, previousPackage)
					} else {
						// file does not exist, we don't know where the message was deleted from
						add(nil, nil, `Previously present message %q was deleted from package %q.`, previousNestedName, previousPackage)
					}
				}
			}
		}
	}
	return nil
}

// CheckPackageNoDelete is a check function.
var CheckPackageNoDelete = newFilesCheckFunc(checkPackageNoDelete)

func checkPackageNoDelete(add addFunc, previousFiles []protodesc.File, files []protodesc.File) error {
	previousPackageToFiles, err := protodesc.PackageToFiles(previousFiles...)
	if err != nil {
		return err
	}
	packageToFiles, err := protodesc.PackageToFiles(files...)
	if err != nil {
		return err
	}
	for previousPackage := range previousPackageToFiles {
		if _, ok := packageToFiles[previousPackage]; !ok {
			add(nil, nil, `Previously present package %q was deleted.`, previousPackage)
		}
	}
	return nil
}

// CheckPackageServiceNoDelete is a check function.
var CheckPackageServiceNoDelete = newFilesCheckFunc(checkPackageServiceNoDelete)

func checkPackageServiceNoDelete(add addFunc, previousFiles []protodesc.File, files []protodesc.File) error {
	previousPackageToNameToService, err := protodesc.PackageToNameToService(previousFiles...)
	if err != nil {
		return err
	}
	packageToNameToService, err := protodesc.PackageToNameToService(files...)
	if err != nil {
		return err
	}
	// caching across loops
	var filePathToFile map[string]protodesc.File
	for previousPackage, previousNameToService := range previousPackageToNameToService {
		if nameToService, ok := packageToNameToService[previousPackage]; ok {
			for previousName, previousService := range previousNameToService {
				if _, ok := nameToService[previousName]; !ok {
					// if cache not populated, populate it
					if filePathToFile == nil {
						filePathToFile, err = protodesc.FilePathToFile(files...)
						if err != nil {
							return err
						}
					}
					// check if the file still exists
					file, ok := filePathToFile[previousService.FilePath()]
					if ok {
						// file exists
						add(file, nil, `Previously present service %q was deleted from package %q.`, previousName, previousPackage)
					} else {
						// file does not exist, we don't know where the service was deleted from
						// TODO: find the service and print that this moved?
						add(nil, nil, `Previously present service %q was deleted from package %q.`, previousName, previousPackage)
					}
				}
			}
		}
	}
	return nil
}

// CheckReservedEnumNoDelete is a check function.
var CheckReservedEnumNoDelete = newEnumPairCheckFunc(checkReservedEnumNoDelete)

func checkReservedEnumNoDelete(add addFunc, previousEnum protodesc.Enum, enum protodesc.Enum) error {
	previousStringToReservedRange := protodesc.StringToReservedRange(previousEnum)
	stringToReservedRange := protodesc.StringToReservedRange(enum)
	for previousString := range previousStringToReservedRange {
		if _, ok := stringToReservedRange[previousString]; !ok {
			add(enum, enum.Location(), `Previously present reserved range %q on enum %q was deleted.`, previousString, enum.Name())
		}
	}
	previousValueToReservedName := protodesc.ValueToReservedName(previousEnum)
	valueToReservedName := protodesc.ValueToReservedName(enum)
	for previousValue := range previousValueToReservedName {
		if _, ok := valueToReservedName[previousValue]; !ok {
			add(enum, enum.Location(), `Previously present reserved name %q on enum %q was deleted.`, previousValue, enum.Name())
		}
	}
	return nil
}

// CheckReservedMessageNoDelete is a check function.
var CheckReservedMessageNoDelete = newMessagePairCheckFunc(checkReservedMessageNoDelete)

func checkReservedMessageNoDelete(add addFunc, previousMessage protodesc.Message, message protodesc.Message) error {
	previousStringToReservedRange := protodesc.StringToReservedRange(previousMessage)
	stringToReservedRange := protodesc.StringToReservedRange(message)
	for previousString := range previousStringToReservedRange {
		if _, ok := stringToReservedRange[previousString]; !ok {
			add(message, message.Location(), `Previously present reserved range %q on message %q was deleted.`, previousString, message.Name())
		}
	}
	previousValueToReservedName := protodesc.ValueToReservedName(previousMessage)
	valueToReservedName := protodesc.ValueToReservedName(message)
	for previousValue := range previousValueToReservedName {
		if _, ok := valueToReservedName[previousValue]; !ok {
			add(message, message.Location(), `Previously present reserved name %q on message %q was deleted.`, previousValue, message.Name())
		}
	}
	return nil
}

// CheckRPCNoDelete is a check function.
var CheckRPCNoDelete = newServicePairCheckFunc(checkRPCNoDelete)

func checkRPCNoDelete(add addFunc, previousService protodesc.Service, service protodesc.Service) error {
	previousNameToMethod, err := protodesc.NameToMethod(previousService)
	if err != nil {
		return err
	}
	nameToMethod, err := protodesc.NameToMethod(service)
	if err != nil {
		return err
	}
	for previousName := range previousNameToMethod {
		if _, ok := nameToMethod[previousName]; !ok {
			add(service, service.Location(), `Previously present RPC %q on service %q was deleted.`, previousName, service.Name())
		}
	}
	return nil
}

// CheckRPCSameClientStreaming is a check function.
var CheckRPCSameClientStreaming = newMethodPairCheckFunc(checkRPCSameClientStreaming)

func checkRPCSameClientStreaming(add addFunc, previousMethod protodesc.Method, method protodesc.Method) error {
	if previousMethod.ClientStreaming() != method.ClientStreaming() {
		previous := "streaming"
		current := "unary"
		if method.ClientStreaming() {
			previous = "unary"
			current = "streaming"
		}
		add(method, method.Location(), `RPC %q on service %q changed from client %s to client %s.`, method.Name(), method.Service().Name(), previous, current)
	}
	return nil
}

// CheckRPCSameIdempotencyLevel is a check function.
var CheckRPCSameIdempotencyLevel = newMethodPairCheckFunc(checkRPCSameIdempotencyLevel)

func checkRPCSameIdempotencyLevel(add addFunc, previousMethod protodesc.Method, method protodesc.Method) error {
	previous := previousMethod.IdempotencyLevel()
	current := method.IdempotencyLevel()
	if previous != current {
		add(method, method.IdempotencyLevelLocation(), `RPC %q on service %q changed option "idempotency_level" from %q to %q.`, method.Name(), method.Service().Name(), previous.String(), current.String())
	}
	return nil
}

// CheckRPCSameRequestType is a check function.
var CheckRPCSameRequestType = newMethodPairCheckFunc(checkRPCSameRequestType)

func checkRPCSameRequestType(add addFunc, previousMethod protodesc.Method, method protodesc.Method) error {
	if previousMethod.InputTypeName() != method.InputTypeName() {
		add(method, method.InputTypeLocation(), `RPC %q on service %q changed request type from %q to %q.`, method.Name(), method.Service().Name(), previousMethod.InputTypeName(), method.InputTypeName())
	}
	return nil
}

// CheckRPCSameResponseType is a check function.
var CheckRPCSameResponseType = newMethodPairCheckFunc(checkRPCSameResponseType)

func checkRPCSameResponseType(add addFunc, previousMethod protodesc.Method, method protodesc.Method) error {
	if previousMethod.OutputTypeName() != method.OutputTypeName() {
		add(method, method.OutputTypeLocation(), `RPC %q on service %q changed response type from %q to %q.`, method.Name(), method.Service().Name(), previousMethod.OutputTypeName(), method.OutputTypeName())
	}
	return nil
}

// CheckRPCSameServerStreaming is a check function.
var CheckRPCSameServerStreaming = newMethodPairCheckFunc(checkRPCSameServerStreaming)

func checkRPCSameServerStreaming(add addFunc, previousMethod protodesc.Method, method protodesc.Method) error {
	if previousMethod.ServerStreaming() != method.ServerStreaming() {
		previous := "streaming"
		current := "unary"
		if method.ServerStreaming() {
			previous = "unary"
			current = "streaming"
		}
		add(method, method.Location(), `RPC %q on service %q changed from server %s to server %s.`, method.Name(), method.Service().Name(), previous, current)
	}
	return nil
}

// CheckServiceNoDelete is a check function.
var CheckServiceNoDelete = newFilePairCheckFunc(checkServiceNoDelete)

func checkServiceNoDelete(add addFunc, previousFile protodesc.File, file protodesc.File) error {
	previousNameToService, err := protodesc.NameToService(previousFile)
	if err != nil {
		return err
	}
	nameToService, err := protodesc.NameToService(file)
	if err != nil {
		return err
	}
	for previousName := range previousNameToService {
		if _, ok := nameToService[previousName]; !ok {
			add(file, nil, `Previously present service %q was deleted from file.`, previousName)
		}
	}
	return nil
}
