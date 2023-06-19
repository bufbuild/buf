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

	"github.com/bufbuild/buf/private/buf/bufgen"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/bufimagemodifyv2"
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
	mergeDisabledFuncs(nil)
	mergeOverrideFuncs(nil)
	mergeFileOptionToOverrideFuncs(nil)
}

func readConfigV2(
	ctx context.Context,
	logger *zap.Logger,
	provider bufgen.ConfigDataProvider,
	readBucket storage.ReadBucket,
	options ...bufgen.ReadConfigOption,
) (*Config, error) {
	return bufgen.ReadFromConfig(
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
	managedConfig, err := newManagedConfig(externalConfigV2.Managed)
	if err != nil {
		return nil, err
	}
	config.Managed = managedConfig
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
	module := externalConfig.Module
	path := normalpath.Normalize(externalConfig.Path)
	// You could simplify this, but this helped me reason about it
	return func(fileOption FileOption, imageFile ImageFileIdentity) bool {
		// If we did not specify a file option, we match all file options
		return (selectorFileOption == 0 || fileOption == selectorFileOption) &&
			matchesPathAndModule(path, module, imageFile)
	}, nil
}

func newOverrideFunc(externalConfig ExternalManagedOverrideConfigV2) (OverrideFunc, error) {
	fileOption, err := ParseFileOption(externalConfig.FileOption)
	if err != nil {
		// This should never happen because we already validated that this is set and non-empty
		return nil, err
	}
	if externalConfig.Prefix != nil && externalConfig.Value != nil {
		return nil, errors.New("only one of value and prefix can be set")
	}
	if externalConfig.Prefix == nil && externalConfig.Value == nil {
		return nil, errors.New("one of value and prefix must be set")
	}
	var override bufimagemodifyv2.Override
	if externalConfig.Prefix != nil {
		if !fileOption.AllowPrefix() {
			return nil, fmt.Errorf("prefix is not allowed for %v", fileOption)
		}
		override = bufimagemodifyv2.NewPrefixOverride(*externalConfig.Prefix)
	} else if externalConfig.Value != nil {
		getOverride := fileOption.ValueOverrideGetter()
		if getOverride == nil {
			// this should not happen
			return nil, fmt.Errorf("unable to parse value for %v", fileOption)
		}
		override, err = getOverride(externalConfig.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid value for %v", fileOption)
		}
	}
	return func(imageFile ImageFileIdentity) bufimagemodifyv2.Override {
		// We don't need to match on FileOption - we only call this OverrideFunc when we
		// know we are applying for a given FileOption.
		// The FileOption we parsed above is assumed to be the FileOption.
		if !matchesPathAndModule(externalConfig.Path, externalConfig.Module, imageFile) {
			return nil
		}
		return override
	}, nil
}

func matchesPathAndModule(
	pathRequired string,
	moduleRequired string,
	imageFile ImageFileIdentity,
) bool {
	// If neither is required, it matches.
	if pathRequired == "" && moduleRequired == "" {
		return true
	}
	// If path is required, it must match on path.
	path := normalpath.Normalize(imageFile.Path())
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

func mergeFileOptionToOverrideFuncs(fileOptionToOverrideFuncs map[FileOption][]OverrideFunc) map[FileOption]OverrideFunc {
	fileOptionToOverrideFunc := make(map[FileOption]OverrideFunc, len(fileOptionToOverrideFuncs))
	for fileOption, overrideFuncs := range fileOptionToOverrideFuncs {
		fileOptionToOverrideFunc[fileOption] = mergeOverrideFuncs(overrideFuncs)
	}
	return fileOptionToOverrideFunc
}

func mergeDisabledFuncs(disabledFuncs []DisabledFunc) DisabledFunc {
	// If any disables, then we disable for this FileOption and ImageFile
	return func(fileOption FileOption, imageFile ImageFileIdentity) bool {
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
	return func(imageFile ImageFileIdentity) bufimagemodifyv2.Override {
		var override bufimagemodifyv2.Override
		for _, overrideFunc := range overrideFuncs {
			iOverride := overrideFunc(imageFile)
			// TODO: likely want something like *string or otherwise, see https://github.com/bufbuild/buf/issues/1949
			if iOverride != nil {
				override = iOverride
			}
		}
		return override
	}
}
