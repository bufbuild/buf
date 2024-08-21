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

package bufcheckserver

import (
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/internal/bufcheckserver/internal/bufcheckserverbuild"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/internal/bufcheckserver/internal/bufcheckserverutil"
	"github.com/bufbuild/bufplugin-go/check"
)

var (
	// V1Beta1Spec is the v1beta1 check.Spec.
	V1Beta1Spec = &check.Spec{
		Rules: []*check.RuleSpec{
			bufcheckserverbuild.BreakingEnumNoDeleteRuleSpecBuilder.Build(true, []string{"FILE"}),
			bufcheckserverbuild.BreakingFileNoDeleteRuleSpecBuilder.Build(true, []string{"FILE"}),
			bufcheckserverbuild.BreakingMessageNoDeleteRuleSpecBuilder.Build(true, []string{"FILE"}),
			bufcheckserverbuild.BreakingServiceNoDeleteRuleSpecBuilder.Build(true, []string{"FILE"}),
			bufcheckserverbuild.BreakingEnumSameTypeRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingEnumValueNoDeleteRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingExtensionMessageNoDeleteRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFieldNoDeleteRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFieldSameCardinalityRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingFieldSameCppStringTypeRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFieldSameJavaUTF8ValidationRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFieldSameJSTypeRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFieldSameTypeRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingFieldSameUTF8ValidationRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameCcEnableArenasRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameCcGenericServicesRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameCsharpNamespaceRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameGoPackageRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameJavaGenericServicesRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameJavaMultipleFilesRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameJavaOuterClassnameRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameJavaPackageRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameObjcClassPrefixRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameOptimizeForRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSamePhpClassPrefixRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSamePhpMetadataNamespaceRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSamePhpNamespaceRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSamePyGenericServicesRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameRubyPackageRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameSwiftPrefixRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameSyntaxRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingMessageNoRemoveStandardDescriptorAccessorRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingOneofNoDeleteRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingRPCNoDeleteRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingEnumSameJSONFormatRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON"}),
			bufcheckserverbuild.BreakingEnumValueSameNameRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON"}),
			bufcheckserverbuild.BreakingFieldSameJSONNameRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON"}),
			bufcheckserverbuild.BreakingFieldSameNameRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON"}),
			bufcheckserverbuild.BreakingMessageSameJSONFormatRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON"}),
			bufcheckserverbuild.BreakingFieldSameOneofRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingFileSamePackageRuleSpecBuilder.Build(true, []string{"FILE"}),
			bufcheckserverbuild.BreakingMessageSameRequiredFieldsRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingReservedEnumNoDeleteRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingReservedMessageNoDeleteRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingRPCSameClientStreamingRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingRPCSameIdempotencyLevelRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingRPCSameRequestTypeRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingRPCSameResponseTypeRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingRPCSameServerStreamingRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingPackageEnumNoDeleteRuleSpecBuilder.Build(false, []string{"PACKAGE"}),
			bufcheckserverbuild.BreakingPackageMessageNoDeleteRuleSpecBuilder.Build(false, []string{"PACKAGE"}),
			bufcheckserverbuild.BreakingPackageNoDeleteRuleSpecBuilder.Build(false, []string{"PACKAGE"}),
			bufcheckserverbuild.BreakingPackageServiceNoDeleteRuleSpecBuilder.Build(false, []string{"PACKAGE"}),
			bufcheckserverbuild.BreakingEnumValueNoDeleteUnlessNameReservedRuleSpecBuilder.Build(false, []string{"WIRE_JSON"}),
			bufcheckserverbuild.BreakingFieldNoDeleteUnlessNameReservedRuleSpecBuilder.Build(false, []string{"WIRE_JSON"}),
			bufcheckserverbuild.BreakingFieldWireJSONCompatibleCardinalityRuleSpecBuilder.Build(false, []string{"WIRE_JSON"}),
			bufcheckserverbuild.BreakingEnumValueNoDeleteUnlessNumberReservedRuleSpecBuilder.Build(false, []string{"WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingFieldNoDeleteUnlessNumberReservedRuleSpecBuilder.Build(false, []string{"WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingFieldWireCompatibleCardinalityRuleSpecBuilder.Build(false, []string{"WIRE"}),
			bufcheckserverbuild.BreakingFieldSameCTypeRuleSpecBuilder.Build(false, []string{}),
			bufcheckserverbuild.BreakingFieldSameLabelRuleSpecBuilder.Build(false, []string{}),
			bufcheckserverbuild.BreakingFileSameJavaStringCheckUtf8RuleSpecBuilder.Build(false, []string{}),
			bufcheckserverbuild.BreakingFileSamePhpGenericServicesRuleSpecBuilder.Build(false, []string{}),
			bufcheckserverbuild.BreakingMessageSameMessageSetWireFormatRuleSpecBuilder.Build(false, []string{}),
			bufcheckserverbuild.LintCommentEnumRuleSpecBuilder.Build(false, []string{"COMMENTS"}),
			bufcheckserverbuild.LintCommentEnumValueRuleSpecBuilder.Build(false, []string{"COMMENTS"}),
			bufcheckserverbuild.LintCommentFieldRuleSpecBuilder.Build(false, []string{"COMMENTS"}),
			bufcheckserverbuild.LintCommentMessageRuleSpecBuilder.Build(false, []string{"COMMENTS"}),
			bufcheckserverbuild.LintCommentOneofRuleSpecBuilder.Build(false, []string{"COMMENTS"}),
			bufcheckserverbuild.LintCommentRPCRuleSpecBuilder.Build(false, []string{"COMMENTS"}),
			bufcheckserverbuild.LintCommentServiceRuleSpecBuilder.Build(false, []string{"COMMENTS"}),
			bufcheckserverbuild.LintDirectorySamePackageRuleSpecBuilder.Build(true, []string{"MINIMAL", "BASIC", "DEFAULT", "FILE_LAYOUT"}),
			bufcheckserverbuild.LintEnumFirstValueZeroRuleSpecBuilder.Build(false, []string{"OTHER"}),
			bufcheckserverbuild.LintEnumNoAllowAliasRuleSpecBuilder.Build(true, []string{"MINIMAL", "BASIC", "DEFAULT", "SENSIBLE"}),
			bufcheckserverbuild.LintEnumPascalCaseRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT", "STYLE_BASIC", "STYLE_DEFAULT"}),
			bufcheckserverbuild.LintEnumValuePrefixRuleSpecBuilder.Build(true, []string{"DEFAULT", "STYLE_DEFAULT"}),
			bufcheckserverbuild.LintEnumValueUpperSnakeCaseRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT", "STYLE_BASIC", "STYLE_DEFAULT"}),
			bufcheckserverbuild.LintEnumZeroValueSuffixRuleSpecBuilder.Build(true, []string{"DEFAULT", "STYLE_DEFAULT"}),
			bufcheckserverbuild.LintFieldLowerSnakeCaseRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT", "STYLE_BASIC", "STYLE_DEFAULT"}),
			bufcheckserverbuild.LintFieldNoDescriptorRuleSpecBuilder.Build(true, []string{"MINIMAL", "BASIC", "DEFAULT", "SENSIBLE"}),
			bufcheckserverbuild.LintFileLowerSnakeCaseRuleSpecBuilder.Build(true, []string{"DEFAULT", "STYLE_DEFAULT"}),
			bufcheckserverbuild.LintImportNoPublicRuleSpecBuilder.Build(true, []string{"MINIMAL", "BASIC", "DEFAULT", "SENSIBLE"}),
			bufcheckserverbuild.LintImportNoWeakRuleSpecBuilder.Build(true, []string{"MINIMAL", "BASIC", "DEFAULT", "SENSIBLE"}),
			bufcheckserverbuild.LintMessagePascalCaseRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT", "STYLE_BASIC", "STYLE_DEFAULT"}),
			bufcheckserverbuild.LintOneofLowerSnakeCaseRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT", "STYLE_BASIC", "STYLE_DEFAULT"}),
			bufcheckserverbuild.LintPackageDefinedRuleSpecBuilder.Build(true, []string{"MINIMAL", "BASIC", "DEFAULT", "SENSIBLE"}),
			bufcheckserverbuild.LintPackageDirectoryMatchRuleSpecBuilder.Build(true, []string{"MINIMAL", "BASIC", "DEFAULT", "FILE_LAYOUT"}),
			bufcheckserverbuild.LintPackageLowerSnakeCaseRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT", "STYLE_BASIC", "STYLE_DEFAULT"}),
			bufcheckserverbuild.LintPackageSameCsharpNamespaceRuleSpecBuilder.Build(true, []string{"MINIMAL", "BASIC", "DEFAULT", "PACKAGE_AFFINITY"}),
			bufcheckserverbuild.LintPackageSameDirectoryRuleSpecBuilder.Build(true, []string{"MINIMAL", "BASIC", "DEFAULT", "FILE_LAYOUT"}),
			bufcheckserverbuild.LintPackageSameGoPackageRuleSpecBuilder.Build(true, []string{"MINIMAL", "BASIC", "DEFAULT", "PACKAGE_AFFINITY"}),
			bufcheckserverbuild.LintPackageSameJavaMultipleFilesRuleSpecBuilder.Build(true, []string{"MINIMAL", "BASIC", "DEFAULT", "PACKAGE_AFFINITY"}),
			bufcheckserverbuild.LintPackageSameJavaPackageRuleSpecBuilder.Build(true, []string{"MINIMAL", "BASIC", "DEFAULT", "PACKAGE_AFFINITY"}),
			bufcheckserverbuild.LintPackageSamePhpNamespaceRuleSpecBuilder.Build(true, []string{"MINIMAL", "BASIC", "DEFAULT", "PACKAGE_AFFINITY"}),
			bufcheckserverbuild.LintPackageSameRubyPackageRuleSpecBuilder.Build(true, []string{"MINIMAL", "BASIC", "DEFAULT", "PACKAGE_AFFINITY"}),
			bufcheckserverbuild.LintPackageSameSwiftPrefixRuleSpecBuilder.Build(true, []string{"MINIMAL", "BASIC", "DEFAULT", "PACKAGE_AFFINITY"}),
			bufcheckserverbuild.LintPackageVersionSuffixRuleSpecBuilder.Build(true, []string{"DEFAULT", "STYLE_DEFAULT"}),
			bufcheckserverbuild.LintRPCNoClientStreamingRuleSpecBuilder.Build(false, []string{"UNARY_RPC"}),
			bufcheckserverbuild.LintRPCNoServerStreamingRuleSpecBuilder.Build(false, []string{"UNARY_RPC"}),
			bufcheckserverbuild.LintRPCPascalCaseRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT", "STYLE_BASIC", "STYLE_DEFAULT"}),
			bufcheckserverbuild.LintRPCRequestResponseUniqueRuleSpecBuilder.Build(true, []string{"DEFAULT", "STYLE_DEFAULT"}),
			bufcheckserverbuild.LintRPCRequestStandardNameRuleSpecBuilder.Build(true, []string{"DEFAULT", "STYLE_DEFAULT"}),
			bufcheckserverbuild.LintRPCResponseStandardNameRuleSpecBuilder.Build(true, []string{"DEFAULT", "STYLE_DEFAULT"}),
			bufcheckserverbuild.LintServicePascalCaseRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT", "STYLE_BASIC", "STYLE_DEFAULT"}),
			bufcheckserverbuild.LintServiceSuffixRuleSpecBuilder.Build(true, []string{"DEFAULT", "STYLE_DEFAULT"}),
		},
		Categories: []*check.CategorySpec{
			bufcheckserverbuild.FileCategorySpec,
			bufcheckserverbuild.PackageCategorySpec,
			bufcheckserverbuild.WireCategorySpec,
			bufcheckserverbuild.WireJSONCategorySpec,
			bufcheckserverbuild.BasicCategorySpec,
			bufcheckserverbuild.CommentsCategorySpec,
			bufcheckserverbuild.DefaultCategorySpec,
			bufcheckserverbuild.FileLayoutCategorySpec,
			bufcheckserverbuild.MinimalCategorySpec,
			bufcheckserverbuild.OtherCategorySpec,
			bufcheckserverbuild.PackageAffinityCategorySpec,
			bufcheckserverbuild.SensibleCategorySpec,
			bufcheckserverbuild.StyleBasicCategorySpec,
			bufcheckserverbuild.StyleDefaultCategorySpec,
			bufcheckserverbuild.UnaryRPCCategorySpec,
		},
		Before: bufcheckserverutil.Before,
	}

	// V1Spec is the v1beta1 check.Spec.
	V1Spec = &check.Spec{
		Rules: []*check.RuleSpec{
			bufcheckserverbuild.BreakingEnumNoDeleteRuleSpecBuilder.Build(true, []string{"FILE"}),
			bufcheckserverbuild.BreakingFileNoDeleteRuleSpecBuilder.Build(true, []string{"FILE"}),
			bufcheckserverbuild.BreakingMessageNoDeleteRuleSpecBuilder.Build(true, []string{"FILE"}),
			bufcheckserverbuild.BreakingServiceNoDeleteRuleSpecBuilder.Build(true, []string{"FILE"}),
			bufcheckserverbuild.BreakingEnumSameTypeRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingEnumValueNoDeleteRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingExtensionMessageNoDeleteRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFieldNoDeleteRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFieldSameCardinalityRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFieldSameCppStringTypeRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFieldSameJavaUTF8ValidationRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFieldSameJSTypeRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFieldSameTypeRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFieldSameUTF8ValidationRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameCcEnableArenasRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameCcGenericServicesRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameCsharpNamespaceRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameGoPackageRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameJavaGenericServicesRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameJavaMultipleFilesRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameJavaOuterClassnameRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameJavaPackageRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameObjcClassPrefixRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameOptimizeForRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSamePhpClassPrefixRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSamePhpMetadataNamespaceRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSamePhpNamespaceRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSamePyGenericServicesRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameRubyPackageRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameSwiftPrefixRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameSyntaxRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingMessageNoRemoveStandardDescriptorAccessorRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingOneofNoDeleteRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingRPCNoDeleteRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingEnumSameJSONFormatRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON"}),
			bufcheckserverbuild.BreakingEnumValueSameNameRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON"}),
			bufcheckserverbuild.BreakingFieldSameJSONNameRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON"}),
			bufcheckserverbuild.BreakingFieldSameNameRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON"}),
			bufcheckserverbuild.BreakingMessageSameJSONFormatRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON"}),
			bufcheckserverbuild.BreakingFieldSameOneofRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingFileSamePackageRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingMessageSameRequiredFieldsRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingReservedEnumNoDeleteRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingReservedMessageNoDeleteRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingRPCSameClientStreamingRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingRPCSameIdempotencyLevelRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingRPCSameRequestTypeRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingRPCSameResponseTypeRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingRPCSameServerStreamingRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingPackageEnumNoDeleteRuleSpecBuilder.Build(false, []string{"PACKAGE"}),
			bufcheckserverbuild.BreakingPackageMessageNoDeleteRuleSpecBuilder.Build(false, []string{"PACKAGE"}),
			bufcheckserverbuild.BreakingPackageNoDeleteRuleSpecBuilder.Build(false, []string{"PACKAGE"}),
			bufcheckserverbuild.BreakingPackageServiceNoDeleteRuleSpecBuilder.Build(false, []string{"PACKAGE"}),
			bufcheckserverbuild.BreakingEnumValueNoDeleteUnlessNameReservedRuleSpecBuilder.Build(false, []string{"WIRE_JSON"}),
			bufcheckserverbuild.BreakingFieldNoDeleteUnlessNameReservedRuleSpecBuilder.Build(false, []string{"WIRE_JSON"}),
			bufcheckserverbuild.BreakingFieldWireJSONCompatibleCardinalityRuleSpecBuilder.Build(false, []string{"WIRE_JSON"}),
			bufcheckserverbuild.BreakingFieldWireJSONCompatibleTypeRuleSpecBuilder.Build(false, []string{"WIRE_JSON"}),
			bufcheckserverbuild.BreakingEnumValueNoDeleteUnlessNumberReservedRuleSpecBuilder.Build(false, []string{"WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingFieldNoDeleteUnlessNumberReservedRuleSpecBuilder.Build(false, []string{"WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingFieldWireCompatibleCardinalityRuleSpecBuilder.Build(false, []string{"WIRE"}),
			bufcheckserverbuild.BreakingFieldWireCompatibleTypeRuleSpecBuilder.Build(false, []string{"WIRE"}),
			bufcheckserverbuild.BreakingFieldSameCTypeRuleSpecBuilder.Build(false, []string{}),
			bufcheckserverbuild.BreakingFieldSameLabelRuleSpecBuilder.Build(false, []string{}),
			bufcheckserverbuild.BreakingMessageSameMessageSetWireFormatRuleSpecBuilder.Build(false, []string{}),
			bufcheckserverbuild.BreakingFileSameJavaStringCheckUtf8RuleSpecBuilder.Build(false, []string{}),
			bufcheckserverbuild.BreakingFileSamePhpGenericServicesRuleSpecBuilder.Build(false, []string{}),
			bufcheckserverbuild.LintCommentEnumRuleSpecBuilder.Build(false, []string{"COMMENTS"}),
			bufcheckserverbuild.LintCommentEnumValueRuleSpecBuilder.Build(false, []string{"COMMENTS"}),
			bufcheckserverbuild.LintCommentFieldRuleSpecBuilder.Build(false, []string{"COMMENTS"}),
			bufcheckserverbuild.LintCommentMessageRuleSpecBuilder.Build(false, []string{"COMMENTS"}),
			bufcheckserverbuild.LintCommentOneofRuleSpecBuilder.Build(false, []string{"COMMENTS"}),
			bufcheckserverbuild.LintCommentRPCRuleSpecBuilder.Build(false, []string{"COMMENTS"}),
			bufcheckserverbuild.LintCommentServiceRuleSpecBuilder.Build(false, []string{"COMMENTS"}),
			bufcheckserverbuild.LintDirectorySamePackageRuleSpecBuilder.Build(true, []string{"MINIMAL", "BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintEnumFirstValueZeroRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintEnumNoAllowAliasRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintEnumPascalCaseRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintEnumValuePrefixRuleSpecBuilder.Build(true, []string{"DEFAULT"}),
			bufcheckserverbuild.LintEnumValueUpperSnakeCaseRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintEnumZeroValueSuffixRuleSpecBuilder.Build(true, []string{"DEFAULT"}),
			bufcheckserverbuild.LintFieldLowerSnakeCaseRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintFileLowerSnakeCaseRuleSpecBuilder.Build(true, []string{"DEFAULT"}),
			bufcheckserverbuild.LintImportNoPublicRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintImportNoWeakRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintImportUsedRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintMessagePascalCaseRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintOneofLowerSnakeCaseRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintPackageDefinedRuleSpecBuilder.Build(true, []string{"MINIMAL", "BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintPackageDirectoryMatchRuleSpecBuilder.Build(true, []string{"MINIMAL", "BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintPackageLowerSnakeCaseRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintPackageNoImportCycleRuleSpecBuilder.Build(false, []string{}),
			bufcheckserverbuild.LintPackageSameCsharpNamespaceRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintPackageSameDirectoryRuleSpecBuilder.Build(true, []string{"MINIMAL", "BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintPackageSameGoPackageRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintPackageSameJavaMultipleFilesRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintPackageSameJavaPackageRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintPackageSamePhpNamespaceRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintPackageSameRubyPackageRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintPackageSameSwiftPrefixRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintPackageVersionSuffixRuleSpecBuilder.Build(true, []string{"DEFAULT"}),
			bufcheckserverbuild.LintProtovalidateRuleSpecBuilder.Build(true, []string{"DEFAULT"}),
			bufcheckserverbuild.LintRPCNoClientStreamingRuleSpecBuilder.Build(false, []string{"UNARY_RPC"}),
			bufcheckserverbuild.LintRPCNoServerStreamingRuleSpecBuilder.Build(false, []string{"UNARY_RPC"}),
			bufcheckserverbuild.LintRPCPascalCaseRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintRPCRequestResponseUniqueRuleSpecBuilder.Build(true, []string{"DEFAULT"}),
			bufcheckserverbuild.LintRPCRequestStandardNameRuleSpecBuilder.Build(true, []string{"DEFAULT"}),
			bufcheckserverbuild.LintRPCResponseStandardNameRuleSpecBuilder.Build(true, []string{"DEFAULT"}),
			bufcheckserverbuild.LintServicePascalCaseRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintServiceSuffixRuleSpecBuilder.Build(true, []string{"DEFAULT"}),
			bufcheckserverbuild.LintSyntaxSpecifiedRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
		},
		Categories: []*check.CategorySpec{
			bufcheckserverbuild.FileCategorySpec,
			bufcheckserverbuild.PackageCategorySpec,
			bufcheckserverbuild.WireCategorySpec,
			bufcheckserverbuild.WireJSONCategorySpec,
			bufcheckserverbuild.BasicCategorySpec,
			bufcheckserverbuild.CommentsCategorySpec,
			bufcheckserverbuild.DefaultCategorySpec,
			bufcheckserverbuild.MinimalCategorySpec,
			bufcheckserverbuild.UnaryRPCCategorySpec,
		},
		Before: bufcheckserverutil.Before,
	}

	// V2Spec is the v2 check.Spec.
	V2Spec = &check.Spec{
		Rules: []*check.RuleSpec{
			bufcheckserverbuild.BreakingEnumNoDeleteRuleSpecBuilder.Build(true, []string{"FILE"}),
			bufcheckserverbuild.BreakingExtensionNoDeleteRuleSpecBuilder.Build(true, []string{"FILE"}),
			bufcheckserverbuild.BreakingFileNoDeleteRuleSpecBuilder.Build(true, []string{"FILE"}),
			bufcheckserverbuild.BreakingMessageNoDeleteRuleSpecBuilder.Build(true, []string{"FILE"}),
			bufcheckserverbuild.BreakingServiceNoDeleteRuleSpecBuilder.Build(true, []string{"FILE"}),
			bufcheckserverbuild.BreakingEnumSameTypeRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingEnumValueNoDeleteRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingExtensionMessageNoDeleteRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFieldNoDeleteRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFieldSameCardinalityRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFieldSameCppStringTypeRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFieldSameJavaUTF8ValidationRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFieldSameJSTypeRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFieldSameTypeRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFieldSameUTF8ValidationRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameCcEnableArenasRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameCcGenericServicesRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameCsharpNamespaceRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameGoPackageRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameJavaGenericServicesRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameJavaMultipleFilesRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameJavaOuterClassnameRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameJavaPackageRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameObjcClassPrefixRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameOptimizeForRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSamePhpClassPrefixRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSamePhpMetadataNamespaceRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSamePhpNamespaceRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSamePyGenericServicesRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameRubyPackageRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameSwiftPrefixRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingFileSameSyntaxRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingMessageNoRemoveStandardDescriptorAccessorRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingOneofNoDeleteRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingRPCNoDeleteRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.BreakingEnumSameJSONFormatRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON"}),
			bufcheckserverbuild.BreakingEnumValueSameNameRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON"}),
			bufcheckserverbuild.BreakingFieldSameJSONNameRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON"}),
			bufcheckserverbuild.BreakingFieldSameNameRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON"}),
			bufcheckserverbuild.BreakingMessageSameJSONFormatRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON"}),
			bufcheckserverbuild.BreakingFieldSameDefaultRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingFieldSameOneofRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingFileSamePackageRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingMessageSameRequiredFieldsRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingReservedEnumNoDeleteRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingReservedMessageNoDeleteRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingRPCSameClientStreamingRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingRPCSameIdempotencyLevelRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingRPCSameRequestTypeRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingRPCSameResponseTypeRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingRPCSameServerStreamingRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingPackageEnumNoDeleteRuleSpecBuilder.Build(false, []string{"PACKAGE"}),
			bufcheckserverbuild.BreakingPackageExtensionNoDeleteRuleSpecBuilder.Build(false, []string{"PACKAGE"}),
			bufcheckserverbuild.BreakingPackageMessageNoDeleteRuleSpecBuilder.Build(false, []string{"PACKAGE"}),
			bufcheckserverbuild.BreakingPackageNoDeleteRuleSpecBuilder.Build(false, []string{"PACKAGE"}),
			bufcheckserverbuild.BreakingPackageServiceNoDeleteRuleSpecBuilder.Build(false, []string{"PACKAGE"}),
			bufcheckserverbuild.BreakingEnumValueNoDeleteUnlessNameReservedRuleSpecBuilder.Build(false, []string{"WIRE_JSON"}),
			bufcheckserverbuild.BreakingFieldNoDeleteUnlessNameReservedRuleSpecBuilder.Build(false, []string{"WIRE_JSON"}),
			bufcheckserverbuild.BreakingFieldWireJSONCompatibleCardinalityRuleSpecBuilder.Build(false, []string{"WIRE_JSON"}),
			bufcheckserverbuild.BreakingFieldWireJSONCompatibleTypeRuleSpecBuilder.Build(false, []string{"WIRE_JSON"}),
			bufcheckserverbuild.BreakingEnumValueNoDeleteUnlessNumberReservedRuleSpecBuilder.Build(false, []string{"WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingFieldNoDeleteUnlessNumberReservedRuleSpecBuilder.Build(false, []string{"WIRE_JSON", "WIRE"}),
			bufcheckserverbuild.BreakingFieldWireCompatibleCardinalityRuleSpecBuilder.Build(false, []string{"WIRE"}),
			bufcheckserverbuild.BreakingFieldWireCompatibleTypeRuleSpecBuilder.Build(false, []string{"WIRE"}),
			bufcheckserverbuild.BreakingMessageSameMessageSetWireFormatRuleSpecBuilder.Build(false, []string{}),
			bufcheckserverbuild.LintCommentEnumRuleSpecBuilder.Build(false, []string{"COMMENTS"}),
			bufcheckserverbuild.LintCommentEnumValueRuleSpecBuilder.Build(false, []string{"COMMENTS"}),
			bufcheckserverbuild.LintCommentFieldRuleSpecBuilder.Build(false, []string{"COMMENTS"}),
			bufcheckserverbuild.LintCommentMessageRuleSpecBuilder.Build(false, []string{"COMMENTS"}),
			bufcheckserverbuild.LintCommentOneofRuleSpecBuilder.Build(false, []string{"COMMENTS"}),
			bufcheckserverbuild.LintCommentRPCRuleSpecBuilder.Build(false, []string{"COMMENTS"}),
			bufcheckserverbuild.LintCommentServiceRuleSpecBuilder.Build(false, []string{"COMMENTS"}),
			bufcheckserverbuild.LintDirectorySamePackageRuleSpecBuilder.Build(true, []string{"MINIMAL", "BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintEnumFirstValueZeroRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintEnumNoAllowAliasRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintEnumPascalCaseRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintEnumValuePrefixRuleSpecBuilder.Build(true, []string{"DEFAULT"}),
			bufcheckserverbuild.LintEnumValueUpperSnakeCaseRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintEnumZeroValueSuffixRuleSpecBuilder.Build(true, []string{"DEFAULT"}),
			bufcheckserverbuild.LintFieldLowerSnakeCaseRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintFieldNotRequiredRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintFileLowerSnakeCaseRuleSpecBuilder.Build(true, []string{"DEFAULT"}),
			bufcheckserverbuild.LintImportNoPublicRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintImportNoWeakRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintImportUsedRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintMessagePascalCaseRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintOneofLowerSnakeCaseRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintPackageDefinedRuleSpecBuilder.Build(true, []string{"MINIMAL", "BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintPackageDirectoryMatchRuleSpecBuilder.Build(true, []string{"MINIMAL", "BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintPackageLowerSnakeCaseRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintPackageNoImportCycleRuleSpecBuilder.Build(true, []string{"MINIMAL", "BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintPackageSameCsharpNamespaceRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintPackageSameDirectoryRuleSpecBuilder.Build(true, []string{"MINIMAL", "BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintPackageSameGoPackageRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintPackageSameJavaMultipleFilesRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintPackageSameJavaPackageRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintPackageSamePhpNamespaceRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintPackageSameRubyPackageRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintPackageSameSwiftPrefixRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintPackageVersionSuffixRuleSpecBuilder.Build(true, []string{"DEFAULT"}),
			bufcheckserverbuild.LintProtovalidateRuleSpecBuilder.Build(true, []string{"DEFAULT"}),
			bufcheckserverbuild.LintRPCNoClientStreamingRuleSpecBuilder.Build(false, []string{"UNARY_RPC"}),
			bufcheckserverbuild.LintRPCNoServerStreamingRuleSpecBuilder.Build(false, []string{"UNARY_RPC"}),
			bufcheckserverbuild.LintRPCPascalCaseRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintRPCRequestResponseUniqueRuleSpecBuilder.Build(true, []string{"DEFAULT"}),
			bufcheckserverbuild.LintRPCRequestStandardNameRuleSpecBuilder.Build(true, []string{"DEFAULT"}),
			bufcheckserverbuild.LintRPCResponseStandardNameRuleSpecBuilder.Build(true, []string{"DEFAULT"}),
			bufcheckserverbuild.LintServicePascalCaseRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintServiceSuffixRuleSpecBuilder.Build(true, []string{"DEFAULT"}),
			bufcheckserverbuild.LintStablePackageNoImportUnstableRuleSpecBuilder.Build(false, []string{}),
			bufcheckserverbuild.LintSyntaxSpecifiedRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
		},
		Categories: []*check.CategorySpec{
			bufcheckserverbuild.FileCategorySpec,
			bufcheckserverbuild.PackageCategorySpec,
			bufcheckserverbuild.WireCategorySpec,
			bufcheckserverbuild.WireJSONCategorySpec,
			bufcheckserverbuild.BasicCategorySpec,
			bufcheckserverbuild.CommentsCategorySpec,
			bufcheckserverbuild.DefaultCategorySpec,
			bufcheckserverbuild.MinimalCategorySpec,
			bufcheckserverbuild.UnaryRPCCategorySpec,
		},
		Before: bufcheckserverutil.Before,
	}
)
