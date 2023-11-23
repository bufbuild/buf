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

	"github.com/bufbuild/buf/private/bufnew/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// TODO: this is a temporary file, although it might stay. Need to rename the file at least.

func Modify(
	ctx context.Context,
	image bufimage.Image,
	config bufconfig.GenerateManagedConfig,
) error {
	sweeper := NewFileOptionSweeper()
	for _, imageFile := range image.Files() {
		if isWellKnownType(ctx, imageFile) {
			continue
		}
		// TODO: order them like before or by name or by field number
		if err := modifyStringOption(
			sweeper,
			imageFile,
			config,
			bufconfig.FileOptionJavaOuterClassname,
			bufconfig.FileOptionUnspecified,
			bufconfig.FileOptionUnspecified,
			func(bufimage.ImageFile) stringOverride {
				return stringOverride{value: javaOuterClassnameValue(imageFile)}
			},
			func(imageFile bufimage.ImageFile, _ stringOverride) string {
				return javaOuterClassnameValue(imageFile)
			},
			func(options *descriptorpb.FileOptions) string {
				return options.GetJavaOuterClassname()
			},
			func(options *descriptorpb.FileOptions, value string) {
				options.JavaOuterClassname = proto.String(value)
			},
			javaOuterClassnamePath,
		); err != nil {
			return err
		}
		if err := modifyStringOption(
			sweeper,
			imageFile,
			config,
			bufconfig.FileOptionJavaPackage,
			bufconfig.FileOptionJavaPackagePrefix,
			bufconfig.FileOptionJavaPackageSuffix,
			func(bufimage.ImageFile) stringOverride {
				return stringOverride{prefix: "com"}
			},
			getJavaPackageValue,
			func(options *descriptorpb.FileOptions) string {
				return options.GetJavaPackage()
			},
			func(options *descriptorpb.FileOptions, value string) {
				options.JavaPackage = proto.String(value)
			},
			javaPackagePath,
		); err != nil {
			return err
		}
		if err := modifyStringOption(
			sweeper,
			imageFile,
			config,
			bufconfig.FileOptionGoPackage,
			bufconfig.FileOptionGoPackagePrefix,
			bufconfig.FileOptionUnspecified,
			func(bufimage.ImageFile) stringOverride {
				return stringOverride{}
			},
			func(imageFile bufimage.ImageFile, stringOverride stringOverride) string {
				return GoPackageImportPathForFile(imageFile, stringOverride.prefix)
			},
			func(options *descriptorpb.FileOptions) string {
				return options.GetGoPackage()
			},
			func(options *descriptorpb.FileOptions, value string) {
				options.GoPackage = proto.String(value)
			},
			goPackagePath,
		); err != nil {
			return err
		}
		if err := modifyStringOption(
			sweeper,
			imageFile,
			config,
			bufconfig.FileOptionObjcClassPrefix,
			bufconfig.FileOptionUnspecified,
			bufconfig.FileOptionUnspecified,
			func(bufimage.ImageFile) stringOverride {
				return stringOverride{value: objcClassPrefixValue(imageFile)}
			},
			func(imageFile bufimage.ImageFile, _ stringOverride) string {
				return objcClassPrefixValue(imageFile)
			},
			func(options *descriptorpb.FileOptions) string {
				return options.GetObjcClassPrefix()
			},
			func(options *descriptorpb.FileOptions, value string) {
				options.ObjcClassPrefix = proto.String(value)
			},
			objcClassPrefixPath,
		); err != nil {
			return err
		}
		if err := modifyStringOption(
			sweeper,
			imageFile,
			config,
			bufconfig.FileOptionCsharpNamespace,
			bufconfig.FileOptionCsharpNamespacePrefix,
			bufconfig.FileOptionUnspecified,
			func(bufimage.ImageFile) stringOverride {
				return stringOverride{value: csharpNamespaceValue(imageFile)}
			},
			func(imageFile bufimage.ImageFile, stringOverride stringOverride) string {
				return getCsharpNamespaceValue(imageFile, stringOverride.prefix)
			},
			func(options *descriptorpb.FileOptions) string {
				return options.GetCsharpNamespace()
			},
			func(options *descriptorpb.FileOptions, value string) {
				options.CsharpNamespace = proto.String(value)
			},
			csharpNamespacePath,
		); err != nil {
			return err
		}
		if err := modifyStringOption(
			sweeper,
			imageFile,
			config,
			bufconfig.FileOptionPhpNamespace,
			bufconfig.FileOptionUnspecified,
			bufconfig.FileOptionUnspecified,
			func(bufimage.ImageFile) stringOverride {
				return stringOverride{value: phpNamespaceValue(imageFile)}
			},
			func(imageFile bufimage.ImageFile, _ stringOverride) string {
				return phpNamespaceValue(imageFile)
			},
			func(options *descriptorpb.FileOptions) string {
				return options.GetPhpNamespace()
			},
			func(options *descriptorpb.FileOptions, value string) {
				options.PhpNamespace = proto.String(value)
			},
			phpNamespacePath,
		); err != nil {
			return err
		}
		if err := modifyStringOption(
			sweeper,
			imageFile,
			config,
			bufconfig.FileOptionPhpMetadataNamespace,
			bufconfig.FileOptionUnspecified,
			bufconfig.FileOptionPhpMetadataNamespaceSuffix,
			func(bufimage.ImageFile) stringOverride {
				return stringOverride{value: phpMetadataNamespaceValue(imageFile)}
			},
			func(imageFile bufimage.ImageFile, stringOverride stringOverride) string {
				return getPhpMetadataNamespaceValue(imageFile, stringOverride.suffix)
			},
			func(options *descriptorpb.FileOptions) string {
				return options.GetPhpMetadataNamespace()
			},
			func(options *descriptorpb.FileOptions, value string) {
				options.PhpMetadataNamespace = proto.String(value)
			},
			phpMetadataNamespacePath,
		); err != nil {
			return err
		}
		if err := modifyStringOption(
			sweeper,
			imageFile,
			config,
			bufconfig.FileOptionRubyPackage,
			bufconfig.FileOptionUnspecified,
			bufconfig.FileOptionRubyPackageSuffix,
			func(bufimage.ImageFile) stringOverride {
				return stringOverride{value: rubyPackageValue(imageFile)}
			},
			func(imageFile bufimage.ImageFile, stringOverride stringOverride) string {
				return getRubyPackageValue(imageFile, stringOverride.suffix)
			},
			func(options *descriptorpb.FileOptions) string {
				return options.GetRubyPackage()
			},
			func(options *descriptorpb.FileOptions, value string) {
				options.RubyPackage = proto.String(value)
			},
			rubyPackagePath,
		); err != nil {
			return err
		}
		if err := modifyOption[bool](
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
		); err != nil {
			return err
		}
		if err := modifyOption[bool](
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
		); err != nil {
			return err
		}
		if err := modifyOption[bool](
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
		); err != nil {
			return err
		}
		if err := modifyOption[descriptorpb.FileOptions_OptimizeMode](
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
		); err != nil {
			return err
		}
	}
	return nil
}
