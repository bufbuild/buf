// Copyright 2020-2022 Buf Technologies, Inc.
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
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
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
	for moduleIdentity, csharpNamespacePrefix := range moduleOverrides {
		overrideModuleIdentityStrings[moduleIdentity.IdentityString()] = csharpNamespacePrefix
	}
	return ModifierFunc(
		func(ctx context.Context, image bufimage.Image) error {
			seenModuleIdentityStrings := make(map[string]struct{}, len(overrideModuleIdentityStrings))
			seenOverrideFiles := make(map[string]struct{}, len(overrides))
			for _, imageFile := range image.Files() {
				namespacePrefix := defaultPrefix
				if moduleIdentity := imageFile.ModuleIdentity(); moduleIdentity != nil {
					moduleIdentityString := moduleIdentity.IdentityString()
					if modulePrefixOverride, ok := overrideModuleIdentityStrings[moduleIdentityString]; ok {
						namespacePrefix = modulePrefixOverride
						seenModuleIdentityStrings[moduleIdentityString] = struct{}{}
					}
				}
				csharpNamespaceValue := csharpNamespaceValue(imageFile, namespacePrefix)
				if overrideValue, ok := overrides[imageFile.Path()]; ok {
					csharpNamespaceValue = overrideValue
					seenOverrideFiles[imageFile.Path()] = struct{}{}
				}
				if err := csharpNamespaceForFile(ctx, sweeper, imageFile, csharpNamespaceValue); err != nil {
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
) error {
	descriptor := imageFile.Proto()
	if isWellKnownType(ctx, imageFile) || csharpNamespaceValue == "" {
		// This is a well-known type or we could not resolve a non-empty csharp_namespace
		// value, so this is a no-op.
		return nil
	}
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.CsharpNamespace = proto.String(csharpNamespaceValue)
	if sweeper != nil {
		sweeper.mark(imageFile.Path(), csharpNamespacePath)
	}
	return nil
}

func csharpNamespaceValue(imageFile bufimage.ImageFile, namespacePrefix string) string {
	subNamespace := csharpSubNamespaceValue(imageFile)
	// TODO: check if prefix is trimmed before passed here
	if namespacePrefix == "" {
		return subNamespace
	}
	if subNamespace == "" {
		return namespacePrefix
	}
	return strings.Join(
		[]string{
			namespacePrefix,
			subNamespace,
		},
		".",
	)
}

// csharpNamespaceValue returns the csharp_namespace for the given ImageFile based on its
// package declaration. If the image file doesn't have a package declaration, an
// empty string is returned.
func csharpSubNamespaceValue(imageFile bufimage.ImageFile) string {
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
