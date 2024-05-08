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

package configinit

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/spf13/pflag"
)

const (
	documentationCommentsFlagName = "doc"
	outDirPathFlagName            = "output"
	outDirPathFlagShortName       = "o"
	uncommentFlagName             = "uncomment"
)

// NewCommand returns a new init Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
	deprecated string,
	hidden bool,
	bindOldFlags bool,
) *appcmd.Command {
	flags := newFlags(bindOldFlags)
	return &appcmd.Command{
		Use:   name + " [buf.build/owner/foobar]",
		Short: "Initialize buf configuration files for your local development",
		Long: `This command will write a new buf.yaml file to start your local development.

If a buf.yaml already exists, this command will not overwrite it, and will produce an error.

The effects of this command may change over time - this command may initialize i.e. buf.gen.yaml files in the future.`,
		Args:       appcmd.MaximumNArgs(1),
		Deprecated: deprecated,
		Hidden:     hidden,
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	OutDirPath string

	// Hidden.
	DocumentationComments bool
	// Hidden.
	Uncomment bool

	bindOldFlags bool
}

func newFlags(bindOldFlags bool) *flags {
	return &flags{
		bindOldFlags: bindOldFlags,
	}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVarP(
		&f.OutDirPath,
		outDirPathFlagName,
		outDirPathFlagShortName,
		".",
		`The directory to write the configuration files to`,
	)
	if f.bindOldFlags {
		// TODO FUTURE: Bring this flag back in future versions if we decide it's important.
		// We're not breaking anyone by not actually producing comments for now.
		flagSet.BoolVar(
			&f.DocumentationComments,
			documentationCommentsFlagName,
			false,
			"Write inline documentation in the form of comments in the resulting configuration file",
		)
		_ = flagSet.MarkHidden(documentationCommentsFlagName)
		flagSet.BoolVar(
			&f.Uncomment,
			uncommentFlagName,
			false,
			"Uncomment examples in the resulting configuration file",
		)
		_ = flagSet.MarkHidden(uncommentFlagName)
	}
}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) error {
	if err := bufcli.ValidateRequiredFlag(outDirPathFlagName, flags.OutDirPath); err != nil {
		return err
	}
	exists, err := bufcli.BufYAMLFileExistsForDirPath(ctx, flags.OutDirPath)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("buf.yaml already exists in directory %s, will not overwrite", flags.OutDirPath)
	}

	fileVersion := bufconfig.FileVersionV2
	var moduleFullName bufmodule.ModuleFullName
	if container.NumArgs() > 0 {
		moduleFullName, err = bufmodule.ParseModuleFullName(container.Arg(0))
		if err != nil {
			return err
		}
	}

	moduleConfig, err := bufconfig.NewModuleConfig(
		".",
		moduleFullName,
		map[string][]string{
			".": {},
		},
		bufconfig.NewLintConfig(
			bufconfig.NewEnabledCheckConfigForUseIDsAndCategories(
				fileVersion,
				[]string{"DEFAULT"},
			),
			"",
			false,
			false,
			false,
			"",
			// We actually want comment ignores enabled by default
			true,
		),
		bufconfig.NewBreakingConfig(
			bufconfig.NewEnabledCheckConfigForUseIDsAndCategories(
				fileVersion,
				[]string{"FILE"},
			),
			false,
		),
	)
	if err != nil {
		return err
	}
	bufYAMLFile, err := bufconfig.NewBufYAMLFile(
		fileVersion,
		[]bufconfig.ModuleConfig{
			moduleConfig,
		},
		nil,
	)
	if err != nil {
		return err
	}

	return bufcli.PutBufYAMLFileForDirPath(ctx, flags.OutDirPath, bufYAMLFile)
}
