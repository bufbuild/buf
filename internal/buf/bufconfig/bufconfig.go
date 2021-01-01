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

// Package bufconfig contains the configuration functionality.
package bufconfig

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"

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
	ModuleIdentity bufmodule.ModuleIdentity
	Build          *bufmodulebuild.Config
	Breaking       *bufbreaking.Config
	Lint           *buflint.Config
}

// Provider is a provider.
type Provider interface {
	// GetConfig gets the Config for the YAML data at ConfigFilePath.
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

// CreateConfig writes an initial configuration file into the bucket.
func CreateConfig(ctx context.Context, writeBucket storage.WriteBucket, moduleIdentityString string, deps ...string) error {
	if _, err := bufmodule.ModuleIdentityForString(moduleIdentityString); err != nil {
		return err
	}
	data, err := encoding.MarshalYAML(
		externalConfigV1Beta1{
			Version: v1beta1Version,
			Name:    moduleIdentityString,
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

// ReadConfig reads the configuration, including potentially reading from the OS for an override.
//
// If no override is set, this reads ConfigFilePath in the bucket..
// If override is set, this will first check if the override ends in .json or .yaml, if so,
// this reads the file at this path and uses it. Otherwise, this assumes this is configuration
// data in either JSON or YAML format, and unmarshals it.
//
// Only use in CLI tools.
func ReadConfig(ctx context.Context, provider Provider, readBucket storage.ReadBucket, override string) (*Config, error) {
	if override != "" {
		var data []byte
		var err error
		switch filepath.Ext(override) {
		case ".json", ".yaml":
			data, err = ioutil.ReadFile(override)
			if err != nil {
				return nil, fmt.Errorf("could not read file: %v", err)
			}
		default:
			data = []byte(override)
		}
		return provider.GetConfigForData(ctx, data)
	}
	return provider.GetConfig(ctx, readBucket)
}

type externalConfigVersion struct {
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
}

type externalConfigV1Beta1 struct {
	Version  string                               `json:"version,omitempty" yaml:"version,omitempty"`
	Name     string                               `json:"name,omitempty" yaml:"name,omitempty"`
	Build    bufmodulebuild.ExternalConfigV1Beta1 `json:"build,omitempty" yaml:"build,omitempty"`
	Breaking bufbreaking.ExternalConfigV1Beta1    `json:"breaking,omitempty" yaml:"breaking,omitempty"`
	Lint     buflint.ExternalConfigV1Beta1        `json:"lint,omitempty" yaml:"lint,omitempty"`
	Deps     []string                             `json:"deps,omitempty" yaml:"deps,omitempty"`
}
