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
	"context"
	"fmt"
	"io"

	"github.com/bufbuild/buf/private/buf/bufcheck/bufcheckserver"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/bufplugin-go/check"
	"go.uber.org/zap"
)

// All functions that take a config ignore the FileVersion. The FileVersion should instruct
// what check.Client is passed to NewClient, ie a v1beta1, v1, or v2 default client.
//
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
	ConfiguredLintRules(ctx context.Context, config bufconfig.LintConfig) ([]check.Rule, error)
	AllLintRules(ctx context.Context) ([]check.Rule, error)

	// Breaking checks the given Images for breaking changes with the given BreakingConfig.
	//
	// The Images should have source code info for this to work properly.
	//
	// Images should *not* be filtered with regards to imports before passing to this function.
	// To exclude imports, pass BreakingWithExcludeImports.
	//
	// An error of type bufanalysis.FileAnnotationSet will be returned lint failure.
	Breaking(ctx context.Context, config bufconfig.BreakingConfig, image bufimage.Image, againstImage bufimage.Image, options ...BreakingOption) error
	ConfiguredBreakingRules(ctx context.Context, config bufconfig.BreakingConfig) ([]check.Rule, error)
	AllBreakingRules(ctx context.Context) ([]check.Rule, error)
}

type LintOption func(*lintOptions)

type BreakingOption func(*breakingOptions)

func BreakingWithExcludeImports() BreakingOption {
	return func(breakingOptions *breakingOptions) {
		breakingOptions.excludeImports = true
	}
}

// If you want to use the default v1beta1/v1/v2 Client, pass it.
// If you want to also use a plugin Client, merge the Clients with a check.NewMultiClient.
func NewClient(checkClient check.Client) Client {
	return newClient(checkClient)
}

// This will eventually parse for plugins as well and create a multi client.
func NewClientForBufYAMLFile(logger *zap.Logger, bufYAMLFile bufconfig.BufYAMLFile) (Client, error) {
	checkClient, err := NewBuiltinCheckClientForFileVersion(bufYAMLFile.FileVersion())
	if err != nil {
		return nil, err
	}
	return newClient(checkClient), nil
}

func NewBuiltinCheckClientForFileVersion(fileVersion bufconfig.FileVersion) (check.Client, error) {
	switch fileVersion {
	case bufconfig.FileVersionV1Beta1:
		return check.NewClientForSpec(bufcheckserver.V1Beta1Spec, check.ClientWithCacheRules())
	case bufconfig.FileVersionV1:
		return check.NewClientForSpec(bufcheckserver.V1Spec, check.ClientWithCacheRules())
	case bufconfig.FileVersionV2:
		return check.NewClientForSpec(bufcheckserver.V2Spec, check.ClientWithCacheRules())
	default:
		return nil, fmt.Errorf("unknown FileVersion: %v", fileVersion)
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
