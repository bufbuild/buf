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

	"buf.build/go/bufplugin/check"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"go.uber.org/multierr"
)

const (
	idHeader         = "ID"
	categoriesHeader = "CATEGORIES"
	defaultHeader    = "DEFAULT"
	purposeHeader    = "PURPOSE"

	textHeader = idHeader + "\t" + categoriesHeader + "\t" + defaultHeader + "\t" + purposeHeader
)

// topLevelCategoryIDToPriority is a map from builtin Category ID to the
// order in which it should be printed by the ls-.*-rules commands.
//
// This has been a crude way to do ordering from early on.
//
// priority 1 should be printed before priority 2.
var topLevelCategoryIDToPriority = map[string]int{
	"MINIMAL":   1,
	"BASIC":     2,
	"STANDARD":  3,
	"DEFAULT":   4,
	"COMMENTS":  5,
	"UNARY_RPC": 6,
	"OTHER":     7,
	"FILE":      1,
	"PACKAGE":   2,
	"WIRE_JSON": 3,
	"WIRE":      4,
}

func printRules(writer io.Writer, rules []Rule, options ...PrintRulesOption) (retErr error) {
	printRulesOptions := newPrintRulesOptions()
	for _, option := range options {
		option(printRulesOptions)
	}
	if len(rules) == 0 {
		return nil
	}
	rules = cloneAndSortRulesForPrint(rules)
	categoriesFunc := Rule.Categories
	if !printRulesOptions.includeDeprecated {
		rules = slicesext.Filter(rules, func(rule Rule) bool { return !rule.Deprecated() })
		categoriesFunc = func(rule Rule) []check.Category {
			return slicesext.Filter(rule.Categories(), func(category check.Category) bool { return !category.Deprecated() })
		}
	}
	if printRulesOptions.asJSON {
		return printRulesJSON(writer, rules, categoriesFunc)
	}
	return printRulesText(writer, rules, categoriesFunc)
}

// Rules already sorted in correct order.
// Rules already filtered for deprecated.
func printRulesJSON(writer io.Writer, rules []Rule, categoriesFunc func(Rule) []check.Category) error {
	for _, rule := range rules {
		data, err := json.Marshal(newExternalRule(rule, categoriesFunc))
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
func printRulesText(writer io.Writer, rules []Rule, categoriesFunc func(Rule) []check.Category) (retErr error) {
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
	longestRuleCategories := getLongestRuleCategories(rules, categoriesFunc)

	tabWriter := tabwriter.NewWriter(writer, 0, 0, 2, ' ', 0)
	defer func() {
		retErr = multierr.Append(retErr, tabWriter.Flush())
	}()
	writer = tabWriter

	havePrintedSection := false
	if len(defaultRules) > 0 {
		if err := printRulesTextSection(writer, defaultRules, categoriesFunc, "", havePrintedSection, longestRuleID, longestRuleCategories); err != nil {
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
		if err := printRulesTextSection(writer, rules, categoriesFunc, pluginName, havePrintedSection, longestRuleID, longestRuleCategories); err != nil {
			return err
		}
		havePrintedSection = true
	}
	return nil
}

func printRulesTextSection(
	writer io.Writer,
	rules []Rule,
	categoriesFunc func(Rule) []check.Category,
	pluginName string,
	havePrintedSection bool,
	globallyLongestRuleID string,
	globallyLongestRuleCategories string,
) error {
	subLongestRuleID := getLongestRuleID(rules)
	subLongestRuleCategories := getLongestRuleCategories(rules, categoriesFunc)
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
		if rule.Default() {
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
		categories := getCategoriesString(categoriesFunc(rule))
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

func getLongestRuleCategories(
	rules []Rule,
	categoriesFunc func(Rule) []check.Category,
) string {
	return slicesext.Reduce(
		rules,
		func(accumulator string, rule Rule) string {
			categories := getCategoriesString(categoriesFunc(rule))
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
			// Sort builtin rules before plugin rules, then plugin rules by plugin name.
			onePluginName := one.PluginName()
			twoPluginName := two.PluginName()
			if onePluginName == "" && twoPluginName != "" {
				return true
			}
			if onePluginName != "" && twoPluginName == "" {
				return false
			}
			if compare := strings.Compare(onePluginName, twoPluginName); compare != 0 {
				return compare < 0
			}
			// Sort default rules before non-default.
			if one.Default() && !two.Default() {
				return true
			}
			if !one.Default() && two.Default() {
				return false
			}
			oneCategories := one.Categories()
			sort.Slice(oneCategories, func(i int, j int) bool { return printCategoryIDLess(oneCategories[i].ID(), oneCategories[j].ID()) })
			twoCategories := two.Categories()
			sort.Slice(twoCategories, func(i int, j int) bool { return printCategoryIDLess(twoCategories[i].ID(), twoCategories[j].ID()) })
			if len(oneCategories) == 0 && len(twoCategories) > 0 {
				return false
			}
			if len(oneCategories) > 0 && len(twoCategories) == 0 {
				return true
			}
			if len(oneCategories) > 0 && len(twoCategories) > 0 {
				compare := printCategoryIDCompare(oneCategories[0].ID(), twoCategories[0].ID())
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

func printCategoryIDLess(one string, two string) bool {
	return printCategoryIDCompare(one, two) < 0
}

func printCategoryIDCompare(one string, two string) int {
	onePriority, oneIsTopLevel := topLevelCategoryIDToPriority[one]
	twoPriority, twoIsTopLevel := topLevelCategoryIDToPriority[two]
	if oneIsTopLevel && !twoIsTopLevel {
		return -1
	}
	if !oneIsTopLevel && twoIsTopLevel {
		return 1
	}
	if oneIsTopLevel && twoIsTopLevel {
		if onePriority < twoPriority {
			return -1
		}
		if onePriority > twoPriority {
			return 1
		}
	}
	if one < two {
		return -1
	}
	if one > two {
		return 1
	}
	return 0
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

func newExternalRule(
	rule Rule,
	categoriesFunc func(Rule) []check.Category,
) *externalRule {
	return &externalRule{
		ID:           rule.ID(),
		Categories:   slicesext.Map(categoriesFunc(rule), check.Category.ID),
		Default:      rule.Default(),
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
