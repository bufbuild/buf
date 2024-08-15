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
	// BreakingEnumSameTypeRuleSpecBuilder is a rule builder.
	BreakingEnumSameTypeRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "ENUM_SAME_TYPE",
		Purpose: "Checks that enums have the same type (open vs closed).",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingEnumSameType,
	}
	// LintCommentEnumRuleSpecBuilder is a rule builder.
	LintCommentEnumRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "COMMENT_ENUM",
		Purpose: "Checks that enums have non-empty comments.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintCommentEnum,
	}
	// LintCommentEnumValueRuleSpecBuilder is a rule builder.
	LintCommentEnumValueRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "COMMENT_ENUM_VALUE",
		Purpose: "Checks that enum values have non-empty comments.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintCommentEnumValue,
	}
	// LintCommentFieldRuleSpecBuilder is a rule builder.
	LintCommentFieldRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "COMMENT_FIELD",
		Purpose: "Checks that fields have non-empty comments.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintCommentField,
	}
	// LintCommentMessageRuleSpecBuilder is a rule builder.
	LintCommentMessageRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "COMMENT_MESSAGE",
		Purpose: "Checks that messages have non-empty comments.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintCommentMessage,
	}
	// LintCommentOneofRuleSpecBuilder is a rule builder.
	LintCommentOneofRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "COMMENT_ONEOF",
		Purpose: "Checks that oneofs have non-empty comments.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintCommentOneof,
	}
	// LintCommentRPCRuleSpecBuilder is a rule builder.
	LintCommentRPCRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "COMMENT_RPC",
		Purpose: "Checks that RPCs have non-empty comments.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintCommentRPC,
	}
	// LintCommentServiceRuleSpecBuilder is a rule builder.
	LintCommentServiceRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "COMMENT_SERVICE",
		Purpose: "Checks that services have non-empty comments.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintCommentService,
	}
	// LintDirectorySamePackageRuleSpecBuilder is a rule builder.
	LintDirectorySamePackageRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "DIRECTORY_SAME_PACKAGE",
		Purpose: "Checks that all files in a given directory are in the same package.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintDirectorySamePackage,
	}
	// EnumFirstValueZeroRuleBuilder is a rule builder.
	EnumFirstValueZeroRuleBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "ENUM_FIRST_VALUE_ZERO",
		Purpose: "Checks that all first values of enums have a numeric value of 0.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintEnumFirstValueZero,
	}
	// EnumNoAllowAliasRuleBuilder is a rule builder.
	EnumNoAllowAliasRuleBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "ENUM_NO_ALLOW_ALIAS",
		Purpose: "Checks that enums do not have the allow_alias option set.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintEnumNoAllowAlias,
	}
	// EnumPascalCaseRuleBuilder is a rule builder.
	EnumPascalCaseRuleBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "ENUM_PASCAL_CASE",
		Purpose: "Checks that enums are PascalCase.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintEnumPascalCase,
	}
	// EnumValuePrefixRuleBuilder is a rule builder.
	EnumValuePrefixRuleBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "ENUM_VALUE_PREFIX",
		Purpose: "Checks that enum values are prefixed with ENUM_NAME_UPPER_SNAKE_CASE.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintEnumValuePrefix,
	}
	// EnumValueUpperSnakeCaseRuleBuilder is a rule builder.
	EnumValueUpperSnakeCaseRuleBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "ENUM_VALUE_UPPER_SNAKE_CASE",
		Purpose: "Checks that enum values are UPPER_SNAKE_CASE.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintEnumValueUpperSnakeCase,
	}
	// EnumZeroValueSuffixRuleBuilder is a rule builder.
	EnumZeroValueSuffixRuleBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "ENUM_ZERO_VALUE_SUFFIX",
		Purpose: `Checks that enum zero values have a consistent suffix (configurable, default suffix is "_UNSPECIFIED").`,
		Type:    check.RuleTypeLint,
	}
	// FieldLowerSnakeCaseRuleBuilder is a rule builder.
	FieldLowerSnakeCaseRuleBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FIELD_LOWER_SNAKE_CASE",
		Purpose: "Checks that field names are lower_snake_case.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintFieldLowerSnakeCase,
	}
	// FieldNoDescriptorRuleBuilder is a rule builder.
	FieldNoDescriptorRuleBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FIELD_NO_DESCRIPTOR",
		Purpose: `Checks that field names are not any capitalization of "descriptor" with any number of prefix or suffix underscores.`,
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintFieldNoDescriptor,
	}
	// FieldNotRequiredRuleBuilder is a rule builder.
	FieldNotRequiredRuleBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FIELD_NOT_REQUIRED",
		Purpose: `Checks that fields are not configured to be required.`,
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintFieldNotRequired,
	}
	// FileLowerSnakeCaseRuleBuilder is a rule builder.
	FileLowerSnakeCaseRuleBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_LOWER_SNAKE_CASE",
		Purpose: "Checks that filenames are lower_snake_case.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintFileLowerSnakeCase,
	}
	// ImportNoPublicRuleBuilder is a rule builder.
	ImportNoPublicRuleBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "IMPORT_NO_PUBLIC",
		Purpose: "Checks that imports are not public.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintImportNoPublic,
	}
	// ImportNoWeakRuleBuilder is a rule builder.
	ImportNoWeakRuleBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "IMPORT_NO_WEAK",
		Purpose: "Checks that imports are not weak.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintImportNoWeak,
	}
	// ImportUsedRuleBuilder is a rule builder.
	ImportUsedRuleBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "IMPORT_USED",
		Purpose: "Checks that imports are used.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintImportUsed,
	}
	// MessagePascalCaseRuleBuilder is a rule builder.
	MessagePascalCaseRuleBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "MESSAGE_PASCAL_CASE",
		Purpose: "Checks that messages are PascalCase.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintMessagePascalCase,
	}
	// OneofLowerSnakeCaseRuleBuilder is a rule builder.
	OneofLowerSnakeCaseRuleBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "ONEOF_LOWER_SNAKE_CASE",
		Purpose: "Checks that oneof names are lower_snake_case.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintOneofLowerSnakeCase,
	}
	// PackageDefinedRuleBuilder is a rule builder.
	PackageDefinedRuleBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "PACKAGE_DEFINED",
		Purpose: "Checks that all files have a package defined.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintPackageDefined,
	}
	// PackageDirectoryMatchRuleBuilder is a rule builder.
	PackageDirectoryMatchRuleBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "PACKAGE_DIRECTORY_MATCH",
		Purpose: "Checks that all files are in a directory that matches their package name.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintPackageDirectoryMatch,
	}
	// PackageLowerSnakeCaseRuleBuilder is a rule builder.
	PackageLowerSnakeCaseRuleBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "PACKAGE_LOWER_SNAKE_CASE",
		Purpose: "Checks that packages are lower_snake.case.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintPackageLowerSnakeCase,
	}
	// PackageNoImportCycleRuleBuilder is a rule builder.
	PackageNoImportCycleRuleBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "PACKAGE_NO_IMPORT_CYCLE",
		Purpose: "Checks that packages do not have import cycles.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintPackageNoImportCycle,
	}
	// PackageSameCsharpNamespaceRuleBuilder is a rule builder.
	PackageSameCsharpNamespaceRuleBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "PACKAGE_SAME_CSHARP_NAMESPACE",
		Purpose: "Checks that all files with a given package have the same value for the csharp_namespace option.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintPackageSameCsharpNamespace,
	}
	// PackageSameDirectoryRuleBuilder is a rule builder.
	PackageSameDirectoryRuleBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "PACKAGE_SAME_DIRECTORY",
		Purpose: "Checks that all files with a given package are in the same directory.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintPackageSameDirectory,
	}
	// PackageSameGoPackageRuleBuilder is a rule builder.
	PackageSameGoPackageRuleBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "PACKAGE_SAME_GO_PACKAGE",
		Purpose: "Checks that all files with a given package have the same value for the go_package option.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintPackageSameGoPackage,
	}
	// PackageSameJavaMultipleFilesRuleBuilder is a rule builder.
	PackageSameJavaMultipleFilesRuleBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "PACKAGE_SAME_JAVA_MULTIPLE_FILES",
		Purpose: "Checks that all files with a given package have the same value for the java_multiple_files option.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintPackageSameJavaMultipleFiles,
	}
	// PackageSameJavaPackageRuleBuilder is a rule builder.
	PackageSameJavaPackageRuleBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "PACKAGE_SAME_JAVA_PACKAGE",
		Purpose: "Checks that all files with a given package have the same value for the java_package option.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintPackageSameJavaPackage,
	}
	// PackageSamePhpNamespaceRuleBuilder is a rule builder.
	PackageSamePhpNamespaceRuleBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "PACKAGE_SAME_PHP_NAMESPACE",
		Purpose: "Checks that all files with a given package have the same value for the php_namespace option.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintPackageSamePhpNamespace,
	}
	// PackageSameRubyPackageRuleBuilder is a rule builder.
	PackageSameRubyPackageRuleBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "PACKAGE_SAME_RUBY_PACKAGE",
		Purpose: "Checks that all files with a given package have the same value for the ruby_package option.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintPackageSameRubyPackage,
	}
	// PackageSameSwiftPrefixRuleBuilder is a rule builder.
	PackageSameSwiftPrefixRuleBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "PACKAGE_SAME_SWIFT_PREFIX",
		Purpose: "Checks that all files with a given package have the same value for the swift_prefix option.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintPackageSameSwiftPrefix,
	}
	// PackageVersionSuffixRuleBuilder is a rule builder.
	PackageVersionSuffixRuleBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "PACKAGE_VERSION_SUFFIX",
		Purpose: `Checks that the last component of all packages is a version of the form v\d+, v\d+test.*, v\d+(alpha|beta)\d+, or v\d+p\d+(alpha|beta)\d+, where numbers are >=1.`,
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintPackageVersionSuffix,
	}
	// ProtovalidateRuleBuilder is a rule builder.
	ProtovalidateRuleBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "PROTOVALIDATE",
		Purpose: "Checks that protovalidate rules are valid and all CEL expressions compile.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintProtovalidate,
	}
	// RPCNoClientStreamingRuleBuilder is a rule builder.
	RPCNoClientStreamingRuleBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "RPC_NO_CLIENT_STREAMING",
		Purpose: "Checks that RPCs are not client streaming.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintRPCNoClientStreaming,
	}
	// RPCNoServerStreamingRuleBuilder is a rule builder.
	RPCNoServerStreamingRuleBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "RPC_NO_SERVER_STREAMING",
		Purpose: "Checks that RPCs are not server streaming.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintRPCNoServerStreaming,
	}
	// RPCPascalCaseRuleBuilder is a rule builder.
	RPCPascalCaseRuleBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "RPC_PASCAL_CASE",
		Purpose: "Checks that RPCs are PascalCase.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintRPCPascalCase,
	}
	// RPCRequestResponseUniqueRuleBuilder is a rule builder.
	RPCRequestResponseUniqueRuleBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "RPC_REQUEST_RESPONSE_UNIQUE",
		Purpose: "Checks that RPC request and response types are only used in one RPC (configurable).",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintRPCRequestResponseUnique,
	}
	// RPCRequestStandardNameRuleBuilder is a rule builder.
	RPCRequestStandardNameRuleBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "RPC_REQUEST_STANDARD_NAME",
		Purpose: "Checks that RPC request type names are RPCNameRequest or ServiceNameRPCNameRequest (configurable).",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintRPCRequestStandardName,
	}
	// RPCResponseStandardNameRuleBuilder is a rule builder.
	RPCResponseStandardNameRuleBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "RPC_RESPONSE_STANDARD_NAME",
		Purpose: "Checks that RPC response type names are RPCNameResponse or ServiceNameRPCNameResponse (configurable).",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintRPCResponseStandardName,
	}
	// LintServicePascalCaseRuleSpecBuilder is a rule builder.
	LintServicePascalCaseRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "SERVICE_PASCAL_CASE",
		Purpose: "Checks that services are PascalCase.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintServicePascalCase,
	}
	// LintServiceSuffixRuleSpecBuilder is a rule builder.
	LintServiceSuffixRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "SERVICE_SUFFIX",
		Purpose: `Checks that services have a consistent suffix (configurable, default suffix is "Service").`,
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintServiceSuffix,
	}
	// StablePackageNoImportUnstableRuleBuilder is a rule builder.
	StablePackageNoImportUnstableRuleBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "STABLE_PACKAGE_NO_IMPORT_UNSTABLE",
		Purpose: "Checks that all files that have stable versioned packages do not import packages with unstable version packages.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintStablePackageNoImportUnstable,
	}
	// SyntaxSpecifiedRuleBuilder is a rule builder.
	SyntaxSpecifiedRuleBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "SYNTAX_SPECIFIED",
		Purpose: "Checks that all files have a syntax specified.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintSyntaxSpecified,
	}
)
