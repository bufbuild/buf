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
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
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
	imageFileInfos, err := controller.GetImageFileInfos(
		ctx,
		input,
		bufctl.WithConfigOverride(flags.Config),
	)
	if err != nil {
		return err
	}
	if !flags.IncludeImports {
		imageFileInfos = slicesext.Filter(
			imageFileInfos,
			func(imageFileInfo bufimage.ImageFileInfo) bool {
				return !imageFileInfo.IsImport()
			},
		)
	} else {
		// Also automatically adds imported WKTs if not present.
		imageFileInfos, err = bufimage.ImageFileInfosWithOnlyTargetsAndTargetImports(imageFileInfos)
		if err != nil {
			return err
		}
	}
	pathFunc := bufimage.ImageFileInfo.ExternalPath
	if flags.AsImportPaths {
		pathFunc = bufimage.ImageFileInfo.Path
	}
	paths := slicesext.Map(
		imageFileInfos,
		func(imageFileInfo bufimage.ImageFileInfo) string {
			return pathFunc(imageFileInfo)
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

//type externalFileInfo struct {
//Path      string `json:"path" yaml:"path"`
//LocalPath string `json:"local_path" yaml:"local_path"`
//Module    string `json:"module" yaml:"module"`
//Commit    string `json:"commit" yaml:"commit"`
//Target    bool   `json:"target" yaml:"target"`
//}

//func newExternalFileInfo(fileInfo bufmodule.FileInfo) *externalFileInfo {
//var module string
//if moduleFullName := fileInfo.Module().ModuleFullName(); moduleFullName != nil {
//module = moduleFullName.String()
//}
//var commit string
//if commitID := fileInfo.Module().CommitID(); !commitID.IsNil() {
//commit = commitID.String()
//}
//return &externalFileInfo{
//Path:      fileInfo.Path(),
//LocalPath: fileInfo.LocalPath(),
//Module:    module,
//Commit:    commit,
//Target:    fileInfo.IsTargetFile(),
//}
//}
