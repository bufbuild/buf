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

package bufimagemodifyv2

import (
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/internal"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

type Marker interface {
	Mark(bufimage.ImageFile, []int32)
}

type Sweeper interface {
	Sweep() error
}

type MarkSweeper interface {
	Marker
	Sweeper
}

func NewMarkSweeper(image bufimage.Image) MarkSweeper {
	return nil
}

func ModifyJavaPackage(
	marker Marker,
	imageFile bufimage.ImageFile,
	prefix string,
) error {
	descriptor := imageFile.Proto()
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.JavaPackage = proto.String(getJavaPackageValue(imageFile, prefix))
	marker.Mark(imageFile, internal.JavaPackagePath)
	return nil
}

func getJavaPackageValue(imageFile bufimage.ImageFile, prefix string) string {
	if pkg := imageFile.Proto().GetPackage(); pkg != "" {
		if prefix == "" {
			return pkg
		}
		return prefix + "." + pkg
	}
	return ""
}
