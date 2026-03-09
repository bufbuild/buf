// Copyright 2020-2025 Buf Technologies, Inc.
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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"strings"

	"buf.build/go/bufplugin/check"
	"buf.build/go/bufplugin/descriptor"
	"buf.build/go/bufplugin/option"
	"buf.build/go/standard/xlog/xslog"
	"buf.build/go/standard/xslices"
	"buf.build/go/standard/xstrings"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufcheckserver"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin"
	"github.com/bufbuild/buf/private/bufpkg/bufpolicy"
	"github.com/bufbuild/buf/private/bufpkg/bufpolicy/bufpolicyconfig"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/protosourcepath"
	"github.com/bufbuild/buf/private/pkg/protoversion"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/google/uuid"
	"google.golang.org/protobuf/reflect/protoreflect"
	"pluginrpc.com/pluginrpc"
)

type client struct {
	logger                          *slog.Logger
	stderr                          io.Writer
	fileVersionToDefaultCheckClient map[bufconfig.FileVersion]check.Client
	runnerProvider                  RunnerProvider
	pluginReadFile                  func(string) ([]byte, error)
	pluginKeyProvider               bufplugin.PluginKeyProvider
	pluginDataProvider              bufplugin.PluginDataProvider
	policyReadFile                  func(string) ([]byte, error)
	policyKeyProvider               bufpolicy.PolicyKeyProvider
	policyDataProvider              bufpolicy.PolicyDataProvider
	policyPluginKeyProvider         bufpolicy.PolicyPluginKeyProvider
	policyPluginDataProvider        bufpolicy.PolicyPluginDataProvider
}

func newClient(
	logger *slog.Logger,
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
		logger: logger,
		stderr: clientOptions.stderr,
		fileVersionToDefaultCheckClient: map[bufconfig.FileVersion]check.Client{
			bufconfig.FileVersionV1Beta1: v1beta1DefaultCheckClient,
			bufconfig.FileVersionV1:      v1DefaultCheckClient,
			bufconfig.FileVersionV2:      v2DefaultCheckClient,
		},
		runnerProvider:           clientOptions.runnerProvider,
		pluginReadFile:           clientOptions.pluginReadFile,
		pluginKeyProvider:        clientOptions.pluginKeyProvider,
		pluginDataProvider:       clientOptions.pluginDataProvider,
		policyReadFile:           clientOptions.policyReadFile,
		policyKeyProvider:        clientOptions.policyKeyProvider,
		policyDataProvider:       clientOptions.policyDataProvider,
		policyPluginKeyProvider:  clientOptions.policyPluginKeyProvider,
		policyPluginDataProvider: clientOptions.policyPluginDataProvider,
	}, nil
}

func (c *client) Lint(
	ctx context.Context,
	lintConfig bufconfig.LintConfig,
	image bufimage.Image,
	options ...LintOption,
) error {
	defer xslog.DebugProfile(c.logger)()

	lintOptions := newLintOptions()
	for _, option := range options {
		option.applyToLint(lintOptions)
	}
	// Run lint checks.
	var annotations []*annotation
	lintAnnotations, err := c.lint(
		ctx,
		image,
		lintConfig,
		lintOptions.pluginConfigs,
		nil, // policyConfig.
		lintOptions.relatedCheckConfigs,
		len(lintOptions.policyConfigs) > 0, // hasPolicyConfigs.
	)
	if err != nil {
		return err
	}
	annotations = append(annotations, lintAnnotations...)
	// Run lint policy checks.
	policies, err := c.getPolicies(ctx, lintOptions.policyConfigs)
	if err != nil {
		return err
	}
	for index, policy := range policies {
		policyConfig := lintOptions.policyConfigs[index]
		policyLintConfig, err := policyToBufConfigLintConfig(policy, policyConfig)
		if err != nil {
			return err
		}
		pluginConfigs, err := policyToBufConfigPluginConfigs(policy)
		if err != nil {
			return err
		}
		policyAnnotations, err := c.lint(
			ctx,
			image,
			policyLintConfig,
			pluginConfigs,
			policyConfig,
			nil,  // relatedCheckConfigs.
			true, // hasPolicyConfigs.
		)
		if err != nil {
			return err
		}
		annotations = append(annotations, policyAnnotations...)
	}
	if len(annotations) == 0 {
		return nil
	}
	return bufanalysis.NewFileAnnotationSet(
		annotationsToFileAnnotations(
			imageToPathToExternalPath(
				image,
			),
			annotations,
		)...,
	)
}

func (c *client) lint(
	ctx context.Context,
	image bufimage.Image,
	lintConfig bufconfig.LintConfig,
	pluginConfigs []bufconfig.PluginConfig,
	policyConfig bufconfig.PolicyConfig,
	relatedCheckConfigs []bufconfig.CheckConfig,
	hasPolicyConfigs bool,
) ([]*annotation, error) {
	if lintConfig.Disabled() {
		return nil, nil
	}
	allRules, allCategories, err := c.allRulesAndCategories(
		ctx,
		lintConfig.FileVersion(),
		pluginConfigs,
		policyConfig,
		lintConfig.DisableBuiltin(),
	)
	if err != nil {
		return nil, err
	}
	config, err := configForLintConfig(lintConfig, allRules, allCategories, relatedCheckConfigs)
	if err != nil {
		return nil, err
	}
	configName := bufconfig.DefaultBufYAMLFileName
	if policyConfig != nil {
		configName = policyConfig.Name()
	}
	logRulesConfig(c.logger, configName, config.rulesConfig, hasPolicyConfigs)
	files, err := descriptor.FileDescriptorsForProtoFileDescriptors(imageToProtoFileDescriptors(image))
	if err != nil {
		// An Image may be invalid if it does not contain all of the required dependencies.
		return nil, fmt.Errorf("input image: %w", err)
	}
	request, err := check.NewRequest(
		files,
		check.WithRuleIDs(config.RuleIDs...),
		check.WithOptions(config.DefaultOptions),
	)
	if err != nil {
		return nil, err
	}
	multiClient, err := c.getMultiClient(
		ctx,
		lintConfig.FileVersion(),
		pluginConfigs,
		policyConfig,
		lintConfig.DisableBuiltin(),
		config.DefaultOptions,
	)
	if err != nil {
		return nil, err
	}
	annotations, err := multiClient.Check(ctx, request)
	if err != nil {
		return nil, err
	}
	if len(annotations) == 0 {
		return nil, nil
	}
	return filterAnnotations(config, annotations)
}

func (c *client) Breaking(
	ctx context.Context,
	breakingConfig bufconfig.BreakingConfig,
	image bufimage.Image,
	againstImage bufimage.Image,
	options ...BreakingOption,
) error {
	defer xslog.DebugProfile(c.logger)()

	breakingOptions := newBreakingOptions()
	for _, option := range options {
		option.applyToBreaking(breakingOptions)
	}
	// Run breaking checks.
	var annotations []*annotation
	breakingAnnotations, err := c.breaking(
		ctx,
		image,
		againstImage,
		breakingConfig,
		breakingOptions.pluginConfigs,
		nil, // policyConfig.
		breakingOptions.excludeImports,
		breakingOptions.relatedCheckConfigs,
		len(breakingOptions.policyConfigs) > 0, // hasPolicyConfigs.
	)
	if err != nil {
		return err
	}
	annotations = append(annotations, breakingAnnotations...)
	// Run breaking policy checks.
	policies, err := c.getPolicies(ctx, breakingOptions.policyConfigs)
	if err != nil {
		return err
	}
	for index, policy := range policies {
		policyConfig := breakingOptions.policyConfigs[index]
		policyBreakingConfig, err := policyToBufConfigBreakingConfig(policy, policyConfig)
		if err != nil {
			return err
		}
		pluginConfigs, err := policyToBufConfigPluginConfigs(policy)
		if err != nil {
			return err
		}
		policyAnnotations, err := c.breaking(
			ctx,
			image,
			againstImage,
			policyBreakingConfig,
			pluginConfigs,
			policyConfig,
			breakingOptions.excludeImports,
			nil,  // relatedCheckConfigs.
			true, // hasPolicyConfigs.
		)
		if err != nil {
			return err
		}
		annotations = append(annotations, policyAnnotations...)
	}
	if len(annotations) == 0 {
		return nil
	}
	return bufanalysis.NewFileAnnotationSet(
		annotationsToFileAnnotations(
			imageToPathToExternalPath(
				image,
			),
			annotations,
		)...,
	)
}

func (c *client) breaking(
	ctx context.Context,
	image bufimage.Image,
	againstImage bufimage.Image,
	breakingConfig bufconfig.BreakingConfig,
	pluginConfigs []bufconfig.PluginConfig,
	policyConfig bufconfig.PolicyConfig,
	excludeImports bool,
	relatedCheckConfigs []bufconfig.CheckConfig,
	hasPolicyConfigs bool,
) ([]*annotation, error) {
	if breakingConfig.Disabled() {
		return nil, nil
	}
	allRules, allCategories, err := c.allRulesAndCategories(
		ctx,
		breakingConfig.FileVersion(),
		pluginConfigs,
		policyConfig,
		breakingConfig.DisableBuiltin(),
	)
	if err != nil {
		return nil, err
	}
	config, err := configForBreakingConfig(
		breakingConfig,
		allRules,
		allCategories,
		excludeImports,
		relatedCheckConfigs,
	)
	if err != nil {
		return nil, err
	}
	configName := bufconfig.DefaultBufYAMLFileName
	if policyConfig != nil {
		configName = policyConfig.Name()
	}
	logRulesConfig(c.logger, configName, config.rulesConfig, hasPolicyConfigs)
	fileDescriptors, err := descriptor.FileDescriptorsForProtoFileDescriptors(imageToProtoFileDescriptors(image))
	if err != nil {
		// An Image may be invalid if it does not contain all of the required dependencies.
		return nil, fmt.Errorf("input image: %w", err)
	}
	againstFileDescriptors, err := descriptor.FileDescriptorsForProtoFileDescriptors(imageToProtoFileDescriptors(againstImage))
	if err != nil {
		// An Image may be invalid if it does not contain all of the required dependencies.
		return nil, fmt.Errorf("against image: %w", err)
	}
	request, err := check.NewRequest(
		fileDescriptors,
		check.WithRuleIDs(config.RuleIDs...),
		check.WithAgainstFileDescriptors(againstFileDescriptors),
		check.WithOptions(config.DefaultOptions),
	)
	if err != nil {
		return nil, err
	}
	multiClient, err := c.getMultiClient(
		ctx,
		breakingConfig.FileVersion(),
		pluginConfigs,
		policyConfig,
		breakingConfig.DisableBuiltin(),
		config.DefaultOptions,
	)
	if err != nil {
		return nil, err
	}
	annotations, err := multiClient.Check(ctx, request)
	if err != nil {
		return nil, err
	}
	if len(annotations) == 0 {
		return nil, nil
	}
	return filterAnnotations(config, annotations)
}

func (c *client) ConfiguredRules(
	ctx context.Context,
	ruleType check.RuleType,
	checkConfig bufconfig.CheckConfig,
	options ...ConfiguredRulesOption,
) ([]Rule, error) {
	defer xslog.DebugProfile(c.logger)()

	configuredRulesOptions := newConfiguredRulesOptions()
	for _, option := range options {
		option.applyToConfiguredRules(configuredRulesOptions)
	}
	rules, categories, err := c.allRulesAndCategories(
		ctx,
		checkConfig.FileVersion(),
		configuredRulesOptions.pluginConfigs,
		nil, // PolicyConfig.
		checkConfig.DisableBuiltin(),
	)
	if err != nil {
		return nil, err
	}
	rulesConfig, err := rulesConfigForCheckConfig(checkConfig, rules, categories, ruleType, configuredRulesOptions.relatedCheckConfigs)
	if err != nil {
		return nil, err
	}
	logRulesConfig(c.logger, "", rulesConfig, len(configuredRulesOptions.policyConfigs) > 0)
	allRules := rulesForRuleIDs(rules, rulesConfig.RuleIDs)
	policies, err := c.getPolicies(ctx, configuredRulesOptions.policyConfigs)
	if err != nil {
		return nil, err
	}
	for index, policy := range policies {
		policyConfig := configuredRulesOptions.policyConfigs[index]
		pluginConfigs, err := policyToBufConfigPluginConfigs(policy)
		if err != nil {
			return nil, err
		}
		// Load the check config for the rule type.
		var policyCheckConfig bufconfig.CheckConfig
		switch ruleType {
		case check.RuleTypeLint:
			policyCheckConfig, err = policyToBufConfigLintConfig(policy, policyConfig)
		case check.RuleTypeBreaking:
			policyCheckConfig, err = policyToBufConfigBreakingConfig(policy, policyConfig)
		default:
			return nil, fmt.Errorf("unknown check.RuleType: %v", ruleType)
		}
		if err != nil {
			return nil, err
		}
		policyRules, policyCategories, err := c.allRulesAndCategories(ctx, policyCheckConfig.FileVersion(), pluginConfigs, policyConfig, false)
		if err != nil {
			return nil, err
		}
		policyRulesConfig, err := rulesConfigForCheckConfig(policyCheckConfig, policyRules, policyCategories, ruleType, nil)
		if err != nil {
			return nil, err
		}
		allRules = append(allRules, rulesForRuleIDs(policyRules, policyRulesConfig.RuleIDs)...)
	}
	return allRules, nil
}

func (c *client) AllRules(
	ctx context.Context,
	ruleType check.RuleType,
	fileVersion bufconfig.FileVersion,
	options ...AllRulesOption,
) ([]Rule, error) {
	defer xslog.DebugProfile(c.logger)()

	allRulesOptions := newAllRulesOptions()
	for _, option := range options {
		option.applyToAllRules(allRulesOptions)
	}
	rules, _, err := c.allRulesAndCategories(ctx, fileVersion, allRulesOptions.pluginConfigs, nil, false)
	if err != nil {
		return nil, err
	}
	policies, err := c.getPolicies(ctx, allRulesOptions.policyConfigs)
	if err != nil {
		return nil, err
	}
	for index, policy := range policies {
		policyConfig := allRulesOptions.policyConfigs[index]
		pluginConfigs, err := policyToBufConfigPluginConfigs(policy)
		if err != nil {
			return nil, err
		}
		policyRules, _, err := c.allRulesAndCategories(ctx, fileVersion, pluginConfigs, policyConfig, false)
		if err != nil {
			return nil, err
		}
		rules = append(rules, policyRules...)
	}
	return rulesForType(rules, ruleType), nil
}

func (c *client) AllCategories(
	ctx context.Context,
	fileVersion bufconfig.FileVersion,
	options ...AllCategoriesOption,
) ([]Category, error) {
	defer xslog.DebugProfile(c.logger)()

	allCategoriesOptions := newAllCategoriesOptions()
	for _, option := range options {
		option.applyToAllCategories(allCategoriesOptions)
	}
	_, categories, err := c.allRulesAndCategories(ctx, fileVersion, allCategoriesOptions.pluginConfigs, nil, false)
	return categories, err
}

func (c *client) allRulesAndCategories(
	ctx context.Context,
	fileVersion bufconfig.FileVersion,
	pluginConfigs []bufconfig.PluginConfig,
	policyConfig bufconfig.PolicyConfig, // May be nil.
	disableBuiltin bool,
) ([]Rule, []Category, error) {
	// Just passing through to fulfill all contracts, ie checkClientSpec has non-nil Options.
	// Options are not used here.
	// config struct really just needs refactoring.
	multiClient, err := c.getMultiClient(ctx, fileVersion, pluginConfigs, policyConfig, disableBuiltin, option.EmptyOptions)
	if err != nil {
		return nil, nil, err
	}
	return multiClient.ListRulesAndCategories(ctx)
}

func (c *client) getMultiClient(
	ctx context.Context,
	fileVersion bufconfig.FileVersion,
	pluginConfigs []bufconfig.PluginConfig,
	policyConfig bufconfig.PolicyConfig,
	disableBuiltin bool,
	defaultOptions option.Options,
) (*multiClient, error) {
	var policyConfigName string
	if policyConfig != nil {
		policyConfigName = policyConfig.Name()
	}
	var checkClientSpecs []*checkClientSpec
	if !disableBuiltin {
		defaultCheckClient, ok := c.fileVersionToDefaultCheckClient[fileVersion]
		if !ok {
			return nil, fmt.Errorf("unknown FileVersion: %v", fileVersion)
		}
		checkClientSpecs = append(
			checkClientSpecs,
			// We do not set PluginName for default check.Clients.
			newCheckClientSpec("", policyConfigName, defaultCheckClient, defaultOptions),
		)
	}
	plugins, err := c.getPlugins(ctx, pluginConfigs, policyConfig)
	if err != nil {
		return nil, err
	}
	for index, pluginConfig := range pluginConfigs {
		options, err := option.NewOptions(pluginConfig.Options())
		if err != nil {
			return nil, fmt.Errorf("could not parse options for plugin %q: %w", pluginConfig.Name(), err)
		}
		if c.runnerProvider == nil {
			return nil, fmt.Errorf("must set a RunnerProvider to use plugins")
		}
		runner, err := c.runnerProvider.NewRunner(plugins[index])
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
			newCheckClientSpec(pluginConfig.Name(), policyConfigName, checkClient, options),
		)
	}
	return newMultiClient(c.logger, checkClientSpecs), nil
}

func (c *client) getPlugins(ctx context.Context, pluginConfigs []bufconfig.PluginConfig, policyConfig bufconfig.PolicyConfig) ([]bufplugin.Plugin, error) {
	if len(pluginConfigs) == 0 {
		return nil, nil
	}
	plugins := make([]bufplugin.Plugin, len(pluginConfigs))

	var indexedPluginRefs []xslices.Indexed[bufparse.Ref]
	for index, pluginConfig := range pluginConfigs {
		switch pluginConfig.Type() {
		case bufconfig.PluginConfigTypeLocal:
			plugin, err := bufplugin.NewLocalPlugin(
				pluginConfig.Name(),
				pluginConfig.Args(),
			)
			if err != nil {
				return nil, fmt.Errorf("could not create local Plugin %q: %w", pluginConfig.Name(), err)
			}
			plugins[index] = plugin
		case bufconfig.PluginConfigTypeLocalWasm:
			if c.pluginReadFile == nil {
				// Local Wasm plugins are not supported without a pluginReadFile.
				return nil, fmt.Errorf("unable to read local Wasm Plugin %q", pluginConfig.Name())
			}
			var pluginFullName bufparse.FullName
			if ref := pluginConfig.Ref(); ref != nil {
				pluginFullName = ref.FullName()
			}
			plugin, err := bufplugin.NewLocalWasmPlugin(
				pluginFullName,
				pluginConfig.Name(),
				pluginConfig.Args(),
				func() ([]byte, error) {
					return c.pluginReadFile(pluginConfig.Name())
				},
			)
			if err != nil {
				return nil, err
			}
			plugins[index] = plugin
		case bufconfig.PluginConfigTypeRemoteWasm:
			pluginRef := pluginConfig.Ref()
			if pluginRef == nil {
				return nil, syserror.Newf("missing Ref for remote PluginConfig %q", pluginConfig.Name())
			}
			indexedPluginRefs = append(indexedPluginRefs, xslices.Indexed[bufparse.Ref]{
				Value: pluginRef,
				Index: index,
			})
		default:
			return nil, fmt.Errorf("unknown PluginConfig type %q", pluginConfig.Type())
		}
	}
	// Load the remote plugin data for each plugin ref.
	if len(indexedPluginRefs) > 0 {
		pluginKeyProvider := c.pluginKeyProvider
		pluginDataProvider := c.pluginDataProvider
		if policyConfig != nil {
			// Resolve the Plugin providers for the policy config.
			pluginKeyProvider = c.policyPluginKeyProvider.GetPluginKeyProviderForPolicy(policyConfig.Name())
			pluginDataProvider = c.policyPluginDataProvider.GetPluginDataProviderForPolicy(policyConfig.Name())
		}
		pluginRefs := xslices.IndexedToValues(indexedPluginRefs)
		pluginKeys, err := pluginKeyProvider.GetPluginKeysForPluginRefs(ctx, pluginRefs, bufplugin.DigestTypeP1)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				if policyConfig != nil {
					return nil, fmt.Errorf("unable to resolve plugins for policy %q: %w", policyConfig.Name(), err)
				}
				return nil, fmt.Errorf("unable to resolve plugins: %w", err)
			}
			return nil, err
		}
		pluginDatas, err := pluginDataProvider.GetPluginDatasForPluginKeys(ctx, pluginKeys)
		if err != nil {
			return nil, err
		}
		if len(pluginDatas) != len(pluginRefs) {
			return nil, syserror.Newf("expected %d PluginData, got %d", len(pluginRefs), len(pluginDatas))
		}
		for dataIndex, indexedPluginRef := range indexedPluginRefs {
			pluginData := pluginDatas[dataIndex]
			pluginRef := indexedPluginRef.Value
			index := indexedPluginRef.Index
			pluginConfig := pluginConfigs[index]
			plugin, err := bufplugin.NewRemoteWasmPlugin(
				pluginRef.FullName(),
				pluginConfig.Args(),
				pluginData.PluginKey().CommitID(),
				pluginData.Data,
			)
			if err != nil {
				return nil, fmt.Errorf("could not create remote Plugin %q: %w", pluginRef.String(), err)
			}
			plugins[index] = plugin
		}
	}
	return plugins, nil
}

func (c *client) getPolicies(ctx context.Context, policyConfigs []bufconfig.PolicyConfig) ([]bufpolicy.Policy, error) {
	if len(policyConfigs) == 0 {
		return nil, nil
	}
	policies := make([]bufpolicy.Policy, len(policyConfigs))

	var indexedPolicyRefs []xslices.Indexed[bufparse.Ref]
	for index, policyConfig := range policyConfigs {
		if ref := policyConfig.Ref(); ref != nil {
			indexedPolicyRefs = append(indexedPolicyRefs, xslices.Indexed[bufparse.Ref]{
				Value: ref,
				Index: index,
			})
			continue
		}
		// Local policy config.
		if c.policyReadFile == nil {
			// Local policy configs are not supported without a policyReadFile.
			return nil, fmt.Errorf("unable to read local Policy %q", policyConfig.Name())
		}
		policyData, err := c.policyReadFile(policyConfig.Name())
		if err != nil {
			return nil, fmt.Errorf("could not read local policy config %q: %w", policyConfig.Name(), err)
		}
		reader := bytes.NewReader(policyData)
		policyFile, err := bufpolicyconfig.ReadBufPolicyYAMLFile(reader, policyConfig.Name())
		if err != nil {
			return nil, fmt.Errorf("could not read policy file %q: %w", policyConfig.Name(), err)
		}
		policy, err := bufpolicy.NewPolicy("", nil, policyConfig.Name(), uuid.Nil, policyFile.PolicyConfig)
		if err != nil {
			return nil, err
		}
		policies[index] = policy
	}
	// Load the remote policy data for each policy ref.
	if len(indexedPolicyRefs) > 0 {
		policyRefs := xslices.IndexedToValues(indexedPolicyRefs)
		policyKeys, err := c.policyKeyProvider.GetPolicyKeysForPolicyRefs(ctx, policyRefs, bufpolicy.DigestTypeO1)
		if err != nil {
			return nil, fmt.Errorf("could not get PolicyKeys for PolicyRefs: %w", err)
		}
		policyDatas, err := c.policyDataProvider.GetPolicyDatasForPolicyKeys(ctx, policyKeys)
		if err != nil {
			return nil, fmt.Errorf("could not get PolicyDatas for PolicyKeys: %w", err)
		}
		if len(policyDatas) != len(policyRefs) {
			return nil, syserror.Newf("expected %d PolicyData, got %d", len(policyRefs), len(policyDatas))
		}
		for dataIndex, indexedPolicyRef := range indexedPolicyRefs {
			policyData := policyDatas[dataIndex]
			policyKey := policyData.PolicyKey()
			index := indexedPolicyRef.Index
			policy, err := bufpolicy.NewPolicy("", policyKey.FullName(), policyKey.FullName().String(), policyKey.CommitID(), func() (bufpolicy.PolicyConfig, error) {
				return policyData.Config()
			})
			if err != nil {
				return nil, err
			}
			policies[index] = policy
		}
	}
	return policies, nil
}

func filterAnnotations(
	config *config,
	annotations []*annotation,
) ([]*annotation, error) {
	return xslices.FilterError(
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
		// For group fields, comments in source get attributed to the nested message
		// (the synthetic message that holds the group's fields), not the group field itself.
		// We need to also check the group field's source path for comment ignores.
		if groupFieldSourcePath := getGroupFieldSourcePathForNestedMessage(sourcePath, protoreflectFileDescriptor); groupFieldSourcePath != nil {
			associatedSourcePaths = append(associatedSourcePaths, groupFieldSourcePath)
		}
		sourceLocations := protoreflectFileDescriptor.SourceLocations()
		for _, associatedSourcePath := range associatedSourcePaths {
			sourceLocation := sourceLocations.ByPath(associatedSourcePath)
			if leadingComments := sourceLocation.LeadingComments; leadingComments != "" {
				for _, line := range xstrings.SplitTrimLinesNoEmpty(leadingComments) {
					if checkCommentLineForCheckIgnore(line, config.CommentIgnorePrefix, ruleID) {
						return true, nil
					}
				}
			}
		}
	}
	return false, nil
}

// getGroupFieldSourcePathForNestedMessage checks if the given source path points to a
// descriptor within a synthetic message that was created for a group field. If so, it
// returns the source path of the parent group field so that comment ignores on the group
// field declaration are properly respected.
//
// Group fields in proto2 create a synthetic message to hold their fields. The source
// path for elements within this synthetic message includes the nested message path
// component. For example, for a group field "Metadata" at index 0 in a message, the
// source path might be [4,0,2,0,3,0,...] where:
// - [4,0] is the parent message
// - [2,0] is the group field
// - [3,0] is the nested message (synthetic message for the group)
// - ... is the rest of the path within the synthetic message
//
// In this case, we want to also check [4,0,2,0] (the group field) for comment ignores.
func getGroupFieldSourcePathForNestedMessage(
	sourcePath protoreflect.SourcePath,
	fileDescriptor protoreflect.FileDescriptor,
) protoreflect.SourcePath {
	// A source path pointing to a descriptor within a synthetic group message will have
	// at least 6 elements: [4, msg_idx, 2, field_idx, 3, nested_msg_idx, ...]
	// where 3 is the field number for nested_type in DescriptorProto.
	//
	// We need to check if the message at nested_msg_idx is a synthetic message created
	// for a group field. A synthetic message for a group field has:
	// 1. The same name as the parent field
	// 2. The parent field has type GROUP
	//
	// We find the parent field by looking at the field at [2, field_idx] within the
	// parent message at [4, msg_idx].
	if len(sourcePath) < 6 {
		return nil
	}
	// Check if we have the pattern [4, msg_idx, 2, field_idx, 3, nested_msg_idx, ...]
	// where 4 = message_type, 2 = field, 3 = nested_type
	const (
		messageTypeTag  = 4  // FieldDescriptorProto.message_type
		fieldTypeTag    = 2  // DescriptorProto.field
		nestedTypeTag   = 3  // DescriptorProto.nested_type
		fieldNumberTag  = 3  // FieldDescriptorProto.number
		fieldTypeTag2   = 5  // FieldDescriptorProto.type
		fieldTypeNameTag = 6 // FieldDescriptorProto.type_name
	)
	// Walk through the source path to find the pattern
	for i := 0; i < len(sourcePath)-5; i++ {
		if sourcePath[i] == messageTypeTag &&
			sourcePath[i+2] == fieldTypeTag &&
			sourcePath[i+4] == nestedTypeTag {
			// Found the pattern [4, msg_idx, 2, field_idx, 3, nested_msg_idx]
			msgIdx := int(sourcePath[i+1])
			fieldIdx := int(sourcePath[i+3])
			nestedMsgIdx := int(sourcePath[i+5])

			messages := fileDescriptor.Messages()
			if msgIdx >= messages.Len() {
				continue
			}
			message := messages.Get(msgIdx)
			fields := message.Fields()
			if fieldIdx >= fields.Len() {
				continue
			}
			field := fields.Get(fieldIdx)
			// Check if this is a group field
			if field.Kind() != protoreflect.GroupKind {
				continue
			}
			// Check if the nested message name matches the field name
			// (this confirms it's the synthetic message for the group field)
			nestedMessages := message.Messages()
			if nestedMsgIdx >= nestedMessages.Len() {
				continue
			}
			nestedMessage := nestedMessages.Get(nestedMsgIdx)
			if string(nestedMessage.Name()) != string(field.Name()) {
				continue
			}
			// This is a synthetic message for a group field
			// Return the source path of the group field: [4, msg_idx, 2, field_idx]
			groupFieldPath := make(protoreflect.SourcePath, i+4)
			copy(groupFieldPath, sourcePath[:i+4])
			return groupFieldPath
		}
	}
	return nil
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
	pluginConfigs       []bufconfig.PluginConfig
	policyConfigs       []bufconfig.PolicyConfig
	relatedCheckConfigs []bufconfig.CheckConfig
}

func newLintOptions() *lintOptions {
	return &lintOptions{}
}

type breakingOptions struct {
	pluginConfigs       []bufconfig.PluginConfig
	policyConfigs       []bufconfig.PolicyConfig
	excludeImports      bool
	relatedCheckConfigs []bufconfig.CheckConfig
}

func newBreakingOptions() *breakingOptions {
	return &breakingOptions{}
}

type configuredRulesOptions struct {
	pluginConfigs       []bufconfig.PluginConfig
	policyConfigs       []bufconfig.PolicyConfig
	relatedCheckConfigs []bufconfig.CheckConfig
}

func newConfiguredRulesOptions() *configuredRulesOptions {
	return &configuredRulesOptions{}
}

type allRulesOptions struct {
	pluginConfigs []bufconfig.PluginConfig
	policyConfigs []bufconfig.PolicyConfig
}

func newAllRulesOptions() *allRulesOptions {
	return &allRulesOptions{}
}

type allCategoriesOptions struct {
	pluginConfigs []bufconfig.PluginConfig
	policyConfigs []bufconfig.PolicyConfig
}

func newAllCategoriesOptions() *allCategoriesOptions {
	return &allCategoriesOptions{}
}

type clientOptions struct {
	stderr                   io.Writer
	runnerProvider           RunnerProvider
	pluginReadFile           func(string) ([]byte, error)
	pluginKeyProvider        bufplugin.PluginKeyProvider
	pluginDataProvider       bufplugin.PluginDataProvider
	policyReadFile           func(string) ([]byte, error)
	policyKeyProvider        bufpolicy.PolicyKeyProvider
	policyDataProvider       bufpolicy.PolicyDataProvider
	policyPluginKeyProvider  bufpolicy.PolicyPluginKeyProvider
	policyPluginDataProvider bufpolicy.PolicyPluginDataProvider
}

func newClientOptions() *clientOptions {
	return &clientOptions{
		pluginKeyProvider:        bufplugin.NopPluginKeyProvider,
		pluginDataProvider:       bufplugin.NopPluginDataProvider,
		policyKeyProvider:        bufpolicy.NopPolicyKeyProvider,
		policyDataProvider:       bufpolicy.NopPolicyDataProvider,
		policyPluginKeyProvider:  bufpolicy.NopPolicyPluginKeyProvider,
		policyPluginDataProvider: bufpolicy.NopPolicyPluginDataProvider,
	}
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

type policyConfigsOption struct {
	policyConfigs []bufconfig.PolicyConfig
}

func (p *policyConfigsOption) applyToLint(lintOptions *lintOptions) {
	lintOptions.policyConfigs = append(lintOptions.policyConfigs, p.policyConfigs...)
}

func (p *policyConfigsOption) applyToBreaking(breakingOptions *breakingOptions) {
	breakingOptions.policyConfigs = append(breakingOptions.policyConfigs, p.policyConfigs...)
}

func (p *policyConfigsOption) applyToConfiguredRules(configuredRulesOptions *configuredRulesOptions) {
	configuredRulesOptions.policyConfigs = append(configuredRulesOptions.policyConfigs, p.policyConfigs...)
}

func (p *policyConfigsOption) applyToAllRules(allRulesOptions *allRulesOptions) {
	allRulesOptions.policyConfigs = append(allRulesOptions.policyConfigs, p.policyConfigs...)
}

func (p *policyConfigsOption) applyToAllCategories(allCategoriesOptions *allCategoriesOptions) {
	allCategoriesOptions.policyConfigs = append(allCategoriesOptions.policyConfigs, p.policyConfigs...)
}

type relatedCheckConfigsOption struct {
	relatedCheckConfigs []bufconfig.CheckConfig
}

func (r *relatedCheckConfigsOption) applyToLint(lintOptions *lintOptions) {
	lintOptions.relatedCheckConfigs = append(lintOptions.relatedCheckConfigs, r.relatedCheckConfigs...)
}

func (r *relatedCheckConfigsOption) applyToBreaking(breakingOptions *breakingOptions) {
	breakingOptions.relatedCheckConfigs = append(breakingOptions.relatedCheckConfigs, r.relatedCheckConfigs...)
}

func (r *relatedCheckConfigsOption) applyToConfiguredRules(configuredRulesOptions *configuredRulesOptions) {
	configuredRulesOptions.relatedCheckConfigs = append(configuredRulesOptions.relatedCheckConfigs, r.relatedCheckConfigs...)
}
