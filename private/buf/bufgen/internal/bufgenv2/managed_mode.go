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
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/bufimagemodifyv2"
)

const (
	defaultJavaPackagePrefix = "com"
)

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
	for _, fileOptionGroup := range allFileOptionGroups {
		var override bufimagemodifyv2.Override
		overrideFunc, ok := managedConfig.FileOptionGroupToOverrideFunc[fileOptionGroup]
		if ok {
			override = overrideFunc(imageFile)
		}
		// TODO do the rest
		switch fileOptionGroup {
		case groupJavaPackage:
			if managedConfig.DisabledFunc(fileOptionJavaPackage, imageFile) {
				continue
			}
			override = addPrefixIfNotExist(override, defaultJavaPackagePrefix)
			if managedConfig.DisabledFunc(fileOptionJavaPackagePrefix, imageFile) {
				override = disablePrefix(override)
			}
			if managedConfig.DisabledFunc(fileOptionJavaPackageSuffix, imageFile) {
				override = disableSuffix(override)
			}
			modifyOptions, err := getModifyOptions(override)
			if err != nil {
				return err
			}
			err = bufimagemodifyv2.ModifyJavaPackage(marker, imageFile, modifyOptions...)
			if err != nil {
				return err
			}
		default:
			// this should not happen
			return fmt.Errorf("unknown file option")
		}
	}
	return nil
}

// disablePrefix returns an override that does the same thing as the override provided,
// except that the one returned does not modify prefix.
func disablePrefix(override bufimagemodifyv2.Override) bufimagemodifyv2.Override {
	switch t := override.(type) {
	case bufimagemodifyv2.PrefixOverride:
		return nil
	case bufimagemodifyv2.PrefixSuffixOverride:
		return bufimagemodifyv2.NewSuffixOverride(t.GetSuffix())
	}
	return override
}

// disableSuffix returns an override that does the same thing as the override provided,
// except that the one returned does not modify suffix.
func disableSuffix(override bufimagemodifyv2.Override) bufimagemodifyv2.Override {
	switch t := override.(type) {
	case bufimagemodifyv2.SuffixOverride:
		return nil
	case bufimagemodifyv2.PrefixSuffixOverride:
		return bufimagemodifyv2.NewPrefixOverride(t.GetPrefix())
	}
	return override
}

// addPrefixIfNotExist returns an override that does the same thing  as the override provided,
// except that the one returned also modifies prefix. If the override provided already modifies
// prefix, or if it modifies the value directly, the function returns the same override.
func addPrefixIfNotExist(override bufimagemodifyv2.Override, prefix string) bufimagemodifyv2.Override {
	switch t := override.(type) {
	case bufimagemodifyv2.SuffixOverride:
		return bufimagemodifyv2.NewPrefixSuffixOverride(prefix, t.Get())
	case nil:
		return bufimagemodifyv2.NewPrefixOverride(prefix)
	}
	return override
}

func getModifyOptions(override bufimagemodifyv2.Override) ([]bufimagemodifyv2.ModifyOption, error) {
	if override == nil {
		return nil, nil
	}
	option, err := bufimagemodifyv2.ModifyWithOverride(override)
	if err != nil {
		return nil, err
	}
	return []bufimagemodifyv2.ModifyOption{option}, nil
}
