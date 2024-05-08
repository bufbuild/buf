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

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/stringutil"
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
	ruleType string,
	getAllRulesV1Beta1 func() ([]bufcheck.Rule, error),
	getAllRulesV1 func() ([]bufcheck.Rule, error),
	getAllRulesV2 func() ([]bufcheck.Rule, error),
	getRulesForModuleConfig func(bufconfig.ModuleConfig) ([]bufcheck.Rule, error),
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name,
		Short: fmt.Sprintf("List %s rules", ruleType),
		Args:  appcmd.NoArgs,
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return lsRun(
					ctx,
					container,
					flags,
					name,
					getAllRulesV1Beta1,
					getAllRulesV1,
					getAllRulesV2,
					getRulesForModuleConfig,
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
			stringutil.SliceToString(bufcheck.AllRuleFormatStrings),
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
	getAllRulesV1Beta1 func() ([]bufcheck.Rule, error),
	getAllRulesV1 func() ([]bufcheck.Rule, error),
	getAllRulesV2 func() ([]bufcheck.Rule, error),
	getRulesForModuleConfig func(bufconfig.ModuleConfig) ([]bufcheck.Rule, error),
) error {
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
		)
		if err != nil {
			return err
		}
	}

	var rules []bufcheck.Rule
	if flags.ConfiguredOnly {
		moduleConfigs := bufYAMLFile.ModuleConfigs()
		switch fileVersion := bufYAMLFile.FileVersion(); fileVersion {
		case bufconfig.FileVersionV1Beta1, bufconfig.FileVersionV1:
			if len(moduleConfigs) != 1 {
				return syserror.Newf("got %d ModuleConfigs for a v1beta1/v1 buf.yaml", len(moduleConfigs))
			}
			rules, err = getRulesForModuleConfig(moduleConfigs[0])
			if err != nil {
				return err
			}
		case bufconfig.FileVersionV2:
			switch len(moduleConfigs) {
			case 0:
				return syserror.New("got 0 ModuleConfigs from a BufYAMLFile")
			case 1:
				rules, err = getRulesForModuleConfig(moduleConfigs[0])
				if err != nil {
					return err
				}
			default:
				if flags.ModulePath == "" {
					return appcmd.NewInvalidArgumentErrorf("--%s must be specified if the the buf.yaml has more than one module", modulePathFlagName)
				}
				moduleConfig, err := getModuleConfigForModulePath(moduleConfigs, flags.ModulePath)
				if err != nil {
					return err
				}
				rules, err = getRulesForModuleConfig(moduleConfig)
				if err != nil {
					return err
				}
			}
		default:
			return syserror.Newf("unknown FileVersion: %v", fileVersion)
		}
	} else {
		switch fileVersion := bufYAMLFile.FileVersion(); fileVersion {
		case bufconfig.FileVersionV1Beta1:
			rules, err = getAllRulesV1Beta1()
			if err != nil {
				return err
			}
		case bufconfig.FileVersionV1:
			rules, err = getAllRulesV1()
			if err != nil {
				return err
			}
		case bufconfig.FileVersionV2:
			rules, err = getAllRulesV2()
			if err != nil {
				return err
			}
		default:
			return syserror.Newf("unknown FileVersion: %v", fileVersion)
		}
	}
	return bufcheck.PrintRules(
		container.Stdout(),
		rules,
		flags.Format,
		flags.IncludeDeprecated,
	)
}

func getModuleConfigForModulePath(moduleConfigs []bufconfig.ModuleConfig, modulePath string) (bufconfig.ModuleConfig, error) {
	modulePath = normalpath.Normalize(modulePath)
	for _, moduleConfig := range moduleConfigs {
		if moduleConfig.DirPath() == modulePath {
			return moduleConfig, nil
		}
	}
	return nil, fmt.Errorf("no module found for path %q", modulePath)
}
