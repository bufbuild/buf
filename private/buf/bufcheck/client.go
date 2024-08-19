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

package bufcheck

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/bufbuild/buf/private/buf/bufcheck/internal/bufcheckserver"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/pluginrpcutil"
	"github.com/bufbuild/buf/private/pkg/protosourcepath"
	"github.com/bufbuild/buf/private/pkg/protoversion"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/bufplugin-go/check"
	"github.com/bufbuild/pluginrpc-go"
)

type client struct {
	runner                          command.Runner
	stderr                          io.Writer
	fileVersionToDefaultCheckClient map[bufconfig.FileVersion]check.Client
}

func newClient(runner command.Runner, options ...ClientOption) (*client, error) {
	clientOptions := newClientOptions()
	for _, option := range options {
		option(clientOptions)
	}
	// We want to keep our check.Clients static for caching instead of creating them on every lint and breaking call.
	v1beta1DefaultCheckClient, err := check.NewClientForSpec(bufcheckserver.V1Beta1Spec, check.ClientWithCacheRules())
	if err != nil {
		return nil, syserror.Wrap(err)
	}
	v1DefaultCheckClient, err := check.NewClientForSpec(bufcheckserver.V1Spec, check.ClientWithCacheRules())
	if err != nil {
		return nil, syserror.Wrap(err)
	}
	v2DefaultCheckClient, err := check.NewClientForSpec(bufcheckserver.V2Spec, check.ClientWithCacheRules())
	if err != nil {
		return nil, syserror.Wrap(err)
	}

	return &client{
		runner: runner,
		stderr: clientOptions.stderr,
		fileVersionToDefaultCheckClient: map[bufconfig.FileVersion]check.Client{
			bufconfig.FileVersionV1Beta1: v1beta1DefaultCheckClient,
			bufconfig.FileVersionV1:      v1DefaultCheckClient,
			bufconfig.FileVersionV2:      v2DefaultCheckClient,
		},
	}, nil
}

func (c *client) Lint(
	ctx context.Context,
	lintConfig bufconfig.LintConfig,
	image bufimage.Image,
	options ...LintOption,
) error {
	lintOptions := newLintOptions()
	for _, option := range options {
		option.applyToLint(lintOptions)
	}
	allRules, err := c.allRules(ctx, check.RuleTypeLint, lintConfig.FileVersion(), lintOptions.pluginConfigs)
	if err != nil {
		return err
	}
	config, err := configForLintConfig(lintConfig, allRules)
	if err != nil {
		return err
	}
	files, err := check.FilesForProtoFiles(imageToProtoFiles(image))
	if err != nil {
		// If a validated Image results in an error, this is a system error.
		return syserror.Wrap(err)
	}
	request, err := check.NewRequest(
		files,
		// Note that if we did not set Use or Except in the buf.yaml config, this will be empty,
		// which is correct - this will result in the default Rules being used per the bufplugin-go API.
		check.WithRuleIDs(config.RuleIDs...),
		check.WithOptions(config.DefaultOptions),
	)
	if err != nil {
		return err
	}
	multiClient, err := c.getMultiClient(lintConfig.FileVersion(), lintOptions.pluginConfigs, config.DefaultOptions)
	if err != nil {
		return err
	}
	annotations, err := multiClient.Check(ctx, request)
	if err != nil {
		return err
	}
	return annotationsToFilteredFileAnnotationSetOrError(config, image, annotations)
}

func (c *client) Breaking(
	ctx context.Context,
	breakingConfig bufconfig.BreakingConfig,
	image bufimage.Image,
	againstImage bufimage.Image,
	options ...BreakingOption,
) error {
	breakingOptions := newBreakingOptions()
	for _, option := range options {
		option.applyToBreaking(breakingOptions)
	}
	allRules, err := c.allRules(ctx, check.RuleTypeBreaking, breakingConfig.FileVersion(), breakingOptions.pluginConfigs)
	if err != nil {
		return err
	}
	config, err := configForBreakingConfig(breakingConfig, allRules, breakingOptions.excludeImports)
	if err != nil {
		return err
	}
	files, err := check.FilesForProtoFiles(imageToProtoFiles(image))
	if err != nil {
		// If a validated Image results in an error, this is a system error.
		return syserror.Wrap(err)
	}
	againstFiles, err := check.FilesForProtoFiles(imageToProtoFiles(againstImage))
	if err != nil {
		// If a validated Image results in an error, this is a system error.
		return syserror.Wrap(err)
	}
	request, err := check.NewRequest(
		files,
		check.WithRuleIDs(config.RuleIDs...),
		check.WithAgainstFiles(againstFiles),
		check.WithOptions(config.DefaultOptions),
	)
	if err != nil {
		return err
	}
	multiClient, err := c.getMultiClient(breakingConfig.FileVersion(), breakingOptions.pluginConfigs, config.DefaultOptions)
	if err != nil {
		return err
	}
	annotations, err := multiClient.Check(ctx, request)
	if err != nil {
		return err
	}
	return annotationsToFilteredFileAnnotationSetOrError(config, image, annotations)
}

func (c *client) ConfiguredRules(
	ctx context.Context,
	ruleType check.RuleType,
	checkConfig bufconfig.CheckConfig,
	options ...ConfiguredRulesOption,
) ([]check.Rule, error) {
	configuredRulesOptions := newConfiguredRulesOptions()
	for _, option := range options {
		option.applyToConfiguredRules(configuredRulesOptions)
	}
	allRules, err := c.allRules(ctx, ruleType, checkConfig.FileVersion(), configuredRulesOptions.pluginConfigs)
	if err != nil {
		return nil, err
	}
	config, err := configForCheckConfig(checkConfig, allRules)
	if err != nil {
		return nil, err
	}
	if len(config.RuleIDs) == 0 {
		return slicesext.Filter(allRules, check.Rule.IsDefault), nil
	}
	return rulesForRuleIDs(allRules, config.RuleIDs), nil
}

func (c *client) AllRules(
	ctx context.Context,
	ruleType check.RuleType,
	fileVersion bufconfig.FileVersion,
	options ...AllRulesOption,
) ([]check.Rule, error) {
	allRulesOptions := newAllRulesOptions()
	for _, option := range options {
		option.applyToAllRules(allRulesOptions)
	}
	return c.allRules(ctx, ruleType, fileVersion, allRulesOptions.pluginConfigs)
}

func (c *client) allRules(
	ctx context.Context,
	ruleType check.RuleType,
	fileVersion bufconfig.FileVersion,
	pluginConfigs []bufconfig.PluginConfig,
) ([]check.Rule, error) {
	// Just passing through to fufill all contracts, ie checkClientSpec has non-nil Options.
	// Options are not used here.
	// config struct really just needs refactoring.
	emptyOptions, err := check.NewOptions(nil)
	if err != nil {
		return nil, err
	}
	multiClient, err := c.getMultiClient(fileVersion, pluginConfigs, emptyOptions)
	if err != nil {
		return nil, err
	}
	allRules, err := multiClient.ListRules(ctx)
	if err != nil {
		return nil, err
	}
	return rulesForType(allRules, ruleType), nil
}

func (c *client) getMultiClient(
	fileVersion bufconfig.FileVersion,
	pluginConfigs []bufconfig.PluginConfig,
	defaultOptions check.Options,
) (*multiClient, error) {
	defaultCheckClient, ok := c.fileVersionToDefaultCheckClient[fileVersion]
	if !ok {
		return nil, fmt.Errorf("unknown FileVersion: %v", fileVersion)
	}
	checkClientSpecs := []*checkClientSpec{
		newCheckClientSpec(defaultCheckClient, defaultOptions),
	}
	for _, pluginConfig := range pluginConfigs {
		if pluginConfig.Type() != bufconfig.PluginConfigTypeLocal {
			return nil, syserror.New("we only handle local plugins for now with lint and breaking")
		}
		options, err := check.NewOptions(pluginConfig.Options())
		if err != nil {
			return nil, fmt.Errorf("could not parse options for plugin %q: %w", pluginConfig.Name(), err)
		}
		pluginPath := pluginConfig.Path()
		checkClient := check.NewClient(
			pluginrpc.NewClient(
				pluginrpcutil.NewRunner(
					c.runner,
					// We know that Path is of at least length 1.
					pluginPath[0],
					pluginrpcutil.RunnerWithArgs(pluginPath[1:]...),
				),
				pluginrpc.ClientWithStderr(c.stderr),
			),
			check.ClientWithCacheRules(),
		)
		if err != nil {
			return nil, err
		}
		checkClientSpecs = append(
			checkClientSpecs,
			newCheckClientSpec(checkClient, options),
		)
	}
	return newMultiClient(checkClientSpecs), nil
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
			ignore, err := ignoreAnnotation(config, annotation)
			if err != nil {
				return false, err
			}
			return !ignore, nil
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

type lintOptions struct {
	pluginConfigs []bufconfig.PluginConfig
}

func newLintOptions() *lintOptions {
	return &lintOptions{}
}

type breakingOptions struct {
	pluginConfigs  []bufconfig.PluginConfig
	excludeImports bool
}

func newBreakingOptions() *breakingOptions {
	return &breakingOptions{}
}

type configuredRulesOptions struct {
	pluginConfigs []bufconfig.PluginConfig
}

func newConfiguredRulesOptions() *configuredRulesOptions {
	return &configuredRulesOptions{}
}

type allRulesOptions struct {
	pluginConfigs []bufconfig.PluginConfig
}

func newAllRulesOptions() *allRulesOptions {
	return &allRulesOptions{}
}

type clientOptions struct {
	stderr io.Writer
}

func newClientOptions() *clientOptions {
	return &clientOptions{}
}

type excludeImportsOption struct{}

func (e *excludeImportsOption) applyToBreaking(breakingOptions *breakingOptions) {
	breakingOptions.excludeImports = true
}

type pluginConfigsOption struct {
	pluginConfigs []bufconfig.PluginConfig
}

func (p *pluginConfigsOption) applyToLint(lintOptions *lintOptions) {
	lintOptions.pluginConfigs = append(lintOptions.pluginConfigs, p.pluginConfigs...)
}

func (p *pluginConfigsOption) applyToBreaking(breakingOptions *breakingOptions) {
	breakingOptions.pluginConfigs = append(breakingOptions.pluginConfigs, p.pluginConfigs...)
}

func (p *pluginConfigsOption) applyToConfiguredRules(configuredRulesOptions *configuredRulesOptions) {
	configuredRulesOptions.pluginConfigs = append(configuredRulesOptions.pluginConfigs, p.pluginConfigs...)
}

func (p *pluginConfigsOption) applyToAllRules(allRulesOptions *allRulesOptions) {
	allRulesOptions.pluginConfigs = append(allRulesOptions.pluginConfigs, p.pluginConfigs...)
}
