// Copyright 2020-2023 Buf Technologies, Inc.
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

package modlsbreakingrules

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/bufbuild/buf/private/buf/bufcli"
	modinternal "github.com/bufbuild/buf/private/buf/cmd/buf/command/mod/internal"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufbreaking"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	allFlagName     = "all"
	configFlagName  = "config"
	formatFlagName  = "format"
	versionFlagName = "version"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appflag.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name,
		Short: "List breaking rules",
		Args:  cobra.NoArgs,
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appflag.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	All     bool
	Config  string
	Format  string
	Version string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	modinternal.BindLSRulesAll(flagSet, &f.All, allFlagName)
	modinternal.BindLSRulesConfig(flagSet, &f.Config, configFlagName, allFlagName, versionFlagName)
	modinternal.BindLSRulesFormat(flagSet, &f.Format, formatFlagName)
	modinternal.BindLSRulesVersion(flagSet, &f.Version, versionFlagName, allFlagName)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
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
				bufconfig.DefaultModuleConfig,
			},
			nil,
		)
		if err != nil {
			return err
		}
	}
	var rules []bufcheck.Rule
	if flags.All {
		switch fileVersion := bufYAMLFile.FileVersion(); fileVersion {
		case bufconfig.FileVersionV1Beta1:
			rules, err = bufbreaking.GetAllRulesV1Beta1()
			if err != nil {
				return err
			}
		case bufconfig.FileVersionV1:
			rules, err = bufbreaking.GetAllRulesV1()
			if err != nil {
				return err
			}
		case bufconfig.FileVersionV2:
			rules, err = bufbreaking.GetAllRulesV2()
			if err != nil {
				return err
			}
		default:
			return syserror.Newf("unknown FileVersion: %v", fileVersion)
		}
	} else {
		moduleConfigs := bufYAMLFile.ModuleConfigs()
		switch len(moduleConfigs) {
		case 0:
			return fmt.Errorf("no modules specified in buf.yaml")
		case 1:
			rules, err = bufbreaking.RulesForConfig(moduleConfigs[0].BreakingConfig())
			if err != nil {
				return err
			}
		default:
			if bufYAMLFile.FileVersion() == bufconfig.FileVersionV2 {
				return errors.New("buf mod ls-breaking-rules does not work for buf.yaml v2 yet")
			}
			return syserror.New("multiple ModuleConfigs for a non-v2 buf.yaml")
		}
	}
	return bufcheck.PrintRules(
		container.Stdout(),
		rules,
		flags.Format,
	)
}
