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

package bufimagemodifyv1

import (
	"context"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/internal"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// ObjcClassPrefixID is the ID of the objc_class_prefix modifier.
const ObjcClassPrefixID = "OBJC_CLASS_PREFIX"

func objcClassPrefix(
	logger *zap.Logger,
	sweeper Sweeper,
	defaultPrefix string,
	except []bufmoduleref.ModuleIdentity,
	moduleOverrides map[bufmoduleref.ModuleIdentity]string,
	overrides map[string]string,
) Modifier {
	// Convert the bufmoduleref.ModuleIdentity types into
	// strings so that they're comparable.
	exceptModuleIdentityStrings := make(map[string]struct{}, len(except))
	for _, moduleIdentity := range except {
		exceptModuleIdentityStrings[moduleIdentity.IdentityString()] = struct{}{}
	}
	overrideModuleIdentityStrings := make(map[string]string, len(moduleOverrides))
	for moduleIdentity, goPackagePrefix := range moduleOverrides {
		overrideModuleIdentityStrings[moduleIdentity.IdentityString()] = goPackagePrefix
	}
	return ModifierFunc(
		func(ctx context.Context, image bufimage.Image) error {
			seenModuleIdentityStrings := make(map[string]struct{}, len(overrideModuleIdentityStrings))
			seenOverrideFiles := make(map[string]struct{}, len(overrides))
			for _, imageFile := range image.Files() {
				objcClassPrefixValue := internal.GetDefaultObjcClassPrefixValue(imageFile)
				if defaultPrefix != "" {
					objcClassPrefixValue = defaultPrefix
				}
				if moduleIdentity := imageFile.ModuleIdentity(); moduleIdentity != nil {
					moduleIdentityString := moduleIdentity.IdentityString()
					if modulePrefixOverride, ok := overrideModuleIdentityStrings[moduleIdentityString]; ok {
						objcClassPrefixValue = modulePrefixOverride
						seenModuleIdentityStrings[moduleIdentityString] = struct{}{}
					}
				}
				if overrideValue, ok := overrides[imageFile.Path()]; ok {
					objcClassPrefixValue = overrideValue
					seenOverrideFiles[imageFile.Path()] = struct{}{}
				}
				if err := objcClassPrefixForFile(ctx, sweeper, imageFile, objcClassPrefixValue, exceptModuleIdentityStrings); err != nil {
					return err
				}
			}
			for moduleIdentityString := range overrideModuleIdentityStrings {
				if _, ok := seenModuleIdentityStrings[moduleIdentityString]; !ok {
					logger.Sugar().Warnf("%s override for %q was unused", ObjcClassPrefixID, moduleIdentityString)
				}
			}
			for overrideFile := range overrides {
				if _, ok := seenOverrideFiles[overrideFile]; !ok {
					logger.Sugar().Warnf("%s override for %q was unused", ObjcClassPrefixID, overrideFile)
				}
			}
			return nil
		},
	)
}

func objcClassPrefixForFile(
	ctx context.Context,
	sweeper Sweeper,
	imageFile bufimage.ImageFile,
	objcClassPrefixValue string,
	exceptModuleIdentityStrings map[string]struct{},
) error {
	descriptor := imageFile.Proto()
	if internal.IsWellKnownType(imageFile) || objcClassPrefixValue == "" {
		// This is a well-known type or we could not resolve a non-empty objc_class_prefix
		// value, so this is a no-op.
		return nil
	}
	if moduleIdentity := imageFile.ModuleIdentity(); moduleIdentity != nil {
		if _, ok := exceptModuleIdentityStrings[moduleIdentity.IdentityString()]; ok {
			return nil
		}
	}
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.ObjcClassPrefix = proto.String(objcClassPrefixValue)
	if sweeper != nil {
		sweeper.mark(imageFile.Path(), internal.ObjcClassPrefixPath)
	}
	return nil
}
