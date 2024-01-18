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

// Package bufbreaking contains the breaking change detection functionality.
//
// The primary entry point to this package is the Handler.
package bufbreaking

import (
	"context"

	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufbreaking/bufbreakingconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufbreaking/internal/bufbreakingv1"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufbreaking/internal/bufbreakingv1beta1"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/internal"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"go.uber.org/zap"
)

// Handler handles the main breaking functionality.
type Handler interface {
	// Check runs the breaking checks.
	//
	// The image should have source code info for this to work properly. The previousImage
	// does not need to have source code info.
	//
	// Images should be filtered with regards to imports before passing to this function.
	Check(
		ctx context.Context,
		config *bufbreakingconfig.Config,
		previousImage bufimage.Image,
		image bufimage.Image,
	) ([]bufanalysis.FileAnnotation, error)
}

// NewHandler returns a new Handler.
func NewHandler(logger *zap.Logger) Handler {
	return newHandler(logger)
}

// RulesForConfig returns the rules for a given config.
//
// Should only be used for printing.
func RulesForConfig(config *bufbreakingconfig.Config) ([]bufcheck.Rule, error) {
	internalConfig, err := internalConfigForConfig(config)
	if err != nil {
		return nil, err
	}
	return rulesForInternalRules(internalConfig.Rules), nil
}

// GetAllRulesV1Beta1 gets all known rules.
//
// Should only be used for printing.
func GetAllRulesV1Beta1() ([]bufcheck.Rule, error) {
	internalConfig, err := internalConfigForConfig(
		&bufbreakingconfig.Config{
			Use:     internal.AllIDsForVersionSpec(bufbreakingv1beta1.VersionSpec),
			Version: bufconfig.V1Beta1Version,
		},
	)
	if err != nil {
		return nil, err
	}
	return rulesForInternalRules(internalConfig.Rules), nil
}

// GetAllRulesV1 gets all known rules.
//
// Should only be used for printing.
func GetAllRulesV1() ([]bufcheck.Rule, error) {
	internalConfig, err := internalConfigForConfig(
		&bufbreakingconfig.Config{
			Use:     internal.AllIDsForVersionSpec(bufbreakingv1.VersionSpec),
			Version: bufconfig.V1Version,
		},
	)
	if err != nil {
		return nil, err
	}
	return rulesForInternalRules(internalConfig.Rules), nil
}

// GetAllRulesAndCategoriesV1Beta1 returns all rules and categories for v1beta1 as a string slice.
//
// This is used for validation purposes only.
func GetAllRulesAndCategoriesV1Beta1() []string {
	return internal.AllCategoriesAndIDsForVersionSpec(bufbreakingv1beta1.VersionSpec)
}

// GetAllRulesAndCategoriesV1 returns all rules and categories for v1 as a string slice.
//
// This is used for validation purposes only.
func GetAllRulesAndCategoriesV1() []string {
	return internal.AllCategoriesAndIDsForVersionSpec(bufbreakingv1.VersionSpec)
}

func internalConfigForConfig(config *bufbreakingconfig.Config) (*internal.Config, error) {
	var versionSpec *internal.VersionSpec
	switch config.Version {
	case bufconfig.V1Beta1Version:
		versionSpec = bufbreakingv1beta1.VersionSpec
	case bufconfig.V1Version:
		versionSpec = bufbreakingv1.VersionSpec
	}
	return internal.ConfigBuilder{
		Use:                           config.Use,
		Except:                        config.Except,
		IgnoreRootPaths:               config.IgnoreRootPaths,
		IgnoreIDOrCategoryToRootPaths: config.IgnoreIDOrCategoryToRootPaths,
		IgnoreUnstablePackages:        config.IgnoreUnstablePackages,
	}.NewConfig(
		versionSpec,
	)
}

func rulesForInternalRules(rules []*internal.Rule) []bufcheck.Rule {
	if rules == nil {
		return nil
	}
	s := make([]bufcheck.Rule, len(rules))
	for i, e := range rules {
		s[i] = e
	}
	return s
}
