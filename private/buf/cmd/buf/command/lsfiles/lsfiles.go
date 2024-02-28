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

package lsfiles

import (
	"context"
	"fmt"
	"sort"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/spf13/pflag"
)

const (
	asImportPathsFlagName   = "as-import-paths"
	configFlagName          = "config"
	errorFormatFlagName     = "error-format"
	includeImportsFlagName  = "include-imports"
	disableSymlinksFlagName = "disable-symlinks"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <input>",
		Short: "List Protobuf files",
		Long:  bufcli.GetInputLong(`the source, module, or image to list from`),
		Args:  appcmd.MaximumNArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	AsImportPaths   bool
	Config          string
	ErrorFormat     string
	IncludeImports  bool
	DisableSymlinks bool
	// special
	InputHashtag string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	bufcli.BindInputHashtag(flagSet, &f.InputHashtag)
	bufcli.BindDisableSymlinks(flagSet, &f.DisableSymlinks, disableSymlinksFlagName)
	flagSet.BoolVar(
		&f.AsImportPaths,
		asImportPathsFlagName,
		false,
		"Strip local directory paths and print filepaths as they are imported",
	)
	flagSet.StringVar(
		&f.Config,
		configFlagName,
		"",
		`The buf.yaml configuration file or data to use`,
	)
	flagSet.StringVar(
		&f.ErrorFormat,
		errorFormatFlagName,
		"text",
		fmt.Sprintf(
			"The format for build errors printed to stderr. Must be one of %s",
			stringutil.SliceToString(bufanalysis.AllFormatStrings),
		),
	)
	flagSet.BoolVar(
		&f.IncludeImports,
		includeImportsFlagName,
		false,
		"Include imports",
	)
}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) error {
	input, err := bufcli.GetInputValue(container, flags.InputHashtag, ".")
	if err != nil {
		return err
	}
	controller, err := bufcli.NewController(
		container,
		bufctl.WithDisableSymlinks(flags.DisableSymlinks),
		bufctl.WithFileAnnotationErrorFormat(flags.ErrorFormat),
	)
	if err != nil {
		return err
	}
	protoFileInfos, err := controller.GetProtoFileInfos(
		ctx,
		input,
		bufctl.WithProtoFileInfosIncludeImports(flags.IncludeImports),
		bufctl.WithConfigOverride(flags.Config),
	)
	if err != nil {
		return err
	}
	pathFunc := bufctl.ProtoFileInfo.ExternalPath
	if flags.AsImportPaths {
		pathFunc = bufctl.ProtoFileInfo.Path
	}
	paths := slicesext.Map(
		protoFileInfos,
		func(protoFileInfo bufctl.ProtoFileInfo) string {
			return pathFunc(protoFileInfo)
		},
	)
	sort.Strings(paths)
	for _, path := range paths {
		if _, err := fmt.Fprintln(container.Stdout(), path); err != nil {
			return err
		}
	}
	return nil
}
