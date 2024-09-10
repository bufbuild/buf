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
	"buf.build/go/bufplugin/check"
	"github.com/bufbuild/buf/private/pkg/slicesext"
)

var _ check.Rule = &rule{}
var _ Rule = &rule{}
var _ RuleOrCategory = &rule{}

type rule struct {
	check.Rule

	pluginName string
}

func newRule(checkRule check.Rule, pluginName string) *rule {
	return &rule{
		Rule:       checkRule,
		pluginName: pluginName,
	}
}

func (r *rule) BufcheckCategories() []Category {
	return slicesext.Map(
		r.Rule.Categories(),
		func(checkCategory check.Category) Category {
			return newCategory(checkCategory, r.pluginName)
		},
	)
}

func (r *rule) PluginName() string {
	return r.pluginName
}

func (*rule) isRule()           {}
func (*rule) isRuleOrCategory() {}

// Returns Rules in same order as in allRules.
func rulesForType[R check.Rule](allRules []R, ruleType check.RuleType) []R {
	return slicesext.Filter(allRules, func(rule R) bool { return rule.Type() == ruleType })
}

// Returns Rules in same order as in allRules.
func rulesForRuleIDs[R check.Rule](allRules []R, ruleIDs []string) []R {
	rules := make([]R, 0, len(allRules))
	ruleIDMap := slicesext.ToStructMap(ruleIDs)
	for _, rule := range allRules {
		if _, ok := ruleIDMap[rule.ID()]; ok {
			rules = append(rules, rule)
		}
	}
	return rules
}
