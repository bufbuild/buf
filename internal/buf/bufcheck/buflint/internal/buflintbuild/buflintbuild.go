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

// Package buflintbuild contains the CheckerBuilders used by buflintv*.
package buflintbuild

import (
	"errors"

	"github.com/bufbuild/buf/internal/buf/bufanalysis"
	"github.com/bufbuild/buf/internal/buf/bufcheck/buflint/internal/buflintcheck"
	"github.com/bufbuild/buf/internal/buf/bufcheck/internal"
	"github.com/bufbuild/buf/internal/pkg/protosource"
)

var (
	// CommentEnumCheckerBuilder is a checker builder.
	CommentEnumCheckerBuilder = internal.NewNopCheckerBuilder(
		"COMMENT_ENUM",
		"enums have non-empty comments",
		newAdapter(buflintcheck.CheckCommentEnum),
	)
	// CommentEnumValueCheckerBuilder is a checker builder.
	CommentEnumValueCheckerBuilder = internal.NewNopCheckerBuilder(
		"COMMENT_ENUM_VALUE",
		"enum values have non-empty comments",
		newAdapter(buflintcheck.CheckCommentEnumValue),
	)
	// CommentFieldCheckerBuilder is a checker builder.
	CommentFieldCheckerBuilder = internal.NewNopCheckerBuilder(
		"COMMENT_FIELD",
		"fields have non-empty comments",
		newAdapter(buflintcheck.CheckCommentField),
	)
	// CommentMessageCheckerBuilder is a checker builder.
	CommentMessageCheckerBuilder = internal.NewNopCheckerBuilder(
		"COMMENT_MESSAGE",
		"messages have non-empty comments",
		newAdapter(buflintcheck.CheckCommentMessage),
	)
	// CommentOneofCheckerBuilder is a checker builder.
	CommentOneofCheckerBuilder = internal.NewNopCheckerBuilder(
		"COMMENT_ONEOF",
		"oneof have non-empty comments",
		newAdapter(buflintcheck.CheckCommentOneof),
	)
	// CommentRPCCheckerBuilder is a checker builder.
	CommentRPCCheckerBuilder = internal.NewNopCheckerBuilder(
		"COMMENT_RPC",
		"RPCs have non-empty comments",
		newAdapter(buflintcheck.CheckCommentRPC),
	)
	// CommentServiceCheckerBuilder is a checker builder.
	CommentServiceCheckerBuilder = internal.NewNopCheckerBuilder(
		"COMMENT_SERVICE",
		"services have non-empty comments",
		newAdapter(buflintcheck.CheckCommentService),
	)
	// DirectorySamePackageCheckerBuilder is a checker builder.
	DirectorySamePackageCheckerBuilder = internal.NewNopCheckerBuilder(
		"DIRECTORY_SAME_PACKAGE",
		"all files in a given directory are in the same package",
		newAdapter(buflintcheck.CheckDirectorySamePackage),
	)
	// EnumFirstValueZeroCheckerBuilder is a checker builder.
	EnumFirstValueZeroCheckerBuilder = internal.NewNopCheckerBuilder(
		"ENUM_FIRST_VALUE_ZERO",
		"all first values of enums have a numeric value of 0",
		newAdapter(buflintcheck.CheckEnumFirstValueZero),
	)
	// EnumNoAllowAliasCheckerBuilder is a checker builder.
	EnumNoAllowAliasCheckerBuilder = internal.NewNopCheckerBuilder(
		"ENUM_NO_ALLOW_ALIAS",
		"enums do not have the allow_alias option set",
		newAdapter(buflintcheck.CheckEnumNoAllowAlias),
	)
	// EnumPascalCaseCheckerBuilder is a checker builder.
	EnumPascalCaseCheckerBuilder = internal.NewNopCheckerBuilder(
		"ENUM_PASCAL_CASE",
		"enums are PascalCase",
		newAdapter(buflintcheck.CheckEnumPascalCase),
	)
	// EnumValuePrefixCheckerBuilder is a checker builder.
	EnumValuePrefixCheckerBuilder = internal.NewNopCheckerBuilder(
		"ENUM_VALUE_PREFIX",
		"enum values are prefixed with ENUM_NAME_UPPER_SNAKE_CASE",
		newAdapter(buflintcheck.CheckEnumValuePrefix),
	)
	// EnumValueUpperSnakeCaseCheckerBuilder is a checker builder.
	EnumValueUpperSnakeCaseCheckerBuilder = internal.NewNopCheckerBuilder(
		"ENUM_VALUE_UPPER_SNAKE_CASE",
		"enum values are UPPER_SNAKE_CASE",
		newAdapter(buflintcheck.CheckEnumValueUpperSnakeCase),
	)
	// EnumZeroValueSuffixCheckerBuilder is a checker builder.
	EnumZeroValueSuffixCheckerBuilder = internal.NewCheckerBuilder(
		"ENUM_ZERO_VALUE_SUFFIX",
		func(configBuilder internal.ConfigBuilder) (string, error) {
			if configBuilder.EnumZeroValueSuffix == "" {
				return "", errors.New("enum_zero_value_suffix is empty")
			}
			return "enum zero values are suffixed with " + configBuilder.EnumZeroValueSuffix + " (suffix is configurable)", nil
		},
		func(configBuilder internal.ConfigBuilder) (internal.CheckFunc, error) {
			if configBuilder.EnumZeroValueSuffix == "" {
				return nil, errors.New("enum_zero_value_suffix is empty")
			}
			return internal.CheckFunc(func(id string, ignoreFunc internal.IgnoreFunc, _ []protosource.File, files []protosource.File) ([]bufanalysis.FileAnnotation, error) {
				return buflintcheck.CheckEnumZeroValueSuffix(id, ignoreFunc, files, configBuilder.EnumZeroValueSuffix)
			}), nil
		},
	)
	// FieldLowerSnakeCaseCheckerBuilder is a checker builder.
	FieldLowerSnakeCaseCheckerBuilder = internal.NewNopCheckerBuilder(
		"FIELD_LOWER_SNAKE_CASE",
		"field names are lower_snake_case",
		newAdapter(buflintcheck.CheckFieldLowerSnakeCase),
	)
	// FieldNoDescriptorCheckerBuilder is a checker builder.
	FieldNoDescriptorCheckerBuilder = internal.NewNopCheckerBuilder(
		"FIELD_NO_DESCRIPTOR",
		`field names are not name capitalization of "descriptor" with any number of prefix or suffix underscores`,
		newAdapter(buflintcheck.CheckFieldNoDescriptor),
	)
	// FileLowerSnakeCaseCheckerBuilder is a checker builder.
	FileLowerSnakeCaseCheckerBuilder = internal.NewNopCheckerBuilder(
		"FILE_LOWER_SNAKE_CASE",
		"filenames are lower_snake_case",
		newAdapter(buflintcheck.CheckFileLowerSnakeCase),
	)
	// ImportNoPublicCheckerBuilder is a checker builder.
	ImportNoPublicCheckerBuilder = internal.NewNopCheckerBuilder(
		"IMPORT_NO_PUBLIC",
		"imports are not public",
		newAdapter(buflintcheck.CheckImportNoPublic),
	)
	// ImportNoWeakCheckerBuilder is a checker builder.
	ImportNoWeakCheckerBuilder = internal.NewNopCheckerBuilder(
		"IMPORT_NO_WEAK",
		"imports are not weak",
		newAdapter(buflintcheck.CheckImportNoWeak),
	)
	// MessagePascalCaseCheckerBuilder is a checker builder.
	MessagePascalCaseCheckerBuilder = internal.NewNopCheckerBuilder(
		"MESSAGE_PASCAL_CASE",
		"messages are PascalCase",
		newAdapter(buflintcheck.CheckMessagePascalCase),
	)
	// OneofLowerSnakeCaseCheckerBuilder is a checker builder.
	OneofLowerSnakeCaseCheckerBuilder = internal.NewNopCheckerBuilder(
		"ONEOF_LOWER_SNAKE_CASE",
		"oneof names are lower_snake_case",
		newAdapter(buflintcheck.CheckOneofLowerSnakeCase),
	)
	// PackageDefinedCheckerBuilder is a checker builder.
	PackageDefinedCheckerBuilder = internal.NewNopCheckerBuilder(
		"PACKAGE_DEFINED",
		"all files have a package defined",
		newAdapter(buflintcheck.CheckPackageDefined),
	)
	// PackageDirectoryMatchCheckerBuilder is a checker builder.
	PackageDirectoryMatchCheckerBuilder = internal.NewNopCheckerBuilder(
		"PACKAGE_DIRECTORY_MATCH",
		"all files are in a directory that matches their package name",
		newAdapter(buflintcheck.CheckPackageDirectoryMatch),
	)
	// PackageLowerSnakeCaseCheckerBuilder is a checker builder.
	PackageLowerSnakeCaseCheckerBuilder = internal.NewNopCheckerBuilder(
		"PACKAGE_LOWER_SNAKE_CASE",
		"packages are lower_snake.case",
		newAdapter(buflintcheck.CheckPackageLowerSnakeCase),
	)
	// PackageSameCsharpNamespaceCheckerBuilder is a checker builder.
	PackageSameCsharpNamespaceCheckerBuilder = internal.NewNopCheckerBuilder(
		"PACKAGE_SAME_CSHARP_NAMESPACE",
		"all files with a given package have the same value for the csharp_namespace option",
		newAdapter(buflintcheck.CheckPackageSameCsharpNamespace),
	)
	// PackageSameDirectoryCheckerBuilder is a checker builder.
	PackageSameDirectoryCheckerBuilder = internal.NewNopCheckerBuilder(
		"PACKAGE_SAME_DIRECTORY",
		"all files with a given package are in the same directory",
		newAdapter(buflintcheck.CheckPackageSameDirectory),
	)
	// PackageSameGoPackageCheckerBuilder is a checker builder.
	PackageSameGoPackageCheckerBuilder = internal.NewNopCheckerBuilder(
		"PACKAGE_SAME_GO_PACKAGE",
		"all files with a given package have the same value for the go_package option",
		newAdapter(buflintcheck.CheckPackageSameGoPackage),
	)
	// PackageSameJavaMultipleFilesCheckerBuilder is a checker builder.
	PackageSameJavaMultipleFilesCheckerBuilder = internal.NewNopCheckerBuilder(
		"PACKAGE_SAME_JAVA_MULTIPLE_FILES",
		"all files with a given package have the same value for the java_multiple_files option",
		newAdapter(buflintcheck.CheckPackageSameJavaMultipleFiles),
	)
	// PackageSameJavaPackageCheckerBuilder is a checker builder.
	PackageSameJavaPackageCheckerBuilder = internal.NewNopCheckerBuilder(
		"PACKAGE_SAME_JAVA_PACKAGE",
		"all files with a given package have the same value for the java_package option",
		newAdapter(buflintcheck.CheckPackageSameJavaPackage),
	)
	// PackageSamePhpNamespaceCheckerBuilder is a checker builder.
	PackageSamePhpNamespaceCheckerBuilder = internal.NewNopCheckerBuilder(
		"PACKAGE_SAME_PHP_NAMESPACE",
		"all files with a given package have the same value for the php_namespace option",
		newAdapter(buflintcheck.CheckPackageSamePhpNamespace),
	)
	// PackageSameRubyPackageCheckerBuilder is a checker builder.
	PackageSameRubyPackageCheckerBuilder = internal.NewNopCheckerBuilder(
		"PACKAGE_SAME_RUBY_PACKAGE",
		"all files with a given package have the same value for the ruby_package option",
		newAdapter(buflintcheck.CheckPackageSameRubyPackage),
	)
	// PackageSameSwiftPrefixCheckerBuilder is a checker builder.
	PackageSameSwiftPrefixCheckerBuilder = internal.NewNopCheckerBuilder(
		"PACKAGE_SAME_SWIFT_PREFIX",
		"all files with a given package have the same value for the swift_prefix option",
		newAdapter(buflintcheck.CheckPackageSameSwiftPrefix),
	)
	// PackageVersionSuffixCheckerBuilder is a checker builder.
	PackageVersionSuffixCheckerBuilder = internal.NewNopCheckerBuilder(
		"PACKAGE_VERSION_SUFFIX",
		`the last component of all packages is a version of the form v\d+, v\d+test.*, v\d+(alpha|beta)\d+, or v\d+p\d+(alpha|beta)\d+, where numbers are >=1`,
		newAdapter(buflintcheck.CheckPackageVersionSuffix),
	)
	// RPCNoClientStreamingCheckerBuilder is a checker builder.
	RPCNoClientStreamingCheckerBuilder = internal.NewNopCheckerBuilder(
		"RPC_NO_CLIENT_STREAMING",
		"RPCs are not client streaming",
		newAdapter(buflintcheck.CheckRPCNoClientStreaming),
	)
	// RPCNoServerStreamingCheckerBuilder is a checker builder.
	RPCNoServerStreamingCheckerBuilder = internal.NewNopCheckerBuilder(
		"RPC_NO_SERVER_STREAMING",
		"RPCs are not server streaming",
		newAdapter(buflintcheck.CheckRPCNoServerStreaming),
	)
	// RPCPascalCaseCheckerBuilder is a checker builder.
	RPCPascalCaseCheckerBuilder = internal.NewNopCheckerBuilder(
		"RPC_PASCAL_CASE",
		"RPCs are PascalCase",
		newAdapter(buflintcheck.CheckRPCPascalCase),
	)
	// RPCRequestResponseUniqueCheckerBuilder is a checker builder.
	RPCRequestResponseUniqueCheckerBuilder = internal.NewCheckerBuilder(
		"RPC_REQUEST_RESPONSE_UNIQUE",
		func(configBuilder internal.ConfigBuilder) (string, error) {
			return "RPC request and response types are only used in one RPC (configurable)", nil
		},
		func(configBuilder internal.ConfigBuilder) (internal.CheckFunc, error) {
			return internal.CheckFunc(func(id string, ignoreFunc internal.IgnoreFunc, _ []protosource.File, files []protosource.File) ([]bufanalysis.FileAnnotation, error) {
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
	// RPCRequestStandardNameCheckerBuilder is a checker builder.
	RPCRequestStandardNameCheckerBuilder = internal.NewCheckerBuilder(
		"RPC_REQUEST_STANDARD_NAME",
		func(configBuilder internal.ConfigBuilder) (string, error) {
			return "RPC request type names are RPCNameRequest or ServiceNameRPCNameRequest (configurable)", nil
		},
		func(configBuilder internal.ConfigBuilder) (internal.CheckFunc, error) {
			return internal.CheckFunc(func(id string, ignoreFunc internal.IgnoreFunc, _ []protosource.File, files []protosource.File) ([]bufanalysis.FileAnnotation, error) {
				return buflintcheck.CheckRPCRequestStandardName(
					id,
					ignoreFunc,
					files,
					configBuilder.RPCAllowGoogleProtobufEmptyRequests,
				)
			}), nil
		},
	)
	// RPCResponseStandardNameCheckerBuilder is a checker builder.
	RPCResponseStandardNameCheckerBuilder = internal.NewCheckerBuilder(
		"RPC_RESPONSE_STANDARD_NAME",
		func(configBuilder internal.ConfigBuilder) (string, error) {
			return "RPC response type names are RPCNameResponse or ServiceNameRPCNameResponse (configurable)", nil
		},
		func(configBuilder internal.ConfigBuilder) (internal.CheckFunc, error) {
			return internal.CheckFunc(func(id string, ignoreFunc internal.IgnoreFunc, _ []protosource.File, files []protosource.File) ([]bufanalysis.FileAnnotation, error) {
				return buflintcheck.CheckRPCResponseStandardName(
					id,
					ignoreFunc,
					files,
					configBuilder.RPCAllowGoogleProtobufEmptyResponses,
				)
			}), nil
		},
	)
	// ServicePascalCaseCheckerBuilder is a checker builder.
	ServicePascalCaseCheckerBuilder = internal.NewNopCheckerBuilder(
		"SERVICE_PASCAL_CASE",
		"services are PascalCase",
		newAdapter(buflintcheck.CheckServicePascalCase),
	)
	// ServiceSuffixCheckerBuilder is a checker builder.
	ServiceSuffixCheckerBuilder = internal.NewCheckerBuilder(
		"SERVICE_SUFFIX",
		func(configBuilder internal.ConfigBuilder) (string, error) {
			if configBuilder.ServiceSuffix == "" {
				return "", errors.New("service_suffix is empty")
			}
			return "services are suffixed with " + configBuilder.ServiceSuffix + " (suffix is configurable)", nil
		},
		func(configBuilder internal.ConfigBuilder) (internal.CheckFunc, error) {
			if configBuilder.ServiceSuffix == "" {
				return nil, errors.New("service_suffix is empty")
			}
			return internal.CheckFunc(func(id string, ignoreFunc internal.IgnoreFunc, _ []protosource.File, files []protosource.File) ([]bufanalysis.FileAnnotation, error) {
				return buflintcheck.CheckServiceSuffix(id, ignoreFunc, files, configBuilder.ServiceSuffix)
			}), nil
		},
	)
)

func newAdapter(
	f func(string, internal.IgnoreFunc, []protosource.File) ([]bufanalysis.FileAnnotation, error),
) func(string, internal.IgnoreFunc, []protosource.File, []protosource.File) ([]bufanalysis.FileAnnotation, error) {
	return func(id string, ignoreFunc internal.IgnoreFunc, _ []protosource.File, files []protosource.File) ([]bufanalysis.FileAnnotation, error) {
		return f(id, ignoreFunc, files)
	}
}
