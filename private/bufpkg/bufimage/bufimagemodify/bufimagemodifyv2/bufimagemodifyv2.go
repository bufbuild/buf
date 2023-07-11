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
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/internal"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// Marker markssource SourceCodeInfo_Location indices.
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

// NewMarkSweeper returns a MarkSweeper.
func NewMarkSweeper(image bufimage.Image) MarkSweeper {
	return newMarkSweeper(image)
}

// Override describes how to modify a file option, and
// may be passed to ModifyXYZ.
type Override interface {
	override()
}

// PrefixOverride is an override that applies a prefix.
type PrefixOverride interface {
	Override
	get() string
	prefixOverride()
}

// NewPrefixOverride returns an override on prefix.
func NewPrefixOverride(prefix string) PrefixOverride {
	return newPrefixOverride(prefix)
}

// SuffixOverride is an override that applies a suffix.
type SuffixOverride interface {
	Override
	get() string
	suffixOverride()
}

// NewSuffixOverride returns an override on suffix.
func NewSuffixOverride(suffix string) SuffixOverride {
	return newSuffixOverride(suffix)
}

// PrefixSuffixOverride is an override that applies a suffix and a prefix.
type PrefixSuffixOverride interface {
	Override
	prefixSuffixOverride()
}

// CombinePrefixSuffixOverride returns an override on both prefix and suffix.
func CombinePrefixSuffixOverride(
	prefixOverride PrefixOverride,
	suffixOverride SuffixOverride,
) PrefixSuffixOverride {
	return newPrefixSuffixOverride(prefixOverride.get(), suffixOverride.get())
}

// ValueOverride is an override that directly modifies a file option.
type ValueOverride interface {
	Override
	valueOverride()
}

// NewValueOverride returns a new override on value.
func NewValueOverride[T string | bool | descriptorpb.FileOptions_OptimizeMode](val T) ValueOverride {
	return newValueOverride[T](val)
}

// ModifyOption is an option for ModifyXYZ.
type ModifyOption func(*modifyOptions)

// ModifyWithOverride modifies an option with override.
func ModifyWithOverride(override Override) (ModifyOption, error) {
	if override == nil {
		return nil, errors.New("override must not be nil")
	}
	return func(options *modifyOptions) {
		options.override = override
	}, nil
}

type modifyOptions struct {
	override Override
}

// ModifyJavaPackage modifies java_package.
func ModifyJavaPackage(
	marker Marker,
	imageFile bufimage.ImageFile,
	modifyJavaPackageOptions ...ModifyOption,
) error {
	options := &modifyOptions{}
	for _, option := range modifyJavaPackageOptions {
		option(options)
	}
	var javaPackageValue string
	switch t := options.override.(type) {
	case prefixOverride:
		javaPackageValue = getJavaPackageValue(imageFile, t.get(), internal.DefaultJavaPackageSuffix)
	case suffixOverride:
		return errors.New("cannot modify java_package with a suffix but without a prefix")
	case prefixSuffixOverride:
		javaPackageValue = getJavaPackageValue(imageFile, t.prefix, t.suffix)
	case valueOverride[string]:
		javaPackageValue = t.get()
	case nil:
		javaPackageValue = getJavaPackageValue(imageFile, internal.DefaultJavaPackagePrefix, internal.DefaultJavaPackageSuffix)
	default:
		return fmt.Errorf("unknown Override type: %T", options.override)
	}
	if internal.IsWellKnownType(imageFile) {
		return nil
	}
	descriptor := imageFile.Proto()
	if descriptor.Options != nil && descriptor.Options.GetJavaPackage() == javaPackageValue {
		// The option is already set to the same value, don't modify or mark it.
		return nil
	}
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.JavaPackage = proto.String(javaPackageValue)
	marker.Mark(imageFile, internal.JavaPackagePath)
	return nil
}

func getJavaPackageValue(imageFile bufimage.ImageFile, prefix string, suffix string) string {
	if pkg := imageFile.Proto().GetPackage(); pkg != "" {
		if prefix != "" {
			pkg = prefix + "." + pkg
		}
		if suffix != "" {
			pkg = pkg + "." + suffix
		}
		return pkg
	}
	return ""
}
