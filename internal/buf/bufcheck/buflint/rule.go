// Copyright 2020-2021 Buf Technologies, Inc.
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

package buflint

import (
	"github.com/bufbuild/buf/internal/buf/bufcheck/internal"
)

type rule struct {
	*internal.Rule
}

func newRule(internalRule *internal.Rule) *rule {
	return &rule{Rule: internalRule}
}

func (c *rule) internalLint() *internal.Rule {
	return c.Rule
}

func internalRulesToRules(internalRules []*internal.Rule) []Rule {
	if internalRules == nil {
		return nil
	}
	rules := make([]Rule, len(internalRules))
	for i, internalRule := range internalRules {
		rules[i] = newRule(internalRule)
	}
	return rules
}

func rulesToInternalRules(rules []Rule) []*internal.Rule {
	if rules == nil {
		return nil
	}
	internalRules := make([]*internal.Rule, len(rules))
	for i, rule := range rules {
		internalRules[i] = rule.internalLint()
	}
	return internalRules
}
