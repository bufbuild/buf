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

package bufcheckserverbuild

import (
	"context"

	"buf.build/go/bufplugin/check"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufcheckserver/internal/bufcheckserverhandle"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufcheckserver/internal/bufcheckserverutil"
)

var (
	// BreakingEnumNoDeleteRuleSpecBuilder is a rule spec builder.
	BreakingEnumNoDeleteRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "ENUM_NO_DELETE",
		Purpose: "Checks that enums are not deleted from a given file.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingEnumNoDelete,
	}
	// BreakingEnumSameJSONFormatRuleSpecBuilder is a rule spec builder.
	BreakingEnumSameJSONFormatRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "ENUM_SAME_JSON_FORMAT",
		Purpose: "Checks that enums have the same JSON format support.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingEnumSameJSONFormat,
	}
	// BreakingEnumSameTypeRuleSpecBuilder is a rule spec builder.
	BreakingEnumSameTypeRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "ENUM_SAME_TYPE",
		Purpose: "Checks that enums have the same type (open vs closed).",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingEnumSameType,
	}
	// BreakingEnumValueNoDeleteRuleSpecBuilder is a rule spec builder.
	BreakingEnumValueNoDeleteRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "ENUM_VALUE_NO_DELETE",
		Purpose: "Checks that enum values are not deleted from a given enum.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingEnumValueNoDelete,
	}
	// BreakingEnumValueNoDeleteUnlessNameReservedRuleSpecBuilder is a rule spec builder.
	BreakingEnumValueNoDeleteUnlessNameReservedRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "ENUM_VALUE_NO_DELETE_UNLESS_NAME_RESERVED",
		Purpose: "Checks that enum values are not deleted from a given enum unless the name is reserved.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingEnumValueNoDeleteUnlessNameReserved,
	}
	// BreakingEnumValueNoDeleteUnlessNumberReservedRuleSpecBuilder is a rule spec builder.
	BreakingEnumValueNoDeleteUnlessNumberReservedRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "ENUM_VALUE_NO_DELETE_UNLESS_NUMBER_RESERVED",
		Purpose: "Checks that enum values are not deleted from a given enum unless the number is reserved.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingEnumValueNoDeleteUnlessNumberReserved,
	}
	// BreakingEnumValueSameNameRuleSpecBuilder is a rule spec builder.
	BreakingEnumValueSameNameRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "ENUM_VALUE_SAME_NAME",
		Purpose: "Checks that enum values have the same name.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingEnumValueSameName,
	}
	// BreakingExtensionMessageNoDeleteRuleSpecBuilder is a rule spec builder.
	BreakingExtensionMessageNoDeleteRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "EXTENSION_MESSAGE_NO_DELETE",
		Purpose: "Checks that extension ranges are not deleted from a given message.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingExtensionMessageNoDelete,
	}
	// BreakingExtensionNoDeleteRuleSpecBuilder is a rule spec builder.
	BreakingExtensionNoDeleteRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "EXTENSION_NO_DELETE",
		Purpose: "Checks that extensions are not deleted from a given file.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingExtensionNoDelete,
	}
	// BreakingFieldNoDeleteRuleSpecBuilder is a rule spec builder.
	BreakingFieldNoDeleteRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FIELD_NO_DELETE",
		Purpose: "Checks that fields are not deleted from a given message.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFieldNoDelete,
	}
	// BreakingFieldNoDeleteUnlessNameReservedRuleSpecBuilder is a rule spec builder.
	BreakingFieldNoDeleteUnlessNameReservedRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FIELD_NO_DELETE_UNLESS_NAME_RESERVED",
		Purpose: "Checks that fields are not deleted from a given message unless the name is reserved.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFieldNoDeleteUnlessNameReserved,
	}
	// BreakingFieldNoDeleteUnlessNumberReservedRuleSpecBuilder is a rule spec builder.
	BreakingFieldNoDeleteUnlessNumberReservedRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FIELD_NO_DELETE_UNLESS_NUMBER_RESERVED",
		Purpose: "Checks that fields are not deleted from a given message unless the number is reserved.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFieldNoDeleteUnlessNumberReserved,
	}
	// BreakingFieldSameCardinalityRuleSpecBuilder is a rule spec builder.
	BreakingFieldSameCardinalityRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FIELD_SAME_CARDINALITY",
		Purpose: "Checks that fields have the same cardinalities in a given message.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFieldSameCardinality,
	}
	// BreakingFieldSameCppStringTypeRuleSpecBuilder is a rule spec builder.
	BreakingFieldSameCppStringTypeRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FIELD_SAME_CPP_STRING_TYPE",
		Purpose: "Checks that fields have the same C++ string type, based on ctype field option or (pb.cpp).string_type feature.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFieldSameCppStringType,
	}
	// BreakingFieldSameCTypeRuleSpecBuilder is a rule spec builder.
	BreakingFieldSameCTypeRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:             "FIELD_SAME_CTYPE",
		Purpose:        "Checks that fields have the same value for the ctype option.",
		Deprecated:     true,
		Type:           check.RuleTypeBreaking,
		ReplacementIDs: []string{"FIELD_SAME_CPP_STRING_TYPE"},
		Handler: check.RuleHandlerFunc(
			func(context.Context, check.ResponseWriter, check.Request) error {
				return nil
			},
		),
	}
	// BreakingFieldSameJavaUTF8ValidationRuleSpecBuilder is a rule spec builder.
	BreakingFieldSameJavaUTF8ValidationRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FIELD_SAME_JAVA_UTF8_VALIDATION",
		Purpose: "Checks that fields have the same Java string UTF8 validation, based on java_string_check_utf8 file option or (pb.java).utf8_validation feature.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFieldSameJavaUTF8Validation,
	}
	// BreakingFieldSameDefaultRuleSpecBuilder is a rule spec builder.
	BreakingFieldSameDefaultRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FIELD_SAME_DEFAULT",
		Purpose: "Checks that fields have the same default value, if a default is specified.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFieldSameDefault,
	}
	// BreakingFieldSameJSONNameRuleSpecBuilder is a rule spec builder.
	BreakingFieldSameJSONNameRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FIELD_SAME_JSON_NAME",
		Purpose: "Checks that fields have the same value for the json_name option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFieldSameJSONName,
	}
	// BreakingFieldSameJSTypeRuleSpecBuilder is a rule spec builder.
	BreakingFieldSameJSTypeRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FIELD_SAME_JSTYPE",
		Purpose: "Checks that fields have the same value for the jstype option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFieldSameJSType,
	}
	// BreakingFieldSameLabelRuleSpecBuilder is a rule spec builder.
	BreakingFieldSameLabelRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:         "FIELD_SAME_LABEL",
		Purpose:    "Checks that fields have the same labels in a given message.",
		Deprecated: true,
		Type:       check.RuleTypeBreaking,
		ReplacementIDs: []string{
			"FIELD_SAME_CARDINALITY",
			"FIELD_WIRE_COMPATIBLE_CARDINALITY",
			"FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY",
		},
		Handler: check.RuleHandlerFunc(
			func(context.Context, check.ResponseWriter, check.Request) error {
				return nil
			},
		),
	}
	// FieldSameLabelV1Beta1RuleBuilder is a rule spec builder.
	BreakingFieldSameLabelV1Beta1RuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:             "FIELD_SAME_LABEL",
		Purpose:        "Checks that fields have the same labels in a given message.",
		Deprecated:     true,
		Type:           check.RuleTypeBreaking,
		ReplacementIDs: []string{"FIELD_SAME_CARDINALITY"},
		Handler: check.RuleHandlerFunc(
			func(context.Context, check.ResponseWriter, check.Request) error {
				return nil
			},
		),
	}
	// BreakingFieldSameNameRuleSpecBuilder is a rule spec builder.
	BreakingFieldSameNameRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FIELD_SAME_NAME",
		Purpose: "Checks that fields have the same names in a given message.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFieldSameName,
	}
	// BreakingFieldSameOneofRuleSpecBuilder is a rule spec builder.
	BreakingFieldSameOneofRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FIELD_SAME_ONEOF",
		Purpose: "Checks that fields have the same oneofs in a given message.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFieldSameOneof,
	}
	// BreakingFieldSameUTF8ValidationRuleSpecBuilder is a rule spec builder.
	BreakingFieldSameUTF8ValidationRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FIELD_SAME_UTF8_VALIDATION",
		Purpose: "Checks that string fields have the same UTF8 validation mode.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFieldSameUTF8Validation,
	}
	// BreakingFieldSameTypeRuleSpecBuilder is a rule spec builder.
	BreakingFieldSameTypeRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FIELD_SAME_TYPE",
		Purpose: "Checks that fields have the same types in a given message.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFieldSameType,
	}
	// BreakingFieldWireCompatibleCardinalityRuleSpecBuilder is a rule spec builder.
	BreakingFieldWireCompatibleCardinalityRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FIELD_WIRE_COMPATIBLE_CARDINALITY",
		Purpose: "Checks that fields have wire-compatible cardinalities in a given message.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFieldWireCompatibleCardinality,
	}
	// BreakingFieldWireCompatibleTypeRuleSpecBuilder  is a rule spec builder.
	BreakingFieldWireCompatibleTypeRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FIELD_WIRE_COMPATIBLE_TYPE",
		Purpose: "Checks that fields have wire-compatible types in a given message.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFieldWireCompatibleType,
	}
	// BreakingFieldWireJSONCompatibleCardinalityRuleSpecBuilder is a rule spec builder.
	BreakingFieldWireJSONCompatibleCardinalityRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FIELD_WIRE_JSON_COMPATIBLE_CARDINALITY",
		Purpose: "Checks that fields have wire and JSON compatible cardinalities in a given message.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFieldWireJSONCompatibleCardinality,
	}
	// BreakingFieldWireJSONCompatibleTypeRuleSpecBuilder is a rule spec builder.
	BreakingFieldWireJSONCompatibleTypeRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FIELD_WIRE_JSON_COMPATIBLE_TYPE",
		Purpose: "Checks that fields have wire and JSON compatible types in a given message.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFieldWireJSONCompatibleType,
	}
	// BreakingFileNoDeleteRuleSpecBuilder is a rule spec builder.
	BreakingFileNoDeleteRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_NO_DELETE",
		Purpose: "Checks that files are not deleted.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileNoDelete,
	}
	// BreakingFileSameCsharpNamesapceRuleSpecBuilder is a rule spec builder.
	BreakingFileSameCsharpNamespaceRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_SAME_CSHARP_NAMESPACE",
		Purpose: "Checks that files have the same value for the csharp_namespace option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileSameCsharpNamespace,
	}
	// BreakingFileSameGoPackageRuleSpecBuilder is a rule spec builder.
	BreakingFileSameGoPackageRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_SAME_GO_PACKAGE",
		Purpose: "Checks that files have the same value for the go_package option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileSameGoPackage,
	}
	// BreakingFileSameJavaMultipleFilesRuleSpecBuilder is a rule spec builder.
	BreakingFileSameJavaMultipleFilesRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_SAME_JAVA_MULTIPLE_FILES",
		Purpose: "Checks that files have the same value for the java_multiple_files option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileSameJavaMultipleFiles,
	}
	// BreakingFileSameJavaOuterClassnameRuleSpecBuilder is a rule spec builder.
	BreakingFileSameJavaOuterClassnameRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_SAME_JAVA_OUTER_CLASSNAME",
		Purpose: "Checks that files have the same value for the java_outer_classname option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileSameJavaOuterClassname,
	}
	// BreakingFileSameJavaPackageRuleSpecBuilder is a rule spec builder.
	BreakingFileSameJavaPackageRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_SAME_JAVA_PACKAGE",
		Purpose: "Checks that files have the same value for the java_package option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileSameJavaPackage,
	}
	// BreakingFileSameJavaStringCheckUtf8RuleSpecBuilder is a rule spec builder.
	BreakingFileSameJavaStringCheckUtf8RuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:             "FILE_SAME_JAVA_STRING_CHECK_UTF8",
		Purpose:        "Checks that files have the same value for the java_string_check_utf8 option.",
		Deprecated:     true,
		Type:           check.RuleTypeBreaking,
		ReplacementIDs: []string{"FIELD_SAME_JAVA_UTF8_VALIDATION"},
		Handler: check.RuleHandlerFunc(
			func(context.Context, check.ResponseWriter, check.Request) error {
				return nil
			},
		),
	}
	// BreakingFileSameObjcClassPrefixRuleSpecBuilder is a rule spec builder.
	BreakingFileSameObjcClassPrefixRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_SAME_OBJC_CLASS_PREFIX",
		Purpose: "Checks that files have the same value for the objc_class_prefix option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileSameObjcClassPrefix,
	}
	// BreakingFileSamePackageRuleSpecBuilder is a rule spec builder.
	BreakingFileSamePackageRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_SAME_PACKAGE",
		Purpose: "Checks that files have the same package.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileSamePackage,
	}
	// BreakingFileSamePhpClassPrefixRuleSpecBuilder is a rule spec builder.
	BreakingFileSamePhpClassPrefixRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_SAME_PHP_CLASS_PREFIX",
		Purpose: "Checks that files have the same value for the php_class_prefix option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileSamePhpClassPrefix,
	}
	// BreakingFileSamePhpMetadataNamespaceRuleSpecBuilder is a rule spec builder.
	BreakingFileSamePhpMetadataNamespaceRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_SAME_PHP_METADATA_NAMESPACE",
		Purpose: "Checks that files have the same value for the php_metadata_namespace option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileSamePhpMetadataNamespace,
	}
	// BreakingFileSamePhpNamespaceRuleSpecBuilder is a rule spec builder.
	BreakingFileSamePhpNamespaceRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_SAME_PHP_NAMESPACE",
		Purpose: "Checks that files have the same value for the php_namespace option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileSamePhpNamespace,
	}
	// BreakingFileSameRubyPackageRuleSpecBuilder is a rule spec builder.
	BreakingFileSameRubyPackageRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_SAME_RUBY_PACKAGE",
		Purpose: "Checks that files have the same value for the ruby_package option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileSameRubyPackage,
	}
	// BreakingFileSameSwiftPrefixRuleSpecBuilder is a rule spec builder.
	BreakingFileSameSwiftPrefixRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_SAME_SWIFT_PREFIX",
		Purpose: "Checks that files have the same value for the swift_prefix option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileSameSwiftPrefix,
	}
	// BreakingFileSameOptimizeForRuleSpecBuilder is a rule spec builder.
	BreakingFileSameOptimizeForRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_SAME_OPTIMIZE_FOR",
		Purpose: "Checks that files have the same value for the optimize_for option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileSameOptimizeFor,
	}
	// BreakingFileSameCcGenericServicesRuleSpecBuilder is a rule spec builder.
	BreakingFileSameCcGenericServicesRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_SAME_CC_GENERIC_SERVICES",
		Purpose: "Checks that files have the same value for the cc_generic_services option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileSameCcGenericServices,
	}
	// BreakingFileSameJavaGenericServicesRuleSpecBuilder is a rule spec builder.
	BreakingFileSameJavaGenericServicesRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_SAME_JAVA_GENERIC_SERVICES",
		Purpose: "Checks that files have the same value for the java_generic_services option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileSameJavaGenericServices,
	}
	// BreakingFileSamePyGenericServicesRuleBuilder is a rule spec builder.
	BreakingFileSamePyGenericServicesRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_SAME_PY_GENERIC_SERVICES",
		Purpose: "Checks that files have the same value for the py_generic_services option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileSamePyGenericServices,
	}
	// BreakingFileSamePhpGenericServicesRuleSpecBuilder is a rule spec builder.
	BreakingFileSamePhpGenericServicesRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:         "FILE_SAME_PHP_GENERIC_SERVICES",
		Purpose:    "Checks that files have the same value for the php_generic_services option.",
		Deprecated: true,
		Type:       check.RuleTypeBreaking,
		Handler: check.RuleHandlerFunc(
			func(context.Context, check.ResponseWriter, check.Request) error {
				return nil
			},
		),
	}
	// BreakingFileSameCcEnableArenasRuleSpecBuilder is a rule spec builder.
	BreakingFileSameCcEnableArenasRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_SAME_CC_ENABLE_ARENAS",
		Purpose: "Checks that files have the same value for the cc_enable_arenas option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileSameCcEnableArenas,
	}
	// BreakingFileSameSyntaxRuleSpecBuilder is a rule spec builder.
	BreakingFileSameSyntaxRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_SAME_SYNTAX",
		Purpose: "Checks that files have the same syntax.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileSameSyntax,
	}
	// BreakingMessageNoDeleteRuleSpecBuilder is a rule spec builder.
	BreakingMessageNoDeleteRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "MESSAGE_NO_DELETE",
		Purpose: "Checks that messages are not deleted from a given file.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingMessageNoDelete,
	}
	// BreakingMessageNoRemoveStandardDescriptorAccessorRuleSpecBuilder is a rule spec builder.
	BreakingMessageNoRemoveStandardDescriptorAccessorRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "MESSAGE_NO_REMOVE_STANDARD_DESCRIPTOR_ACCESSOR",
		Purpose: "Checks that messages do not change the no_standard_descriptor_accessor option from false or unset to true.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingMessageNoRemoveStandardDescriptorAccessor,
	}
	// BreakingMessageSameJSONFormatRuleSpecBuilder is a rule spec builder.
	BreakingMessageSameJSONFormatRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "MESSAGE_SAME_JSON_FORMAT",
		Purpose: "Checks that messages have the same JSON format support.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingMessageSameJSONFormat,
	}
	// BreakingMessageSameMessageSetWireFormatRuleSpecBuilder is a rule spec builder.
	//
	// This is deprecated. This rule is now a no-op. We can't do this rule anymore because protobuf-go
	// locked out handling of MessageSets. Because reasons. Grep "ProtoLegacy" in protobuf-go for more details.
	BreakingMessageSameMessageSetWireFormatRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:         "MESSAGE_SAME_MESSAGE_SET_WIRE_FORMAT",
		Purpose:    "Checks that messages have the same value for the message_set_wire_format option.",
		Type:       check.RuleTypeBreaking,
		Deprecated: true,
		Handler: check.RuleHandlerFunc(
			func(context.Context, check.ResponseWriter, check.Request) error {
				return nil
			},
		),
	}
	// BreakingMessageSameRequiredFieldsRuleSpecBuilder is a rule spec builder.
	BreakingMessageSameRequiredFieldsRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "MESSAGE_SAME_REQUIRED_FIELDS",
		Purpose: "Checks that messages have no added or deleted required fields.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingMessageSameRequiredFields,
	}
	// BreakingOneofNoDeleteRuleSpecBuilder is a rule spec builder.
	BreakingOneofNoDeleteRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "ONEOF_NO_DELETE",
		Purpose: "Checks that oneofs are not deleted from a given message.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingOneofNoDelete,
	}
	// BreakingPackageEnumNoDeleteRuleSpecBuilder is a rule spec builder.
	BreakingPackageEnumNoDeleteRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "PACKAGE_ENUM_NO_DELETE",
		Purpose: "Checks that enums are not deleted from a given package.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingPackageEnumNoDelete,
	}
	// BreakingPackageExtensionNoDeleteRuleSpecBuilder is a rule spec builder.
	BreakingPackageExtensionNoDeleteRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "PACKAGE_EXTENSION_NO_DELETE",
		Purpose: "Checks that extensions are not deleted from a given package.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingPackageExtensionNoDelete,
	}
	// BreakingPackageMessageNoDeleteRuleSpecBuilder is a rule spec builder.
	BreakingPackageMessageNoDeleteRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "PACKAGE_MESSAGE_NO_DELETE",
		Purpose: "Checks that messages are not deleted from a given package.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingPackageMessageNoDelete,
	}
	// BreakingPackageNoDeleteRuleSpecBuilder is a rule spec builder.
	BreakingPackageNoDeleteRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "PACKAGE_NO_DELETE",
		Purpose: "Checks that packages are not deleted.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingPackageNoDelete,
	}
	// BreakingPackageServiceNoDeleteRuleSpecBuilder is a rule spec builder.
	BreakingPackageServiceNoDeleteRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "PACKAGE_SERVICE_NO_DELETE",
		Purpose: "Checks that services are not deleted from a given package.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingPackageServiceNoDelete,
	}
	// BreakingReservedEnumNoDeleteRuleSpecBuilder is a rule spec builder.
	BreakingReservedEnumNoDeleteRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "RESERVED_ENUM_NO_DELETE",
		Purpose: "Checks that reserved ranges and names are not deleted from a given enum.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingReservedEnumNoDelete,
	}
	// BreakingReservedMessageNoDeleteRuleSpecBuilder is a rule spec builder.
	BreakingReservedMessageNoDeleteRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "RESERVED_MESSAGE_NO_DELETE",
		Purpose: "Checks that reserved ranges and names are not deleted from a given message.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingReservedMessageNoDelete,
	}
	// BreakingRPCNoDeleteRuleSpecBuilder is a rule spec builder.
	BreakingRPCNoDeleteRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "RPC_NO_DELETE",
		Purpose: "Checks that rpcs are not deleted from a given service.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingRPCNoDelete,
	}
	// BreakingRPCSameClientStreamingRuleSpecBuilder is a rule spec builder.
	BreakingRPCSameClientStreamingRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "RPC_SAME_CLIENT_STREAMING",
		Purpose: "Checks that rpcs have the same client streaming value.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingRPCSameClientStreaming,
	}
	// BreakingRPCSameIdempotencyLevelRuleSpecBuilder is a rule spec builder.
	BreakingRPCSameIdempotencyLevelRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "RPC_SAME_IDEMPOTENCY_LEVEL",
		Purpose: "Checks that rpcs have the same value for the idempotency_level option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingRPCSameIdempotencyLevel,
	}
	// BreakingRPCSameRequestTypeRuleSpecBuilder is a rule spec builder.
	BreakingRPCSameRequestTypeRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "RPC_SAME_REQUEST_TYPE",
		Purpose: "Checks that rpcs are have the same request type.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingRPCSameRequestType,
	}
	// BreakingRPCSameResponseTypeRuleSpecBuilder is a rule spec builder.
	BreakingRPCSameResponseTypeRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "RPC_SAME_RESPONSE_TYPE",
		Purpose: "Checks that rpcs are have the same response type.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingRPCSameResponseType,
	}
	// BreakingRPCSameServerStreamingRuleSpecBuilder is a rule spec builder.
	BreakingRPCSameServerStreamingRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "RPC_SAME_SERVER_STREAMING",
		Purpose: "Checks that rpcs have the same server streaming value.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingRPCSameServerStreaming,
	}
	// BreakingServiceNoDeleteRuleSpecBuilder is a rule spec builder.
	BreakingServiceNoDeleteRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "SERVICE_NO_DELETE",
		Purpose: "Checks that services are not deleted from a given file.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingServiceNoDelete,
	}
	// LintCommentEnumRuleSpecBuilder is a rule spec builder.
	LintCommentEnumRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "COMMENT_ENUM",
		Purpose: "Checks that enums have non-empty comments.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintCommentEnum,
	}
	// LintCommentEnumValueRuleSpecBuilder is a rule spec builder.
	LintCommentEnumValueRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "COMMENT_ENUM_VALUE",
		Purpose: "Checks that enum values have non-empty comments.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintCommentEnumValue,
	}
	// LintCommentFieldRuleSpecBuilder is a rule spec builder.
	LintCommentFieldRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "COMMENT_FIELD",
		Purpose: "Checks that fields have non-empty comments.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintCommentField,
	}
	// LintCommentMessageRuleSpecBuilder is a rule spec builder.
	LintCommentMessageRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "COMMENT_MESSAGE",
		Purpose: "Checks that messages have non-empty comments.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintCommentMessage,
	}
	// LintCommentOneofRuleSpecBuilder is a rule spec builder.
	LintCommentOneofRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "COMMENT_ONEOF",
		Purpose: "Checks that oneofs have non-empty comments.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintCommentOneof,
	}
	// LintCommentRPCRuleSpecBuilder is a rule spec builder.
	LintCommentRPCRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "COMMENT_RPC",
		Purpose: "Checks that RPCs have non-empty comments.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintCommentRPC,
	}
	// LintCommentServiceRuleSpecBuilder is a rule spec builder.
	LintCommentServiceRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "COMMENT_SERVICE",
		Purpose: "Checks that services have non-empty comments.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintCommentService,
	}
	// LintDirectorySamePackageRuleSpecBuilder is a rule spec builder.
	LintDirectorySamePackageRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "DIRECTORY_SAME_PACKAGE",
		Purpose: "Checks that all files in a given directory are in the same package.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintDirectorySamePackage,
	}
	// LintEnumFirstValueZeroRuleSpecBuilder is a rule spec builder.
	LintEnumFirstValueZeroRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "ENUM_FIRST_VALUE_ZERO",
		Purpose: "Checks that all first values of enums have a numeric value of 0.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintEnumFirstValueZero,
	}
	// LintEnumNoAllowAliasRuleSpecBuilder is a rule spec builder.
	LintEnumNoAllowAliasRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "ENUM_NO_ALLOW_ALIAS",
		Purpose: "Checks that enums do not have the allow_alias option set.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintEnumNoAllowAlias,
	}
	// LintEnumPascalCaseRuleSpecBuilder is a rule spec builder.
	LintEnumPascalCaseRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "ENUM_PASCAL_CASE",
		Purpose: "Checks that enums are PascalCase.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintEnumPascalCase,
	}
	// LintEnumValuePrefixRuleSpecBuilder is a rule spec builder.
	LintEnumValuePrefixRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "ENUM_VALUE_PREFIX",
		Purpose: "Checks that enum values are prefixed with ENUM_NAME_UPPER_SNAKE_CASE.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintEnumValuePrefix,
	}
	// LintEnumValueUpperSnakeCaseRuleSpecBuilder is a rule spec builder.
	LintEnumValueUpperSnakeCaseRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "ENUM_VALUE_UPPER_SNAKE_CASE",
		Purpose: "Checks that enum values are UPPER_SNAKE_CASE.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintEnumValueUpperSnakeCase,
	}
	// LintEnumZeroValueSuffixRuleSpecBuilder is a rule spec builder.
	LintEnumZeroValueSuffixRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "ENUM_ZERO_VALUE_SUFFIX",
		Purpose: `Checks that enum zero values have a consistent suffix (configurable, default suffix is "_UNSPECIFIED").`,
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintEnumZeroValueSuffix,
	}
	// LintFieldLowerSnakeCaseRuleSpecBuilder is a rule spec builder.
	LintFieldLowerSnakeCaseRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FIELD_LOWER_SNAKE_CASE",
		Purpose: "Checks that field names are lower_snake_case.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintFieldLowerSnakeCase,
	}
	// LintFieldNoDescriptorRuleSpecBuilder is a rule spec builder.
	LintFieldNoDescriptorRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FIELD_NO_DESCRIPTOR",
		Purpose: `Checks that field names are not any capitalization of "descriptor" with any number of prefix or suffix underscores.`,
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintFieldNoDescriptor,
	}
	// LintFieldNotRequiredRuleSpecBuilder is a rule spec builder.
	LintFieldNotRequiredRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FIELD_NOT_REQUIRED",
		Purpose: `Checks that fields are not configured to be required.`,
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintFieldNotRequired,
	}
	// LintFileLowerSnakeCaseRuleSpecBuilder is a rule spec builder.
	LintFileLowerSnakeCaseRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_LOWER_SNAKE_CASE",
		Purpose: "Checks that filenames are lower_snake_case.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintFileLowerSnakeCase,
	}
	// LintImportNoPublicRuleSpecBuilder is a rule spec builder.
	LintImportNoPublicRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "IMPORT_NO_PUBLIC",
		Purpose: "Checks that imports are not public.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintImportNoPublic,
	}
	// LintImportNoWeakRuleSpecBuilder is a rule spec builder.
	LintImportNoWeakRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "IMPORT_NO_WEAK",
		Purpose: "Checks that imports are not weak.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintImportNoWeak,
	}
	// LintImportUsedRuleSpecBuilder is a rule spec builder.
	LintImportUsedRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "IMPORT_USED",
		Purpose: "Checks that imports are used.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintImportUsed,
	}
	// LintMessagePascalCaseRuleSpecBuilder is a rule spec builder.
	LintMessagePascalCaseRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "MESSAGE_PASCAL_CASE",
		Purpose: "Checks that messages are PascalCase.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintMessagePascalCase,
	}
	// LintOneofLowerSnakeCaseRuleSpecBuilder is a rule spec builder.
	LintOneofLowerSnakeCaseRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "ONEOF_LOWER_SNAKE_CASE",
		Purpose: "Checks that oneof names are lower_snake_case.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintOneofLowerSnakeCase,
	}
	// LintPackageDefinedRuleSpecBuilder is a rule spec builder.
	LintPackageDefinedRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "PACKAGE_DEFINED",
		Purpose: "Checks that all files have a package defined.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintPackageDefined,
	}
	// LintPackageDirectoryMatchRuleSpecBuilder is a rule spec builder.
	LintPackageDirectoryMatchRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "PACKAGE_DIRECTORY_MATCH",
		Purpose: "Checks that all files are in a directory that matches their package name.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintPackageDirectoryMatch,
	}
	// LintPackageLowerSnakeCaseRuleSpecBuilder is a rule spec builder.
	LintPackageLowerSnakeCaseRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "PACKAGE_LOWER_SNAKE_CASE",
		Purpose: "Checks that packages are lower_snake.case.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintPackageLowerSnakeCase,
	}
	// LintPackageNoImportCycleRuleSpecBuilder is a rule spec builder.
	LintPackageNoImportCycleRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "PACKAGE_NO_IMPORT_CYCLE",
		Purpose: "Checks that packages do not have import cycles.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintPackageNoImportCycle,
	}
	// LintPackageSameCsharpNamespaceRuleSpecBuilder is a rule spec builder.
	LintPackageSameCsharpNamespaceRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "PACKAGE_SAME_CSHARP_NAMESPACE",
		Purpose: "Checks that all files with a given package have the same value for the csharp_namespace option.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintPackageSameCsharpNamespace,
	}
	// LintPackageSameDirectoryRuleSpecBuilder is a rule spec builder.
	LintPackageSameDirectoryRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "PACKAGE_SAME_DIRECTORY",
		Purpose: "Checks that all files with a given package are in the same directory.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintPackageSameDirectory,
	}
	// LintPackageSameGoPackageRuleSpecBuilder is a rule spec builder.
	LintPackageSameGoPackageRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "PACKAGE_SAME_GO_PACKAGE",
		Purpose: "Checks that all files with a given package have the same value for the go_package option.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintPackageSameGoPackage,
	}
	// LintPackageSameJavaMultipleFilesRuleSpecBuilder is a rule spec builder.
	LintPackageSameJavaMultipleFilesRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "PACKAGE_SAME_JAVA_MULTIPLE_FILES",
		Purpose: "Checks that all files with a given package have the same value for the java_multiple_files option.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintPackageSameJavaMultipleFiles,
	}
	// LintPackageSameJavaPackageRuleSpecBuilder is a rule spec builder.
	LintPackageSameJavaPackageRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "PACKAGE_SAME_JAVA_PACKAGE",
		Purpose: "Checks that all files with a given package have the same value for the java_package option.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintPackageSameJavaPackage,
	}
	// LintPackageSamePhpNamespaceRuleSpecBuilder is a rule spec builder.
	LintPackageSamePhpNamespaceRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "PACKAGE_SAME_PHP_NAMESPACE",
		Purpose: "Checks that all files with a given package have the same value for the php_namespace option.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintPackageSamePhpNamespace,
	}
	// LintPackageSameRubyPackageRuleSpecBuilder is a rule spec builder.
	LintPackageSameRubyPackageRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "PACKAGE_SAME_RUBY_PACKAGE",
		Purpose: "Checks that all files with a given package have the same value for the ruby_package option.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintPackageSameRubyPackage,
	}
	// LintPackageSameSwiftPrefixRuleSpecBuilder is a rule spec builder.
	LintPackageSameSwiftPrefixRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "PACKAGE_SAME_SWIFT_PREFIX",
		Purpose: "Checks that all files with a given package have the same value for the swift_prefix option.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintPackageSameSwiftPrefix,
	}
	// LintPackageVersionSuffixRuleSpecBuilder is a rule spec builder.
	LintPackageVersionSuffixRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "PACKAGE_VERSION_SUFFIX",
		Purpose: `Checks that the last component of all packages is a version of the form v\d+, v\d+test.*, v\d+(alpha|beta)\d+, or v\d+p\d+(alpha|beta)\d+, where numbers are >=1.`,
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintPackageVersionSuffix,
	}
	// LintProtovalidateRuleSpecBuilder is a rule spec builder.
	LintProtovalidateRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "PROTOVALIDATE",
		Purpose: "Checks that protovalidate rules are valid and all CEL expressions compile.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintProtovalidate,
	}
	// LintRPCNoClientStreamingRuleSpecBuilder is a rule spec builder.
	LintRPCNoClientStreamingRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "RPC_NO_CLIENT_STREAMING",
		Purpose: "Checks that RPCs are not client streaming.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintRPCNoClientStreaming,
	}
	// LintRPCNoServerStreamingRuleSpecBuilder is a rule spec builder.
	LintRPCNoServerStreamingRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "RPC_NO_SERVER_STREAMING",
		Purpose: "Checks that RPCs are not server streaming.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintRPCNoServerStreaming,
	}
	// LintRPCPascalCaseRuleSpecBuilder is a rule spec builder.
	LintRPCPascalCaseRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "RPC_PASCAL_CASE",
		Purpose: "Checks that RPCs are PascalCase.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintRPCPascalCase,
	}
	// LintRPCRequestResponseUniqueRuleSpecBuilder is a rule spec builder.
	LintRPCRequestResponseUniqueRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "RPC_REQUEST_RESPONSE_UNIQUE",
		Purpose: "Checks that RPC request and response types are only used in one RPC (configurable).",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintRPCRequestResponseUnique,
	}
	// LintRPCRequestStandardNameRuleSpecBuilder is a rule spec builder.
	LintRPCRequestStandardNameRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "RPC_REQUEST_STANDARD_NAME",
		Purpose: "Checks that RPC request type names are RPCNameRequest or ServiceNameRPCNameRequest (configurable).",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintRPCRequestStandardName,
	}
	// LintRPCResponseStandardNameRuleSpecBuilder is a rule spec builder.
	LintRPCResponseStandardNameRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "RPC_RESPONSE_STANDARD_NAME",
		Purpose: "Checks that RPC response type names are RPCNameResponse or ServiceNameRPCNameResponse (configurable).",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintRPCResponseStandardName,
	}
	// LintServicePascalCaseRuleSpecBuilder is a rule spec builder.
	LintServicePascalCaseRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "SERVICE_PASCAL_CASE",
		Purpose: "Checks that services are PascalCase.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintServicePascalCase,
	}
	// LintServiceSuffixRuleSpecBuilder is a rule spec builder.
	LintServiceSuffixRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "SERVICE_SUFFIX",
		Purpose: `Checks that services have a consistent suffix (configurable, default suffix is "Service").`,
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintServiceSuffix,
	}
	// LintStablePackageNoImportUnstableRuleSpecBuilder is a rule spec builder.
	LintStablePackageNoImportUnstableRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "STABLE_PACKAGE_NO_IMPORT_UNSTABLE",
		Purpose: "Checks that all files that have stable versioned packages do not import packages with unstable version packages.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintStablePackageNoImportUnstable,
	}
	// LintSyntaxSpecifiedRuleSpecBuilder is a rule spec builder.
	LintSyntaxSpecifiedRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "SYNTAX_SPECIFIED",
		Purpose: "Checks that all files have a syntax specified.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintSyntaxSpecified,
	}

	// TODO: Improve purposes. These are in buf.build/docs. Perhaps we can abandon the "checks that" prefix.

	// FileCategorySpec is a category spec.
	FileCategorySpec = &check.CategorySpec{
		ID:      "FILE",
		Purpose: "Checks that there are no source-code breaking changes at the per-file level.",
	}
	// PackageCategorySpec is a category spec.
	PackageCategorySpec = &check.CategorySpec{
		ID:      "PACKAGE",
		Purpose: "Checks that there are no source-code breaking changes at the per-package level.",
	}
	// WireCategorySpec is a category spec.
	WireCategorySpec = &check.CategorySpec{
		ID:      "WIRE",
		Purpose: "Checks that there are no wire breaking changes for the binary encoding.",
	}
	// WireJSONCategorySpec is a category spec.
	WireJSONCategorySpec = &check.CategorySpec{
		ID:      "WIRE_JSON",
		Purpose: "Checks that there are no wire breaking changes for the binary or JSON encodings.",
	}

	// BasicCategorySpec is a category spec.
	BasicCategorySpec = &check.CategorySpec{
		ID:      "BASIC",
		Purpose: "Checks that basic lint rules are followed.",
	}
	// CommentsCategorySpec is a category spec.
	CommentsCategorySpec = &check.CategorySpec{
		ID:      "COMMENTS",
		Purpose: "Checks that all types have comments.",
	}
	// DefaultCategorySpec is a category spec.
	DefaultCategorySpec = &check.CategorySpec{
		ID:             "DEFAULT",
		Purpose:        "Checks that default lint rules are followed.",
		Deprecated:     true,
		ReplacementIDs: []string{"STANDARD"},
	}
	// FileLayoutCategorySpec is a category spec.
	FileLayoutCategorySpec = &check.CategorySpec{
		ID:      "FILE_LAYOUT",
		Purpose: "Checks the file layout.",
	}
	// MinimalCategorySpec is a category spec.
	MinimalCategorySpec = &check.CategorySpec{
		ID:      "MINIMAL",
		Purpose: "Checks that bare-minimum lint rules are followed.",
	}
	// OtherCategorySpec is a category spec.
	OtherCategorySpec = &check.CategorySpec{
		ID:      "OTHER",
		Purpose: "Checks other lint rules.",
	}
	// PackageAffinityCategorySpec is a category spec.
	PackageAffinityCategorySpec = &check.CategorySpec{
		ID:      "PACKAGE_AFFINITY",
		Purpose: "Checks package affinity.",
	}
	// SensibleCategorySpec is a category spec.
	SensibleCategorySpec = &check.CategorySpec{
		ID:      "SENSIBLE",
		Purpose: "Checks sensible lint rules.",
	}
	// StyleBasicCategorySpec is a category spec.
	StyleBasicCategorySpec = &check.CategorySpec{
		ID:      "STYLE_BASIC",
		Purpose: "Checks that basic style lint rules are followed.",
	}
	// StyleDefaultCategorySpec is a category spec.
	StyleDefaultCategorySpec = &check.CategorySpec{
		ID:             "STYLE_DEFAULT",
		Purpose:        "Checks that default style lint rules are followed.",
		Deprecated:     true,
		ReplacementIDs: []string{"STYLE_STANDARD"},
	}
	// StyleStandardCategorySpec is a category spec.
	StyleStandardCategorySpec = &check.CategorySpec{
		ID:      "STYLE_STANDARD",
		Purpose: "Checks that standard style lint rules are followed.",
	}
	// StandardCategorySpec is a category spec.
	StandardCategorySpec = &check.CategorySpec{
		ID:      "STANDARD",
		Purpose: "Checks that standard lint rules are followed.",
	}
	// UnaryRPCCategorySpec is a category spec.
	UnaryRPCCategorySpec = &check.CategorySpec{
		ID:      "UNARY_RPC",
		Purpose: "Checks that all RPCs are unary.",
	}
)
