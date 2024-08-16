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
	"strings"

	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

const (
	// PluginConfigTypeLocal is the local plugin config type.
	PluginConfigTypeLocal PluginConfigType = iota + 1
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
	var pluginTypeCount int
	if externalConfig.Local != nil {
		pluginTypeCount++
	}
	if pluginTypeCount == 0 {
		return nil, errors.New("must specify local")
	}
	if pluginTypeCount > 1 {
		return nil, errors.New("must specify local")
	}
	options := make(map[string]any)
	for _, option := range externalConfig.Options {
		if len(option.Key) == 0 {
			return nil, errors.New("must specify option key")
		}
		// TODO: Validation here, how to expose from bufplugin?
		if option.Value == nil {
			return nil, errors.New("must specify option value")
		}
		options[option.Key] = option.Value
	}
	switch {
	case externalConfig.Local != nil:
		path, err := encoding.InterfaceSliceOrStringToStringSlice(externalConfig.Local)
		if err != nil {
			return nil, err
		}
		return newLocalPluginConfig(
			strings.Join(path, " "),
			options,
			path,
		)
	default:
		return nil, syserror.Newf("must specify local")
	}
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
	externalBufYAMLFilePluginV2 := externalBufYAMLFilePluginV2{}
	for key, value := range pluginConfig.Options() {
		externalBufYAMLFilePluginV2.Options = append(
			externalBufYAMLFilePluginV2.Options,
			externalBufYAMLFilePluginOptionV2{
				Key:   key,
				Value: value,
			},
		)
	}
	switch pluginConfig.Type() {
	case PluginConfigTypeLocal:
		path := pluginConfig.Path()
		switch {
		case len(path) == 1:
			externalBufYAMLFilePluginV2.Local = path[0]
		case len(path) > 1:
			externalBufYAMLFilePluginV2.Local = path
		}
	}
	return externalBufYAMLFilePluginV2, nil
}
