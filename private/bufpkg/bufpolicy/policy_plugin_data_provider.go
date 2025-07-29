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

package bufpolicy

import (
	"github.com/bufbuild/buf/private/bufpkg/bufplugin"
)

var (
	// NopPolicyPluginDataProvider is a no-op PolicyDataProvider.
	NopPolicyPluginDataProvider PolicyPluginDataProvider = nopPolicyPluginDataProvider{}
)

// PolicyPluginDataProvider provides PluginData for a specific Policy.
type PolicyPluginDataProvider interface {
	// GetPluginDataProviderForPolicy returns the PluginDataProvider for the given policy name.
	//
	// The PluginDataProvider returned will be used to resolve the PluginData for the given policy.
	// If the Policy is not found a bufplugin.NopPluginDataProvider will be returned.
	GetPluginDataProviderForPolicy(policyName string) bufplugin.PluginDataProvider
}

// NewStaticPolicyPluginDataProvider returns a new PolicyPluginDataProvider for a static set of
// PolicyNames to PluginData.
//
// Each set of PluginDatas must be unique by FullName. If there are duplicates,
// an error will be returned.
//
// When resolving Refs, the Ref will be matched to the PolicyPluginData by FullName.
// If the Ref is not found in the set of provided keys, an fs.ErrNotExist will be returned.
func NewStaticPolicyPluginDataProvider(policyNameToPluginDataProvider map[string]bufplugin.PluginDataProvider) (PolicyPluginDataProvider, error) {
	return newStaticPolicyPluginDataProvider(policyNameToPluginDataProvider)
}

// *** PRIVATE ***

type nopPolicyPluginDataProvider struct{}

var _ PolicyPluginDataProvider = nopPolicyPluginDataProvider{}

func (nopPolicyPluginDataProvider) GetPluginDataProviderForPolicy(policyName string) bufplugin.PluginDataProvider {
	return bufplugin.NopPluginDataProvider
}

type staticPolicyPluginDataProvider struct {
	policyNameToPluginDataProvider map[string]bufplugin.PluginDataProvider
}

func newStaticPolicyPluginDataProvider(policyNameToPluginDataProvider map[string]bufplugin.PluginDataProvider) (*staticPolicyPluginDataProvider, error) {
	return &staticPolicyPluginDataProvider{
		policyNameToPluginDataProvider: policyNameToPluginDataProvider,
	}, nil
}

func (s staticPolicyPluginDataProvider) GetPluginDataProviderForPolicy(policyName string) bufplugin.PluginDataProvider {
	if pluginDataProvider, ok := s.policyNameToPluginDataProvider[policyName]; ok {
		return pluginDataProvider
	}
	return bufplugin.NopPluginDataProvider
}
