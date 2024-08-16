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
	"github.com/bufbuild/buf/private/buf/bufcheck/internal/bufcheckserver/internal/bufcheckserverbuild"
	"github.com/bufbuild/buf/private/buf/bufcheck/internal/bufcheckserver/internal/bufcheckserverutil"
	"github.com/bufbuild/bufplugin-go/check"
)

var (
	// V1Beta1Spec is the v1beta1 check.Spec.
	V1Beta1Spec = &check.Spec{
		Rules: []*check.RuleSpec{
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
		Before: bufcheckserverutil.Before,
	}

	// V1Spec is the v1beta1 check.Spec.
	V1Spec = &check.Spec{
		Rules: []*check.RuleSpec{
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
		Before: bufcheckserverutil.Before,
	}

	// V2Spec is the v2 check.Spec.
	V2Spec = &check.Spec{
		Rules: []*check.RuleSpec{
			bufcheckserverbuild.BreakingEnumSameTypeRuleSpecBuilder.Build(false, []string{"FILE", "PACKAGE"}),
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
		Before: bufcheckserverutil.Before,
	}
)
