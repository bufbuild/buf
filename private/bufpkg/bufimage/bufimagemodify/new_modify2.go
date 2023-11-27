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
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"google.golang.org/protobuf/types/descriptorpb"
)

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

// returns the override value and whether managed mode is DISABLED for this file for this file option.
func overrideFromConfig[T bool | descriptorpb.FileOptions_OptimizeMode](
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
	fileOption bufconfig.FileOption,
) (*T, error) {
	var override *T
	for _, overrideRule := range config.Overrides() {
		if !fileMatchConfig(imageFile, overrideRule.Path(), overrideRule.ModuleFullName()) {
			continue
		}
		if overrideRule.FileOption() != fileOption {
			continue
		}
		value, ok := overrideRule.Value().(T)
		if !ok {
			// This should never happen, since the override rule has been validated.
			return nil, fmt.Errorf("invalid value type for %v override: %T", fileOption, overrideRule.Value())
		}
		override = &value
	}
	return override, nil
}

func modifyStringOption(
	sweeper Sweeper,
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
	valueOption bufconfig.FileOption,
	prefixOption bufconfig.FileOption,
	suffixOption bufconfig.FileOption,
	// todo: options? unify
	defaultOptionsFunc func(bufimage.ImageFile) stringOverrideOptions,
	valueFunc func(bufimage.ImageFile, stringOverrideOptions) string,
	getOptionFunc func(*descriptorpb.FileOptions) string,
	setOptionFunc func(*descriptorpb.FileOptions, string),
	sourceLocationPath []int32,
) error {
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
		// TODO: pass in prefix and suffix, instead of just override options
		value = valueFunc(imageFile, overrideOptions)
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

// TODO: see if this still needs to return a pointer as opposed to a struct
// the first value is nil when no override rule is matched
func stringOverrideFromConfig(
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
	defaultOverrideOptions stringOverrideOptions,
	valueFileOption bufconfig.FileOption,
	prefixFileOption bufconfig.FileOption,
	suffixFileOption bufconfig.FileOption,
) (stringOverrideOptions, error) {
	if isFileOptionDisabledForFile(
		imageFile,
		valueFileOption,
		config,
	) {
		return stringOverrideOptions{}, nil
	}
	overrideOptions := defaultOverrideOptions
	ignorePrefix := prefixFileOption == bufconfig.FileOptionUnspecified || isFileOptionDisabledForFile(imageFile, prefixFileOption, config)
	ignoreSuffix := suffixFileOption == bufconfig.FileOptionUnspecified || isFileOptionDisabledForFile(imageFile, suffixFileOption, config)
	if ignorePrefix {
		overrideOptions.prefix = ""
	}
	if ignoreSuffix {
		overrideOptions.suffix = ""
	}
	for _, overrideRule := range config.Overrides() {
		if !fileMatchConfig(imageFile, overrideRule.Path(), overrideRule.ModuleFullName()) {
			continue
		}
		switch overrideRule.FileOption() {
		case valueFileOption:
			valueString, ok := overrideRule.Value().(string)
			if !ok {
				// This should never happen, since the override rule has been validated.
				return stringOverrideOptions{}, fmt.Errorf("invalid value type for %v override: %T", valueFileOption, overrideRule.Value())
			}
			// If the latest override matched is a value override (java_package as opposed to java_package_prefix), use the value.
			overrideOptions = stringOverrideOptions{value: valueString}
		case prefixFileOption:
			if ignorePrefix {
				continue
			}
			prefix, ok := overrideRule.Value().(string)
			if !ok {
				// This should never happen, since the override rule has been validated.
				return stringOverrideOptions{}, fmt.Errorf("invalid value type for %v override: %T", prefixFileOption, overrideRule.Value())
			}
			// Keep the suffix if the last two overrides are suffix and prefix.
			overrideOptions = stringOverrideOptions{
				prefix: prefix,
				suffix: overrideOptions.suffix,
			}
		case suffixFileOption:
			if ignoreSuffix {
				continue
			}
			suffix, ok := overrideRule.Value().(string)
			if !ok {
				// This should never happen, since the override rule has been validated.
				return stringOverrideOptions{}, fmt.Errorf("invalid value type for %v override: %T", suffixFileOption, overrideRule.Value())
			}
			// Keep the prefix if the last two overrides are suffix and prefix.
			overrideOptions = stringOverrideOptions{
				prefix: overrideOptions.prefix,
				suffix: suffix,
			}
		}
	}
	return overrideOptions, nil
}

// TODO: rename to string override options maybe?
type stringOverrideOptions struct {
	value  string
	prefix string
	suffix string
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

// TODO: rename/clean up these helpers (and merge with the other original ones as well)
func getJavaPackageValue(imageFile bufimage.ImageFile, stringOverride stringOverrideOptions) string {
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
