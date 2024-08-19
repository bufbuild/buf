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
	"context"
	"io"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/bufplugin-go/check"
)

// Rules are returned sorted by ID, but PrintRules does our sort by category.
type Client interface {
	// Lint lints the given Image with the given LintConfig.
	//
	// The Image should have source code info for this to work properly.
	//
	// Images should *not* be filtered with regards to imports before passing to this function.
	//
	// An error of type bufanalysis.FileAnnotationSet will be returned lint failure.
	Lint(ctx context.Context, config bufconfig.LintConfig, image bufimage.Image, options ...LintOption) error
	// Breaking checks the given Images for breaking changes with the given BreakingConfig.
	//
	// The Images should have source code info for this to work properly.
	//
	// Images should *not* be filtered with regards to imports before passing to this function.
	// To exclude imports, pass BreakingWithExcludeImports.
	//
	// An error of type bufanalysis.FileAnnotationSet will be returned lint failure.
	Breaking(ctx context.Context, config bufconfig.BreakingConfig, image bufimage.Image, againstImage bufimage.Image, options ...BreakingOption) error
	ConfiguredRules(ctx context.Context, ruleType check.RuleType, config bufconfig.CheckConfig, options ...ConfiguredRulesOption) ([]check.Rule, error)
	AllRules(ctx context.Context, ruleType check.RuleType, fileVersion bufconfig.FileVersion, options ...AllRulesOption) ([]check.Rule, error)
}

type LintOption interface {
	applyToLint(*lintOptions)
}

type BreakingOption interface {
	applyToBreaking(*breakingOptions)
}

func BreakingWithExcludeImports() BreakingOption {
	return &excludeImportsOption{}
}

type ConfiguredRulesOption interface {
	applyToConfiguredRules(*configuredRulesOptions)
}

type AllRulesOption interface {
	applyToAllRules(*allRulesOptions)
}

type PluginOption interface {
	LintOption
	BreakingOption
	ConfiguredRulesOption
	AllRulesOption
}

func WithPluginConfigs(pluginConfigs ...bufconfig.PluginConfig) PluginOption {
	return &pluginConfigsOption{
		pluginConfigs: pluginConfigs,
	}
}

func NewClient(runner command.Runner, options ...ClientOption) (Client, error) {
	return newClient(runner, options...)
}

type ClientOption func(*clientOptions)

func ClientWithStderr(stderr io.Writer) ClientOption {
	return func(clientOptions *clientOptions) {
		clientOptions.stderr = stderr
	}
}

// PrintRules prints the rules to the Writer.
func PrintRules(writer io.Writer, rules []check.Rule, options ...PrintRulesOption) (retErr error) {
	return printRules(writer, rules, options...)
}

// PrintRulesOption is an option for PrintRules.
type PrintRulesOption func(*printRulesOptions)

// PrintRulesWithJSON returns a new PrintRulesOption that says to print the rules as JSON.
//
// The default is to print as text.
func PrintRulesWithJSON() PrintRulesOption {
	return func(printRulesOptions *printRulesOptions) {
		printRulesOptions.asJSON = true
	}
}

// PrintRulesWithDeprecated returns a new PrintRulesOption that resullts in deprecated rules  being printed.
func PrintRulesWithDeprecated() PrintRulesOption {
	return func(printRulesOptions *printRulesOptions) {
		printRulesOptions.includeDeprecated = true
	}
}

// GetDeprecatedIDToReplacementIDs gets a map from deprecated ID to replacement IDs.
func GetDeprecatedIDToReplacementIDs(rules []check.Rule) (map[string][]string, error) {
	idToRule, err := slicesext.ToUniqueValuesMap(rules, check.Rule.ID)
	if err != nil {
		return nil, err
	}
	idToReplacementIDs := make(map[string][]string)
	for _, rule := range rules {
		if rule.Deprecated() {
			replacementIDs := rule.ReplacementIDs()
			if replacementIDs == nil {
				replacementIDs = []string{}
			}
			for _, replacementID := range replacementIDs {
				if _, ok := idToRule[replacementID]; !ok {
					return nil, syserror.Newf("unknown rule ID given as a replacement ID: %q", replacementID)
				}
			}
			idToReplacementIDs[rule.ID()] = replacementIDs
		}
	}
	return idToReplacementIDs, nil
}
