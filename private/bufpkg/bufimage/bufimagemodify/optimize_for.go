// Copyright 2020-2024 Buf Technologies, Inc.
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
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
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
	except []bufmoduleref.ModuleIdentity,
	moduleOverrides map[bufmoduleref.ModuleIdentity]descriptorpb.FileOptions_OptimizeMode,
	overrides map[string]descriptorpb.FileOptions_OptimizeMode,
) Modifier {
	// Convert the bufmoduleref.ModuleIdentity types into
	// strings so that they're comparable.
	exceptModuleIdentityStrings := make(map[string]struct{}, len(except))
	for _, moduleIdentity := range except {
		exceptModuleIdentityStrings[moduleIdentity.IdentityString()] = struct{}{}
	}
	overrideModuleIdentityStrings := make(
		map[string]descriptorpb.FileOptions_OptimizeMode,
		len(moduleOverrides),
	)
	for moduleIdentity, optimizeFor := range moduleOverrides {
		overrideModuleIdentityStrings[moduleIdentity.IdentityString()] = optimizeFor
	}
	return ModifierFunc(
		func(ctx context.Context, image bufimage.Image) error {
			seenModuleIdentityStrings := make(map[string]struct{}, len(overrideModuleIdentityStrings))
			seenOverrideFiles := make(map[string]struct{}, len(overrides))
			for _, imageFile := range image.Files() {
				modifierValue := defaultOptimizeFor
				if moduleIdentity := imageFile.ModuleIdentity(); moduleIdentity != nil {
					moduleIdentityString := moduleIdentity.IdentityString()
					if optimizeForOverrdie, ok := overrideModuleIdentityStrings[moduleIdentityString]; ok {
						modifierValue = optimizeForOverrdie
						seenModuleIdentityStrings[moduleIdentityString] = struct{}{}
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
					exceptModuleIdentityStrings,
				); err != nil {
					return err
				}
			}
			for moduleIdentityString := range overrideModuleIdentityStrings {
				if _, ok := seenModuleIdentityStrings[moduleIdentityString]; !ok {
					logger.Sugar().Warnf("optimize_for override for %q was unused", moduleIdentityString)
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
	exceptModuleIdentityStrings map[string]struct{},
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
	if moduleIdentity := imageFile.ModuleIdentity(); moduleIdentity != nil {
		if _, ok := exceptModuleIdentityStrings[moduleIdentity.IdentityString()]; ok {
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
