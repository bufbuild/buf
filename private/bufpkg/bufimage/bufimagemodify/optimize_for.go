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

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/descriptorpb"
)

// OptimizeForID is the ID for the optimize_for modifier.
const OptimizeForID = "OPTIMIZE_FOR"

// optimizeFor is the SourceCodeInfo path for the optimize_for option.
// https://github.com/protocolbuffers/protobuf/blob/61689226c0e3ec88287eaed66164614d9c4f2bf7/src/google/protobuf/descriptor.proto#L385
var optimizeForPath = []int32{8, 9}

func optimizeFor(
	logger *zap.Logger,
	sweeper Sweeper,
	defaultOptimizeFor descriptorpb.FileOptions_OptimizeMode,
	except []bufmodule.ModuleFullName,
	moduleOverrides map[bufmodule.ModuleFullName]descriptorpb.FileOptions_OptimizeMode,
	overrides map[string]descriptorpb.FileOptions_OptimizeMode,
) Modifier {
	// Convert the bufmodule.ModuleFullName types into
	// strings so that they're comparable.
	exceptModuleFullNameStrings := make(map[string]struct{}, len(except))
	for _, moduleFullName := range except {
		exceptModuleFullNameStrings[moduleFullName.String()] = struct{}{}
	}
	overrideModuleFullNameStrings := make(
		map[string]descriptorpb.FileOptions_OptimizeMode,
		len(moduleOverrides),
	)
	for moduleFullName, optimizeFor := range moduleOverrides {
		overrideModuleFullNameStrings[moduleFullName.String()] = optimizeFor
	}
	return ModifierFunc(
		func(ctx context.Context, image bufimage.Image) error {
			seenModuleFullNameStrings := make(map[string]struct{}, len(overrideModuleFullNameStrings))
			seenOverrideFiles := make(map[string]struct{}, len(overrides))
			for _, imageFile := range image.Files() {
				modifierValue := defaultOptimizeFor
				if moduleFullName := imageFile.ModuleFullName(); moduleFullName != nil {
					moduleFullNameString := moduleFullName.String()
					if optimizeForOverrdie, ok := overrideModuleFullNameStrings[moduleFullNameString]; ok {
						modifierValue = optimizeForOverrdie
						seenModuleFullNameStrings[moduleFullNameString] = struct{}{}
					}
				}
				if overrideValue, ok := overrides[imageFile.Path()]; ok {
					modifierValue = overrideValue
					seenOverrideFiles[imageFile.Path()] = struct{}{}
				}
				if err := optimizeForForFile(
					ctx,
					sweeper,
					imageFile,
					modifierValue,
					exceptModuleFullNameStrings,
				); err != nil {
					return err
				}
			}
			for moduleFullNameString := range overrideModuleFullNameStrings {
				if _, ok := seenModuleFullNameStrings[moduleFullNameString]; !ok {
					logger.Sugar().Warnf("optimize_for override for %q was unused", moduleFullNameString)
				}
			}
			for overrideFile := range overrides {
				if _, ok := seenOverrideFiles[overrideFile]; !ok {
					logger.Sugar().Warnf("%s override for %q was unused", OptimizeForID, overrideFile)
				}
			}
			return nil
		},
	)
}

func optimizeForForFile(
	ctx context.Context,
	sweeper Sweeper,
	imageFile bufimage.ImageFile,
	value descriptorpb.FileOptions_OptimizeMode,
	exceptModuleFullNameStrings map[string]struct{},
) error {
	descriptor := imageFile.FileDescriptorProto()
	options := descriptor.GetOptions()
	switch {
	case isWellKnownType(ctx, imageFile):
		// The file is a well-known type, don't do anything.
		return nil
	case options != nil && options.GetOptimizeFor() == value:
		// The option is already set to the same value, don't do anything.
		return nil
	case options == nil && descriptorpb.Default_FileOptions_OptimizeFor == value:
		// The option is not set, but the value we want to set is the
		// same as the default, don't do anything.
		return nil
	}
	if moduleFullName := imageFile.ModuleFullName(); moduleFullName != nil {
		if _, ok := exceptModuleFullNameStrings[moduleFullName.String()]; ok {
			return nil
		}
	}
	if options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.OptimizeFor = &value
	if sweeper != nil {
		sweeper.mark(imageFile.Path(), optimizeForPath)
	}
	return nil
}
