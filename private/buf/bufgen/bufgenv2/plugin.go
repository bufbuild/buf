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

	"github.com/bufbuild/buf/private/buf/bufgen"
	"github.com/bufbuild/buf/private/buf/bufgen/internal"
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
)

var allowedOptionsForType = map[string](map[string]bool){
	typeRemote: {
		optionRevision: true,
	},
	typeBinary: nil,
	typeProtocBuiltin: {
		optionProtocPath: true,
	},
}

func newPluginConfig(externalConfig ExternalPluginConfigV2) (bufgen.PluginConfig, error) {
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
			return nil, fmt.Errorf("%s is not allowed for %s", option, pluginType)
		}
	}
	strategy, err := internal.ParseStrategy(externalConfig.Strategy)
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
		return bufgen.NewCuratedPluginConfig(
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
		return bufgen.NewBinaryPluginConfig(
			"",
			path,
			strategy,
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
		return bufgen.NewProtocBuiltinPluginConfig(
			*externalConfig.ProtocBuiltin,
			protocPath,
			externalConfig.Out,
			opt,
			externalConfig.IncludeImports,
			externalConfig.IncludeWKT,
			strategy,
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
	return types, options, nil
}
