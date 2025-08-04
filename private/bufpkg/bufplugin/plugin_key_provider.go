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

package bufplugin

import (
	"context"
	"fmt"
	"io/fs"

	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
)

var (
	// NopPluginKeyProvider is a no-op PluginKeyProvider.
	NopPluginKeyProvider PluginKeyProvider = nopPluginKeyProvider{}
)

// PluginKeyProvider provides PluginKeys for bufparse.Refs.
type PluginKeyProvider interface {
	// GetPluginKeysForPluginRefs gets the PluginKeys for the given PluginRefs.
	//
	// Returned PluginKeys will be in the same order as the input PluginRefs.
	//
	// The input PluginRefs are expected to be unique by FullName. The implementation
	// may error if this is not the case.
	//
	// If there is no error, the length of the PluginKeys returned will match the length of the Refs.
	// If there is an error, no PluginKeys will be returned.
	// If any PluginRef is not found, an error with fs.ErrNotExist will be returned.
	GetPluginKeysForPluginRefs(context.Context, []bufparse.Ref, DigestType) ([]PluginKey, error)
}

// NewStaticPluginKeyProvider returns a new PluginKeyProvider for a static set of PluginKeys.
//
// The set of PluginKeys must be unique by FullName. If there are duplicates,
// an error will be returned.
//
// When resolving Refs, the Ref will be matched to the PluginKey by FullName.
// If the Ref is not found in the set of provided keys, an fs.ErrNotExist will be returned.
func NewStaticPluginKeyProvider(pluginKeys []PluginKey) (PluginKeyProvider, error) {
	return newStaticPluginKeyProvider(pluginKeys)
}

// *** PRIVATE ***

type nopPluginKeyProvider struct{}

func (nopPluginKeyProvider) GetPluginKeysForPluginRefs(
	context.Context,
	[]bufparse.Ref,
	DigestType,
) ([]PluginKey, error) {
	return nil, fs.ErrNotExist
}

type staticPluginKeyProvider struct {
	pluginKeysByFullName map[string]PluginKey
}

func newStaticPluginKeyProvider(pluginKeys []PluginKey) (*staticPluginKeyProvider, error) {
	var pluginKeysByFullName map[string]PluginKey
	if len(pluginKeys) > 0 {
		var err error
		pluginKeysByFullName, err = xslices.ToUniqueValuesMap(pluginKeys, func(pluginKey PluginKey) string {
			return pluginKey.FullName().String()
		})
		if err != nil {
			return nil, err
		}
	}
	return &staticPluginKeyProvider{
		pluginKeysByFullName: pluginKeysByFullName,
	}, nil
}

func (s staticPluginKeyProvider) GetPluginKeysForPluginRefs(
	_ context.Context,
	refs []bufparse.Ref,
	digestType DigestType,
) ([]PluginKey, error) {
	pluginKeys := make([]PluginKey, len(refs))
	for i, ref := range refs {
		// Only the FullName is used to match the PluginKey. The Ref is not
		// validated to match the PluginKey as there is not enough information
		// to do so.
		pluginKey, ok := s.pluginKeysByFullName[ref.FullName().String()]
		if !ok {
			return nil, fs.ErrNotExist
		}
		digest, err := pluginKey.Digest()
		if err != nil {
			return nil, err
		}
		if digest.Type() != digestType {
			return nil, fmt.Errorf("expected DigestType %v, got %v", digestType, digest.Type())
		}
		pluginKeys[i] = pluginKey
	}
	return pluginKeys, nil
}
