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

// Package buflint contains the linting functionality.
//
// The primary entry point to this package is the Handler.
package buflint

import (
	"context"

	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/buflint/internal/buflintv1"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/buflint/internal/buflintv1beta1"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/buflint/internal/buflintv2"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/internal"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"go.uber.org/zap"
)

// AllFormatStrings are all format strings.
var AllFormatStrings = append(
	bufanalysis.AllFormatStrings,
	"config-ignore-yaml",
)

// Handler handles the main lint functionality.
type Handler interface {
	// Check runs the lint checks.
	//
	// The image should have source code info for this to work properly.
	//
	// Images should *not* be filtered with regards to imports before passing to this function.
	Check(
		ctx context.Context,
		config bufconfig.LintConfig,
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
func RulesForConfig(config bufconfig.LintConfig) ([]bufcheck.Rule, error) {
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
	internalConfig, err := internalConfigForConfig(buflintv1beta1.VersionSpec)
	if err != nil {
		return nil, err
	}
	return rulesForInternalRules(internalConfig.Rules), nil
}

// GetAllRulesV1 gets all known rules.
//
// Should only be used for printing.
func GetAllRulesV1() ([]bufcheck.Rule, error) {
	internalConfig, err := internalConfigForConfig(buflintv1.VersionSpec)
	if err != nil {
		return nil, err
	}
	return rulesForInternalRules(internalConfig.Rules), nil
}

// GetAllRulesV2 gets all known rules.
//
// Should only be used for printing.
func GetAllRulesV2() ([]bufcheck.Rule, error) {
	internalConfig, err := internalConfigForConfig(buflintv2.VersionSpec)
	if err != nil {
		return nil, err
	}
	return rulesForInternalRules(internalConfig.Rules), nil
}

// GetAllRulesAndCategoriesV1Beta1 returns all rules and categories for v1beta1 as a string slice.
//
// This is used for validation purposes only.
func GetAllRulesAndCategoriesV1Beta1() []string {
	return internal.AllCategoriesAndIDsForVersionSpec(buflintv1beta1.VersionSpec)
}

// GetAllRulesAndCategoriesV1 returns all rules and categories for v1 as a string slice.
//
// This is used for validation purposes only.
func GetAllRulesAndCategoriesV1() []string {
	return internal.AllCategoriesAndIDsForVersionSpec(buflintv1.VersionSpec)
}

// GetAllRulesAndCategoriesV2 returns all rules and categories for v2 as a string slice.
//
// This is used for validation purposes only.
func GetAllRulesAndCategoriesV2() []string {
	return internal.AllCategoriesAndIDsForVersionSpec(buflintv2.VersionSpec)
}

func internalConfigForConfig(config bufconfig.LintConfig) (*internal.Config, error) {
	var versionSpec *internal.VersionSpec
	switch fileVersion := config.FileVersion(); fileVersion {
	case bufconfig.FileVersionV1Beta1:
		versionSpec = buflintv1beta1.VersionSpec
	case bufconfig.FileVersionV1:
		versionSpec = buflintv1.VersionSpec
	case bufconfig.FileVersionV2:
		versionSpec = buflintv2.VersionSpec
	default:
		return nil, fmt.Errorf("unknown FileVersion: %v", fileVersion)
	}
	return internal.ConfigBuilder{
		Use:                                  config.UseIDsAndCategories(),
		Except:                               config.ExceptIDsAndCategories(),
		IgnoreRootPaths:                      config.IgnorePaths(),
		IgnoreIDOrCategoryToRootPaths:        config.IgnoreIDOrCategoryToPaths(),
		AllowCommentIgnores:                  config.AllowCommentIgnores(),
		EnumZeroValueSuffix:                  config.EnumZeroValueSuffix(),
		RPCAllowSameRequestResponse:          config.RPCAllowSameRequestResponse(),
		RPCAllowGoogleProtobufEmptyRequests:  config.RPCAllowGoogleProtobufEmptyRequests(),
		RPCAllowGoogleProtobufEmptyResponses: config.RPCAllowGoogleProtobufEmptyResponses(),
		ServiceSuffix:                        config.ServiceSuffix(),
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

func newLintConfigForVersionSpec(versionSpec *internal.VersionSpec) bufconfig.LintConfig{
	return bufconfig.NewLintConfig(
			bufconfig.NewCheckConfig(
				bufconfig.FileVersionV1Beta1,
				internal.AllIDsForVersionSpec(versionSpec),
				nil,
				nil,
				nil,
			),
			"",
			false,
			false,
			false,
			"",
			false,
		),
}

