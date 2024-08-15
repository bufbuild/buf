// Copyright 2024 Buf Technologies, Inc.
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

package bufcheck

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/thread"
	"github.com/bufbuild/bufplugin-go/check"
)

type multiClient struct {
	checkClientSpecs []*checkClientSpec
}

func newMultiClient(checkClientSpecs []*checkClientSpec) *multiClient {
	return &multiClient{
		checkClientSpecs: checkClientSpecs,
	}
}

func (c *multiClient) Check(ctx context.Context, request check.Request) ([]check.Annotation, error) {
	allRules, chunkedRuleIDs, err := c.getRulesAndChunkedRuleIDs(ctx)
	if err != nil {
		return nil, err
	}
	// These are the specific ruleIDs that were requested.
	requestRuleIDs := request.RuleIDs()
	if len(requestRuleIDs) == 0 {
		// If we didn't have specific ruleIDs, the requested ruleIDs are all default ruleIDs.
		requestRuleIDs = slicesext.Map(slicesext.Filter(allRules, check.Rule.IsDefault), check.Rule.ID)
	}
	// This is a map of the requested ruleIDs.
	requestRuleIDMap := make(map[string]struct{})
	for _, requestRuleID := range requestRuleIDs {
		requestRuleIDMap[requestRuleID] = struct{}{}
	}

	var allAnnotations []check.Annotation
	var jobs []func(context.Context) error
	var lock sync.Mutex
	for i, delegate := range c.checkClientSpecs {
		delegate := delegate
		// This is all ruleIDs for this client.
		allDelegateRuleIDs := chunkedRuleIDs[i]
		// This is the specific requested ruleIDs for this client
		requestDelegateRuleIDs := make([]string, 0, len(allDelegateRuleIDs))
		for _, delegateRuleID := range allDelegateRuleIDs {
			// If this ruleID was requested, add it to requestDelegateRuleIDs.
			// This will result it being part of the delegate Request.
			if _, ok := requestRuleIDMap[delegateRuleID]; ok {
				requestDelegateRuleIDs = append(requestDelegateRuleIDs, delegateRuleID)
			}
		}
		delegateRequest, err := check.NewRequest(
			request.Files(),
			check.WithAgainstFiles(request.AgainstFiles()),
			// Do not use the options from Request. We parsed the options to the config or to
			// the checkClientSpec.
			check.WithOptions(delegate.Options),
			check.WithRuleIDs(requestDelegateRuleIDs...),
		)
		if err != nil {
			return nil, err
		}
		jobs = append(
			jobs,
			func(ctx context.Context) error {
				delegateResponse, err := delegate.Client.Check(ctx, delegateRequest)
				if err != nil {
					return err
				}
				lock.Lock()
				allAnnotations = append(allAnnotations, delegateResponse.Annotations()...)
				lock.Unlock()
				return nil
			},
		)
	}
	if err := thread.Parallelize(ctx, jobs); err != nil {
		return nil, err
	}
	sort.Slice(
		allAnnotations,
		func(i int, j int) bool {
			return check.CompareAnnotations(allAnnotations[i], allAnnotations[j]) < 0
		},
	)
	return allAnnotations, nil
}

func (c *multiClient) ListRules(ctx context.Context) ([]check.Rule, error) {
	rules, _, err := c.getRulesAndChunkedRuleIDs(ctx)
	if err != nil {
		return nil, err
	}
	return rules, nil
}

// Each []string within the returned [][]string is a slice of ruleIDs that corresponds
// to the client at the same index.
//
// For example, chunkedRuleIDs[1] corresponds to the ruleIDs for c.clients[1].
func (c *multiClient) getRulesAndChunkedRuleIDs(ctx context.Context) ([]check.Rule, [][]string, error) {
	var rules []check.Rule
	chunkedRuleIDs := make([][]string, len(c.checkClientSpecs))
	for i, delegate := range c.checkClientSpecs {
		delegateRules, err := delegate.Client.ListRules(ctx)
		if err != nil {
			return nil, nil, err
		}
		rules = append(rules, delegateRules...)
		chunkedRuleIDs[i] = slicesext.Map(delegateRules, check.Rule.ID)
	}
	if err := validateNoDuplicateRules(rules); err != nil {
		return nil, nil, err
	}
	sort.Slice(
		rules,
		func(i int, j int) bool {
			return check.CompareRules(rules[i], rules[j]) < 0
		},
	)
	return rules, chunkedRuleIDs, nil
}

func validateNoDuplicateRules(rules []check.Rule) error {
	return validateNoDuplicateRuleIDs(slicesext.Map(rules, check.Rule.ID))
}

func validateNoDuplicateRuleIDs(ruleIDs []string) error {
	ruleIDToCount := make(map[string]int, len(ruleIDs))
	for _, ruleID := range ruleIDs {
		ruleIDToCount[ruleID]++
	}
	var duplicateRuleIDs []string
	for ruleID, count := range ruleIDToCount {
		if count > 1 {
			duplicateRuleIDs = append(duplicateRuleIDs, ruleID)
		}
	}
	if len(duplicateRuleIDs) > 0 {
		sort.Strings(duplicateRuleIDs)
		return newDuplicateRuleError(duplicateRuleIDs)
	}
	return nil
}

type duplicateRuleError struct {
	duplicateRuleIDs []string
}

func newDuplicateRuleError(duplicateRuleIDs []string) *duplicateRuleError {
	return &duplicateRuleError{
		duplicateRuleIDs: duplicateRuleIDs,
	}
}

func (d *duplicateRuleError) Error() string {
	if d == nil {
		return ""
	}
	if len(d.duplicateRuleIDs) == 0 {
		return ""
	}
	var sb strings.Builder
	_, _ = sb.WriteString("duplicate rule IDs detected from plugins: ")
	_, _ = sb.WriteString(strings.Join(d.duplicateRuleIDs, ", "))
	return sb.String()
}