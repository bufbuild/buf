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

package bufcheckserverv2

import (
	"github.com/bufbuild/buf/private/buf/bufcheckserver/internal/bufcheckserverbuild"
	"github.com/bufbuild/buf/private/buf/bufcheckserver/internal/bufcheckserverutil"
	"github.com/bufbuild/bufplugin-go/check"
)

var (
	// Spec is the v2 check.Spec.
	Spec = &check.Spec{
		Rules: []*check.RuleSpec{
			bufcheckserverbuild.BreakingEnumSameTypeRuleSpecBuilder.Build([]string{"FILE", "PACKAGE"}),
			bufcheckserverbuild.LintCommentEnumRuleSpecBuilder.Build([]string{"COMMENTS"}),
			bufcheckserverbuild.LintCommentEnumValueRuleSpecBuilder.Build([]string{"COMMENTS"}),
			bufcheckserverbuild.LintCommentFieldRuleSpecBuilder.Build([]string{"COMMENTS"}),
			bufcheckserverbuild.LintCommentMessageRuleSpecBuilder.Build([]string{"COMMENTS"}),
			bufcheckserverbuild.LintCommentOneofRuleSpecBuilder.Build([]string{"COMMENTS"}),
			bufcheckserverbuild.LintCommentRPCRuleSpecBuilder.Build([]string{"COMMENTS"}),
			bufcheckserverbuild.LintCommentServiceRuleSpecBuilder.Build([]string{"COMMENTS"}),
			bufcheckserverbuild.LintDirectorySamePackageRuleSpecBuilder.Build([]string{"MINIMAL", "BASIC"}), // leaving out DEFAULT
			bufcheckserverbuild.LintServicePascalCaseRuleSpecBuilder.Build([]string{"BASIC", "DEFAULT"}),
			bufcheckserverbuild.LintServiceSuffixRuleSpecBuilder.Build([]string{"DEFAULT"}),
		},
		Before: bufcheckserverutil.Before,
	}
)
