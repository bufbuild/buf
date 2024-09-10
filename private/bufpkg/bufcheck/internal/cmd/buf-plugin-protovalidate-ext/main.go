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

package main

import (
	"buf.build/go/bufplugin/check"
	"buf.build/go/bufplugin/check/checkutil"
)

const (
	messageNotDisabledRuleID                = "MESSAGE_NOT_DISABLED"
	validateIDDashlessRuleID                = "VALIDATE_ID_DASHLESS"
	fieldValidationNotSkippedNoImportRuleID = "FIELD_VALIDATION_NOT_SKIPPED_NO_IMPORT"
	fieldValidationNotSkippedRuleID         = "FIELD_VALIDATION_NOT_SKIPPED"
	stringLenRangeNoShrinkRuleID            = "STRING_LEN_RANGE_NO_SHRINK"
)

func main() {
	check.Main(&check.Spec{
		Rules: []*check.RuleSpec{
			{
				ID:             messageNotDisabledRuleID,
				CategoryIDs:    nil,
				Default:        true,
				Deprecated:     false,
				Purpose:        `Checks that no message has (buf.validate.message).disabled set.`,
				Type:           check.RuleTypeLint,
				ReplacementIDs: nil,
				Handler:        checkutil.NewMessageRuleHandler(checkMessageNotDisabled),
			},
			{
				ID:             validateIDDashlessRuleID,
				CategoryIDs:    nil,
				Default:        true,
				Deprecated:     false,
				Purpose:        `Checks that all fields named with a certain name (default is "id") are validated as dashless UUIDs in protovalidate.`,
				Type:           check.RuleTypeLint,
				ReplacementIDs: nil,
				Handler:        checkutil.NewFieldRuleHandler(checkValidateIDDashless),
			},
			{
				ID:             fieldValidationNotSkippedNoImportRuleID,
				CategoryIDs:    nil,
				Default:        false,
				Deprecated:     false,
				Purpose:        `Checks that no field is marked as skipped in protovalidate.`,
				Type:           check.RuleTypeLint,
				ReplacementIDs: nil,
				Handler: checkutil.NewFieldRuleHandler(
					checkFieldNotSkippedNoImport,
					checkutil.WithoutImports(),
				),
			},
			{
				ID:             fieldValidationNotSkippedRuleID,
				CategoryIDs:    nil,
				Default:        false,
				Deprecated:     true,
				Purpose:        `Checks that no field is marked as skipped in protovalidate.`,
				Type:           check.RuleTypeLint,
				ReplacementIDs: []string{fieldValidationNotSkippedNoImportRuleID},
				Handler: checkutil.NewFieldRuleHandler(
					checkFieldNotSkipped,
				),
			},
			{
				ID:             stringLenRangeNoShrinkRuleID,
				CategoryIDs:    nil,
				Purpose:        `Checks that string field length ranges in protovalidate do not shrink.`,
				Type:           check.RuleTypeBreaking,
				Default:        false,
				Deprecated:     false,
				ReplacementIDs: nil,
				Handler:        breakingRuleHandlerForField(checkStringLenRangeDontShrink, true),
			},
		},
	})
}
