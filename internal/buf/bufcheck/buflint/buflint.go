// Copyright 2020 Buf Technologies Inc.
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

	"github.com/bufbuild/buf/internal/buf/bufanalysis"
	"github.com/bufbuild/buf/internal/buf/bufcheck"
	"github.com/bufbuild/buf/internal/buf/bufcheck/internal"
	"github.com/bufbuild/buf/internal/buf/bufimage"
	"github.com/bufbuild/buf/internal/buf/bufsrc"
	"go.uber.org/zap"
)

// Handler handles the main lint functionality.
type Handler interface {
	// LintCheck runs the lint checks.
	//
	// The image should have source code info for this to work properly.
	//
	// Images should be filtered with regards to imports before passing to this function.
	//
	// FileAnnotations will use the image file paths, if these should be relative, use
	// FixFileAnnotationPaths.
	LintCheck(
		ctx context.Context,
		lintConfig *Config,
		image bufimage.Image,
	) ([]bufanalysis.FileAnnotation, error)
}

// NewHandler returns a new Handler.
func NewHandler(
	logger *zap.Logger,
	lintRunner Runner,
) Handler {
	return newHandler(
		logger,
		lintRunner,
	)
}

// Checker is a checker.
type Checker interface {
	bufcheck.Checker

	internalLint() *internal.Checker
}

// Runner is a runner.
type Runner interface {
	// Check runs lint checkers, returning a system error if any system error occurs
	// or returning the FileAnnotations otherwise.
	//
	// FileAnnotations will be sorted, but Paths will not have the roots as a prefix, instead
	// they will be relative to the roots. This should be fixed for linter outputs if image
	// mode is not used.
	Check(context.Context, *Config, []bufsrc.File) ([]bufanalysis.FileAnnotation, error)
}

// NewRunner returns a new Runner.
func NewRunner(logger *zap.Logger) Runner {
	return newRunner(logger)
}

// Config is the check config.
type Config struct {
	// Checkers are the lint checkers to run.
	//
	// Checkers will be sorted by first categories, then id when Configs are
	// created from this package, i.e. created wth ConfigBuilder.NewConfig.
	Checkers            []Checker
	IgnoreIDToRootPaths map[string]map[string]struct{}
	IgnoreRootPaths     map[string]struct{}
}

// GetCheckers returns the checkers for the given categories.
//
// If categories is empty, this returns all checkers as bufcheck.Checkers.
//
// Should only be used for printing.
func (c *Config) GetCheckers(categories ...string) ([]bufcheck.Checker, error) {
	return checkersToBufcheckCheckers(c.Checkers, categories)
}

// ConfigBuilder is a config builder.
type ConfigBuilder struct {
	Use                                  []string
	Except                               []string
	IgnoreIDOrCategoryToRootPaths        map[string][]string
	IgnoreRootPaths                      []string
	EnumZeroValueSuffix                  string
	RPCAllowSameRequestResponse          bool
	RPCAllowGoogleProtobufEmptyRequests  bool
	RPCAllowGoogleProtobufEmptyResponses bool
	ServiceSuffix                        string
}

// NewConfig returns a new Config.
func (b ConfigBuilder) NewConfig() (*Config, error) {
	internalConfig, err := internal.ConfigBuilder{
		Use:                                  b.Use,
		Except:                               b.Except,
		IgnoreIDOrCategoryToRootPaths:        b.IgnoreIDOrCategoryToRootPaths,
		IgnoreRootPaths:                      b.IgnoreRootPaths,
		EnumZeroValueSuffix:                  b.EnumZeroValueSuffix,
		RPCAllowSameRequestResponse:          b.RPCAllowSameRequestResponse,
		RPCAllowGoogleProtobufEmptyRequests:  b.RPCAllowGoogleProtobufEmptyRequests,
		RPCAllowGoogleProtobufEmptyResponses: b.RPCAllowGoogleProtobufEmptyResponses,
		ServiceSuffix:                        b.ServiceSuffix,
	}.NewConfig(
		v1CheckerBuilders,
		v1IDToCategories,
		v1DefaultCategories,
	)
	if err != nil {
		return nil, err
	}
	return internalConfigToConfig(internalConfig), nil
}

// GetAllCheckers gets all known checkers for the given categories.
//
// If categories is empty, this returns all checkers as bufcheck.Checkers.
//
// Should only be used for printing.
func GetAllCheckers(categories ...string) ([]bufcheck.Checker, error) {
	config, err := ConfigBuilder{
		Use: v1AllCategories,
	}.NewConfig()
	if err != nil {
		return nil, err
	}
	return checkersToBufcheckCheckers(config.Checkers, categories)
}

func internalConfigToConfig(internalConfig *internal.Config) *Config {
	return &Config{
		Checkers:            internalCheckersToCheckers(internalConfig.Checkers),
		IgnoreIDToRootPaths: internalConfig.IgnoreIDToRootPaths,
		IgnoreRootPaths:     internalConfig.IgnoreRootPaths,
	}
}

func configToInternalConfig(config *Config) *internal.Config {
	return &internal.Config{
		Checkers:            checkersToInternalCheckers(config.Checkers),
		IgnoreIDToRootPaths: config.IgnoreIDToRootPaths,
		IgnoreRootPaths:     config.IgnoreRootPaths,
	}
}

func checkersToBufcheckCheckers(checkers []Checker, categories []string) ([]bufcheck.Checker, error) {
	if checkers == nil {
		return nil, nil
	}
	s := make([]bufcheck.Checker, len(checkers))
	for i, e := range checkers {
		s[i] = e
	}
	if len(categories) == 0 {
		return s, nil
	}
	return internal.GetCheckersForCategories(s, v1AllCategories, categories)
}
