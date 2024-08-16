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
	"github.com/bufbuild/buf/private/buf/bufcheck/internal/bufcheckserver/internal/bufcheckserverhandle"
	"github.com/bufbuild/buf/private/buf/bufcheck/internal/bufcheckserver/internal/bufcheckserverutil"
	"github.com/bufbuild/bufplugin-go/check"
)

var (
	// BreakingEnumNoDeleteRuleSpecBuilder is a rule spec builder.
	BreakingEnumNoDeleteRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "ENUM_NO_DELETE",
		Purpose: "Checks enums are not deleted from a given file.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingEnumNoDelete,
	}
	// BreakingExtensionNoDeleteRuleSpecBuilder is a rule spec builder.
	BreakingExtensionNoDeleteRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "EXTENSION_NO_DELETE",
		Purpose: "Checks extensions are not deleted from a given file.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingExtensionNoDelete,
	}
	// BreakingFileNoDeleteRuleSpecBuilder is a rule spec builder.
	BreakingFileNoDeleteRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_NO_DELETE",
		Purpose: "Checks files are not deleted.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileNoDelete,
	}
	// BreakingMessageNoDeleteRuleSpecBuilder is a rule spec builder.
	BreakingMessageNoDeleteRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "MESSAGE_NO_DELETE",
		Purpose: "Checks messages are not deleted from a given file.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingMessageNoDelete,
	}
	// BreakingServiceNoDeleteRuleSpecBuilder is a rule spec builder.
	BreakingServiceNoDeleteRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "SERVICE_NO_DELETE",
		Purpose: "Checks services are not deleted from a given file.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingServiceNoDelete,
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
		Purpose: "Check enum values are not deleted from a given enum.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingEnumValueNoDelete,
	}
	// BreakingExtensionMessageNoDeleteRuleSpecBuilder is a rule spec builder.
	BreakingExtensionMessageNoDeleteRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "EXTENSION_MESSAGE_NO_DELETE",
		Purpose: "Checks extension ranges are not deleted from a given message.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingExtensionMessageNoDelete,
	}
	// BreakingFieldNoDeleteRuleSpecBuilder is a rule spec builder.
	BreakingFieldNoDeleteRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FIELD_NO_DELETE",
		Purpose: "Checks fields are not deleted from a given message.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFieldNoDelete,
	}
	// BreakingFieldSameCardinalityRuleSpecBuilder is a rule spec builder.
	BreakingFieldSameCardinalityRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FIELD_SAME_CARDINALITY",
		Purpose: "Checks fields have the same cardinalities in a given message.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFieldSameCardinality,
	}
	// BreakingFieldSameCppStringTypeRuleSpecBuilder is a rule spec builder.
	BreakingFieldSameCppStringTypeRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FIELD_SAME_CPP_STRING_TYPE",
		Purpose: "Checks fields have the same C++ string type, based on ctype field option or (pb.cpp).string_type feature.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFieldSameCppStringType,
	}
	// BreakingFieldSameJavaUTF8ValidationRuleSpecBuilder is a rule spec builder.
	BreakingFieldSameJavaUTF8ValidationRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FIELD_SAME_JAVA_UTF8_VALIDATION",
		Purpose: "Checks fields have the same Java string UTF8 validation, based on java_string_check_utf8 file option or (pb.java).utf8_validation feature.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFieldSameJavaUTF8Validation,
	}
	// BreakingFieldSameJSTypeRuleSpecBuilder is a rule spec builder.
	BreakingFieldSameJSTypeRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FIELD_SAME_JSTYPE",
		Purpose: "Checks fields have the same value for the jstype option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFieldSameJSType,
	}
	// BreakingFieldSameTypeRuleSpecBuilder is a rule spec builder.
	BreakingFieldSameTypeRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FIELD_SAME_TYPE",
		Purpose: "Checks fields have the same types in a given message.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFieldSameType,
	}
	// BreakingFieldSameUTF8ValidationRuleSpecBuilder is a rule spec builder.
	BreakingFieldSameUTF8ValidationRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FIELD_SAME_UTF8_VALIDATION",
		Purpose: "Checks string fields have the same UTF8 validation mode.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFieldSameUTF8Validation,
	}
	// BreakingFileSameCcEnableArenasRuleSpecBuilder is a rule spec builder.
	BreakingFileSameCcEnableArenasRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_SAME_CC_ENABLE_ARENAS",
		Purpose: "Check files have the same value for the cc_enable_arenas option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileSameCcEnableArenas,
	}
	// BreakingFileSameCcGenericServicesRuleSpecBuilder is a rule spec builder.
	BreakingFileSameCcGenericServicesRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_SAME_CC_GENERIC_SERVICES",
		Purpose: "Checks files have the same value for the cc_generic_services option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileSameCcGenericServices,
	}
	// BreakingFileSameCsharpNamesapceRuleSpecBuilder is a rule spec builder.
	BreakingFileSameCsharpNamespaceRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_SAME_CSHARP_NAMESPACE",
		Purpose: "Checks files have the same value for the csharp_namespace option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileSameCsharpNamespace,
	}
	// BreakingFileSameGoPackageRuleSpecBuilder is a rule spec builder.
	BreakingFileSameGoPackageRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_SAME_GO_PACKAGE",
		Purpose: "Checks files have the same value for the go_package option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileSameGoPackage,
	}
	// BreakingFileSameJavaGenericServicesRuleSpecBuilder is a rule spec builder.
	BreakingFileSameJavaGenericServicesRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_SAME_JAVA_GENERIC_SERVICES",
		Purpose: "Checks files have the same value for the java_generic_services option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileSameJavaGenericServices,
	}
	// BreakingFileSameJavaMultipleFilesRuleSpecBuilder is a rule spec builder.
	BreakingFileSameJavaMultipleFilesRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_SAME_JAVA_MULTIPLE_FILES",
		Purpose: "Checks files have the same value for the java_multiple_files option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileSameJavaMultipleFiles,
	}
	// BreakingFileSameJavaOuterClassnameRuleSpecBuilder is a rule spec builder.
	BreakingFileSameJavaOuterClassnameRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_SAME_JAVA_OUTER_CLASSNAME",
		Purpose: "Check files have the same value for the java_outer_classname option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileSameJavaOuterClassname,
	}
	// BreakingFileSameJavaPackageRuleSpecBuilder is a rule spec builder.
	BreakingFileSameJavaPackageRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_SAME_JAVA_PACKAGE",
		Purpose: "Checks files have the same value for the java_package option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileSameJavaPackage,
	}
	// BreakingFileSameObjcClassPrefixRuleSpecBuilder is a rule spec builder.
	BreakingFileSameObjcClassPrefixRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_SAME_OBJC_CLASS_PREFIX",
		Purpose: "Checks files have the same value for the objc_class_prefix option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileSameObjcClassPrefix,
	}
	// BreakingFileSameOptimizeForRuleSpecBuilder is a rule spec builder.
	BreakingFileSameOptimizeForRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_SAME_OPTIMIZE_FOR",
		Purpose: "Checks files have the same value for the optimize_for option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileSameOptimizeFor,
	}
	// BreakingFileSamePhpClassPrefixRuleSpecBuilder is a rule spec builder.
	BreakingFileSamePhpClassPrefixRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_SAME_PHP_CLASS_PREFIX",
		Purpose: "Checks files have the same value for the php_class_prefix option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileSamePhpClassPrefix,
	}
	// BreakingFileSamePhpMetadataNamespaceRuleSpecBuilder is a rule spec builder.
	BreakingFileSamePhpMetadataNamespaceRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_SAME_PHP_METADATA_NAMESPACE",
		Purpose: "Checks files have the same value for the php_metadata_namespace option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileSamePhpMetadataNamespace,
	}
	// BreakingFileSamePhpNamespaceRuleSpecBuilder is a rule spec builder.
	BreakingFileSamePhpNamespaceRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_SAME_PHP_NAMESPACE",
		Purpose: "Checks files have the same value for the php_namespace option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileSamePhpNamespace,
	}
	// BreakingFileSamePyGenericServicesRuleBuilder is a rule spec builder.
	BreakingFileSamePyGenericServicesRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_SAME_PY_GENERIC_SERVICES",
		Purpose: "Checks files have the same value for the py_generic_services option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileSamePyGenericServices,
	}
	// BreakingFileSameRubyPackageRuleSpecBuilder is a rule spec builder.
	BreakingFileSameRubyPackageRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_SAME_RUBY_PACKAGE",
		Purpose: "Checks files have the same value for the ruby_package option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileSameRubyPackage,
	}
	// BreakingFileSameSwiftPrefixRuleSpecBuilder is a rule spec builder.
	BreakingFileSameSwiftPrefixRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_SAME_SWIFT_PREFIX",
		Purpose: "Checks files have the same value for the swift_prefix option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileSameSwiftPrefix,
	}
	// BreakingFileSameSyntaxRuleSpecBuilder is a rule spec builder.
	BreakingFileSameSyntaxRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_SAME_SYNTAX",
		Purpose: "Checks files have the same syntax.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileSameSyntax,
	}
	// BreakingMessageNoRemoveStandardDescriptorAccessorRuleSpecBuilder is a rule spec builder.
	BreakingMessageNoRemoveStandardDescriptorAccessorRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "MESSAGE_NO_REMOVE_STANDARD_DESCRIPTOR_ACCESSOR",
		Purpose: "Checks messages do not change the no_standard_descriptor_accessor option from false or unset to true.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingMessageNoRemoveStandardDescriptorAccessor,
	}
	// BreakingOneofNoDeleteRuleSpecBuilder is a rule spec builder.
	BreakingOneofNoDeleteRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "ONEOF_NO_DELETE",
		Purpose: "Checks oneofs are not deleted from a given message.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingOneofNoDelete,
	}
	// BreakingRPCNoDeleteRuleSpecBuilder is a rule spec builder.
	BreakingRPCNoDeleteRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "RPC_NO_DELETE",
		Purpose: "Checks rpcs are not deleted from a given service.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingRPCNoDelete,
	}
	// BreakingEnumSameJSONFormatRuleSpecBuilder is a rule spec builder.
	BreakingEnumSameJSONFormatRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "ENUM_SAME_JSON_FORMAT",
		Purpose: "Checks enums have the same JSON format support.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingEnumSameJSONFormat,
	}
	// BreakingEnumValueSameNameRuleSpecBuilder is a rule spec builder.
	BreakingEnumValueSameNameRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "ENUM_VALUE_SAME_NAME",
		Purpose: "Checks enum values have the same name.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingEnumValueSameName,
	}
	// BreakingFieldSameJSONNameRuleSpecBuilder is a rule spec builder.
	BreakingFieldSameJSONNameRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FIELD_SAME_JSON_NAME",
		Purpose: "Checks fields have the same value for the json_name option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFieldSameJSONName,
	}
	// BreakingFieldSameNameRuleSpecBuilder is a rule spec builder.
	BreakingFieldSameNameRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FIELD_SAME_NAME",
		Purpose: "Checks fields have the same names in a given message.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFieldSameName,
	}
	// BreakingMessageSameJSONFormatRuleSpecBuilder is a rule spec builder.
	BreakingMessageSameJSONFormatRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "MESSAGE_SAME_JSON_FORMAT",
		Purpose: "Checks messages have the same JSON format support.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingMessageSameJSONFormat,
	}
	// BreakingFieldSameDefaultRuleSpecBuilder is a rule spec builder.
	BreakingFieldSameDefaultRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FIELD_SAME_DEFAULT",
		Purpose: "Checks fields have the same default value, if a default is specified.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFieldSameDefault,
	}
	// BreakingFieldSameOneofRuleSpecBuilder is a rule spec builder.
	BreakingFieldSameOneofRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FIELD_SAME_ONEOF",
		Purpose: "Checks fields have the same oneofs in a given message.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFieldSameOneof,
	}
	// BreakingFileSamePackageRuleSpecBuilder is a rule spec builder.
	BreakingFileSamePackageRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_SAME_PACKAGE",
		Purpose: "Checks files have the same package.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileSamePackage,
	}
	// BreakingMessageSameMessageSetWireFormatRuleSpecBuilder is a rule spec builder.
	BreakingMessageSameMessageSetWireFormatRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "MESSAGE_SAME_MESSAGE_SET_WIRE_FORMAT",
		Purpose: "Checks messages have the same value for the message_set_wire_format option.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingMessageSameMessageSetWireFormat,
	}
	// BreakingMessageSameRequiredFieldsRuleSpecBuilder is a rule spec builder.
	BreakingMessageSameRequiredFieldsRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "MESSAGE_SAME_REQUIRED_FIELDS",
		Purpose: "Checks messages have no added or deleted required fields.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingMessageSameRequiredFields,
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
)
