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
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodifyv2"
)

// TODO: move these two functions to a separate
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
		var valueOrPrefix string
		var err error
		overrideFunc, ok := managedConfig.FileOptionToOverrideFunc[fileOption]
		if ok {
			valueOrPrefix, err = overrideFunc(imageFile)
			if err != nil {
				return err
			}
		}
		// TODO do the rest
		switch fileOption {
		case FileOptionJavaPackage:
			// Will need to do *string or similar for unset
			if valueOrPrefix == "" {
				valueOrPrefix = defaultJavaPackagePrefix
			}
			if err := bufimagemodifyv2.ModifyJavaPackage(marker, imageFile, valueOrPrefix); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown FileOption: %q", fileOption)
		}
	}
	return nil
}
