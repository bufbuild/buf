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

package bufbreakingv1beta1

import (
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufbreaking/internal/bufbreakingbuild"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/internal"
)

var (
	// v1beta1RuleBuilders are the rule builders.
	v1beta1RuleBuilders = []*internal.RuleBuilder{
		bufbreakingbuild.EnumNoDeleteRuleBuilder,
		bufbreakingbuild.EnumValueNoDeleteRuleBuilder,
		bufbreakingbuild.EnumValueNoDeleteUnlessNameReservedRuleBuilder,
		bufbreakingbuild.EnumValueNoDeleteUnlessNumberReservedRuleBuilder,
		bufbreakingbuild.EnumSameJSONFormatRuleBuilder,
		bufbreakingbuild.EnumSameTypeRuleBuilder,
		bufbreakingbuild.EnumValueSameNameRuleBuilder,
		bufbreakingbuild.ExtensionMessageNoDeleteRuleBuilder,
		bufbreakingbuild.FieldNoDeleteRuleBuilder,
		bufbreakingbuild.FieldNoDeleteUnlessNameReservedRuleBuilder,
		bufbreakingbuild.FieldNoDeleteUnlessNumberReservedRuleBuilder,
		bufbreakingbuild.FieldSameCardinalityRuleBuilder,
		bufbreakingbuild.FieldSameCppStringTypeRuleBuilder,
		bufbreakingbuild.FieldSameCTypeRuleBuilder,
		bufbreakingbuild.FieldSameJavaUTF8ValidationRuleBuilder,
		bufbreakingbuild.FieldSameJSONNameRuleBuilder,
		bufbreakingbuild.FieldSameJSTypeRuleBuilder,
		bufbreakingbuild.FieldSameLabelV1Beta1RuleBuilder,
		bufbreakingbuild.FieldSameNameRuleBuilder,
		bufbreakingbuild.FieldSameOneofRuleBuilder,
		bufbreakingbuild.FieldSameTypeRuleBuilder,
		bufbreakingbuild.FieldSameUTF8ValidationRuleBuilder,
		bufbreakingbuild.FileNoDeleteRuleBuilder,
		bufbreakingbuild.FileSameCsharpNamespaceRuleBuilder,
		bufbreakingbuild.FileSameGoPackageRuleBuilder,
		bufbreakingbuild.FileSameJavaMultipleFilesRuleBuilder,
		bufbreakingbuild.FileSameJavaOuterClassnameRuleBuilder,
		bufbreakingbuild.FileSameJavaPackageRuleBuilder,
		bufbreakingbuild.FileSameJavaStringCheckUtf8RuleBuilder,
		bufbreakingbuild.FileSameObjcClassPrefixRuleBuilder,
		bufbreakingbuild.FileSamePackageRuleBuilder,
		bufbreakingbuild.FileSamePhpClassPrefixRuleBuilder,
		bufbreakingbuild.FileSamePhpMetadataNamespaceRuleBuilder,
		bufbreakingbuild.FileSamePhpNamespaceRuleBuilder,
		bufbreakingbuild.FileSameRubyPackageRuleBuilder,
		bufbreakingbuild.FileSameSwiftPrefixRuleBuilder,
		bufbreakingbuild.FileSameOptimizeForRuleBuilder,
		bufbreakingbuild.FileSameCcGenericServicesRuleBuilder,
		bufbreakingbuild.FileSameJavaGenericServicesRuleBuilder,
		bufbreakingbuild.FileSamePyGenericServicesRuleBuilder,
		bufbreakingbuild.FileSamePhpGenericServicesRuleBuilder,
		bufbreakingbuild.FileSameCcEnableArenasRuleBuilder,
		bufbreakingbuild.FileSameSyntaxRuleBuilder,
		bufbreakingbuild.MessageNoDeleteRuleBuilder,
		bufbreakingbuild.MessageNoRemoveStandardDescriptorAccessorRuleBuilder,
		bufbreakingbuild.MessageSameJSONFormatRuleBuilder,
		bufbreakingbuild.MessageSameMessageSetWireFormatRuleBuilder,
		bufbreakingbuild.MessageSameRequiredFieldsRuleBuilder,
		bufbreakingbuild.OneofNoDeleteRuleBuilder,
		bufbreakingbuild.PackageEnumNoDeleteRuleBuilder,
		bufbreakingbuild.PackageMessageNoDeleteRuleBuilder,
		bufbreakingbuild.PackageNoDeleteRuleBuilder,
		bufbreakingbuild.PackageServiceNoDeleteRuleBuilder,
		bufbreakingbuild.ReservedEnumNoDeleteRuleBuilder,
		bufbreakingbuild.ReservedMessageNoDeleteRuleBuilder,
		bufbreakingbuild.RPCNoDeleteRuleBuilder,
		bufbreakingbuild.RPCSameClientStreamingRuleBuilder,
		bufbreakingbuild.RPCSameIdempotencyLevelRuleBuilder,
		bufbreakingbuild.RPCSameRequestTypeRuleBuilder,
		bufbreakingbuild.RPCSameResponseTypeRuleBuilder,
		bufbreakingbuild.RPCSameServerStreamingRuleBuilder,
		bufbreakingbuild.ServiceNoDeleteRuleBuilder,
	}

	// v1beta1DefaultCategories are the default categories.
	v1beta1DefaultCategories = []string{
		"FILE",
	}
	// v1beta1IDToCategories associates IDs to categories.
	v1beta1IDToCategories = map[string][]string{
		"ENUM_NO_DELETE": {
			"FILE",
		},
		"ENUM_SAME_JSON_FORMAT": {
			"FILE",
			"PACKAGE",
			"WIRE_JSON",
		},
		"ENUM_SAME_TYPE": {
			"FILE",
			"PACKAGE",
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
		"FIELD_SAME_CARDINALITY": {
			"FILE",
			"PACKAGE",
			"WIRE_JSON",
			"WIRE",
		},
		"FIELD_SAME_CPP_STRING_TYPE": {
			"FILE",
			"PACKAGE",
		},
		"FIELD_SAME_CTYPE": {}, // deprecated, so not part of any category by default
		"FIELD_SAME_JAVA_UTF8_VALIDATION": {
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
		"FIELD_SAME_LABEL": {}, // deprecated, so not part of any category by default
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
		"FIELD_SAME_UTF8_VALIDATION": {
			"FILE",
			"PACKAGE",
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
		"FILE_SAME_JAVA_STRING_CHECK_UTF8": {}, // deprecated, so not part of any category by default
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
		"FILE_SAME_PHP_GENERIC_SERVICES": {}, // deprecated, so not part of any category by default
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
		"MESSAGE_SAME_JSON_FORMAT": {
			"FILE",
			"PACKAGE",
			"WIRE_JSON",
		},
		"MESSAGE_SAME_MESSAGE_SET_WIRE_FORMAT": {
			"FILE",
			"PACKAGE",
			"WIRE_JSON",
			"WIRE",
		},
		"MESSAGE_SAME_REQUIRED_FIELDS": {
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
)
