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
	"fmt"
	"strings"

	"github.com/bufbuild/buf/internal/buf/bufcore/bufimage"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/protoversion"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// goPackagePath is the SourceCodeInfo path for the go_package option.
// https://github.com/protocolbuffers/protobuf/blob/ee04809540c098718121e092107fbc0abc231725/src/google/protobuf/descriptor.proto#L392
var goPackagePath = []int32{8, 11}

func goPackage(
	sweeper Sweeper,
	importPathPrefix string,
) (Modifier, error) {
	if importPathPrefix == "" {
		return nil, fmt.Errorf("a non-empty import path prefix is required")
	}
	return ModifierFunc(
		func(ctx context.Context, image bufimage.Image) error {
			for _, imageFile := range image.Files() {
				if err := goPackageForFile(ctx, sweeper, imageFile, importPathPrefix); err != nil {
					return err
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
	importPathPrefix string,
) error {
	descriptor := imageFile.Proto()
	if isWellKnownType(ctx, imageFile) && descriptor.GetOptions().GetGoPackage() != "" {
		// The well-known type defines the go_package option, so this is a no-op.
		// If a well-known type ever omits the go_package option, we make sure
		// to include it.
		return nil
	}
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.GoPackage = proto.String(goPackageImportPathForFile(imageFile, importPathPrefix))
	if sweeper != nil {
		sweeper.mark(imageFile.Path(), goPackagePath)
	}
	return nil
}

// goPackageImportPathForFile returns the go_package import path for the given
// ImageFile. If the package contains a version suffix, and if there are more
// than two comonents, concatenate the final two components. Otherwise, we
// exclude the ';' separator and adopt the default behavior from the import path.
func goPackageImportPathForFile(imageFile bufimage.ImageFile, importPathPrefix string) string {
	goPackageImportPath := normalpath.Join(importPathPrefix, normalpath.Dir(imageFile.Path()))
	packageName := imageFile.Proto().GetPackage()
	if _, ok := protoversion.NewPackageVersionForPackage(packageName); ok {
		parts := strings.Split(packageName, ".")
		if len(parts) >= 2 {
			goPackageImportPath += ";" + parts[len(parts)-2] + parts[len(parts)-1]
		}
	}
	return goPackageImportPath
}
