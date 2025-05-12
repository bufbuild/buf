// Copyright 2020-2025 Buf Technologies, Inc.
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

package bufconfig

import (
	"errors"
	"slices"

	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

// PluginConfig is a configuration for a policy.
type PolicyConfig interface {
	// Name returns the policy name. This is never empty.
	Name() string
	// Paths are specific to the Module. Users cannot ignore paths outside of their modules for check
	// configs, which includes any imports from outside of the module.
	// Paths are relative to roots.
	// Paths are sorted.
	IgnorePaths() []string
	// Paths are specific to the Module. Users cannot ignore paths outside of their modules for
	// check configs, which includes any imports from outside of the module.
	// Paths are relative to roots.
	// Paths are sorted.
	IgnoreIDOrCategoryToPaths() map[string][]string

	isPolicyConfig()
}

// NewPolicyConfig returns a new PolicyConfig.
func NewPolicyConfig(
	name string,
	ignore []string,
	ignoreOnly map[string][]string,
) (PolicyConfig, error) {
	return newPolicyConfig(name, ignore, ignoreOnly)
}

// *** PRIVATE ***

type policyConfig struct {
	name       string
	ignore     []string
	ignoreOnly map[string][]string
}

func newPolicyConfigForExternalV2(
	externalConfig externalBufYAMLFilePolicyV2,
) (*policyConfig, error) {
	return newPolicyConfig(
		externalConfig.Policy,
		externalConfig.Ignore,
		externalConfig.IgnoreOnly,
	)
}

func newPolicyConfig(
	name string,
	ignore []string,
	ignoreOnly map[string][]string,
) (*policyConfig, error) {
	if name == "" {
		return nil, errors.New("must specify a name to the policy")
	}
	ignore = xslices.ToUniqueSorted(ignore)
	ignore, err := normalizeAndCheckPaths(ignore, "ignore")
	if err != nil {
		return nil, err
	}
	newIgnoreOnly := make(map[string][]string, len(ignoreOnly))
	for k, v := range ignoreOnly {
		v = xslices.ToUniqueSorted(v)
		v, err := normalizeAndCheckPaths(v, "ignore_only path")
		if err != nil {
			return nil, err
		}
		newIgnoreOnly[k] = v
	}
	ignoreOnly = newIgnoreOnly
	return &policyConfig{
		name:       name,
		ignore:     ignore,
		ignoreOnly: ignoreOnly,
	}, nil
}

func (p *policyConfig) Name() string {
	return p.name
}

func (p *policyConfig) IgnorePaths() []string {
	return slices.Clone(p.ignore)
}

func (p *policyConfig) IgnoreIDOrCategoryToPaths() map[string][]string {
	return copyStringToStringSliceMap(p.ignoreOnly)
}

func (p *policyConfig) isPolicyConfig() {}

func newExternalV2ForPolicyConfig(
	config PolicyConfig,
) (externalBufYAMLFilePolicyV2, error) {
	pluginConfig, ok := config.(*policyConfig)
	if !ok {
		return externalBufYAMLFilePolicyV2{}, syserror.Newf("unknown implementation of PolicyConfig: %T", pluginConfig)
	}
	return externalBufYAMLFilePolicyV2{
		Policy:     pluginConfig.Name(),
		Ignore:     slices.Clone(pluginConfig.IgnorePaths()),
		IgnoreOnly: copyStringToStringSliceMap(pluginConfig.IgnoreIDOrCategoryToPaths()),
	}, nil
}
