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

	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/bufplugin-go/check"
	"go.uber.org/multierr"
)

const (
	idHeader         = "ID"
	categoriesHeader = "CATEGORIES"
	defaultHeader    = "DEFAULT"
	purposeHeader    = "PURPOSE"

	textHeader = idHeader + "\t" + categoriesHeader + "\t" + defaultHeader + "\t" + purposeHeader
)

func printRules(writer io.Writer, rules []Rule, options ...PrintRulesOption) (retErr error) {
	printRulesOptions := newPrintRulesOptions()
	for _, option := range options {
		option(printRulesOptions)
	}
	if len(rules) == 0 {
		return nil
	}
	rules = cloneAndSortRulesForPrint(rules)
	if !printRulesOptions.includeDeprecated {
		rules = slicesext.Filter(rules, func(rule Rule) bool { return !rule.Deprecated() })
	}
	if printRulesOptions.asJSON {
		return printRulesJSON(writer, rules)
	}
	return printRulesText(writer, rules)
}

// Rules already sorted in correct order.
// Rules already filtered for deprecated.
func printRulesJSON(writer io.Writer, rules []Rule) error {
	for _, rule := range rules {
		data, err := json.Marshal(newExternalRule(rule))
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintln(writer, string(data)); err != nil {
			return err
		}
	}
	return nil
}

// Rules already sorted in correct order.
// Rules already filtered for deprecated.
func printRulesText(writer io.Writer, rules []Rule) (retErr error) {
	var defaultRules []Rule
	pluginNameToRules := make(map[string][]Rule)
	var pluginNames []string

	for _, rule := range rules {
		pluginName := rule.PluginName()
		if pluginName == "" {
			defaultRules = append(defaultRules, rule)
		} else {
			if _, ok := pluginNameToRules[pluginName]; !ok {
				pluginNames = append(pluginNames, pluginName)
			}
			pluginNameToRules[pluginName] = append(pluginNameToRules[pluginName], rule)
		}
	}
	sort.Strings(pluginNames)
	longestRuleID := getLongestRuleID(rules)
	longestRuleCategories := getLongestRuleCategories(rules)

	tabWriter := tabwriter.NewWriter(writer, 0, 0, 2, ' ', 0)
	defer func() {
		retErr = multierr.Append(retErr, tabWriter.Flush())
	}()
	writer = tabWriter

	havePrintedSection := false
	if len(defaultRules) > 0 {
		if err := printRulesTextSection(writer, defaultRules, "", havePrintedSection, longestRuleID, longestRuleCategories); err != nil {
			return err
		}
		havePrintedSection = true
	}
	for _, pluginName := range pluginNames {
		if havePrintedSection {
			if _, err := fmt.Fprintln(writer); err != nil {
				return err
			}
		}
		rules := pluginNameToRules[pluginName]
		if len(rules) == 0 {
			// This should never happen.
			return syserror.Newf("no rules for plugin name %q", pluginName)
		}
		if err := printRulesTextSection(writer, rules, pluginName, havePrintedSection, longestRuleID, longestRuleCategories); err != nil {
			return err
		}
		havePrintedSection = true
	}
	return nil
}

func printRulesTextSection(writer io.Writer, rules []Rule, pluginName string, havePrintedSection bool, globallyLongestRuleID string, globallyLongestRuleCategories string) error {
	subLongestRuleID := getLongestRuleID(rules)
	subLongestRuleCategories := getLongestRuleCategories(rules)
	if pluginName != "" {
		if _, err := fmt.Fprintf(writer, "%s\n\n", pluginName); err != nil {
			return err
		}
	}
	if !havePrintedSection {
		if _, err := fmt.Fprintln(writer, textHeader); err != nil {
			return err
		}
	}
	for _, rule := range rules {
		var defaultString string
		if rule.IsDefault() {
			defaultString = "*" + strings.Repeat(" ", len(defaultHeader)-1)
		} else {
			defaultString = strings.Repeat(" ", len(defaultHeader))
		}
		id := rule.ID()
		// If our globally-longest ID is longer than any ID we have in this section, AND this current ID
		// is the longest, pad it with spaces so that all the sections have their columns aligned.
		if len(globallyLongestRuleID) > len(subLongestRuleID) && id == subLongestRuleID {
			id = id + strings.Repeat(" ", len(globallyLongestRuleID)-len(subLongestRuleID))
		}
		categories := getCategoriesString(rule.Categories())
		if len(globallyLongestRuleCategories) > len(subLongestRuleCategories) && categories == subLongestRuleCategories {
			categories = categories + strings.Repeat(" ", len(globallyLongestRuleCategories)-len(subLongestRuleCategories))
		}
		// Same logic for category strings.
		if _, err := fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", id, categories, defaultString, rule.Purpose()); err != nil {
			return err
		}
	}
	return nil
}

func getLongestRuleID(rules []Rule) string {
	return slicesext.Reduce(
		rules,
		func(accumulator string, rule Rule) string {
			id := rule.ID()
			if len(accumulator) > len(id) {
				return accumulator
			}
			return id
		},
		"",
	)
}

func getLongestRuleCategories(rules []Rule) string {
	return slicesext.Reduce(
		rules,
		func(accumulator string, rule Rule) string {
			categories := getCategoriesString(rule.Categories())
			if len(accumulator) > len(categories) {
				return accumulator
			}
			return categories
		},
		"",
	)
}

func getCategoriesString(categories []check.Category) string {
	return strings.Join(slicesext.Map(categories, check.Category.ID), ", ")
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
			sort.Slice(oneCategories, func(i int, j int) bool { return categoryIDLess(oneCategories[i].ID(), oneCategories[j].ID()) })
			twoCategories := two.Categories()
			sort.Slice(twoCategories, func(i int, j int) bool { return categoryIDLess(twoCategories[i].ID(), twoCategories[j].ID()) })
			if len(oneCategories) == 0 && len(twoCategories) > 0 {
				return false
			}
			if len(oneCategories) > 0 && len(twoCategories) == 0 {
				return true
			}
			if len(oneCategories) > 0 && len(twoCategories) > 0 {
				compare := categoryIDCompare(oneCategories[0].ID(), twoCategories[0].ID())
				if compare < 0 {
					return true
				}
				if compare > 0 {
					return false
				}
			}
			oneCategoriesString := getCategoriesString(oneCategories)
			twoCategoriesString := getCategoriesString(twoCategories)
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
		Categories:   slicesext.Map(rule.Categories(), check.Category.ID),
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
