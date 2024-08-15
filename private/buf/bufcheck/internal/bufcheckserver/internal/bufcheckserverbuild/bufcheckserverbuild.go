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
	// BreakingEnumNoDeleteRuleSpecBuilder is a rule spec builder.
	BreakingEnumNoDeleteRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "ENUM_NO_DELETE",
		Purpose: "Checks enums are not deleted from a given file.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingEnumNoDelete,
	}
	// BreakingExtensionNoDeleteRuleSpecBuilder is a rule spec builder.
	BreakingExtensionNoDeleteRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "EXTENSION_NO_DELETE",
		Purpose: "Checks extensions are not deleted from a given file.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingExtensionNoDelete,
	}
	// BreakingFileNoDeleteRuleSpecBuilder is a rule spec builder.
	BreakingFileNoDeleteRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "FILE_NO_DELETE",
		Purpose: "Checks files are not deleted.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingFileNoDelete,
	}
	// BreakingMessageNoDeleteRuleSpecBuilder is a rule spec builder.
	BreakingMessageNoDeleteRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "MESSAGE_NO_DELETE",
		Purpose: "Checks messages are not deleted from a given file.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingMessageNoDelete,
	}
	// BreakingServiceNoDeleteRuleSpecBuilder is a rule spec builder.
	BreakingServiceNoDeleteRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "SERVICE_NO_DELETE",
		Purpose: "Checks services are not deleted from a given file.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingServiceNoDelete,
	}
	// BreakingEnumSameTypeRuleSpecBuilder is a rule spec builder.
	BreakingEnumSameTypeRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "ENUM_SAME_TYPE",
		Purpose: "Checks that enums have the same type (open vs closed).",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingEnumSameType,
	}
	// BreakingEnumValueNoDeleteRuleSpecBuilder is a rule spec builder.
	BreakingEnumValueNoDeleteRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "ENUM_VALUE_NO_DELETE",
		Purpose: "Check enum values are not deleted from a given enum.",
		Type:    check.RuleTypeBreaking,
		Handler: bufcheckserverhandle.HandleBreakingEnumValueNoDelete,
	}
	// LintCommentEnumRuleSpecBuilder is a rule spec builder.
	LintCommentEnumRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "COMMENT_ENUM",
		Purpose: "Checks that enums have non-empty comments.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintCommentEnum,
	}
	// LintCommentEnumValueRuleSpecBuilder is a rule spec builder.
	LintCommentEnumValueRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "COMMENT_ENUM_VALUE",
		Purpose: "Checks that enum values have non-empty comments.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintCommentEnumValue,
	}
	// LintCommentFieldRuleSpecBuilder is a rule spec builder.
	LintCommentFieldRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "COMMENT_FIELD",
		Purpose: "Checks that fields have non-empty comments.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintCommentField,
	}
	// LintCommentMessageRuleSpecBuilder is a rule spec builder.
	LintCommentMessageRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "COMMENT_MESSAGE",
		Purpose: "Checks that messages have non-empty comments.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintCommentMessage,
	}
	// LintCommentOneofRuleSpecBuilder is a rule spec builder.
	LintCommentOneofRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "COMMENT_ONEOF",
		Purpose: "Checks that oneofs have non-empty comments.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintCommentOneof,
	}
	// LintCommentRPCRuleSpecBuilder is a rule spec builder.
	LintCommentRPCRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "COMMENT_RPC",
		Purpose: "Checks that RPCs have non-empty comments.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintCommentRPC,
	}
	// LintCommentServiceRuleSpecBuilder is a rule spec builder.
	LintCommentServiceRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "COMMENT_SERVICE",
		Purpose: "Checks that services have non-empty comments.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintCommentService,
	}
	// LintDirectorySamePackageRuleSpecBuilder is a rule spec builder.
	LintDirectorySamePackageRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "DIRECTORY_SAME_PACKAGE",
		Purpose: "Checks that all files in a given directory are in the same package.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintDirectorySamePackage,
	}
	// LintServicePascalCaseRuleSpecBuilder is a rule spec builder.
	LintServicePascalCaseRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "SERVICE_PASCAL_CASE",
		Purpose: "Checks that services are PascalCase.",
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintServicePascalCase,
	}
	// LintServiceSuffixRuleSpecBuilder is a rule spec builder.
	LintServiceSuffixRuleSpecBuilder = &bufcheckserverutil.RuleSpecBuilder{
		ID:      "SERVICE_SUFFIX",
		Purpose: `Checks that services have a consistent suffix (configurable, default suffix is "Service").`,
		Type:    check.RuleTypeLint,
		Handler: bufcheckserverhandle.HandleLintServiceSuffix,
	}
)
