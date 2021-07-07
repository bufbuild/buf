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

package bufconfig

import (
	"context"
	"fmt"
	"io"

	"github.com/bufbuild/buf/internal/buf/bufcheck/bufbreaking"
	"github.com/bufbuild/buf/internal/buf/bufcheck/buflint"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/internal/pkg/encoding"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"go.opencensus.io/trace"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type provider struct {
	logger *zap.Logger
}

func newProvider(logger *zap.Logger) *provider {
	return &provider{
		logger: logger,
	}
}

func (p *provider) GetConfig(ctx context.Context, readBucket storage.ReadBucket) (_ *Config, retErr error) {
	ctx, span := trace.StartSpan(ctx, "get_config")
	defer span.End()

	readObjectCloser, err := readBucket.Get(ctx, ExternalConfigFilePath)
	if err != nil {
		if !storage.IsNotExist(err) {
			return nil, err
		}
		// Look for old config file
		readObjectCloser, err = readBucket.Get(ctx, ExternalConfigV1Beta1FilePath)
		if err != nil {
			if storage.IsNotExist(err) {
				return p.newConfigV1(ExternalConfigV1{})
			}
			return nil, err
		}
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
		data,
		readObjectCloser.ExternalPath(),
	)
}

func (p *provider) GetConfigForData(ctx context.Context, data []byte) (*Config, error) {
	_, span := trace.StartSpan(ctx, "get_config_for_data")
	defer span.End()
	return p.getConfigForData(
		ctx,
		encoding.UnmarshalJSONOrYAMLNonStrict,
		encoding.UnmarshalJSONOrYAMLStrict,
		data,
		"Configuration data",
	)
}

func (p *provider) getConfigForData(
	ctx context.Context,
	unmarshalNonStrict func([]byte, interface{}) error,
	unmarshalStrict func([]byte, interface{}) error,
	data []byte,
	id string,
) (*Config, error) {
	var externalConfigVersion ExternalConfigVersion
	if err := unmarshalNonStrict(data, &externalConfigVersion); err != nil {
		return nil, err
	}
	switch externalConfigVersion.Version {
	case "":
		// If a version is not specified, use the default v1beta1 configuration
		// to maintain compatibility with previous releases.
		p.logger.Sugar().Warnf(`%s has no version set. Please add "version: %s". See https://docs.buf.build/faq for more details.`, id, V1Beta1Version)
		var externalConfigV1Beta1 ExternalConfigV1Beta1
		if err := unmarshalStrict(data, &externalConfigV1Beta1); err != nil {
			return nil, err
		}
		return p.newConfigV1Beta1(externalConfigV1Beta1)
	case V1Beta1Version:
		var externalConfigV1Beta1 ExternalConfigV1Beta1
		if err := unmarshalStrict(data, &externalConfigV1Beta1); err != nil {
			return nil, err
		}
		return p.newConfigV1Beta1(externalConfigV1Beta1)
	case V1Version:
		var externalConfigV1 ExternalConfigV1
		if err := unmarshalStrict(data, &externalConfigV1); err != nil {
			return nil, err
		}
		return p.newConfigV1(externalConfigV1)
	default:
		return nil, fmt.Errorf(
			`%s has an invalid "version: %s" set. Please add "version: %s". See https://docs.buf.build/faq for more details`,
			id,
			externalConfigVersion.Version,
			V1Version,
		)
	}
}

func (p *provider) newConfigV1Beta1(externalConfig ExternalConfigV1Beta1) (*Config, error) {
	buildConfig, err := bufmodulebuild.NewConfigV1Beta1(externalConfig.Build, externalConfig.Deps...)
	if err != nil {
		return nil, err
	}
	breakingConfig, err := bufbreaking.NewConfigV1Beta1(externalConfig.Breaking)
	if err != nil {
		return nil, err
	}
	lintConfig, err := buflint.NewConfigV1Beta1(externalConfig.Lint)
	if err != nil {
		return nil, err
	}
	var moduleIdentity bufmodule.ModuleIdentity
	if externalConfig.Name != "" {
		moduleIdentity, err = bufmodule.ModuleIdentityForString(externalConfig.Name)
		if err != nil {
			return nil, err
		}
	}
	return &Config{
		Version:        V1Beta1Version,
		ModuleIdentity: moduleIdentity,
		Build:          buildConfig,
		Breaking:       breakingConfig,
		Lint:           lintConfig,
	}, nil
}

func (p *provider) newConfigV1(externalConfig ExternalConfigV1) (*Config, error) {
	buildConfig, err := bufmodulebuild.NewConfigV1(externalConfig.Build, externalConfig.Deps...)
	if err != nil {
		return nil, err
	}
	breakingConfig, err := bufbreaking.NewConfigV1(externalConfig.Breaking)
	if err != nil {
		return nil, err
	}
	lintConfig, err := buflint.NewConfigV1(externalConfig.Lint)
	if err != nil {
		return nil, err
	}
	var moduleIdentity bufmodule.ModuleIdentity
	if externalConfig.Name != "" {
		moduleIdentity, err = bufmodule.ModuleIdentityForString(externalConfig.Name)
		if err != nil {
			return nil, err
		}
	}
	return &Config{
		Version:        V1Version,
		ModuleIdentity: moduleIdentity,
		Build:          buildConfig,
		Breaking:       breakingConfig,
		Lint:           lintConfig,
	}, nil
}
