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
	"fmt"
	"strings"

	"github.com/bufbuild/buf/private/buf/bufcheck/bufcheckserver"
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
	fileVersionToCheckClient map[bufconfig.FileVersion]check.Client
}

func newClient(...ClientOption) (*client, error) {
	// Eventually, we're going to have to make a MultiClient for each of these with the plugin Clients,
	// and that MultiClient may do caching of its own, so we want to keep these static instead of creating
	// them on every lint and breaking  call.
	v1beta1CheckClient, err := check.NewClientForSpec(bufcheckserver.V1Beta1Spec, check.ClientWithCacheRules())
	if err != nil {
		return nil, err
	}
	v1CheckClient, err := check.NewClientForSpec(bufcheckserver.V1Spec, check.ClientWithCacheRules())
	if err != nil {
		return nil, err
	}
	v2CheckClient, err := check.NewClientForSpec(bufcheckserver.V2Spec, check.ClientWithCacheRules())
	if err != nil {
		return nil, err
	}
	return &client{
		fileVersionToCheckClient: map[bufconfig.FileVersion]check.Client{
			bufconfig.FileVersionV1Beta1: v1beta1CheckClient,
			bufconfig.FileVersionV1:      v1CheckClient,
			bufconfig.FileVersionV2:      v2CheckClient,
		},
	}, nil
}

func (c *client) Lint(ctx context.Context, lintConfig bufconfig.LintConfig, image bufimage.Image, _ ...LintOption) error {
	allRules, err := c.AllLintRules(ctx, lintConfig.FileVersion())
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
	checkClient, ok := c.fileVersionToCheckClient[lintConfig.FileVersion()]
	if !ok {
		return fmt.Errorf("unknown FileVersion: %v", lintConfig.FileVersion())
	}
	response, err := checkClient.Check(ctx, request)
	if err != nil {
		return err
	}
	return annotationsToFilteredFileAnnotationSetOrError(config, image, response.Annotations())
}

func (c *client) ConfiguredLintRules(ctx context.Context, lintConfig bufconfig.LintConfig) ([]check.Rule, error) {
	allRules, err := c.AllLintRules(ctx, lintConfig.FileVersion())
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

func (c *client) AllLintRules(ctx context.Context, fileVersion bufconfig.FileVersion) ([]check.Rule, error) {
	checkClient, ok := c.fileVersionToCheckClient[fileVersion]
	if !ok {
		return nil, fmt.Errorf("unknown FileVersion: %v", fileVersion)
	}
	allRules, err := checkClient.ListRules(ctx)
	if err != nil {
		return nil, err
	}
	return rulesForType(allRules, check.RuleTypeLint), nil
}

func (c *client) Breaking(ctx context.Context, breakingConfig bufconfig.BreakingConfig, image bufimage.Image, againstImage bufimage.Image, options ...BreakingOption) error {
	breakingOptions := newBreakingOptions()
	for _, option := range options {
		option(breakingOptions)
	}
	allRules, err := c.AllBreakingRules(ctx, breakingConfig.FileVersion())
	if err != nil {
		return err
	}
	config, err := configForBreakingConfig(breakingConfig, allRules, breakingOptions.excludeImports)
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
	checkClient, ok := c.fileVersionToCheckClient[breakingConfig.FileVersion()]
	if !ok {
		return fmt.Errorf("unknown FileVersion: %v", breakingConfig.FileVersion())
	}
	response, err := checkClient.Check(ctx, request)
	if err != nil {
		return err
	}
	return annotationsToFilteredFileAnnotationSetOrError(config, image, response.Annotations())
}

func (c *client) ConfiguredBreakingRules(ctx context.Context, breakingConfig bufconfig.BreakingConfig) ([]check.Rule, error) {
	allRules, err := c.AllBreakingRules(ctx, breakingConfig.FileVersion())
	if err != nil {
		return nil, err
	}
	config, err := configForBreakingConfig(breakingConfig, allRules, false)
	if err != nil {
		return nil, err
	}
	if len(config.RuleIDs) == 0 {
		return slicesext.Filter(allRules, check.Rule.IsDefault), nil
	}
	return rulesForRuleIDs(allRules, config.RuleIDs), nil
}

func (c *client) AllBreakingRules(ctx context.Context, fileVersion bufconfig.FileVersion) ([]check.Rule, error) {
	checkClient, ok := c.fileVersionToCheckClient[fileVersion]
	if !ok {
		return nil, fmt.Errorf("unknown FileVersion: %v", fileVersion)
	}
	allRules, err := checkClient.ListRules(ctx)
	if err != nil {
		return nil, err
	}
	return rulesForType(allRules, check.RuleTypeBreaking), nil
}

func newBuiltinCheckClientForFileVersion(fileVersion bufconfig.FileVersion) (check.Client, error) {
	switch fileVersion {
	case bufconfig.FileVersionV1Beta1:
		return check.NewClientForSpec(bufcheckserver.V1Beta1Spec, check.ClientWithCacheRules())
	case bufconfig.FileVersionV1:
		return check.NewClientForSpec(bufcheckserver.V1Spec, check.ClientWithCacheRules())
	case bufconfig.FileVersionV2:
		return check.NewClientForSpec(bufcheckserver.V2Spec, check.ClientWithCacheRules())
	default:
		return nil, fmt.Errorf("unknown FileVersion: %v", fileVersion)
	}
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
	if config.ExcludeImports && file.IsImport() {
		return true, nil
	}

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

	// Not a great design, but will never be triggered by lint since this is never set.
	if config.IgnoreUnstablePackages {
		if packageVersion, ok := protoversion.NewPackageVersionForPackage(string(fileDescriptor.Package())); ok {
			if packageVersion.StabilityLevel() != protoversion.StabilityLevelStable {
				return true, nil
			}
		}
	}

	// Not a great design, but will never be triggered by breaking since this is never set.
	// Therefore, never called for an againstLocation  (since lint neve has againstLocations).
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

type lintOptions struct{}

type breakingOptions struct {
	excludeImports bool
}

func newBreakingOptions() *breakingOptions {
	return &breakingOptions{}
}

type clientOptions struct{}
