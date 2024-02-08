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

// Package buflint contains the linting functionality.
//
// The primary entry point to this package is the Handler.
package buflint

import (
	"context"

	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/buflint/buflintconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/buflint/internal/buflintv1"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/buflint/internal/buflintv1beta1"
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
		config *buflintconfig.Config,
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
func RulesForConfig(config *buflintconfig.Config) ([]bufcheck.Rule, error) {
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
		&buflintconfig.Config{
			Use:     internal.AllIDsForVersionSpec(buflintv1beta1.VersionSpec),
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
		&buflintconfig.Config{
			Use:     internal.AllIDsForVersionSpec(buflintv1.VersionSpec),
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
	return internal.AllCategoriesAndIDsForVersionSpec(buflintv1beta1.VersionSpec)
}

// GetAllRulesAndCategoriesV1 returns all rules and categories for v1 as a string slice.
//
// This is used for validation purposes only.
func GetAllRulesAndCategoriesV1() []string {
	return internal.AllCategoriesAndIDsForVersionSpec(buflintv1.VersionSpec)
}

func internalConfigForConfig(config *buflintconfig.Config) (*internal.Config, error) {
	var versionSpec *internal.VersionSpec
	switch config.Version {
	case bufconfig.V1Beta1Version:
		versionSpec = buflintv1beta1.VersionSpec
	case bufconfig.V1Version:
		versionSpec = buflintv1.VersionSpec
	}
	return internal.ConfigBuilder{
		Use:                                  config.Use,
		Except:                               config.Except,
		IgnoreRootPaths:                      config.IgnoreRootPaths,
		IgnoreIDOrCategoryToRootPaths:        config.IgnoreIDOrCategoryToRootPaths,
		AllowCommentIgnores:                  config.AllowCommentIgnores,
		EnumZeroValueSuffix:                  config.EnumZeroValueSuffix,
		RPCAllowSameRequestResponse:          config.RPCAllowSameRequestResponse,
		RPCAllowGoogleProtobufEmptyRequests:  config.RPCAllowGoogleProtobufEmptyRequests,
		RPCAllowGoogleProtobufEmptyResponses: config.RPCAllowGoogleProtobufEmptyResponses,
		ServiceSuffix:                        config.ServiceSuffix,
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
