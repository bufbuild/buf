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
	"io/fs"

	"buf.build/go/bufplugin/check"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/wasm"
	"github.com/spf13/pflag"
)

const (
	allFlagName               = "all"
	configFlagName            = "config"
	includeDeprecatedFlagName = "include-deprecated"
	formatFlagName            = "format"
	versionFlagName           = "version"
)

// NewLSCommand returns a new ls Command.
func NewLSCommand(
	name string,
	builder appext.SubCommandBuilder,
	ruleType check.RuleType,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:        name,
		Short:      fmt.Sprintf("List %s rules", ruleType.String()),
		Args:       appcmd.NoArgs,
		Deprecated: fmt.Sprintf(`use "buf config %s" instead. However, "buf mod %s" will continue to work.`, name, name),
		Hidden:     true,
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
	All               bool
	Config            string
	IncludeDeprecated bool
	Format            string
	Version           string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.BoolVar(
		&f.All,
		allFlagName,
		false,
		"List all rules and not just those currently configured",
	)
	flagSet.StringVar(
		&f.Config,
		configFlagName,
		"",
		fmt.Sprintf(
			`The buf.yaml file or data to use for configuration. Ignored if --%s or --%s is specified`,
			allFlagName,
			versionFlagName,
		),
	)
	flagSet.BoolVar(
		&f.IncludeDeprecated,
		includeDeprecatedFlagName,
		false,
		fmt.Sprintf(
			`Also print deprecated rules. Has no effect if --%s is not set.`,
			allFlagName,
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
			"List all the rules for the given configuration version. Implies --%s. Must be one of %s",
			allFlagName,
			slicesext.Map(
				bufconfig.AllFileVersions,
				func(fileVersion bufconfig.FileVersion) string {
					return fileVersion.String()
				},
			),
		),
	)
}

func lsRun(
	ctx context.Context,
	container appext.Container,
	flags *flags,
	commandName string,
	ruleType check.RuleType,
) error {
	if flags.All {
		// We explicitly document that if all is set, config is ignored.
		// If a user wants to override the version while using all, they should use version.
		flags.Config = ""
	}
	if flags.Version != "" {
		// If version is set, all is implied, and we use the config override to specify the version.
		flags.All = true
		// This also results in config being ignored per the documentation.
		flags.Config = fmt.Sprintf(`{"version":"%s"}`, flags.Version)
	}
	bufYAMLFile, err := bufcli.GetBufYAMLFileForDirPathOrOverride(ctx, ".", flags.Config)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		bufYAMLFile, err = bufconfig.NewBufYAMLFile(
			bufconfig.FileVersionV1,
			[]bufconfig.ModuleConfig{
				bufconfig.DefaultModuleConfigV1,
			},
			nil,
			nil,
		)
		if err != nil {
			return err
		}
	}
	fileVersion := bufYAMLFile.FileVersion()
	if fileVersion == bufconfig.FileVersionV2 {
		return fmt.Errorf(`"buf mod %s" does not work for v2 buf.yaml files, use "buf config %s" instead`, commandName, commandName)
	}
	// BufYAMLFiles <=v1 never had plugins.
	client, err := bufcheck.NewClient(
		container.Logger(),
		bufcheck.NewRunnerProvider(
			command.NewRunner(),
			wasm.UnimplementedRuntime,
			bufplugin.NopPluginKeyProvider,
			bufplugin.NopPluginDataProvider,
		),
		bufcheck.ClientWithStderr(container.Stderr()),
	)
	if err != nil {
		return err
	}
	var rules []bufcheck.Rule
	if flags.All {
		rules, err = client.AllRules(ctx, ruleType, bufYAMLFile.FileVersion())
		if err != nil {
			return err
		}
	} else {
		moduleConfigs := bufYAMLFile.ModuleConfigs()
		if len(moduleConfigs) != 1 {
			return syserror.Newf("got %d ModuleConfigs for a v1beta1/v1 buf.yaml", len(moduleConfigs))
		}
		var checkConfig bufconfig.CheckConfig
		switch ruleType {
		case check.RuleTypeLint:
			checkConfig = moduleConfigs[0].LintConfig()
		case check.RuleTypeBreaking:
			checkConfig = moduleConfigs[0].BreakingConfig()
		default:
			return fmt.Errorf("unknown check.RuleType: %v", ruleType)
		}
		rules, err = client.ConfiguredRules(ctx, ruleType, checkConfig)
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
