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
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"google.golang.org/protobuf/proto"
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

// ModifyJavaOuterClassname modifies java_outer_classname. By default, it modifies this option to
// `<ProtoFileName>Proto`. Specify an override option to set it to a specific value.
func ModifyJavaOuterClassname(
	marker Marker,
	imageFile bufimage.ImageFile,
	modifyJavaOuterClassnameOptions ...ModifyOption,
) error {
	if internal.IsWellKnownType(imageFile) {
		return nil
	}
	options := &modifyOptions{}
	for _, option := range modifyJavaOuterClassnameOptions {
		option(options)
	}
	javaOuterClassname := stringutil.ToPascalCase(normalpath.Base(imageFile.Path()))
	if options.override != nil {
		override, ok := options.override.(valueOverride[string])
		if !ok {
			return fmt.Errorf("unknown Override type: %T", options.override)
		}
		javaOuterClassname = override.get()
	}
	descriptor := imageFile.Proto()
	if descriptor.Options != nil && descriptor.Options.GetJavaOuterClassname() == javaOuterClassname {
		return nil
	}
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.JavaOuterClassname = proto.String(javaOuterClassname)
	marker.Mark(imageFile, internal.JavaOuterClassnamePath)
	return nil
}

func ModifyJavaMultipleFiles(
	marker Marker,
	imageFile bufimage.ImageFile,
	override Override,
) error {
	if internal.IsWellKnownType(imageFile) {
		return nil
	}
	javaMultipleFilesOverride, ok := override.(valueOverride[bool])
	if !ok {
		return fmt.Errorf("unknown Override type: %T", override)
	}
	javaMultipleFiles := javaMultipleFilesOverride.get()
	descriptor := imageFile.Proto()
	if descriptor.Options != nil && descriptor.Options.GetJavaMultipleFiles() == javaMultipleFiles {
		return nil
	}
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.JavaMultipleFiles = proto.Bool(javaMultipleFiles)
	marker.Mark(imageFile, internal.JavaMultipleFilesPath)
	return nil
}

func ModifyJavaStringCheckUtf8(
	marker Marker,
	imageFile bufimage.ImageFile,
	override Override,
) error {
	if internal.IsWellKnownType(imageFile) {
		return nil
	}
	javaStringCheckUtf8Override, ok := override.(valueOverride[bool])
	if !ok {
		return fmt.Errorf("unknown Override type: %T", override)
	}
	javaStringCheckUtf8 := javaStringCheckUtf8Override.get()
	descriptor := imageFile.Proto()
	if descriptor.Options != nil && descriptor.Options.GetJavaStringCheckUtf8() == javaStringCheckUtf8 {
		return nil
	}
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.JavaStringCheckUtf8 = proto.Bool(javaStringCheckUtf8)
	marker.Mark(imageFile, internal.JavaStringCheckUtf8Path)
	return nil
}

func ModifyGoPackage(
	marker Marker,
	imageFile bufimage.ImageFile,
	override Override,
) error {
	if internal.IsWellKnownType(imageFile) {
		return nil
	}
	var goPackageValue string
	switch t := override.(type) {
	case prefixOverride:
		goPackageValue = internal.GoPackageImportPathForFile(imageFile, t.Get())
	case valueOverride[string]:
		goPackageValue = t.get()
	default:
		return fmt.Errorf("unknown Override type: %T", override)
	}
	descriptor := imageFile.Proto()
	if descriptor.Options != nil && descriptor.Options.GetGoPackage() == goPackageValue {
		return nil
	}
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.GoPackage = proto.String(goPackageValue)
	marker.Mark(imageFile, internal.GoPackagePath)
	return nil
}

func ModifyOptimizeFor(
	marker Marker,
	imageFile bufimage.ImageFile,
	override Override,
) error {
	if internal.IsWellKnownType(imageFile) {
		return nil
	}
	optimizeForOverride, ok := override.(valueOverride[descriptorpb.FileOptions_OptimizeMode])
	if !ok {
		return fmt.Errorf("unknown Override type: %T", override)
	}
	optimizeForValue := optimizeForOverride.get()
	descriptor := imageFile.Proto()
	if descriptor.Options != nil && descriptor.Options.GetOptimizeFor() == optimizeForValue {
		return nil
	}
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.OptimizeFor = &optimizeForValue
	marker.Mark(imageFile, internal.OptimizeForPath)
	return nil
}

func ModifyObjcClassPrefix(
	marker Marker,
	imageFile bufimage.ImageFile,
	modifyObjcClassPrefixOptions ...ModifyOption,
) error {
	if internal.IsWellKnownType(imageFile) {
		return nil
	}
	options := &modifyOptions{}
	for _, option := range modifyObjcClassPrefixOptions {
		option(options)
	}
	var objcClassPrefixValue string
	switch t := options.override.(type) {
	case valueOverride[string]:
		objcClassPrefixValue = t.get()
	case nil:
		objcClassPrefixValue = internal.DefaultObjcClassPrefixValue(imageFile)
	default:
		return fmt.Errorf("unknown Override type: %T", options.override)
	}
	descriptor := imageFile.Proto()
	if descriptor.Options != nil && descriptor.Options.GetObjcClassPrefix() == objcClassPrefixValue {
		return nil
	}
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.ObjcClassPrefix = proto.String(objcClassPrefixValue)
	marker.Mark(imageFile, internal.ObjcClassPrefixPath)
	return nil
}

func ModifyCsharpNamespace(
	marker Marker,
	imageFile bufimage.ImageFile,
	modifyCsharpNamespaceOptions ...ModifyOption,
) error {
	if internal.IsWellKnownType(imageFile) {
		return nil
	}
	options := &modifyOptions{}
	for _, option := range modifyCsharpNamespaceOptions {
		option(options)
	}
	var csharpNamespaceValue string
	switch t := options.override.(type) {
	case prefixOverride:
		csharpNamespaceValue = getCsharpNamespaceValue(imageFile, t.Get())
	case valueOverride[string]:
		csharpNamespaceValue = t.get()
	case nil:
		csharpNamespaceValue = internal.DefaultCsharpNamespace(imageFile)
	default:
		return fmt.Errorf("unknown Override type: %T", options.override)
	}
	descriptor := imageFile.Proto()
	if descriptor.Options != nil && descriptor.Options.GetCsharpNamespace() == csharpNamespaceValue {
		return nil
	}
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.CsharpNamespace = proto.String(csharpNamespaceValue)
	marker.Mark(imageFile, internal.CsharpNamespacePath)
	return nil
}

func ModifyPhpNamespace(
	marker Marker,
	imageFile bufimage.ImageFile,
	modifyPhpNamespaceOptions ...ModifyOption,
) error {
	if internal.IsWellKnownType(imageFile) {
		return nil
	}
	options := &modifyOptions{}
	for _, option := range modifyPhpNamespaceOptions {
		option(options)
	}
	var phpNamespaceValue string
	switch t := options.override.(type) {
	case valueOverride[string]:
		phpNamespaceValue = t.get()
	case nil:
		phpNamespaceValue = internal.DefaultPhpNamespaceValue(imageFile)
	default:
		return fmt.Errorf("unknown Override type: %T", options.override)
	}
	descriptor := imageFile.Proto()
	if descriptor.Options != nil && descriptor.Options.GetPhpNamespace() == phpNamespaceValue {
		return nil
	}
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.PhpNamespace = proto.String(phpNamespaceValue)
	marker.Mark(imageFile, internal.PhpNamespacePath)
	return nil
}

func ModifyPhpMetadataNamespace(
	marker Marker,
	imageFile bufimage.ImageFile,
	modifyPhpMetadataNamespaceOptions ...ModifyOption,
) error {
	if internal.IsWellKnownType(imageFile) {
		return nil
	}
	options := &modifyOptions{}
	for _, option := range modifyPhpMetadataNamespaceOptions {
		option(options)
	}
	var phpMetadataNamespaceValue string
	switch t := options.override.(type) {
	case suffixOverride:
		phpMetadataNamespaceValue = getPhpMetadataNamespaceValue(imageFile, t.Get())
	case valueOverride[string]:
		phpMetadataNamespaceValue = t.get()
	case nil:
		phpMetadataNamespaceValue = getPhpMetadataNamespaceValue(imageFile, "")
	default:
		return fmt.Errorf("unknown Override type: %T", options.override)
	}
	descriptor := imageFile.Proto()
	if descriptor.Options != nil && descriptor.Options.GetPhpMetadataNamespace() == phpMetadataNamespaceValue {
		return nil
	}
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.PhpMetadataNamespace = proto.String(phpMetadataNamespaceValue)
	marker.Mark(imageFile, internal.PhpMetadataNamespacePath)
	return nil
}

func ModifyRubyPackage(
	marker Marker,
	imageFile bufimage.ImageFile,
	modifyRubyPackageOptions ...ModifyOption,
) error {
	if internal.IsWellKnownType(imageFile) {
		return nil
	}
	options := &modifyOptions{}
	for _, option := range modifyRubyPackageOptions {
		option(options)
	}
	var rubyPackageValue string
	switch t := options.override.(type) {
	case suffixOverride:
		rubyPackageValue = getRubyPackageValue(imageFile, t.Get())
	case valueOverride[string]:
		rubyPackageValue = t.get()
	case nil:
		rubyPackageValue = getRubyPackageValue(imageFile, "")
	default:
		return fmt.Errorf("unknown Override type: %T", options.override)
	}
	descriptor := imageFile.Proto()
	if descriptor.Options != nil && descriptor.Options.GetRubyPackage() == rubyPackageValue {
		return nil
	}
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.RubyPackage = proto.String(rubyPackageValue)
	marker.Mark(imageFile, internal.RubyPackagePath)
	return nil
}

// FieldOptionModifier modifies field option. A new FieldOptionModifier
// should be created for each file to be modified.
type FieldOptionModifier interface {
	// FieldNames returns all fields' names from the image file.
	FieldNames() []string
	// ModifyJSType modifies field option jstype.
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

func getCsharpNamespaceValue(imageFile bufimage.ImageFile, prefix string) string {
	namespace := internal.DefaultCsharpNamespace(imageFile)
	if namespace == "" {
		return ""
	}
	if prefix == "" {
		return namespace
	}
	return prefix + "." + namespace
}

func getPhpMetadataNamespaceValue(imageFile bufimage.ImageFile, suffix string) string {
	namespace := internal.DefaultPhpNamespaceValue(imageFile)
	if namespace == "" {
		return ""
	}
	if suffix == "" {
		return namespace
	}
	return namespace + `\` + suffix
}

func getRubyPackageValue(imageFile bufimage.ImageFile, suffix string) string {
	rubyPackage := internal.DefaultRubyPackageValue(imageFile)
	if rubyPackage == "" {
		return ""
	}
	if suffix == "" {
		return rubyPackage
	}
	return rubyPackage + "::" + suffix
}
