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
	"google.golang.org/protobuf/types/descriptorpb"
)

// optimizeFor is the SourceCodeInfo path for the optimize_for option.
// https://github.com/protocolbuffers/protobuf/blob/61689226c0e3ec88287eaed66164614d9c4f2bf7/src/google/protobuf/descriptor.proto#L385
var optimizeForPath = []int32{8, 9}

func optimizeFor(
	sweeper Sweeper,
	value descriptorpb.FileOptions_OptimizeMode,
) Modifier {
	return ModifierFunc(
		func(ctx context.Context, image bufimage.Image) error {
			for _, imageFile := range image.Files() {
				if err := optimizeForForFile(ctx, sweeper, imageFile, value); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

func optimizeForForFile(
	ctx context.Context,
	sweeper Sweeper,
	imageFile bufimage.ImageFile,
	value descriptorpb.FileOptions_OptimizeMode,
) error {
	descriptor := imageFile.Proto()
	if options := descriptor.GetOptions(); isWellKnownType(ctx, imageFile) || (options != nil && options.GetOptimizeFor() == value) {
		// The file is a well-known type or already defines the optimize_for
		// option with the given value, so this is a no-op.
		return nil
	}
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.OptimizeFor = &value
	if sweeper != nil {
		sweeper.mark(imageFile.Path(), optimizeForPath)
	}
	return nil
}
