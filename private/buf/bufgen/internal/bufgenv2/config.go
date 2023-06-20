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

	"github.com/bufbuild/buf/private/buf/bufgen/internal"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"go.uber.org/zap"
)

// TODO: remove this
func SilenceLinter() {
	empty := ExternalConfigV2{}
	_, _ = newManagedConfig(empty.Managed)
	_ = validateExternalManagedConfigV2(empty.Managed)
	_, _ = newDisabledFunc(ExternalManagedDisableConfigV2{})
	_, _ = newOverrideFunc(ExternalManagedOverrideConfigV2{})
	matchesModule(nil, "")
	matchesPath(nil, "")
	mergeDisabledFuncs(nil)
	mergeOverrideFuncs(nil)
	mergeFileOptionToOverrideFuncs(nil)
}

func readConfigV2(
	ctx context.Context,
	logger *zap.Logger,
	provider internal.ConfigDataProvider,
	readBucket storage.ReadBucket,
	options ...internal.ReadConfigOption,
) (*Config, error) {
	return internal.ReadFromConfig(
		ctx,
		logger,
		provider,
		readBucket,
		getConfig,
		options...,
	)
}

func getConfig(
	ctx context.Context,
	logger *zap.Logger,
	_ func([]byte, interface{}) error,
	unmarshalStrict func([]byte, interface{}) error,
	data []byte,
	id string,
) (*Config, error) {
	var externalConfigV2 ExternalConfigV2
	if err := unmarshalStrict(data, &externalConfigV2); err != nil {
		return nil, err
	}
	if err := validateExternalConfigV2(externalConfigV2, id); err != nil {
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
	return &config, nil
}

func newManagedConfig(externalConfig ExternalManagedConfigV2) (*ManagedConfig, error) {
	if externalConfig.IsEmpty() {
		return nil, nil
	}
	if err := validateExternalManagedConfigV2(externalConfig); err != nil {
		return nil, err
	}
	var disabledFuncs []DisabledFunc
	fileOptionToOverrideFuncs := make(map[FileOption][]OverrideFunc)
	for _, externalDisableConfig := range externalConfig.Disable {
		disabledFunc, err := newDisabledFunc(externalDisableConfig)
		if err != nil {
			return nil, err
		}
		disabledFuncs = append(disabledFuncs, disabledFunc)
	}
	for _, externalOverrideConfig := range externalConfig.Override {
		fileOption, err := ParseFileOption(externalOverrideConfig.FileOption)
		if err != nil {
			// This should never happen because we already validated
			return nil, err
		}
		overrideFunc, err := newOverrideFunc(externalOverrideConfig)
		if err != nil {
			return nil, err
		}
		fileOptionToOverrideFuncs[fileOption] = append(
			fileOptionToOverrideFuncs[fileOption],
			overrideFunc,
		)
	}
	return &ManagedConfig{
		DisabledFunc:             mergeDisabledFuncs(disabledFuncs),
		FileOptionToOverrideFunc: mergeFileOptionToOverrideFuncs(fileOptionToOverrideFuncs),
	}, nil
}

func validateExternalConfigV2(externalConfig ExternalConfigV2, id string) error {
	// TODO: implement this
	return nil
}

func validateExternalManagedConfigV2(externalConfig ExternalManagedConfigV2) error {
	if externalConfig.IsEmpty() {
		return nil
	}
	for _, externalDisableConfig := range externalConfig.Disable {
		if len(externalDisableConfig.FileOption) == 0 && len(externalDisableConfig.Module) == 0 && len(externalDisableConfig.Path) == 0 {
			return errors.New("must set one of file_option, module, path for a disable rule")
		}
		// TODO
	}
	for _, externalOverrideConfig := range externalConfig.Override {
		// TODO
		_ = externalOverrideConfig
	}
	return nil
}

func newDisabledFunc(externalConfig ExternalManagedDisableConfigV2) (DisabledFunc, error) {
	var selectorFileOption FileOption
	var err error
	if len(externalConfig.FileOption) > 0 {
		selectorFileOption, err = ParseFileOption(externalConfig.FileOption)
		if err != nil {
			// This should never happen because we already validated
			return nil, err
		}
	}
	// You could simplify this, but this helped me reason about it
	return func(fileOption FileOption, imageFile bufimage.ImageFile) bool {
		// If we did not specify a file option, we match all file options
		return (selectorFileOption == 0 || fileOption == selectorFileOption) &&
			matchesModule(imageFile, externalConfig.Module) &&
			matchesPath(imageFile, externalConfig.Path)
	}, nil
}

func newOverrideFunc(externalConfig ExternalManagedOverrideConfigV2) (OverrideFunc, error) {
	fileOption, err := ParseFileOption(externalConfig.FileOption)
	if err != nil {
		// This should never happen because we already validated that this is set and non-empty
		return nil, err
	}
	return func(imageFile bufimage.ImageFile) (string, error) {
		// We don't need to match on FileOption - we only call this OverrideFunc when we
		// know we are applying for a given FileOption.
		// The FileOption we parsed above is assumed to be the FileOption.

		if !matchesModule(imageFile, externalConfig.Module) {
			return "", nil
		}
		if !matchesPath(imageFile, externalConfig.Path) {
			return "", nil
		}

		switch t := fileOption.Type(); t {
		case FileOptionTypeValue:
			return externalConfig.Value, nil
		case FileOptionTypePrefix:
			return externalConfig.Prefix, nil
		default:
			return "", fmt.Errorf("unknown FileOptionType: %q", t)
		}
	}, nil
}

// matchesModule returns true if the given external module config value matches the ImageFile.
//
// An empty value matches - this means we did not filter on modules.
func matchesModule(imageFile bufimage.ImageFile, module string) bool {
	// If we did not specify a module, we match all modules
	if len(module) == 0 {
		return true
	}
	// If we do not have a module, the module filter does nothing
	moduleIdentity := imageFile.ModuleIdentity()
	if moduleIdentity == nil {
		return true
	}
	return module == moduleIdentity.IdentityString()
}

// matchesPath returns true if the given external path config value matches the ImageFile.
//
// An empty value matches - this means we did not filter on modules.
func matchesPath(imageFile bufimage.ImageFile, path string) bool {
	// If we did not specify a path, we match all paths
	if len(path) == 0 {
		return true
	}
	return normalpath.EqualsOrContainsPath(path, imageFile.Path(), normalpath.Relative)
}

func mergeFileOptionToOverrideFuncs(fileOptionToOverrideFuncs map[FileOption][]OverrideFunc) map[FileOption]OverrideFunc {
	fileOptionToOverrideFunc := make(map[FileOption]OverrideFunc, len(fileOptionToOverrideFuncs))
	for fileOption, overrideFuncs := range fileOptionToOverrideFuncs {
		fileOptionToOverrideFunc[fileOption] = mergeOverrideFuncs(overrideFuncs)
	}
	return fileOptionToOverrideFunc
}

func mergeDisabledFuncs(disabledFuncs []DisabledFunc) DisabledFunc {
	// If any disables, then we disable for this FileOption and ImageFile
	return func(fileOption FileOption, imageFile bufimage.ImageFile) bool {
		for _, disabledFunc := range disabledFuncs {
			if disabledFunc(fileOption, imageFile) {
				return true
			}
		}
		return false
	}
}

func mergeOverrideFuncs(overrideFuncs []OverrideFunc) OverrideFunc {
	// Last override listed wins
	return func(imageFile bufimage.ImageFile) (string, error) {
		var override string
		for _, overrideFunc := range overrideFuncs {
			iOverride, err := overrideFunc(imageFile)
			if err != nil {
				return "", err
			}
			// TODO: likely want something like *string or otherwise, see https://github.com/bufbuild/buf/issues/1949
			if iOverride != "" {
				override = iOverride
			}
		}
		return override, nil
	}
}
