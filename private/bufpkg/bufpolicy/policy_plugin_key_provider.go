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

// *** PRIVATE ***

type nopPolicyPluginKeyProvider struct{}

var _ PolicyPluginKeyProvider = nopPolicyPluginKeyProvider{}

func (nopPolicyPluginKeyProvider) GetPluginKeyProviderForPolicy(policyName string) bufplugin.PluginKeyProvider {
	return bufplugin.NopPluginKeyProvider
}
