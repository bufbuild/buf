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
	"io/fs"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/verbose"
	"github.com/bufbuild/buf/private/pkg/zaputil"
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
	for _, moduleConfig := range bufYAMLFile.ModuleConfigs() {
		// If we have a v1beta1 or v1 buf.yaml, dirPath will be ".". Using the ModuleConfig from
		// a v1beta1 or v1 buf.yaml file matches the pre-refactor behavior.
		//
		// If we have a v2 buf.yaml, users have to provide a module path or full name, otherwise
		// we can't deduce what ModuleConfig to use.
		if dirPath := moduleConfig.DirPath(); dirPath == module {
			return moduleConfig, nil
		}
		if fullName := moduleConfig.ModuleFullName(); fullName != nil && fullName.String() == module {
			return moduleConfig, nil
		}
	}
	// TODO: point to a webpage that explains this.
	return nil, errors.New(`could not determine which module to pull configuration from. See the docs for more details.`)
}

// NewAppextContainerForPluginEnv creates a new appext.Container for the PluginEnv.
//
// This isu sed bt the protoc plugins.
func NewAppextContainerForPluginEnv(
	pluginEnv protoplugin.PluginEnv,
	appName string,
	logLevel string,
	logFormat string,
) (appext.Container, error) {
	logger, err := zaputil.NewLoggerForFlagValues(
		pluginEnv.Stderr,
		logLevel,
		logFormat,
	)
	if err != nil {
		return nil, err
	}
	appContainer, err := newAppContainerForPluginEnv(pluginEnv)
	if err != nil {
		return nil, err
	}
	return appext.NewContainer(
		appContainer,
		appName,
		logger,
		verbose.NopPrinter,
	)
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
