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

package internal

import (
	"context"
	"errors"
	"fmt"

	"buf.build/go/bufplugin/check"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/wasm"
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
			stringutil.SliceToString(bufcli.AllRuleFormatStrings),
		),
	)
	flagSet.StringVar(
		&f.Version,
		versionFlagName,
		"", // do not set a default as we need to know if this is unset
		fmt.Sprintf(
			"List all the rules for the given configuration version. By default, the version in the buf.yaml in the current directory is used, or the latest version otherwise (currently v2). Cannot be set if --%s is set. Must be one of %s",
			configuredOnlyFlagName,
			slicesext.Map(
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

	// If the module path is set, we need to ensure that the Workspace is a v2 Workspace.
	if !workspace.IsV2() && flags.ModulePath != "" {
		return appcmd.NewInvalidArgumentErrorf("--%s can only be specified for v2 workspaces", modulePathFlagName)
	}

	wasmRuntimeCacheDir, err := bufcli.CreateWasmRuntimeCacheDir(container)
	if err != nil {
		return err
	}
	wasmRuntime, err := wasm.NewRuntime(ctx, wasm.WithLocalCacheDir(wasmRuntimeCacheDir))
	if err != nil {
		return err
	}
	defer func() {
		retErr = errors.Join(retErr, wasmRuntime.Close(ctx))
	}()
	client, err := bufcheck.NewClient(
		container.Logger(),
		bufcheck.NewRunnerProvider(wasmRuntime),
		bufcheck.ClientWithStderr(container.Stderr()),
	)
	if err != nil {
		return err
	}

	var rules []bufcheck.Rule
	if flags.ConfiguredOnly {
		var module bufmodule.Module
		if flags.ModulePath != "" {
			// If the module path is set, we need to find the module.
			//
			// In a v2 Workspace, the module path is the BucketID within the Workspace.
			// For v1, the module path is derived by the workspace constructor. It is
			// a user error if the module path is set on a v1 Workspace as validated above.
			module = workspace.GetModuleForBucketID(flags.ModulePath)
			if module == nil {
				return fmt.Errorf("no module for path %q", flags.ModulePath)
			}
		} else {
			// If the module path is not set, we need to ensure
			// that there is only one module in the Workspace.
			modules := workspace.Modules()
			if len(modules) == 0 {
				return syserror.New("got 0 Modules for Workspace")
			}
			if len(modules) > 1 {
				return appcmd.NewInvalidArgumentErrorf("--%s must be specified if the the buf.yaml has more than one module", modulePathFlagName)
			}
			module = modules[0]
		}

		var checkConfig bufconfig.CheckConfig
		switch ruleType {
		case check.RuleTypeLint:
			checkConfig = workspace.GetLintConfigForOpaqueID(module.OpaqueID())
		case check.RuleTypeBreaking:
			checkConfig = workspace.GetBreakingConfigForOpaqueID(module.OpaqueID())
		default:
			return fmt.Errorf("unknown check.RuleType: %v", ruleType)
		}
		configuredRuleOptions := []bufcheck.ConfiguredRulesOption{
			bufcheck.WithPluginConfigs(workspace.PluginConfigs()...),
		}
		rules, err = client.ConfiguredRules(
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
			bufcheck.WithPluginConfigs(workspace.PluginConfigs()...),
		}
		fileVersion := bufconfig.FileVersionV1
		if workspace.IsV2() {
			fileVersion = bufconfig.FileVersionV2
		}
		rules, err = client.AllRules(
			ctx,
			ruleType,
			fileVersion,
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
