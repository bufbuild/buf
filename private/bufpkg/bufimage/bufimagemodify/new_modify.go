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
	"fmt"

	"github.com/bufbuild/buf/private/bufnew/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/normalpath"
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

func modifyOption[T bool | descriptorpb.FileOptions_OptimizeMode](
	sweeper Sweeper,
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
	fileOption bufconfig.FileOption,
	// You can set this value to the same as protobuf default, in order to not modify a value by default.
	defaultValue T,
	getOptionFunc func(*descriptorpb.FileOptions) T,
	setOptionFunc func(*descriptorpb.FileOptions, T),
	sourceLocationPath []int32,
) error {
	if isFileOptionDisabledForFile(
		imageFile,
		fileOption,
		config,
	) {
		return nil
	}
	value := defaultValue
	for _, overrideRule := range config.Overrides() {
		if !fileMatchConfig(imageFile, overrideRule.Path(), overrideRule.ModuleFullName()) {
			continue
		}
		if overrideRule.FileOption() != fileOption {
			continue
		}
		var ok bool
		value, ok = overrideRule.Value().(T)
		if !ok {
			// This should never happen, since the override rule has been validated.
			return fmt.Errorf("invalid value type for %v override: %T", fileOption, overrideRule.Value())
		}
	}
	descriptor := imageFile.FileDescriptorProto()
	if getOptionFunc(descriptor.Options) == value {
		// The option is already set to the same value, don't modify or mark it.
		return nil
	}
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	setOptionFunc(descriptor.Options, value)
	sweeper.mark(imageFile.Path(), sourceLocationPath)
	return nil
}

func modifyStringOption(
	sweeper Sweeper,
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
	valueOption bufconfig.FileOption,
	prefixOption bufconfig.FileOption,
	suffixOption bufconfig.FileOption,
	defaultOptionsFunc func(bufimage.ImageFile) stringOverride,
	valueFunc func(bufimage.ImageFile, stringOverride) string,
	getOptionFunc func(*descriptorpb.FileOptions) string,
	setOptionFunc func(*descriptorpb.FileOptions, string),
	sourceLocationPath []int32,
) error {
	override, ok, err := stringOverrideForFile(
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
	if !ok {
		return nil
	}
	emptyOverride := stringOverride{}
	if override == emptyOverride {
		return nil
	}
	// either value is set or one of prefix and suffix is set.
	value := override.value
	if value == "" {
		value = valueFunc(imageFile, override)
	}
	if value == "" {
		return nil
	}
	descriptor := imageFile.FileDescriptorProto()
	if getOptionFunc(descriptor.Options) == value {
		// The option is already set to the same value, don't modify or mark it.
		return nil
	}
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	setOptionFunc(descriptor.Options, value)
	sweeper.mark(imageFile.Path(), sourceLocationPath)
	return nil
}

// TODO: rename to string override options maybe?
type stringOverride struct {
	value  string
	prefix string
	suffix string
}

// Goal: for all file options, do not modify it if it returns an empty string options
//
// For those with a default prefix / suffix (java_package_prefix: com and php_metadata_namespace_suffix: GPBMetadata),
// an empty string modify options means the prefix or suffix has been disabled. Do not modify in this case.
//
// For those with a default computed value (csharp_namespace, ruby_package, php_namespace, objc_class_prefix, java_outer_classname),
// an empty string options means no override has been additionally specified, so compute it in the default way.
//
// For those without a default value / prefix / suffix (go_package), an empty value means no change.

// If the second returns whehter it's ok to modify this option. If it returns
// false, it means either the config specifies not to modify this option at all,
// or the function returns with an error.  Otherwise, it's up to the caller whether
// or not to modify this option.
func stringOverrideForFile(
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
	defaultOverride stringOverride,
	valueFileOption bufconfig.FileOption,
	prefixFileOption bufconfig.FileOption,
	suffixFileOption bufconfig.FileOption,
) (stringOverride, bool, error) {
	if isFileOptionDisabledForFile(imageFile, valueFileOption, config) {
		return stringOverride{}, false, nil
	}
	override := stringOverride{
		value:  defaultOverride.value,
		prefix: defaultOverride.prefix,
		suffix: defaultOverride.suffix,
	}
	ignorePrefix := prefixFileOption == bufconfig.FileOptionUnspecified || isFileOptionDisabledForFile(imageFile, prefixFileOption, config)
	ignoreSuffix := suffixFileOption == bufconfig.FileOptionUnspecified || isFileOptionDisabledForFile(imageFile, suffixFileOption, config)
	for _, overrideRule := range config.Overrides() {
		if !fileMatchConfig(imageFile, overrideRule.Path(), overrideRule.ModuleFullName()) {
			continue
		}
		switch overrideRule.FileOption() {
		case valueFileOption:
			valueString, ok := overrideRule.Value().(string)
			if !ok {
				// This should never happen, since the override rule has been validated.
				return stringOverride{}, false, fmt.Errorf("invalid value type for %v override: %T", valueFileOption, overrideRule.Value())
			}
			override = stringOverride{value: valueString}
		case prefixFileOption:
			if ignorePrefix {
				continue
			}
			prefixString, ok := overrideRule.Value().(string)
			if !ok {
				// This should never happen, since the override rule has been validated.
				return stringOverride{}, false, fmt.Errorf("invalid value type for %v override: %T", prefixFileOption, overrideRule.Value())
			}
			override.prefix = prefixString
			// Do not clear suffix here, because if the last two overrides are suffix and prefix, both are used.
			override.value = ""
		case suffixFileOption:
			if ignoreSuffix {
				continue
			}
			suffixString, ok := overrideRule.Value().(string)
			if !ok {
				// This should never happen, since the override rule has been validated.
				return stringOverride{}, false, fmt.Errorf("invalid value type for %v override: %T", suffixFileOption, overrideRule.Value())
			}
			override.suffix = suffixString
			// Do not clear prefix here, because if the last two overrides are suffix and prefix, both are used.
			override.value = ""
		}
	}
	return override, true, nil
}

func isFileOptionDisabledForFile(
	imageFile bufimage.ImageFile,
	fileOption bufconfig.FileOption,
	config bufconfig.GenerateManagedConfig,
) bool {
	for _, disableRule := range config.Disables() {
		if disableRule.FileOption() != bufconfig.FileOptionUnspecified && disableRule.FileOption() != fileOption {
			continue
		}
		if !fileMatchConfig(imageFile, disableRule.Path(), disableRule.ModuleFullName()) {
			continue
		}
		return true
	}
	return false
}

func getValueForFileOption(
	imageFile bufimage.ImageFile,
	fileOption bufconfig.FileOption,
	config bufconfig.GenerateManagedConfig,
) {
}

func fileMatchConfig(
	imageFile bufimage.ImageFile,
	requiredPath string,
	requiredModuleFullName string,
) bool {
	if requiredPath != "" && !normalpath.EqualsOrContainsPath(requiredPath, imageFile.Path(), normalpath.Relative) {
		return false
	}
	if requiredModuleFullName != "" && (imageFile.ModuleFullName() == nil || imageFile.ModuleFullName().String() != requiredModuleFullName) {
		return false
	}
	return true
}

// TODO: rename these helpers

func getJavaPackageValue(imageFile bufimage.ImageFile, stringOverride stringOverride) string {
	if pkg := imageFile.FileDescriptorProto().GetPackage(); pkg != "" {
		if stringOverride.prefix != "" {
			pkg = stringOverride.prefix + "." + pkg
		}
		if stringOverride.suffix != "" {
			pkg = pkg + "." + stringOverride.suffix
		}
		return pkg
	}
	return ""
}

func getCsharpNamespaceValue(imageFile bufimage.ImageFile, prefix string) string {
	namespace := csharpNamespaceValue(imageFile)
	if namespace == "" {
		return ""
	}
	if prefix == "" {
		return namespace
	}
	return prefix + "." + namespace
}

func getPhpMetadataNamespaceValue(imageFile bufimage.ImageFile, suffix string) string {
	namespace := phpNamespaceValue(imageFile)
	if namespace == "" {
		return ""
	}
	if suffix == "" {
		return namespace
	}
	return namespace + `\` + suffix
}

func getRubyPackageValue(imageFile bufimage.ImageFile, suffix string) string {
	rubyPackage := rubyPackageValue(imageFile)
	if rubyPackage == "" {
		return ""
	}
	if suffix == "" {
		return rubyPackage
	}
	return rubyPackage + "::" + suffix
}
