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

package bufplugin

import (
	"context"
	"io/fs"
)

var (
	// NopPluginKeyProvider is a no-op PluginKeyProvider.
	NopPluginKeyProvider PluginKeyProvider = nopPluginKeyProvider{}
)

// PluginKeyProvider provides PluginKeys for PluginRefs.
type PluginKeyProvider interface {
	// GetPluginKeysForPluginRefs gets the PluginKets for the given PluginRefs.
	//
	// Returned PluginKeys will be in the same order as the input PluginRefs.
	//
	// The input PluginRefs are expected to be unique by PluginFullName. The implementation
	// may error if this is not the case.
	//
	// If there is no error, the length of the PluginKeys returned will match the length of the PluginRefs.
	// If there is an error, no PluginKeys will be returned.
	// If any PluginRef is not found, an error with fs.ErrNotExist will be returned.
	GetPluginKeysForPluginRefs(context.Context, []PluginRef, DigestType) ([]PluginKey, error)
}

// *** PRIVATE ***

type nopPluginKeyProvider struct{}

func (nopPluginKeyProvider) GetPluginKeysForPluginRefs(
	context.Context,
	[]PluginRef,
	DigestType,
) ([]PluginKey, error) {
	return nil, fs.ErrNotExist
}
