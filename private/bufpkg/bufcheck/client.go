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
	"log/slog"
	"strings"

	"buf.build/go/bufplugin/check"
	"buf.build/go/bufplugin/descriptor"
	"buf.build/go/bufplugin/option"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufcheckserver"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/protosourcepath"
	"github.com/bufbuild/buf/private/pkg/protoversion"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/slogext"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"pluginrpc.com/pluginrpc"
)

type client struct {
	logger                          *slog.Logger
	runnerProvider                  RunnerProvider
	stderr                          io.Writer
	fileVersionToDefaultCheckClient map[bufconfig.FileVersion]check.Client
}

func newClient(
	logger *slog.Logger,
	runnerProvider RunnerProvider,
	options ...ClientOption,
) (*client, error) {
	clientOptions := newClientOptions()
	for _, option := range options {
		option(clientOptions)
	}
	// We want to keep our check.Clients static for caching instead of creating them on every lint and breaking call.
	v1beta1DefaultCheckClient, err := check.NewClientForSpec(bufcheckserver.V1Beta1Spec, check.ClientWithCaching())
	if err != nil {
		return nil, syserror.Wrap(err)
	}
	v1DefaultCheckClient, err := check.NewClientForSpec(bufcheckserver.V1Spec, check.ClientWithCaching())
	if err != nil {
		return nil, syserror.Wrap(err)
	}
	v2DefaultCheckClient, err := check.NewClientForSpec(bufcheckserver.V2Spec, check.ClientWithCaching())
	if err != nil {
		return nil, syserror.Wrap(err)
	}

	return &client{
		logger:         logger,
		runnerProvider: runnerProvider,
		stderr:         clientOptions.stderr,
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
	defer slogext.DebugProfile(c.logger)()

	if lintConfig.Disabled() {
		return nil
	}
	lintOptions := newLintOptions()
	for _, option := range options {
		option.applyToLint(lintOptions)
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
	files, err := descriptor.FileDescriptorsForProtoFileDescriptors(imageToProtoFileDescriptors(image))
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
) error {
	defer slogext.DebugProfile(c.logger)()

	if breakingConfig.Disabled() {
		return nil
	}
	breakingOptions := newBreakingOptions()
	for _, option := range options {
		option.applyToBreaking(breakingOptions)
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
	fileDescriptors, err := descriptor.FileDescriptorsForProtoFileDescriptors(imageToProtoFileDescriptors(image))
	if err != nil {
		// If a validated Image results in an error, this is a system error.
		return syserror.Wrap(err)
	}
	againstFileDescriptors, err := descriptor.FileDescriptorsForProtoFileDescriptors(imageToProtoFileDescriptors(againstImage))
	if err != nil {
		// If a validated Image results in an error, this is a system error.
		return syserror.Wrap(err)
	}
	request, err := check.NewRequest(
		fileDescriptors,
		check.WithRuleIDs(config.RuleIDs...),
		check.WithAgainstFileDescriptors(againstFileDescriptors),
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
) ([]Rule, error) {
	defer slogext.DebugProfile(c.logger)()

	configuredRulesOptions := newConfiguredRulesOptions()
	for _, option := range options {
		option.applyToConfiguredRules(configuredRulesOptions)
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
) ([]Rule, error) {
	defer slogext.DebugProfile(c.logger)()

	allRulesOptions := newAllRulesOptions()
	for _, option := range options {
		option.applyToAllRules(allRulesOptions)
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
) ([]Category, error) {
	defer slogext.DebugProfile(c.logger)()

	allCategoriesOptions := newAllCategoriesOptions()
	for _, option := range options {
		option.applyToAllCategories(allCategoriesOptions)
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
	multiClient, err := c.getMultiClient(fileVersion, pluginConfigs, disableBuiltin, option.EmptyOptions)
	if err != nil {
		return nil, nil, err
	}
	return multiClient.ListRulesAndCategories(ctx)
}

func (c *client) getMultiClient(
	fileVersion bufconfig.FileVersion,
	pluginConfigs []bufconfig.PluginConfig,
	disableBuiltin bool,
	defaultOptions option.Options,
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
		options, err := option.NewOptions(pluginConfig.Options())
		if err != nil {
			return nil, fmt.Errorf("could not parse options for plugin %q: %w", pluginConfig.Name(), err)
		}
		runner, err := c.runnerProvider.NewRunner(pluginConfig)
		if err != nil {
			return nil, fmt.Errorf("could not create runner for plugin %q: %w", pluginConfig.Name(), err)
		}
		checkClient := check.NewClient(
			pluginrpc.NewClient(
				runner,
				pluginrpc.ClientWithStderr(c.stderr),
				// We have to set binary as some things cannot be encoded as JSON.
				// Example: google.protobuf.Timestamps with positive seconds and negative nanos.
				// We still want to send this over the wire to lint.
				//
				// FormatBinary is the default, but we're just being explicit here.
				pluginrpc.ClientWithFormat(pluginrpc.FormatBinary),
			),
			check.ClientWithCaching(),
		)
		checkClientSpecs = append(
			checkClientSpecs,
			newCheckClientSpec(pluginConfig.Name(), checkClient, options),
		)
	}
	return newMultiClient(c.logger, checkClientSpecs), nil
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
	if fileLocation := annotation.FileLocation(); fileLocation != nil {
		ignore, err := ignoreFileLocation(config, annotation.RuleID(), fileLocation)
		if err != nil {
			return false, err
		}
		if ignore {
			return true, nil
		}
	}
	if againstFileLocation := annotation.AgainstFileLocation(); againstFileLocation != nil {
		return ignoreFileLocation(config, annotation.RuleID(), againstFileLocation)
	}
	return false, nil
}

func ignoreFileLocation(
	config *config,
	ruleID string,
	fileLocation descriptor.FileLocation,
) (bool, error) {
	fileDescriptor := fileLocation.FileDescriptor()
	if config.ExcludeImports && fileDescriptor.IsImport() {
		return true, nil
	}

	protoreflectFileDescriptor := fileDescriptor.ProtoreflectFileDescriptor()
	path := protoreflectFileDescriptor.Path()
	if normalpath.MapHasEqualOrContainingPath(config.IgnoreRootPaths, path, normalpath.Relative) {
		return true, nil
	}
	// If the config says to ignore this specific rule for this path, ignore this location, otherwise we look for other forms of ignores.
	if ignoreRootPaths, ok := config.IgnoreRuleIDToRootPaths[ruleID]; ok && normalpath.MapHasEqualOrContainingPath(ignoreRootPaths, path, normalpath.Relative) {
		return true, nil
	}

	// Not a great design, but will never be triggered by lint since this is never set.
	if config.IgnoreUnstablePackages {
		if packageVersion, ok := protoversion.NewPackageVersionForPackage(string(protoreflectFileDescriptor.Package())); ok {
			if packageVersion.StabilityLevel() != protoversion.StabilityLevelStable {
				return true, nil
			}
		}
	}

	// Not a great design, but will never be triggered by breaking since this is never set.
	// Therefore, never called for an againstLocation  (since lint never has againstLocations).
	if config.AllowCommentIgnores && config.CommentIgnorePrefix != "" {
		sourcePath := fileLocation.SourcePath()
		if len(sourcePath) == 0 {
			return false, nil
		}
		associatedSourcePaths, err := protosourcepath.GetAssociatedSourcePaths(sourcePath)
		if err != nil {
			return false, err
		}
		sourceLocations := protoreflectFileDescriptor.SourceLocations()
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
// comment ignore prefix, a space and the ruleID of the check.
//
// All of the following comments are valid, ignoring SERVICE_PASCAL_CASE and this rule only:
//
//	// buf:lint:ignore SERVICE_PASCAL_CASE, SERVICE_SUFFIX (only SERVICE_PASCAL_CASE is ignored)
//	// buf:lint:ignore SERVICE_PASCAL_CASE
//	// buf:lint:ignore SERVICE_PASCAL_CASEsome other comment
//	// buf:lint:ignore SERVICE_PASCAL_CASE some other comment
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

type allCategoriesOptions struct {
	pluginConfigs []bufconfig.PluginConfig
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
