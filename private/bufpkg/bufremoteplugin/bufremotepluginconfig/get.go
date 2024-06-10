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

package bufremotepluginconfig

import (
	"context"
	"fmt"
	"io"

	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"go.uber.org/multierr"
)

func getConfigForBucket(ctx context.Context, readBucket storage.ReadBucket, options []ConfigOption) (_ *Config, retErr error) {
	// This will be in the order of precedence.
	var foundConfigFilePaths []string
	// Go through all valid config file paths and see which ones are present.
	// If none are present, return the default config.
	// If multiple are present, error.
	for _, configFilePath := range AllConfigFilePaths {
		exists, err := storage.Exists(ctx, readBucket, configFilePath)
		if err != nil {
			return nil, err
		}
		if exists {
			foundConfigFilePaths = append(foundConfigFilePaths, configFilePath)
		}
	}
	switch len(foundConfigFilePaths) {
	case 0:
		// Did not find anything, return the default.
		return newConfig(ExternalConfig{}, options)
	case 1:
		readObjectCloser, err := readBucket.Get(ctx, foundConfigFilePaths[0])
		if err != nil {
			return nil, err
		}
		defer func() {
			retErr = multierr.Append(retErr, readObjectCloser.Close())
		}()
		data, err := io.ReadAll(readObjectCloser)
		if err != nil {
			return nil, err
		}
		return getConfigForDataInternal(
			ctx,
			encoding.UnmarshalYAMLNonStrict,
			encoding.UnmarshalYAMLStrict,
			data,
			readObjectCloser.ExternalPath(),
			options,
		)
	default:
		return nil, fmt.Errorf("only one plugin file can exist but found multiple plugin files: %s", stringutil.SliceToString(foundConfigFilePaths))
	}
}

func getConfigForData(ctx context.Context, data []byte, options []ConfigOption) (_ *Config, retErr error) {
	return getConfigForDataInternal(
		ctx,
		encoding.UnmarshalJSONOrYAMLNonStrict,
		encoding.UnmarshalJSONOrYAMLStrict,
		data,
		"Plugin configuration data",
		options,
	)
}

func getConfigForDataInternal(
	ctx context.Context,
	unmarshalNonStrict func([]byte, interface{}) error,
	unmarshalStrict func([]byte, interface{}) error,
	data []byte,
	id string,
	options []ConfigOption,
) (*Config, error) {
	var externalConfigVersion externalConfigVersion
	if err := unmarshalNonStrict(data, &externalConfigVersion); err != nil {
		return nil, err
	}
	if err := validateExternalConfigVersion(externalConfigVersion, id); err != nil {
		return nil, err
	}
	var externalConfig ExternalConfig
	if err := unmarshalStrict(data, &externalConfig); err != nil {
		return nil, err
	}
	return newConfig(externalConfig, options)
}

func validateExternalConfigVersion(externalConfigVersion externalConfigVersion, id string) error {
	switch externalConfigVersion.Version {
	case "":
		return fmt.Errorf(
			`%s has no version set. Please add "version: %s"`,
			id,
			V1Version,
		)
	case V1Version:
		return nil
	default:
		return fmt.Errorf(
			`%s has an invalid "version: %s" set. Please add "version: %s"`,
			id,
			externalConfigVersion.Version,
			V1Version,
		)
	}
}
