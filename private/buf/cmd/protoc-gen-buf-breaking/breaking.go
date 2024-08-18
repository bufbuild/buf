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

package breaking

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/bufbuild/buf/private/buf/bufcheck"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/buf/cmd/internal"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/bufbuild/buf/private/pkg/protodescriptor"
	"github.com/bufbuild/buf/private/pkg/verbose"
	"github.com/bufbuild/buf/private/pkg/zaputil"
	"github.com/bufbuild/protoplugin"
)

const (
	appName        = "protoc-gen-buf-breaking"
	defaultTimeout = 10 * time.Second
)

// Main is the main.
func Main() {
	protoplugin.Main(protoplugin.HandlerFunc(handle))
}

func handle(
	ctx context.Context,
	pluginEnv protoplugin.PluginEnv,
	responseWriter protoplugin.ResponseWriter,
	request protoplugin.Request,
) error {
	responseWriter.SetFeatureProto3Optional()
	responseWriter.SetFeatureSupportsEditions(protodescriptor.MinSupportedEdition, protodescriptor.MaxSupportedEdition)
	externalConfig := &externalConfig{}
	if err := encoding.UnmarshalJSONOrYAMLStrict(
		[]byte(request.Parameter()),
		externalConfig,
	); err != nil {
		return err
	}
	if externalConfig.AgainstInput == "" {
		// this is actually checked as part of ReadImageEnv but just in case
		return errors.New(`"against_input" is required`)
	}
	container, err := newAppextContainer(
		pluginEnv,
		externalConfig.LogLevel,
		externalConfig.LogFormat,
	)
	if err != nil {
		return err
	}
	timeout := externalConfig.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var targetPaths []string
	if externalConfig.LimitToInputFiles {
		targetPaths = request.CodeGeneratorRequest().GetFileToGenerate()
	}
	controller, err := bufcli.NewController(
		container,
		bufctl.WithFileAnnotationErrorFormat(externalConfig.ErrorFormat),
	)
	if err != nil {
		return err
	}
	againstImage, err := controller.GetImage(
		ctx,
		externalConfig.AgainstInput,
		// limit to the input files if specified
		bufctl.WithTargetPaths(targetPaths, nil),
	)
	if err != nil {
		return err
	}
	var breakingOptions []bufcheck.BreakingOption
	if externalConfig.ExcludeImports {
		breakingOptions = append(breakingOptions, bufcheck.BreakingWithExcludeImports())
	}
	moduleConfig, err := internal.GetModuleConfigForProtocPlugin(
		ctx,
		encoding.GetJSONStringOrStringValue(externalConfig.InputConfig),
		externalConfig.Module,
	)
	if err != nil {
		return err
	}
	image, err := bufimage.NewImageForCodeGeneratorRequest(request.CodeGeneratorRequest())
	if err != nil {
		return err
	}
	// The protoc plugins do not support custom lint/breaking change plugins for now.
	client, err := bufcheck.NewClient(command.NewRunner(), bufcheck.ClientWithStderr(pluginEnv.Stderr))
	if err != nil {
		return err
	}
	if err := client.Breaking(
		ctx,
		moduleConfig.BreakingConfig(),
		image,
		againstImage,
		breakingOptions...,
	); err != nil {
		var fileAnnotationSet bufanalysis.FileAnnotationSet
		if errors.As(err, &fileAnnotationSet) {
			buffer := bytes.NewBuffer(nil)
			if err := bufanalysis.PrintFileAnnotationSet(
				buffer,
				fileAnnotationSet,
				externalConfig.ErrorFormat,
			); err != nil {
				return err
			}
			responseWriter.SetError(strings.TrimSpace(buffer.String()))
			return nil
		}
		return err
	}
	return nil
}

type externalConfig struct {
	AgainstInput string `json:"against_input,omitempty" yaml:"against_input,omitempty"`
	// This was never actually used, but we keep it around for we can do unmarshal strict without breaking anyone.
	AgainstInputConfig json.RawMessage `json:"against_input_config,omitempty" yaml:"against_input_config,omitempty"`
	InputConfig        json.RawMessage `json:"input_config,omitempty" yaml:"input_config,omitempty"`
	Module             string          `json:"module,omitempty" yaml:"module,omitempty"`
	LimitToInputFiles  bool            `json:"limit_to_input_files,omitempty" yaml:"limit_to_input_files,omitempty"`
	ExcludeImports     bool            `json:"exclude_imports,omitempty" yaml:"exclude_imports,omitempty"`
	LogLevel           string          `json:"log_level,omitempty" yaml:"log_level,omitempty"`
	LogFormat          string          `json:"log_format,omitempty" yaml:"log_format,omitempty"`
	ErrorFormat        string          `json:"error_format,omitempty" yaml:"error_format,omitempty"`
	Timeout            time.Duration   `json:"timeout,omitempty" yaml:"timeout,omitempty"`
}

func newAppextContainer(
	pluginEnv protoplugin.PluginEnv,
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
	appContainer, err := newAppContainer(pluginEnv)
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

func newAppContainer(pluginEnv protoplugin.PluginEnv) (*appContainer, error) {
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
