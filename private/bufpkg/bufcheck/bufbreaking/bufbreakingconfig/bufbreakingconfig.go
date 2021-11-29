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
	"github.com/bufbuild/buf/private/bufpkg/bufcheck"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufbreaking/internal/bufbreakingv1"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufbreaking/internal/bufbreakingv1beta1"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/internal"
)

const (
	v1Beta1Version = "v1beta1"
	v1Version      = "v1"
)

// Config is the breaking check config.
type Config struct {
	Use    []string
	Except []string
	// IgnoreRootPaths
	Ignore []string
	// IgnoreIDOrCategoryToRootPaths
	IgnoreOnly             map[string][]string
	IgnoreUnstablePackages bool
	Version                string
}

// GetRules returns the rules.
//
// Should only be used for printing.
func (c *Config) GetRules() ([]bufcheck.Rule, error) {
	return buildRules(c)
}

// NewConfigV1Beta1 returns a new Config.
func NewConfigV1Beta1(externalConfig ExternalConfigV1Beta1) *Config {
	return &Config{
		Use:                    externalConfig.Use,
		Except:                 externalConfig.Except,
		Ignore:                 externalConfig.Ignore,
		IgnoreOnly:             externalConfig.IgnoreOnly,
		IgnoreUnstablePackages: externalConfig.IgnoreUnstablePackages,
		Version:                v1Beta1Version,
	}
}

// NewConfigV1 returns a new Config.
func NewConfigV1(externalConfig ExternalConfigV1) *Config {
	return &Config{
		Use:                    externalConfig.Use,
		Except:                 externalConfig.Except,
		Ignore:                 externalConfig.Ignore,
		IgnoreOnly:             externalConfig.IgnoreOnly,
		IgnoreUnstablePackages: externalConfig.IgnoreUnstablePackages,
		Version:                v1Version,
	}
}

// GetAllRulesV1Beta1 gets all known rules.
//
// Should only be used for printing.
func GetAllRulesV1Beta1() ([]bufcheck.Rule, error) {
	return buildRules(&Config{
		Use:     bufbreakingv1beta1.VersionSpec.AllCategories,
		Version: v1Beta1Version,
	})
}

// GetAllRulesV1 gets all known rules.
//
// Should only be used for printing.
func GetAllRulesV1() ([]bufcheck.Rule, error) {
	return buildRules(&Config{
		Use:     bufbreakingv1.VersionSpec.AllCategories,
		Version: v1Version,
	})
}

// BuildBufcheckInternalConfig takes a *Config and builds the internal.Config.
func BuildBufcheckInternalConfig(config *Config) (*internal.Config, error) {
	var versionSpec *internal.VersionSpec
	switch config.Version {
	case v1Beta1Version:
		versionSpec = bufbreakingv1beta1.VersionSpec
	case v1Version:
		versionSpec = bufbreakingv1.VersionSpec
	}
	return internal.ConfigBuilder{
		Use:                           config.Use,
		Except:                        config.Except,
		IgnoreRootPaths:               config.Ignore,
		IgnoreIDOrCategoryToRootPaths: config.IgnoreOnly,
		IgnoreUnstablePackages:        config.IgnoreUnstablePackages,
	}.NewConfig(
		versionSpec,
	)
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

func buildRules(config *Config) ([]bufcheck.Rule, error) {
	internalConfig, err := BuildBufcheckInternalConfig(config)
	if err != nil {
		return nil, err
	}
	return internalRulesToBufcheckRules(internalConfig.Rules), nil
}

func internalRulesToBufcheckRules(rules []*internal.Rule) []bufcheck.Rule {
	if rules == nil {
		return nil
	}
	s := make([]bufcheck.Rule, len(rules))
	for i, e := range rules {
		s[i] = e
	}
	return s
}
