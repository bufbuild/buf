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

package bufbreakingconfig

import (
	"encoding/json"
	"sort"

	"github.com/bufbuild/buf/private/bufpkg/bufcheck"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufbreaking/internal/bufbreakingv1"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufbreaking/internal/bufbreakingv1beta1"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/internal"
	breakingv1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/breaking/v1"
)

// Rule is a rule.
type Rule interface {
	bufcheck.Rule

	// InternalRule returns the internal Rule.
	InternalRule() *internal.Rule
}

// Config is the check config.
type Config struct {
	// Rules are the rules to run.
	//
	// Rules will be sorted by first categories, then id when Configs are
	// created from this package, i.e. created wth ConfigBuilder.NewConfig.
	Rules                  []Rule
	IgnoreIDToRootPaths    map[string]map[string]struct{}
	IgnoreRootPaths        map[string]struct{}
	IgnoreUnstablePackages bool
}

// GetRules returns the rules.
//
// Should only be used for printing.
func (c *Config) GetRules() []bufcheck.Rule {
	return rulesToBufcheckRules(c.Rules)
}

// NewConfigV1Beta1 returns a new Config.
func NewConfigV1Beta1(externalConfig ExternalConfigV1Beta1) (*Config, error) {
	internalConfig, err := internal.ConfigBuilder{
		Use:                           externalConfig.Use,
		Except:                        externalConfig.Except,
		IgnoreRootPaths:               externalConfig.Ignore,
		IgnoreIDOrCategoryToRootPaths: externalConfig.IgnoreOnly,
		IgnoreUnstablePackages:        externalConfig.IgnoreUnstablePackages,
	}.NewConfig(
		bufbreakingv1beta1.VersionSpec,
	)
	if err != nil {
		return nil, err
	}
	return internalConfigToConfig(internalConfig), nil
}

// NewConfigV1 returns a new Config.
func NewConfigV1(externalConfig ExternalConfigV1) (*Config, error) {
	internalConfig, err := internal.ConfigBuilder{
		Use:                           externalConfig.Use,
		Except:                        externalConfig.Except,
		IgnoreRootPaths:               externalConfig.Ignore,
		IgnoreIDOrCategoryToRootPaths: externalConfig.IgnoreOnly,
		IgnoreUnstablePackages:        externalConfig.IgnoreUnstablePackages,
	}.NewConfig(
		bufbreakingv1.VersionSpec,
	)
	if err != nil {
		return nil, err
	}
	return internalConfigToConfig(internalConfig), nil
}

// NewConfigV1Beta1ForProto returns a new Config for the given proto.
func NewConfigV1Beta1ForProto(protoConfig *breakingv1.Config) (*Config, error) {
	internalConfig, err := internal.ConfigBuilder{
		Use:                           protoConfig.GetUseIds(),
		Except:                        protoConfig.GetExceptIds(),
		IgnoreRootPaths:               protoConfig.GetIgnorePaths(),
		IgnoreIDOrCategoryToRootPaths: ignoreOnlyMapForProto(protoConfig.GetIgnoreIdPaths()),
		IgnoreUnstablePackages:        protoConfig.GetIgnoreUnstablePackages(),
	}.NewConfig(
		bufbreakingv1beta1.VersionSpec,
	)
	if err != nil {
		return nil, err
	}
	return internalConfigToConfig(internalConfig), nil
}

// NewConfigV1ForProto returns a new Config for the given proto.
func NewConfigV1ForProto(protoConfig *breakingv1.Config) (*Config, error) {
	internalConfig, err := internal.ConfigBuilder{
		Use:                           protoConfig.GetUseIds(),
		Except:                        protoConfig.GetExceptIds(),
		IgnoreRootPaths:               protoConfig.GetIgnorePaths(),
		IgnoreIDOrCategoryToRootPaths: ignoreOnlyMapForProto(protoConfig.GetIgnoreIdPaths()),
		IgnoreUnstablePackages:        protoConfig.GetIgnoreUnstablePackages(),
	}.NewConfig(
		bufbreakingv1.VersionSpec,
	)
	if err != nil {
		return nil, err
	}
	return internalConfigToConfig(internalConfig), nil
}

// NewBreakingConfigToBytes takes a Config and returns the []byte representation.
func NewBreakingConfigToBytes(config *Config) ([]byte, error) {
	if config == nil {
		return nil, nil
	}
	var rulesJSON []ruleJSON
	sort.Slice(config.Rules, func(i, j int) bool { return config.Rules[i].ID() < config.Rules[j].ID() })
	for _, rule := range config.Rules {
		categories := rule.Categories()
		sort.Strings(categories)
		rulesJSON = append(rulesJSON, ruleJSON{
			ID:         rule.ID(),
			Purpose:    rule.Purpose(),
			Categories: categories,
		})
	}
	var ignoreIDs []string
	for ignoreID := range config.IgnoreIDToRootPaths {
		ignoreIDs = append(ignoreIDs, ignoreID)
	}
	sort.Strings(ignoreIDs)
	ignoreIDToRootPaths := make(map[string][]string)
	for _, ignoreID := range ignoreIDs {
		var rootPaths []string
		rootPathsMap := config.IgnoreIDToRootPaths[ignoreID]
		for rootPath := range rootPathsMap {
			rootPaths = append(rootPaths, rootPath)
		}
		sort.Strings(rootPaths)
		ignoreIDToRootPaths[ignoreID] = rootPaths
	}
	var ignoreRootPaths []string
	for ignoreRootPath := range config.IgnoreRootPaths {
		ignoreRootPaths = append(ignoreRootPaths, ignoreRootPath)
	}
	sort.Strings(ignoreRootPaths)
	return json.Marshal(&breakingConfigJSON{
		Rules:                  rulesJSON,
		IgnoreIDToRootPaths:    ignoreIDToRootPaths,
		IgnoreRootPaths:        ignoreRootPaths,
		IgnoreUnstablePackages: config.IgnoreUnstablePackages,
	})
}

// GetAllRulesV1Beta1 gets all known rules.
// GetAllRulesV1Beta1 gets all known rules.
//
// Should only be used for printing.
func GetAllRulesV1Beta1() ([]bufcheck.Rule, error) {
	config, err := NewConfigV1Beta1(
		ExternalConfigV1Beta1{
			Use: bufbreakingv1beta1.VersionSpec.AllCategories,
		},
	)
	if err != nil {
		return nil, err
	}
	return rulesToBufcheckRules(config.Rules), nil
}

// GetAllRulesV1 gets all known rules.
//
// Should only be used for printing.
func GetAllRulesV1() ([]bufcheck.Rule, error) {
	config, err := NewConfigV1(
		ExternalConfigV1{
			Use: bufbreakingv1.VersionSpec.AllCategories,
		},
	)
	if err != nil {
		return nil, err
	}
	return rulesToBufcheckRules(config.Rules), nil
}

// ExternalConfigV1Beta1 is an external config.
type ExternalConfigV1Beta1 struct {
	Use    []string `json:"use,omitempty" yaml:"use,omitempty"`
	Except []string `json:"except,omitempty" yaml:"except,omitempty"`
	// IgnoreRootPaths
	Ignore []string `json:"ignore,omitempty" yaml:"ignore,omitempty"`
	// IgnoreIDOrCategoryToRootPaths
	IgnoreOnly             map[string][]string `json:"ignore_only,omitempty" yaml:"ignore_only,omitempty"`
	IgnoreUnstablePackages bool                `json:"ignore_unstable_packages,omitempty" yaml:"ignore_unstable_packages,omitempty"`
}

// ExternalConfigV1 is an external config.
type ExternalConfigV1 struct {
	Use    []string `json:"use,omitempty" yaml:"use,omitempty"`
	Except []string `json:"except,omitempty" yaml:"except,omitempty"`
	// IgnoreRootPaths
	Ignore []string `json:"ignore,omitempty" yaml:"ignore,omitempty"`
	// IgnoreIDOrCategoryToRootPaths
	IgnoreOnly             map[string][]string `json:"ignore_only,omitempty" yaml:"ignore_only,omitempty"`
	IgnoreUnstablePackages bool                `json:"ignore_unstable_packages,omitempty" yaml:"ignore_unstable_packages,omitempty"`
}

func internalRulesToRules(internalRules []*internal.Rule) []Rule {
	if internalRules == nil {
		return nil
	}
	rules := make([]Rule, len(internalRules))
	for i, internalRule := range internalRules {
		rules[i] = newRule(internalRule)
	}
	return rules
}

func rulesToBufcheckRules(rules []Rule) []bufcheck.Rule {
	if rules == nil {
		return nil
	}
	s := make([]bufcheck.Rule, len(rules))
	for i, e := range rules {
		s[i] = e
	}
	return s
}

func ignoreOnlyMapForProto(protoIDPaths []*breakingv1.IDPaths) map[string][]string {
	ignoreIDToRootPaths := make(map[string][]string)
	for _, protoIDPath := range protoIDPaths {
		ignoreIDToRootPaths[protoIDPath.GetId()] = protoIDPath.GetPaths()
	}
	return ignoreIDToRootPaths
}

type breakingConfigJSON struct {
	Rules                  []ruleJSON          `json:"rules"`
	IgnoreIDToRootPaths    map[string][]string `json:"ignore_id_to_root_paths"`
	IgnoreRootPaths        []string            `json:"ignore_root_paths"`
	IgnoreUnstablePackages bool                `json:"ignore_unstable_packages"`
}

type ruleJSON struct {
	ID         string   `json:"id"`
	Purpose    string   `json:"purpose"`
	Categories []string `json:"categories"`
}

func internalConfigToConfig(internalConfig *internal.Config) *Config {
	return &Config{
		Rules:                  internalRulesToRules(internalConfig.Rules),
		IgnoreIDToRootPaths:    internalConfig.IgnoreIDToRootPaths,
		IgnoreRootPaths:        internalConfig.IgnoreRootPaths,
		IgnoreUnstablePackages: internalConfig.IgnoreUnstablePackages,
	}
}
