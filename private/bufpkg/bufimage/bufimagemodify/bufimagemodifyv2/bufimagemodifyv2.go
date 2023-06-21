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

// Override describes how to modify a file option, and
// is passed to ModifyXYZ.
type Override interface {
	override()
}

// NewPrefixOverride returns a new override on prefix.
func NewPrefixOverride(prefix string) Override {
	return newPrefixOverride(prefix)
}

// NewValueOverride returns a new override on value.
func NewValueOverride[T string | bool | descriptorpb.FileOptions_OptimizeMode](val T) Override {
	return NewValueOverride[T](val)
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
	case prefixOverride:
		descriptor.Options.JavaPackage = proto.String(getJavaPackageValue(imageFile, t.get()))
	case valueOverride[string]:
		descriptor.Options.JavaPackage = proto.String(t.get())
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
