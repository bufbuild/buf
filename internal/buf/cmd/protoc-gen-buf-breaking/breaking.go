// Copyright 2020-2021 Buf Technologies, Inc.
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
	"fmt"
	"strings"
	"time"

	"github.com/bufbuild/buf/internal/buf/bufanalysis"
	"github.com/bufbuild/buf/internal/buf/bufcheck/bufbreaking"
	"github.com/bufbuild/buf/internal/buf/bufcli"
	"github.com/bufbuild/buf/internal/buf/bufconfig"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufimage"
	"github.com/bufbuild/buf/internal/buf/buffetch"
	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/app/applog"
	"github.com/bufbuild/buf/internal/pkg/app/appproto"
	"github.com/bufbuild/buf/internal/pkg/encoding"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"google.golang.org/protobuf/types/pluginpb"
)

const defaultTimeout = 10 * time.Second

// Main is the main.
func Main(options ...MainOption) {
	mainOptions := newMainOptions()
	for _, option := range options {
		option(mainOptions)
	}
	appproto.Main(
		context.Background(),
		appproto.HandlerFunc(
			func(
				ctx context.Context,
				container app.EnvStderrContainer,
				responseWriter appproto.ResponseWriter,
				request *pluginpb.CodeGeneratorRequest,
			) error {
				return handle(
					ctx,
					container,
					responseWriter,
					request,
					mainOptions.oldBinaryName,
					mainOptions.newBinaryName,
				)
			},
		),
	)
}

// MainOption is an option for Main.
type MainOption func(*mainOptions)

// WithDeprecatedBinaryName returns a new MainOption that marks this binary
// as deprecated and points from the old binary name to the new binary name.
func WithDeprecatedBinaryName(oldBinaryName string, newBinaryName string) MainOption {
	return func(mainOptions *mainOptions) {
		mainOptions.oldBinaryName = oldBinaryName
		mainOptions.newBinaryName = newBinaryName
	}
}

func handle(
	ctx context.Context,
	container app.EnvStderrContainer,
	responseWriter appproto.ResponseWriter,
	request *pluginpb.CodeGeneratorRequest,
	oldBinaryName string,
	newBinaryName string,
) error {
	responseWriter.SetFeatureProto3Optional()
	externalConfig := &externalConfig{}
	if err := encoding.UnmarshalJSONOrYAMLStrict(
		[]byte(request.GetParameter()),
		externalConfig,
	); err != nil {
		return err
	}
	if externalConfig.AgainstInput == "" {
		// this is actually checked as part of ReadImageEnv but just in case
		return errors.New(`"against_input" is required`)
	}
	timeout := externalConfig.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	logger, err := applog.NewLogger(container.Stderr(), externalConfig.LogLevel, externalConfig.LogFormat)
	if err != nil {
		return err
	}
	if oldBinaryName != "" && newBinaryName != "" {
		logger.Sugar().Warnf(
			"%s is deprecated. Use %s instead. %s can be installed in the same manner as %s, whether from GitHub Releases, Homebrew, AUR, or direct Go installation.",
			oldBinaryName,
			newBinaryName,
			newBinaryName,
			oldBinaryName,
		)
	}

	files := request.FileToGenerate
	if !externalConfig.LimitToInputFiles {
		files = nil
	}
	againstImageRef, err := buffetch.NewImageRefParser(logger).GetImageRef(ctx, externalConfig.AgainstInput)
	if err != nil {
		return fmt.Errorf("against_input: %v", err)
	}
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	imageReader := bufcli.NewWireImageReader(logger, storageosProvider)
	againstImage, err := imageReader.GetImage(
		ctx,
		newContainer(container),
		againstImageRef,
		files, // limit to the input files if specified
		true,  // allow files in the against input to not exist
		false, // keep for now
	)
	if err != nil {
		return err
	}
	if externalConfig.ExcludeImports {
		againstImage = bufimage.ImageWithoutImports(againstImage)
	}
	readWriteBucket, err := storageosProvider.NewReadWriteBucket(
		".",
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return err
	}
	config, err := bufconfig.ReadConfig(
		ctx,
		bufconfig.NewProvider(logger),
		readWriteBucket,
		bufconfig.ReadConfigWithOverride(
			encoding.GetJSONStringOrStringValue(externalConfig.InputConfig),
		),
	)
	if err != nil {
		return err
	}
	image, err := bufimage.NewImageForCodeGeneratorRequest(request)
	if err != nil {
		return err
	}
	fileAnnotations, err := bufbreaking.NewHandler(logger).Check(
		ctx,
		config.Breaking,
		againstImage,
		image,
	)
	if err != nil {
		return err
	}
	if len(fileAnnotations) > 0 {
		buffer := bytes.NewBuffer(nil)
		if err := bufanalysis.PrintFileAnnotations(buffer, fileAnnotations, externalConfig.ErrorFormat); err != nil {
			return err
		}
		responseWriter.AddError(strings.TrimSpace(buffer.String()))
	}
	return nil
}

type externalConfig struct {
	AgainstInput       string          `json:"against_input,omitempty" yaml:"against_input,omitempty"`
	AgainstInputConfig json.RawMessage `json:"against_input_config,omitempty" yaml:"against_input_config,omitempty"`
	InputConfig        json.RawMessage `json:"input_config,omitempty" yaml:"input_config,omitempty"`
	LimitToInputFiles  bool            `json:"limit_to_input_files,omitempty" yaml:"limit_to_input_files,omitempty"`
	ExcludeImports     bool            `json:"exclude_imports,omitempty" yaml:"exclude_imports,omitempty"`
	LogLevel           string          `json:"log_level,omitempty" yaml:"log_level,omitempty"`
	LogFormat          string          `json:"log_format,omitempty" yaml:"log_format,omitempty"`
	ErrorFormat        string          `json:"error_format,omitempty" yaml:"error_format,omitempty"`
	Timeout            time.Duration   `json:"timeout,omitempty" yaml:"timeout,omitempty"`
}

type container struct {
	app.EnvContainer
	app.StdinContainer
}

func newContainer(c app.EnvContainer) *container {
	return &container{
		EnvContainer: c,
		// cannot read against input from stdin, this is for the CodeGeneratorRequest
		StdinContainer: app.NewStdinContainer(nil),
	}
}

type mainOptions struct {
	oldBinaryName string
	newBinaryName string
}

func newMainOptions() *mainOptions {
	return &mainOptions{}
}
