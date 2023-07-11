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
	for _, fileOption := range allFileOptions {
		// disable has higher precedence
		if managedConfig.DisabledFunc(fileOption, imageFile) {
			continue
		}
		var (
			err           error
			override      bufimagemodifyv2.Override
			modifyOptions []bufimagemodifyv2.ModifyOption
		)
		overrideFunc, ok := managedConfig.FileOptionToOverrideFunc[fileOption]
		if ok {
			override = overrideFunc(imageFile)
			if err != nil {
				return err
			}
			overrideOption, err := bufimagemodifyv2.ModifyWithOverride(override)
			if err != nil {
				return err
			}
			modifyOptions = append(modifyOptions, overrideOption)
		}
		// TODO do the rest
		switch fileOption {
		case fileOptionJavaPackage:
			return bufimagemodifyv2.ModifyJavaPackage(marker, imageFile, modifyOptions...)
		default:
			return fmt.Errorf("unknown FileOption: %q", fileOption)
		}
	}
	return nil
}
