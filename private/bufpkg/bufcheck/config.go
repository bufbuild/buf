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
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/internal/bufcheckopt"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/bufplugin-go/check"
)

const lintCommentIgnorePrefix = "buf:lint:ignore"

// config is the check config.
//
// This should only be built via a configSpec. If we were exposing this API publicly, we would
// enforce this.
type config struct {
	*rulesConfig

	// DefaultOptions are the options that should be passed to the default check.Client.
	//
	// Do not pass these to plugin check.Clients. Use options from checkClientSpecs instead.
	// Will never be nil.
	DefaultOptions         check.Options
	AllowCommentIgnores    bool
	IgnoreUnstablePackages bool
	CommentIgnorePrefix    string
	ExcludeImports         bool
}

func configForLintConfig(lintConfig bufconfig.LintConfig, allRules []Rule) (*config, error) {
	return configSpecForLintConfig(lintConfig).newConfig(allRules, check.RuleTypeLint)
}

func configForBreakingConfig(breakingConfig bufconfig.BreakingConfig, allRules []Rule, excludeImports bool) (*config, error) {
	return configSpecForBreakingConfig(breakingConfig, excludeImports).newConfig(allRules, check.RuleTypeBreaking)
}

// *** BELOW THIS LINE SHOULD ONLY BE USED BY THIS FILE ***

// configSpec is a config spec.
type configSpec struct {
	// May contain deprecated IDs.
	Use []string
	// May contain deprecated IDs.
	Except []string

	IgnoreRootPaths []string
	// May contain deprecated IDs.
	IgnoreIDOrCategoryToRootPaths map[string][]string

	AllowCommentIgnores    bool
	IgnoreUnstablePackages bool

	EnumZeroValueSuffix                  string
	RPCAllowSameRequestResponse          bool
	RPCAllowGoogleProtobufEmptyRequests  bool
	RPCAllowGoogleProtobufEmptyResponses bool
	ServiceSuffix                        string

	CommentIgnorePrefix string
	ExcludeImports      bool
}

func configSpecForLintConfig(lintConfig bufconfig.LintConfig) *configSpec {
	return &configSpec{
		Use:                                  lintConfig.UseIDsAndCategories(),
		Except:                               lintConfig.ExceptIDsAndCategories(),
		IgnoreRootPaths:                      lintConfig.IgnorePaths(),
		IgnoreIDOrCategoryToRootPaths:        lintConfig.IgnoreIDOrCategoryToPaths(),
		AllowCommentIgnores:                  lintConfig.AllowCommentIgnores(),
		IgnoreUnstablePackages:               false,
		EnumZeroValueSuffix:                  lintConfig.EnumZeroValueSuffix(),
		RPCAllowSameRequestResponse:          lintConfig.RPCAllowSameRequestResponse(),
		RPCAllowGoogleProtobufEmptyRequests:  lintConfig.RPCAllowGoogleProtobufEmptyRequests(),
		RPCAllowGoogleProtobufEmptyResponses: lintConfig.RPCAllowGoogleProtobufEmptyResponses(),
		ServiceSuffix:                        lintConfig.ServiceSuffix(),
		CommentIgnorePrefix:                  lintCommentIgnorePrefix,
		ExcludeImports:                       false,
	}
}

func configSpecForBreakingConfig(breakingConfig bufconfig.BreakingConfig, excludeImports bool) *configSpec {
	return &configSpec{
		Use:                                  breakingConfig.UseIDsAndCategories(),
		Except:                               breakingConfig.ExceptIDsAndCategories(),
		IgnoreRootPaths:                      breakingConfig.IgnorePaths(),
		IgnoreIDOrCategoryToRootPaths:        breakingConfig.IgnoreIDOrCategoryToPaths(),
		AllowCommentIgnores:                  false,
		IgnoreUnstablePackages:               breakingConfig.IgnoreUnstablePackages(),
		EnumZeroValueSuffix:                  "",
		RPCAllowSameRequestResponse:          false,
		RPCAllowGoogleProtobufEmptyRequests:  false,
		RPCAllowGoogleProtobufEmptyResponses: false,
		ServiceSuffix:                        "",
		CommentIgnorePrefix:                  "",
		ExcludeImports:                       excludeImports,
	}
}

// newConfig returns a new Config.
func (b *configSpec) newConfig(allRules []Rule, ruleType check.RuleType) (*config, error) {
	rulesConfig, err := newRulesConfig(
		b.Use,
		b.Except,
		b.IgnoreRootPaths,
		b.IgnoreIDOrCategoryToRootPaths,
		allRules,
		ruleType,
	)
	if err != nil {
		return nil, err
	}

	optionsSpec := &bufcheckopt.OptionsSpec{
		EnumZeroValueSuffix:                  b.EnumZeroValueSuffix,
		RPCAllowSameRequestResponse:          b.RPCAllowSameRequestResponse,
		RPCAllowGoogleProtobufEmptyRequests:  b.RPCAllowGoogleProtobufEmptyRequests,
		RPCAllowGoogleProtobufEmptyResponses: b.RPCAllowGoogleProtobufEmptyResponses,
		ServiceSuffix:                        b.ServiceSuffix,
	}
	if b.CommentIgnorePrefix != "" {
		optionsSpec.CommentExcludes = []string{b.CommentIgnorePrefix}
	}
	options, err := optionsSpec.ToOptions()
	if err != nil {
		return nil, err
	}

	return &config{
		rulesConfig:            rulesConfig,
		DefaultOptions:         options,
		AllowCommentIgnores:    b.AllowCommentIgnores,
		IgnoreUnstablePackages: b.IgnoreUnstablePackages,
		CommentIgnorePrefix:    b.CommentIgnorePrefix,
		ExcludeImports:         b.ExcludeImports,
	}, nil
}
