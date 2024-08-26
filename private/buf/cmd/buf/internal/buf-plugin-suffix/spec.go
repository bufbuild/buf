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

package suffixesplugin

import (
	"context"

	"github.com/bufbuild/bufplugin-go/check"
)

const (
	lintServiceBannedSuffixesRuleID       = "SERVICE_BANNED_SUFFIXES"
	lintRPCBannedSuffixesRuleID           = "RPC_BANNED_SUFFIXES"
	lintFieldBannedSuffixesRuleID         = "FIELD_BANNED_SUFFIXES"
	lintEnumValueBannedSuffixesRuleID     = "ENUM_VALUE_BANNED_SUFFIXES"
	breakingServiceSuffixesNoChangeRuleID = "SERVICE_SUFFIXES_NO_CHANGE"
	breakingMessageSuffixesNoChangeRuleID = "MESSAGE_SUFFIXES_NO_CHANGE"
	breakingEnumSuffixesNoChangeRuleID    = "ENUM_SUFFIXES_NO_CHANGE"

	categoryOperationSuffixesID  = "OPERATION_SUFFIXES"
	categoryAttributesSuffixesID = "ATTRIBUTES_SUFFIXES"

	// deprecated rules and category IDs
	lintMessageBannedSuffixesRuleID = "MESSAGE_BANNED_SUFFIXES"
	lintEnumBannedSuffixesRuleID    = "ENUM_BANNED_SUFFIXES"
	categoryResourceSuffixesID      = "RESOURCE_SUFFIXES"
)

var (
	Spec = &check.Spec{
		Rules: []*check.RuleSpec{
			lintServiceBannedSuffixesRuleSpec,
			lintRPCBannedSuffixesRuleSpec,
			lintFieldBannedSuffixesRuleSpec,
			lintEnumValueBannedSuffixesRuleSpec,
			lintMessageBannedSuffixesRuleSpec,
			lintEnumBannedSuffixesRuleSpec,
			breakingServiceSuffixesRPCsNoChangeRuleSpec,
			breakingMessageSuffixesFieldsNoChangeRuleSpec,
			breakingEnumSuffixesEnumValuesNoChangeRuleSpec,
		},
		Categories: []*check.CategorySpec{
			{
				ID:      categoryOperationSuffixesID,
				Purpose: "Check that all operations (services and methods) have valid suffixes and those with specific suffixes have no change.",
			},
			{
				ID:         categoryResourceSuffixesID,
				Purpose:    "Check that all resources (messages and enums) have valid suffixes and those with specific suffixes have no change.",
				Deprecated: true,
				ReplacementIDs: []string{
					// Deprecated in favour for attributes category to incorporate fields and enum values checks.
					categoryAttributesSuffixesID,
				},
			},
			{
				ID:      categoryAttributesSuffixesID,
				Purpose: "Check that all fields and enum values have valid suffixes and messages and enums with specific suffixes have no chnage.",
			},
		},
	}

	lintServiceBannedSuffixesRuleSpec = &check.RuleSpec{
		ID:          lintServiceBannedSuffixesRuleID,
		Purpose:     "Ensure that there are no services with the list of configured banned suffixes.",
		Type:        check.RuleTypeLint,
		CategoryIDs: []string{categoryOperationSuffixesID},
		IsDefault:   true,
		Handler:     check.RuleHandlerFunc(handleLintServiceBannedSuffixes),
	}
	lintRPCBannedSuffixesRuleSpec = &check.RuleSpec{
		ID:          lintRPCBannedSuffixesRuleID,
		Purpose:     "Ensure that there are no RPCs with the list of configured banned suffixes.",
		Type:        check.RuleTypeLint,
		CategoryIDs: []string{categoryOperationSuffixesID},
		IsDefault:   true,
		Handler:     check.RuleHandlerFunc(handleLintRPCBannedSuffixes),
	}
	lintMessageBannedSuffixesRuleSpec = &check.RuleSpec{
		ID:          lintMessageBannedSuffixesRuleID,
		Purpose:     "Ensure that there are no messages with the list of configured banned suffixes.",
		Type:        check.RuleTypeLint,
		CategoryIDs: []string{categoryResourceSuffixesID},
		IsDefault:   false, // TODO(doria): what happens if a default rule is deprecated
		Deprecated:  true,
		ReplacementIDs: []string{
			// Mesasges encapsulate too many use-cases, we only lint fields instead.
			lintFieldBannedSuffixesRuleID,
		},
		Handler: check.RuleHandlerFunc(func(_ context.Context, _ check.ResponseWriter, _ check.Request) error { return nil }),
	}
	lintFieldBannedSuffixesRuleSpec = &check.RuleSpec{
		ID:          lintFieldBannedSuffixesRuleID,
		Purpose:     "Ensure that there are no fields with the list of configured banned suffixes.",
		Type:        check.RuleTypeLint,
		CategoryIDs: []string{categoryAttributesSuffixesID},
		IsDefault:   false,
		Handler:     check.RuleHandlerFunc(handleLintFieldBannedSuffixes),
	}
	lintEnumBannedSuffixesRuleSpec = &check.RuleSpec{
		ID:          lintEnumBannedSuffixesRuleID,
		Purpose:     "Ensure that there are no enums with the list of configured banned suffixes.",
		Type:        check.RuleTypeLint,
		CategoryIDs: []string{categoryResourceSuffixesID},
		IsDefault:   false,
		Deprecated:  true,
		ReplacementIDs: []string{
			// Enums encapsulate too many use-cases, we only lint enum values instead.
			lintEnumValueBannedSuffixesRuleID,
		},
		Handler: check.RuleHandlerFunc(func(_ context.Context, _ check.ResponseWriter, _ check.Request) error { return nil }),
	}
	lintEnumValueBannedSuffixesRuleSpec = &check.RuleSpec{
		ID:          lintEnumValueBannedSuffixesRuleID,
		Purpose:     "Ensure that there are no enum values of top-level enums with the list of configured banned suffixes.",
		Type:        check.RuleTypeLint,
		CategoryIDs: []string{categoryAttributesSuffixesID},
		IsDefault:   false,
		Handler:     check.RuleHandlerFunc(handleLintEnumValueBannedSuffixes),
	}
	breakingServiceSuffixesRPCsNoChangeRuleSpec = &check.RuleSpec{
		ID:          breakingServiceSuffixesNoChangeRuleID,
		Purpose:     "Ensure that services with configured suffixes are not deleted and do not have new RPCs or delete RPCs.",
		Type:        check.RuleTypeBreaking,
		CategoryIDs: []string{categoryOperationSuffixesID},
		IsDefault:   true,
		Handler:     check.RuleHandlerFunc(handleBreakingServiceSuffixesNoChange),
	}
	breakingMessageSuffixesFieldsNoChangeRuleSpec = &check.RuleSpec{
		ID:          breakingMessageSuffixesNoChangeRuleID,
		Purpose:     "Ensure that messages with configured suffixes are not deleted and do not have new fields or delete fields.",
		Type:        check.RuleTypeBreaking,
		CategoryIDs: []string{categoryResourceSuffixesID, categoryAttributesSuffixesID},
		IsDefault:   false,
		Handler:     check.RuleHandlerFunc(handleBreakingMessageSuffixesNoChange),
	}
	breakingEnumSuffixesEnumValuesNoChangeRuleSpec = &check.RuleSpec{
		ID:          breakingEnumSuffixesNoChangeRuleID,
		Purpose:     "Ensure that enums with configured suffixes are not deleted and do not have new enum values or delete enum values.",
		Type:        check.RuleTypeBreaking,
		CategoryIDs: []string{categoryResourceSuffixesID, categoryAttributesSuffixesID},
		IsDefault:   false,
		Handler:     check.RuleHandlerFunc(handleBreakingEnumSuffixesNoChange),
	}
)
