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

package bufgenv1

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"go.uber.org/zap"
)

// modifyImage modifies the image according to the given configuration (i.e. managed mode).
func NewModifier(
	logger *zap.Logger,
	config *config,
) (bufimagemodify.Modifier, error) {
	if config.ManagedConfig == nil {
		// If the config is nil, it implies that the
		// user has not enabled managed mode.
		return &nopModifier{}, nil
	}
	sweeper := bufimagemodify.NewFileOptionSweeper()
	modifier, err := newModifier(logger, config.ManagedConfig, sweeper)
	if err != nil {
		return nil, err
	}
	modifier = bufimagemodify.Merge(modifier, bufimagemodify.ModifierFunc(sweeper.Sweep))
	return modifier, nil
}

type nopModifier struct{}

func (a *nopModifier) Modify(_ context.Context, _ bufimage.Image) error {
	return nil
}

func newModifier(
	logger *zap.Logger,
	managedConfig *ManagedConfig,
	sweeper bufimagemodify.Sweeper,
) (bufimagemodify.Modifier, error) {
	modifier := bufimagemodify.NewMultiModifier(
		bufimagemodify.JavaOuterClassname(
			logger,
			sweeper,
			managedConfig.Override[bufimagemodify.JavaOuterClassNameID],
			false, // preserveExistingValue
		),
		bufimagemodify.PhpNamespace(logger, sweeper, managedConfig.Override[bufimagemodify.PhpNamespaceID]),
		bufimagemodify.PhpMetadataNamespace(logger, sweeper, managedConfig.Override[bufimagemodify.PhpMetadataNamespaceID]),
	)
	javaPackagePrefix := &JavaPackagePrefixConfig{Default: bufimagemodify.DefaultJavaPackagePrefix}
	if managedConfig.JavaPackagePrefixConfig != nil {
		javaPackagePrefix = managedConfig.JavaPackagePrefixConfig
	}
	javaPackageModifier, err := bufimagemodify.JavaPackage(
		logger,
		sweeper,
		javaPackagePrefix.Default,
		javaPackagePrefix.Except,
		javaPackagePrefix.Override,
		managedConfig.Override[bufimagemodify.JavaPackageID],
	)
	if err != nil {
		return nil, fmt.Errorf("failed to construct java_package modifier: %w", err)
	}
	modifier = bufimagemodify.Merge(
		modifier,
		javaPackageModifier,
	)
	javaMultipleFilesValue := bufimagemodify.DefaultJavaMultipleFilesValue
	if managedConfig.JavaMultipleFiles != nil {
		javaMultipleFilesValue = *managedConfig.JavaMultipleFiles
	}
	javaMultipleFilesModifier, err := bufimagemodify.JavaMultipleFiles(
		logger,
		sweeper,
		javaMultipleFilesValue,
		managedConfig.Override[bufimagemodify.JavaMultipleFilesID],
		false, // preserveExistingValue
	)
	if err != nil {
		return nil, err
	}
	modifier = bufimagemodify.Merge(modifier, javaMultipleFilesModifier)
	if managedConfig.CcEnableArenas != nil {
		ccEnableArenasModifier, err := bufimagemodify.CcEnableArenas(
			logger,
			sweeper,
			*managedConfig.CcEnableArenas,
			managedConfig.Override[bufimagemodify.CcEnableArenasID],
		)
		if err != nil {
			return nil, err
		}
		modifier = bufimagemodify.Merge(modifier, ccEnableArenasModifier)
	}
	if managedConfig.JavaStringCheckUtf8 != nil {
		javaStringCheckUtf8, err := bufimagemodify.JavaStringCheckUtf8(
			logger,
			sweeper,
			*managedConfig.JavaStringCheckUtf8,
			managedConfig.Override[bufimagemodify.JavaStringCheckUtf8ID],
		)
		if err != nil {
			return nil, err
		}
		modifier = bufimagemodify.Merge(modifier, javaStringCheckUtf8)
	}
	var (
		csharpNamespaceExcept   []bufmoduleref.ModuleIdentity
		csharpNamespaceOverride map[bufmoduleref.ModuleIdentity]string
	)
	if csharpNameSpaceConfig := managedConfig.CsharpNameSpaceConfig; csharpNameSpaceConfig != nil {
		csharpNamespaceExcept = csharpNameSpaceConfig.Except
		csharpNamespaceOverride = csharpNameSpaceConfig.Override
	}
	csharpNamespaceModifier := bufimagemodify.CsharpNamespace(
		logger,
		sweeper,
		csharpNamespaceExcept,
		csharpNamespaceOverride,
		managedConfig.Override[bufimagemodify.CsharpNamespaceID],
	)
	modifier = bufimagemodify.Merge(modifier, csharpNamespaceModifier)
	if managedConfig.OptimizeForConfig != nil {
		optimizeFor, err := bufimagemodify.OptimizeFor(
			logger,
			sweeper,
			managedConfig.OptimizeForConfig.Default,
			managedConfig.OptimizeForConfig.Except,
			managedConfig.OptimizeForConfig.Override,
			managedConfig.Override[bufimagemodify.OptimizeForID],
		)
		if err != nil {
			return nil, err
		}
		modifier = bufimagemodify.Merge(
			modifier,
			optimizeFor,
		)
	}
	if managedConfig.GoPackagePrefixConfig != nil {
		goPackageModifier, err := bufimagemodify.GoPackage(
			logger,
			sweeper,
			managedConfig.GoPackagePrefixConfig.Default,
			managedConfig.GoPackagePrefixConfig.Except,
			managedConfig.GoPackagePrefixConfig.Override,
			managedConfig.Override[bufimagemodify.GoPackageID],
		)
		if err != nil {
			return nil, fmt.Errorf("failed to construct go_package modifier: %w", err)
		}
		modifier = bufimagemodify.Merge(
			modifier,
			goPackageModifier,
		)
	}
	var (
		objcClassPrefixDefault  string
		objcClassPrefixExcept   []bufmoduleref.ModuleIdentity
		objcClassPrefixOverride map[bufmoduleref.ModuleIdentity]string
	)
	if objcClassPrefixConfig := managedConfig.ObjcClassPrefixConfig; objcClassPrefixConfig != nil {
		objcClassPrefixDefault = objcClassPrefixConfig.Default
		objcClassPrefixExcept = objcClassPrefixConfig.Except
		objcClassPrefixOverride = objcClassPrefixConfig.Override
	}
	objcClassPrefixModifier := bufimagemodify.ObjcClassPrefix(
		logger,
		sweeper,
		objcClassPrefixDefault,
		objcClassPrefixExcept,
		objcClassPrefixOverride,
		managedConfig.Override[bufimagemodify.ObjcClassPrefixID],
	)
	modifier = bufimagemodify.Merge(
		modifier,
		objcClassPrefixModifier,
	)
	var (
		rubyPackageExcept    []bufmoduleref.ModuleIdentity
		rubyPackageOverrides map[bufmoduleref.ModuleIdentity]string
	)
	if rubyPackageConfig := managedConfig.RubyPackageConfig; rubyPackageConfig != nil {
		rubyPackageExcept = rubyPackageConfig.Except
		rubyPackageOverrides = rubyPackageConfig.Override
	}
	rubyPackageModifier := bufimagemodify.RubyPackage(
		logger,
		sweeper,
		rubyPackageExcept,
		rubyPackageOverrides,
		managedConfig.Override[bufimagemodify.RubyPackageID],
	)
	modifier = bufimagemodify.Merge(
		modifier,
		rubyPackageModifier,
	)
	return modifier, nil
}
