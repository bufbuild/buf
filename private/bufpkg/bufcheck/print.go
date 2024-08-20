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
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/bufbuild/bufplugin-go/check"
	"go.uber.org/multierr"
)

func printRules(writer io.Writer, rules []Rule, options ...PrintRulesOption) (retErr error) {
	printRulesOptions := newPrintRulesOptions()
	for _, option := range options {
		option(printRulesOptions)
	}
	if len(rules) == 0 {
		return nil
	}
	if !printRulesOptions.asJSON {
		tabWriter := tabwriter.NewWriter(writer, 0, 0, 2, ' ', 0)
		defer func() {
			retErr = multierr.Append(retErr, tabWriter.Flush())
		}()
		writer = tabWriter
		if _, err := fmt.Fprintln(writer, "ID\tCATEGORIES\tDEFAULT\tPLUGIN\tPURPOSE"); err != nil {
			return err
		}
	}
	for _, rule := range cloneAndSortRulesForPrint(rules) {
		if !printRulesOptions.includeDeprecated && rule.Deprecated() {
			continue
		}
		if err := printRule(writer, rule, printRulesOptions.asJSON); err != nil {
			return err
		}
	}
	return nil
}

func printRule(writer io.Writer, rule Rule, asJSON bool) error {
	if asJSON {
		data, err := json.Marshal(newExternalRule(rule))
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintln(writer, string(data)); err != nil {
			return err
		}
		return nil
	}
	var defaultString string
	if rule.IsDefault() {
		defaultString = "*"
	}
	if _, err := fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n", rule.ID(), strings.Join(rule.Categories(), ", "), defaultString, rule.PluginName(), rule.Purpose()); err != nil {
		return err
	}
	return nil
}

// cloneAndSortRulesForPrint sorts the rules just for printing.
//
// This has different sorting than the result of check.CompareRules.
func cloneAndSortRulesForPrint(rules []Rule) []Rule {
	rules = slices.Clone(rules)
	// Apply the default sorting to start.
	sort.Slice(rules, func(i int, j int) bool { return check.CompareRules(rules[i], rules[j]) < 0 })
	// Then, apply our own sorting.
	sort.SliceStable(
		rules,
		func(i int, j int) bool {
			// categories are sorted at this point
			// so we know the first category is a top-level category if present
			one := rules[i]
			two := rules[j]
			// Sort default rules before non-default.
			if one.IsDefault() && !two.IsDefault() {
				return true
			}
			if !one.IsDefault() && two.IsDefault() {
				return false
			}
			// Next, sort builtin rules before plugin rules, then plugin rules by plugin name.
			onePluginName := one.PluginName()
			twoPluginName := two.PluginName()
			if onePluginName == "" && twoPluginName != "" {
				return true
			}
			if onePluginName != "" && twoPluginName == "" {
				return false
			}
			if compare := strings.Compare(onePluginName, twoPluginName); compare != 0 {
				if compare < 0 {
					return true
				}
				return false
			}
			oneCategories := one.Categories()
			sort.Slice(oneCategories, func(i int, j int) bool { return categoryLess(oneCategories[i], oneCategories[j]) })
			twoCategories := two.Categories()
			sort.Slice(twoCategories, func(i int, j int) bool { return categoryLess(twoCategories[i], twoCategories[j]) })
			if len(oneCategories) == 0 && len(twoCategories) > 0 {
				return false
			}
			if len(oneCategories) > 0 && len(twoCategories) == 0 {
				return true
			}
			if len(oneCategories) > 0 && len(twoCategories) > 0 {
				compare := categoryCompare(oneCategories[0], twoCategories[0])
				if compare < 0 {
					return true
				}
				if compare > 0 {
					return false
				}
			}
			oneCategoriesString := strings.Join(oneCategories, ",")
			twoCategoriesString := strings.Join(twoCategories, ",")
			if oneCategoriesString < twoCategoriesString {
				return true
			}
			if oneCategoriesString > twoCategoriesString {
				return false
			}
			return one.ID() < two.ID()
		},
	)
	return rules
}

type externalRule struct {
	ID           string   `json:"id" yaml:"id"`
	Categories   []string `json:"categories" yaml:"categories"`
	Default      bool     `json:"default" yaml:"default"`
	Purpose      string   `json:"purpose" yaml:"purpose"`
	Plugin       string   `json:"plugin" yaml:"plugin"`
	Deprecated   bool     `json:"deprecated" yaml:"deprecated"`
	Replacements []string `json:"replacements" yaml:"replacements"`
}

func newExternalRule(rule Rule) *externalRule {
	return &externalRule{
		ID:           rule.ID(),
		Categories:   rule.Categories(),
		Default:      rule.IsDefault(),
		Purpose:      rule.Purpose(),
		Plugin:       rule.PluginName(),
		Deprecated:   rule.Deprecated(),
		Replacements: rule.ReplacementIDs(),
	}
}

type printRulesOptions struct {
	asJSON            bool
	includeDeprecated bool
}

func newPrintRulesOptions() *printRulesOptions {
	return &printRulesOptions{}
}
