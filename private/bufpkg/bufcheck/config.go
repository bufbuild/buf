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
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/bufplugin-go/check"
)

type config struct {
	*rulesConfig
	*optionsConfig
}

func configForLintConfig(
	lintConfig bufconfig.LintConfig,
	allRules []Rule,
) (*config, error) {
	rulesConfig, err := rulesConfigForCheckConfig(lintConfig, allRules, check.RuleTypeLint)
	if err != nil {
		return nil, err
	}
	optionsConfig, err := optionsConfigForLintConfig(lintConfig)
	if err != nil {
		return nil, err
	}
	return &config{
		rulesConfig:   rulesConfig,
		optionsConfig: optionsConfig,
	}, nil
}

func configForBreakingConfig(
	breakingConfig bufconfig.BreakingConfig,
	allRules []Rule,
	excludeImports bool,
) (*config, error) {
	rulesConfig, err := rulesConfigForCheckConfig(breakingConfig, allRules, check.RuleTypeBreaking)
	if err != nil {
		return nil, err
	}
	optionsConfig, err := optionsConfigForBreakingConfig(breakingConfig, excludeImports)
	if err != nil {
		return nil, err
	}
	return &config{
		rulesConfig:   rulesConfig,
		optionsConfig: optionsConfig,
	}, nil
}
