// Copyright 2020 Buf Technologies, Inc.
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

// Package bufbreakingbuild contains the CheckerBuilders used by bufbreakingv*.
//
// In the future, we can have multiple versions of a CheckerBuilder here, and then
// include them separately in the bufbreakingv* packages. For example, FieldSameTypeCheckerBuilder
// could be split into FieldSameTypeCheckerBuilder/FieldSameTypeCheckerBuilderV2 which handle
// primitives differently, and we could use the former in v1beta1, and the latter in v1.
package bufbreakingbuild

import (
	"github.com/bufbuild/buf/internal/buf/bufcheck/bufbreaking/internal/bufbreakingcheck"
	"github.com/bufbuild/buf/internal/buf/bufcheck/internal"
)

var (
	// EnumNoDeleteCheckerBuilder is a checker builder.
	EnumNoDeleteCheckerBuilder = internal.NewNopCheckerBuilder(
		"ENUM_NO_DELETE",
		"enums are not deleted from a given file",
		bufbreakingcheck.CheckEnumNoDelete,
	)
	// EnumValueNoDeleteCheckerBuilder is a checker builder.
	EnumValueNoDeleteCheckerBuilder = internal.NewNopCheckerBuilder(
		"ENUM_VALUE_NO_DELETE",
		"enum values are not deleted from a given enum",
		bufbreakingcheck.CheckEnumValueNoDelete,
	)
	// EnumValueNoDeleteUnlessNameReservedCheckerBuilder is a checker builder.
	EnumValueNoDeleteUnlessNameReservedCheckerBuilder = internal.NewNopCheckerBuilder(
		"ENUM_VALUE_NO_DELETE_UNLESS_NAME_RESERVED",
		"enum values are not deleted from a given enum unless the name is reserved",
		bufbreakingcheck.CheckEnumValueNoDeleteUnlessNameReserved,
	)
	// EnumValueNoDeleteUnlessNumberReservedCheckerBuilder is a checker builder.
	EnumValueNoDeleteUnlessNumberReservedCheckerBuilder = internal.NewNopCheckerBuilder(
		"ENUM_VALUE_NO_DELETE_UNLESS_NUMBER_RESERVED",
		"enum values are not deleted from a given enum unless the number is reserved",
		bufbreakingcheck.CheckEnumValueNoDeleteUnlessNumberReserved,
	)
	// EnumValueSameNameCheckerBuilder is a checker builder.
	EnumValueSameNameCheckerBuilder = internal.NewNopCheckerBuilder(
		"ENUM_VALUE_SAME_NAME",
		"enum values have the same name",
		bufbreakingcheck.CheckEnumValueSameName,
	)
	// ExtensionMessageNoDeleteCheckerBuilder is a checker builder.
	ExtensionMessageNoDeleteCheckerBuilder = internal.NewNopCheckerBuilder(
		"EXTENSION_MESSAGE_NO_DELETE",
		"extension ranges are not deleted from a given message",
		bufbreakingcheck.CheckExtensionMessageNoDelete,
	)
	// FieldNoDeleteCheckerBuilder is a checker builder.
	FieldNoDeleteCheckerBuilder = internal.NewNopCheckerBuilder(
		"FIELD_NO_DELETE",
		"fields are not deleted from a given message",
		bufbreakingcheck.CheckFieldNoDelete,
	)
	// FieldNoDeleteUnlessNameReservedCheckerBuilder is a checker builder.
	FieldNoDeleteUnlessNameReservedCheckerBuilder = internal.NewNopCheckerBuilder(
		"FIELD_NO_DELETE_UNLESS_NAME_RESERVED",
		"fields are not deleted from a given message unless the name is reserved",
		bufbreakingcheck.CheckFieldNoDeleteUnlessNameReserved,
	)
	// FieldNoDeleteUnlessNumberReservedCheckerBuilder is a checker builder.
	FieldNoDeleteUnlessNumberReservedCheckerBuilder = internal.NewNopCheckerBuilder(
		"FIELD_NO_DELETE_UNLESS_NUMBER_RESERVED",
		"fields are not deleted from a given message unless the number is reserved",
		bufbreakingcheck.CheckFieldNoDeleteUnlessNumberReserved,
	)
	// FieldSameCTypeCheckerBuilder is a checker builder.
	FieldSameCTypeCheckerBuilder = internal.NewNopCheckerBuilder(
		"FIELD_SAME_CTYPE",
		"fields have the same value for the ctype option",
		bufbreakingcheck.CheckFieldSameCType,
	)
	// FieldSameJSONNameCheckerBuilder is a checker builder.
	FieldSameJSONNameCheckerBuilder = internal.NewNopCheckerBuilder(
		"FIELD_SAME_JSON_NAME",
		"fields have the same value for the json_name option",
		bufbreakingcheck.CheckFieldSameJSONName,
	)
	// FieldSameJSTypeCheckerBuilder is a checker builder.
	FieldSameJSTypeCheckerBuilder = internal.NewNopCheckerBuilder(
		"FIELD_SAME_JSTYPE",
		"fields have the same value for the jstype option",
		bufbreakingcheck.CheckFieldSameJSType,
	)
	// FieldSameLabelCheckerBuilder is a checker builder.
	FieldSameLabelCheckerBuilder = internal.NewNopCheckerBuilder(
		"FIELD_SAME_LABEL",
		"fields have the same labels in a given message",
		bufbreakingcheck.CheckFieldSameLabel,
	)
	// FieldSameNameCheckerBuilder is a checker builder.
	FieldSameNameCheckerBuilder = internal.NewNopCheckerBuilder(
		"FIELD_SAME_NAME",
		"fields have the same names in a given message",
		bufbreakingcheck.CheckFieldSameName,
	)
	// FieldSameOneofCheckerBuilder is a checker builder.
	FieldSameOneofCheckerBuilder = internal.NewNopCheckerBuilder(
		"FIELD_SAME_ONEOF",
		"fields have the same oneofs in a given message",
		bufbreakingcheck.CheckFieldSameOneof,
	)
	// FieldSameTypeCheckerBuilder is a checker builder.
	FieldSameTypeCheckerBuilder = internal.NewNopCheckerBuilder(
		"FIELD_SAME_TYPE",
		"fields have the same types in a given message",
		bufbreakingcheck.CheckFieldSameType,
	)
	// FileNoDeleteCheckerBuilder is a checker builder.
	FileNoDeleteCheckerBuilder = internal.NewNopCheckerBuilder(
		"FILE_NO_DELETE",
		"files are not deleted",
		bufbreakingcheck.CheckFileNoDelete,
	)
	// FileSameCsharpNamespaceCheckerBuilder is a checker builder.
	FileSameCsharpNamespaceCheckerBuilder = internal.NewNopCheckerBuilder(
		"FILE_SAME_CSHARP_NAMESPACE",
		"files have the same value for the csharp_namespace option",
		bufbreakingcheck.CheckFileSameCsharpNamespace,
	)
	// FileSameGoPackageCheckerBuilder is a checker builder.
	FileSameGoPackageCheckerBuilder = internal.NewNopCheckerBuilder(
		"FILE_SAME_GO_PACKAGE",
		"files have the same value for the go_package option",
		bufbreakingcheck.CheckFileSameGoPackage,
	)
	// FileSameJavaMultipleFilesCheckerBuilder is a checker builder.
	FileSameJavaMultipleFilesCheckerBuilder = internal.NewNopCheckerBuilder(
		"FILE_SAME_JAVA_MULTIPLE_FILES",
		"files have the same value for the java_multiple_files option",
		bufbreakingcheck.CheckFileSameJavaMultipleFiles,
	)
	// FileSameJavaOuterClassnameCheckerBuilder is a checker builder.
	FileSameJavaOuterClassnameCheckerBuilder = internal.NewNopCheckerBuilder(
		"FILE_SAME_JAVA_OUTER_CLASSNAME",
		"files have the same value for the java_outer_classname option",
		bufbreakingcheck.CheckFileSameJavaOuterClassname,
	)
	// FileSameJavaPackageCheckerBuilder is a checker builder.
	FileSameJavaPackageCheckerBuilder = internal.NewNopCheckerBuilder(
		"FILE_SAME_JAVA_PACKAGE",
		"files have the same value for the java_package option",
		bufbreakingcheck.CheckFileSameJavaPackage,
	)
	// FileSameJavaStringCheckUtf8CheckerBuilder is a checker builder.
	FileSameJavaStringCheckUtf8CheckerBuilder = internal.NewNopCheckerBuilder(
		"FILE_SAME_JAVA_STRING_CHECK_UTF8",
		"files have the same value for the java_string_check_utf8 option",
		bufbreakingcheck.CheckFileSameJavaStringCheckUtf8,
	)
	// FileSameObjcClassPrefixCheckerBuilder is a checker builder.
	FileSameObjcClassPrefixCheckerBuilder = internal.NewNopCheckerBuilder(
		"FILE_SAME_OBJC_CLASS_PREFIX",
		"files have the same value for the objc_class_prefix option",
		bufbreakingcheck.CheckFileSameObjcClassPrefix,
	)
	// FileSamePackageCheckerBuilder is a checker builder.
	FileSamePackageCheckerBuilder = internal.NewNopCheckerBuilder(
		"FILE_SAME_PACKAGE",
		"files have the same package",
		bufbreakingcheck.CheckFileSamePackage,
	)
	// FileSamePhpClassPrefixCheckerBuilder is a checker builder.
	FileSamePhpClassPrefixCheckerBuilder = internal.NewNopCheckerBuilder(
		"FILE_SAME_PHP_CLASS_PREFIX",
		"files have the same value for the php_class_prefix option",
		bufbreakingcheck.CheckFileSamePhpClassPrefix,
	)
	// FileSamePhpMetadataNamespaceCheckerBuilder is a checker builder.
	FileSamePhpMetadataNamespaceCheckerBuilder = internal.NewNopCheckerBuilder(
		"FILE_SAME_PHP_METADATA_NAMESPACE",
		"files have the same value for the php_metadata_namespace option",
		bufbreakingcheck.CheckFileSamePhpMetadataNamespace,
	)
	// FileSamePhpNamespaceCheckerBuilder is a checker builder.
	FileSamePhpNamespaceCheckerBuilder = internal.NewNopCheckerBuilder(
		"FILE_SAME_PHP_NAMESPACE",
		"files have the same value for the php_namespace option",
		bufbreakingcheck.CheckFileSamePhpNamespace,
	)
	// FileSameRubyPackageCheckerBuilder is a checker builder.
	FileSameRubyPackageCheckerBuilder = internal.NewNopCheckerBuilder(
		"FILE_SAME_RUBY_PACKAGE",
		"files have the same value for the ruby_package option",
		bufbreakingcheck.CheckFileSameRubyPackage,
	)
	// FileSameSwiftPrefixCheckerBuilder is a checker builder.
	FileSameSwiftPrefixCheckerBuilder = internal.NewNopCheckerBuilder(
		"FILE_SAME_SWIFT_PREFIX",
		"files have the same value for the swift_prefix option",
		bufbreakingcheck.CheckFileSameSwiftPrefix,
	)
	// FileSameOptimizeForCheckerBuilder is a checker builder.
	FileSameOptimizeForCheckerBuilder = internal.NewNopCheckerBuilder(
		"FILE_SAME_OPTIMIZE_FOR",
		"files have the same value for the optimize_for option",
		bufbreakingcheck.CheckFileSameOptimizeFor,
	)
	// FileSameCcGenericServicesCheckerBuilder is a checker builder.
	FileSameCcGenericServicesCheckerBuilder = internal.NewNopCheckerBuilder(
		"FILE_SAME_CC_GENERIC_SERVICES",
		"files have the same value for the cc_generic_services option",
		bufbreakingcheck.CheckFileSameCcGenericServices,
	)
	// FileSameJavaGenericServicesCheckerBuilder is a checker builder.
	FileSameJavaGenericServicesCheckerBuilder = internal.NewNopCheckerBuilder(
		"FILE_SAME_JAVA_GENERIC_SERVICES",
		"files have the same value for the java_generic_services option",
		bufbreakingcheck.CheckFileSameJavaGenericServices,
	)
	// FileSamePyGenericServicesCheckerBuilder is a checker builder.
	FileSamePyGenericServicesCheckerBuilder = internal.NewNopCheckerBuilder(
		"FILE_SAME_PY_GENERIC_SERVICES",
		"files have the same value for the py_generic_services option",
		bufbreakingcheck.CheckFileSamePyGenericServices,
	)
	// FileSamePhpGenericServicesCheckerBuilder is a checker builder.
	FileSamePhpGenericServicesCheckerBuilder = internal.NewNopCheckerBuilder(
		"FILE_SAME_PHP_GENERIC_SERVICES",
		"files have the same value for the php_generic_services option",
		bufbreakingcheck.CheckFileSamePhpGenericServices,
	)
	// FileSameCcEnableArenasCheckerBuilder is a checker builder.
	FileSameCcEnableArenasCheckerBuilder = internal.NewNopCheckerBuilder(
		"FILE_SAME_CC_ENABLE_ARENAS",
		"files have the same value for the cc_enable_arenas option",
		bufbreakingcheck.CheckFileSameCcEnableArenas,
	)
	// FileSameSyntaxCheckerBuilder is a checker builder.
	FileSameSyntaxCheckerBuilder = internal.NewNopCheckerBuilder(
		"FILE_SAME_SYNTAX",
		"files have the same syntax",
		bufbreakingcheck.CheckFileSameSyntax,
	)
	// MessageNoDeleteCheckerBuilder is a checker builder.
	MessageNoDeleteCheckerBuilder = internal.NewNopCheckerBuilder(
		"MESSAGE_NO_DELETE",
		"messages are not deleted from a given file",
		bufbreakingcheck.CheckMessageNoDelete,
	)
	// MessageNoRemoveStandardDescriptorAccessorCheckerBuilder is a checker builder.
	MessageNoRemoveStandardDescriptorAccessorCheckerBuilder = internal.NewNopCheckerBuilder(
		"MESSAGE_NO_REMOVE_STANDARD_DESCRIPTOR_ACCESSOR",
		"messages do not change the no_standard_descriptor_accessor option from false or unset to true",
		bufbreakingcheck.CheckMessageNoRemoveStandardDescriptorAccessor,
	)
	// MessageSameMessageSetWireFormatCheckerBuilder is a checker builder.
	MessageSameMessageSetWireFormatCheckerBuilder = internal.NewNopCheckerBuilder(
		"MESSAGE_SAME_MESSAGE_SET_WIRE_FORMAT",
		"messages have the same value for the message_set_wire_format option",
		bufbreakingcheck.CheckMessageSameMessageSetWireFormat,
	)
	// OneofNoDeleteCheckerBuilder is a checker builder.
	OneofNoDeleteCheckerBuilder = internal.NewNopCheckerBuilder(
		"ONEOF_NO_DELETE",
		"oneofs are not deleted from a given message",
		bufbreakingcheck.CheckOneofNoDelete,
	)
	// PackageEnumNoDeleteCheckerBuilder is a checker builder.
	PackageEnumNoDeleteCheckerBuilder = internal.NewNopCheckerBuilder(
		"PACKAGE_ENUM_NO_DELETE",
		"enums are not deleted from a given package",
		bufbreakingcheck.CheckPackageEnumNoDelete,
	)
	// PackageMessageNoDeleteCheckerBuilder is a checker builder.
	PackageMessageNoDeleteCheckerBuilder = internal.NewNopCheckerBuilder(
		"PACKAGE_MESSAGE_NO_DELETE",
		"messages are not deleted from a given package",
		bufbreakingcheck.CheckPackageMessageNoDelete,
	)
	// PackageNoDeleteCheckerBuilder is a checker builder.
	PackageNoDeleteCheckerBuilder = internal.NewNopCheckerBuilder(
		"PACKAGE_NO_DELETE",
		"packages are not deleted",
		bufbreakingcheck.CheckPackageNoDelete,
	)
	// PackageServiceNoDeleteCheckerBuilder is a checker builder.
	PackageServiceNoDeleteCheckerBuilder = internal.NewNopCheckerBuilder(
		"PACKAGE_SERVICE_NO_DELETE",
		"services are not deleted from a given package",
		bufbreakingcheck.CheckPackageServiceNoDelete,
	)
	// ReservedEnumNoDeleteCheckerBuilder is a checker builder.
	ReservedEnumNoDeleteCheckerBuilder = internal.NewNopCheckerBuilder(
		"RESERVED_ENUM_NO_DELETE",
		"reserved ranges and names are not deleted from a given enum",
		bufbreakingcheck.CheckReservedEnumNoDelete,
	)
	// ReservedMessageNoDeleteCheckerBuilder is a checker builder.
	ReservedMessageNoDeleteCheckerBuilder = internal.NewNopCheckerBuilder(
		"RESERVED_MESSAGE_NO_DELETE",
		"reserved ranges and names are not deleted from a given message",
		bufbreakingcheck.CheckReservedMessageNoDelete,
	)
	// RPCNoDeleteCheckerBuilder is a checker builder.
	RPCNoDeleteCheckerBuilder = internal.NewNopCheckerBuilder(
		"RPC_NO_DELETE",
		"rpcs are not deleted from a given service",
		bufbreakingcheck.CheckRPCNoDelete,
	)
	// RPCSameClientStreamingCheckerBuilder is a checker builder.
	RPCSameClientStreamingCheckerBuilder = internal.NewNopCheckerBuilder(
		"RPC_SAME_CLIENT_STREAMING",
		"rpcs have the same client streaming value",
		bufbreakingcheck.CheckRPCSameClientStreaming,
	)
	// RPCSameIdempotencyLevelCheckerBuilder is a checker builder.
	RPCSameIdempotencyLevelCheckerBuilder = internal.NewNopCheckerBuilder(
		"RPC_SAME_IDEMPOTENCY_LEVEL",
		"rpcs have the same value for the idempotency_level option",
		bufbreakingcheck.CheckRPCSameIdempotencyLevel,
	)
	// RPCSameRequestTypeCheckerBuilder is a checker builder.
	RPCSameRequestTypeCheckerBuilder = internal.NewNopCheckerBuilder(
		"RPC_SAME_REQUEST_TYPE",
		"rpcs are have the same request type",
		bufbreakingcheck.CheckRPCSameRequestType,
	)
	// RPCSameResponseTypeCheckerBuilder is a checker builder.
	RPCSameResponseTypeCheckerBuilder = internal.NewNopCheckerBuilder(
		"RPC_SAME_RESPONSE_TYPE",
		"rpcs are have the same response type",
		bufbreakingcheck.CheckRPCSameResponseType,
	)
	// RPCSameServerStreamingCheckerBuilder is a checker builder.
	RPCSameServerStreamingCheckerBuilder = internal.NewNopCheckerBuilder(
		"RPC_SAME_SERVER_STREAMING",
		"rpcs have the same server streaming value",
		bufbreakingcheck.CheckRPCSameServerStreaming,
	)
	// ServiceNoDeleteCheckerBuilder is a checker builder.
	ServiceNoDeleteCheckerBuilder = internal.NewNopCheckerBuilder(
		"SERVICE_NO_DELETE",
		"services are not deleted from a given file",
		bufbreakingcheck.CheckServiceNoDelete,
	)
)
