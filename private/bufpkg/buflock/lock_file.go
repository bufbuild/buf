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

package buflock

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/bufbuild/buf/private/pkg/storage"
	"go.uber.org/zap"
)

var deprecatedFormatToPrefix = map[string]string{
	"b1": "b1-",
	"b3": "b3-",
}

func readConfig(ctx context.Context, readBucket storage.ReadBucket) (_ *Config, retErr error) {
	configBytes, err := storage.ReadPath(ctx, readBucket, ExternalConfigFilePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// If the lock file doesn't exist, just return no dependencies.
			return &Config{}, nil
		}
		return nil, fmt.Errorf("failed to read lock file: %w", err)
	}
	var configVersion ExternalConfigVersion
	if err := encoding.UnmarshalYAMLNonStrict(configBytes, &configVersion); err != nil {
		return nil, fmt.Errorf("failed to decode lock file as YAML: %w", err)
	}
	switch configVersion.Version {
	case "", V1Beta1Version:
		var externalConfig ExternalConfigV1Beta1
		if err := encoding.UnmarshalYAMLStrict(configBytes, &externalConfig); err != nil {
			return nil, fmt.Errorf("failed to unmarshal lock file at %s: %w", V1Beta1Version, err)
		}
		config := &Config{}
		for _, dep := range externalConfig.Deps {
			config.Dependencies = append(config.Dependencies, DependencyForExternalConfigDependencyV1Beta1(dep))
		}
		return config, nil
	case V1Version:
		var externalConfig ExternalConfigV1
		if err := encoding.UnmarshalYAMLStrict(configBytes, &externalConfig); err != nil {
			return nil, fmt.Errorf("failed to unmarshal lock file at %s: %w", V1Version, err)
		}
		config := &Config{}
		for _, dep := range externalConfig.Deps {
			config.Dependencies = append(config.Dependencies, DependencyForExternalConfigDependencyV1(dep))
		}
		return config, nil
	default:
		return nil, fmt.Errorf("unknown lock file versions %q", configVersion.Version)
	}
}

func writeConfig(ctx context.Context, writeBucket storage.WriteBucket, config *Config) error {
	externalConfig := ExternalConfigV1{
		Version: V1Version,
		Deps:    make([]ExternalConfigDependencyV1, 0, len(config.Dependencies)),
	}
	for _, dep := range config.Dependencies {
		externalConfig.Deps = append(externalConfig.Deps, ExternalConfigDependencyV1ForDependency(dep))
	}
	configBytes, err := encoding.MarshalYAML(&externalConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal lock file: %w", err)
	}
	if err := storage.PutPath(
		ctx,
		writeBucket,
		ExternalConfigFilePath,
		append([]byte(Header), configBytes...),
	); err != nil {
		return fmt.Errorf("failed to write lock file: %w", err)
	}
	return nil
}

func checkDeprecatedDigests(
	ctx context.Context,
	logger *zap.Logger,
	readBucket storage.ReadBucket,
) error {
	configBytes, err := storage.ReadPath(ctx, readBucket, ExternalConfigFilePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("failed to read lock file: %w", err)
	}
	var configVersion ExternalConfigVersion
	if err := encoding.UnmarshalYAMLNonStrict(configBytes, &configVersion); err != nil {
		return fmt.Errorf("failed to decode lock file as YAML: %w", err)
	}
	if configVersion.Version != V1Version {
		return nil
	}
	var externalConfig ExternalConfigV1
	if err := encoding.UnmarshalYAMLStrict(configBytes, &externalConfig); err != nil {
		return fmt.Errorf("failed to unmarshal lock file at %s: %w", V1Version, err)
	}
	for _, dep := range externalConfig.Deps {
		for deprecatedFormat, prefix := range deprecatedFormatToPrefix {
			if strings.HasPrefix(dep.Digest, prefix) {
				logger.Sugar().Warnf(
					`found %s digest in buf.yaml, which will no longer be supported in a future version. Run "buf mod update" to update the lock file.`,
					deprecatedFormat,
				)
				return nil
			}
		}
	}
	return nil
}
