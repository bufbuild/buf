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

package bufgen

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/bufbuild/buf/private/pkg/storage"
	"go.uber.org/zap"
)

func newReadConfigOptions() *readConfigOptions {
	return &readConfigOptions{}
}

type readConfigOptions struct {
	override string
}

func readConfigVersion(
	ctx context.Context,
	logger *zap.Logger,
	provider ConfigDataProvider,
	readBucket storage.ReadBucket,
	options ...ReadConfigOption,
) (string, error) {
	version, err := ReadFromConfig(
		ctx,
		logger,
		provider,
		readBucket,
		getConfigVersion,
		options...,
	)
	if err != nil || version == nil {
		return "", err
	}
	return *version, nil
}

func readFromConfig[V any](
	ctx context.Context,
	logger *zap.Logger,
	provider ConfigDataProvider,
	readBucket storage.ReadBucket,
	configGetter ConfigGetter[V],
	options ...ReadConfigOption,
) (*V, error) {
	readConfigOptions := newReadConfigOptions()
	for _, option := range options {
		option(readConfigOptions)
	}
	if override := readConfigOptions.override; override != "" {
		switch filepath.Ext(override) {
		case ".json":
			return getConfigJSONFile(logger, override, configGetter)
		case ".yaml", ".yml":
			return getConfigYAMLFile(logger, override, configGetter)
		default:
			return getConfigJSONOrYAMLData(logger, override, configGetter)
		}
	}
	data, id, err := provider.GetConfigData(ctx, readBucket)
	if err != nil {
		return nil, err
	}
	return configGetter(
		logger,
		encoding.UnmarshalYAMLNonStrict,
		encoding.UnmarshalYAMLStrict,
		data,
		id,
	)
}

func getConfigJSONFile[V any](
	logger *zap.Logger,
	file string,
	configGetter ConfigGetter[V],
) (*V, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("could not read file %s: %v", file, err)
	}
	return configGetter(
		logger,
		encoding.UnmarshalJSONNonStrict,
		encoding.UnmarshalJSONStrict,
		data,
		file,
	)
}

func getConfigYAMLFile[V any](
	logger *zap.Logger,
	file string,
	configGetter ConfigGetter[V],
) (*V, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("could not read file %s: %v", file, err)
	}
	return configGetter(
		logger,
		encoding.UnmarshalYAMLNonStrict,
		encoding.UnmarshalYAMLStrict,
		data,
		file,
	)
}

func getConfigJSONOrYAMLData[V any](
	logger *zap.Logger,
	data string,
	configGetter ConfigGetter[V],
) (*V, error) {
	return configGetter(
		logger,
		encoding.UnmarshalJSONOrYAMLNonStrict,
		encoding.UnmarshalJSONOrYAMLStrict,
		[]byte(data),
		"Generate configuration data",
	)
}

func getConfigVersion(
	_ *zap.Logger,
	unmarshalNonStrict func([]byte, interface{}) error,
	_ func([]byte, interface{}) error,
	data []byte,
	_ string,
) (*string, error) {
	var externalConfigVersion ExternalConfigVersion
	if err := unmarshalNonStrict(data, &externalConfigVersion); err != nil {
		return nil, err
	}
	return &externalConfigVersion.Version, nil
}
