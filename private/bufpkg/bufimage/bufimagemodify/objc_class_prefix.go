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
	"strings"
	"unicode"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/protoversion"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// ObjcClassPrefixID is the ID of the objc_class_prefix modifier.
const ObjcClassPrefixID = "OBJC_CLASS_PREFIX"

// objcClassPrefixPath is the SourceCodeInfo path for the objc_class_prefix option.
// https://github.com/protocolbuffers/protobuf/blob/61689226c0e3ec88287eaed66164614d9c4f2bf7/src/google/protobuf/descriptor.proto#L425
var objcClassPrefixPath = []int32{8, 36}

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
				objcClassPrefixValue := objcClassPrefixValue(imageFile)
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
	descriptor := imageFile.FileDescriptorProto()
	if isWellKnownType(ctx, imageFile) || objcClassPrefixValue == "" {
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
		sweeper.mark(imageFile.Path(), objcClassPrefixPath)
	}
	return nil
}

// objcClassPrefixValue returns the objc_class_prefix for the given ImageFile based on its
// package declaration. If the image file doesn't have a package declaration, an
// empty string is returned.
func objcClassPrefixValue(imageFile bufimage.ImageFile) string {
	pkg := imageFile.FileDescriptorProto().GetPackage()
	if pkg == "" {
		return ""
	}
	_, hasPackageVersion := protoversion.NewPackageVersionForPackage(pkg)
	packageParts := strings.Split(pkg, ".")
	var prefixParts []rune
	for i, part := range packageParts {
		// Check if last part is a version before appending.
		if i == len(packageParts)-1 && hasPackageVersion {
			continue
		}
		// Probably should never be a non-ASCII character,
		// but why not support it just in case?
		runeSlice := []rune(part)
		prefixParts = append(prefixParts, unicode.ToUpper(runeSlice[0]))
	}
	for len(prefixParts) < 3 {
		prefixParts = append(prefixParts, 'X')
	}
	prefix := string(prefixParts)
	if prefix == "GPB" {
		prefix = "GPX"
	}
	return prefix
}
