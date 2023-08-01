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
)

const (
	defaultJavaMultipleFiles      = true
	defaultPhpMetaNamespaceSuffix = "GPBMetadata"
)

func applyManagementForFile(
	marker bufimagemodifyv2.Marker,
	imageFile bufimage.ImageFile,
	managedConfig *ManagedConfig,
) error {
	for _, fileOptionGroup := range allFileOptionGroups {
		var override bufimagemodifyv2.Override
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
			modifyOptions, err := getModifyOptions(override)
			if err != nil {
				return err
			}
			err = bufimagemodifyv2.ModifyJavaPackage(marker, imageFile, modifyOptions...)
			if err != nil {
				return err
			}
		case groupJavaOuterClassname:
			if managedConfig.DisabledFunc(fileOptionJavaOuterClassname, imageFile) {
				continue
			}
			modifyOptions, err := getModifyOptions(override)
			if err != nil {
				return err
			}
			err = bufimagemodifyv2.ModifyJavaOuterClassname(marker, imageFile, modifyOptions...)
			if err != nil {
				return err
			}
		case groupJavaMultipleFiles:
			if managedConfig.DisabledFunc(fileOptionJavaMultipleFiles, imageFile) {
				continue
			}
			if override == nil {
				override = bufimagemodifyv2.NewValueOverride(defaultJavaMultipleFiles)
			}
			err := bufimagemodifyv2.ModifyJavaMultipleFiles(marker, imageFile, override)
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
			err := bufimagemodifyv2.ModifyJavaStringCheckUtf8(marker, imageFile, override)
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
			err := bufimagemodifyv2.ModifyOptimizeFor(marker, imageFile, override)
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
			if override == nil {
				// Do not modify go_package is override is nil.
				continue
			}
			err := bufimagemodifyv2.ModifyGoPackage(marker, imageFile, override)
			if err != nil {
				return err
			}
		case groupObjcClassPrefix:
			if managedConfig.DisabledFunc(fileOptionObjcClassPrefix, imageFile) {
				continue
			}
			modifyOptions, err := getModifyOptions(override)
			if err != nil {
				return err
			}
			err = bufimagemodifyv2.ModifyObjcClassPrefix(marker, imageFile, modifyOptions...)
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
			modifyOptions, err := getModifyOptions(override)
			if err != nil {
				return err
			}
			err = bufimagemodifyv2.ModifyCsharpNamespace(marker, imageFile, modifyOptions...)
			if err != nil {
				return err
			}
		case groupPhpNamespace:
			if managedConfig.DisabledFunc(fileOptionPhpNamespace, imageFile) {
				continue
			}
			modifyOptions, err := getModifyOptions(override)
			if err != nil {
				return err
			}
			err = bufimagemodifyv2.ModifyPhpNamespace(marker, imageFile, modifyOptions...)
			if err != nil {
				return err
			}
		case groupPhpMetadataNamespace:
			if managedConfig.DisabledFunc(fileOptionPhpMetadataNamespace, imageFile) {
				continue
			}
			if override == nil {
				override = bufimagemodifyv2.NewSuffixOverride(defaultPhpMetaNamespaceSuffix)
			}
			if managedConfig.DisabledFunc(fileOptionPhpMetadataNamespaceSuffix, imageFile) {
				override = disableSuffix(override)
			}
			modifyOptions, err := getModifyOptions(override)
			if err != nil {
				return err
			}
			err = bufimagemodifyv2.ModifyPhpMetadataNamespace(marker, imageFile, modifyOptions...)
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
			modifyOptions, err := getModifyOptions(override)
			if err != nil {
				return err
			}
			err = bufimagemodifyv2.ModifyRubyPackage(marker, imageFile, modifyOptions...)
			if err != nil {
				return err
			}
		default:
			// this should not happen
			return fmt.Errorf("unknown file option")
		}
	}
	return nil
}
