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
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

const (
	// DefaultJavaMultipleFilesValue is the default value for the java_multiple_files modifier.
	DefaultJavaMultipleFilesValue = true

	// JavaMultipleFilesID is the ID of the java_multiple_files modifier.
	JavaMultipleFilesID = "JAVA_MULTIPLE_FILES"
)

// javaMultipleFilesPath is the SourceCodeInfo path for the java_multiple_files option.
// https://github.com/protocolbuffers/protobuf/blob/ee04809540c098718121e092107fbc0abc231725/src/google/protobuf/descriptor.proto#L364
var javaMultipleFilesPath = []int32{8, 10}

func javaMultipleFiles(
	logger *zap.Logger,
	sweeper Sweeper,
	value bool,
	overrides map[string]bool,
	preserveExistingValue bool,
) Modifier {
	return ModifierFunc(
		func(ctx context.Context, image bufimage.Image) error {
			seenOverrideFiles := make(map[string]struct{}, len(overrides))
			for _, imageFile := range image.Files() {
				modifierValue := value
				if overrideValue, ok := overrides[imageFile.Path()]; ok {
					modifierValue = overrideValue
					seenOverrideFiles[imageFile.Path()] = struct{}{}
				}
				if err := javaMultipleFilesForFile(ctx, sweeper, imageFile, modifierValue, preserveExistingValue); err != nil {
					return err
				}
			}
			for overrideFile := range overrides {
				if _, ok := seenOverrideFiles[overrideFile]; !ok {
					logger.Sugar().Warnf("%s override for %q was unused", JavaMultipleFilesID, overrideFile)
				}
			}
			return nil
		},
	)
}

func javaMultipleFilesForFile(
	ctx context.Context,
	sweeper Sweeper,
	imageFile bufimage.ImageFile,
	value bool,
	preserveExistingValue bool,
) error {
	descriptor := imageFile.FileDescriptorProto()
	options := descriptor.GetOptions()
	switch {
	case isWellKnownType(ctx, imageFile):
		// The file is a well-known type, don't do anything.
		return nil
	case options != nil:
		if preserveExistingValue && options.JavaMultipleFiles != nil {
			// This option is explicitly set to a value - preserve existing value.
			return nil
		}
		if options.GetJavaMultipleFiles() == value {
			// The option is already set to the same value, don't do anything.
			return nil
		}
	case descriptorpb.Default_FileOptions_JavaMultipleFiles == value:
		// The option is not set, but the value we want to set is the
		// same as the default, don't do anything.
		return nil
	}
	if options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.JavaMultipleFiles = proto.Bool(value)
	if sweeper != nil {
		sweeper.mark(imageFile.Path(), javaMultipleFilesPath)
	}
	return nil
}
