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
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/internal"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

var (
	defaultJavaPackageOverride = NewPrefixOverride("com")
)

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
	return newValueOverride[T](val)
}

func ModifyJavaPackage(
	marker Marker,
	imageFile bufimage.ImageFile,
	override Override,
) error {
	if override == nil {
		override = defaultJavaPackageOverride
	}
	var javaPackageValue string
	switch t := override.(type) {
	case prefixOverride:
		javaPackageValue = getJavaPackageValue(imageFile, t.get())
	case valueOverride[string]:
		javaPackageValue = t.get()
	default:
		return errors.New("a valid override is required for java_package")
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
	descriptor.Options.JavaPackage = &javaPackageValue
	marker.Mark(imageFile, internal.JavaPackagePath)
	return nil
}

func ModifyCcEnableArenas(
	marker Marker,
	imageFile bufimage.ImageFile,
	override Override,
) error {
	var ccEnableArenasValue bool
	switch t := override.(type) {
	case valueOverride[bool]:
		ccEnableArenasValue = t.get()
	case nil:
		// Do nothing and use Protobuf's default.
		return nil
	default:
		return errors.New("a valid override is required for cc_enable_arenas")
	}
	if internal.IsWellKnownType(imageFile) {
		return nil
	}
	descriptor := imageFile.Proto()
	if descriptor.Options != nil && descriptor.Options.GetCcEnableArenas() == ccEnableArenasValue {
		// The option is already set to the same value, don't modify or mark it.
		return nil
	}
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.CcEnableArenas = proto.Bool(ccEnableArenasValue)
	marker.Mark(imageFile, internal.CCEnableArenasPath)
	return nil
}

func ModifyCsharpNamespace(
	marker Marker,
	imageFile bufimage.ImageFile,
	override Override,
) error {
	var csharpNamespaceValue string
	switch t := override.(type) {
	case valueOverride[string]:
		csharpNamespaceValue = t.get()
	case nil:
		csharpNamespaceValue = internal.GetDefaultCsharpNamespace(imageFile)
	default:
		return errors.New("a valid override is required for csharp_namespace")
	}
	if internal.IsWellKnownType(imageFile) {
		return nil
	}
	descriptor := imageFile.Proto()
	if descriptor.Options != nil && descriptor.Options.GetCsharpNamespace() == csharpNamespaceValue {
		// The option is already set to the same value, don't modify or mark it.
		return nil
	}
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.CsharpNamespace = proto.String(csharpNamespaceValue)
	marker.Mark(imageFile, internal.CsharpNamespacePath)
	return nil
}

func ModifyGoPackage(
	marker Marker,
	imageFile bufimage.ImageFile,
	override Override,
) error {
	var goPackageValue string
	switch t := override.(type) {
	case valueOverride[string]:
		goPackageValue = t.get()
	case prefixOverride:
		goPackageValue = internal.GoPackageImportPathForFile(imageFile, t.get())
	case nil:
		// Do nothing and use Protobuf's default.
		return nil
	default:
		return errors.New("a valid override is required for go_package")
	}
	if internal.IsWellKnownType(imageFile) {
		return nil
	}
	descriptor := imageFile.Proto()
	if descriptor.Options != nil && descriptor.Options.GetGoPackage() == goPackageValue {
		// The option is already set to the same value, don't modify or mark it.
		return nil
	}
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.GoPackage = proto.String(goPackageValue)
	marker.Mark(imageFile, internal.GoPackagePath)
	return nil
}

func ModifyJavaMultipleFiles(
	marker Marker,
	imageFile bufimage.ImageFile,
	override Override,
) error {
	var javaMultipleFilesValue bool
	switch t := override.(type) {
	case valueOverride[bool]:
		javaMultipleFilesValue = t.get()
	case nil:
		javaMultipleFilesValue = internal.DefaultJavaMultipleFilesValue
	default:
		return errors.New("a valid override is required for java_multiple_files")
	}
	if internal.IsWellKnownType(imageFile) {
		return nil
	}
	descriptor := imageFile.Proto()
	if descriptor.Options != nil && descriptor.Options.GetJavaMultipleFiles() == javaMultipleFilesValue {
		// The option is already set to the same value, don't modify or mark it.
		return nil
	}
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.JavaMultipleFiles = proto.Bool(javaMultipleFilesValue)
	marker.Mark(imageFile, internal.JavaMultipleFilesPath)
	return nil
}

func ModifyJavaOuterClassname(
	marker Marker,
	imageFile bufimage.ImageFile,
	override Override,
) error {
	var javaOuterClassname string
	switch t := override.(type) {
	case valueOverride[string]:
		javaOuterClassname = t.get()
	case nil:
		javaOuterClassname = internal.GetDefaultJavaOuterClassname(imageFile)
	default:
		return errors.New("a valid override is required for java_outer_classname")
	}
	if internal.IsWellKnownType(imageFile) {
		return nil
	}
	descriptor := imageFile.Proto()
	if descriptor.Options != nil && descriptor.Options.GetJavaOuterClassname() == javaOuterClassname {
		// The option is already set to the same value, don't modify or mark it.
		return nil
	}
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.JavaOuterClassname = proto.String(javaOuterClassname)
	marker.Mark(imageFile, internal.JavaOuterClassnamePath)
	return nil
}

func ModifyJavaStringCheck(
	marker Marker,
	imageFile bufimage.ImageFile,
	override Override,
) error {
	var javaStringCheckUtf8Value bool
	switch t := override.(type) {
	case valueOverride[bool]:
		javaStringCheckUtf8Value = t.get()
	case nil:
		// Do nothing and use Protobuf's default.
		return nil
	default:
		return errors.New("a valid override is required for java_string_check_utf8")
	}
	if internal.IsWellKnownType(imageFile) {
		return nil
	}
	descriptor := imageFile.Proto()
	if descriptor.Options != nil && descriptor.Options.GetJavaStringCheckUtf8() == javaStringCheckUtf8Value {
		// The option is already set to the same value, don't modify or mark it.
		return nil
	}
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.JavaStringCheckUtf8 = proto.Bool(javaStringCheckUtf8Value)
	marker.Mark(imageFile, internal.JavaStringCheckUtf8Path)
	return nil
}

func ModifyObjcClassPrefix(
	marker Marker,
	imageFile bufimage.ImageFile,
	override Override,
) error {
	var objcClassPrefixValue string
	switch t := override.(type) {
	case valueOverride[string]:
		objcClassPrefixValue = t.get()
	case nil:
		objcClassPrefixValue = internal.GetDefaultObjcClassPrefixValue(imageFile)
	default:
		return errors.New("a valid override is required for objc_class_prefix")
	}
	if internal.IsWellKnownType(imageFile) {
		return nil
	}
	descriptor := imageFile.Proto()
	if descriptor.Options != nil && descriptor.Options.GetObjcClassPrefix() == objcClassPrefixValue {
		// The option is already set to the same value, don't modify or mark it.
		return nil
	}
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.ObjcClassPrefix = proto.String(objcClassPrefixValue)
	marker.Mark(imageFile, internal.ObjcClassPrefixPath)
	return nil
}

func ModifyOptimizeFor(
	marker Marker,
	imageFile bufimage.ImageFile,
	override Override,
) error {
	var optimizeForValue descriptorpb.FileOptions_OptimizeMode
	switch t := override.(type) {
	case valueOverride[descriptorpb.FileOptions_OptimizeMode]:
		optimizeForValue = t.get()
	case nil:
		// Do nothing and use Protobuf's default.
		return nil
	default:
		return errors.New("a valid override is required for optimize_for")
	}
	if internal.IsWellKnownType(imageFile) {
		return nil
	}
	descriptor := imageFile.Proto()
	if descriptor.Options != nil && descriptor.Options.GetOptimizeFor() == optimizeForValue {
		// The option is already set to the same value, don't modify or mark it.
		return nil
	}
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.OptimizeFor = &optimizeForValue
	marker.Mark(imageFile, internal.OptimizeForPath)
	return nil
}

func ModifyPhpMetadataNamespace(
	marker Marker,
	imageFile bufimage.ImageFile,
	override Override,
) error {
	var phpMetadataNamespaceValue string
	switch t := override.(type) {
	case valueOverride[string]:
		phpMetadataNamespaceValue = t.get()
	case nil:
		phpMetadataNamespaceValue = internal.GetDefaultPhpMetadataNamespaceValue(imageFile)
	default:
		return errors.New("a valid override is required for php_metadata_namespace")
	}
	if internal.IsWellKnownType(imageFile) {
		return nil
	}
	descriptor := imageFile.Proto()
	if descriptor.Options != nil && descriptor.Options.GetPhpMetadataNamespace() == phpMetadataNamespaceValue {
		// The option is already set to the same value, don't modify or mark it.
		return nil
	}
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.PhpMetadataNamespace = proto.String(phpMetadataNamespaceValue)
	marker.Mark(imageFile, internal.PhpMetadataNamespacePath)
	return nil
}

func ModifyPhpNamespace(
	marker Marker,
	imageFile bufimage.ImageFile,
	override Override,
) error {
	var phpNamespaceValue string
	switch t := override.(type) {
	case valueOverride[string]:
		phpNamespaceValue = t.get()
	case nil:
		phpNamespaceValue = internal.GetDefaultPhpNamespaceValue(imageFile)
	default:
		return errors.New("a valid override is required for php_namespace")
	}
	if internal.IsWellKnownType(imageFile) {
		return nil
	}
	descriptor := imageFile.Proto()
	if descriptor.Options != nil && descriptor.Options.GetPhpNamespace() == phpNamespaceValue {
		// The option is already set to the same value, don't modify or mark it.
		return nil
	}
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.PhpNamespace = proto.String(phpNamespaceValue)
	marker.Mark(imageFile, internal.PhpNamespacePath)
	return nil
}

func ModifyRubyPackage(
	marker Marker,
	imageFile bufimage.ImageFile,
	override Override,
) error {
	var rubyPackageValue string
	switch t := override.(type) {
	case valueOverride[string]:
		rubyPackageValue = t.get()
	case nil:
		rubyPackageValue = internal.GetDefaultRubyPackageValue(imageFile)
	default:
		return errors.New("a valid override is required for ruby_package")
	}
	if internal.IsWellKnownType(imageFile) {
		return nil
	}
	descriptor := imageFile.Proto()
	if descriptor.Options != nil && descriptor.Options.GetRubyPackage() == rubyPackageValue {
		// The option is already set to the same value, don't modify or mark it.
		return nil
	}
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.RubyPackage = proto.String(rubyPackageValue)
	marker.Mark(imageFile, internal.RubyPackagePath)
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
