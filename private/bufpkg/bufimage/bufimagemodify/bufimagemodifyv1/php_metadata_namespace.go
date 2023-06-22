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
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// PhpMetadataNamespaceID is the ID of the php_metadata_namespace modifier.
const PhpMetadataNamespaceID = "PHP_METADATA_NAMESPACE"

func phpMetadataNamespace(
	logger *zap.Logger,
	sweeper Sweeper,
	overrides map[string]string,
) Modifier {
	return ModifierFunc(
		func(ctx context.Context, image bufimage.Image) error {
			seenOverrideFiles := make(map[string]struct{}, len(overrides))
			for _, imageFile := range image.Files() {
				phpMetadataNamespaceValue := phpMetadataNamespaceValue(imageFile)
				if overrideValue, ok := overrides[imageFile.Path()]; ok {
					phpMetadataNamespaceValue = overrideValue
					seenOverrideFiles[imageFile.Path()] = struct{}{}
				}
				if err := phpMetadataNamespaceForFile(ctx, sweeper, imageFile, phpMetadataNamespaceValue); err != nil {
					return err
				}
			}
			for overrideFile := range overrides {
				if _, ok := seenOverrideFiles[overrideFile]; !ok {
					logger.Sugar().Warnf("%s override for %q was unused", PhpMetadataNamespaceID, overrideFile)
				}
			}
			return nil
		},
	)
}

func phpMetadataNamespaceForFile(
	ctx context.Context,
	sweeper Sweeper,
	imageFile bufimage.ImageFile,
	phpMetadataNamespaceValue string,
) error {
	descriptor := imageFile.Proto()
	if internal.IsWellKnownType(ctx, imageFile) || phpMetadataNamespaceValue == "" {
		// This is a well-known type or we could not resolve a non-empty php_metadata_namespace
		// value, so this is a no-op.
		return nil
	}
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.PhpMetadataNamespace = proto.String(phpMetadataNamespaceValue)
	if sweeper != nil {
		sweeper.mark(imageFile.Path(), internal.PhpMetadataNamespacePath)
	}
	return nil
}

// phpMetadataNamespaceValue returns the php_metadata_namespace for the given ImageFile based on its
// package declaration. If the image file doesn't have a package declaration, an
// empty string is returned.
func phpMetadataNamespaceValue(imageFile bufimage.ImageFile) string {
	phpNamespace := phpNamespaceValue(imageFile)
	if phpNamespace == "" {
		return ""
	}
	return phpNamespace + `\GPBMetadata`
}
