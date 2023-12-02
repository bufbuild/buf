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

package bufimagemodify

import (
	"context"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/internal"
	"github.com/bufbuild/buf/private/gen/data/datawkt"
)

// TODO: move this package into bufgen/internal

// Modify modifies the image according to the managed config.
func Modify(
	ctx context.Context,
	image bufimage.Image,
	config bufconfig.GenerateManagedConfig,
) error {
	if !config.Enabled() {
		return nil
	}
	sweeper := internal.NewMarkSweeper(image)
	for _, imageFile := range image.Files() {
		if datawkt.Exists(imageFile.Path()) {
			continue
		}
		modifyFuncs := []func(internal.MarkSweeper, bufimage.ImageFile, bufconfig.GenerateManagedConfig) error{
			modifyCcEnableArenas,
			modifyCsharpNamespace,
			modifyGoPackage,
			modifyJavaMultipleFiles,
			modifyJavaOuterClass,
			modifyJavaPackage,
			modifyJavaStringCheckUtf8,
			modifyObjcClassPrefix,
			modifyOptmizeFor,
			modifyPhpMetadataNamespace,
			modifyPhpNamespace,
			modifyRubyPackage,
			modifyJsType,
		}
		for _, modifyFunc := range modifyFuncs {
			if err := modifyFunc(sweeper, imageFile, config); err != nil {
				return err
			}
		}
	}
	return nil
}
