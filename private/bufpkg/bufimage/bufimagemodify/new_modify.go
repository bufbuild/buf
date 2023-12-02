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

package bufimagemodify

import (
	"context"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// TODO: this is a temporary file, although it might stay. Need to rename the file at least. (The rest of the package can be deleted, except for )
// TODO: move this package into bufgen/internal
func Modify(
	ctx context.Context,
	image bufimage.Image,
	config bufconfig.GenerateManagedConfig,
) error {
	if !config.Enabled() {
		return nil
	}
	sweeper := newMarkSweeper(image)
	for _, imageFile := range image.Files() {
		if isWellKnownType(imageFile) {
			continue
		}
		modifyFuncs := []func(*markSweeper, bufimage.ImageFile, bufconfig.GenerateManagedConfig) error{
			modifyCcEnableArenas,
			modifyCsharpNamespace,
			modifyGoPackage,
			modifyJavaMultipleFiles,
			modifyJavaOuterClass,
			modifyJavaPackage,
			modifyJavaStringCheckUtf8,
			modifyObjcClassPrefix,
			modifyOptmizeFor,
			modifyPhpMetadataNamespace,
			modifyPhpNamespace,
			modifyRubyPackage,
			modifyJsType,
		}
		for _, modifyFunc := range modifyFuncs {
			if err := modifyFunc(sweeper, imageFile, config); err != nil {
				return err
			}
		}
	}
	return nil
}

func modifyJavaOuterClass(
	sweeper *markSweeper,
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
) error {
	return modifyStringOption(
		sweeper,
		imageFile,
		config,
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
		javaOuterClassnamePath,
	)
}

func modifyJavaPackage(
	sweeper *markSweeper,
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
) error {
	return modifyStringOption(
		sweeper,
		imageFile,
		config,
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
		javaPackagePath,
	)
}

func modifyGoPackage(
	sweeper *markSweeper,
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
) error {
	return modifyStringOption(
		sweeper,
		imageFile,
		config,
		bufconfig.FileOptionGoPackage,
		bufconfig.FileOptionGoPackagePrefix,
		bufconfig.FileOptionUnspecified,
		func(bufimage.ImageFile) stringOverrideOptions {
			return stringOverrideOptions{}
		},
		func(imageFile bufimage.ImageFile, stringOverride stringOverrideOptions) string {
			if stringOverride.prefix == "" {
				return ""
			}
			return GoPackageImportPathForFile(imageFile, stringOverride.prefix)
		},
		func(options *descriptorpb.FileOptions) string {
			return options.GetGoPackage()
		},
		func(options *descriptorpb.FileOptions, value string) {
			options.GoPackage = proto.String(value)
		},
		goPackagePath,
	)
}

func modifyObjcClassPrefix(
	sweeper *markSweeper,
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
) error {
	return modifyStringOption(
		sweeper,
		imageFile,
		config,
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
		objcClassPrefixPath,
	)
}

func modifyCsharpNamespace(
	sweeper *markSweeper,
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
) error {
	return modifyStringOption(
		sweeper,
		imageFile,
		config,
		bufconfig.FileOptionCsharpNamespace,
		bufconfig.FileOptionCsharpNamespacePrefix,
		bufconfig.FileOptionUnspecified,
		func(bufimage.ImageFile) stringOverrideOptions {
			return stringOverrideOptions{value: csharpNamespaceValue(imageFile)}
		},
		func(imageFile bufimage.ImageFile, stringOverride stringOverrideOptions) string {
			return getCsharpNamespaceValue(imageFile, stringOverride.prefix)
		},
		func(options *descriptorpb.FileOptions) string {
			return options.GetCsharpNamespace()
		},
		func(options *descriptorpb.FileOptions, value string) {
			options.CsharpNamespace = proto.String(value)
		},
		csharpNamespacePath,
	)
}

func modifyPhpNamespace(
	sweeper *markSweeper,
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
) error {
	return modifyStringOption(
		sweeper,
		imageFile,
		config,
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
		phpNamespacePath,
	)
}

func modifyPhpMetadataNamespace(
	sweeper *markSweeper,
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
) error {
	return modifyStringOption(
		sweeper,
		imageFile,
		config,
		bufconfig.FileOptionPhpMetadataNamespace,
		bufconfig.FileOptionUnspecified,
		bufconfig.FileOptionPhpMetadataNamespaceSuffix,
		func(bufimage.ImageFile) stringOverrideOptions {
			return stringOverrideOptions{value: phpMetadataNamespaceValue(imageFile)}
		},
		func(imageFile bufimage.ImageFile, stringOverride stringOverrideOptions) string {
			return getPhpMetadataNamespaceValue(imageFile, stringOverride.suffix)
		},
		func(options *descriptorpb.FileOptions) string {
			return options.GetPhpMetadataNamespace()
		},
		func(options *descriptorpb.FileOptions, value string) {
			options.PhpMetadataNamespace = proto.String(value)
		},
		phpMetadataNamespacePath,
	)
}

func modifyRubyPackage(
	sweeper *markSweeper,
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
) error {
	return modifyStringOption(
		sweeper,
		imageFile,
		config,
		bufconfig.FileOptionRubyPackage,
		bufconfig.FileOptionUnspecified,
		bufconfig.FileOptionRubyPackageSuffix,
		func(bufimage.ImageFile) stringOverrideOptions {
			return stringOverrideOptions{value: rubyPackageValue(imageFile)}
		},
		func(imageFile bufimage.ImageFile, stringOverride stringOverrideOptions) string {
			return getRubyPackageValue(imageFile, stringOverride.suffix)
		},
		func(options *descriptorpb.FileOptions) string {
			return options.GetRubyPackage()
		},
		func(options *descriptorpb.FileOptions, value string) {
			options.RubyPackage = proto.String(value)
		},
		rubyPackagePath,
	)
}

func modifyCcEnableArenas(
	sweeper *markSweeper,
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
) error {
	return modifyOption[bool](
		sweeper,
		imageFile,
		config,
		bufconfig.FileOptionCcEnableArenas,
		true,
		func(options *descriptorpb.FileOptions) bool {
			return options.GetCcEnableArenas()
		},
		func(options *descriptorpb.FileOptions, value bool) {
			options.CcEnableArenas = proto.Bool(value)
		},
		ccEnableArenasPath,
	)
}

func modifyJavaMultipleFiles(
	sweeper *markSweeper,
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
) error {
	return modifyOption[bool](
		sweeper,
		imageFile,
		config,
		bufconfig.FileOptionJavaMultipleFiles,
		true,
		func(options *descriptorpb.FileOptions) bool {
			return options.GetJavaMultipleFiles()
		},
		func(options *descriptorpb.FileOptions, value bool) {
			options.JavaMultipleFiles = proto.Bool(value)
		},
		javaMultipleFilesPath,
	)
}

func modifyJavaStringCheckUtf8(
	sweeper *markSweeper,
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
) error {
	return modifyOption[bool](
		sweeper,
		imageFile,
		config,
		bufconfig.FileOptionJavaStringCheckUtf8,
		false,
		func(options *descriptorpb.FileOptions) bool {
			return options.GetJavaStringCheckUtf8()
		},
		func(options *descriptorpb.FileOptions, value bool) {
			options.JavaStringCheckUtf8 = proto.Bool(value)
		},
		javaStringCheckUtf8Path,
	)
}

func modifyOptmizeFor(
	sweeper *markSweeper,
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
) error {
	return modifyOption[descriptorpb.FileOptions_OptimizeMode](
		sweeper,
		imageFile,
		config,
		bufconfig.FileOptionOptimizeFor,
		descriptorpb.FileOptions_SPEED,
		func(options *descriptorpb.FileOptions) descriptorpb.FileOptions_OptimizeMode {
			return options.GetOptimizeFor()
		},
		func(options *descriptorpb.FileOptions, value descriptorpb.FileOptions_OptimizeMode) {
			options.OptimizeFor = value.Enum()
		},
		optimizeForPath,
	)
}
