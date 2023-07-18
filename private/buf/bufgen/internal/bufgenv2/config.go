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
	"errors"
	"fmt"

	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/buf/bufgen/internal"
	"github.com/bufbuild/buf/private/buf/bufgen/internal/bufgenplugin"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/bufimagemodifyv2"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"go.uber.org/zap"
)

// disableFunc decides whether a file option should be disabled for a file.
type disabledFunc func(fileOption, imageFileIdentity) bool

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

// TODO: unexport these names

// Config is a configuration.
type Config struct {
	Managed *ManagedConfig
	Plugins []bufgenplugin.PluginConfig
	Inputs  []*InputConfig
}

// ManagedConfig is a managed mode configuration.
type ManagedConfig struct {
	Enabled                       bool
	DisabledFunc                  disabledFunc
	FileOptionGroupToOverrideFunc map[fileOptionGroup]overrideFunc
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

func newManagedConfig(logger *zap.Logger, externalConfig ExternalManagedConfigV2) (*ManagedConfig, error) {
	if externalConfig.isEmpty() {
		return nil, nil
	}
	if !externalConfig.Enabled && !externalConfig.isEmpty() {
		logger.Sugar().Warn("managed mode options are set but are not enabled")
		// continue to validate this config
	}
	var disabledFuncs []disabledFunc
	fileOptionGroupToOverrideFuncs := make(map[fileOptionGroup][]overrideFunc)
	for _, externalDisableConfig := range externalConfig.Disable {
		if len(externalDisableConfig.FileOption) == 0 && len(externalDisableConfig.Module) == 0 && len(externalDisableConfig.Path) == 0 {
			return nil, errors.New("must set one of file_option, module and path for a disable rule")
		}
		disabledFunc, err := newDisabledFunc(externalDisableConfig)
		if err != nil {
			return nil, err
		}
		disabledFuncs = append(disabledFuncs, disabledFunc)
	}
	for _, externalOverrideConfig := range externalConfig.Override {
		if len(externalOverrideConfig.Option) == 0 {
			return nil, errors.New("must set a file option to override")
		}
		if externalOverrideConfig.Value == nil {
			return nil, errors.New("must set an value to override")
		}
		fileOption, err := parseFileOption(externalOverrideConfig.Option)
		if err != nil {
			// This should never happen because we already validated
			return nil, err
		}
		fileOptionGroup, ok := fileOptionToGroup[fileOption]
		if !ok {
			// this should not happen, the map should cover all valid file options.
			return nil, err
		}
		overrideFunc, err := newOverrideFunc(externalOverrideConfig)
		if err != nil {
			return nil, err
		}
		// Putting rules from the same group in the same list preserves the order among them.
		// An example where this is useful:
		// override: ## values omitted
		//   - file_option: java_package
		//   - file_option: java_package_prefix
		// and
		// override:
		//   - file_option: java_package_prefix
		//   - file_option: java_package
		// have different effects.
		fileOptionGroupToOverrideFuncs[fileOptionGroup] = append(
			fileOptionGroupToOverrideFuncs[fileOptionGroup],
			overrideFunc,
		)
	}
	return &ManagedConfig{
		Enabled:                       externalConfig.Enabled,
		DisabledFunc:                  mergeDisabledFuncs(disabledFuncs),
		FileOptionGroupToOverrideFunc: mergeFileOptionToOverrideFuncs(fileOptionGroupToOverrideFuncs),
	}, nil
}

func newDisabledFunc(externalConfig ExternalManagedDisableConfigV2) (disabledFunc, error) {
	if len(externalConfig.FileOption) == 0 && len(externalConfig.Module) == 0 && len(externalConfig.Path) == 0 {
		return nil, errors.New("must set one of file_option, module and path for a disable rule")
	}
	var selectorFileOption fileOption
	var err error
	if len(externalConfig.FileOption) > 0 {
		selectorFileOption, err = parseFileOption(externalConfig.FileOption)
		if err != nil {
			return nil, err
		}
	}
	return func(fileOption fileOption, imageFile imageFileIdentity) bool {
		// If we did not specify a file option, we match all file options
		return (selectorFileOption == 0 || fileOption == selectorFileOption) &&
			matchesPathAndModule(externalConfig.Path, externalConfig.Module, imageFile)
	}, nil
}

func newOverrideFunc(externalConfig ExternalManagedOverrideConfigV2) (overrideFunc, error) {
	fileOption, err := parseFileOption(externalConfig.Option)
	if err != nil {
		// This should never happen because we already validated
		return nil, err
	}
	parseFunc, ok := fileOptionToOverrideParseFunc[fileOption]
	if !ok {
		// this should not happen
		return nil, fmt.Errorf("invalid file option: %v", fileOption)
	}
	override, err := parseFunc(externalConfig.Value, fileOption)
	if err != nil {
		return nil, err
	}
	return func(imageFile imageFileIdentity) bufimagemodifyv2.Override {
		// We don't need to match on FileOption - we only call this OverrideFunc when we
		// know we are applying for a given FileOption.
		// The FileOption we parsed above is assumed to be the FileOption.
		if matchesPathAndModule(externalConfig.Path, externalConfig.Module, imageFile) {
			return override
		}
		return nil
	}, nil
}

func matchesPathAndModule(
	pathRequired string,
	moduleRequired string,
	imageFile imageFileIdentity,
) bool {
	// If neither is required, it matches.
	if pathRequired == "" && moduleRequired == "" {
		return true
	}
	// If path is required, it must match on path.
	path := normalpath.Normalize(imageFile.Path())
	pathRequired = normalpath.Normalize(pathRequired)
	if pathRequired != "" && !normalpath.EqualsOrContainsPath(pathRequired, path, normalpath.Relative) {
		return false
	}
	// At this point, path requirement is met. If module is not required, it matches.
	if moduleRequired == "" {
		return true
	}
	// Module is required, now check if it matches.
	if imageFile.ModuleIdentity() == nil {
		return false
	}
	if imageFile.ModuleIdentity().IdentityString() != moduleRequired {
		return false
	}
	return true
}

func mergeFileOptionToOverrideFuncs(fileOptionGroupToOverrideFuncs map[fileOptionGroup][]overrideFunc) map[fileOptionGroup]overrideFunc {
	fileOptionToOverrideFunc := make(map[fileOptionGroup]overrideFunc, len(fileOptionGroupToOverrideFuncs))
	for fileOptionGroup, overrideFuncs := range fileOptionGroupToOverrideFuncs {
		fileOptionToOverrideFunc[fileOptionGroup] = mergeOverrideFuncs(overrideFuncs)
	}
	return fileOptionToOverrideFunc
}

func mergeDisabledFuncs(disabledFuncs []disabledFunc) disabledFunc {
	// If any disables, then we disable for this fileOption and ImageFile
	return func(fileOption fileOption, imageFile imageFileIdentity) bool {
		for _, disabledFunc := range disabledFuncs {
			if disabledFunc(fileOption, imageFile) {
				return true
			}
		}
		return false
	}
}

func mergeOverrideFuncs(overrideFuncs []overrideFunc) overrideFunc {
	// Last override listed wins, but if the last two are prefix and suffix, both win.
	return func(imageFile imageFileIdentity) bufimagemodifyv2.Override {
		var (
			secondLastOverride bufimagemodifyv2.Override
			lastOverride       bufimagemodifyv2.Override
		)
		for _, overrideFunc := range overrideFuncs {
			currentOverride := overrideFunc(imageFile)
			if currentOverride != nil {
				secondLastOverride = lastOverride
				lastOverride = currentOverride
			}
		}
		if prefixOverride, ok := secondLastOverride.(bufimagemodifyv2.PrefixOverride); ok {
			if suffixOverride, ok := lastOverride.(bufimagemodifyv2.SuffixOverride); ok {
				return bufimagemodifyv2.NewPrefxiSuffixOverride(
					prefixOverride.Get(),
					suffixOverride.Get(),
				)
			}
		}
		if suffixOverride, ok := secondLastOverride.(bufimagemodifyv2.SuffixOverride); ok {
			if prefixOverride, ok := lastOverride.(bufimagemodifyv2.PrefixOverride); ok {
				return bufimagemodifyv2.NewPrefxiSuffixOverride(
					prefixOverride.Get(),
					suffixOverride.Get(),
				)
			}
		}
		return lastOverride
	}
}
