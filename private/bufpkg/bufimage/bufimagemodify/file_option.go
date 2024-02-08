// Copyright 2020-2024 Buf Technologies, Inc.
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
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/internal"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

var (
	// ccEnableArenas is the SourceCodeInfo path for the cc_enable_arenas option.
	// https://github.com/protocolbuffers/protobuf/blob/29152fbc064921ca982d64a3a9eae1daa8f979bb/src/google/protobuf/descriptor.proto#L420
	ccEnableArenasPath = []int32{8, 31}
	// csharpNamespacePath is the SourceCodeInfo path for the csharp_namespace option.
	// https://github.com/protocolbuffers/protobuf/blob/61689226c0e3ec88287eaed66164614d9c4f2bf7/src/google/protobuf/descriptor.proto#L428
	csharpNamespacePath = []int32{8, 37}
	// goPackagePath is the SourceCodeInfo path for the go_package option.
	// https://github.com/protocolbuffers/protobuf/blob/ee04809540c098718121e092107fbc0abc231725/src/google/protobuf/descriptor.proto#L392
	goPackagePath = []int32{8, 11}
	// javaMultipleFilesPath is the SourceCodeInfo path for the java_multiple_files option.
	// https://github.com/protocolbuffers/protobuf/blob/ee04809540c098718121e092107fbc0abc231725/src/google/protobuf/descriptor.proto#L364
	javaMultipleFilesPath = []int32{8, 10}
	// javaOuterClassnamePath is the SourceCodeInfo path for the java_outer_classname option.
	// https://github.com/protocolbuffers/protobuf/blob/87d140f851131fb8a6e8a80449cf08e73e568259/src/google/protobuf/descriptor.proto#L356
	javaOuterClassnamePath = []int32{8, 8}
	// javaPackagePath is the SourceCodeInfo path for the java_package option.
	// https://github.com/protocolbuffers/protobuf/blob/61689226c0e3ec88287eaed66164614d9c4f2bf7/src/google/protobuf/descriptor.proto#L348
	javaPackagePath = []int32{8, 1}
	// javaStringCheckUtf8Path is the SourceCodeInfo path for the java_string_check_utf8 option.
	// https://github.com/protocolbuffers/protobuf/blob/61689226c0e3ec88287eaed66164614d9c4f2bf7/src/google/protobuf/descriptor.proto#L375
	javaStringCheckUtf8Path = []int32{8, 27}
	// objcClassPrefixPath is the SourceCodeInfo path for the objc_class_prefix option.
	// https://github.com/protocolbuffers/protobuf/blob/61689226c0e3ec88287eaed66164614d9c4f2bf7/src/google/protobuf/descriptor.proto#L425
	objcClassPrefixPath = []int32{8, 36}
	// optimizeFor is the SourceCodeInfo path for the optimize_for option.
	// https://github.com/protocolbuffers/protobuf/blob/61689226c0e3ec88287eaed66164614d9c4f2bf7/src/google/protobuf/descriptor.proto#L385
	optimizeForPath = []int32{8, 9}
	// phpMetadataNamespacePath is the SourceCodeInfo path for the php_metadata_namespace option.
	// Ref: https://github.com/protocolbuffers/protobuf/blob/61689226c0e3ec88287eaed66164614d9c4f2bf7/src/google/protobuf/descriptor.proto#L448
	phpMetadataNamespacePath = []int32{8, 44}
	// phpNamespacePath is the SourceCodeInfo path for the php_namespace option.
	// Ref: https://github.com/protocolbuffers/protobuf/blob/61689226c0e3ec88287eaed66164614d9c4f2bf7/src/google/protobuf/descriptor.proto#L443
	phpNamespacePath = []int32{8, 41}

	// rubyPackagePath is the SourceCodeInfo path for the ruby_package option.
	// https://github.com/protocolbuffers/protobuf/blob/61689226c0e3ec88287eaed66164614d9c4f2bf7/src/google/protobuf/descriptor.proto#L453
	rubyPackagePath = []int32{8, 45}
)

func modifyJavaOuterClass(
	sweeper internal.MarkSweeper,
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
	options ...ModifyOption,
) error {
	modifyOptions := newModifyOptions()
	for _, option := range options {
		option(modifyOptions)
	}
	return modifyStringOption(
		sweeper,
		imageFile,
		config,
		modifyOptions.preserveExisting,
		bufconfig.FileOptionJavaOuterClassname,
		bufconfig.FileOptionUnspecified,
		bufconfig.FileOptionUnspecified,
		func(bufimage.ImageFile) stringOverrideOptions {
			return stringOverrideOptions{value: javaOuterClassnameValue(imageFile)}
		},
		func(imageFile bufimage.ImageFile, _ stringOverrideOptions) string {
			return javaOuterClassnameValue(imageFile)
		},
		func(options *descriptorpb.FileOptions) string {
			return options.GetJavaOuterClassname()
		},
		func(options *descriptorpb.FileOptions, value string) {
			options.JavaOuterClassname = proto.String(value)
		},
		func(options *descriptorpb.FileOptions) bool {
			return options != nil && options.JavaOuterClassname != nil
		},
		javaOuterClassnamePath,
	)
}

func modifyJavaPackage(
	sweeper internal.MarkSweeper,
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
	options ...ModifyOption,
) error {
	modifyOptions := newModifyOptions()
	for _, option := range options {
		option(modifyOptions)
	}
	return modifyStringOption(
		sweeper,
		imageFile,
		config,
		modifyOptions.preserveExisting,
		bufconfig.FileOptionJavaPackage,
		bufconfig.FileOptionJavaPackagePrefix,
		bufconfig.FileOptionJavaPackageSuffix,
		func(bufimage.ImageFile) stringOverrideOptions {
			return stringOverrideOptions{prefix: "com"}
		},
		getJavaPackageValue,
		func(options *descriptorpb.FileOptions) string {
			return options.GetJavaPackage()
		},
		func(options *descriptorpb.FileOptions, value string) {
			options.JavaPackage = proto.String(value)
		},
		func(options *descriptorpb.FileOptions) bool {
			return options != nil && options.JavaPackage != nil
		},
		javaPackagePath,
	)
}

func modifyGoPackage(
	sweeper internal.MarkSweeper,
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
	options ...ModifyOption,
) error {
	modifyOptions := newModifyOptions()
	for _, option := range options {
		option(modifyOptions)
	}
	return modifyStringOption(
		sweeper,
		imageFile,
		config,
		modifyOptions.preserveExisting,
		bufconfig.FileOptionGoPackage,
		bufconfig.FileOptionGoPackagePrefix,
		bufconfig.FileOptionUnspecified,
		func(bufimage.ImageFile) stringOverrideOptions {
			return stringOverrideOptions{}
		},
		func(imageFile bufimage.ImageFile, stringOverrideOptions stringOverrideOptions) string {
			if stringOverrideOptions.prefix == "" {
				return ""
			}
			return goPackageImportPathForFile(imageFile, stringOverrideOptions.prefix)
		},
		func(options *descriptorpb.FileOptions) string {
			return options.GetGoPackage()
		},
		func(options *descriptorpb.FileOptions, value string) {
			options.GoPackage = proto.String(value)
		},
		func(options *descriptorpb.FileOptions) bool {
			return options != nil && options.GoPackage != nil
		},
		goPackagePath,
	)
}

func modifyObjcClassPrefix(
	sweeper internal.MarkSweeper,
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
	options ...ModifyOption,
) error {
	modifyOptions := newModifyOptions()
	for _, option := range options {
		option(modifyOptions)
	}
	return modifyStringOption(
		sweeper,
		imageFile,
		config,
		modifyOptions.preserveExisting,
		bufconfig.FileOptionObjcClassPrefix,
		bufconfig.FileOptionUnspecified,
		bufconfig.FileOptionUnspecified,
		func(bufimage.ImageFile) stringOverrideOptions {
			return stringOverrideOptions{value: objcClassPrefixValue(imageFile)}
		},
		func(imageFile bufimage.ImageFile, _ stringOverrideOptions) string {
			return objcClassPrefixValue(imageFile)
		},
		func(options *descriptorpb.FileOptions) string {
			return options.GetObjcClassPrefix()
		},
		func(options *descriptorpb.FileOptions, value string) {
			options.ObjcClassPrefix = proto.String(value)
		},
		func(options *descriptorpb.FileOptions) bool {
			return options != nil && options.ObjcClassPrefix != nil
		},
		objcClassPrefixPath,
	)
}

func modifyCsharpNamespace(
	sweeper internal.MarkSweeper,
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
	options ...ModifyOption,
) error {
	modifyOptions := newModifyOptions()
	for _, option := range options {
		option(modifyOptions)
	}
	return modifyStringOption(
		sweeper,
		imageFile,
		config,
		modifyOptions.preserveExisting,
		bufconfig.FileOptionCsharpNamespace,
		bufconfig.FileOptionCsharpNamespacePrefix,
		bufconfig.FileOptionUnspecified,
		func(bufimage.ImageFile) stringOverrideOptions {
			return stringOverrideOptions{value: csharpNamespaceValue(imageFile)}
		},
		func(imageFile bufimage.ImageFile, stringOverrideOptions stringOverrideOptions) string {
			return getCsharpNamespaceValue(imageFile, stringOverrideOptions.prefix)
		},
		func(options *descriptorpb.FileOptions) string {
			return options.GetCsharpNamespace()
		},
		func(options *descriptorpb.FileOptions, value string) {
			options.CsharpNamespace = proto.String(value)
		},
		func(options *descriptorpb.FileOptions) bool {
			return options != nil && options.CsharpNamespace != nil
		},
		csharpNamespacePath,
	)
}

func modifyPhpNamespace(
	sweeper internal.MarkSweeper,
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
	options ...ModifyOption,
) error {
	modifyOptions := newModifyOptions()
	for _, option := range options {
		option(modifyOptions)
	}
	return modifyStringOption(
		sweeper,
		imageFile,
		config,
		modifyOptions.preserveExisting,
		bufconfig.FileOptionPhpNamespace,
		bufconfig.FileOptionUnspecified,
		bufconfig.FileOptionUnspecified,
		func(bufimage.ImageFile) stringOverrideOptions {
			return stringOverrideOptions{value: phpNamespaceValue(imageFile)}
		},
		func(imageFile bufimage.ImageFile, _ stringOverrideOptions) string {
			return phpNamespaceValue(imageFile)
		},
		func(options *descriptorpb.FileOptions) string {
			return options.GetPhpNamespace()
		},
		func(options *descriptorpb.FileOptions, value string) {
			options.PhpNamespace = proto.String(value)
		},
		func(options *descriptorpb.FileOptions) bool {
			return options != nil && options.PhpNamespace != nil
		},
		phpNamespacePath,
	)
}

func modifyPhpMetadataNamespace(
	sweeper internal.MarkSweeper,
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
	options ...ModifyOption,
) error {
	modifyOptions := newModifyOptions()
	for _, option := range options {
		option(modifyOptions)
	}
	return modifyStringOption(
		sweeper,
		imageFile,
		config,
		modifyOptions.preserveExisting,
		bufconfig.FileOptionPhpMetadataNamespace,
		bufconfig.FileOptionUnspecified,
		bufconfig.FileOptionPhpMetadataNamespaceSuffix,
		func(bufimage.ImageFile) stringOverrideOptions {
			return stringOverrideOptions{value: phpMetadataNamespaceValue(imageFile)}
		},
		func(imageFile bufimage.ImageFile, stringOverrideOptions stringOverrideOptions) string {
			return getPhpMetadataNamespaceValue(imageFile, stringOverrideOptions.suffix)
		},
		func(options *descriptorpb.FileOptions) string {
			return options.GetPhpMetadataNamespace()
		},
		func(options *descriptorpb.FileOptions, value string) {
			options.PhpMetadataNamespace = proto.String(value)
		},
		func(options *descriptorpb.FileOptions) bool {
			return options != nil && options.PhpMetadataNamespace != nil
		},
		phpMetadataNamespacePath,
	)
}

func modifyRubyPackage(
	sweeper internal.MarkSweeper,
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
	options ...ModifyOption,
) error {
	modifyOptions := newModifyOptions()
	for _, option := range options {
		option(modifyOptions)
	}
	return modifyStringOption(
		sweeper,
		imageFile,
		config,
		modifyOptions.preserveExisting,
		bufconfig.FileOptionRubyPackage,
		bufconfig.FileOptionUnspecified,
		bufconfig.FileOptionRubyPackageSuffix,
		func(bufimage.ImageFile) stringOverrideOptions {
			return stringOverrideOptions{value: rubyPackageValue(imageFile)}
		},
		func(imageFile bufimage.ImageFile, stringOverrideOptions stringOverrideOptions) string {
			return getRubyPackageValue(imageFile, stringOverrideOptions.suffix)
		},
		func(options *descriptorpb.FileOptions) string {
			return options.GetRubyPackage()
		},
		func(options *descriptorpb.FileOptions, value string) {
			options.RubyPackage = proto.String(value)
		},
		func(options *descriptorpb.FileOptions) bool {
			return options != nil && options.RubyPackage != nil
		},
		rubyPackagePath,
	)
}

func modifyCcEnableArenas(
	sweeper internal.MarkSweeper,
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
	options ...ModifyOption,
) error {
	modifyOptions := newModifyOptions()
	for _, option := range options {
		option(modifyOptions)
	}
	return modifyFileOption(
		sweeper,
		imageFile,
		config,
		modifyOptions.preserveExisting,
		bufconfig.FileOptionCcEnableArenas,
		true,
		func(options *descriptorpb.FileOptions) bool {
			return options.GetCcEnableArenas()
		},
		func(options *descriptorpb.FileOptions, value bool) {
			options.CcEnableArenas = proto.Bool(value)
		},
		func(options *descriptorpb.FileOptions) bool {
			return options != nil && options.CcEnableArenas != nil
		},
		ccEnableArenasPath,
	)
}

func modifyJavaMultipleFiles(
	sweeper internal.MarkSweeper,
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
	options ...ModifyOption,
) error {
	modifyOptions := newModifyOptions()
	for _, option := range options {
		option(modifyOptions)
	}
	return modifyFileOption(
		sweeper,
		imageFile,
		config,
		modifyOptions.preserveExisting,
		bufconfig.FileOptionJavaMultipleFiles,
		true,
		func(options *descriptorpb.FileOptions) bool {
			return options.GetJavaMultipleFiles()
		},
		func(options *descriptorpb.FileOptions, value bool) {
			options.JavaMultipleFiles = proto.Bool(value)
		},
		func(options *descriptorpb.FileOptions) bool {
			return options != nil && options.JavaMultipleFiles != nil
		},
		javaMultipleFilesPath,
	)
}

func modifyJavaStringCheckUtf8(
	sweeper internal.MarkSweeper,
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
	options ...ModifyOption,
) error {
	modifyOptions := newModifyOptions()
	for _, option := range options {
		option(modifyOptions)
	}
	return modifyFileOption(
		sweeper,
		imageFile,
		config,
		modifyOptions.preserveExisting,
		bufconfig.FileOptionJavaStringCheckUtf8,
		false,
		func(options *descriptorpb.FileOptions) bool {
			return options.GetJavaStringCheckUtf8()
		},
		func(options *descriptorpb.FileOptions, value bool) {
			options.JavaStringCheckUtf8 = proto.Bool(value)
		},
		func(options *descriptorpb.FileOptions) bool {
			return options != nil && options.JavaStringCheckUtf8 != nil
		},
		javaStringCheckUtf8Path,
	)
}

func modifyOptmizeFor(
	sweeper internal.MarkSweeper,
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
	options ...ModifyOption,
) error {
	modifyOptions := newModifyOptions()
	for _, option := range options {
		option(modifyOptions)
	}
	return modifyFileOption(
		sweeper,
		imageFile,
		config,
		modifyOptions.preserveExisting,
		bufconfig.FileOptionOptimizeFor,
		descriptorpb.FileOptions_SPEED,
		func(options *descriptorpb.FileOptions) descriptorpb.FileOptions_OptimizeMode {
			return options.GetOptimizeFor()
		},
		func(options *descriptorpb.FileOptions, value descriptorpb.FileOptions_OptimizeMode) {
			options.OptimizeFor = value.Enum()
		},
		func(options *descriptorpb.FileOptions) bool {
			return options != nil && options.OptimizeFor != nil
		},
		optimizeForPath,
	)
}

// *** PRIVATE ***

func modifyFileOption[T bool | descriptorpb.FileOptions_OptimizeMode](
	sweeper internal.MarkSweeper,
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
	preserveExisting bool,
	fileOption bufconfig.FileOption,
	// You can set this value to the same as protobuf default, in order to not modify a value by default.
	defaultValue T,
	getOptionFunc func(*descriptorpb.FileOptions) T,
	setOptionFunc func(*descriptorpb.FileOptions, T),
	checkOptionSetFunc func(*descriptorpb.FileOptions) bool,
	sourceLocationPath []int32,
) error {
	descriptor := imageFile.FileDescriptorProto()
	if preserveExisting && checkOptionSetFunc(descriptor.Options) {
		return nil
	}
	value := defaultValue
	if isFileOptionDisabledForFile(
		imageFile,
		fileOption,
		config,
	) {
		return nil
	}
	override, err := overrideFromConfig[T](
		imageFile,
		config,
		fileOption,
	)
	if err != nil {
		return err
	}
	if override != nil {
		value = *override
	}
	if getOptionFunc(descriptor.Options) == value {
		// The option is already set to the same value, don't modify or mark it.
		return nil
	}
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	setOptionFunc(descriptor.Options, value)
	sweeper.Mark(imageFile, sourceLocationPath)
	return nil
}

func modifyStringOption(
	sweeper internal.MarkSweeper,
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
	preserveExisting bool,
	valueOption bufconfig.FileOption,
	prefixOption bufconfig.FileOption,
	suffixOption bufconfig.FileOption,
	defaultOptionsFunc func(bufimage.ImageFile) stringOverrideOptions,
	valueFunc func(bufimage.ImageFile, stringOverrideOptions) string,
	getOptionFunc func(*descriptorpb.FileOptions) string,
	setOptionFunc func(*descriptorpb.FileOptions, string),
	checkOptionSetFunc func(*descriptorpb.FileOptions) bool,
	sourceLocationPath []int32,
) error {
	descriptor := imageFile.FileDescriptorProto()
	if preserveExisting && checkOptionSetFunc(descriptor.Options) {
		return nil
	}
	overrideOptions, err := stringOverrideFromConfig(
		imageFile,
		config,
		defaultOptionsFunc(imageFile),
		valueOption,
		prefixOption,
		suffixOption,
	)
	if err != nil {
		return err
	}
	var emptyOverrideOptions stringOverrideOptions
	// This means the options are all disabled.
	if overrideOptions == emptyOverrideOptions {
		return nil
	}
	// Now either value is set or prefix and/or suffix is set.
	value := overrideOptions.value
	if value == "" {
		// TODO FUTURE: pass in prefix and suffix, instead of just override options
		value = valueFunc(imageFile, overrideOptions)
	}
	if value == "" {
		return nil
	}
	if getOptionFunc(descriptor.Options) == value {
		// The option is already set to the same value, don't modify or mark it.
		return nil
	}
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	setOptionFunc(descriptor.Options, value)
	sweeper.Mark(imageFile, sourceLocationPath)
	return nil
}
