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
	"context"
	"fmt"
	"io/fs"

	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin"
)

var (
	// NopPolicyPluginKeyProvider is a no-op PolicyPluginKeyProvider.
	NopPolicyPluginKeyProvider PolicyPluginKeyProvider = nopPolicyPluginKeyProvider{}
)

// PolicyPluginKeyProvider provides PluginKeys for a specific Policy.
type PolicyPluginKeyProvider interface {
	// GetPluginKeyProviderForPolicy returns the PluginKeyProvider for the given policy name.
	//
	// The PluginKeyProvider returned will be used to resolve the PluginKeys for the given policy.
	// If the Policy is not found a bufplugin.NopPluginKeyProvider will be returned.
	GetPluginKeyProviderForPolicy(policyName string) bufplugin.PluginKeyProvider
}

// NewStaticPolicyPluginKeyProvider returns a new PolicyPluginKeyProvider for a static set of
// PolicyNames to PluginKeys.
//
// Each set of PluginKeys must be unique by FullName. If there are duplicates,
// an error will be returned.
//
// When resolving Refs, the Ref will be matched to the PolicyPluginKey by FullName.
// If the Ref is not found in the set of provided keys, an fs.ErrNotExist will be returned.
func NewStaticPolicyPluginKeyProvider(policyNameToPluginKeys map[string][]bufplugin.PluginKey) (PolicyPluginKeyProvider, error) {
	return newStaticPolicyPluginKeyProvider(policyNameToPluginKeys)
}

// *** PRIVATE ***

type nopPolicyPluginKeyProvider struct{}

var _ PolicyPluginKeyProvider = nopPolicyPluginKeyProvider{}

func (nopPolicyPluginKeyProvider) GetPluginKeyProviderForPolicy(policyName string) bufplugin.PluginKeyProvider {
	return bufplugin.NopPluginKeyProvider
}

type staticPolicyPluginKeyProvider struct {
	policyNameToPluginKeyProvider map[string]bufplugin.PluginKeyProvider
}

func newStaticPolicyPluginKeyProvider(policyNameToPluginKeys map[string][]bufplugin.PluginKey) (*staticPolicyPluginKeyProvider, error) {
	policyNameToPluginKeyProvider := make(map[string]bufplugin.PluginKeyProvider)
	for policyName, pluginKeys := range policyNameToPluginKeys {
		pluginKeyProvider, err := bufplugin.NewStaticPluginKeyProvider(pluginKeys)
		if err != nil {
			return nil, fmt.Errorf("failed to create PluginKeyProvider for policy %q: %w", policyName, err)
		}
		policyNameToPluginKeyProvider[policyName] = pluginKeyProvider
	}
	return &staticPolicyPluginKeyProvider{
		policyNameToPluginKeyProvider: policyNameToPluginKeyProvider,
	}, nil
}

func (s staticPolicyPluginKeyProvider) GetPluginKeyProviderForPolicy(policyName string) bufplugin.PluginKeyProvider {
	if pluginKeyProvider, ok := s.policyNameToPluginKeyProvider[policyName]; ok {
		return pluginKeyProvider
	}
	// Check if the policyName is a valid ref. If so, also check by full name.
	// Remote policies are cached by full name in the buf.lock file.
	if ref, err := bufparse.ParseRef(policyName); err == nil && ref.Ref() != "" {
		if pluginKeyProvider, ok := s.policyNameToPluginKeyProvider[ref.FullName().String()]; ok {
			return pluginKeyProvider
		}
	}
	return newNopPluginKeyProviderForPolicy(policyName)
}

type nopPluginKeyProviderForPolicy struct {
	policyName string
}

func newNopPluginKeyProviderForPolicy(policyName string) bufplugin.PluginKeyProvider {
	return &nopPluginKeyProviderForPolicy{
		policyName: policyName,
	}
}

func (p *nopPluginKeyProviderForPolicy) GetPluginKeysForPluginRefs(
	context.Context,
	[]bufparse.Ref,
	bufplugin.DigestType,
) ([]bufplugin.PluginKey, error) {
	return nil, fmt.Errorf("no plugins configured for policy %q: %w", p.policyName, fs.ErrNotExist)
}
