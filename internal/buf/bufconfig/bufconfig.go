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

// Package bufconfig contains the configuration functionality.
package bufconfig

import (
	"context"

	"github.com/bufbuild/buf/internal/buf/bufbuild/bufbuildcfg"
	"github.com/bufbuild/buf/internal/buf/bufcheck/bufbreaking"
	"github.com/bufbuild/buf/internal/buf/bufcheck/bufbreaking/bufbreakingcfg"
	"github.com/bufbuild/buf/internal/buf/bufcheck/buflint"
	"github.com/bufbuild/buf/internal/buf/bufcheck/buflint/buflintcfg"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"go.uber.org/zap"
)

// ConfigFilePath is the default config file path within a bucket.
//
// TODO: make sure copied for git
const ConfigFilePath = "buf.yaml"

// Config is the user config.
type Config struct {
	Roots    []string
	Excludes []string
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
	GetConfigForData(data []byte) (*Config, error)
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

// NewProvider returns a new Provider.
func NewProvider(logger *zap.Logger, options ...ProviderOption) Provider {
	return newProvider(logger, options...)
}

// ExternalConfig is an external config.
type ExternalConfig struct {
	Build    bufbuildcfg.ExternalConfig    `json:"build,omitempty" yaml:"build,omitempty"`
	Breaking bufbreakingcfg.ExternalConfig `json:"breaking,omitempty" yaml:"breaking,omitempty"`
	Lint     buflintcfg.ExternalConfig     `json:"lint,omitempty" yaml:"lint,omitempty"`
}
