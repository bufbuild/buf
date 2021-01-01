// Copyright 2020-2021 Buf Technologies, Inc.
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
	"github.com/bufbuild/buf/internal/buf/bufcheck/bufbreaking/internal/bufbreakingbuild"
	"github.com/bufbuild/buf/internal/buf/bufcheck/internal"
)

var (
	// v1beta1CheckerBuilders are the checker builders.
	v1beta1CheckerBuilders = []*internal.CheckerBuilder{
		bufbreakingbuild.EnumNoDeleteCheckerBuilder,
		bufbreakingbuild.EnumValueNoDeleteCheckerBuilder,
		bufbreakingbuild.EnumValueNoDeleteUnlessNameReservedCheckerBuilder,
		bufbreakingbuild.EnumValueNoDeleteUnlessNumberReservedCheckerBuilder,
		bufbreakingbuild.EnumValueSameNameCheckerBuilder,
		bufbreakingbuild.ExtensionMessageNoDeleteCheckerBuilder,
		bufbreakingbuild.FieldNoDeleteCheckerBuilder,
		bufbreakingbuild.FieldNoDeleteUnlessNameReservedCheckerBuilder,
		bufbreakingbuild.FieldNoDeleteUnlessNumberReservedCheckerBuilder,
		bufbreakingbuild.FieldSameCTypeCheckerBuilder,
		bufbreakingbuild.FieldSameJSONNameCheckerBuilder,
		bufbreakingbuild.FieldSameJSTypeCheckerBuilder,
		bufbreakingbuild.FieldSameLabelCheckerBuilder,
		bufbreakingbuild.FieldSameNameCheckerBuilder,
		bufbreakingbuild.FieldSameOneofCheckerBuilder,
		bufbreakingbuild.FieldSameTypeCheckerBuilder,
		bufbreakingbuild.FileNoDeleteCheckerBuilder,
		bufbreakingbuild.FileSameCsharpNamespaceCheckerBuilder,
		bufbreakingbuild.FileSameGoPackageCheckerBuilder,
		bufbreakingbuild.FileSameJavaMultipleFilesCheckerBuilder,
		bufbreakingbuild.FileSameJavaOuterClassnameCheckerBuilder,
		bufbreakingbuild.FileSameJavaPackageCheckerBuilder,
		bufbreakingbuild.FileSameJavaStringCheckUtf8CheckerBuilder,
		bufbreakingbuild.FileSameObjcClassPrefixCheckerBuilder,
		bufbreakingbuild.FileSamePackageCheckerBuilder,
		bufbreakingbuild.FileSamePhpClassPrefixCheckerBuilder,
		bufbreakingbuild.FileSamePhpMetadataNamespaceCheckerBuilder,
		bufbreakingbuild.FileSamePhpNamespaceCheckerBuilder,
		bufbreakingbuild.FileSameRubyPackageCheckerBuilder,
		bufbreakingbuild.FileSameSwiftPrefixCheckerBuilder,
		bufbreakingbuild.FileSameOptimizeForCheckerBuilder,
		bufbreakingbuild.FileSameCcGenericServicesCheckerBuilder,
		bufbreakingbuild.FileSameJavaGenericServicesCheckerBuilder,
		bufbreakingbuild.FileSamePyGenericServicesCheckerBuilder,
		bufbreakingbuild.FileSamePhpGenericServicesCheckerBuilder,
		bufbreakingbuild.FileSameCcEnableArenasCheckerBuilder,
		bufbreakingbuild.FileSameSyntaxCheckerBuilder,
		bufbreakingbuild.MessageNoDeleteCheckerBuilder,
		bufbreakingbuild.MessageNoRemoveStandardDescriptorAccessorCheckerBuilder,
		bufbreakingbuild.MessageSameMessageSetWireFormatCheckerBuilder,
		bufbreakingbuild.OneofNoDeleteCheckerBuilder,
		bufbreakingbuild.PackageEnumNoDeleteCheckerBuilder,
		bufbreakingbuild.PackageMessageNoDeleteCheckerBuilder,
		bufbreakingbuild.PackageNoDeleteCheckerBuilder,
		bufbreakingbuild.PackageServiceNoDeleteCheckerBuilder,
		bufbreakingbuild.ReservedEnumNoDeleteCheckerBuilder,
		bufbreakingbuild.ReservedMessageNoDeleteCheckerBuilder,
		bufbreakingbuild.RPCNoDeleteCheckerBuilder,
		bufbreakingbuild.RPCSameClientStreamingCheckerBuilder,
		bufbreakingbuild.RPCSameIdempotencyLevelCheckerBuilder,
		bufbreakingbuild.RPCSameRequestTypeCheckerBuilder,
		bufbreakingbuild.RPCSameResponseTypeCheckerBuilder,
		bufbreakingbuild.RPCSameServerStreamingCheckerBuilder,
		bufbreakingbuild.ServiceNoDeleteCheckerBuilder,
	}

	// v1beta1DefaultCategories are the default categories.
	v1beta1DefaultCategories = []string{
		"FILE",
	}
	// v1beta1AllCategories are all categories.
	v1beta1AllCategories = []string{
		"FILE",
		"PACKAGE",
		"WIRE_JSON",
		"WIRE",
	}
	// v1beta1IDToCategories are the revision 1 ID to categories.
	v1beta1IDToCategories = map[string][]string{
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
)
