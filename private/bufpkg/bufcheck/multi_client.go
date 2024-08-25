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

package bufcheck

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/thread"
	"github.com/bufbuild/bufplugin-go/check"
	"go.uber.org/zap"
)

// TODO: Test plugins with duplicate rules and categories across builtin + plugins, and just plugins

type multiClient struct {
	logger           *zap.Logger
	checkClientSpecs []*checkClientSpec
}

func newMultiClient(logger *zap.Logger, checkClientSpecs []*checkClientSpec) *multiClient {
	return &multiClient{
		logger:           logger,
		checkClientSpecs: checkClientSpecs,
	}
}

func (c *multiClient) Check(ctx context.Context, request check.Request) ([]*annotation, error) {
	allRules, chunkedRuleIDs, _, _, err := c.getRulesCategoriesAndChunkedIDs(ctx)
	if err != nil {
		return nil, err
	}
	// These are the specific ruleIDs that were requested.
	requestRuleIDs := request.RuleIDs()
	if len(requestRuleIDs) == 0 {
		// If we didn't have specific ruleIDs, the requested ruleIDs are all default ruleIDs.
		requestRuleIDs = slicesext.Map(slicesext.Filter(allRules, Rule.IsDefault), Rule.ID)
	}
	// This is a map of the requested ruleIDs.
	requestRuleIDMap := make(map[string]struct{})
	for _, requestRuleID := range requestRuleIDs {
		requestRuleIDMap[requestRuleID] = struct{}{}
	}

	var allAnnotations []*annotation
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
				annotations := slicesext.Map(
					delegateResponse.Annotations(),
					func(checkAnnotation check.Annotation) *annotation {
						return newAnnotation(checkAnnotation, delegate.PluginName)
					},
				)
				allAnnotations = append(allAnnotations, annotations...)
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

func (c *multiClient) ListRulesAndCategories(ctx context.Context) ([]Rule, []Category, error) {
	rules, _, categories, _, err := c.getRulesCategoriesAndChunkedIDs(ctx)
	if err != nil {
		return nil, nil, err
	}
	return rules, categories, nil
}

// Each []string within the returned [][]string is a slice of ruleIDs that corresponds
// to the client at the same index.
//
// For example, chunkedRuleIDs[1] corresponds to the ruleIDs for c.clients[1].
//
// This function does duplicate checking across all the Rules and Categories
// across the plugins.
func (c *multiClient) getRulesCategoriesAndChunkedIDs(ctx context.Context) (
	retRules []Rule,
	retChunkedRuleIDs [][]string,
	retCategories []Category,
	retChunkedCategoryIDs [][]string,
	retErr error,
) {
	var rules []Rule
	chunkedRuleIDs := make([][]string, len(c.checkClientSpecs))
	for i, delegate := range c.checkClientSpecs {
		delegateCheckRules, err := delegate.Client.ListRules(ctx)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		delegateRules := slicesext.Map(
			delegateCheckRules,
			func(checkRule check.Rule) Rule { return newRule(checkRule, delegate.PluginName) },
		)
		rules = append(rules, delegateRules...)
		// Already sorted.
		chunkedRuleIDs[i] = slicesext.Map(delegateRules, Rule.ID)
	}

	var categories []Category
	chunkedCategoryIDs := make([][]string, len(c.checkClientSpecs))
	for i, delegate := range c.checkClientSpecs {
		delegateCheckCategories, err := delegate.Client.ListCategories(ctx)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		delegateCategories := slicesext.Map(
			delegateCheckCategories,
			func(checkCategory check.Category) Category { return newCategory(checkCategory, delegate.PluginName) },
		)
		categories = append(categories, delegateCategories...)
		// Already sorted.
		chunkedCategoryIDs[i] = slicesext.Map(delegateCategories, Category.ID)
	}

	if err := validateNoDuplicateRulesOrCategories(rules, categories); err != nil {
		return nil, nil, nil, nil, err
	}

	sort.Slice(
		rules,
		func(i int, j int) bool {
			return check.CompareRules(rules[i], rules[j]) < 0
		},
	)
	sort.Slice(
		categories,
		func(i int, j int) bool {
			return check.CompareCategories(categories[i], categories[j]) < 0
		},
	)

	return rules, chunkedRuleIDs, categories, chunkedCategoryIDs, nil
}

func validateNoDuplicateRulesOrCategories(rules []Rule, categories []Category) error {
	idToRuleOrCategories := make(map[string][]ruleOrCategory)
	for _, rule := range rules {
		idToRuleOrCategories[rule.ID()] = append(
			idToRuleOrCategories[rule.ID()],
			rule,
		)
	}
	for _, category := range categories {
		idToRuleOrCategories[category.ID()] = append(
			idToRuleOrCategories[category.ID()],
			category,
		)
	}
	for id, ruleOrCategories := range idToRuleOrCategories {
		if len(ruleOrCategories) <= 1 {
			delete(idToRuleOrCategories, id)
		}
	}
	if len(idToRuleOrCategories) > 0 {
		return newDuplicateRuleOrCategoryError(idToRuleOrCategories)
	}
	return nil
}

type duplicateRuleOrCategoryError struct {
	duplicateIDToRuleOrCategories map[string][]ruleOrCategory
}

func newDuplicateRuleOrCategoryError(
	duplicateIDToRuleOrCategories map[string][]ruleOrCategory,
) *duplicateRuleOrCategoryError {
	return &duplicateRuleOrCategoryError{
		duplicateIDToRuleOrCategories: duplicateIDToRuleOrCategories,
	}
}

func (d *duplicateRuleOrCategoryError) Error() string {
	if d == nil {
		return ""
	}
	if len(d.duplicateIDToRuleOrCategories) == 0 {
		return ""
	}

	var sb strings.Builder
	_, _ = sb.WriteString("duplicate rule IDs detected from plugins:\n")
	for _, duplicateID := range slicesext.MapKeysToSortedSlice(d.duplicateIDToRuleOrCategories) {
		// Example of this loop:
		//
		// RULE_FOO: builtin, buf-plugin-foo, buf-plugin-bar
		// CATEGORY_BAR: buf-plugin-foo, buf-plugin-baz
		_, _ = sb.WriteString(duplicateID)
		_, _ = sb.WriteString(": ")
		ruleOrCategories := d.duplicateIDToRuleOrCategories[duplicateID]
		sort.Slice(
			ruleOrCategories,
			func(i int, j int) bool {
				return ruleOrCategories[i].ID() < ruleOrCategories[j].ID()
			},
		)
		_, _ = sb.WriteString(
			strings.Join(
				slicesext.Map(
					ruleOrCategories,
					func(ruleOrCategory ruleOrCategory) string {
						if pluginName := ruleOrCategory.PluginName(); pluginName != "" {
							return pluginName
						}
						return "builtin"
					},
				),
				", ",
			),
		)
	}
	return sb.String()
}
