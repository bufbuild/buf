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

// Package bufcheck contains the implementations of the lint and breaking change detection rules.
//
// There is a lot of shared logic between the two, and originally they were actually combined into
// one logical entity (where some checks happened to be linters, and some checks happen to be
// breaking change detectors), so some of this is historical.
package bufcheck

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"go.uber.org/multierr"
)

// AllRuleFormatStrings is all rule format strings.
var AllRuleFormatStrings = []string{
	"text",
	"json",
}

// Rule is a rule.
type Rule interface {
	json.Marshaler

	// ID returns the ID of the Rule.
	//
	// UPPER_SNAKE_CASE.
	ID() string
	// Categories returns the categories of the Rule.
	//
	// UPPER_SNAKE_CASE.
	// Sorted.
	// May be empty.
	Categories() []string
	// Purpose returns the purpose of the Rule.
	//
	// Full sentence.
	Purpose() string

	// Deprecated returns whether or not this rule is deprecated.
	//
	// If it is, it may be replaced by 0 or more rules. These will be denoted with Replacements.
	Deprecated() bool
	// ReplacementIDs returns the IDs of the Rules that replace this Rule.
	//
	// This means that the combination of the Rules specified by ReplacementIDs replace this Rule entirely,
	// and this Rule is considered equivalent to the AND of the rules specified by ReplacementIDs.
	//
	// This will only be non-empty if Deprecated is true.
	//
	// Is it not valid for a Deprecated Rule to specify another Deprecated Rule as a replacement. We verify
	// that this does not happen for any VersionSpec in testing. TODO
	ReplacementIDs() []string
}

// PrintRules prints the rules to the writer.
//
// The empty string defaults to text.
func PrintRules(writer io.Writer, rules []Rule, formatString string, includeDeprecated bool) (retErr error) {
	if len(rules) == 0 {
		return nil
	}
	var asJSON bool
	switch s := strings.ToLower(strings.TrimSpace(formatString)); s {
	case "", "text":
		asJSON = false
	case "json":
		asJSON = true
	default:
		return fmt.Errorf("unknown format: %q", s)
	}
	if !asJSON {
		tabWriter := tabwriter.NewWriter(writer, 0, 0, 2, ' ', 0)
		defer func() {
			retErr = multierr.Append(retErr, tabWriter.Flush())
		}()
		writer = tabWriter
		if _, err := fmt.Fprintln(writer, "ID\tCATEGORIES\tPURPOSE"); err != nil {
			return err
		}
	}
	for _, rule := range rules {
		if !includeDeprecated && rule.Deprecated() {
			continue
		}
		if err := printRule(writer, rule, asJSON); err != nil {
			return err
		}
	}
	return nil
}

func printRule(writer io.Writer, rule Rule, asJSON bool) error {
	if asJSON {
		data, err := json.Marshal(rule)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintln(writer, string(data)); err != nil {
			return err
		}
		return nil
	}
	if _, err := fmt.Fprintf(writer, "%s\t%s\t%s\n", rule.ID(), strings.Join(rule.Categories(), ", "), rule.Purpose()); err != nil {
		return err
	}
	return nil
}
