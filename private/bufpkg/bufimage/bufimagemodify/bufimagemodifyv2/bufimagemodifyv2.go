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
	"github.com/bufbuild/protocompile/walk"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// Marker marks SourceCodeInfo_Location paths.
type Marker interface {
	// Mark marks the given SourceCodeInfo_Location path.
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

// Override describes how to modify a file or field option, and
// may be passed to ModifyXYZ.
type Override interface {
	override()
}

// PrefixOverride is an override that applies a prefix.
type PrefixOverride interface {
	Override
	Get() string
	prefixOverride()
}

// NewPrefixOverride returns an override on prefix.
func NewPrefixOverride(prefix string) PrefixOverride {
	return newPrefixOverride(prefix)
}

// SuffixOverride is an override that applies a suffix.
type SuffixOverride interface {
	Override
	Get() string
	suffixOverride()
}

// NewSuffixOverride returns an override on suffix.
func NewSuffixOverride(suffix string) SuffixOverride {
	return newSuffixOverride(suffix)
}

// PrefixSuffixOverride is an override that applies a suffix and a prefix.
type PrefixSuffixOverride interface {
	Override
	GetPrefix() string
	GetSuffix() string
	prefixSuffixOverride()
}

// NewPrefixSuffixOverride returns an override on both prefix and suffix.
func NewPrefixSuffixOverride(
	prefix string,
	suffix string,
) PrefixSuffixOverride {
	return newPrefixSuffixOverride(prefix, suffix)
}

// ValueOverride is an override that directly modifies a file or field option.
type ValueOverride interface {
	Override
	valueOverride()
}

// NewValueOverride returns a new override on value.
func NewValueOverride[
	T string |
		bool |
		descriptorpb.FileOptions_OptimizeMode |
		descriptorpb.FieldOptions_JSType,
](val T) ValueOverride {
	return newValueOverride(val)
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

// ModifyJavaPackage modifies java_package. By default, it modifies java_package to
// the file's proto package. Specify an override option to add prefix and/or suffix,
// or set a value for this option.
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
	case prefixSuffixOverride:
		javaPackageValue = getJavaPackageValue(imageFile, t.prefix, t.suffix)
	case suffixOverride:
		javaPackageValue = getJavaPackageValue(imageFile, "", t.Get())
	case prefixOverride:
		javaPackageValue = getJavaPackageValue(imageFile, t.Get(), "")
	case valueOverride[string]:
		javaPackageValue = t.get()
	case nil:
		javaPackageValue = getJavaPackageValue(imageFile, "", "")
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

// TODO: Decide whether or not to use this one (this isn't the function we are calling right now,
// but it's an alternative way of modifying). The interface is not as nice as that of FieldOptionModifier,
// but this mean less code.
// ModifyJSType modifies JS_TYPE field option.
func ModifyJSType(
	imageFile bufimage.ImageFile,
	marker Marker,
	valueSelector func(fieldName string) (value Override, shouldModify bool),
) error {
	if internal.IsWellKnownType(imageFile) {
		return nil
	}
	err := walk.DescriptorProtosWithPath(imageFile.Proto(), func(
		fullName protoreflect.FullName,
		messageSourcePath protoreflect.SourcePath,
		message proto.Message,
	) error {
		fieldDescriptor, ok := message.(*descriptorpb.FieldDescriptorProto)
		if !ok {
			return nil
		}
		if fieldDescriptor.Type == nil || !isJsTypePermittedForType(*fieldDescriptor.Type) {
			return nil
		}
		if override, shouldModify := valueSelector(string(fullName)); shouldModify {
			jstypeOverride, ok := override.(valueOverride[descriptorpb.FieldOptions_JSType])
			if !ok {
				return fmt.Errorf("unknown Override type: %T", override)
			}
			jsType := jstypeOverride.get()
			if fieldDescriptor.Options == nil {
				fieldDescriptor.Options = &descriptorpb.FieldOptions{}
			}
			fieldDescriptor.Options.Jstype = &jsType
			if len(messageSourcePath) > 0 {
				jsTypeOptionPath := append(messageSourcePath, internal.JSTypeSubPath...)
				marker.Mark(imageFile, jsTypeOptionPath)
			}
		}
		return nil
	})
	return err
}

// FieldOptionModifier modifies field option. A new FieldOptionModifier
// should be created for each file to be modified.
type FieldOptionModifier interface {
	// FieldNames returns all fields' names from the image file.
	FieldNames() []string
	// ModifyJSType modifies field option js_type.
	ModifyJSType(string, Override) error
}

// NewFieldOptionModifier returns a new FieldOptionModifier
func NewFieldOptionModifier(
	imageFile bufimage.ImageFile,
	marker Marker,
) (FieldOptionModifier, error) {
	return newFieldOptionModifier(imageFile, marker)
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
