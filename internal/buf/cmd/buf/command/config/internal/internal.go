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

package internal

import (
	"fmt"

	"github.com/bufbuild/buf/internal/buf/bufcheck"
	"github.com/bufbuild/buf/internal/pkg/app/appcmd"
	"github.com/bufbuild/buf/internal/pkg/stringutil"
	"github.com/spf13/pflag"
)

// BindLSRulesAll binds the all flag for an ls rules command.
func BindLSRulesAll(flagSet *pflag.FlagSet, addr *bool, flagName string) {
	flagSet.BoolVar(
		addr,
		flagName,
		false,
		"List all rules and not just those currently configured.",
	)
}

// BindLSRulesConfig binds the config flag for an ls rules command.
func BindLSRulesConfig(flagSet *pflag.FlagSet, addr *string, flagName string, allFlagName string) {
	flagSet.StringVar(
		addr,
		flagName,
		"",
		fmt.Sprintf(
			`The config file or data to use. If --%s is specified, this is ignored.`,
			allFlagName,
		),
	)
}

// BindLSRulesFormat binds the format flag for an ls rules command.
func BindLSRulesFormat(flagSet *pflag.FlagSet, addr *string, flagName string) {
	flagSet.StringVar(
		addr,
		flagName,
		"text",
		fmt.Sprintf(
			"The format to print rules as. Must be one of %s.",
			stringutil.SliceToString(bufcheck.AllRuleFormatStrings),
		),
	)
}

// BindLSRulesCategories binds the categories flag for an ls rules command.
func BindLSRulesCategories(flagSet *pflag.FlagSet, addr *[]string, flagName string) {
	flagSet.StringSliceVar(
		addr,
		flagName,
		nil,
		"Only list the rules in these categories.",
	)
	_ = flagSet.MarkHidden(flagName)
}

// CheckLSRulesCategories checks that value is empty as this flag is deprecated.
func CheckLSRulesCategories(value []string, flagName string) error {
	if len(value) > 0 {
		return appcmd.NewInvalidArgumentErrorf("Flag --%s has been removed in v0.26.0 in preparation for v1.0. This flag is difficult to reconcile with the concept of configuration versions. If filtering by category is necessary, print in JSON format and filter.", flagName)
	}
	return nil
}
