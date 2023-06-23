// Copyright 2020-2023 Buf Technologies, Inc.
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

package bufgenv2

import (
	"fmt"
	"math"

	"github.com/bufbuild/buf/private/buf/bufgen/internal"
	"github.com/bufbuild/buf/private/buf/bufgen/internal/bufgenplugin"
	"github.com/bufbuild/buf/private/pkg/encoding"
)

const (
	typeRemote        = "remote"
	typeBinary        = "binary"
	typeProtocBuiltin = "protoc_builtin"
)

const (
	optionRevision   = "revision"
	optionProtocPath = "protoc_path"
	optionStrategy   = "strategy"
)

var allowedOptionsForType = map[string](map[string]bool){
	typeRemote: {
		optionRevision: true,
	},
	typeBinary: {
		optionStrategy: true,
	},
	typeProtocBuiltin: {
		optionProtocPath: true,
		optionStrategy:   true,
	},
}

func newPluginConfigs(externalConfigs []ExternalPluginConfigV2, id string) ([]bufgenplugin.PluginConfig, error) {
	if len(externalConfigs) == 0 {
		return nil, fmt.Errorf("%s: no plugins set", id)
	}
	pluginConfigs := make([]bufgenplugin.PluginConfig, 0, len(externalConfigs))
	for _, externalConfig := range externalConfigs {
		pluginConfig, err := newPluginConfig(externalConfig)
		if err != nil {
			return nil, err
		}
		pluginConfigs = append(pluginConfigs, pluginConfig)
	}
	return pluginConfigs, nil
}

func newPluginConfig(externalConfig ExternalPluginConfigV2) (bufgenplugin.PluginConfig, error) {
	pluginTypes, options, err := getTypesAndOptions(externalConfig)
	if err != nil {
		return nil, err
	}
	if len(pluginTypes) == 0 {
		return nil, fmt.Errorf("must specify one of %s, %s and %s", typeRemote, typeBinary, typeProtocBuiltin)
	}
	if len(pluginTypes) > 1 {
		return nil, fmt.Errorf("only one of %s, %s and %s is allowed", typeRemote, typeBinary, typeProtocBuiltin)
	}
	pluginType := pluginTypes[0]
	allowedOptions := allowedOptionsForType[pluginType]
	for _, option := range options {
		if !allowedOptions[option] {
			return nil, fmt.Errorf("%s is not allowed for %s plugin", option, pluginType)
		}
	}
	var strategy string
	if externalConfig.Strategy != nil {
		strategy = *externalConfig.Strategy
	}
	parsedStrategy, err := internal.ParseStrategy(strategy)
	if err != nil {
		return nil, err
	}
	opt, err := encoding.InterfaceSliceOrStringToCommaSepString(externalConfig.Opt)
	if err != nil {
		return nil, err
	}
	switch pluginType {
	case typeRemote:
		var revision int
		if externalConfig.Revision != nil {
			revision = *externalConfig.Revision
		}
		if revision < 0 || revision > math.MaxInt32 {
			return nil, fmt.Errorf("revision %d is out of accepted range %d-%d", revision, 0, math.MaxInt32)
		}
		return bufgenplugin.NewCuratedPluginConfig(
			*externalConfig.Remote,
			revision,
			externalConfig.Out,
			opt,
			externalConfig.IncludeImports,
			externalConfig.IncludeWKT,
		)
	case typeBinary:
		path, err := encoding.InterfaceSliceOrStringToStringSlice(externalConfig.Binary)
		if err != nil {
			return nil, err
		}
		return bufgenplugin.NewBinaryPluginConfig(
			"",
			path,
			parsedStrategy,
			externalConfig.Out,
			opt,
			externalConfig.IncludeImports,
			externalConfig.IncludeWKT,
		)
	case typeProtocBuiltin:
		var protocPath string
		if externalConfig.ProtocPath != nil {
			protocPath = *externalConfig.ProtocPath
		}
		return bufgenplugin.NewProtocBuiltinPluginConfig(
			*externalConfig.ProtocBuiltin,
			protocPath,
			externalConfig.Out,
			opt,
			externalConfig.IncludeImports,
			externalConfig.IncludeWKT,
			parsedStrategy,
		)
	default:
		// this should not happen
		return nil, fmt.Errorf("must specify one of %s, %s and %s", typeRemote, typeBinary, typeProtocBuiltin)
	}
}

func getTypesAndOptions(externalConfig ExternalPluginConfigV2) ([]string, []string, error) {
	var (
		types   []string
		options []string
	)
	if externalConfig.Remote != nil {
		types = append(types, typeRemote)
	}
	path, err := encoding.InterfaceSliceOrStringToStringSlice(externalConfig.Binary)
	if err != nil {
		return nil, nil, err
	}
	if len(path) > 0 {
		types = append(types, typeBinary)
	}
	if externalConfig.ProtocBuiltin != nil {
		types = append(types, typeProtocBuiltin)
	}
	if externalConfig.Revision != nil {
		options = append(options, optionRevision)
	}
	if externalConfig.ProtocPath != nil {
		options = append(options, optionProtocPath)
	}
	if externalConfig.Strategy != nil {
		options = append(options, optionStrategy)
	}
	return types, options, nil
}
