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

package bufwork

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sort"

	"github.com/bufbuild/buf/internal/pkg/encoding"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/stringutil"
	"go.opencensus.io/trace"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

const v1beta1Version = "v1beta1"

type provider struct {
	logger *zap.Logger
}

func newProvider(logger *zap.Logger) *provider {
	return &provider{
		logger: logger,
	}
}

func (p *provider) GetConfig(ctx context.Context, readBucket storage.ReadBucket, relativeRootPath string) (_ *Config, retErr error) {
	ctx, span := trace.StartSpan(ctx, "get_config")
	defer span.End()

	readObjectCloser, err := readBucket.Get(ctx, ExternalConfigV1Beta1FilePath)
	if err != nil {
		if storage.IsNotExist(err) {
			return p.newConfigV1Beta1(externalConfigV1Beta1{}, "default configuration")
		}
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, readObjectCloser.Close())
	}()
	data, err := io.ReadAll(readObjectCloser)
	if err != nil {
		return nil, err
	}
	return p.getConfigForData(
		ctx,
		encoding.UnmarshalYAMLNonStrict,
		encoding.UnmarshalYAMLStrict,
		filepath.Join(normalpath.Unnormalize(relativeRootPath), ExternalConfigV1Beta1FilePath),
		data,
		`File "`+readObjectCloser.ExternalPath()+`"`,
	)
}

func (p *provider) GetConfigForData(ctx context.Context, data []byte) (*Config, error) {
	_, span := trace.StartSpan(ctx, "get_config_for_data")
	defer span.End()
	return p.getConfigForData(
		ctx,
		encoding.UnmarshalJSONOrYAMLNonStrict,
		encoding.UnmarshalJSONOrYAMLStrict,
		"configuration data",
		data,
		"Configuration data",
	)
}

func (p *provider) getConfigForData(
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
	if err := p.validateExternalConfigVersion(externalConfigVersion, id); err != nil {
		return nil, err
	}
	var externalConfigV1Beta1 externalConfigV1Beta1
	if err := unmarshalStrict(data, &externalConfigV1Beta1); err != nil {
		return nil, err
	}
	return p.newConfigV1Beta1(externalConfigV1Beta1, workspaceID)
}

func (p *provider) newConfigV1Beta1(externalConfig externalConfigV1Beta1, workspaceID string) (*Config, error) {
	directorySet := make(map[string]struct{}, len(externalConfig.Directories))
	for _, directory := range externalConfig.Directories {
		if filepath.IsAbs(directory) {
			return nil, fmt.Errorf(
				"module %q listed in %s must be a relative path",
				normalpath.Unnormalize(directory),
				workspaceID,
			)
		}
		normalizedDirectory, err := normalpath.NormalizeAndValidate(directory)
		if err != nil {
			return nil, err
		}
		if _, ok := directorySet[normalizedDirectory]; ok {
			return nil, fmt.Errorf(
				"module %q is listed more than once in %s",
				normalpath.Unnormalize(normalizedDirectory),
				workspaceID,
			)
		}
		directorySet[normalizedDirectory] = struct{}{}
	}
	// It's very important that we sort the directories here so that the
	// constructed modules and/or images are in a deterministic order.
	directories := stringutil.MapToSlice(directorySet)
	sort.Slice(directories, func(i int, j int) bool {
		return directories[i] < directories[j]
	})
	if err := validateConfigurationOverlap(directories, workspaceID); err != nil {
		return nil, err
	}
	return &Config{
		Directories: directories,
	}, nil
}

// validateOverlap returns a non-nil error if any of the directories overlap
// with each other. The given directories are expected to be sorted.
func validateConfigurationOverlap(directories []string, workspaceID string) error {
	for i := 0; i < len(directories); i++ {
		for j := i + 1; j < len(directories); j++ {
			left := directories[i]
			right := directories[j]
			if normalpath.ContainsPath(left, right, normalpath.Relative) {
				return fmt.Errorf(
					"module %q contains module %q in %s; see %s for more details",
					normalpath.Unnormalize(left),
					normalpath.Unnormalize(right),
					workspaceID,
					faqPage,
				)
			}
			if normalpath.ContainsPath(right, left, normalpath.Relative) {
				return fmt.Errorf(
					"module %q contains module %q in %s; see %s for more details",
					normalpath.Unnormalize(right),
					normalpath.Unnormalize(left),
					workspaceID,
					faqPage,
				)
			}
		}
	}
	return nil
}

func (p *provider) validateExternalConfigVersion(externalConfigVersion externalConfigVersion, id string) error {
	switch externalConfigVersion.Version {
	case "":
		p.logger.Sugar().Warnf(`%s has no version set. Please add "version: %s". See https://docs.buf.build/faq for more details.`, id, v1beta1Version)
		return nil
	case v1beta1Version:
		return nil
	default:
		return fmt.Errorf("%s has unknown configuration version: %s", id, externalConfigVersion.Version)
	}
}
