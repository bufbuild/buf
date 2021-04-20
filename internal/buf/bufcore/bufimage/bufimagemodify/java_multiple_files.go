// Copyright 2020-2021 Buf Technologies, Inc.
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

	"github.com/bufbuild/buf/internal/buf/bufcore/bufimage"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// javaMultipleFilesPath is the SourceCodeInfo path for the java_multiple_files option.
// https://github.com/protocolbuffers/protobuf/blob/ee04809540c098718121e092107fbc0abc231725/src/google/protobuf/descriptor.proto#L364
var javaMultipleFilesPath = []int32{8, 10}

func javaMultipleFiles(
	sweeper Sweeper,
	value bool,
) Modifier {
	return ModifierFunc(
		func(ctx context.Context, image bufimage.Image) error {
			for _, imageFile := range image.Files() {
				if err := javaMultipleFilesForFile(ctx, sweeper, imageFile, value); err != nil {
					return err
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
) error {
	descriptor := imageFile.Proto()
	if options := descriptor.GetOptions(); isWellKnownType(ctx, imageFile) || (options != nil && options.GetJavaMultipleFiles() == value) {
		// The file is a well-known type or already defines the java_multiple_files
		// option with the given value, so this is a no-op.
		return nil
	}
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.JavaMultipleFiles = proto.Bool(value)
	if sweeper != nil {
		sweeper.mark(imageFile.Path(), javaMultipleFilesPath)
	}
	return nil
}
