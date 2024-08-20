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

package bufcli

import (
	"fmt"
	"io"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufcheck"
)

// AllRuleFormatStrings is all rule format strings.
var AllRuleFormatStrings = []string{
	"text",
	"json",
}

// PrintRules prints the Rules to the writer given the --format and --include-deprecated flag values.
func PrintRules(writer io.Writer, rules []bufcheck.Rule, format string, includeDeprecated bool) error {
	var printRulesOptions []bufcheck.PrintRulesOption
	switch s := strings.ToLower(strings.TrimSpace(format)); s {
	case "", "text":
	case "json":
		printRulesOptions = append(printRulesOptions, bufcheck.PrintRulesWithJSON())
	default:
		return fmt.Errorf("unknown format: %q", s)
	}
	if includeDeprecated {
		printRulesOptions = append(printRulesOptions, bufcheck.PrintRulesWithDeprecated())
	}
	return bufcheck.PrintRules(writer, rules, printRulesOptions...)
}
