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
	"github.com/bufbuild/buf/internal/pkg/encoding"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"go.uber.org/zap"
)

// ConfigFilePath is the configuration file path.
const ConfigFilePath = "buf.yaml"

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
func NewProvider(logger *zap.Logger) Provider {
	return newProvider(logger)
}

// ConfigCreate writes an initial configuration file into the bucket.
func ConfigCreate(ctx context.Context, writeBucket storage.WriteBucket, name string, deps ...string) error {
	data, err := encoding.MarshalYAML(
		externalConfigV1Beta1{
			Version: v1beta1Version,
			Name:    name,
			Deps:    deps,
		},
	)
	if err != nil {
		return err
	}
	return storage.PutPath(ctx, writeBucket, ConfigFilePath, data)
}

// ConfigExists checks if a configuration file exists.
func ConfigExists(ctx context.Context, readBucket storage.ReadBucket) (bool, error) {
	return storage.Exists(ctx, readBucket, ConfigFilePath)
}
