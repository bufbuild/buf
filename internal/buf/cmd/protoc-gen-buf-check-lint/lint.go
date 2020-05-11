// Copyright 2020 Buf Technologies Inc.
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
	"time"

	"github.com/bufbuild/buf/internal/buf/bufconfig"
	"github.com/bufbuild/buf/internal/buf/cmd/internal"
	"github.com/bufbuild/buf/internal/buf/ext/extfile"
	"github.com/bufbuild/buf/internal/buf/ext/extimage"
	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/app/applog"
	"github.com/bufbuild/buf/internal/pkg/app/appproto"
	"github.com/bufbuild/buf/internal/pkg/util/utilencoding"
	"google.golang.org/protobuf/types/pluginpb"
)

const defaultTimeout = 10 * time.Second

// Main is the main.
func Main() {
	appproto.Main(context.Background(), handle)
}

func handle(
	ctx context.Context,
	container app.EnvStderrContainer,
	responseWriter appproto.ResponseWriter,
	request *pluginpb.CodeGeneratorRequest,
) {
	externalConfig := &externalConfig{}
	if err := utilencoding.UnmarshalJSONOrYAMLStrict(
		[]byte(request.GetParameter()),
		externalConfig,
	); err != nil {
		responseWriter.WriteError(err.Error())
		return
	}
	timeout := externalConfig.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	logger, err := applog.NewLogger(container.Stderr(), externalConfig.LogLevel, externalConfig.LogFormat)
	if err != nil {
		responseWriter.WriteError(err.Error())
		return
	}
	envReader := internal.NewBufosEnvReader(logger, "", "input_config", false)
	config, err := envReader.GetConfig(ctx, utilencoding.GetJSONStringOrStringValue(externalConfig.InputConfig))
	if err != nil {
		responseWriter.WriteError(err.Error())
		return
	}
	image, err := extimage.CodeGeneratorRequestToImage(request)
	if err != nil {
		responseWriter.WriteError(err.Error())
		return
	}
	fileAnnotations, err := internal.NewBuflintHandler(logger).LintCheck(
		ctx,
		config.Lint,
		image,
	)
	if err != nil {
		responseWriter.WriteError(err.Error())
		return
	}
	asJSON, err := internal.IsLintFormatJSON("error_format", externalConfig.ErrorFormat)
	if err != nil {
		responseWriter.WriteError(err.Error())
		return
	}
	asConfigIgnoreYAML, err := internal.IsLintFormatConfigIgnoreYAML("error_format", externalConfig.ErrorFormat)
	if err != nil {
		responseWriter.WriteError(err.Error())
		return
	}
	buffer := bytes.NewBuffer(nil)
	if asConfigIgnoreYAML {
		if err := bufconfig.PrintFileAnnotationsLintConfigIgnoreYAML(buffer, fileAnnotations); err != nil {
			responseWriter.WriteError(err.Error())
			return
		}
	} else {
		if err := extfile.PrintFileAnnotations(buffer, fileAnnotations, asJSON); err != nil {
			responseWriter.WriteError(err.Error())
			return
		}
	}
	responseWriter.WriteError(buffer.String())
}

type externalConfig struct {
	InputConfig json.RawMessage `json:"input_config,omitempty" yaml:"input_config,omitempty"`
	LogLevel    string          `json:"log_level,omitempty" yaml:"log_level,omitempty"`
	LogFormat   string          `json:"log_format,omitempty" yaml:"log_format,omitempty"`
	ErrorFormat string          `json:"error_format,omitempty" yaml:"error_format,omitempty"`
	Timeout     time.Duration   `json:"timeout,omitempty" yaml:"timeout,omitempty"`
}
