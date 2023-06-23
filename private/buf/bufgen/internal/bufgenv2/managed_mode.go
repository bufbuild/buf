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
	"errors"
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/bufimagemodifyv2"
	"github.com/bufbuild/buf/private/pkg/normalpath"
)

// TODO this would be part of a runner or likewise
// this is just for demonstration of bringing the management stuff into one function
// applyManagement modifies an image based on managed mode configuration.
func applyManagement(image bufimage.Image, managedConfig *ManagedConfig) error {
	markSweeper := bufimagemodifyv2.NewMarkSweeper(image)
	for _, imageFile := range image.Files() {
		if err := applyManagementForFile(markSweeper, imageFile, managedConfig); err != nil {
			return err
		}
	}
	return markSweeper.Sweep()
}

func applyManagementForFile(
	marker bufimagemodifyv2.Marker,
	imageFile bufimage.ImageFile,
	managedConfig *ManagedConfig,
) error {
	for _, fileOption := range AllFileOptions {
		if managedConfig.DisabledFunc(fileOption, imageFile) {
			continue
		}
		var override bufimagemodifyv2.Override
		if overrideFunc, ok := managedConfig.FileOptionToOverrideFunc[fileOption]; ok {
			override = overrideFunc(imageFile)
		}
		// override can be nil at this point, in which case ModifyXYZ will either use
		// our default value for the option, or leave the option alone, implicitly
		// using Protobuf's default value.
		switch fileOption {
		case FileOptionJavaPackage:
			if err := bufimagemodifyv2.ModifyJavaPackage(marker, imageFile, override); err != nil {
				return err
			}
		// TODO: do the rest
		default:
			return fmt.Errorf("unknown FileOption: %q", fileOption)
		}
	}
	return nil
}

func newManagedConfig(externalConfig ExternalManagedConfigV2) (*ManagedConfig, error) {
	if externalConfig.isEmpty() {
		return nil, nil
	}
	if err := validateExternalManagedConfigV2(externalConfig); err != nil {
		return nil, err
	}
	var disabledFuncs []disabledFunc
	fileOptionToOverrideFuncs := make(map[FileOption][]overrideFunc)
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

func newDisabledFunc(externalConfig ExternalManagedDisableConfigV2) (disabledFunc, error) {
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

func newOverrideFunc(externalConfig ExternalManagedOverrideConfigV2) (overrideFunc, error) {
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
	overrideParser, ok := fileOptionToParser[fileOption]
	if !ok {
		// this should not happen
		return nil, fmt.Errorf("unable to parse override for %v", fileOption)
	}
	override, err := overrideParser.parse(externalConfig.Prefix, externalConfig.Value, fileOption)
	if err != nil {
		return nil, err
	}
	return func(imageFile ImageFileIdentity) bufimagemodifyv2.Override {
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

func mergeFileOptionToOverrideFuncs(fileOptionToOverrideFuncs map[FileOption][]overrideFunc) map[FileOption]overrideFunc {
	fileOptionToOverrideFunc := make(map[FileOption]overrideFunc, len(fileOptionToOverrideFuncs))
	for fileOption, overrideFuncs := range fileOptionToOverrideFuncs {
		fileOptionToOverrideFunc[fileOption] = mergeOverrideFuncs(overrideFuncs)
	}
	return fileOptionToOverrideFunc
}

func mergeDisabledFuncs(disabledFuncs []disabledFunc) disabledFunc {
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

func mergeOverrideFuncs(overrideFuncs []overrideFunc) overrideFunc {
	// Last override listed wins
	return func(imageFile ImageFileIdentity) bufimagemodifyv2.Override {
		var override bufimagemodifyv2.Override
		for _, overrideFunc := range overrideFuncs {
			if iOverride := overrideFunc(imageFile); iOverride != nil {
				override = iOverride
			}
		}
		return override
	}
}
