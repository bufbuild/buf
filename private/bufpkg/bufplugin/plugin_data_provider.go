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
	// NopPluginDataProvider is a no-op PluginDataProvider.
	NopPluginDataProvider PluginDataProvider = nopPluginDataProvider{}
)

// PluginDataProvider provides PluginsDatas.
type PluginDataProvider interface {
	// GetPluginDatasForPluginKeys gets the PluginDatas for the PluginKeys.
	//
	// Returned PluginDatas will be in the same order as the input PluginKeys.
	//
	// The input PluginKeys are expected to be unique by PluginFullName. The implementation
	// may error if this is not the case.
	//
	// The input PluginKeys are expected to have the same DigestType. The implementation
	// may error if this is not the case.
	//
	// If there is no error, the length of the PluginDatas returned will match the length of the PluginKeys.
	// If there is an error, no PluginDatas will be returned.
	// If any PluginKey is not found, an error with fs.ErrNotExist will be returned.
	GetPluginDatasForPluginKeys(
		context.Context,
		[]PluginKey,
	) ([]PluginData, error)
}

// *** PRIVATE ***

type nopPluginDataProvider struct{}

func (nopPluginDataProvider) GetPluginDatasForPluginKeys(
	context.Context,
	[]PluginKey,
) ([]PluginData, error) {
	return nil, fs.ErrNotExist
}
