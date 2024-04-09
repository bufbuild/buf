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

// Package buflintbuild contains the RuleBuilders used by buflintv*.
package buflintbuild

import (
	"errors"

	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/buflint/internal/buflintcheck"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/internal"
	"github.com/bufbuild/buf/private/bufpkg/bufprotosource"
)

var (
	// CommentEnumRuleBuilder is a rule builder.
	CommentEnumRuleBuilder = internal.NewNopRuleBuilder(
		"COMMENT_ENUM",
		"enums have non-empty comments",
		false,
		nil,
		newAdapter(buflintcheck.CheckCommentEnum),
	)
	// CommentEnumValueRuleBuilder is a rule builder.
	CommentEnumValueRuleBuilder = internal.NewNopRuleBuilder(
		"COMMENT_ENUM_VALUE",
		"enum values have non-empty comments",
		false,
		nil,
		newAdapter(buflintcheck.CheckCommentEnumValue),
	)
	// CommentFieldRuleBuilder is a rule builder.
	CommentFieldRuleBuilder = internal.NewNopRuleBuilder(
		"COMMENT_FIELD",
		"fields have non-empty comments",
		false,
		nil,
		newAdapter(buflintcheck.CheckCommentField),
	)
	// CommentMessageRuleBuilder is a rule builder.
	CommentMessageRuleBuilder = internal.NewNopRuleBuilder(
		"COMMENT_MESSAGE",
		"messages have non-empty comments",
		false,
		nil,
		newAdapter(buflintcheck.CheckCommentMessage),
	)
	// CommentOneofRuleBuilder is a rule builder.
	CommentOneofRuleBuilder = internal.NewNopRuleBuilder(
		"COMMENT_ONEOF",
		"oneof have non-empty comments",
		false,
		nil,
		newAdapter(buflintcheck.CheckCommentOneof),
	)
	// CommentRPCRuleBuilder is a rule builder.
	CommentRPCRuleBuilder = internal.NewNopRuleBuilder(
		"COMMENT_RPC",
		"RPCs have non-empty comments",
		false,
		nil,
		newAdapter(buflintcheck.CheckCommentRPC),
	)
	// CommentServiceRuleBuilder is a rule builder.
	CommentServiceRuleBuilder = internal.NewNopRuleBuilder(
		"COMMENT_SERVICE",
		"services have non-empty comments",
		false,
		nil,
		newAdapter(buflintcheck.CheckCommentService),
	)
	// DirectorySamePackageRuleBuilder is a rule builder.
	DirectorySamePackageRuleBuilder = internal.NewNopRuleBuilder(
		"DIRECTORY_SAME_PACKAGE",
		"all files in a given directory are in the same package",
		false,
		nil,
		newAdapter(buflintcheck.CheckDirectorySamePackage),
	)
	// EnumFirstValueZeroRuleBuilder is a rule builder.
	EnumFirstValueZeroRuleBuilder = internal.NewNopRuleBuilder(
		"ENUM_FIRST_VALUE_ZERO",
		"all first values of enums have a numeric value of 0",
		false,
		nil,
		newAdapter(buflintcheck.CheckEnumFirstValueZero),
	)
	// EnumNoAllowAliasRuleBuilder is a rule builder.
	EnumNoAllowAliasRuleBuilder = internal.NewNopRuleBuilder(
		"ENUM_NO_ALLOW_ALIAS",
		"enums do not have the allow_alias option set",
		false,
		nil,
		newAdapter(buflintcheck.CheckEnumNoAllowAlias),
	)
	// EnumPascalCaseRuleBuilder is a rule builder.
	EnumPascalCaseRuleBuilder = internal.NewNopRuleBuilder(
		"ENUM_PASCAL_CASE",
		"enums are PascalCase",
		false,
		nil,
		newAdapter(buflintcheck.CheckEnumPascalCase),
	)
	// EnumValuePrefixRuleBuilder is a rule builder.
	EnumValuePrefixRuleBuilder = internal.NewNopRuleBuilder(
		"ENUM_VALUE_PREFIX",
		"enum values are prefixed with ENUM_NAME_UPPER_SNAKE_CASE",
		false,
		nil,
		newAdapter(buflintcheck.CheckEnumValuePrefix),
	)
	// EnumValueUpperSnakeCaseRuleBuilder is a rule builder.
	EnumValueUpperSnakeCaseRuleBuilder = internal.NewNopRuleBuilder(
		"ENUM_VALUE_UPPER_SNAKE_CASE",
		"enum values are UPPER_SNAKE_CASE",
		false,
		nil,
		newAdapter(buflintcheck.CheckEnumValueUpperSnakeCase),
	)
	// EnumZeroValueSuffixRuleBuilder is a rule builder.
	EnumZeroValueSuffixRuleBuilder = internal.NewRuleBuilder(
		"ENUM_ZERO_VALUE_SUFFIX",
		func(configBuilder internal.ConfigBuilder) (string, error) {
			if configBuilder.EnumZeroValueSuffix == "" {
				return "", errors.New("enum_zero_value_suffix is empty")
			}
			return "enum zero values are suffixed with " + configBuilder.EnumZeroValueSuffix + " (suffix is configurable)", nil
		},
		false,
		nil,
		func(configBuilder internal.ConfigBuilder) (internal.CheckFunc, error) {
			if configBuilder.EnumZeroValueSuffix == "" {
				return nil, errors.New("enum_zero_value_suffix is empty")
			}
			return internal.CheckFunc(func(id string, ignoreFunc internal.IgnoreFunc, _ []bufprotosource.File, files []bufprotosource.File) ([]bufanalysis.FileAnnotation, error) {
				return buflintcheck.CheckEnumZeroValueSuffix(id, ignoreFunc, files, configBuilder.EnumZeroValueSuffix)
			}), nil
		},
	)
	// FieldLowerSnakeCaseRuleBuilder is a rule builder.
	FieldLowerSnakeCaseRuleBuilder = internal.NewNopRuleBuilder(
		"FIELD_LOWER_SNAKE_CASE",
		"field names are lower_snake_case",
		false,
		nil,
		newAdapter(buflintcheck.CheckFieldLowerSnakeCase),
	)
	// FieldNoDescriptorRuleBuilder is a rule builder.
	FieldNoDescriptorRuleBuilder = internal.NewNopRuleBuilder(
		"FIELD_NO_DESCRIPTOR",
		`field names are not name capitalization of "descriptor" with any number of prefix or suffix underscores`,
		false,
		nil,
		newAdapter(buflintcheck.CheckFieldNoDescriptor),
	)
	// FileLowerSnakeCaseRuleBuilder is a rule builder.
	FileLowerSnakeCaseRuleBuilder = internal.NewNopRuleBuilder(
		"FILE_LOWER_SNAKE_CASE",
		"filenames are lower_snake_case",
		false,
		nil,
		newAdapter(buflintcheck.CheckFileLowerSnakeCase),
	)
	// ImportNoPublicRuleBuilder is a rule builder.
	ImportNoPublicRuleBuilder = internal.NewNopRuleBuilder(
		"IMPORT_NO_PUBLIC",
		"imports are not public",
		false,
		nil,
		newAdapter(buflintcheck.CheckImportNoPublic),
	)
	// ImportNoWeakRuleBuilder is a rule builder.
	ImportNoWeakRuleBuilder = internal.NewNopRuleBuilder(
		"IMPORT_NO_WEAK",
		"imports are not weak",
		false,
		nil,
		newAdapter(buflintcheck.CheckImportNoWeak),
	)
	// ImportUsedRuleBuilder is a rule builder.
	ImportUsedRuleBuilder = internal.NewNopRuleBuilder(
		"IMPORT_USED",
		"imports are used",
		false,
		nil,
		newAdapter(buflintcheck.CheckImportUsed),
	)
	// MessagePascalCaseRuleBuilder is a rule builder.
	MessagePascalCaseRuleBuilder = internal.NewNopRuleBuilder(
		"MESSAGE_PASCAL_CASE",
		"messages are PascalCase",
		false,
		nil,
		newAdapter(buflintcheck.CheckMessagePascalCase),
	)
	// OneofLowerSnakeCaseRuleBuilder is a rule builder.
	OneofLowerSnakeCaseRuleBuilder = internal.NewNopRuleBuilder(
		"ONEOF_LOWER_SNAKE_CASE",
		"oneof names are lower_snake_case",
		false,
		nil,
		newAdapter(buflintcheck.CheckOneofLowerSnakeCase),
	)
	// PackageDefinedRuleBuilder is a rule builder.
	PackageDefinedRuleBuilder = internal.NewNopRuleBuilder(
		"PACKAGE_DEFINED",
		"all files have a package defined",
		false,
		nil,
		newAdapter(buflintcheck.CheckPackageDefined),
	)
	// PackageDirectoryMatchRuleBuilder is a rule builder.
	PackageDirectoryMatchRuleBuilder = internal.NewNopRuleBuilder(
		"PACKAGE_DIRECTORY_MATCH",
		"all files are in a directory that matches their package name",
		false,
		nil,
		newAdapter(buflintcheck.CheckPackageDirectoryMatch),
	)
	// PackageLowerSnakeCaseRuleBuilder is a rule builder.
	PackageLowerSnakeCaseRuleBuilder = internal.NewNopRuleBuilder(
		"PACKAGE_LOWER_SNAKE_CASE",
		"packages are lower_snake.case",
		false,
		nil,
		newAdapter(buflintcheck.CheckPackageLowerSnakeCase),
	)
	// PackageNoImportCycleRuleBuilder is a rule builder.
	PackageNoImportCycleRuleBuilder = internal.NewNopRuleBuilder(
		"PACKAGE_NO_IMPORT_CYCLE",
		"packages do not have import cycles",
		false,
		nil,
		newAdapter(buflintcheck.CheckPackageNoImportCycle),
	)
	// PackageSameCsharpNamespaceRuleBuilder is a rule builder.
	PackageSameCsharpNamespaceRuleBuilder = internal.NewNopRuleBuilder(
		"PACKAGE_SAME_CSHARP_NAMESPACE",
		"all files with a given package have the same value for the csharp_namespace option",
		false,
		nil,
		newAdapter(buflintcheck.CheckPackageSameCsharpNamespace),
	)
	// PackageSameDirectoryRuleBuilder is a rule builder.
	PackageSameDirectoryRuleBuilder = internal.NewNopRuleBuilder(
		"PACKAGE_SAME_DIRECTORY",
		"all files with a given package are in the same directory",
		false,
		nil,
		newAdapter(buflintcheck.CheckPackageSameDirectory),
	)
	// PackageSameGoPackageRuleBuilder is a rule builder.
	PackageSameGoPackageRuleBuilder = internal.NewNopRuleBuilder(
		"PACKAGE_SAME_GO_PACKAGE",
		"all files with a given package have the same value for the go_package option",
		false,
		nil,
		newAdapter(buflintcheck.CheckPackageSameGoPackage),
	)
	// PackageSameJavaMultipleFilesRuleBuilder is a rule builder.
	PackageSameJavaMultipleFilesRuleBuilder = internal.NewNopRuleBuilder(
		"PACKAGE_SAME_JAVA_MULTIPLE_FILES",
		"all files with a given package have the same value for the java_multiple_files option",
		false,
		nil,
		newAdapter(buflintcheck.CheckPackageSameJavaMultipleFiles),
	)
	// PackageSameJavaPackageRuleBuilder is a rule builder.
	PackageSameJavaPackageRuleBuilder = internal.NewNopRuleBuilder(
		"PACKAGE_SAME_JAVA_PACKAGE",
		"all files with a given package have the same value for the java_package option",
		false,
		nil,
		newAdapter(buflintcheck.CheckPackageSameJavaPackage),
	)
	// PackageSamePhpNamespaceRuleBuilder is a rule builder.
	PackageSamePhpNamespaceRuleBuilder = internal.NewNopRuleBuilder(
		"PACKAGE_SAME_PHP_NAMESPACE",
		"all files with a given package have the same value for the php_namespace option",
		false,
		nil,
		newAdapter(buflintcheck.CheckPackageSamePhpNamespace),
	)
	// PackageSameRubyPackageRuleBuilder is a rule builder.
	PackageSameRubyPackageRuleBuilder = internal.NewNopRuleBuilder(
		"PACKAGE_SAME_RUBY_PACKAGE",
		"all files with a given package have the same value for the ruby_package option",
		false,
		nil,
		newAdapter(buflintcheck.CheckPackageSameRubyPackage),
	)
	// PackageSameSwiftPrefixRuleBuilder is a rule builder.
	PackageSameSwiftPrefixRuleBuilder = internal.NewNopRuleBuilder(
		"PACKAGE_SAME_SWIFT_PREFIX",
		"all files with a given package have the same value for the swift_prefix option",
		false,
		nil,
		newAdapter(buflintcheck.CheckPackageSameSwiftPrefix),
	)
	// PackageVersionSuffixRuleBuilder is a rule builder.
	PackageVersionSuffixRuleBuilder = internal.NewNopRuleBuilder(
		"PACKAGE_VERSION_SUFFIX",
		`the last component of all packages is a version of the form v\d+, v\d+test.*, v\d+(alpha|beta)\d+, or v\d+p\d+(alpha|beta)\d+, where numbers are >=1`,
		false,
		nil,
		newAdapter(buflintcheck.CheckPackageVersionSuffix),
	)
	// ProtovalidateRuleBuilder is a rule builder.
	ProtovalidateRuleBuilder = internal.NewNopRuleBuilder(
		"PROTOVALIDATE",
		"protovalidate rules are valid and all CEL expressions compile",
		false,
		nil,
		newAdapter(buflintcheck.CheckProtovalidate),
	)
	// RPCNoClientStreamingRuleBuilder is a rule builder.
	RPCNoClientStreamingRuleBuilder = internal.NewNopRuleBuilder(
		"RPC_NO_CLIENT_STREAMING",
		"RPCs are not client streaming",
		false,
		nil,
		newAdapter(buflintcheck.CheckRPCNoClientStreaming),
	)
	// RPCNoServerStreamingRuleBuilder is a rule builder.
	RPCNoServerStreamingRuleBuilder = internal.NewNopRuleBuilder(
		"RPC_NO_SERVER_STREAMING",
		"RPCs are not server streaming",
		false,
		nil,
		newAdapter(buflintcheck.CheckRPCNoServerStreaming),
	)
	// RPCPascalCaseRuleBuilder is a rule builder.
	RPCPascalCaseRuleBuilder = internal.NewNopRuleBuilder(
		"RPC_PASCAL_CASE",
		"RPCs are PascalCase",
		false,
		nil,
		newAdapter(buflintcheck.CheckRPCPascalCase),
	)
	// RPCRequestResponseUniqueRuleBuilder is a rule builder.
	RPCRequestResponseUniqueRuleBuilder = internal.NewRuleBuilder(
		"RPC_REQUEST_RESPONSE_UNIQUE",
		func(configBuilder internal.ConfigBuilder) (string, error) {
			return "RPC request and response types are only used in one RPC (configurable)", nil
		},
		false,
		nil,
		func(configBuilder internal.ConfigBuilder) (internal.CheckFunc, error) {
			return internal.CheckFunc(func(id string, ignoreFunc internal.IgnoreFunc, _ []bufprotosource.File, files []bufprotosource.File) ([]bufanalysis.FileAnnotation, error) {
				return buflintcheck.CheckRPCRequestResponseUnique(
					id,
					ignoreFunc,
					files,
					configBuilder.RPCAllowSameRequestResponse,
					configBuilder.RPCAllowGoogleProtobufEmptyRequests,
					configBuilder.RPCAllowGoogleProtobufEmptyResponses,
				)
			}), nil
		},
	)
	// RPCRequestStandardNameRuleBuilder is a rule builder.
	RPCRequestStandardNameRuleBuilder = internal.NewRuleBuilder(
		"RPC_REQUEST_STANDARD_NAME",
		func(configBuilder internal.ConfigBuilder) (string, error) {
			return "RPC request type names are RPCNameRequest or ServiceNameRPCNameRequest (configurable)", nil
		},
		false,
		nil,
		func(configBuilder internal.ConfigBuilder) (internal.CheckFunc, error) {
			return internal.CheckFunc(func(id string, ignoreFunc internal.IgnoreFunc, _ []bufprotosource.File, files []bufprotosource.File) ([]bufanalysis.FileAnnotation, error) {
				return buflintcheck.CheckRPCRequestStandardName(
					id,
					ignoreFunc,
					files,
					configBuilder.RPCAllowGoogleProtobufEmptyRequests,
				)
			}), nil
		},
	)
	// RPCResponseStandardNameRuleBuilder is a rule builder.
	RPCResponseStandardNameRuleBuilder = internal.NewRuleBuilder(
		"RPC_RESPONSE_STANDARD_NAME",
		func(configBuilder internal.ConfigBuilder) (string, error) {
			return "RPC response type names are RPCNameResponse or ServiceNameRPCNameResponse (configurable)", nil
		},
		false,
		nil,
		func(configBuilder internal.ConfigBuilder) (internal.CheckFunc, error) {
			return internal.CheckFunc(func(id string, ignoreFunc internal.IgnoreFunc, _ []bufprotosource.File, files []bufprotosource.File) ([]bufanalysis.FileAnnotation, error) {
				return buflintcheck.CheckRPCResponseStandardName(
					id,
					ignoreFunc,
					files,
					configBuilder.RPCAllowGoogleProtobufEmptyResponses,
				)
			}), nil
		},
	)
	// ServicePascalCaseRuleBuilder is a rule builder.
	ServicePascalCaseRuleBuilder = internal.NewNopRuleBuilder(
		"SERVICE_PASCAL_CASE",
		"services are PascalCase",
		false,
		nil,
		newAdapter(buflintcheck.CheckServicePascalCase),
	)
	// ServiceSuffixRuleBuilder is a rule builder.
	ServiceSuffixRuleBuilder = internal.NewRuleBuilder(
		"SERVICE_SUFFIX",
		func(configBuilder internal.ConfigBuilder) (string, error) {
			if configBuilder.ServiceSuffix == "" {
				return "", errors.New("service_suffix is empty")
			}
			return "services are suffixed with " + configBuilder.ServiceSuffix + " (suffix is configurable)", nil
		},
		false,
		nil,
		func(configBuilder internal.ConfigBuilder) (internal.CheckFunc, error) {
			if configBuilder.ServiceSuffix == "" {
				return nil, errors.New("service_suffix is empty")
			}
			return internal.CheckFunc(func(id string, ignoreFunc internal.IgnoreFunc, _ []bufprotosource.File, files []bufprotosource.File) ([]bufanalysis.FileAnnotation, error) {
				return buflintcheck.CheckServiceSuffix(id, ignoreFunc, files, configBuilder.ServiceSuffix)
			}), nil
		},
	)
	// SyntaxSpecifiedRuleBuilder is a rule builder.
	SyntaxSpecifiedRuleBuilder = internal.NewNopRuleBuilder(
		"SYNTAX_SPECIFIED",
		"all files have a syntax specified",
		false,
		nil,
		newAdapter(buflintcheck.CheckSyntaxSpecified),
	)
)

func newAdapter(
	f func(string, internal.IgnoreFunc, []bufprotosource.File) ([]bufanalysis.FileAnnotation, error),
) func(string, internal.IgnoreFunc, []bufprotosource.File, []bufprotosource.File) ([]bufanalysis.FileAnnotation, error) {
	return func(id string, ignoreFunc internal.IgnoreFunc, _ []bufprotosource.File, files []bufprotosource.File) ([]bufanalysis.FileAnnotation, error) {
		return f(id, ignoreFunc, files)
	}
}
