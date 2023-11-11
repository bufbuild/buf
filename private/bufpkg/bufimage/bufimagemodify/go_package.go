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
	"fmt"

	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// GoPackageID is the ID of the go_package modifier.
const GoPackageID = "GO_PACKAGE"

// goPackagePath is the SourceCodeInfo path for the go_package option.
// https://github.com/protocolbuffers/protobuf/blob/ee04809540c098718121e092107fbc0abc231725/src/google/protobuf/descriptor.proto#L392
var goPackagePath = []int32{8, 11}

func goPackage(
	logger *zap.Logger,
	sweeper Sweeper,
	defaultImportPathPrefix string,
	except []bufmodule.ModuleFullName,
	moduleOverrides map[bufmodule.ModuleFullName]string,
	overrides map[string]string,
) (Modifier, error) {
	if defaultImportPathPrefix == "" {
		return nil, fmt.Errorf("a non-empty import path prefix is required")
	}
	// Convert the bufmodule.ModuleFullName types into
	// strings so that they're comparable.
	exceptModuleFullNameStrings := make(map[string]struct{}, len(except))
	for _, moduleFullName := range except {
		exceptModuleFullNameStrings[moduleFullName.String()] = struct{}{}
	}
	overrideModuleFullNameStrings := make(map[string]string, len(moduleOverrides))
	for moduleFullName, goPackagePrefix := range moduleOverrides {
		overrideModuleFullNameStrings[moduleFullName.String()] = goPackagePrefix
	}
	seenModuleFullNameStrings := make(map[string]struct{}, len(overrideModuleFullNameStrings))
	seenOverrideFiles := make(map[string]struct{}, len(overrides))
	return ModifierFunc(
		func(ctx context.Context, image bufimage.Image) error {
			for _, imageFile := range image.Files() {
				importPathPrefix := defaultImportPathPrefix
				if moduleFullName := imageFile.ModuleFullName(); moduleFullName != nil {
					moduleFullNameString := moduleFullName.String()
					if modulePrefixOverride, ok := overrideModuleFullNameStrings[moduleFullNameString]; ok {
						importPathPrefix = modulePrefixOverride
						seenModuleFullNameStrings[moduleFullNameString] = struct{}{}
					}
				}
				goPackageValue := GoPackageImportPathForFile(imageFile, importPathPrefix)
				if overrideValue, ok := overrides[imageFile.Path()]; ok {
					goPackageValue = overrideValue
					seenOverrideFiles[imageFile.Path()] = struct{}{}
				}
				if err := goPackageForFile(
					ctx,
					sweeper,
					imageFile,
					goPackageValue,
					exceptModuleFullNameStrings,
				); err != nil {
					return err
				}
			}
			for moduleFullNameString := range overrideModuleFullNameStrings {
				if _, ok := seenModuleFullNameStrings[moduleFullNameString]; !ok {
					logger.Sugar().Warnf("go_package_prefix override for %q was unused", moduleFullNameString)
				}
			}
			for overrideFile := range overrides {
				if _, ok := seenOverrideFiles[overrideFile]; !ok {
					logger.Sugar().Warnf("%s override for %q was unused", GoPackageID, overrideFile)
				}
			}
			return nil
		},
	), nil
}

func goPackageForFile(
	ctx context.Context,
	sweeper Sweeper,
	imageFile bufimage.ImageFile,
	goPackageValue string,
	exceptModuleFullNameStrings map[string]struct{},
) error {
	if shouldSkipGoPackageForFile(ctx, imageFile, exceptModuleFullNameStrings) {
		return nil
	}
	descriptor := imageFile.FileDescriptorProto()
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.GoPackage = proto.String(goPackageValue)
	if sweeper != nil {
		sweeper.mark(imageFile.Path(), goPackagePath)
	}
	return nil
}

func shouldSkipGoPackageForFile(
	ctx context.Context,
	imageFile bufimage.ImageFile,
	exceptModuleFullNameStrings map[string]struct{},
) bool {
	if isWellKnownType(ctx, imageFile) && imageFile.FileDescriptorProto().GetOptions().GetGoPackage() != "" {
		// The well-known type defines the go_package option, so this is a no-op.
		// If a well-known type ever omits the go_package option, we make sure
		// to include it.
		return true
	}
	if moduleFullName := imageFile.ModuleFullName(); moduleFullName != nil {
		if _, ok := exceptModuleFullNameStrings[moduleFullName.String()]; ok {
			return true
		}
	}
	return false
}
