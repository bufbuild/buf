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

package bufgenv2

import (
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/bufimagemodifyv2"
	"google.golang.org/protobuf/types/descriptorpb"
)

const (
	defaultJavaPackagePrefix      = "com"
	defaultJavaMultipleFiles      = true
	defaultPhpMetaNamespaceSuffix = "GPBMetadata"
)

// applyManagement modifies an image based on managed mode configuration.
func applyManagement(image bufimage.Image, managedConfig *ManagedConfig) error {
	markSweeper := bufimagemodifyv2.NewMarkSweeper(image)
	for _, imageFile := range image.Files() {
		if err := applyManagementForFile(markSweeper, imageFile, managedConfig); err != nil {
			return err
		}
	}
	return markSweeper.Sweep()
}

func applyManagementForFile(
	marker bufimagemodifyv2.Marker,
	imageFile bufimage.ImageFile,
	managedConfig *ManagedConfig,
) error {
	for _, fileOptionGroup := range allFileOptionGroups {
		var override override
		overrideFunc, ok := managedConfig.FileOptionGroupToOverrideFunc[fileOptionGroup]
		if ok {
			override = overrideFunc(imageFile)
		}
		switch fileOptionGroup {
		case groupJavaPackage:
			if managedConfig.DisabledFunc(fileOptionJavaPackage, imageFile) {
				continue
			}
			override = addPrefixIfNotExist(override, defaultJavaPackagePrefix)
			if managedConfig.DisabledFunc(fileOptionJavaPackagePrefix, imageFile) {
				override = disablePrefix(override)
			}
			if managedConfig.DisabledFunc(fileOptionJavaPackageSuffix, imageFile) {
				override = disableSuffix(override)
			}
			var modfiyOptions []bufimagemodifyv2.ModifyJavaPackageOption
			switch t := override.(type) {
			case nil:
				// If nil it means java_package_prefix is disabled but java_package is not disabled,
				// continue to modify without prefix.
			case valueOverride[string]:
				modfiyOptions = []bufimagemodifyv2.ModifyJavaPackageOption{
					bufimagemodifyv2.ModifyJavaPackageWithValue(t.get()),
				}
			case prefixOverride:
				modfiyOptions = []bufimagemodifyv2.ModifyJavaPackageOption{
					bufimagemodifyv2.ModifyJavaPackageWithPrefix(t.get()),
				}
			case suffixOverride:
				modfiyOptions = []bufimagemodifyv2.ModifyJavaPackageOption{
					bufimagemodifyv2.ModifyJavaPackageWithSuffix(t.get()),
				}
			case prefixSuffixOverride:
				modfiyOptions = []bufimagemodifyv2.ModifyJavaPackageOption{
					bufimagemodifyv2.ModifyJavaPackageWithPrefix(t.getPrefix()),
					bufimagemodifyv2.ModifyJavaPackageWithSuffix(t.getSuffix()),
				}
			default:
				return fmt.Errorf("invalid override type %T", override)
			}
			err := bufimagemodifyv2.ModifyJavaPackage(marker, imageFile, modfiyOptions...)
			if err != nil {
				return err
			}
		case groupJavaOuterClassname:
			if managedConfig.DisabledFunc(fileOptionJavaOuterClassname, imageFile) {
				continue
			}
			var modifyOptions []bufimagemodifyv2.ModifyJavaOuterClassnameOption
			switch t := override.(type) {
			case valueOverride[string]:
				modifyOptions = []bufimagemodifyv2.ModifyJavaOuterClassnameOption{
					bufimagemodifyv2.ModifyJavaOuterClassnameWithValue(t.get()),
				}
			case nil:
				// modify options will be empty
			default:
				return fmt.Errorf("invalid override type: %T", override)
			}
			err := bufimagemodifyv2.ModifyJavaOuterClassname(marker, imageFile, modifyOptions...)
			if err != nil {
				return err
			}
		case groupJavaMultipleFiles:
			if managedConfig.DisabledFunc(fileOptionJavaMultipleFiles, imageFile) {
				continue
			}
			javaMultipleFiles := defaultJavaMultipleFiles
			if override != nil {
				javaMultipleFilesOverride, ok := override.(valueOverride[bool])
				if !ok {
					return fmt.Errorf("invalid override type %T", override)
				}
				javaMultipleFiles = javaMultipleFilesOverride.get()
			}
			err := bufimagemodifyv2.ModifyJavaMultipleFiles(marker, imageFile, javaMultipleFiles)
			if err != nil {
				return err
			}
		case groupJavaStringCheckUtf8:
			if managedConfig.DisabledFunc(fileOptionJavaStringCheckUtf8, imageFile) {
				continue
			}
			if override == nil {
				// Do not modify java_string_check_utf8 if no override is specified.
				continue
			}
			javaStringCheckUtf8Override, ok := override.(valueOverride[bool])
			if !ok {
				return fmt.Errorf("invalid override type %T", override)
			}
			err := bufimagemodifyv2.ModifyJavaStringCheckUtf8(marker, imageFile, javaStringCheckUtf8Override.get())
			if err != nil {
				return err
			}
		case groupOptimizeFor:
			if managedConfig.DisabledFunc(fileOptionOptimizeFor, imageFile) {
				continue
			}
			if override == nil {
				// Do not modify optimize_for if no override is matched.
				continue
			}
			optimizeForOverride, ok := override.(valueOverride[descriptorpb.FileOptions_OptimizeMode])
			if !ok {
				return fmt.Errorf("invalid override type %T", override)
			}
			err := bufimagemodifyv2.ModifyOptimizeFor(marker, imageFile, optimizeForOverride.get())
			if err != nil {
				return err
			}
		case groupGoPackage:
			if managedConfig.DisabledFunc(fileOptionGoPackage, imageFile) {
				continue
			}
			if managedConfig.DisabledFunc(fileOptionGoPackagePrefix, imageFile) {
				override = disablePrefix(override)
			}
			var modifyOption bufimagemodifyv2.ModifyGoPackageOption
			switch t := override.(type) {
			case valueOverride[string]:
				modifyOption = bufimagemodifyv2.ModifyGoPackageWithValue(t.get())
			case prefixOverride:
				modifyOption = bufimagemodifyv2.ModifyGoPackageWithPrefix(t.get())
			case nil:
				// Do not modify go_package is override is nil.
				continue
			default:
				return fmt.Errorf("invalid override type: %T", override)
			}
			err := bufimagemodifyv2.ModifyGoPackage(marker, imageFile, modifyOption)
			if err != nil {
				return err
			}
		case groupCcEnableArenas:
			if managedConfig.DisabledFunc(fileOptionCcEnableArenas, imageFile) {
				continue
			}
			if override == nil {
				// Do not modify cc_enable_arenas if no override is matched.
				continue
			}
			ccEnableArenasOverride, ok := override.(valueOverride[bool])
			if !ok {
				return fmt.Errorf("invalid override type %T", override)
			}
			err := bufimagemodifyv2.ModifyCcEnableArenas(marker, imageFile, ccEnableArenasOverride.get())
			if err != nil {
				return err
			}
		case groupObjcClassPrefix:
			if managedConfig.DisabledFunc(fileOptionObjcClassPrefix, imageFile) {
				continue
			}
			var modifyOptions []bufimagemodifyv2.ModifyObjcClassPrefixOption
			switch t := override.(type) {
			case valueOverride[string]:
				modifyOptions = []bufimagemodifyv2.ModifyObjcClassPrefixOption{
					bufimagemodifyv2.ModifyObjcClassPrefixWithValue(t.get()),
				}
			case nil:
				// modify options will be empty
			default:
				return fmt.Errorf("invalid override type: %T", override)
			}
			err := bufimagemodifyv2.ModifyObjcClassPrefix(marker, imageFile, modifyOptions...)
			if err != nil {
				return err
			}
		case groupCsharpNamespace:
			if managedConfig.DisabledFunc(fileOptionCsharpNamespace, imageFile) {
				continue
			}
			if managedConfig.DisabledFunc(fileOptionCsharpNamespacePrefix, imageFile) {
				override = disablePrefix(override)
			}
			var modifyOptions []bufimagemodifyv2.ModifyCsharpNamespaceOption
			switch t := override.(type) {
			case valueOverride[string]:
				modifyOptions = []bufimagemodifyv2.ModifyCsharpNamespaceOption{
					bufimagemodifyv2.ModifyCsharpNamespaceWithValue(t.get()),
				}
			case prefixOverride:
				modifyOptions = []bufimagemodifyv2.ModifyCsharpNamespaceOption{
					bufimagemodifyv2.ModifyCsharpNamespaceWithPrefix(t.get()),
				}
			case nil:
				// modify options will be empty
			default:
				return fmt.Errorf("invalid override type: %T", override)
			}
			err := bufimagemodifyv2.ModifyCsharpNamespace(marker, imageFile, modifyOptions...)
			if err != nil {
				return err
			}
		case groupPhpNamespace:
			if managedConfig.DisabledFunc(fileOptionPhpNamespace, imageFile) {
				continue
			}
			var modifyOptions []bufimagemodifyv2.ModifyPhpNamespaceOption
			switch t := override.(type) {
			case valueOverride[string]:
				modifyOptions = []bufimagemodifyv2.ModifyPhpNamespaceOption{
					bufimagemodifyv2.ModifyPhpNamespaceWithValue(t.get()),
				}
			case nil:
				// modify options will be empty
			default:
				return fmt.Errorf("invalid override type: %T", override)
			}
			err := bufimagemodifyv2.ModifyPhpNamespace(marker, imageFile, modifyOptions...)
			if err != nil {
				return err
			}
		case groupPhpMetadataNamespace:
			if managedConfig.DisabledFunc(fileOptionPhpMetadataNamespace, imageFile) {
				continue
			}
			if override == nil {
				override = newSuffixOverride(defaultPhpMetaNamespaceSuffix)
			}
			if managedConfig.DisabledFunc(fileOptionPhpMetadataNamespaceSuffix, imageFile) {
				override = disableSuffix(override)
			}
			var modifyOptions []bufimagemodifyv2.ModifyPhpMetadataNamespaceOption
			switch t := override.(type) {
			case valueOverride[string]:
				modifyOptions = []bufimagemodifyv2.ModifyPhpMetadataNamespaceOption{
					bufimagemodifyv2.ModifyPhpMetadataNamespaceWithValue(t.get()),
				}
			case suffixOverride:
				modifyOptions = []bufimagemodifyv2.ModifyPhpMetadataNamespaceOption{
					bufimagemodifyv2.ModifyPhpMetadataNamespaceWithSuffix(t.get()),
				}
			case nil:
				// modify options will be empty
			default:
				return fmt.Errorf("invalid override type: %T", override)
			}
			err := bufimagemodifyv2.ModifyPhpMetadataNamespace(marker, imageFile, modifyOptions...)
			if err != nil {
				return err
			}
		case groupRubyPackage:
			if managedConfig.DisabledFunc(fileOptionRubyPackage, imageFile) {
				continue
			}
			if managedConfig.DisabledFunc(fileOptionRubyPackageSuffix, imageFile) {
				override = disableSuffix(override)
			}
			var modifyOptions []bufimagemodifyv2.ModifyRubyPackageOption
			switch t := override.(type) {
			case valueOverride[string]:
				modifyOptions = []bufimagemodifyv2.ModifyRubyPackageOption{
					bufimagemodifyv2.ModifyRubyPackageWithValue(t.get()),
				}
			case suffixOverride:
				modifyOptions = []bufimagemodifyv2.ModifyRubyPackageOption{
					bufimagemodifyv2.ModifyRubyPackageWithSuffix(t.get()),
				}
			case nil:
				// modify options will be empty
			default:
				return fmt.Errorf("invalid override type: %T", override)
			}
			bufimagemodifyv2.ModifyRubyPackage(marker, imageFile, modifyOptions...)
		default:
			// this should not happen
			return fmt.Errorf("unknown file option")
		}
	}
	modifier, err := bufimagemodifyv2.NewFieldOptionModifier(imageFile, marker)
	if err != nil {
		return err
	}
	for _, field := range modifier.FieldNames() {
		for _, fieldOption := range allFieldOptions {
			if managedConfig.FieldDisableFunc(fieldOption, imageFile, field) {
				continue
			}
			var override override
			if fieldOverrideFunc, ok := managedConfig.FieldOptionToOverrideFunc[fieldOption]; ok {
				override = fieldOverrideFunc(imageFile, field)
			}
			switch fieldOption {
			case fieldOptionJsType:
				if override == nil {
					continue
				}
				jsTypeOverride, ok := override.(valueOverride[descriptorpb.FieldOptions_JSType])
				if !ok {
					return fmt.Errorf("invalid override type :%T", override)
				}
				err := modifier.ModifyJSType(field, jsTypeOverride.get())
				if err != nil {
					return err
				}
			default:
				// this should not happen
			}
		}
	}
	return nil
}

// disablePrefix returns an override that does the same thing as the override provided,
// except that the one returned does not modify prefix.
func disablePrefix(override override) override {
	switch t := override.(type) {
	case prefixOverride:
		return nil
	case prefixSuffixOverride:
		return newSuffixOverride(t.getSuffix())
	}
	return override
}

// disableSuffix returns an override that does the same thing as the override provided,
// except that the one returned does not modify suffix.
func disableSuffix(override override) override {
	switch t := override.(type) {
	case suffixOverride:
		return nil
	case prefixSuffixOverride:
		return newPrefixOverride(t.getPrefix())
	}
	return override
}

// addPrefixIfNotExist returns an override that does the same thing  as the override provided,
// except that the one returned also modifies prefix. If the override provided already modifies
// prefix, or if it modifies the value directly, the function returns the same override.
func addPrefixIfNotExist(override override, prefix string) override {
	switch t := override.(type) {
	case suffixOverride:
		return newPrefixSuffixOverride(prefix, t.get())
	case nil:
		return newPrefixOverride(prefix)
	}
	return override
}
