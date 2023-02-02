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

package bufwork

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/multierr"
)

func getConfigForBucket(ctx context.Context, readBucket storage.ReadBucket, relativeRootPath string) (_ *Config, retErr error) {
	ctx, span := otel.GetTracerProvider().Tracer("bufbuild/buf").Start(ctx, "get_workspace_config")
	defer span.End()
	defer func() {
		if retErr != nil {
			span.RecordError(retErr)
			span.SetStatus(codes.Error, retErr.Error())
		}
	}()

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
		return newConfigV1(ExternalConfigV1{}, "default configuration")
	case 1:
		workspaceID := filepath.Join(normalpath.Unnormalize(relativeRootPath), foundConfigFilePaths[0])
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
			workspaceID,
			data,
			readObjectCloser.ExternalPath(),
		)
	default:
		return nil, fmt.Errorf("only one workspace file can exist but found multiple workspace files: %s", stringutil.SliceToString(foundConfigFilePaths))
	}
}

func getConfigForData(ctx context.Context, data []byte) (*Config, error) {
	_, span := otel.GetTracerProvider().Tracer("bufbuild/buf").Start(ctx, "get_workspace_config_for_data")
	defer span.End()
	config, err := getConfigForDataInternal(
		ctx,
		encoding.UnmarshalJSONOrYAMLNonStrict,
		encoding.UnmarshalJSONOrYAMLStrict,
		"configuration data",
		data,
		"Configuration data",
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return config, err
}

func getConfigForDataInternal(
	ctx context.Context,
	unmarshalNonStrict func([]byte, interface{}) error,
	unmarshalStrict func([]byte, interface{}) error,
	workspaceID string,
	data []byte,
	id string,
) (*Config, error) {
	var externalConfigVersion externalConfigVersion
	if err := unmarshalNonStrict(data, &externalConfigVersion); err != nil {
		return nil, err
	}
	if err := validateExternalConfigVersion(externalConfigVersion, id); err != nil {
		return nil, err
	}
	var externalConfigV1 ExternalConfigV1
	if err := unmarshalStrict(data, &externalConfigV1); err != nil {
		return nil, err
	}
	return newConfigV1(externalConfigV1, workspaceID)
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
