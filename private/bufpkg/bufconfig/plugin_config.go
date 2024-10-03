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

package bufconfig

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

const (
	// PluginConfigTypeLocal is the local plugin config type.
	PluginConfigTypeLocal PluginConfigType = iota + 1
	// PluginConfigTypeLocalWasm is the local Wasm plugin config type.
	PluginConfigTypeLocalWasm
)

// PluginConfigType is a generate plugin configuration type.
type PluginConfigType int

// PluginConfig is a configuration for a plugin.
type PluginConfig interface {
	// Type returns the plugin type. This is never the zero value.
	Type() PluginConfigType
	// Name returns the plugin name. This is never empty.
	Name() string
	// Options returns the plugin options.
	//
	// TODO: Will want good validation and good error messages for what this decodes.
	// Otherwise we will confuse users. Do QA.
	Options() map[string]any
	// Path returns the path, including arguments, to invoke the binary plugin.
	//
	// This is not empty only when the plugin is local.
	Path() []string

	isPluginConfig()
}

// NewLocalPluginConfig returns a new PluginConfig for a local plugin.
func NewLocalPluginConfig(
	name string,
	options map[string]any,
	path []string,
) (PluginConfig, error) {
	return newLocalPluginConfig(
		name,
		options,
		path,
	)
}

// NewLocalWasmPluginConfig returns a new PluginConfig for a local Wasm plugin.
//
// The first path argument is the path to the Wasm plugin and must end with .wasm.
// The remaining path arguments are the arguments to the Wasm plugin. These are passed
// to the Wasm plugin as command line arguments.
func NewLocalWasmPluginConfig(
	name string,
	options map[string]any,
	path []string,
) (PluginConfig, error) {
	return newLocalWasmPluginConfig(
		name,
		options,
		path,
	)
}

// *** PRIVATE ***

type pluginConfig struct {
	pluginConfigType PluginConfigType
	name             string
	options          map[string]any
	path             []string
}

func newPluginConfigForExternalV2(
	externalConfig externalBufYAMLFilePluginV2,
) (PluginConfig, error) {
	options := make(map[string]any)
	for key, value := range externalConfig.Options {
		if len(key) == 0 {
			return nil, errors.New("must specify option key")
		}
		// TODO: Validation here, how to expose from bufplugin?
		if value == nil {
			return nil, errors.New("must specify option value")
		}
		options[key] = value
	}
	// TODO: differentiate between local and remote in the future
	// Use the same heuristic that we do for dir vs module in buffetch
	path, err := encoding.InterfaceSliceOrStringToStringSlice(externalConfig.Plugin)
	if err != nil {
		return nil, err
	}
	if len(path) == 0 {
		return nil, errors.New("must specify a path to the plugin")
	}
	// Wasm plugins are suffixed with .wasm. Otherwise, it's a binary.
	if filepath.Ext(path[0]) == ".wasm" {
		return newLocalWasmPluginConfig(
			strings.Join(path, " "),
			options,
			path,
		)
	}
	return newLocalPluginConfig(
		strings.Join(path, " "),
		options,
		path,
	)
}

func newLocalPluginConfig(
	name string,
	options map[string]any,
	path []string,
) (*pluginConfig, error) {
	if len(path) == 0 {
		return nil, errors.New("must specify a path to the plugin")
	}
	return &pluginConfig{
		pluginConfigType: PluginConfigTypeLocal,
		name:             name,
		options:          options,
		path:             path,
	}, nil
}

func newLocalWasmPluginConfig(
	name string,
	options map[string]any,
	path []string,
) (*pluginConfig, error) {
	if len(path) == 0 {
		return nil, errors.New("must specify a path to the plugin")
	}
	if filepath.Ext(path[0]) != ".wasm" {
		return nil, fmt.Errorf("must specify a path to the plugin, and the first path argument must end with .wasm")
	}
	return &pluginConfig{
		pluginConfigType: PluginConfigTypeLocalWasm,
		name:             name,
		options:          options,
		path:             path,
	}, nil
}

func (p *pluginConfig) Type() PluginConfigType {
	return p.pluginConfigType
}

func (p *pluginConfig) Name() string {
	return p.name
}

func (p *pluginConfig) Options() map[string]any {
	return p.options
}

func (p *pluginConfig) Path() []string {
	return p.path
}

func (p *pluginConfig) isPluginConfig() {}

func newExternalV2ForPluginConfig(
	config PluginConfig,
) (externalBufYAMLFilePluginV2, error) {
	pluginConfig, ok := config.(*pluginConfig)
	if !ok {
		return externalBufYAMLFilePluginV2{}, syserror.Newf("unknown implementation of PluginConfig: %T", pluginConfig)
	}
	externalBufYAMLFilePluginV2 := externalBufYAMLFilePluginV2{
		Options: pluginConfig.Options(),
	}
	switch pluginConfig.Type() {
	case PluginConfigTypeLocal:
		path := pluginConfig.Path()
		switch {
		case len(path) == 1:
			externalBufYAMLFilePluginV2.Plugin = path[0]
		case len(path) > 1:
			externalBufYAMLFilePluginV2.Plugin = path
		}
	}
	return externalBufYAMLFilePluginV2, nil
}
