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
	"buf.build/go/bufplugin/check"
	"buf.build/go/bufplugin/option"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/internal/bufcheckopt"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
)

const lintCommentIgnorePrefix = "buf:lint:ignore"

type optionsConfig struct {
	// DefaultOptions are the options that should be passed to the default check.Client.
	//
	// Do not pass these to plugin check.Clients. Use options from checkClientSpecs instead.
	// Will never be nil.
	DefaultOptions         option.Options
	AllowCommentIgnores    bool
	IgnoreUnstablePackages bool
	CommentIgnorePrefix    string
	ExcludeImports         bool
}

func optionsConfigForLintConfig(
	lintConfig bufconfig.LintConfig,
) (*optionsConfig, error) {
	return optionsConfigSpecForLintConfig(lintConfig).newOptionsConfig(
		check.RuleTypeLint,
	)
}

func optionsConfigForBreakingConfig(
	breakingConfig bufconfig.BreakingConfig,
	excludeImports bool,
) (*optionsConfig, error) {
	return optionsConfigSpecForBreakingConfig(breakingConfig, excludeImports).newOptionsConfig(
		check.RuleTypeBreaking,
	)
}

// *** BELOW THIS LINE SHOULD ONLY BE USED BY THIS FILE ***

type optionsConfigSpec struct {
	AllowCommentIgnores                  bool
	IgnoreUnstablePackages               bool
	EnumZeroValueSuffix                  string
	RPCAllowSameRequestResponse          bool
	RPCAllowGoogleProtobufEmptyRequests  bool
	RPCAllowGoogleProtobufEmptyResponses bool
	ServiceSuffix                        string
	CommentIgnorePrefix                  string
	ExcludeImports                       bool
}

func optionsConfigSpecForLintConfig(lintConfig bufconfig.LintConfig) *optionsConfigSpec {
	return &optionsConfigSpec{
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

func optionsConfigSpecForBreakingConfig(
	breakingConfig bufconfig.BreakingConfig,
	excludeImports bool,
) *optionsConfigSpec {
	return &optionsConfigSpec{
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

func (b *optionsConfigSpec) newOptionsConfig(ruleType check.RuleType) (*optionsConfig, error) {
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
	return &optionsConfig{
		DefaultOptions:         options,
		AllowCommentIgnores:    b.AllowCommentIgnores,
		IgnoreUnstablePackages: b.IgnoreUnstablePackages,
		CommentIgnorePrefix:    b.CommentIgnorePrefix,
		ExcludeImports:         b.ExcludeImports,
	}, nil
}
