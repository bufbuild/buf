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
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// RubyPackageID is the ID of the ruby_package modifier.
const RubyPackageID = "RUBY_PACKAGE"

// rubyPackagePath is the SourceCodeInfo path for the ruby_package option.
// https://github.com/protocolbuffers/protobuf/blob/61689226c0e3ec88287eaed66164614d9c4f2bf7/src/google/protobuf/descriptor.proto#L453
var rubyPackagePath = []int32{8, 45}

func rubyPackage(
	logger *zap.Logger,
	sweeper Sweeper,
	except []bufmodule.ModuleFullName,
	moduleOverrides map[bufmodule.ModuleFullName]string,
	overrides map[string]string,
) Modifier {
	// Convert the bufmodule.ModuleFullName types into
	// strings so that they're comparable.
	exceptModuleFullNameStrings := make(map[string]struct{}, len(except))
	for _, moduleFullName := range except {
		exceptModuleFullNameStrings[moduleFullName.String()] = struct{}{}
	}
	overrideModuleFullNameStrings := make(map[string]string, len(moduleOverrides))
	for moduleFullName, rubyPackage := range moduleOverrides {
		overrideModuleFullNameStrings[moduleFullName.String()] = rubyPackage
	}
	return ModifierFunc(
		func(ctx context.Context, image bufimage.Image) error {
			seenModuleFullNameStrings := make(map[string]struct{}, len(overrideModuleFullNameStrings))
			seenOverrideFiles := make(map[string]struct{}, len(overrides))
			for _, imageFile := range image.Files() {
				rubyPackageValue := rubyPackageValue(imageFile)
				if moduleFullName := imageFile.ModuleFullName(); moduleFullName != nil {
					moduleFullNameString := moduleFullName.String()
					if moduleNamespaceOverride, ok := overrideModuleFullNameStrings[moduleFullNameString]; ok {
						seenModuleFullNameStrings[moduleFullNameString] = struct{}{}
						rubyPackageValue = moduleNamespaceOverride
					}
				}
				if overrideValue, ok := overrides[imageFile.Path()]; ok {
					rubyPackageValue = overrideValue
					seenOverrideFiles[imageFile.Path()] = struct{}{}
				}
				if err := rubyPackageForFile(
					ctx,
					sweeper,
					imageFile,
					rubyPackageValue,
					exceptModuleFullNameStrings,
				); err != nil {
					return err
				}
			}
			for moduleFullNameString := range overrideModuleFullNameStrings {
				if _, ok := seenModuleFullNameStrings[moduleFullNameString]; !ok {
					logger.Sugar().Warnf("ruby_package override for %q was unused", moduleFullNameString)
				}
			}
			for overrideFile := range overrides {
				if _, ok := seenOverrideFiles[overrideFile]; !ok {
					logger.Sugar().Warnf("%s override for %q was unused", RubyPackageID, overrideFile)
				}
			}
			return nil
		},
	)
}

func rubyPackageForFile(
	ctx context.Context,
	sweeper Sweeper,
	imageFile bufimage.ImageFile,
	rubyPackageValue string,
	exceptModuleFullNameStrings map[string]struct{},
) error {
	descriptor := imageFile.FileDescriptorProto()
	if isWellKnownType(ctx, imageFile) || rubyPackageValue == "" {
		// This is a well-known type or we could not resolve a non-empty ruby_package
		// value, so this is a no-op.
		return nil
	}
	if moduleFullName := imageFile.ModuleFullName(); moduleFullName != nil {
		if _, ok := exceptModuleFullNameStrings[moduleFullName.String()]; ok {
			return nil
		}
	}
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.RubyPackage = proto.String(rubyPackageValue)
	if sweeper != nil {
		sweeper.mark(imageFile.Path(), rubyPackagePath)
	}
	return nil
}

// rubyPackageValue returns the ruby_package for the given ImageFile based on its
// package declaration. If the image file doesn't have a package declaration, an
// empty string is returned.
func rubyPackageValue(imageFile bufimage.ImageFile) string {
	pkg := imageFile.FileDescriptorProto().GetPackage()
	if pkg == "" {
		return ""
	}
	packageParts := strings.Split(pkg, ".")
	for i, part := range packageParts {
		packageParts[i] = stringutil.ToPascalCase(part)
	}
	return strings.Join(packageParts, "::")
}
