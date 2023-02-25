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

package lint

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/golangci/revgrep"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/buflint"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/buflint/buflintconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/stringutil"
)

const (
	errorFormatFlagName     = "error-format"
	configFlagName          = "config"
	pathsFlagName           = "path"
	excludePathsFlagName    = "exclude-path"
	disableSymlinksFlagName = "disable-symlinks"
	newFlagName             = "new"
	newFromRevFlagName      = "new-from-rev"
	newFromPatchFlagName    = "new-from-patch"
	wholeFilesFlagName      = "whole-files"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <input>",
		Short: "Run linting on Protobuf files",
		Long:  bufcli.GetInputLong(`the source, module, or Image to lint`),
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
	ErrorFormat       string
	Config            string
	Paths             []string
	ExcludePaths      []string
	DisableSymlinks   bool
	DiffFromRevision  string
	DiffPatchFilePath string
	OnlyNew           bool
	WholeFiles        bool

	// special
	InputHashtag string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	bufcli.BindInputHashtag(flagSet, &f.InputHashtag)
	bufcli.BindPaths(flagSet, &f.Paths, pathsFlagName)
	bufcli.BindExcludePaths(flagSet, &f.ExcludePaths, excludePathsFlagName)
	bufcli.BindDisableSymlinks(flagSet, &f.DisableSymlinks, disableSymlinksFlagName)
	flagSet.StringVar(
		&f.ErrorFormat,
		errorFormatFlagName,
		"text",
		fmt.Sprintf(
			"The format for build errors or check violations printed to stdout. Must be one of %s",
			stringutil.SliceToString(buflint.AllFormatStrings),
		),
	)
	flagSet.StringVar(
		&f.Config,
		configFlagName,
		"",
		"The file or data to use for configuration",
	)
	flagSet.BoolVar(
		&f.OnlyNew,
		newFlagName,
		false,
		"Show only new issues: if there are unstaged changes or untracked files, only those changes "+
			"are analyzed, else only changes in HEAD~ are analyzed.\nIt's a super-useful option for integration "+
			"of buf-lint into existing large codebase.\nIt's not practical to fix all existing issues at "+
			"the moment of integration: much better to not allow issues in new code.\nFor CI setups, prefer "+
			"--new-from-rev=HEAD~, as --new can skip linting the current patch if any scripts generate "+
			"unstaged files before buf-lint runs.",
	)
	flagSet.StringVar(
		&f.DiffFromRevision,
		newFromRevFlagName,
		"",
		"Show only new issues created after git revision `REV`",
	)
	flagSet.StringVar(
		&f.DiffPatchFilePath,
		newFromPatchFlagName,
		"",
		"Show only new issues created in git patch with file path `PATH`",
	)
	flagSet.BoolVar(
		&f.WholeFiles,
		wholeFilesFlagName,
		false,
		"Show issues in any part of update files (requires new-from-rev or new-from-patch)",
	)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) (retErr error) {
	if err := bufcli.ValidateErrorFormatFlagLint(flags.ErrorFormat, errorFormatFlagName); err != nil {
		return err
	}
	input, err := bufcli.GetInputValue(container, flags.InputHashtag, ".")
	if err != nil {
		return err
	}
	ref, err := buffetch.NewRefParser(container.Logger(), buffetch.RefParserWithProtoFileRefAllowed()).GetRef(ctx, input)
	if err != nil {
		return err
	}
	storageosProvider := bufcli.NewStorageosProvider(flags.DisableSymlinks)
	runner := command.NewRunner()
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	imageConfigReader, err := bufcli.NewWireImageConfigReader(
		container,
		storageosProvider,
		runner,
		clientConfig,
	)
	if err != nil {
		return err
	}
	imageConfigs, fileAnnotations, err := imageConfigReader.GetImageConfigs(
		ctx,
		container,
		ref,
		flags.Config,
		flags.Paths,        // we filter checks for files
		flags.ExcludePaths, // we exclude these paths
		false,              // input files must exist
		false,              // we must include source info for linting
	)
	if err != nil {
		return err
	}
	if len(fileAnnotations) > 0 {
		formatString := flags.ErrorFormat
		if formatString == "config-ignore-yaml" {
			formatString = "text"
		}
		if err := bufanalysis.PrintFileAnnotations(container.Stdout(), fileAnnotations, formatString); err != nil {
			return err
		}
		return bufcli.ErrFileAnnotation
	}
	var allFileAnnotations []bufanalysis.FileAnnotation
	for _, imageConfig := range imageConfigs {
		fileAnnotations, err := buflint.NewHandler(container.Logger()).Check(
			ctx,
			imageConfig.Config().Lint,
			bufimage.ImageWithoutImports(imageConfig.Image()),
		)
		if err != nil {
			return err
		}
		allFileAnnotations = append(allFileAnnotations, fileAnnotations...)
	}
	allFileAnnotations, err = filterNewFileAnnotations(
		allFileAnnotations,
		flags.DiffFromRevision,
		flags.DiffPatchFilePath,
		flags.OnlyNew,
		flags.WholeFiles,
	)
	if err != nil {
		return err
	}
	if len(allFileAnnotations) > 0 {
		if err := buflintconfig.PrintFileAnnotations(
			container.Stdout(),
			bufanalysis.DeduplicateAndSortFileAnnotations(allFileAnnotations),
			flags.ErrorFormat,
		); err != nil {
			return err
		}
		return bufcli.ErrFileAnnotation
	}
	return nil
}

func filterNewFileAnnotations(
	allFileAnnotations []bufanalysis.FileAnnotation,
	diffFromRevision, diffPatchFilePath string,
	onlyNew, wholeFiles bool,
) ([]bufanalysis.FileAnnotation, error) {
	if !onlyNew && diffFromRevision == "" && diffPatchFilePath == "" {
		return allFileAnnotations, nil
	}

	var patchReader io.Reader
	if diffPatchFilePath != "" {
		patch, err := os.ReadFile(diffPatchFilePath)
		if err != nil {
			return nil, fmt.Errorf("can't read from patch file %s: %s", diffPatchFilePath, err)
		}
		patchReader = bytes.NewReader(patch)
	}
	checker := revgrep.Checker{
		Patch:        patchReader,
		RevisionFrom: diffFromRevision,
		WholeFiles:   wholeFiles,
	}
	if err := checker.Prepare(); err != nil {
		return nil, fmt.Errorf("can't prepare diff by revgrep: %s", err)
	}

	var annotations []bufanalysis.FileAnnotation
	for _, annotation := range allFileAnnotations {
		if _, isNew := checker.IsNewIssue(bufanalysis.NewIssue(annotation)); isNew {
			annotations = append(annotations, annotation)
		}
	}
	return annotations, nil
}
