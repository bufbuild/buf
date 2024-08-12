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
	"github.com/bufbuild/buf/private/buf/bufcheckserver/internal/bufcheckserverhandle"
	"github.com/bufbuild/buf/private/buf/bufcheckserver/internal/bufcheckserverutil"
	"github.com/bufbuild/bufplugin-go/check"
)

var (
	// BreakingEnumSameTypeRuleSpecBuilder is a rule builder.
	BreakingEnumSameTypeRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "ENUM_SAME_TYPE",
		Purpose: "Checks that enums have the same type (open vs closed)",
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
)
