// Copyright 2020 Buf Technologies, Inc.
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

// Package bufconfig contains the configuration functionality.
package bufconfig

import (
	"context"

	"github.com/bufbuild/buf/internal/buf/bufcheck/bufbreaking"
	"github.com/bufbuild/buf/internal/buf/bufcheck/buflint"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"go.uber.org/zap"
)

// Config is the user config.
type Config struct {
	Name     bufmodule.ModuleName
	Build    *bufmodulebuild.Config
	Breaking *bufbreaking.Config
	Lint     *buflint.Config
}

// Provider is a provider.
type Provider interface {
	// GetConfig gets the Config for the given JSON or YAML data.
	//
	// If the data is of length 0, returns the default config.
	GetConfig(ctx context.Context, readBucket storage.ReadBucket) (*Config, error)
	// GetConfig gets the Config for the given JSON or YAML data.
	//
	// If the data is of length 0, returns the default config.
	GetConfigForData(ctx context.Context, data []byte) (*Config, error)
}

// NewProvider returns a new Provider.
func NewProvider(logger *zap.Logger, options ...ProviderOption) Provider {
	return newProvider(logger, options...)
}

// ProviderOption is an option for a new Provider.
type ProviderOption func(*provider)

// ProviderWithExternalConfigModifier returns a new ProviderOption that applies the following
// external config modifier before processing an ExternalConfig.
//
// Useful for testing.
func ProviderWithExternalConfigModifier(externalConfigModifier func(*ExternalConfig) error) ProviderOption {
	return func(provider *provider) {
		provider.externalConfigModifier = externalConfigModifier
	}
}

// ExternalConfig is an external config.
type ExternalConfig struct {
	Name     string                        `json:"name,omitempty" yaml:"name,omitempty"`
	Build    bufmodulebuild.ExternalConfig `json:"build,omitempty" yaml:"build,omitempty"`
	Breaking bufbreaking.ExternalConfig    `json:"breaking,omitempty" yaml:"breaking,omitempty"`
	Lint     buflint.ExternalConfig        `json:"lint,omitempty" yaml:"lint,omitempty"`
	Deps     []string                      `json:"deps,omitempty" yaml:"deps,omitempty"`
}
