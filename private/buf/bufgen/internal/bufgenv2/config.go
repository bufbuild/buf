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

package bufgenv2

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/buf/bufgen/internal"
	"github.com/bufbuild/buf/private/buf/bufgen/internal/bufgenplugin"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/bufimagemodifyv2"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/storage"
	"go.uber.org/zap"
)

// TODO: unexport these names and their fields

// Config is a configuration.
type Config struct {
	Managed *ManagedConfig
	Plugins []bufgenplugin.PluginConfig
	Inputs  []*InputConfig
}

// ManagedConfig is a managed mode configuration.
type ManagedConfig struct {
	Enabled                  bool
	DisabledFunc             disabledFunc
	FileOptionToOverrideFunc map[FileOption]overrideFunc
}

// disableFunc decides whether a file option should be disabled for a file.
type disabledFunc func(FileOption, imageFileIdentity) bool

// overrideFunc is specific to a file option, and returns what thie file option
// should be overridden to for this file.
type overrideFunc func(imageFileIdentity) bufimagemodifyv2.Override

// imageFileIdentity is an image file that can be identified by a path and module identity.
// There two (path and module) are the only information needed to decide whether to disable
// or override a file option for a specific file. Using an interface to for easier testing.
type imageFileIdentity interface {
	Path() string
	ModuleIdentity() bufmoduleref.ModuleIdentity
}

// InputConfig is an input configuration.
type InputConfig struct {
	InputRef     buffetch.Ref
	Types        []string
	ExcludePaths []string
	IncludePaths []string
}

// readConfigV2 reads V2 configuration.
func readConfigV2(
	ctx context.Context,
	logger *zap.Logger,
	readBucket storage.ReadBucket,
	options ...internal.ReadConfigOption,
) (*Config, error) {
	provider := internal.NewConfigDataProvider(logger)
	data, id, unmarshalNonStrict, unmarshalStrict, err := internal.ReadDataFromConfig(
		ctx,
		logger,
		provider,
		readBucket,
		options...,
	)
	if err != nil {
		return nil, err
	}
	var externalConfigVersion internal.ExternalConfigVersion
	if err := unmarshalNonStrict(data, &externalConfigVersion); err != nil {
		return nil, err
	}
	if externalConfigVersion.Version != internal.V2Version {
		return nil, fmt.Errorf(`%s has no version set. Please add "version: %s"`, id, internal.V2Version)
	}
	var externalConfigV2 ExternalConfigV2
	if err := unmarshalStrict(data, &externalConfigV2); err != nil {
		return nil, err
	}
	config := Config{}
	for _, externalInputConfig := range externalConfigV2.Inputs {
		inputConfig, err := newInputConfig(ctx, externalInputConfig)
		if err != nil {
			return nil, err
		}
		config.Inputs = append(config.Inputs, inputConfig)
	}
	pluginConfigs, err := newPluginConfigs(externalConfigV2.Plugins, id)
	if err != nil {
		return nil, err
	}
	config.Plugins = pluginConfigs
	managedConfig, err := newManagedConfig(logger, externalConfigV2.Managed)
	if err != nil {
		return nil, err
	}
	config.Managed = managedConfig
	return &config, nil
}
