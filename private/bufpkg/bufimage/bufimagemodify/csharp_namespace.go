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

	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// CsharpNamespaceID is the ID of the csharp_namespace modifier.
const CsharpNamespaceID = "CSHARP_NAMESPACE"

// csharpNamespacePath is the SourceCodeInfo path for the csharp_namespace option.
// https://github.com/protocolbuffers/protobuf/blob/61689226c0e3ec88287eaed66164614d9c4f2bf7/src/google/protobuf/descriptor.proto#L428
var csharpNamespacePath = []int32{8, 37}

func csharpNamespace(
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
	for moduleFullName, csharpNamespace := range moduleOverrides {
		overrideModuleFullNameStrings[moduleFullName.String()] = csharpNamespace
	}
	return ModifierFunc(
		func(ctx context.Context, image bufimage.Image) error {
			seenModuleFullNameStrings := make(map[string]struct{}, len(overrideModuleFullNameStrings))
			seenOverrideFiles := make(map[string]struct{}, len(overrides))
			for _, imageFile := range image.Files() {
				csharpNamespaceValue := csharpNamespaceValue(imageFile)
				if moduleFullName := imageFile.ModuleFullName(); moduleFullName != nil {
					moduleFullNameString := moduleFullName.String()
					if moduleNamespaceOverride, ok := overrideModuleFullNameStrings[moduleFullNameString]; ok {
						seenModuleFullNameStrings[moduleFullNameString] = struct{}{}
						csharpNamespaceValue = moduleNamespaceOverride
					}
				}
				if overrideValue, ok := overrides[imageFile.Path()]; ok {
					csharpNamespaceValue = overrideValue
					seenOverrideFiles[imageFile.Path()] = struct{}{}
				}
				if err := csharpNamespaceForFile(
					ctx,
					sweeper,
					imageFile,
					csharpNamespaceValue,
					exceptModuleFullNameStrings,
				); err != nil {
					return err
				}
			}
			for moduleFullNameString := range overrideModuleFullNameStrings {
				if _, ok := seenModuleFullNameStrings[moduleFullNameString]; !ok {
					logger.Sugar().Warnf("csharp_namespace_prefix override for %q was unused", moduleFullNameString)
				}
			}
			for overrideFile := range overrides {
				if _, ok := seenOverrideFiles[overrideFile]; !ok {
					logger.Sugar().Warnf("%s override for %q was unused", CsharpNamespaceID, overrideFile)
				}
			}
			return nil
		},
	)
}

func csharpNamespaceForFile(
	ctx context.Context,
	sweeper Sweeper,
	imageFile bufimage.ImageFile,
	csharpNamespaceValue string,
	exceptModuleFullNameStrings map[string]struct{},
) error {
	if shouldSkipCsharpNamespaceForFile(ctx, imageFile, csharpNamespaceValue, exceptModuleFullNameStrings) {
		// This is a well-known type or we could not resolve a non-empty csharp_namespace
		// value, so this is a no-op.
		return nil
	}
	descriptor := imageFile.FileDescriptorProto()
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.CsharpNamespace = proto.String(csharpNamespaceValue)
	if sweeper != nil {
		sweeper.mark(imageFile.Path(), csharpNamespacePath)
	}
	return nil
}

func shouldSkipCsharpNamespaceForFile(
	ctx context.Context,
	imageFile bufimage.ImageFile,
	csharpNamespaceValue string,
	exceptModuleFullNameStrings map[string]struct{},
) bool {
	if isWellKnownType(ctx, imageFile) || csharpNamespaceValue == "" {
		// This is a well-known type or we could not resolve a non-empty csharp_namespace
		// value, so this is a no-op.
		return true
	}

	if moduleFullName := imageFile.ModuleFullName(); moduleFullName != nil {
		if _, ok := exceptModuleFullNameStrings[moduleFullName.String()]; ok {
			return true
		}
	}
	return false
}

// csharpNamespaceValue returns the csharp_namespace for the given ImageFile based on its
// package declaration. If the image file doesn't have a package declaration, an
// empty string is returned.
func csharpNamespaceValue(imageFile bufimage.ImageFile) string {
	pkg := imageFile.FileDescriptorProto().GetPackage()
	if pkg == "" {
		return ""
	}
	packageParts := strings.Split(pkg, ".")
	for i, part := range packageParts {
		packageParts[i] = stringutil.ToPascalCase(part)
	}
	return strings.Join(packageParts, ".")
}
