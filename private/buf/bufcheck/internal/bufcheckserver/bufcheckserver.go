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
	"context"

	"github.com/bufbuild/buf/private/buf/bufcheck/internal/bufcheckserver/internal/bufcheckserverbuild"
	"github.com/bufbuild/buf/private/buf/bufcheck/internal/bufcheckserver/internal/bufcheckserverutil"
	"github.com/bufbuild/bufplugin-go/check"
)

var (
	// V1Beta1Spec is the v1beta1 check.Spec.
	V1Beta1Spec = &check.Spec{
		// TODO
		Rules: []*check.RuleSpec{
			{
				ID:      "PLACEHOLDER",
				Purpose: "Have a single RuleSpec so that all of the code downstream doesn't error.",
				Type:    check.RuleTypeLint,
				Handler: check.RuleHandlerFunc(
					func(context.Context, check.ResponseWriter, check.Request) error {
						return nil
					},
				),
			},
		},
		Before: bufcheckserverutil.Before,
	}

	// V1Spec is the v1beta1 check.Spec.
	V1Spec = &check.Spec{
		// TODO
		Rules: []*check.RuleSpec{
			{
				ID:      "PLACEHOLDER",
				Purpose: "Have a single RuleSpec so that all of the code downstream doesn't error.",
				Type:    check.RuleTypeLint,
				Handler: check.RuleHandlerFunc(
					func(context.Context, check.ResponseWriter, check.Request) error {
						return nil
					},
				),
			},
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
			bufcheckserverbuild.BreakingFieldSameDefaultRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON"}),
			bufcheckserverbuild.BreakingFieldSameOneofRuleSpecBuilder.Build(true, []string{"FILE", "PACKAGE", "WIRE_JSON"}),

			bufcheckserverbuild.LintCommentEnumRuleSpecBuilder.Build(false, []string{"COMMENTS"}),
			bufcheckserverbuild.LintCommentEnumValueRuleSpecBuilder.Build(false, []string{"COMMENTS"}),
			bufcheckserverbuild.LintCommentFieldRuleSpecBuilder.Build(false, []string{"COMMENTS"}),
			bufcheckserverbuild.LintCommentMessageRuleSpecBuilder.Build(false, []string{"COMMENTS"}),
			bufcheckserverbuild.LintCommentOneofRuleSpecBuilder.Build(false, []string{"COMMENTS"}),
			bufcheckserverbuild.LintCommentRPCRuleSpecBuilder.Build(false, []string{"COMMENTS"}),
			bufcheckserverbuild.LintCommentServiceRuleSpecBuilder.Build(false, []string{"COMMENTS"}),
			bufcheckserverbuild.LintDirectorySamePackageRuleSpecBuilder.Build(true, []string{"MINIMAL", "BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintServicePascalCaseRuleSpecBuilder.Build(true, []string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintServiceSuffixRuleSpecBuilder.Build(true, []string{"DEFAULT"}),
		},
		Before: bufcheckserverutil.Before,
	}
)
