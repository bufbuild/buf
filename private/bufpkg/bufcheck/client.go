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

	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufcheckserver"
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
	"github.com/bufbuild/buf/private/pkg/tracing"
	"github.com/bufbuild/bufplugin-go/check"
	"go.uber.org/zap"
	"pluginrpc.com/pluginrpc"
)

type client struct {
	logger                          *zap.Logger
	tracer                          tracing.Tracer
	runner                          command.Runner
	stderr                          io.Writer
	fileVersionToDefaultCheckClient map[bufconfig.FileVersion]check.Client
}

func newClient(
	logger *zap.Logger,
	tracer tracing.Tracer,
	runner command.Runner,
	options ...ClientOption,
) (*client, error) {
	clientOptions := newClientOptions()
	for _, option := range options {
		option(clientOptions)
	}
	// We want to keep our check.Clients static for caching instead of creating them on every lint and breaking call.
	v1beta1DefaultCheckClient, err := check.NewClientForSpec(bufcheckserver.V1Beta1Spec, check.ClientWithCacheRulesAndCategories())
	if err != nil {
		return nil, syserror.Wrap(err)
	}
	v1DefaultCheckClient, err := check.NewClientForSpec(bufcheckserver.V1Spec, check.ClientWithCacheRulesAndCategories())
	if err != nil {
		return nil, syserror.Wrap(err)
	}
	v2DefaultCheckClient, err := check.NewClientForSpec(bufcheckserver.V2Spec, check.ClientWithCacheRulesAndCategories())
	if err != nil {
		return nil, syserror.Wrap(err)
	}

	return &client{
		logger: logger,
		tracer: tracer,
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
) (retErr error) {
	ctx, span := c.tracer.Start(ctx, tracing.WithErr(&retErr))
	defer span.End()

	if lintConfig.Disabled() {
		return nil
	}
	lintOptions := newLintOptions()
	for _, option := range options {
		option.applyToLint(lintOptions)
	}
	if err := validatePluginConfigs(lintOptions.pluginConfigs, lintOptions.pluginEnabled); err != nil {
		return err
	}
	allRules, allCategories, err := c.allRulesAndCategories(
		ctx,
		lintConfig.FileVersion(),
		lintOptions.pluginConfigs,
		lintConfig.DisableBuiltin(),
	)
	if err != nil {
		return err
	}
	config, err := configForLintConfig(lintConfig, allRules, allCategories)
	if err != nil {
		return err
	}
	logRulesConfig(c.logger, config.rulesConfig)
	files, err := check.FilesForProtoFiles(imageToProtoFiles(image))
	if err != nil {
		// If a validated Image results in an error, this is a system error.
		return syserror.Wrap(err)
	}
	request, err := check.NewRequest(
		files,
		check.WithRuleIDs(config.RuleIDs...),
		check.WithOptions(config.DefaultOptions),
	)
	if err != nil {
		return err
	}
	multiClient, err := c.getMultiClient(
		lintConfig.FileVersion(),
		lintOptions.pluginConfigs,
		lintConfig.DisableBuiltin(),
		config.DefaultOptions,
	)
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
) (retErr error) {
	ctx, span := c.tracer.Start(ctx, tracing.WithErr(&retErr))
	defer span.End()

	if breakingConfig.Disabled() {
		return nil
	}
	breakingOptions := newBreakingOptions()
	for _, option := range options {
		option.applyToBreaking(breakingOptions)
	}
	if err := validatePluginConfigs(breakingOptions.pluginConfigs, breakingOptions.pluginEnabled); err != nil {
		return err
	}
	allRules, allCategories, err := c.allRulesAndCategories(
		ctx,
		breakingConfig.FileVersion(),
		breakingOptions.pluginConfigs,
		breakingConfig.DisableBuiltin(),
	)
	if err != nil {
		return err
	}
	config, err := configForBreakingConfig(
		breakingConfig,
		allRules,
		allCategories,
		breakingOptions.excludeImports,
	)
	if err != nil {
		return err
	}
	logRulesConfig(c.logger, config.rulesConfig)
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
	multiClient, err := c.getMultiClient(
		breakingConfig.FileVersion(),
		breakingOptions.pluginConfigs,
		breakingConfig.DisableBuiltin(),
		config.DefaultOptions,
	)
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
) (_ []Rule, retErr error) {
	ctx, span := c.tracer.Start(ctx, tracing.WithErr(&retErr))
	defer span.End()

	configuredRulesOptions := newConfiguredRulesOptions()
	for _, option := range options {
		option.applyToConfiguredRules(configuredRulesOptions)
	}
	if err := validatePluginConfigs(configuredRulesOptions.pluginConfigs, configuredRulesOptions.pluginEnabled); err != nil {
		return nil, err
	}
	allRules, allCategories, err := c.allRulesAndCategories(
		ctx,
		checkConfig.FileVersion(),
		configuredRulesOptions.pluginConfigs,
		checkConfig.DisableBuiltin(),
	)
	if err != nil {
		return nil, err
	}
	rulesConfig, err := rulesConfigForCheckConfig(checkConfig, allRules, allCategories, ruleType)
	if err != nil {
		return nil, err
	}
	logRulesConfig(c.logger, rulesConfig)
	return rulesForRuleIDs(allRules, rulesConfig.RuleIDs), nil
}

func (c *client) AllRules(
	ctx context.Context,
	ruleType check.RuleType,
	fileVersion bufconfig.FileVersion,
	options ...AllRulesOption,
) (_ []Rule, retErr error) {
	ctx, span := c.tracer.Start(ctx, tracing.WithErr(&retErr))
	defer span.End()

	allRulesOptions := newAllRulesOptions()
	for _, option := range options {
		option.applyToAllRules(allRulesOptions)
	}
	if err := validatePluginConfigs(allRulesOptions.pluginConfigs, allRulesOptions.pluginEnabled); err != nil {
		return nil, err
	}
	rules, _, err := c.allRulesAndCategories(ctx, fileVersion, allRulesOptions.pluginConfigs, false)
	if err != nil {
		return nil, err
	}
	return rulesForType(rules, ruleType), nil
}

func (c *client) AllCategories(
	ctx context.Context,
	fileVersion bufconfig.FileVersion,
	options ...AllCategoriesOption,
) (_ []Category, retErr error) {
	ctx, span := c.tracer.Start(ctx, tracing.WithErr(&retErr))
	defer span.End()

	allCategoriesOptions := newAllCategoriesOptions()
	for _, option := range options {
		option.applyToAllCategories(allCategoriesOptions)
	}
	if err := validatePluginConfigs(allCategoriesOptions.pluginConfigs, allCategoriesOptions.pluginEnabled); err != nil {
		return nil, err
	}
	_, categories, err := c.allRulesAndCategories(ctx, fileVersion, allCategoriesOptions.pluginConfigs, false)
	return categories, err
}

func (c *client) allRulesAndCategories(
	ctx context.Context,
	fileVersion bufconfig.FileVersion,
	pluginConfigs []bufconfig.PluginConfig,
	disableBuiltin bool,
) ([]Rule, []Category, error) {
	// Just passing through to fulfill all contracts, ie checkClientSpec has non-nil Options.
	// Options are not used here.
	// config struct really just needs refactoring.
	emptyOptions, err := check.NewOptions(nil)
	if err != nil {
		return nil, nil, err
	}
	multiClient, err := c.getMultiClient(fileVersion, pluginConfigs, disableBuiltin, emptyOptions)
	if err != nil {
		return nil, nil, err
	}
	return multiClient.ListRulesAndCategories(ctx)
}

func (c *client) getMultiClient(
	fileVersion bufconfig.FileVersion,
	pluginConfigs []bufconfig.PluginConfig,
	disableBuiltin bool,
	defaultOptions check.Options,
) (*multiClient, error) {
	var checkClientSpecs []*checkClientSpec
	if !disableBuiltin {
		defaultCheckClient, ok := c.fileVersionToDefaultCheckClient[fileVersion]
		if !ok {
			return nil, fmt.Errorf("unknown FileVersion: %v", fileVersion)
		}
		checkClientSpecs = append(
			checkClientSpecs,
			// We do not set PluginName for default check.Clients.
			newCheckClientSpec("", defaultCheckClient, defaultOptions),
		)
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
				// We have to set binary as some things cannot be encoded as JSON.
				// Example: google.protobuf.Timestamps with positive seconds and negative nanos.
				// We still want to send this over the wire to lint.
				//
				// FormatBinary is the default, but we're just being explicit here.
				pluginrpc.ClientWithFormat(pluginrpc.FormatBinary),
			),
			check.ClientWithCacheRulesAndCategories(),
		)
		checkClientSpecs = append(
			checkClientSpecs,
			newCheckClientSpec(pluginConfig.Name(), checkClient, options),
		)
	}
	return newMultiClient(c.logger, checkClientSpecs), nil
}

// TODO: remove this as part of publicly releasing lint/breaking plugins
func validatePluginConfigs(pluginConfigs []bufconfig.PluginConfig, isPluginEnabled bool) error {
	if len(pluginConfigs) > 0 && !isPluginEnabled {
		return fmt.Errorf("custom plugins are not yet supported. For more information, please contact us at https://buf.build/docs/contact")
	}
	return nil
}

func annotationsToFilteredFileAnnotationSetOrError(
	config *config,
	image bufimage.Image,
	annotations []*annotation,
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
	annotations []*annotation,
) ([]*annotation, error) {
	return slicesext.FilterError(
		annotations,
		func(annotation *annotation) (bool, error) {
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
	annotation *annotation,
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
	// If the config says to ignore this specific rule for this path, ignore this location, otherwise we look for other forms of ignores.
	if ignoreRootPaths, ok := config.IgnoreRuleIDToRootPaths[ruleID]; ok && normalpath.MapHasEqualOrContainingPath(ignoreRootPaths, path, normalpath.Relative) {
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
	// Therefore, never called for an againstLocation  (since lint never has againstLocations).
	if config.AllowCommentIgnores && config.CommentIgnorePrefix != "" {
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
					if checkCommentLineForCheckIgnore(line, config.CommentIgnorePrefix, ruleID) {
						return true, nil
					}
				}
			}
		}
	}
	return false, nil
}

// checkCommentLineForCheckIgnore checks that the comment line starts with the configured
// comment ignore prefix, a number of spaces (at least one), the ruleID of the check.
//
// All of the following comments ignore SERVICE_PASCAL_CASE and this rule only:
//
//	// buf:lint:ignore SERVICE_PASCAL_CASE, SERVICE_SUFFIX
//	// buf:lint:ignore SERVICE_PASCAL_CASE
//	// buf:lint:ignore SERVICE_PASCAL_CASE   some other comment
//
// While the following is invalid and a nop
//
//	// buf:lint:ignoreSERVICE_PASCAL_CASE
func checkCommentLineForCheckIgnore(
	commentLine string,
	commentIgnorePrefix string,
	ruleID string,
) bool {
	fullIgnorePrefix := commentIgnorePrefix + " " + ruleID
	return strings.HasPrefix(commentLine, fullIgnorePrefix)
}

type lintOptions struct {
	pluginConfigs []bufconfig.PluginConfig
	// TODO: remove this as part of publicly releasing lint/breaking plugins
	pluginEnabled bool
}

func newLintOptions() *lintOptions {
	return &lintOptions{}
}

type breakingOptions struct {
	pluginConfigs []bufconfig.PluginConfig
	// TODO: remove this as part of publicly releasing lint/breaking plugins
	pluginEnabled  bool
	excludeImports bool
}

func newBreakingOptions() *breakingOptions {
	return &breakingOptions{}
}

type configuredRulesOptions struct {
	pluginConfigs []bufconfig.PluginConfig
	// TODO: remove this as part of publicly releasing lint/breaking plugins
	pluginEnabled bool
}

func newConfiguredRulesOptions() *configuredRulesOptions {
	return &configuredRulesOptions{}
}

type allRulesOptions struct {
	pluginConfigs []bufconfig.PluginConfig
	// TODO: remove this as part of publicly releasing lint/breaking plugins
	pluginEnabled bool
}

func newAllRulesOptions() *allRulesOptions {
	return &allRulesOptions{}
}

type allCategoriesOptions struct {
	pluginConfigs []bufconfig.PluginConfig
	// TODO: remove this as part of publicly releasing lint/breaking plugins
	pluginEnabled bool
}

func newAllCategoriesOptions() *allCategoriesOptions {
	return &allCategoriesOptions{}
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

func (p *pluginConfigsOption) applyToAllCategories(allCategoriesOptions *allCategoriesOptions) {
	allCategoriesOptions.pluginConfigs = append(allCategoriesOptions.pluginConfigs, p.pluginConfigs...)
}

type pluginsEnabledOption struct{}

func (pluginsEnabledOption) applyToLint(lintOptions *lintOptions) {
	lintOptions.pluginEnabled = true
}

func (pluginsEnabledOption) applyToBreaking(breakingOptions *breakingOptions) {
	breakingOptions.pluginEnabled = true
}

func (pluginsEnabledOption) applyToConfiguredRules(configuredRulesOptions *configuredRulesOptions) {
	configuredRulesOptions.pluginEnabled = true
}

func (pluginsEnabledOption) applyToAllRules(allRulesOptions *allRulesOptions) {
	allRulesOptions.pluginEnabled = true
}

func (pluginsEnabledOption) applyToAllCategories(allCategoriesOptions *allCategoriesOptions) {
	allCategoriesOptions.pluginEnabled = true
}
