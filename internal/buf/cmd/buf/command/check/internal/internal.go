// Copyright 2020 Buf Technologies, Inc.
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

// BindLSCheckersAll binds the all flag for an ls checkers command.
func BindLSCheckersAll(flagSet *pflag.FlagSet, addr *bool, flagName string) {
	flagSet.BoolVar(
		addr,
		flagName,
		false,
		"List all checkers and not just those currently configured.",
	)
}

// BindLSCheckersConfig binds the config flag for an ls checkers command.
func BindLSCheckersConfig(flagSet *pflag.FlagSet, addr *string, flagName string, allFlagName string) {
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

// BindLSCheckersFormat binds the format flag for an ls checkers command.
func BindLSCheckersFormat(flagSet *pflag.FlagSet, addr *string, flagName string) {
	flagSet.StringVar(
		addr,
		flagName,
		"text",
		fmt.Sprintf(
			"The format to print checkers as. Must be one of %s.",
			stringutil.SliceToString(bufcheck.AllCheckerFormatStrings),
		),
	)
}

// BindLSCheckersCategories binds the categories flag for an ls checkers command.
func BindLSCheckersCategories(flagSet *pflag.FlagSet, addr *[]string, flagName string) {
	flagSet.StringSliceVar(
		addr,
		flagName,
		nil,
		"Only list the checkers in these categories.",
	)
	_ = flagSet.MarkHidden(flagName)
}

// CheckLSCheckersCategories checks that value is empty as this flag is deprecated.
func CheckLSCheckersCategories(value []string, flagName string) error {
	if len(value) > 0 {
		return appcmd.NewInvalidArgumentErrorf("Flag --%s has been removed in v0.26.0 in preparation for v1.0. This flag is difficult to reconcile with the concept of configuration versions. If filtering by category is necessary, print in JSON format and filter.", flagName)
	}
	return nil
}
