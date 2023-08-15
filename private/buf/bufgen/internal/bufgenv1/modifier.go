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
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/bufimagemodifyv1"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"go.uber.org/zap"
)

// modifyImage modifies the image according to the given configuration (i.e. managed mode).
func NewModifier(
	logger *zap.Logger,
	config *Config,
) (bufimagemodifyv1.Modifier, error) {
	if config.ManagedConfig == nil {
		// If the config is nil, it implies that the
		// user has not enabled managed mode.
		return &nopModifier{}, nil
	}
	sweeper := bufimagemodifyv1.NewFileOptionSweeper()
	modifier, err := newModifier(logger, config.ManagedConfig, sweeper)
	if err != nil {
		return nil, err
	}
	modifier = bufimagemodifyv1.Merge(modifier, bufimagemodifyv1.ModifierFunc(sweeper.Sweep))
	return modifier, nil
}

type nopModifier struct{}

func (a *nopModifier) Modify(_ context.Context, _ bufimage.Image) error {
	return nil
}

func newModifier(
	logger *zap.Logger,
	managedConfig *ManagedConfig,
	sweeper bufimagemodifyv1.Sweeper,
) (bufimagemodifyv1.Modifier, error) {
	modifier := bufimagemodifyv1.NewMultiModifier(
		bufimagemodifyv1.JavaOuterClassname(
			logger,
			sweeper,
			managedConfig.Override[bufimagemodifyv1.JavaOuterClassNameID],
			false, // preserveExistingValue
		),
		bufimagemodifyv1.PhpNamespace(logger, sweeper, managedConfig.Override[bufimagemodifyv1.PhpNamespaceID]),
		bufimagemodifyv1.PhpMetadataNamespace(logger, sweeper, managedConfig.Override[bufimagemodifyv1.PhpMetadataNamespaceID]),
	)
	javaPackagePrefix := &JavaPackagePrefixConfig{Default: bufimagemodifyv1.DefaultJavaPackagePrefix}
	if managedConfig.JavaPackagePrefixConfig != nil {
		javaPackagePrefix = managedConfig.JavaPackagePrefixConfig
	}
	javaPackageModifier, err := bufimagemodifyv1.JavaPackage(
		logger,
		sweeper,
		javaPackagePrefix.Default,
		javaPackagePrefix.Except,
		javaPackagePrefix.Override,
		managedConfig.Override[bufimagemodifyv1.JavaPackageID],
	)
	if err != nil {
		return nil, fmt.Errorf("failed to construct java_package modifier: %w", err)
	}
	modifier = bufimagemodifyv1.Merge(
		modifier,
		javaPackageModifier,
	)
	javaMultipleFilesValue := bufimagemodifyv1.DefaultJavaMultipleFilesValue
	if managedConfig.JavaMultipleFiles != nil {
		javaMultipleFilesValue = *managedConfig.JavaMultipleFiles
	}
	javaMultipleFilesModifier, err := bufimagemodifyv1.JavaMultipleFiles(
		logger,
		sweeper,
		javaMultipleFilesValue,
		managedConfig.Override[bufimagemodifyv1.JavaMultipleFilesID],
		false, // preserveExistingValue
	)
	if err != nil {
		return nil, err
	}
	modifier = bufimagemodifyv1.Merge(modifier, javaMultipleFilesModifier)
	if managedConfig.CcEnableArenas != nil {
		ccEnableArenasModifier, err := bufimagemodifyv1.CcEnableArenas(
			logger,
			sweeper,
			*managedConfig.CcEnableArenas,
			managedConfig.Override[bufimagemodifyv1.CcEnableArenasID],
		)
		if err != nil {
			return nil, err
		}
		modifier = bufimagemodifyv1.Merge(modifier, ccEnableArenasModifier)
	}
	if managedConfig.JavaStringCheckUtf8 != nil {
		javaStringCheckUtf8, err := bufimagemodifyv1.JavaStringCheckUtf8(
			logger,
			sweeper,
			*managedConfig.JavaStringCheckUtf8,
			managedConfig.Override[bufimagemodifyv1.JavaStringCheckUtf8ID],
		)
		if err != nil {
			return nil, err
		}
		modifier = bufimagemodifyv1.Merge(modifier, javaStringCheckUtf8)
	}
	var (
		csharpNamespaceExcept   []bufmoduleref.ModuleIdentity
		csharpNamespaceOverride map[bufmoduleref.ModuleIdentity]string
	)
	if csharpNameSpaceConfig := managedConfig.CsharpNameSpaceConfig; csharpNameSpaceConfig != nil {
		csharpNamespaceExcept = csharpNameSpaceConfig.Except
		csharpNamespaceOverride = csharpNameSpaceConfig.Override
	}
	csharpNamespaceModifier := bufimagemodifyv1.CsharpNamespace(
		logger,
		sweeper,
		csharpNamespaceExcept,
		csharpNamespaceOverride,
		managedConfig.Override[bufimagemodifyv1.CsharpNamespaceID],
	)
	modifier = bufimagemodifyv1.Merge(modifier, csharpNamespaceModifier)
	if managedConfig.OptimizeForConfig != nil {
		optimizeFor, err := bufimagemodifyv1.OptimizeFor(
			logger,
			sweeper,
			managedConfig.OptimizeForConfig.Default,
			managedConfig.OptimizeForConfig.Except,
			managedConfig.OptimizeForConfig.Override,
			managedConfig.Override[bufimagemodifyv1.OptimizeForID],
		)
		if err != nil {
			return nil, err
		}
		modifier = bufimagemodifyv1.Merge(
			modifier,
			optimizeFor,
		)
	}
	if managedConfig.GoPackagePrefixConfig != nil {
		goPackageModifier, err := bufimagemodifyv1.GoPackage(
			logger,
			sweeper,
			managedConfig.GoPackagePrefixConfig.Default,
			managedConfig.GoPackagePrefixConfig.Except,
			managedConfig.GoPackagePrefixConfig.Override,
			managedConfig.Override[bufimagemodifyv1.GoPackageID],
		)
		if err != nil {
			return nil, fmt.Errorf("failed to construct go_package modifier: %w", err)
		}
		modifier = bufimagemodifyv1.Merge(
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
	objcClassPrefixModifier := bufimagemodifyv1.ObjcClassPrefix(
		logger,
		sweeper,
		objcClassPrefixDefault,
		objcClassPrefixExcept,
		objcClassPrefixOverride,
		managedConfig.Override[bufimagemodifyv1.ObjcClassPrefixID],
	)
	modifier = bufimagemodifyv1.Merge(
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
	rubyPackageModifier := bufimagemodifyv1.RubyPackage(
		logger,
		sweeper,
		rubyPackageExcept,
		rubyPackageOverrides,
		managedConfig.Override[bufimagemodifyv1.RubyPackageID],
	)
	modifier = bufimagemodifyv1.Merge(
		modifier,
		rubyPackageModifier,
	)
	return modifier, nil
}
