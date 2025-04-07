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
	"io/fs"

	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin"
)

var (
	// NopPolicyPluginKeyProvider is a no-op PolicyPluginKeyProvider.
	NopPolicyPluginKeyProvider PolicyPluginKeyProvider = nopPolicyPluginKeyProvider{}
)

// PolicyPluginKeyProvider provides PluginKeys for a specific policy.
type PolicyPluginKeyProvider interface {
	// GetPolicyKeysForPolicyRefs gets the PolicyKeys for the given plugin Refs.
	//
	// Returned PolicyKeys will be in the same order as the input Refs.
	//
	// The input Refs are expected to be unique by FullName. The implementation
	// may error if this is not the case.
	//
	// If there is no error, the length of the PolicyKeys returned will match the length of the Refs.
	// If there is an error, no PolicyKeys will be returned.
	// If any Ref is not found, an error with fs.ErrNotExist will be returned.
	GetPolicyPluginKeysForPluginRefs(
		context.Context,
		PolicyKey,
		[]bufparse.Ref,
		bufplugin.DigestType,
	) ([]bufplugin.PluginKey, error)
}

// *** PRIVATE ***

type nopPolicyPluginKeyProvider struct{}

var _ PolicyPluginKeyProvider = nopPolicyPluginKeyProvider{}

func (nopPolicyPluginKeyProvider) GetPolicyPluginKeysForPluginRefs(
	context.Context,
	PolicyKey,
	[]bufparse.Ref,
	bufplugin.DigestType,
) ([]bufplugin.PluginKey, error) {
	return nil, fs.ErrNotExist
}
