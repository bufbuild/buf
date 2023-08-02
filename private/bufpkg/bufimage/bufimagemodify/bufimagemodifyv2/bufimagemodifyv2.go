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

// FieldOptionModifier modifies field option. A new FieldOptionModifier
// should be created for each file to be modified.
type FieldOptionModifier interface {
	// FieldNames returns all fields' names from the image file.
	FieldNames() []string
	// ModifyJSType modifies field option jstype.
	ModifyJSType(string, descriptorpb.FieldOptions_JSType) error
}

// NewFieldOptionModifier returns a new FieldOptionModifier
func NewFieldOptionModifier(
	imageFile bufimage.ImageFile,
	marker Marker,
) (FieldOptionModifier, error) {
	return newFieldOptionModifier(imageFile, marker)
}

// ModifyJavaPackageOption is an option for ModifyJavaPackage.
type ModifyJavaPackageOption func(*modifyJavaPackageOptions)

// ModifyJavaPackageWithValue is an option for setting java_package to this value.
func ModifyJavaPackageWithValue(value string) ModifyJavaPackageOption {
	return func(options *modifyJavaPackageOptions) {
		options.value = value
	}
}

// ModifyJavaPackageWithPrefix is an option for setting java_package to the prefix
// followed by the proto package.
func ModifyJavaPackageWithPrefix(prefix string) ModifyJavaPackageOption {
	return func(options *modifyJavaPackageOptions) {
		options.prefix = prefix
	}
}

// ModifyJavaPackageWithSuffix is an option for setting java_package to the proto
// package followed by the suffix.
func ModifyJavaPackageWithSuffix(suffix string) ModifyJavaPackageOption {
	return func(options *modifyJavaPackageOptions) {
		options.suffix = suffix
	}
}

// ModifyJavaPackage modifies java_package. By default, it modifies java_package to
// the file's proto package. Specify an override option to add prefix and/or suffix,
// or set a value for this option.
func ModifyJavaPackage(
	marker Marker,
	imageFile bufimage.ImageFile,
	modifyOptions ...ModifyJavaPackageOption,
) error {
	options := &modifyJavaPackageOptions{}
	for _, option := range modifyOptions {
		option(options)
	}
	var javaPackageValue string
	if len(options.value) > 0 {
		if len(options.prefix) > 0 || len(options.suffix) > 0 {
			// the caller must make sure this does not happen
			return errors.New("must not specify prefix or suffix if modifying java package by value")
		}
		javaPackageValue = options.value
	} else {
		javaPackageValue = getJavaPackageValue(imageFile, options.prefix, options.suffix)
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

// ModifyJavaOuterClassnameOption is an option for ModifyJavaOuterClassname.
type ModifyJavaOuterClassnameOption func(*modifyStringValueOptions)

// ModifyJavaOuterClassnameWithValue is an option for setting java_outer_classname to this value.
func ModifyJavaOuterClassnameWithValue(value string) ModifyJavaOuterClassnameOption {
	return func(options *modifyStringValueOptions) {
		options.value = value
	}
}

// ModifyJavaOuterClassname modifies java_outer_classname. By default, it modifies this option to
// `<ProtoFileName>Proto`. Specify an override option to set it to a specific value.
func ModifyJavaOuterClassname(
	marker Marker,
	imageFile bufimage.ImageFile,
	modifyOptions ...ModifyJavaOuterClassnameOption,
) error {
	if internal.IsWellKnownType(imageFile) {
		return nil
	}
	options := &modifyStringValueOptions{
		value: internal.DefaultJavaOuterClassname(imageFile),
	}
	for _, option := range modifyOptions {
		option(options)
	}
	javaOuterClassname := options.value
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

// ModifyJavaMultipleFiles modifies java_multiple_files.
func ModifyJavaMultipleFiles(
	marker Marker,
	imageFile bufimage.ImageFile,
	javaMultipleFiles bool,
) error {
	if internal.IsWellKnownType(imageFile) {
		return nil
	}
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

// ModifyJavaStringCheckUtf8 modifies java_string_check_utf8.
func ModifyJavaStringCheckUtf8(
	marker Marker,
	imageFile bufimage.ImageFile,
	javaStringCheckUtf8 bool,
) error {
	if internal.IsWellKnownType(imageFile) {
		return nil
	}
	descriptor := imageFile.Proto()
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	if descriptor.Options.GetJavaStringCheckUtf8() == javaStringCheckUtf8 {
		return nil
	}
	descriptor.Options.JavaStringCheckUtf8 = proto.Bool(javaStringCheckUtf8)
	marker.Mark(imageFile, internal.JavaStringCheckUtf8Path)
	return nil
}

// ModifyGoPackageOption is an option for ModifyGoPackage.
type ModifyGoPackageOption func(*modifyValueOrPrefixOptions)

// ModifyGoPackageWithValue is an option for setting go_package to this value.
func ModifyGoPackageWithValue(value string) ModifyGoPackageOption {
	return func(options *modifyValueOrPrefixOptions) {
		options.value = value
	}
}

// ModifyGoPackageWithPrefix is an option for modifying go_package with a prefix.
func ModifyGoPackageWithPrefix(prefix string) ModifyGoPackageOption {
	return func(options *modifyValueOrPrefixOptions) {
		options.prefix = prefix
	}
}

// ModifyGoPackage modifies go_package.
func ModifyGoPackage(
	marker Marker,
	imageFile bufimage.ImageFile,
	modifyOption ModifyGoPackageOption,
) error {
	if internal.IsWellKnownType(imageFile) {
		return nil
	}
	options := &modifyValueOrPrefixOptions{}
	modifyOption(options)
	goPackageValue := options.value
	if len(options.prefix) > 0 {
		goPackageValue = internal.GoPackageImportPathForFile(imageFile, options.prefix)
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

// ModifyOptimizeFor modifies optimize_for.
func ModifyOptimizeFor(
	marker Marker,
	imageFile bufimage.ImageFile,
	optimizeFor descriptorpb.FileOptions_OptimizeMode,
) error {
	if internal.IsWellKnownType(imageFile) {
		return nil
	}
	descriptor := imageFile.Proto()
	if descriptor.Options != nil && descriptor.Options.GetOptimizeFor() == optimizeFor {
		return nil
	}
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.OptimizeFor = &optimizeFor
	marker.Mark(imageFile, internal.OptimizeForPath)
	return nil
}

// ModifyObjcClassPrefixOption is an option for ModifyObjcClassPrefix.
type ModifyObjcClassPrefixOption func(*modifyStringValueOptions)

// ModifyObjcClassPrefixWithValue is an option for setting objc_class_prefix to this value.
func ModifyObjcClassPrefixWithValue(value string) ModifyObjcClassPrefixOption {
	return func(options *modifyStringValueOptions) {
		options.value = value
	}
}

// ModifyObjcClassPrefix modifies objc_class_prefix.
func ModifyObjcClassPrefix(
	marker Marker,
	imageFile bufimage.ImageFile,
	modifyObjcClassPrefixOptions ...ModifyObjcClassPrefixOption,
) error {
	if internal.IsWellKnownType(imageFile) {
		return nil
	}
	options := &modifyStringValueOptions{
		value: internal.DefaultObjcClassPrefixValue(imageFile),
	}
	for _, option := range modifyObjcClassPrefixOptions {
		option(options)
	}
	objcClassPrefixValue := options.value
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

// ModifyCsharpNamespaceOption is an option for ModifyCsharpNamespace.
type ModifyCsharpNamespaceOption func(*modifyValueOrPrefixOptions)

// ModifyCsharpNamespaceWithValue is an option that sets csharp_namespace to this value.
func ModifyCsharpNamespaceWithValue(value string) ModifyCsharpNamespaceOption {
	return func(options *modifyValueOrPrefixOptions) {
		options.value = value
	}
}

// ModifyCsharpNamespaceWithPrefix is an option that modifies csharp_namespace with a prefix.
func ModifyCsharpNamespaceWithPrefix(prefix string) ModifyCsharpNamespaceOption {
	return func(options *modifyValueOrPrefixOptions) {
		options.prefix = prefix
	}
}

// ModifyCsharpNamespace modifies csharp_namespace.
func ModifyCsharpNamespace(
	marker Marker,
	imageFile bufimage.ImageFile,
	modifyCsharpNamespaceOptions ...ModifyCsharpNamespaceOption,
) error {
	if internal.IsWellKnownType(imageFile) {
		return nil
	}
	options := &modifyValueOrPrefixOptions{
		value: internal.DefaultCsharpNamespace(imageFile),
	}
	for _, option := range modifyCsharpNamespaceOptions {
		option(options)
	}
	csharpNamespaceValue := options.value
	if len(options.prefix) > 0 {
		csharpNamespaceValue = getCsharpNamespaceValue(imageFile, options.prefix)
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

// ModifyPhpNamespaceOption is an option for ModifyPhpNamespace.
type ModifyPhpNamespaceOption func(*modifyStringValueOptions)

// ModifyPhpNamespaceWithValue is an option for setting php_namespace to this value.
func ModifyPhpNamespaceWithValue(value string) ModifyPhpNamespaceOption {
	return func(options *modifyStringValueOptions) {
		options.value = value
	}
}

// ModifyPhpNamespace modifies php_namespace.
func ModifyPhpNamespace(
	marker Marker,
	imageFile bufimage.ImageFile,
	modifyPhpNamespaceOptions ...ModifyPhpNamespaceOption,
) error {
	if internal.IsWellKnownType(imageFile) {
		return nil
	}
	options := &modifyStringValueOptions{
		value: internal.DefaultPhpNamespaceValue(imageFile),
	}
	for _, option := range modifyPhpNamespaceOptions {
		option(options)
	}
	phpNamespaceValue := options.value
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

// ModifyPhpMetadataNamespaceOption is an option for ModifyPhpMetadataNamespace.
type ModifyPhpMetadataNamespaceOption func(*modifyValueOrSuffixOptions)

// ModifyPhpMetadataNamespaceWithValue is an option that sets php_metadata_namespace to this value.
func ModifyPhpMetadataNamespaceWithValue(value string) ModifyPhpMetadataNamespaceOption {
	return func(options *modifyValueOrSuffixOptions) {
		options.value = value
	}
}

// ModifyPhpMetadataNamespaceWithSuffix is an option that modifies php_metadata_namespace with this suffix.
func ModifyPhpMetadataNamespaceWithSuffix(suffix string) ModifyPhpMetadataNamespaceOption {
	return func(options *modifyValueOrSuffixOptions) {
		options.suffix = suffix
	}
}

// ModifyPhpMetadataNamespace modifies php_metadata_namespace.
func ModifyPhpMetadataNamespace(
	marker Marker,
	imageFile bufimage.ImageFile,
	modifyPhpMetadataNamespaceOptions ...ModifyPhpMetadataNamespaceOption,
) error {
	if internal.IsWellKnownType(imageFile) {
		return nil
	}
	options := &modifyValueOrSuffixOptions{
		value: getPhpMetadataNamespaceValue(imageFile, ""),
	}
	for _, option := range modifyPhpMetadataNamespaceOptions {
		option(options)
	}
	phpMetadataNamespaceValue := options.value
	if len(options.suffix) > 0 {
		phpMetadataNamespaceValue = getPhpMetadataNamespaceValue(imageFile, options.suffix)
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

// ModifyRubyPackageOption is an option for ModifyRubyPackage.
type ModifyRubyPackageOption func(*modifyValueOrSuffixOptions)

// ModifyRubyPackageWithValue is an option that sets ruby_package to this value.
func ModifyRubyPackageWithValue(value string) ModifyRubyPackageOption {
	return func(options *modifyValueOrSuffixOptions) {
		options.value = value
	}
}

// ModifyRubyPackageWithSuffix is an option that modifies ruby_package with this suffix.
func ModifyRubyPackageWithSuffix(suffix string) ModifyRubyPackageOption {
	return func(options *modifyValueOrSuffixOptions) {
		options.suffix = suffix
	}
}

// ModifyRubyPackage modifies ruby_package.
func ModifyRubyPackage(
	marker Marker,
	imageFile bufimage.ImageFile,
	modifyRubyPackageOptions ...ModifyRubyPackageOption,
) {
	if internal.IsWellKnownType(imageFile) {
		return
	}
	options := &modifyValueOrSuffixOptions{
		value: internal.DefaultRubyPackageValue(imageFile),
	}
	for _, option := range modifyRubyPackageOptions {
		option(options)
	}
	rubyPackageValue := options.value
	if len(options.suffix) > 0 {
		rubyPackageValue = getRubyPackageValue(imageFile, options.suffix)
	}
	descriptor := imageFile.Proto()
	if descriptor.Options != nil && descriptor.Options.GetRubyPackage() == rubyPackageValue {
		return
	}
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.RubyPackage = proto.String(rubyPackageValue)
	marker.Mark(imageFile, internal.RubyPackagePath)
	return
}

type modifyStringValueOptions struct {
	value string
}

type modifyValueOrPrefixOptions struct {
	value  string
	prefix string
}

type modifyValueOrSuffixOptions struct {
	value  string
	suffix string
}

type modifyJavaPackageOptions struct {
	value  string
	prefix string
	suffix string
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
