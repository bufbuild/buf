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

package bufcli

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"buf.build/go/app"
	"buf.build/go/app/appext"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/pkg/slogapp"
	"github.com/bufbuild/protoplugin"
)

// GetModuleConfigAndPluginConfigsForProtocPlugin gets the [bufmodule.ModuleConfig] and
// [bufmodule.PluginConfig]s for the specified module for the protoc plugin implementations.
//
// The caller can provide overrides for plugin paths in the plugin configurations. The protoc
// plugin implementations do not support remote plugins. Also, for use-cases such as Bazel,
// access to local binaries might require an explicit path override. So, this allows callers
// to pass a map of plugin name to local path to override the plugin configuration.
//
// We also return all check configs for the option [bufcheck.WithRelatedCheckConfigs] to
// validate the plugin configs when running lint/breaking.
//
// This is the same in both plugins so we just pulled it out to a common spot.
func GetModuleConfigAndPluginConfigsForProtocPlugin(
	ctx context.Context,
	configOverride string,
	module string,
	pluginPathOverrides map[string]string,
) (bufconfig.ModuleConfig, []bufconfig.PluginConfig, []bufconfig.CheckConfig, error) {
	bufYAMLFile, err := GetBufYAMLFileForDirPathOrOverride(
		ctx,
		".",
		configOverride,
	)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// There are no plugin configs by default.
			return bufconfig.DefaultModuleConfigV2, nil, []bufconfig.CheckConfig{bufconfig.DefaultLintConfigV2, bufconfig.DefaultBreakingConfigV2}, nil
		}
		return nil, nil, nil, err
	}
	if module == "" {
		module = "."
	}
	pluginConfigs, err := getPluginConfigsForPluginPathOverrides(bufYAMLFile.PluginConfigs(), pluginPathOverrides)
	if err != nil {
		return nil, nil, nil, err
	}
	var allCheckConfigs []bufconfig.CheckConfig
	// Multiple modules in a v2 workspace may have the same moduleDirPath.
	moduleConfigsFound := []bufconfig.ModuleConfig{}
	for _, moduleConfig := range bufYAMLFile.ModuleConfigs() {
		allCheckConfigs = append(allCheckConfigs, moduleConfig.LintConfig(), moduleConfig.BreakingConfig())
		// If we have a v1beta1 or v1 buf.yaml, dirPath will be ".". Using the ModuleConfig from
		// a v1beta1 or v1 buf.yaml file matches the pre-refactor behavior.
		//
		// If we have a v2 buf.yaml, users have to provide a module path or full name, otherwise
		// we can't deduce what ModuleConfig to use.
		if fullName := moduleConfig.FullName(); fullName != nil && fullName.String() == module {
			moduleConfigsFound = append(moduleConfigsFound, moduleConfig)
			continue
		}
		if dirPath := moduleConfig.DirPath(); dirPath == module {
			moduleConfigsFound = append(moduleConfigsFound, moduleConfig)
		}
	}
	switch len(moduleConfigsFound) {
	case 0:
		return nil, nil, nil, fmt.Errorf("no module found for %q", module)
	case 1:
		return moduleConfigsFound[0], pluginConfigs, allCheckConfigs, nil
	default:
		return nil, nil, nil, fmt.Errorf("multiple modules found at %q, specify its full name as <remote/owner/module> instead", module)
	}
}

// NewAppextContainerForPluginEnv creates a new appext.Container for the PluginEnv.
//
// This is used by the protoc plugins.
func NewAppextContainerForPluginEnv(
	pluginEnv protoplugin.PluginEnv,
	appName string,
	logLevelString string,
	logFormatString string,
) (appext.Container, error) {
	logLevel, err := appext.ParseLogLevel(logLevelString)
	if err != nil {
		return nil, err
	}
	logFormat, err := appext.ParseLogFormat(logFormatString)
	if err != nil {
		return nil, err
	}
	logger, err := slogapp.NewLogger(pluginEnv.Stderr, logLevel, logFormat)
	if err != nil {
		return nil, err
	}
	appContainer, err := newAppContainerForPluginEnv(pluginEnv)
	if err != nil {
		return nil, err
	}
	nameContainer, err := appext.NewNameContainer(appContainer, appName)
	if err != nil {
		return nil, err
	}
	return appext.NewContainer(
		nameContainer,
		logger,
	), nil
}

type appContainer struct {
	app.EnvContainer
	app.StderrContainer
	app.StdinContainer
	app.StdoutContainer
	app.ArgContainer
}

func newAppContainerForPluginEnv(pluginEnv protoplugin.PluginEnv) (*appContainer, error) {
	envContainer, err := app.NewEnvContainerForEnviron(pluginEnv.Environ)
	if err != nil {
		return nil, err
	}
	return &appContainer{
		EnvContainer:    envContainer,
		StderrContainer: app.NewStderrContainer(pluginEnv.Stderr),
		// cannot read against input from stdin, this is for the CodeGeneratorRequest
		StdinContainer: app.NewStdinContainer(nil),
		// cannot write output to stdout, this is for the CodeGeneratorResponse
		StdoutContainer: app.NewStdoutContainer(nil),
		// no args
		ArgContainer: app.NewArgContainer(),
	}, nil
}

// This processes the plugin path overrides for the protoc plugin implementations. It does
// the following:
//   - For each plugin config, it checks if a path override was configured
//   - If an override was found, then a new config override is created based on whether the
//     override is a WASM path.
//   - If no override was found, if the plugin config is a remote plugin, we return an error,
//     since remote plugins are not supported for protoc plugin implementations. Otherwise,
//     we use the plugin config as-is.
func getPluginConfigsForPluginPathOverrides(
	pluginConfigs []bufconfig.PluginConfig,
	pluginPathOverrides map[string]string,
) ([]bufconfig.PluginConfig, error) {
	overridePluginConfigs := make([]bufconfig.PluginConfig, len(pluginConfigs))
	for i, pluginConfig := range pluginConfigs {
		if overridePath, ok := pluginPathOverrides[pluginConfig.Name()]; ok {
			var overridePluginConfig bufconfig.PluginConfig
			var err error
			// Check if the override path is a WASM path, if so, treat as a local WASM plugin
			if filepath.Ext(overridePath) == ".wasm" {
				overridePluginConfig, err = bufconfig.NewLocalWasmPluginConfig(
					overridePath,
					pluginConfig.Options(),
					pluginConfig.Args(),
				)
				if err != nil {
					return nil, err
				}
			} else {
				// Otherwise, treat it as a non-WASM local plugin.
				overridePluginConfig, err = bufconfig.NewLocalPluginConfig(
					overridePath,
					pluginConfig.Options(),
					pluginConfig.Args(),
				)
				if err != nil {
					return nil, err
				}
			}
			overridePluginConfigs[i] = overridePluginConfig
			continue
		}
		if pluginConfig.Type() == bufconfig.PluginConfigTypeRemoteWasm {
			return nil, fmt.Errorf("remote plugin %s cannot be run with protoc plugin", pluginConfig.Name())
		}
		overridePluginConfigs[i] = pluginConfig
	}
	return overridePluginConfigs, nil
}
