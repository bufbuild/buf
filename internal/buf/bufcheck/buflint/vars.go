package buflint

import (
	"github.com/bufbuild/buf/internal/buf/bufcheck/buflint/internal"
	bufcheckinternal "github.com/bufbuild/buf/internal/buf/bufcheck/internal"
	"github.com/bufbuild/buf/internal/pkg/analysis"
	"github.com/bufbuild/buf/internal/pkg/errs"
	"github.com/bufbuild/buf/internal/pkg/protodesc"
)

var (
	// v1CheckerBuilders are the checker builders.
	v1CheckerBuilders = []*bufcheckinternal.CheckerBuilder{
		v1CommentEnumCheckerBuilder,
		v1CommentEnumValueCheckerBuilder,
		v1CommentFieldCheckerBuilder,
		v1CommentMessageCheckerBuilder,
		v1CommentOneofCheckerBuilder,
		v1CommentRPCCheckerBuilder,
		v1CommentServiceCheckerBuilder,
		v1DirectorySamePackageCheckerBuilder,
		v1EnumNoAllowAliasCheckerBuilder,
		v1EnumPascalCaseCheckerBuilder,
		v1EnumValuePrefixCheckerBuilder,
		v1EnumValueUpperSnakeCaseCheckerBuilder,
		v1EnumZeroValueSuffixCheckerBuilder,
		v1FieldLowerSnakeCaseCheckerBuilder,
		v1FieldNoDescriptorCheckerBuilder,
		v1FileLowerSnakeCaseCheckerBuilder,
		v1ImportNoPublicCheckerBuilder,
		v1ImportNoWeakCheckerBuilder,
		v1MessagePascalCaseCheckerBuilder,
		v1OneofLowerSnakeCaseCheckerBuilder,
		v1PackageDefinedCheckerBuilder,
		v1PackageDirectoryMatchCheckerBuilder,
		v1PackageLowerSnakeCaseCheckerBuilder,
		v1PackageSameCsharpNamespaceCheckerBuilder,
		v1PackageSameDirectoryCheckerBuilder,
		v1PackageSameGoPackageCheckerBuilder,
		v1PackageSameJavaMultipleFilesCheckerBuilder,
		v1PackageSameJavaPackageCheckerBuilder,
		v1PackageSamePhpNamespaceCheckerBuilder,
		v1PackageSameRubyPackageCheckerBuilder,
		v1PackageSameSwiftPrefixCheckerBuilder,
		v1PackageVersionSuffixCheckerBuilder,
		v1RPCNoClientStreamingCheckerBuilder,
		v1RPCNoServerStreamingCheckerBuilder,
		v1RPCPascalCaseCheckerBuilder,
		v1RPCRequestResponseUniqueCheckerBuilder,
		v1RPCRequestStandardNameCheckerBuilder,
		v1RPCResponseStandardNameCheckerBuilder,
		v1ServicePascalCaseCheckerBuilder,
		v1ServiceSuffixCheckerBuilder,
	}

	// v1DefaultCategories are the default categories.
	v1DefaultCategories = []string{
		"DEFAULT",
	}
	// v1AllCategories are all categories.
	v1AllCategories = []string{
		"MINIMAL",
		"BASIC",
		"DEFAULT",
		"COMMENTS",
		"UNARY_RPC",
		"FILE_LAYOUT",
		"PACKAGE_AFFINITY",
		"SENSIBLE",
		"STYLE_BASIC",
		"STYLE_DEFAULT",
	}
	// v1IDToCategories are the ID to categories.
	v1IDToCategories = map[string][]string{
		"COMMENT_ENUM": []string{
			"COMMENTS",
		},
		"COMMENT_ENUM_VALUE": []string{
			"COMMENTS",
		},
		"COMMENT_FIELD": []string{
			"COMMENTS",
		},
		"COMMENT_MESSAGE": []string{
			"COMMENTS",
		},
		"COMMENT_ONEOF": []string{
			"COMMENTS",
		},
		"COMMENT_RPC": []string{
			"COMMENTS",
		},
		"COMMENT_SERVICE": []string{
			"COMMENTS",
		},
		"DIRECTORY_SAME_PACKAGE": []string{
			"MINIMAL",
			"BASIC",
			"DEFAULT",
			"FILE_LAYOUT",
		},
		"ENUM_NO_ALLOW_ALIAS": []string{
			"MINIMAL",
			"BASIC",
			"DEFAULT",
			"SENSIBLE",
		},
		"ENUM_PASCAL_CASE": []string{
			"BASIC",
			"DEFAULT",
			"STYLE_BASIC",
			"STYLE_DEFAULT",
		},
		"ENUM_VALUE_PREFIX": []string{
			"DEFAULT",
			"STYLE_DEFAULT",
		},
		"ENUM_VALUE_UPPER_SNAKE_CASE": []string{
			"BASIC",
			"DEFAULT",
			"STYLE_BASIC",
			"STYLE_DEFAULT",
		},
		"ENUM_ZERO_VALUE_SUFFIX": []string{
			"DEFAULT",
			"STYLE_DEFAULT",
		},
		"FIELD_LOWER_SNAKE_CASE": []string{
			"BASIC",
			"DEFAULT",
			"STYLE_BASIC",
			"STYLE_DEFAULT",
		},
		"FIELD_NO_DESCRIPTOR": []string{
			"MINIMAL",
			"BASIC",
			"DEFAULT",
			"SENSIBLE",
		},
		"FILE_LOWER_SNAKE_CASE": []string{
			"DEFAULT",
			"STYLE_DEFAULT",
		},
		"IMPORT_NO_PUBLIC": []string{
			"MINIMAL",
			"BASIC",
			"DEFAULT",
			"SENSIBLE",
		},
		"IMPORT_NO_WEAK": []string{
			"MINIMAL",
			"BASIC",
			"DEFAULT",
			"SENSIBLE",
		},
		"MESSAGE_PASCAL_CASE": []string{
			"BASIC",
			"DEFAULT",
			"STYLE_BASIC",
			"STYLE_DEFAULT",
		},
		"ONEOF_LOWER_SNAKE_CASE": []string{
			"BASIC",
			"DEFAULT",
			"STYLE_BASIC",
			"STYLE_DEFAULT",
		},
		"PACKAGE_DEFINED": []string{
			"MINIMAL",
			"BASIC",
			"DEFAULT",
			"SENSIBLE",
		},
		"PACKAGE_DIRECTORY_MATCH": []string{
			"MINIMAL",
			"BASIC",
			"DEFAULT",
			"FILE_LAYOUT",
		},
		"PACKAGE_LOWER_SNAKE_CASE": []string{
			"BASIC",
			"DEFAULT",
			"STYLE_BASIC",
			"STYLE_DEFAULT",
		},
		"PACKAGE_SAME_CSHARP_NAMESPACE": []string{
			"MINIMAL",
			"BASIC",
			"DEFAULT",
			"PACKAGE_AFFINITY",
		},
		"PACKAGE_SAME_DIRECTORY": []string{
			"MINIMAL",
			"BASIC",
			"DEFAULT",
			"FILE_LAYOUT",
		},
		"PACKAGE_SAME_GO_PACKAGE": []string{
			"MINIMAL",
			"BASIC",
			"DEFAULT",
			"PACKAGE_AFFINITY",
		},
		"PACKAGE_SAME_JAVA_MULTIPLE_FILES": []string{
			"MINIMAL",
			"BASIC",
			"DEFAULT",
			"PACKAGE_AFFINITY",
		},
		"PACKAGE_SAME_JAVA_PACKAGE": []string{
			"MINIMAL",
			"BASIC",
			"DEFAULT",
			"PACKAGE_AFFINITY",
		},
		"PACKAGE_SAME_PHP_NAMESPACE": []string{
			"MINIMAL",
			"BASIC",
			"DEFAULT",
			"PACKAGE_AFFINITY",
		},
		"PACKAGE_SAME_RUBY_PACKAGE": []string{
			"MINIMAL",
			"BASIC",
			"DEFAULT",
			"PACKAGE_AFFINITY",
		},
		"PACKAGE_SAME_SWIFT_PREFIX": []string{
			"MINIMAL",
			"BASIC",
			"DEFAULT",
			"PACKAGE_AFFINITY",
		},
		"PACKAGE_VERSION_SUFFIX": []string{
			"DEFAULT",
			"STYLE_DEFAULT",
		},
		"RPC_NO_CLIENT_STREAMING": []string{
			"UNARY_RPC",
		},
		"RPC_NO_SERVER_STREAMING": []string{
			"UNARY_RPC",
		},
		"RPC_PASCAL_CASE": []string{
			"BASIC",
			"DEFAULT",
			"STYLE_BASIC",
			"STYLE_DEFAULT",
		},
		"RPC_REQUEST_RESPONSE_UNIQUE": []string{
			"DEFAULT",
			"STYLE_DEFAULT",
		},
		"RPC_REQUEST_STANDARD_NAME": []string{
			"DEFAULT",
			"STYLE_DEFAULT",
		},
		"RPC_RESPONSE_STANDARD_NAME": []string{
			"DEFAULT",
			"STYLE_DEFAULT",
		},
		"SERVICE_PASCAL_CASE": []string{
			"BASIC",
			"DEFAULT",
			"STYLE_BASIC",
			"STYLE_DEFAULT",
		},
		"SERVICE_SUFFIX": []string{
			"DEFAULT",
			"STYLE_DEFAULT",
		},
	}

	v1CommentEnumCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"COMMENT_ENUM",
		"enums have non-empty comments",
		newAdapter(internal.CheckCommentEnum),
	)
	v1CommentEnumValueCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"COMMENT_ENUM_VALUE",
		"enum values have non-empty comments",
		newAdapter(internal.CheckCommentEnumValue),
	)
	v1CommentFieldCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"COMMENT_FIELD",
		"fields have non-empty comments",
		newAdapter(internal.CheckCommentField),
	)
	v1CommentMessageCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"COMMENT_MESSAGE",
		"messages have non-empty comments",
		newAdapter(internal.CheckCommentMessage),
	)
	v1CommentOneofCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"COMMENT_ONEOF",
		"oneof have non-empty comments",
		newAdapter(internal.CheckCommentOneof),
	)
	v1CommentRPCCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"COMMENT_RPC",
		"RPCs have non-empty comments",
		newAdapter(internal.CheckCommentRPC),
	)
	v1CommentServiceCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"COMMENT_SERVICE",
		"services have non-empty comments",
		newAdapter(internal.CheckCommentService),
	)
	v1DirectorySamePackageCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"DIRECTORY_SAME_PACKAGE",
		"all files in a given directory are in the same package",
		newAdapter(internal.CheckDirectorySamePackage),
	)
	v1EnumNoAllowAliasCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"ENUM_NO_ALLOW_ALIAS",
		"enums do not have the allow_alias option set",
		newAdapter(internal.CheckEnumNoAllowAlias),
	)
	v1EnumPascalCaseCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"ENUM_PASCAL_CASE",
		"enums are PascalCase",
		newAdapter(internal.CheckEnumPascalCase),
	)
	v1EnumValuePrefixCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"ENUM_VALUE_PREFIX",
		"enum values are prefixed with ENUM_NAME_UPPER_SNAKE_CASE",
		newAdapter(internal.CheckEnumValuePrefix),
	)
	v1EnumValueUpperSnakeCaseCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"ENUM_VALUE_UPPER_SNAKE_CASE",
		"enum values are UPPER_SNAKE_CASE",
		newAdapter(internal.CheckEnumValueUpperSnakeCase),
	)
	v1EnumZeroValueSuffixCheckerBuilder = bufcheckinternal.NewCheckerBuilder(
		"ENUM_ZERO_VALUE_SUFFIX",
		func(configBuilder bufcheckinternal.ConfigBuilder) (string, error) {
			if configBuilder.EnumZeroValueSuffix == "" {
				return "", errs.NewInternal("enum_zero_value_suffix is empty")
			}
			return "enum zero values are suffixed with " + configBuilder.EnumZeroValueSuffix + " (suffix is configurable)", nil
		},
		func(configBuilder bufcheckinternal.ConfigBuilder) (bufcheckinternal.CheckFunc, error) {
			if configBuilder.EnumZeroValueSuffix == "" {
				return nil, errs.NewInternal("enum_zero_value_suffix is empty")
			}
			return bufcheckinternal.CheckFunc(func(id string, _ []protodesc.File, files []protodesc.File) ([]*analysis.Annotation, error) {
				return internal.CheckEnumZeroValueSuffix(id, files, configBuilder.EnumZeroValueSuffix)
			}), nil
		},
	)
	v1FieldLowerSnakeCaseCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"FIELD_LOWER_SNAKE_CASE",
		"field names are lower_snake_case",
		newAdapter(internal.CheckFieldLowerSnakeCase),
	)
	v1FieldNoDescriptorCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"FIELD_NO_DESCRIPTOR",
		`field names are are not name capitalization of "descriptor" with any number of prefix or suffix underscores`,
		newAdapter(internal.CheckFieldNoDescriptor),
	)
	v1FileLowerSnakeCaseCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"FILE_LOWER_SNAKE_CASE",
		"filenames are lower_snake_case",
		newAdapter(internal.CheckFileLowerSnakeCase),
	)
	v1ImportNoPublicCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"IMPORT_NO_PUBLIC",
		"imports are not public",
		newAdapter(internal.CheckImportNoPublic),
	)
	v1ImportNoWeakCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"IMPORT_NO_WEAK",
		"imports are not weak",
		newAdapter(internal.CheckImportNoWeak),
	)
	v1MessagePascalCaseCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"MESSAGE_PASCAL_CASE",
		"messages are PascalCase",
		newAdapter(internal.CheckMessagePascalCase),
	)
	v1OneofLowerSnakeCaseCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"ONEOF_LOWER_SNAKE_CASE",
		"oneof names are lower_snake_case",
		newAdapter(internal.CheckOneofLowerSnakeCase),
	)
	v1PackageDefinedCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"PACKAGE_DEFINED",
		"all files with have a package defined",
		newAdapter(internal.CheckPackageDefined),
	)
	v1PackageDirectoryMatchCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"PACKAGE_DIRECTORY_MATCH",
		"all files with are in a directory that matches their package name",
		newAdapter(internal.CheckPackageDirectoryMatch),
	)
	v1PackageLowerSnakeCaseCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"PACKAGE_LOWER_SNAKE_CASE",
		"packages are lower_snake.case",
		newAdapter(internal.CheckPackageLowerSnakeCase),
	)
	v1PackageSameCsharpNamespaceCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"PACKAGE_SAME_CSHARP_NAMESPACE",
		"all files with a given package have the same value for the csharp_namespace option",
		newAdapter(internal.CheckPackageSameCsharpNamespace),
	)
	v1PackageSameDirectoryCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"PACKAGE_SAME_DIRECTORY",
		"all files with a given package are in the same directory",
		newAdapter(internal.CheckPackageSameDirectory),
	)
	v1PackageSameGoPackageCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"PACKAGE_SAME_GO_PACKAGE",
		"all files with a given package have the same value for the go_package option",
		newAdapter(internal.CheckPackageSameGoPackage),
	)
	v1PackageSameJavaMultipleFilesCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"PACKAGE_SAME_JAVA_MULTIPLE_FILES",
		"all files with a given package have the same value for the java_multiple_files option",
		newAdapter(internal.CheckPackageSameJavaMultipleFiles),
	)
	v1PackageSameJavaPackageCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"PACKAGE_SAME_JAVA_PACKAGE",
		"all files with a given package have the same value for the java_package option",
		newAdapter(internal.CheckPackageSameJavaPackage),
	)
	v1PackageSamePhpNamespaceCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"PACKAGE_SAME_PHP_NAMESPACE",
		"all files with a given package have the same value for the php_namespace option",
		newAdapter(internal.CheckPackageSamePhpNamespace),
	)
	v1PackageSameRubyPackageCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"PACKAGE_SAME_RUBY_PACKAGE",
		"all files with a given package have the same value for the ruby_package option",
		newAdapter(internal.CheckPackageSameRubyPackage),
	)
	v1PackageSameSwiftPrefixCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"PACKAGE_SAME_SWIFT_PREFIX",
		"all files with a given package have the same value for the swift_prefix option",
		newAdapter(internal.CheckPackageSameSwiftPrefix),
	)
	v1PackageVersionSuffixCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"PACKAGE_VERSION_SUFFIX",
		`the last component of all packages is a version of the form v\d+, v\d+test.*, v\d+(alpha|beta)\d+, or v\d+p\d+(alpha|beta)\d+, where numbers are >=1`,
		newAdapter(internal.CheckPackageVersionSuffix),
	)
	v1RPCNoClientStreamingCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"RPC_NO_CLIENT_STREAMING",
		"RPCs are not client streaming",
		newAdapter(internal.CheckRPCNoClientStreaming),
	)
	v1RPCNoServerStreamingCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"RPC_NO_SERVER_STREAMING",
		"RPCs are not server streaming",
		newAdapter(internal.CheckRPCNoServerStreaming),
	)
	v1RPCPascalCaseCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"RPC_PASCAL_CASE",
		"RPCs are PascalCase",
		newAdapter(internal.CheckRPCPascalCase),
	)
	v1RPCRequestResponseUniqueCheckerBuilder = bufcheckinternal.NewCheckerBuilder(
		"RPC_REQUEST_RESPONSE_UNIQUE",
		func(configBuilder bufcheckinternal.ConfigBuilder) (string, error) {
			return "RPCs request and response types are only used in one RPC (configurable)", nil
		},
		func(configBuilder bufcheckinternal.ConfigBuilder) (bufcheckinternal.CheckFunc, error) {
			return bufcheckinternal.CheckFunc(func(id string, _ []protodesc.File, files []protodesc.File) ([]*analysis.Annotation, error) {
				return internal.CheckRPCRequestResponseUnique(
					id,
					files,
					configBuilder.RPCAllowSameRequestResponse,
					configBuilder.RPCAllowGoogleProtobufEmptyRequests,
					configBuilder.RPCAllowGoogleProtobufEmptyResponses,
				)
			}), nil
		},
	)
	v1RPCRequestStandardNameCheckerBuilder = bufcheckinternal.NewCheckerBuilder(
		"RPC_REQUEST_STANDARD_NAME",
		func(configBuilder bufcheckinternal.ConfigBuilder) (string, error) {
			return "RPC request type names are RPCNameRequest or ServiceNameRPCNameRequest (configurable)", nil
		},
		func(configBuilder bufcheckinternal.ConfigBuilder) (bufcheckinternal.CheckFunc, error) {
			return bufcheckinternal.CheckFunc(func(id string, _ []protodesc.File, files []protodesc.File) ([]*analysis.Annotation, error) {
				return internal.CheckRPCRequestStandardName(
					id,
					files,
					configBuilder.RPCAllowGoogleProtobufEmptyRequests,
				)
			}), nil
		},
	)
	v1RPCResponseStandardNameCheckerBuilder = bufcheckinternal.NewCheckerBuilder(
		"RPC_RESPONSE_STANDARD_NAME",
		func(configBuilder bufcheckinternal.ConfigBuilder) (string, error) {
			return "RPC response type names are RPCNameResponse or ServiceNameRPCNameResponse (configurable)", nil
		},
		func(configBuilder bufcheckinternal.ConfigBuilder) (bufcheckinternal.CheckFunc, error) {
			return bufcheckinternal.CheckFunc(func(id string, _ []protodesc.File, files []protodesc.File) ([]*analysis.Annotation, error) {
				return internal.CheckRPCResponseStandardName(
					id,
					files,
					configBuilder.RPCAllowGoogleProtobufEmptyResponses,
				)
			}), nil
		},
	)
	v1ServicePascalCaseCheckerBuilder = bufcheckinternal.NewNopCheckerBuilder(
		"SERVICE_PASCAL_CASE",
		"services are PascalCase",
		newAdapter(internal.CheckServicePascalCase),
	)
	v1ServiceSuffixCheckerBuilder = bufcheckinternal.NewCheckerBuilder(
		"SERVICE_SUFFIX",
		func(configBuilder bufcheckinternal.ConfigBuilder) (string, error) {
			if configBuilder.ServiceSuffix == "" {
				return "", errs.NewInternal("service_suffix is empty")
			}
			return "services are suffixed with " + configBuilder.ServiceSuffix + " (suffix is configurable)", nil
		},
		func(configBuilder bufcheckinternal.ConfigBuilder) (bufcheckinternal.CheckFunc, error) {
			if configBuilder.ServiceSuffix == "" {
				return nil, errs.NewInternal("service_suffix is empty")
			}
			return bufcheckinternal.CheckFunc(func(id string, _ []protodesc.File, files []protodesc.File) ([]*analysis.Annotation, error) {
				return internal.CheckServiceSuffix(id, files, configBuilder.ServiceSuffix)
			}), nil
		},
	)
)

func newAdapter(
	f func(string, []protodesc.File) ([]*analysis.Annotation, error),
) func(string, []protodesc.File, []protodesc.File) ([]*analysis.Annotation, error) {
	return func(id string, _ []protodesc.File, files []protodesc.File) ([]*analysis.Annotation, error) {
		return f(id, files)
	}
}
