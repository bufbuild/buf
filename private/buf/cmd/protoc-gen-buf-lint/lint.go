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

package lint

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/bufbuild/buf/private/pkg/protodescriptor"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/protoplugin"
)

const (
	appName        = "protoc-gen-buf-lint"
	defaultTimeout = 10 * time.Second
)

// Main is the main.
func Main() {
	protoplugin.Main(
		protoplugin.HandlerFunc(handle),
		// An `EmptyResolver` is passed to protoplugin for unmarshalling instead of defaulting to
		// protoregistry.GlobalTypes so that extensions are not inadvertently parsed from generated
		// code linked into the binary. Extensions are later reparsed with the descriptorset itself.
		// https://github.com/bufbuild/buf/issues/3306
		protoplugin.WithExtensionTypeResolver(protoencoding.EmptyResolver),
	)
}

func handle(
	ctx context.Context,
	pluginEnv protoplugin.PluginEnv,
	responseWriter protoplugin.ResponseWriter,
	request protoplugin.Request,
) (retErr error) {
	responseWriter.SetFeatureProto3Optional()
	responseWriter.SetFeatureSupportsEditions(protodescriptor.MinSupportedEdition, protodescriptor.MaxSupportedEdition)
	externalConfig := &externalConfig{}
	if err := encoding.UnmarshalJSONOrYAMLStrict(
		[]byte(request.Parameter()),
		externalConfig,
	); err != nil {
		return err
	}
	container, err := bufcli.NewAppextContainerForPluginEnv(
		pluginEnv,
		appName,
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
	// With the "buf lint" command, we build the image and then the linter can report
	// unused imports that the compiler reports. But with a plugin, we get descriptors
	// that are already built and no access to any possible associated compiler warnings.
	// So we have to analyze the files to compute the unused imports.
	image, err := bufimage.NewImageForCodeGeneratorRequest(request.CodeGeneratorRequest(), bufimage.WithUnusedImportsComputation())
	if err != nil {
		return err
	}
	// The protoc plugins only support local plugins.
	wasmRuntime, err := bufcli.NewWasmRuntime(ctx, container)
	if err != nil {
		return err
	}
	defer func() {
		retErr = errors.Join(retErr, wasmRuntime.Close(ctx))
	}()
	client, err := bufcheck.NewClient(
		container.Logger(),
		bufcheck.ClientWithStderr(pluginEnv.Stderr),
		bufcheck.ClientWithRunnerProvider(
			bufcheck.NewLocalRunnerProvider(wasmRuntime),
		),
		bufcheck.ClientWithLocalWasmPluginsFromOS(),
	)
	if err != nil {
		return err
	}
	moduleConfig, pluginConfigs, allCheckConfigs, err := bufcli.GetModuleConfigAndPluginConfigsForProtocPlugin(
		ctx,
		encoding.GetJSONStringOrStringValue(externalConfig.InputConfig),
		externalConfig.Module,
		externalConfig.PluginOverrides,
	)
	if err != nil {
		return err
	}
	if err := client.Lint(
		ctx,
		moduleConfig.LintConfig(),
		image,
		bufcheck.WithPluginConfigs(pluginConfigs...),
		// We add all check configs (both lint and breaking) across all configured modules in buf.yaml
		// as related configs to check if plugins have rules configured.
		bufcheck.WithRelatedCheckConfigs(allCheckConfigs...),
	); err != nil {
		var fileAnnotationSet bufanalysis.FileAnnotationSet
		if errors.As(err, &fileAnnotationSet) {
			buffer := bytes.NewBuffer(nil)
			if externalConfig.ErrorFormat == "config-ignore-yaml" {
				if err := bufcli.PrintFileAnnotationSetLintConfigIgnoreYAMLV1(
					buffer,
					fileAnnotationSet,
				); err != nil {
					return err
				}
			} else {
				if err := bufanalysis.PrintFileAnnotationSet(
					buffer,
					fileAnnotationSet,
					externalConfig.ErrorFormat,
				); err != nil {
					return err
				}
			}
			responseWriter.AddError(strings.TrimSpace(buffer.String()))
			return nil
		}
		return err
	}
	return nil
}

type externalConfig struct {
	InputConfig     json.RawMessage   `json:"input_config,omitempty" yaml:"input_config,omitempty"`
	Module          string            `json:"module,omitempty" yaml:"module,omitempty"`
	LogLevel        string            `json:"log_level,omitempty" yaml:"log_level,omitempty"`
	LogFormat       string            `json:"log_format,omitempty" yaml:"log_format,omitempty"`
	ErrorFormat     string            `json:"error_format,omitempty" yaml:"error_format,omitempty"`
	Timeout         time.Duration     `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	PluginOverrides map[string]string `json:"plugin_overrides,omitempty" yaml:"plugin_overrides,omitempty"`
}
