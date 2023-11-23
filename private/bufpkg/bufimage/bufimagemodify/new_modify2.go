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

	"github.com/bufbuild/buf/private/bufnew/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"google.golang.org/protobuf/types/descriptorpb"
)

// TODO: these might be useful, keeping them around for now

// type managedConfig interface {
// 	Disables() []managedDisableRule
// 	Overrides() []managedOverrideRule
// }

// type managedDisableRule interface {
// 	Path() string
// 	ModuleFullName() string
// 	FieldName() string
// 	FileOption() bufconfig.FileOption
// 	FieldOption() bufconfig.FieldOption
// }

// type managedOverrideRule interface {
// 	Path() string
// 	ModuleFullName() string
// 	FieldName() string
// 	FileOption() bufconfig.FileOption
// 	FieldOption() bufconfig.FieldOption
// 	Value() interface{}

// 	isManagedOverrideRule()
// }

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
	override, disable, err := overrideAndDisableFromConfig[T](
		imageFile,
		config,
		fileOption,
	)
	if err != nil {
		return err
	}
	if disable {
		return nil
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
func overrideAndDisableFromConfig[T bool | descriptorpb.FileOptions_OptimizeMode](
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
	fileOption bufconfig.FileOption,
) (*T, bool, error) {
	if isFileOptionDisabledForFile(
		imageFile,
		fileOption,
		config,
	) {
		return nil, true, nil
	}
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
			return nil, false, fmt.Errorf("invalid value type for %v override: %T", fileOption, overrideRule.Value())
		}
		override = &value
	}
	return override, false, nil
}

func modifyStringOption(
	sweeper Sweeper,
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
	valueOption bufconfig.FileOption,
	prefixOption bufconfig.FileOption,
	suffixOption bufconfig.FileOption,
	// todo: options? unify
	defaultOptionsFunc func(bufimage.ImageFile) stringOverride,
	valueFunc func(bufimage.ImageFile, stringOverride) string,
	getOptionFunc func(*descriptorpb.FileOptions) string,
	setOptionFunc func(*descriptorpb.FileOptions, string),
	sourceLocationPath []int32,
) error {
	modifyOptions := defaultOptionsFunc(imageFile)
	override, disable, err := stringOverrideAndDisableFromConfig(
		imageFile,
		config,
		valueOption,
		prefixOption,
		suffixOption,
	)
	if err != nil {
		return err
	}
	if disable {
		return nil
	}
	if override != nil {
		if override.value != "" {
			modifyOptions.value = override.value
		}
		if override.prefix != "" {
			modifyOptions.prefix = override.prefix
		}
		if override.suffix != "" {
			modifyOptions.suffix = override.suffix
		}
	}
	// either value is set or one of prefix and suffix is set.
	value := modifyOptions.value
	if value == "" {
		value = valueFunc(imageFile, modifyOptions)
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

// the first value is nil when no override rule is matched
// returns the override value and whether managed mode is DISABLED for this file for this file option.
func stringOverrideAndDisableFromConfig(
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
	valueFileOption bufconfig.FileOption,
	prefixFileOption bufconfig.FileOption,
	suffixFileOption bufconfig.FileOption,
) (*stringOverride, bool, error) {
	if isFileOptionDisabledForFile(imageFile, valueFileOption, config) {
		return nil, true, nil
	}
	var override *stringOverride
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
				return nil, false, fmt.Errorf("invalid value type for %v override: %T", valueFileOption, overrideRule.Value())
			}
			override = &stringOverride{value: valueString}
		case prefixFileOption:
			if ignorePrefix {
				continue
			}
			prefixString, ok := overrideRule.Value().(string)
			if !ok {
				// This should never happen, since the override rule has been validated.
				return &stringOverride{}, false, fmt.Errorf("invalid value type for %v override: %T", prefixFileOption, overrideRule.Value())
			}
			// Keep the suffix if the last two overrides are suffix and prefix.
			override = &stringOverride{
				prefix: prefixString,
				suffix: override.suffix,
			}
		case suffixFileOption:
			if ignoreSuffix {
				continue
			}
			suffixString, ok := overrideRule.Value().(string)
			if !ok {
				// This should never happen, since the override rule has been validated.
				return &stringOverride{}, false, fmt.Errorf("invalid value type for %v override: %T", suffixFileOption, overrideRule.Value())
			}
			// Keep the prefix if the last two overrides are suffix and prefix.
			override = &stringOverride{
				prefix: override.prefix,
				suffix: suffixString,
			}
		}
	}
	return override, false, nil
}

// TODO: rename to string override options maybe?
type stringOverride struct {
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
