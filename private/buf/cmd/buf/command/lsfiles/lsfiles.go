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

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/spf13/cobra"
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
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <input>",
		Short: "List Protobuf files",
		Long:  bufcli.GetInputLong(`the source, module, or image to list from`),
		Args:  cobra.MaximumNArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appflag.Container) error {
				return run(ctx, container, flags)
			},
			bufcli.NewErrorInterceptor(),
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
	container appflag.Container,
	flags *flags,
) error {
	input, err := bufcli.GetInputValue(container, flags.InputHashtag, ".")
	if err != nil {
		return err
	}
	ref, err := buffetch.NewRefParser(container.Logger()).GetRef(ctx, input)
	if err != nil {
		return err
	}
	storageosProvider := bufcli.NewStorageosProvider(flags.DisableSymlinks)
	runner := command.NewRunner()
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	fileLister, err := bufcli.NewWireFileLister(
		container,
		storageosProvider,
		runner,
		clientConfig,
	)
	if err != nil {
		return err
	}
	fileRefs, fileAnnotations, err := fileLister.ListFiles(
		ctx,
		container,
		ref,
		flags.Config,
		flags.IncludeImports,
	)
	if err != nil {
		return err
	}
	if len(fileAnnotations) > 0 {
		// stderr since we do output to stdout potentially
		if err := bufanalysis.PrintFileAnnotations(
			container.Stderr(),
			fileAnnotations,
			flags.ErrorFormat,
		); err != nil {
			return err
		}
		return bufcli.ErrFileAnnotation
	}
	if flags.AsImportPaths {
		bufmoduleref.SortFileInfos(fileRefs)
		for _, fileRef := range fileRefs {
			if _, err := fmt.Fprintln(container.Stdout(), fileRef.Path()); err != nil {
				return err
			}
		}
	} else {
		bufmoduleref.SortFileInfosByExternalPath(fileRefs)
		for _, fileRef := range fileRefs {
			if _, err := fmt.Fprintln(container.Stdout(), fileRef.ExternalPath()); err != nil {
				return err
			}
		}
	}
	return nil
}
