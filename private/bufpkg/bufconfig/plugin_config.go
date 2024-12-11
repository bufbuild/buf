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
	"os"
	"path/filepath"

	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

const (
	// PluginConfigTypeLocal is the local plugin config type.
	PluginConfigTypeLocal PluginConfigType = iota + 1
	// PluginConfigTypeLocalWasm is the local Wasm plugin config type.
	PluginConfigTypeLocalWasm
	// PluginConfigTypeRemoteWasm is the remote Wasm plugin config type.
	PluginConfigTypeRemoteWasm
)

// PluginConfigType is a generate plugin configuration type.
type PluginConfigType int

// PluginConfig is a configuration for a plugin.
type PluginConfig interface {
	// Type returns the plugin type. This is never the zero value.
	Type() PluginConfigType
	// Name returns the plugin name.
	//
	// If the plugin is of Type PluginConfigTypeLocal, the Name is the command
	// name or path to the command.
	// If the plugin is of Type PluginConfigTypeLocalWasm, the Name is the
	// path to the Wasm file. It must end with .wasm.
	// If the plugin is of Type PluginConfigTypeRemoteWasm, the Name is a
	// vaild bufparse.Ref and Ref returns the plugin reference.
	//
	// This is never empty.
	Name() string
	// Options returns the plugin options.
	//
	// TODO: Will want good validation and good error messages for what this decodes.
	// Otherwise we will confuse users. Do QA.
	Options() map[string]any
	// Args returns the arguments, excluding the name, for the plugin.
	//
	// This may be empty.
	Args() []string
	// Ref returns the plugin reference.
	//
	// This is only non-nil when the plugin is remote.
	Ref() bufparse.Ref

	isPluginConfig()
}

// NewLocalPluginConfig returns a new PluginConfig for a local plugin.
func NewLocalPluginConfig(
	name string,
	args []string,
	options map[string]any,
) (PluginConfig, error) {
	return newLocalPluginConfig(
		name,
		args,
		options,
	)
}

// NewLocalWasmPluginConfig returns a new PluginConfig for a local Wasm plugin.
//
// The name is the path to the Wasm plugin and must end with .wasm.
// The args are the arguments to the Wasm plugin. These are passed to the Wasm plugin
// as command line arguments.
func NewLocalWasmPluginConfig(
	name string,
	args []string,
	options map[string]any,
) (PluginConfig, error) {
	return newLocalWasmPluginConfig(
		name,
		args,
		options,
	)
}

// NewRemoteWasmPluginConfig returns a new PluginConfig for a remote Wasm plugin.
//
// The pluginRef is the remote reference to the plugin.
// The args are the arguments to the remote plugin. These are passed to the remote plugin
// as command line arguments.
func NewRemoteWasmPluginConfig(
	pluginRef bufparse.Ref,
	args []string,
	options map[string]any,
) (PluginConfig, error) {
	return newRemotePluginConfig(
		pluginRef,
		args,
		options,
	)
}

// *** PRIVATE ***

type pluginConfig struct {
	pluginConfigType PluginConfigType
	options          map[string]any
	name             string
	args             []string
	ref              bufparse.Ref
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
	// Plugins are specified as a path, remote reference, or Wasm file.
	path, err := encoding.InterfaceSliceOrStringToStringSlice(externalConfig.Plugin)
	if err != nil {
		return nil, err
	}
	if len(path) == 0 {
		return nil, errors.New("must specify a path to the plugin")
	}
	name, args := path[0], path[1:]
	// Remote plugins are specified as plugin references.
	if pluginRef, err := bufparse.ParseRef(path[0]); err == nil {
		// Check if the local filepath exists, if it does presume its
		// not a remote reference. Okay to use os.Stat instead of
		// os.Lstat.
		if _, err := os.Stat(path[0]); os.IsNotExist(err) {
			return newRemotePluginConfig(
				pluginRef,
				args,
				options,
			)
		}
	}
	// Wasm plugins are suffixed with .wasm. Otherwise, it's a binary.
	if filepath.Ext(path[0]) == ".wasm" {
		return newLocalWasmPluginConfig(
			name,
			args,
			options,
		)
	}
	return newLocalPluginConfig(
		name,
		args,
		options,
	)
}

func newLocalPluginConfig(
	name string,
	args []string,
	options map[string]any,
) (*pluginConfig, error) {
	if len(name) == 0 {
		return nil, errors.New("must specify a path to the plugin")
	}
	return &pluginConfig{
		pluginConfigType: PluginConfigTypeLocal,
		options:          options,
		name:             name,
		args:             args,
	}, nil
}

func newLocalWasmPluginConfig(
	name string,
	args []string,
	options map[string]any,
) (*pluginConfig, error) {
	if len(name) == 0 {
		return nil, errors.New("must specify a path to the plugin")
	}
	if filepath.Ext(name) != ".wasm" {
		return nil, fmt.Errorf("must specify a path to the plugin, and the first path argument must end with .wasm")
	}
	return &pluginConfig{
		pluginConfigType: PluginConfigTypeLocalWasm,
		options:          options,
		name:             name,
		args:             args,
	}, nil
}

func newRemotePluginConfig(
	pluginRef bufparse.Ref,
	args []string,
	options map[string]any,
) (*pluginConfig, error) {
	return &pluginConfig{
		pluginConfigType: PluginConfigTypeRemoteWasm,
		name:             pluginRef.String(),
		options:          options,
		args:             args,
		ref:              pluginRef,
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

func (p *pluginConfig) Args() []string {
	return p.args
}

func (p *pluginConfig) Ref() bufparse.Ref {
	return p.ref
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
	args := pluginConfig.Args()
	switch {
	case len(args) == 0:
		externalBufYAMLFilePluginV2.Plugin = pluginConfig.Name()
	case len(args) > 0:
		externalBufYAMLFilePluginV2.Plugin = append([]string{pluginConfig.Name()}, args...)
	}
	return externalBufYAMLFilePluginV2, nil
}
