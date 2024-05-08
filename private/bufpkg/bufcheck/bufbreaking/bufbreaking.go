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
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufcheck"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufbreaking/internal/bufbreakingv1"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufbreaking/internal/bufbreakingv1beta1"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufbreaking/internal/bufbreakingv2"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/internal"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/tracing"
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
	//
	// An error of type bufanalysis.FileAnnotationSet will be returned on breaking failure.
	Check(
		ctx context.Context,
		config bufconfig.BreakingConfig,
		previousImage bufimage.Image,
		image bufimage.Image,
	) error
}

// NewHandler returns a new Handler.
func NewHandler(logger *zap.Logger, tracer tracing.Tracer) Handler {
	return newHandler(logger, tracer)
}

// RulesForConfig returns the rules for a given config.
//
// Does NOT include deprecated rules.
//
// Should only be used for printing.
func RulesForConfig(config bufconfig.BreakingConfig) ([]bufcheck.Rule, error) {
	internalConfig, err := internalConfigForConfig(config, true)
	if err != nil {
		return nil, err
	}
	return rulesForInternalRules(internalConfig.Rules), nil
}

// GetAllRules gets all known rules for the given version.
//
// Should only be used for testing.
func GetAllRules(fileVersion bufconfig.FileVersion) ([]bufcheck.Rule, error) {
	var versionSpec *internal.VersionSpec
	switch fileVersion {
	case bufconfig.FileVersionV1Beta1:
		versionSpec = bufbreakingv1beta1.VersionSpec
	case bufconfig.FileVersionV1:
		versionSpec = bufbreakingv1.VersionSpec
	case bufconfig.FileVersionV2:
		versionSpec = bufbreakingv2.VersionSpec
	default:
		return nil, fmt.Errorf("unknown FileVersion: %v", fileVersion)
	}
	breakingConfig, err := newBreakingConfigForVersionSpec(versionSpec)
	if err != nil {
		return nil, err
	}
	internalConfig, err := internalConfigForConfig(breakingConfig, false)
	if err != nil {
		return nil, err
	}
	return rulesForInternalRules(internalConfig.Rules), nil
}

// GetAllRulesV1Beta1 gets all known rules.
//
// Should only be used for printing.
func GetAllRulesV1Beta1() ([]bufcheck.Rule, error) {
	return GetAllRules(bufconfig.FileVersionV1Beta1)
}

// GetAllRulesV1 gets all known rules.
//
// Should only be used for printing.
func GetAllRulesV1() ([]bufcheck.Rule, error) {
	return GetAllRules(bufconfig.FileVersionV1)
}

// GetAllRulesV2 gets all known rules.
//
// Should only be used for printing.
func GetAllRulesV2() ([]bufcheck.Rule, error) {
	return GetAllRules(bufconfig.FileVersionV2)
}

// GetRelevantDeprecations gets deprecation information for the given
// version. The map is from deprecated rule IDs to zero or more replacement
// rule IDs.
func GetRelevantDeprecations(fileVersion bufconfig.FileVersion) (map[string][]string, error) {
	var versionSpec *internal.VersionSpec
	switch fileVersion {
	case bufconfig.FileVersionV1Beta1:
		versionSpec = bufbreakingv1beta1.VersionSpec
	case bufconfig.FileVersionV1:
		versionSpec = bufbreakingv1.VersionSpec
	case bufconfig.FileVersionV2:
		versionSpec = bufbreakingv2.VersionSpec
	default:
		return nil, fmt.Errorf("unknown FileVersion: %v", fileVersion)
	}
	return internal.RelevantDeprecationsForVersionSpec(versionSpec)
}

func internalConfigForConfig(config bufconfig.BreakingConfig, transformDeprecated bool) (*internal.Config, error) {
	var versionSpec *internal.VersionSpec
	switch fileVersion := config.FileVersion(); fileVersion {
	case bufconfig.FileVersionV1Beta1:
		versionSpec = bufbreakingv1beta1.VersionSpec
	case bufconfig.FileVersionV1:
		versionSpec = bufbreakingv1.VersionSpec
	case bufconfig.FileVersionV2:
		versionSpec = bufbreakingv2.VersionSpec
	default:
		return nil, fmt.Errorf("unknown FileVersion: %v", fileVersion)
	}
	return internal.ConfigBuilder{
		Use:                           config.UseIDsAndCategories(),
		Except:                        config.ExceptIDsAndCategories(),
		IgnoreRootPaths:               config.IgnorePaths(),
		IgnoreIDOrCategoryToRootPaths: config.IgnoreIDOrCategoryToPaths(),
		IgnoreUnstablePackages:        config.IgnoreUnstablePackages(),
	}.NewConfig(
		versionSpec,
		transformDeprecated,
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

func newBreakingConfigForVersionSpec(versionSpec *internal.VersionSpec) (bufconfig.BreakingConfig, error) {
	ids, err := internal.AllIDsForVersionSpec(versionSpec, true)
	if err != nil {
		return nil, err
	}
	return bufconfig.NewBreakingConfig(
		bufconfig.NewEnabledCheckConfigForUseIDsAndCategories(
			versionSpec.FileVersion,
			ids,
		),
		false,
	), nil
}
