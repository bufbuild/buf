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

package bufcheckclient

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/bufbuild/bufplugin-go/check"
	"go.uber.org/multierr"
)

func printRules(writer io.Writer, rules []check.Rule, options ...PrintRulesOption) (retErr error) {
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
		if _, err := fmt.Fprintln(writer, "ID\tCATEGORIES\tPURPOSE"); err != nil {
			return err
		}
	}
	for _, rule := range rules {
		if !printRulesOptions.includeDeprecated && rule.Deprecated() {
			continue
		}
		if err := printRule(writer, rule, printRulesOptions.asJSON); err != nil {
			return err
		}
	}
	return nil
}

func printRule(writer io.Writer, rule check.Rule, asJSON bool) error {
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
	if _, err := fmt.Fprintf(writer, "%s\t%s\t%s\n", rule.ID(), strings.Join(rule.Categories(), ", "), rule.Purpose()); err != nil {
		return err
	}
	return nil
}

type externalRule struct {
	ID           string   `json:"id" yaml:"id"`
	Categories   []string `json:"categories" yaml:"categories"`
	Purpose      string   `json:"purpose" yaml:"purpose"`
	Deprecated   bool     `json:"deprecated" yaml:"deprecated"`
	Replacements []string `json:"replacements" yaml:"replacements"`
}

func newExternalRule(rule check.Rule) *externalRule {
	return &externalRule{
		ID:           rule.ID(),
		Categories:   rule.Categories(),
		Purpose:      rule.Purpose(),
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
