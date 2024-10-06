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

package internal

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/slogapp"
	"github.com/bufbuild/protoplugin"
)

// GetModuleConfigForProtocPlugin gets ModuleConfigs for the protoc plugin implementations.
//
// This is the same in both plugins so we just pulled it out to a common spot.
func GetModuleConfigForProtocPlugin(
	ctx context.Context,
	configOverride string,
	module string,
) (bufconfig.ModuleConfig, error) {
	bufYAMLFile, err := bufcli.GetBufYAMLFileForDirPathOrOverride(
		ctx,
		".",
		configOverride,
	)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return bufconfig.DefaultModuleConfigV1, nil
		}
		return nil, err
	}
	if module == "" {
		module = "."
	}
	// Multiple modules in a v2 workspace may have the same moduleDirPath.
	moduleConfigsFound := []bufconfig.ModuleConfig{}
	for _, moduleConfig := range bufYAMLFile.ModuleConfigs() {
		// If we have a v1beta1 or v1 buf.yaml, dirPath will be ".". Using the ModuleConfig from
		// a v1beta1 or v1 buf.yaml file matches the pre-refactor behavior.
		//
		// If we have a v2 buf.yaml, users have to provide a module path or full name, otherwise
		// we can't deduce what ModuleConfig to use.
		if fullName := moduleConfig.ModuleFullName(); fullName != nil && fullName.String() == module {
			// Can return here because BufYAMLFile guarantees that module full names are unique across
			// its module configs.
			return moduleConfig, nil
		}
		if dirPath := moduleConfig.DirPath(); dirPath == module {
			moduleConfigsFound = append(moduleConfigsFound, moduleConfig)
		}
	}
	switch len(moduleConfigsFound) {
	case 0:
		return nil, fmt.Errorf("no module found for %q", module)
	case 1:
		return moduleConfigsFound[0], nil
	default:
		return nil, fmt.Errorf("multiple modules found at %q, specify its full name as <remote/owner/module> instead", module)
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
