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
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/internal"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// CsharpNamespaceID is the ID of the csharp_namespace modifier.
const CsharpNamespaceID = "CSHARP_NAMESPACE"

func csharpNamespace(
	logger *zap.Logger,
	sweeper Sweeper,
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
	for moduleIdentity, csharpNamespace := range moduleOverrides {
		overrideModuleIdentityStrings[moduleIdentity.IdentityString()] = csharpNamespace
	}
	return ModifierFunc(
		func(ctx context.Context, image bufimage.Image) error {
			seenModuleIdentityStrings := make(map[string]struct{}, len(overrideModuleIdentityStrings))
			seenOverrideFiles := make(map[string]struct{}, len(overrides))
			for _, imageFile := range image.Files() {
				csharpNamespaceValue := csharpNamespaceValue(imageFile)
				if moduleIdentity := imageFile.ModuleIdentity(); moduleIdentity != nil {
					moduleIdentityString := moduleIdentity.IdentityString()
					if moduleNamespaceOverride, ok := overrideModuleIdentityStrings[moduleIdentityString]; ok {
						seenModuleIdentityStrings[moduleIdentityString] = struct{}{}
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
					exceptModuleIdentityStrings,
				); err != nil {
					return err
				}
			}
			for moduleIdentityString := range overrideModuleIdentityStrings {
				if _, ok := seenModuleIdentityStrings[moduleIdentityString]; !ok {
					logger.Sugar().Warnf("csharp_namespace_prefix override for %q was unused", moduleIdentityString)
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
	exceptModuleIdentityStrings map[string]struct{},
) error {
	if shouldSkipCsharpNamespaceForFile(ctx, imageFile, csharpNamespaceValue, exceptModuleIdentityStrings) {
		// This is a well-known type or we could not resolve a non-empty csharp_namespace
		// value, so this is a no-op.
		return nil
	}
	descriptor := imageFile.Proto()
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.CsharpNamespace = proto.String(csharpNamespaceValue)
	if sweeper != nil {
		sweeper.mark(imageFile.Path(), internal.CsharpNamespacePath)
	}
	return nil
}

func shouldSkipCsharpNamespaceForFile(
	ctx context.Context,
	imageFile bufimage.ImageFile,
	csharpNamespaceValue string,
	exceptModuleIdentityStrings map[string]struct{},
) bool {
	if internal.IsWellKnownType(ctx, imageFile) || csharpNamespaceValue == "" {
		// This is a well-known type or we could not resolve a non-empty csharp_namespace
		// value, so this is a no-op.
		return true
	}

	if moduleIdentity := imageFile.ModuleIdentity(); moduleIdentity != nil {
		if _, ok := exceptModuleIdentityStrings[moduleIdentity.IdentityString()]; ok {
			return true
		}
	}
	return false
}

// csharpNamespaceValue returns the csharp_namespace for the given ImageFile based on its
// package declaration. If the image file doesn't have a package declaration, an
// empty string is returned.
func csharpNamespaceValue(imageFile bufimage.ImageFile) string {
	pkg := imageFile.Proto().GetPackage()
	if pkg == "" {
		return ""
	}
	packageParts := strings.Split(pkg, ".")
	for i, part := range packageParts {
		packageParts[i] = stringutil.ToPascalCase(part)
	}
	return strings.Join(packageParts, ".")
}
