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

package rpcextplugin

import (
	"github.com/bufbuild/bufplugin-go/check"
	"github.com/bufbuild/bufplugin-go/check/checkutil"
)

const (
	pageRPCRequestToken  = "PAGE_REQUEST_HAS_TOKEN"
	pageRPCResponseToken = "PAGE_RESPONSE_HAS_TOKEN"
)

var (
	// PageRPCRequestTokenRuleSpec is the RuleSpec for the page request token rule.
	PageRPCRequestTokenRuleSpec = &check.RuleSpec{
		ID:             pageRPCRequestToken,
		CategoryIDs:    nil,
		Default:        true,
		Purpose:        `Checks that all pagination RPC requests has a page token set.`,
		Type:           check.RuleTypeLint,
		ReplacementIDs: nil,
		Handler:        checkutil.NewMessageRuleHandler(checkPageRequestHasToken),
	}
	// IDFieldValidationRuleSpec is the RuleSpec for the ID field validation rule.
	PageRPCResponseTokenRuleSpec = &check.RuleSpec{
		ID:             pageRPCResponseToken,
		CategoryIDs:    nil,
		Default:        true,
		Purpose:        `Checks that all pagination RPC responses has a page token set.`,
		Type:           check.RuleTypeLint,
		ReplacementIDs: nil,
		Handler:        checkutil.NewMessageRuleHandler(checkPageResponseHasToken),
	}
)
