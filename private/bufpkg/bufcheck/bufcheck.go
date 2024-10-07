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
	"log/slog"

	"buf.build/go/bufplugin/check"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/wasm"
	"pluginrpc.com/pluginrpc"
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
	// ConfiguredRules returns all of the Configured Rules for the given RuleType.
	ConfiguredRules(ctx context.Context, ruleType check.RuleType, config bufconfig.CheckConfig, options ...ConfiguredRulesOption) ([]Rule, error)
	// AllRules returns all Rules (configured or not) for the given RuleType.
	AllRules(ctx context.Context, ruleType check.RuleType, fileVersion bufconfig.FileVersion, options ...AllRulesOption) ([]Rule, error)
	// AllCategories returns all Categories.
	AllCategories(ctx context.Context, fileVersion bufconfig.FileVersion, options ...AllCategoriesOption) ([]Category, error)
}

// Rule is an individual line or breaking Rule.
//
// It wraps check.Rule and adds the name of the plugin that implements the Rule.
type Rule interface {
	check.Rule

	// BufcheckCategories returns the Rule's Categories.
	BufcheckCategories() []Category

	// Plugin returns the name of the plugin that created this Rule.
	//
	// Names are freeform.
	//
	// Will be empty for Rules based on builtin plugins.
	PluginName() string

	isRule()
	isRuleOrCategory()
}

// Category is an individual line or breaking Category.
//
// It wraps check.Category and adds the name of the plugin that implements the Category.
type Category interface {
	check.Category

	// Plugin returns the name of the plugin that created this Category.
	//
	// Names are freeform.
	//
	// Will be empty for Categorys based on builtin plugins.
	PluginName() string

	isCategory()
	isRuleOrCategory()
}

// RuleOrCategory is a union interface with the common types in both Rule and Category.
type RuleOrCategory interface {
	ID() string
	Purpose() string
	Deprecated() bool
	ReplacementIDs() []string
	PluginName() string

	isRuleOrCategory()
}

// LintOption is an option for Lint.
type LintOption interface {
	applyToLint(*lintOptions)
}

// BreakingOption is an option for Breaking.
type BreakingOption interface {
	applyToBreaking(*breakingOptions)
}

// BreakingWithExcludeImports returns a new BreakingOption that says to exclude imports from
// breaking change detection.
//
// The default is to check imports for breaking changes.
func BreakingWithExcludeImports() BreakingOption {
	return &excludeImportsOption{}
}

// ConfiguredRulesOption is an option for ConfiguredRules.
type ConfiguredRulesOption interface {
	applyToConfiguredRules(*configuredRulesOptions)
}

// AllRulesOption is an option for AllRules.
type AllRulesOption interface {
	applyToAllRules(*allRulesOptions)
}

// AllCategoriesOption is an option for AllCategories.
type AllCategoriesOption interface {
	applyToAllCategories(*allCategoriesOptions)
}

// ClientFunctionOption is an option that applies to any Client function.
type ClientFunctionOption interface {
	LintOption
	BreakingOption
	ConfiguredRulesOption
	AllRulesOption
	AllCategoriesOption
}

// WithPluginConfigs returns a new ClientFunctionOption that says to also use the given plugins.
//
// The default is to only use the builtin Rules and Categories.
func WithPluginConfigs(pluginConfigs ...bufconfig.PluginConfig) ClientFunctionOption {
	return &pluginConfigsOption{
		pluginConfigs: pluginConfigs,
	}
}

// RunnerProvider provides pluginrpc.Runners for a given plugin config.
type RunnerProvider interface {
	NewRunner(pluginConfig bufconfig.PluginConfig) (pluginrpc.Runner, error)
}

// RunnerProviderFunc is a function that implements RunnerProvider.
type RunnerProviderFunc func(pluginConfig bufconfig.PluginConfig) (pluginrpc.Runner, error)

// NewRunner implements RunnerProvider.
//
// RunnerProvider selects the correct Runner based on the type of pluginConfig.
func (r RunnerProviderFunc) NewRunner(pluginConfig bufconfig.PluginConfig) (pluginrpc.Runner, error) {
	return r(pluginConfig)
}

// NewRunnerProvider returns a new RunnerProvider for the command.Runner and wasm.Runtime.
//
// This implementation should only be used for local applications. It is safe to
// use concurrently.
//
// The RunnerProvider selects the correct Runner based on the PluginConfigType.
// The supported types are:
//   - bufconfig.PluginConfigTypeLocal
//   - bufconfig.PluginConfigTypeLocalWasm
//
// If the PluginConfigType is not supported, an error is returned.
func NewRunnerProvider(commandRunner command.Runner, wasmRuntime wasm.Runtime) RunnerProvider {
	return newRunnerProvider(commandRunner, wasmRuntime)
}

// NewClient returns a new Client.
func NewClient(
	logger *slog.Logger,
	runnerProvider RunnerProvider,
	options ...ClientOption,
) (Client, error) {
	return newClient(logger, runnerProvider, options...)
}

// ClientOption is an option for a new Client.
type ClientOption func(*clientOptions)

// ClientWithStderr returns a new ClientOption that specifies a stderr to proxy plugin stderrs to.
//
// The default is the equivalent of /dev/null.
func ClientWithStderr(stderr io.Writer) ClientOption {
	return func(clientOptions *clientOptions) {
		clientOptions.stderr = stderr
	}
}

// PrintRules prints the rules to the Writer.
func PrintRules(writer io.Writer, rules []Rule, options ...PrintRulesOption) (retErr error) {
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
func GetDeprecatedIDToReplacementIDs[R RuleOrCategory](rulesOrCategories []R) (map[string][]string, error) {
	idToRuleOrCategory, err := slicesext.ToUniqueValuesMap(rulesOrCategories, func(ruleOrCategory R) string { return ruleOrCategory.ID() })
	if err != nil {
		return nil, err
	}
	idToReplacementIDs := make(map[string][]string)
	for _, ruleOrCategory := range rulesOrCategories {
		if ruleOrCategory.Deprecated() {
			replacementIDs := ruleOrCategory.ReplacementIDs()
			if replacementIDs == nil {
				replacementIDs = []string{}
			}
			for _, replacementID := range replacementIDs {
				if _, ok := idToRuleOrCategory[replacementID]; !ok {
					return nil, syserror.Newf("unknown rule or category ID given as a replacement ID: %q", replacementID)
				}
			}
			idToReplacementIDs[ruleOrCategory.ID()] = replacementIDs
		}
	}
	return idToReplacementIDs, nil
}
