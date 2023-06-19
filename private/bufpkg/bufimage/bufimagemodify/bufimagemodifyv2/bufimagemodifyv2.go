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
	"errors"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

var (
	// TODO: double-check these
	javaPackagePath = []int32{8, 1}
	// javaOuterClassnamePath   = []int32{8, 8}
	// javaMultipleFilesPath    = []int32{8, 10}
	// javaStringCheckUtf8Path  = []int32{8, 27}
	// optimizeForPath          = []int32{8, 9}
	// goPackagePath            = []int32{8, 11}
	// ccEnableArenasPath       = []int32{8, 31}
	// objcClassPrefixPath      = []int32{8, 36}
	// csharpNamespacePath      = []int32{8, 37}
	// phpNamespacePath         = []int32{8, 41}
	// phpMetadataNamespacePath = []int32{8, 44}
	// rubyPackagePath          = []int32{8, 45}
)

type Override interface {
	override()
}

type ValueOverride[T string | bool] interface {
	Override
	Get() T

	valueOverride()
}

type PrefixOverride interface {
	Override
	Get() string

	prefixOverride()
}

func NewValueOverride[T string | bool | descriptorpb.FileOptions_OptimizeMode](v T) Override {
	return newValueOverride[T](v)
}

func NewPrefixOverride(p string) Override {
	return newPrefixOverride(p)
}

// Marker marks SourceCodeInfo_Location indices.
type Marker interface {
	// Mark marks the given SourceCodeInfo_Location indices.
	Mark(bufimage.ImageFile, []int32)
}

// Sweeper sweeps SourceCodeInfo_Locations.
type Sweeper interface {
	// Sweep removes SourceCodeInfo_Locations.
	Sweep() error
}

// MarkSweeper marks SourceCodeInfo_Location indices and sweeps source code info
// to remove marked SourceCodeInfo_Locations.
type MarkSweeper interface {
	Marker
	Sweeper
}

func NewMarkSweeper(image bufimage.Image) MarkSweeper {
	return newMarkSweeper(image)
}

func ModifyJavaPackage(
	marker Marker,
	imageFile bufimage.ImageFile,
	override Override,
) error {
	descriptor := imageFile.Proto()
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	switch t := override.(type) {
	case ValueOverride[string]:
		descriptor.Options.JavaPackage = proto.String(
			t.Get(),
		)
	case prefixOverride:
		descriptor.Options.JavaPackage = proto.String(
			getJavaPackageValue(imageFile, t.Get()),
		)
	default:
		// this should not happen
		return errors.New("a valid override is needed")
	}
	marker.Mark(imageFile, javaPackagePath)
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
