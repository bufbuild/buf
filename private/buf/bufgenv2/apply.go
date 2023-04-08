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
)

// this is also called by override funcs
func applyManagementForFile(
	marker Marker,
	imageFile bufimage.ImageFile,
	managedConfig *ManagedConfig,
) error {
	for _, fileOption := range AllFileOptions {
		if managedConfig.DisabledFunc(fileOption, imageFile) {
			continue
		}
		var valueOrPrefixOverride string
		var err error
		overrideFunc, ok := managedConfig.FileOptionToOverrideFunc[fileOption]
		if ok {
			valueOrPrefixOverride, err = overrideFunc(imageFile)
			if err != nil {
				return err
			}
		}
		if err := applyManagementInternal(marker, imageFile, fileOption, valueOrPrefixOverride); err != nil {
			return err
		}
	}
	return nil
}

func applyManagementInternal(
	marker Marker,
	imageFile bufimage.ImageFile,
	fileOption FileOption,
	valueOrPrefixOverride string,
) error {
	// TODO do the rest
	switch fileOption {
	case FileOptionJavaPackage:
		return bufimagemodifyJavaPackage(marker, imageFile, valueOrPrefixOverride)
	default:
		return fmt.Errorf("unknown FileOption: %q", fileOption)
	}
}

func bufimagemodifyJavaPackage(marker Marker, imageFile bufimage.ImageFile, prefixOverride string) error {
	// TODO this would call into a refactored bufimagemodify
	//
	// we would have to figure out who has the responsibility to deal with defaults - is this
	// something that should be defined in bufimagemodify? think about core
	//
	// we need to make sure that bufimagemodify does marking because that logic should be shared
	// so core can use it
	//
	// marker/sweeper wouldn't be defined in bufgen, neither would FileOption*
	return nil
}
