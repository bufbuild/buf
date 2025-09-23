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

package internal

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"buf.build/go/app/appcmd"
	"buf.build/go/app/appext"
	"buf.build/go/bufplugin/check"
	"buf.build/go/standard/xslices"
	"buf.build/go/standard/xstrings"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/spf13/pflag"
)

const (
	configuredOnlyFlagName    = "configured-only"
	configFlagName            = "config"
	includeDeprecatedFlagName = "include-deprecated"
	formatFlagName            = "format"
	versionFlagName           = "version"
	modulePathFlagName        = "module-path"
)

// NewLSCommand returns a new ls Command.
func NewLSCommand(
	name string,
	builder appext.SubCommandBuilder,
	ruleType check.RuleType,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name,
		Short: fmt.Sprintf("List %s rules", ruleType.String()),
		Args:  appcmd.NoArgs,
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return lsRun(
					ctx,
					container,
					flags,
					name,
					ruleType,
				)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	ConfiguredOnly    bool
	Config            string
	IncludeDeprecated bool
	Format            string
	Version           string
	ModulePath        string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.BoolVar(
		&f.ConfiguredOnly,
		configuredOnlyFlagName,
		false,
		"List rules that are configured instead of listing all available rules",
	)
	flagSet.StringVar(
		&f.Config,
		configFlagName,
		"",
		fmt.Sprintf(
			`The buf.yaml file or data to use for configuration. --%s must be set`,
			configuredOnlyFlagName,
		),
	)
	flagSet.BoolVar(
		&f.IncludeDeprecated,
		includeDeprecatedFlagName,
		false,
		fmt.Sprintf(
			`Also print deprecated rules. Has no effect if --%s is set.`,
			configuredOnlyFlagName,
		),
	)
	flagSet.StringVar(
		&f.Format,
		formatFlagName,
		"text",
		fmt.Sprintf(
			"The format to print rules as. Must be one of %s",
			xstrings.SliceToString(bufcli.AllRuleFormatStrings),
		),
	)
	flagSet.StringVar(
		&f.Version,
		versionFlagName,
		"", // do not set a default as we need to know if this is unset
		fmt.Sprintf(
			"List all the rules for the given configuration version. By default, the version in the buf.yaml in the current directory is used, or the latest version otherwise (currently v2). Cannot be set if --%s is set. Must be one of %s",
			configuredOnlyFlagName,
			xslices.Map(
				bufconfig.AllFileVersions,
				func(fileVersion bufconfig.FileVersion) string {
					return fileVersion.String()
				},
			),
		),
	)
	flagSet.StringVar(
		&f.ModulePath,
		modulePathFlagName,
		"",
		fmt.Sprintf(
			"The path to the specific module to list configured rules for as specified in the buf.yaml. If the buf.yaml has more than one module defined, this must be set. --%s must be set",
			configuredOnlyFlagName,
		),
	)
}

func lsRun(
	ctx context.Context,
	container appext.Container,
	flags *flags,
	commandName string,
	ruleType check.RuleType,
) (retErr error) {
	if flags.ConfiguredOnly {
		if flags.Version != "" {
			return appcmd.NewInvalidArgumentErrorf("--%s cannot be specified if --%s is specified", versionFlagName, configFlagName)
		}
	} else {
		if flags.Config != "" {
			return appcmd.NewInvalidArgumentErrorf("--%s must be set if --%s is specified", configuredOnlyFlagName, configFlagName)
		}
		if flags.ModulePath != "" {
			return appcmd.NewInvalidArgumentErrorf("--%s must be set if --%s is specified", configuredOnlyFlagName, modulePathFlagName)
		}
	}
	configOverride := flags.Config
	if flags.Version != "" {
		configOverride = fmt.Sprintf(`{"version":"%s"}`, flags.Version)
	}
	bufYAMLFile, err := bufcli.GetBufYAMLFileForDirPathOrOverride(ctx, ".", configOverride)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		bufYAMLFile, err = bufconfig.NewBufYAMLFile(
			bufconfig.FileVersionV2,
			[]bufconfig.ModuleConfig{
				bufconfig.DefaultModuleConfigV2,
			},
			nil,
			nil,
			nil,
		)
		if err != nil {
			return err
		}
	}
	wasmRuntime, err := bufcli.NewWasmRuntime(ctx, container)
	if err != nil {
		return err
	}
	defer func() {
		retErr = errors.Join(retErr, wasmRuntime.Close(ctx))
	}()
	controller, err := bufcli.NewController(container)
	if err != nil {
		return err
	}
	workspace, err := controller.GetWorkspace(
		ctx,
		".",
		bufctl.WithConfigOverride(configOverride),
	)
	if err != nil {
		return err
	}
	checkClient, err := controller.GetCheckClientForWorkspace(
		ctx,
		workspace,
		wasmRuntime,
	)
	if err != nil {
		return err
	}
	var rules []bufcheck.Rule
	if flags.ConfiguredOnly {
		moduleConfigs := bufYAMLFile.ModuleConfigs()
		var moduleConfig bufconfig.ModuleConfig
		switch fileVersion := bufYAMLFile.FileVersion(); fileVersion {
		case bufconfig.FileVersionV1Beta1, bufconfig.FileVersionV1:
			if len(moduleConfigs) != 1 {
				return syserror.Newf("got %d ModuleConfigs for a v1beta1/v1 buf.yaml", len(moduleConfigs))
			}
			moduleConfig = moduleConfigs[0]
		case bufconfig.FileVersionV2:
			switch len(moduleConfigs) {
			case 0:
				return syserror.New("got 0 ModuleConfigs from a BufYAMLFile")
			case 1:
				moduleConfig = moduleConfigs[0]
			default:
				if flags.ModulePath == "" {
					return appcmd.NewInvalidArgumentErrorf("--%s must be specified if the the buf.yaml has more than one module", modulePathFlagName)
				}
				moduleConfig, err = getModuleConfigForModulePath(moduleConfigs, flags.ModulePath)
				if err != nil {
					return err
				}
			}
		default:
			return syserror.Newf("unknown FileVersion: %v", fileVersion)
		}
		var checkConfig bufconfig.CheckConfig
		// We add all check configs (both lint and breaking) as related configs to check if plugins
		// have rules configured.
		// We allocated twice the size of moduleConfigs for both lint and breaking configs.
		allCheckConfigs := make([]bufconfig.CheckConfig, 0, len(moduleConfigs)*2)
		for _, moduleConfig := range moduleConfigs {
			allCheckConfigs = append(allCheckConfigs, moduleConfig.LintConfig())
			allCheckConfigs = append(allCheckConfigs, moduleConfig.BreakingConfig())
		}
		switch ruleType {
		case check.RuleTypeLint:
			checkConfig = moduleConfig.LintConfig()
		case check.RuleTypeBreaking:
			checkConfig = moduleConfig.BreakingConfig()
		default:
			return fmt.Errorf("unknown check.RuleType: %v", ruleType)
		}
		configuredRuleOptions := []bufcheck.ConfiguredRulesOption{
			bufcheck.WithPluginConfigs(bufYAMLFile.PluginConfigs()...),
			bufcheck.WithPolicyConfigs(bufYAMLFile.PolicyConfigs()...),
			bufcheck.WithRelatedCheckConfigs(allCheckConfigs...),
		}
		rules, err = checkClient.ConfiguredRules(
			ctx,
			ruleType,
			checkConfig,
			configuredRuleOptions...,
		)
		if err != nil {
			return err
		}
	} else {
		allRulesOptions := []bufcheck.AllRulesOption{
			bufcheck.WithPluginConfigs(bufYAMLFile.PluginConfigs()...),
			bufcheck.WithPolicyConfigs(bufYAMLFile.PolicyConfigs()...),
		}
		rules, err = checkClient.AllRules(
			ctx,
			ruleType,
			bufYAMLFile.FileVersion(),
			allRulesOptions...,
		)
		if err != nil {
			return err
		}
	}
	return bufcli.PrintRules(
		container.Stdout(),
		rules,
		flags.Format,
		flags.IncludeDeprecated,
	)
}

func getModuleConfigForModulePath(moduleConfigs []bufconfig.ModuleConfig, modulePath string) (bufconfig.ModuleConfig, error) {
	modulePath = normalpath.Normalize(modulePath)
	// Multiple modules in a v2 workspace may have the same moduleDirPath.
	moduleConfigsFound := []bufconfig.ModuleConfig{}
	for _, moduleConfig := range moduleConfigs {
		if moduleConfig.DirPath() == modulePath {
			moduleConfigsFound = append(moduleConfigsFound, moduleConfig)
		}
	}
	switch len(moduleConfigsFound) {
	case 0:
		return nil, fmt.Errorf("no module found for path %q", modulePath)
	case 1:
		return moduleConfigsFound[0], nil
	default:
		// TODO: add --module-name flag to allow differentiation
		return nil, fmt.Errorf("multiple modules found for %q", modulePath)
	}
}
