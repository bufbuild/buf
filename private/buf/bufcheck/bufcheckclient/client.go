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

package bufcheckclient

import (
	"context"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/protosourcepath"
	"github.com/bufbuild/buf/private/pkg/protoversion"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/bufplugin-go/check"
)

type client struct {
	checkClient check.Client
}

func newClient(
	checkClient check.Client,
) *client {
	return &client{
		checkClient: checkClient,
	}
}

func (c *client) Lint(ctx context.Context, lintConfig bufconfig.LintConfig, image bufimage.Image) error {
	allRules, err := c.AllLintRules(ctx)
	if err != nil {
		return err
	}
	config, err := configForLintConfig(lintConfig, allRules)
	if err != nil {
		return err
	}
	files, err := check.FilesForProtoFiles(imageToProtoFiles(image))
	if err != nil {
		return err
	}
	request, err := check.NewRequest(
		files,
		check.WithRuleIDs(config.RuleIDs...),
		check.WithOptions(config.Options),
	)
	if err != nil {
		return err
	}
	response, err := c.checkClient.Check(ctx, request)
	if err != nil {
		return err
	}
	return annotationsToFilteredFileAnnotationSetOrError(config, image, response.Annotations())
}

func (c *client) ConfiguredLintRules(ctx context.Context, lintConfig bufconfig.LintConfig) ([]check.Rule, error) {
	allRules, err := c.AllLintRules(ctx)
	if err != nil {
		return nil, err
	}
	config, err := configForLintConfig(lintConfig, allRules)
	if err != nil {
		return nil, err
	}
	if len(config.RuleIDs) == 0 {
		return slicesext.Filter(allRules, check.Rule.IsDefault), nil
	}
	return rulesForRuleIDs(allRules, config.RuleIDs), nil
}

func (c *client) AllLintRules(ctx context.Context) ([]check.Rule, error) {
	allRules, err := c.checkClient.ListRules(ctx)
	if err != nil {
		return nil, err
	}
	return rulesForType(allRules, check.RuleTypeLint), nil
}

func (c *client) Breaking(ctx context.Context, breakingConfig bufconfig.BreakingConfig, image bufimage.Image, againstImage bufimage.Image) error {

	allRules, err := c.AllBreakingRules(ctx)
	if err != nil {
		return err
	}
	config, err := configForBreakingConfig(breakingConfig, allRules)
	if err != nil {
		return err
	}
	files, err := check.FilesForProtoFiles(imageToProtoFiles(image))
	if err != nil {
		return err
	}
	againstFiles, err := check.FilesForProtoFiles(imageToProtoFiles(againstImage))
	if err != nil {
		return err
	}
	request, err := check.NewRequest(
		files,
		check.WithRuleIDs(config.RuleIDs...),
		check.WithAgainstFiles(againstFiles),
		check.WithOptions(config.Options),
	)
	if err != nil {
		return err
	}
	response, err := c.checkClient.Check(ctx, request)
	if err != nil {
		return err
	}
	return annotationsToFilteredFileAnnotationSetOrError(config, image, response.Annotations())
}

func (c *client) ConfiguredBreakingRules(ctx context.Context, breakingConfig bufconfig.BreakingConfig) ([]check.Rule, error) {
	allRules, err := c.AllBreakingRules(ctx)
	if err != nil {
		return nil, err
	}
	config, err := configForBreakingConfig(breakingConfig, allRules)
	if err != nil {
		return nil, err
	}
	if len(config.RuleIDs) == 0 {
		return slicesext.Filter(allRules, check.Rule.IsDefault), nil
	}
	return rulesForRuleIDs(allRules, config.RuleIDs), nil
}

func (c *client) AllBreakingRules(ctx context.Context) ([]check.Rule, error) {
	allRules, err := c.checkClient.ListRules(ctx)
	if err != nil {
		return nil, err
	}
	return rulesForType(allRules, check.RuleTypeBreaking), nil
}

func annotationsToFilteredFileAnnotationSetOrError(
	config *config,
	image bufimage.Image,
	annotations []check.Annotation,
) error {
	if len(annotations) == 0 {
		return nil
	}
	annotations, err := filterAnnotations(config, annotations)
	if err != nil {
		return err
	}
	if len(annotations) == 0 {
		return nil
	}
	// Note that NewFileAnnotationSet does its own sorting and deduplication.
	// The bufplugin SDK does this as well, but we don't need to worry about the sort
	// order being different.
	return bufanalysis.NewFileAnnotationSet(
		annotationsToFileAnnotations(
			imageToPathToExternalPath(
				image,
			),
			annotations,
		)...,
	)
}

func filterAnnotations(
	config *config,
	annotations []check.Annotation,
) ([]check.Annotation, error) {
	return slicesext.FilterError(
		annotations,
		func(annotation check.Annotation) (bool, error) {
			return ignoreAnnotation(config, annotation)
		},
	)
}

func ignoreAnnotation(
	config *config,
	annotation check.Annotation,
) (bool, error) {
	if location := annotation.Location(); location != nil {
		ignore, err := ignoreLocation(config, annotation.RuleID(), location)
		if err != nil {
			return false, err
		}
		if ignore {
			return true, nil
		}
	}
	// TODO: Is this right? Does this properly encapsulate old extraIgnoreDescriptors logic?
	if againstLocation := annotation.AgainstLocation(); againstLocation != nil {
		return ignoreLocation(config, annotation.RuleID(), againstLocation)

	}
	return false, nil
}

func ignoreLocation(
	config *config,
	ruleID string,
	location check.Location,
) (bool, error) {
	file := location.File()
	fileDescriptor := file.FileDescriptor()
	path := fileDescriptor.Path()
	if normalpath.MapHasEqualOrContainingPath(config.IgnoreRootPaths, path, normalpath.Relative) {
		return true, nil
	}
	ignoreRootPaths, ok := config.IgnoreIDToRootPaths[ruleID]
	if !ok {
		return false, nil
	}
	if normalpath.MapHasEqualOrContainingPath(ignoreRootPaths, path, normalpath.Relative) {
		return true, nil
	}

	if config.IgnoreUnstablePackages {
		if packageVersion, ok := protoversion.NewPackageVersionForPackage(string(fileDescriptor.Package())); ok {
			if packageVersion.StabilityLevel() != protoversion.StabilityLevelStable {
				return true, nil
			}
		}
	}

	if config.CommentIgnorePrefix != "" {
		sourcePath := location.SourcePath()
		if len(sourcePath) == 0 {
			return false, nil
		}
		associatedSourcePaths, err := protosourcepath.GetAssociatedSourcePaths(sourcePath)
		if err != nil {
			return false, err
		}
		sourceLocations := fileDescriptor.SourceLocations()
		for _, associatedSourcePath := range associatedSourcePaths {
			sourceLocation := sourceLocations.ByPath(associatedSourcePath)
			if leadingComments := sourceLocation.LeadingComments; leadingComments != "" {
				for _, line := range stringutil.SplitTrimLinesNoEmpty(leadingComments) {
					if strings.HasPrefix(line, config.CommentIgnorePrefix) {
						return true, nil
					}
				}
			}
		}
	}

	return false, nil
}
