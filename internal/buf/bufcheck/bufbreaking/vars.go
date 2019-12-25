package bufbreaking

import (
	"github.com/bufbuild/buf/internal/buf/bufcheck/bufbreaking/internal"
	bufcheckinternal "github.com/bufbuild/buf/internal/buf/bufcheck/internal"
)

var (
	// v1CheckerBuilders are the checker builders.
	v1CheckerBuilders = []*bufcheckinternal.CheckerBuilder{
		v1EnumNoDeleteCheckerBuilder,
		v1EnumValueNoDeleteCheckerBuilder,
		v1EnumValueNoDeleteUnlessNameReservedCheckerBuilder,
		v1EnumValueNoDeleteUnlessNumberReservedCheckerBuilder,
		v1EnumValueSameNameCheckerBuilder,
		v1ExtensionMessageNoDeleteCheckerBuilder,
		v1FieldNoDeleteCheckerBuilder,
		v1FieldNoDeleteUnlessNameReservedCheckerBuilder,
		v1FieldNoDeleteUnlessNumberReservedCheckerBuilder,
		v1FieldSameCTypeCheckerBuilder,
		v1FieldSameJSONNameCheckerBuilder,
		v1FieldSameJSTypeCheckerBuilder,
		v1FieldSameLabelCheckerBuilder,
		v1FieldSameNameCheckerBuilder,
		v1FieldSameOneofCheckerBuilder,
		v1FieldSameTypeCheckerBuilder,
		v1FileNoDeleteCheckerBuilder,
		v1FileSameCsharpNamespaceCheckerBuilder,
		v1FileSameGoPackageCheckerBuilder,
		v1FileSameJavaMultipleFilesCheckerBuilder,
		v1FileSameJavaOuterClassnameCheckerBuilder,
		v1FileSameJavaPackageCheckerBuilder,
		v1FileSameJavaStringCheckUtf8CheckerBuilder,
		v1FileSameObjcClassPrefixCheckerBuilder,
		v1FileSamePackageCheckerBuilder,
		v1FileSamePhpClassPrefixCheckerBuilder,
		v1FileSamePhpMetadataNamespaceCheckerBuilder,
		v1FileSamePhpNamespaceCheckerBuilder,
		v1FileSameRubyPackageCheckerBuilder,
		v1FileSameSwiftPrefixCheckerBuilder,
		v1FileSameOptimizeForCheckerBuilder,
		v1FileSameCcGenericServicesCheckerBuilder,
		v1FileSameJavaGenericServicesCheckerBuilder,
		v1FileSamePyGenericServicesCheckerBuilder,
		v1FileSamePhpGenericServicesCheckerBuilder,
		v1FileSameCcEnableArenasCheckerBuilder,
		v1FileSameSyntaxCheckerBuilder,
		v1MessageNoDeleteCheckerBuilder,
		v1MessageNoRemoveStandardDescriptorAccessorCheckerBuilder,
		v1MessageSameMessageSetWireFormatCheckerBuilder,
		v1OneofNoDeleteCheckerBuilder,
		v1PackageEnumNoDeleteCheckerBuilder,
		v1PackageMessageNoDeleteCheckerBuilder,
		v1PackageNoDeleteCheckerBuilder,
		v1PackageServiceNoDeleteCheckerBuilder,
		v1ReservedEnumNoDeleteCheckerBuilder,
		v1ReservedMessageNoDeleteCheckerBuilder,
		v1RPCNoDeleteCheckerBuilder,
		v1RPCSameClientStreamingCheckerBuilder,
		v1RPCSameIdempotencyLevelCheckerBuilder,
		v1RPCSameRequestTypeCheckerBuilder,
		v1RPCSameResponseTypeCheckerBuilder,
		v1RPCSameServerStreamingCheckerBuilder,
		v1ServiceNoDeleteCheckerBuilder,
	}

	// v1DefaultCategories are the default categories.
	v1DefaultCategories = []string{
		"FILE",
	}
	// v1AllCategories are all categories.
	v1AllCategories = []string{
		"FILE",
		"PACKAGE",
		"WIRE_JSON",
		"WIRE",
	}
	// v1IDToCategories are the revision 1 ID to categories.
	v1IDToCategories = map[string][]string{
		"ENUM_NO_DELETE": {
			"FILE",
		},
		"ENUM_VALUE_NO_DELETE": {
			"FILE",
			"PACKAGE",
		},
		"ENUM_VALUE_NO_DELETE_UNLESS_NAME_RESERVED": {
			"WIRE_JSON",
		},
		"ENUM_VALUE_NO_DELETE_UNLESS_NUMBER_RESERVED": {
			"WIRE_JSON",
			"WIRE",
		},
		"ENUM_VALUE_SAME_NAME": {
			"FILE",
			"PACKAGE",
			"WIRE_JSON",
		},
		"EXTENSION_MESSAGE_NO_DELETE": {
			"FILE",
			"PACKAGE",
		},
		"FIELD_NO_DELETE": {
			"FILE",
			"PACKAGE",
		},
		"FIELD_NO_DELETE_UNLESS_NAME_RESERVED": {
			"WIRE_JSON",
		},
		"FIELD_NO_DELETE_UNLESS_NUMBER_RESERVED": {
			"WIRE_JSON",
			"WIRE",
		},
		"FIELD_SAME_CTYPE": {
			"FILE",
			"PACKAGE",
		},
		"FIELD_SAME_JSON_NAME": {
			"FILE",
			"PACKAGE",
			"WIRE_JSON",
		},
		"FIELD_SAME_JSTYPE": {
			"FILE",
			"PACKAGE",
		},
		"FIELD_SAME_LABEL": {
			"FILE",
			"PACKAGE",
			"WIRE_JSON",
			"WIRE",
		},
		"FIELD_SAME_NAME": {
			"FILE",
			"PACKAGE",
			"WIRE_JSON",
		},
		"FIELD_SAME_ONEOF": {
			"FILE",
			"PACKAGE",
			"WIRE_JSON",
			"WIRE",
		},
		"FIELD_SAME_TYPE": {
			"FILE",
			"PACKAGE",
			"WIRE_JSON",
			"WIRE",
		},
		"FILE_NO_DELETE": {
			"FILE",
		},
		"FILE_SAME_CSHARP_NAMESPACE": {
			"FILE",
			"PACKAGE",
		},
		"FILE_SAME_GO_PACKAGE": {
			"FILE",
			"PACKAGE",
		},
		"FILE_SAME_JAVA_MULTIPLE_FILES": {
			"FILE",
			"PACKAGE",
		},
		"FILE_SAME_JAVA_OUTER_CLASSNAME": {
			"FILE",
			"PACKAGE",
		},
		"FILE_SAME_JAVA_PACKAGE": {
			"FILE",
			"PACKAGE",
		},
		"FILE_SAME_JAVA_STRING_CHECK_UTF8": {
			"FILE",
			"PACKAGE",
		},
		"FILE_SAME_OBJC_CLASS_PREFIX": {
			"FILE",
			"PACKAGE",
		},
		"FILE_SAME_PACKAGE": {
			"FILE",
		},
		"FILE_SAME_PHP_CLASS_PREFIX": {
			"FILE",
			"PACKAGE",
		},
		"FILE_SAME_PHP_METADATA_NAMESPACE": {
			"FILE",
			"PACKAGE",
		},
		"FILE_SAME_PHP_NAMESPACE": {
			"FILE",
			"PACKAGE",
		},
		"FILE_SAME_RUBY_PACKAGE": {
			"FILE",
			"PACKAGE",
		},
		"FILE_SAME_SWIFT_PREFIX": {
			"FILE",
			"PACKAGE",
		},
		"FILE_SAME_OPTIMIZE_FOR": {
			"FILE",
			"PACKAGE",
		},
		"FILE_SAME_CC_GENERIC_SERVICES": {
			"FILE",
			"PACKAGE",
		},
		"FILE_SAME_JAVA_GENERIC_SERVICES": {
			"FILE",
			"PACKAGE",
		},
		"FILE_SAME_PY_GENERIC_SERVICES": {
			"FILE",
			"PACKAGE",
		},
		"FILE_SAME_PHP_GENERIC_SERVICES": {
			"FILE",
			"PACKAGE",
		},
		"FILE_SAME_CC_ENABLE_ARENAS": {
			"FILE",
			"PACKAGE",
		},
		"FILE_SAME_SYNTAX": {
			"FILE",
			"PACKAGE",
		},
		"MESSAGE_NO_DELETE": {
			"FILE",
		},
		"MESSAGE_NO_REMOVE_STANDARD_DESCRIPTOR_ACCESSOR": {
			"FILE",
			"PACKAGE",
		},
		"MESSAGE_SAME_MESSAGE_SET_WIRE_FORMAT": {
			"FILE",
			"PACKAGE",
			"WIRE_JSON",
			"WIRE",
		},
		"ONEOF_NO_DELETE": {
			"FILE",
			"PACKAGE",
		},
		"PACKAGE_ENUM_NO_DELETE": {
			"PACKAGE",
		},
		"PACKAGE_MESSAGE_NO_DELETE": {
			"PACKAGE",
		},
		"PACKAGE_NO_DELETE": {
			"PACKAGE",
		},
		"PACKAGE_SERVICE_NO_DELETE": {
			"PACKAGE",
		},
		"RESERVED_ENUM_NO_DELETE": {
			"FILE",
			"PACKAGE",
			"WIRE_JSON",
			"WIRE",
		},
		"RESERVED_MESSAGE_NO_DELETE": {
			"FILE",
			"PACKAGE",
			"WIRE_JSON",
			"WIRE",
		},
		"RPC_NO_DELETE": {
			"FILE",
			"PACKAGE",
		},
		"RPC_SAME_CLIENT_STREAMING": {
			"FILE",
			"PACKAGE",
			"WIRE_JSON",
			"WIRE",
		},
		"RPC_SAME_IDEMPOTENCY_LEVEL": {
			"FILE",
			"PACKAGE",
			"WIRE_JSON",
			"WIRE",
		},
		"RPC_SAME_REQUEST_TYPE": {
			"FILE",
			"PACKAGE",
			"WIRE_JSON",
			"WIRE",
		},
		"RPC_SAME_RESPONSE_TYPE": {
			"FILE",
			"PACKAGE",
			"WIRE_JSON",
			"WIRE",
		},
		"RPC_SAME_SERVER_STREAMING": {
			"FILE",
			"PACKAGE",
			"WIRE_JSON",
			"WIRE",
		},
		"SERVICE_NO_DELETE": {
			"FILE",
		},
	}

	v1EnumNoDeleteCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"ENUM_NO_DELETE",
		"enums are not deleted from a given file",
		internal.CheckEnumNoDelete,
	)
	v1EnumValueNoDeleteCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"ENUM_VALUE_NO_DELETE",
		"enum values are not deleted from a given enum",
		internal.CheckEnumValueNoDelete,
	)
	v1EnumValueNoDeleteUnlessNameReservedCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"ENUM_VALUE_NO_DELETE_UNLESS_NAME_RESERVED",
		"enum values are not deleted from a given enum unless the name is reserved",
		internal.CheckEnumValueNoDeleteUnlessNameReserved,
	)
	v1EnumValueNoDeleteUnlessNumberReservedCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"ENUM_VALUE_NO_DELETE_UNLESS_NUMBER_RESERVED",
		"enum values are not deleted from a given enum unless the number is reserved",
		internal.CheckEnumValueNoDeleteUnlessNumberReserved,
	)
	v1EnumValueSameNameCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"ENUM_VALUE_SAME_NAME",
		"enum values have the same name",
		internal.CheckEnumValueSameName,
	)
	v1ExtensionMessageNoDeleteCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"EXTENSION_MESSAGE_NO_DELETE",
		"extension ranges are not deleted from a given message",
		internal.CheckExtensionMessageNoDelete,
	)
	v1FieldNoDeleteCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"FIELD_NO_DELETE",
		"fields are not deleted from a given message",
		internal.CheckFieldNoDelete,
	)
	v1FieldNoDeleteUnlessNameReservedCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"FIELD_NO_DELETE_UNLESS_NAME_RESERVED",
		"fields are not deleted from a given message unless the name is reserved",
		internal.CheckFieldNoDeleteUnlessNameReserved,
	)
	v1FieldNoDeleteUnlessNumberReservedCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"FIELD_NO_DELETE_UNLESS_NUMBER_RESERVED",
		"fields are not deleted from a given message unless the number is reserved",
		internal.CheckFieldNoDeleteUnlessNumberReserved,
	)
	v1FieldSameCTypeCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"FIELD_SAME_CTYPE",
		"fields have the same value for the ctype option",
		internal.CheckFieldSameCType,
	)
	v1FieldSameJSONNameCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"FIELD_SAME_JSON_NAME",
		"fields have the same value for the json_name option",
		internal.CheckFieldSameJSONName,
	)
	v1FieldSameJSTypeCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"FIELD_SAME_JSTYPE",
		"fields have the same value for the jstype option",
		internal.CheckFieldSameJSType,
	)
	v1FieldSameLabelCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"FIELD_SAME_LABEL",
		"fields have the same labels in a given message",
		internal.CheckFieldSameLabel,
	)
	v1FieldSameNameCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"FIELD_SAME_NAME",
		"fields have the same names in a given message",
		internal.CheckFieldSameName,
	)
	v1FieldSameOneofCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"FIELD_SAME_ONEOF",
		"fields have the same oneofs in a given message",
		internal.CheckFieldSameOneof,
	)
	v1FieldSameTypeCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"FIELD_SAME_TYPE",
		"fields have the same types in a given message",
		internal.CheckFieldSameType,
	)
	v1FileNoDeleteCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"FILE_NO_DELETE",
		"files are not deleted",
		internal.CheckFileNoDelete,
	)
	v1FileSameCsharpNamespaceCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"FILE_SAME_CSHARP_NAMESPACE",
		"files have the same value for the csharp_namespace option",
		internal.CheckFileSameCsharpNamespace,
	)
	v1FileSameGoPackageCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"FILE_SAME_GO_PACKAGE",
		"files have the same value for the go_package option",
		internal.CheckFileSameGoPackage,
	)
	v1FileSameJavaMultipleFilesCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"FILE_SAME_JAVA_MULTIPLE_FILES",
		"files have the same value for the java_multiple_files option",
		internal.CheckFileSameJavaMultipleFiles,
	)
	v1FileSameJavaOuterClassnameCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"FILE_SAME_JAVA_OUTER_CLASSNAME",
		"files have the same value for the java_outer_classname option",
		internal.CheckFileSameJavaOuterClassname,
	)
	v1FileSameJavaPackageCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"FILE_SAME_JAVA_PACKAGE",
		"files have the same value for the java_package option",
		internal.CheckFileSameJavaPackage,
	)
	v1FileSameJavaStringCheckUtf8CheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"FILE_SAME_JAVA_STRING_CHECK_UTF8",
		"files have the same value for the java_string_check_utf8 option",
		internal.CheckFileSameJavaStringCheckUtf8,
	)
	v1FileSameObjcClassPrefixCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"FILE_SAME_OBJC_CLASS_PREFIX",
		"files have the same value for the objc_class_prefix option",
		internal.CheckFileSameObjcClassPrefix,
	)
	v1FileSamePackageCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"FILE_SAME_PACKAGE",
		"files have the same package",
		internal.CheckFileSamePackage,
	)
	v1FileSamePhpClassPrefixCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"FILE_SAME_PHP_CLASS_PREFIX",
		"files have the same value for the php_class_prefix option",
		internal.CheckFileSamePhpClassPrefix,
	)
	v1FileSamePhpMetadataNamespaceCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"FILE_SAME_PHP_METADATA_NAMESPACE",
		"files have the same value for the php_metadata_namespace option",
		internal.CheckFileSamePhpMetadataNamespace,
	)
	v1FileSamePhpNamespaceCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"FILE_SAME_PHP_NAMESPACE",
		"files have the same value for the php_namespace option",
		internal.CheckFileSamePhpNamespace,
	)
	v1FileSameRubyPackageCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"FILE_SAME_RUBY_PACKAGE",
		"files have the same value for the ruby_package option",
		internal.CheckFileSameRubyPackage,
	)
	v1FileSameSwiftPrefixCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"FILE_SAME_SWIFT_PREFIX",
		"files have the same value for the swift_prefix option",
		internal.CheckFileSameSwiftPrefix,
	)
	v1FileSameOptimizeForCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"FILE_SAME_OPTIMIZE_FOR",
		"files have the same value for the optimize_for option",
		internal.CheckFileSameOptimizeFor,
	)
	v1FileSameCcGenericServicesCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"FILE_SAME_CC_GENERIC_SERVICES",
		"files have the same value for the cc_generic_services option",
		internal.CheckFileSameCcGenericServices,
	)
	v1FileSameJavaGenericServicesCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"FILE_SAME_JAVA_GENERIC_SERVICES",
		"files have the same value for the java_generic_services option",
		internal.CheckFileSameJavaGenericServices,
	)
	v1FileSamePyGenericServicesCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"FILE_SAME_PY_GENERIC_SERVICES",
		"files have the same value for the py_generic_services option",
		internal.CheckFileSamePyGenericServices,
	)
	v1FileSamePhpGenericServicesCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"FILE_SAME_PHP_GENERIC_SERVICES",
		"files have the same value for the php_generic_services option",
		internal.CheckFileSamePhpGenericServices,
	)
	v1FileSameCcEnableArenasCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"FILE_SAME_CC_ENABLE_ARENAS",
		"files have the same value for the cc_enable_arenas option",
		internal.CheckFileSameCcEnableArenas,
	)
	v1FileSameSyntaxCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"FILE_SAME_SYNTAX",
		"files have the same syntax",
		internal.CheckFileSameSyntax,
	)
	v1MessageNoDeleteCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"MESSAGE_NO_DELETE",
		"messages are not deleted from a given file",
		internal.CheckMessageNoDelete,
	)
	v1MessageNoRemoveStandardDescriptorAccessorCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"MESSAGE_NO_REMOVE_STANDARD_DESCRIPTOR_ACCESSOR",
		"messages do not change the no_standard_descriptor_accessor option from false or unset to true",
		internal.CheckMessageNoRemoveStandardDescriptorAccessor,
	)
	v1MessageSameMessageSetWireFormatCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"MESSAGE_SAME_MESSAGE_SET_WIRE_FORMAT",
		"messages have the same value for the message_set_wire_format option",
		internal.CheckMessageSameMessageSetWireFormat,
	)
	v1OneofNoDeleteCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"ONEOF_NO_DELETE",
		"oneofs are not deleted from a given message",
		internal.CheckOneofNoDelete,
	)
	v1PackageEnumNoDeleteCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"PACKAGE_ENUM_NO_DELETE",
		"enums are not deleted from a given package",
		internal.CheckPackageEnumNoDelete,
	)
	v1PackageMessageNoDeleteCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"PACKAGE_MESSAGE_NO_DELETE",
		"messages are not deleted from a given package",
		internal.CheckPackageMessageNoDelete,
	)
	v1PackageNoDeleteCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"PACKAGE_NO_DELETE",
		"packages are not deleted",
		internal.CheckPackageNoDelete,
	)
	v1PackageServiceNoDeleteCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"PACKAGE_SERVICE_NO_DELETE",
		"services are not deleted from a given package",
		internal.CheckPackageServiceNoDelete,
	)
	v1ReservedEnumNoDeleteCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"RESERVED_ENUM_NO_DELETE",
		"reserved ranges and names are not deleted from a given enum",
		internal.CheckReservedEnumNoDelete,
	)
	v1ReservedMessageNoDeleteCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"RESERVED_MESSAGE_NO_DELETE",
		"reserved ranges and names are not deleted from a given message",
		internal.CheckReservedMessageNoDelete,
	)
	v1RPCNoDeleteCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"RPC_NO_DELETE",
		"rpcs are not deleted from a given service",
		internal.CheckRPCNoDelete,
	)
	v1RPCSameClientStreamingCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"RPC_SAME_CLIENT_STREAMING",
		"rpcs have the same client streaming value",
		internal.CheckRPCSameClientStreaming,
	)
	v1RPCSameIdempotencyLevelCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"RPC_SAME_IDEMPOTENCY_LEVEL",
		"rpcs have the same value for the idempotency_level option",
		internal.CheckRPCSameIdempotencyLevel,
	)
	v1RPCSameRequestTypeCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"RPC_SAME_REQUEST_TYPE",
		"rpcs are have the same request type",
		internal.CheckRPCSameRequestType,
	)
	v1RPCSameResponseTypeCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"RPC_SAME_RESPONSE_TYPE",
		"rpcs are have the same response type",
		internal.CheckRPCSameResponseType,
	)
	v1RPCSameServerStreamingCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"RPC_SAME_SERVER_STREAMING",
		"rpcs have the same server streaming value",
		internal.CheckRPCSameServerStreaming,
	)
	v1ServiceNoDeleteCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"SERVICE_NO_DELETE",
		"services are not deleted from a given file",
		internal.CheckServiceNoDelete,
	)
)
