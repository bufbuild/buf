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
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"

	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/buflint/internal/buflintv1"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/buflint/internal/buflintv1beta1"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/buflint/internal/buflintv2"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/internal"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/tracing"
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
	//
	// An error of type bufanalysis.FileAnnotationSet will be returned on breaking failure.
	Check(
		ctx context.Context,
		config bufconfig.LintConfig,
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
func RulesForConfig(config bufconfig.LintConfig) ([]bufcheck.Rule, error) {
	internalConfig, err := internalConfigForConfig(config, true)
	if err != nil {
		return nil, err
	}
	return rulesForInternalRules(internalConfig.Rules), nil
}

// GetAllRulesV1Beta1 gets all known rules.
//
// Should only be used for printing.
func GetAllRulesV1Beta1() ([]bufcheck.Rule, error) {
	lintConfig, err := newLintConfigForVersionSpec(buflintv1beta1.VersionSpec)
	if err != nil {
		return nil, err
	}
	internalConfig, err := internalConfigForConfig(lintConfig, false)
	if err != nil {
		return nil, err
	}
	return rulesForInternalRules(internalConfig.Rules), nil
}

// GetAllRulesV1 gets all known rules.
//
// Should only be used for printing.
func GetAllRulesV1() ([]bufcheck.Rule, error) {
	lintConfig, err := newLintConfigForVersionSpec(buflintv1.VersionSpec)
	if err != nil {
		return nil, err
	}
	internalConfig, err := internalConfigForConfig(lintConfig, false)
	if err != nil {
		return nil, err
	}
	return rulesForInternalRules(internalConfig.Rules), nil
}

// GetAllRulesV2 gets all known rules.
//
// Should only be used for printing.
func GetAllRulesV2() ([]bufcheck.Rule, error) {
	lintConfig, err := newLintConfigForVersionSpec(buflintv2.VersionSpec)
	if err != nil {
		return nil, err
	}
	internalConfig, err := internalConfigForConfig(lintConfig, false)
	if err != nil {
		return nil, err
	}
	return rulesForInternalRules(internalConfig.Rules), nil
}

// GetRelevantDeprecations gets deprecation information for the given
// version. The map is from deprecated rule IDs to zero or more replacement
// rule IDs.
func GetRelevantDeprecations(fileVersion bufconfig.FileVersion) (map[string][]string, error) {
	var versionSpec *internal.VersionSpec
	switch fileVersion {
	case bufconfig.FileVersionV1Beta1:
		versionSpec = buflintv1beta1.VersionSpec
	case bufconfig.FileVersionV1:
		versionSpec = buflintv1.VersionSpec
	case bufconfig.FileVersionV2:
		versionSpec = buflintv2.VersionSpec
	default:
		return nil, fmt.Errorf("unknown FileVersion: %v", fileVersion)
	}
	return internal.RelevantDeprecationsForVersionSpec(versionSpec)
}

// PrintFileAnnotationSetConfigIgnoreYAMLV1 prints the FileAnnotationSet to the Writer
// for the config-ignore-yaml format.
//
// TODO FUTURE: This is messed.
func PrintFileAnnotationSetConfigIgnoreYAMLV1(
	writer io.Writer,
	fileAnnotationSet bufanalysis.FileAnnotationSet,
) error {
	ignoreIDToPathMap := make(map[string]map[string]struct{})
	for _, fileAnnotation := range fileAnnotationSet.FileAnnotations() {
		fileInfo := fileAnnotation.FileInfo()
		if fileInfo == nil || fileAnnotation.Type() == "" {
			continue
		}
		pathMap, ok := ignoreIDToPathMap[fileAnnotation.Type()]
		if !ok {
			pathMap = make(map[string]struct{})
			ignoreIDToPathMap[fileAnnotation.Type()] = pathMap
		}
		pathMap[fileInfo.Path()] = struct{}{}
	}
	if len(ignoreIDToPathMap) == 0 {
		return nil
	}

	sortedIgnoreIDs := make([]string, 0, len(ignoreIDToPathMap))
	ignoreIDToSortedPaths := make(map[string][]string, len(ignoreIDToPathMap))
	for id, pathMap := range ignoreIDToPathMap {
		sortedIgnoreIDs = append(sortedIgnoreIDs, id)
		paths := make([]string, 0, len(pathMap))
		for path := range pathMap {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		ignoreIDToSortedPaths[id] = paths
	}
	sort.Strings(sortedIgnoreIDs)

	buffer := bytes.NewBuffer(nil)
	_, _ = buffer.WriteString(`version: v1
lint:
  ignore_only:
`)
	for _, id := range sortedIgnoreIDs {
		_, _ = buffer.WriteString("    ")
		_, _ = buffer.WriteString(id)
		_, _ = buffer.WriteString(":\n")
		for _, rootPath := range ignoreIDToSortedPaths[id] {
			_, _ = buffer.WriteString("      - ")
			_, _ = buffer.WriteString(rootPath)
			_, _ = buffer.WriteString("\n")
		}
	}
	_, err := writer.Write(buffer.Bytes())
	return err
}

func internalConfigForConfig(config bufconfig.LintConfig, transformDeprecated bool) (*internal.Config, error) {
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

func newLintConfigForVersionSpec(versionSpec *internal.VersionSpec) (bufconfig.LintConfig, error) {
	ids, err := internal.AllIDsForVersionSpec(versionSpec, true)
	if err != nil {
		return nil, err
	}
	return bufconfig.NewLintConfig(
		bufconfig.NewEnabledCheckConfigForUseIDsAndCategories(
			versionSpec.FileVersion,
			ids,
		),
		"",
		false,
		false,
		false,
		"",
		false,
	), nil
}
