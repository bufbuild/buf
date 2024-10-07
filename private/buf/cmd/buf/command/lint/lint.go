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

package lint

import (
	"context"
	"errors"
	"fmt"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/buf/private/pkg/wasm"
	"github.com/spf13/pflag"
	"go.uber.org/multierr"
)

const (
	errorFormatFlagName     = "error-format"
	configFlagName          = "config"
	pathsFlagName           = "path"
	excludePathsFlagName    = "exclude-path"
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
		Short: "Run linting on Protobuf files",
		Long:  bufcli.GetInputLong(`the source, module, or Image to lint`),
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
	ErrorFormat     string
	Config          string
	Paths           []string
	ExcludePaths    []string
	DisableSymlinks bool
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
			stringutil.SliceToString(bufcli.AllLintFormatStrings),
		),
	)
	flagSet.StringVar(
		&f.Config,
		configFlagName,
		"",
		`The buf.yaml file or data to use for configuration`,
	)
}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) (retErr error) {
	if err := bufcli.ValidateErrorFormatFlagLint(flags.ErrorFormat, errorFormatFlagName); err != nil {
		return err
	}
	// Parse out if this is config-ignore-yaml.
	// This is messed.
	controllerErrorFormat := flags.ErrorFormat
	if controllerErrorFormat == "config-ignore-yaml" {
		controllerErrorFormat = "text"
	}
	input, err := bufcli.GetInputValue(container, flags.InputHashtag, ".")
	if err != nil {
		return err
	}
	controller, err := bufcli.NewController(
		container,
		bufctl.WithDisableSymlinks(flags.DisableSymlinks),
		bufctl.WithFileAnnotationErrorFormat(controllerErrorFormat),
		bufctl.WithFileAnnotationsToStdout(),
	)
	if err != nil {
		return err
	}
	imageWithConfigs, err := controller.GetTargetImageWithConfigs(
		ctx,
		input,
		bufctl.WithTargetPaths(flags.Paths, flags.ExcludePaths),
		bufctl.WithConfigOverride(flags.Config),
	)
	if err != nil {
		return err
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
		retErr = multierr.Append(retErr, wasmRuntime.Close(ctx))
	}()
	var allFileAnnotations []bufanalysis.FileAnnotation
	for _, imageWithConfig := range imageWithConfigs {
		client, err := bufcheck.NewClient(
			container.Logger(),
			bufcheck.NewRunnerProvider(command.NewRunner(), wasmRuntime),
			bufcheck.ClientWithStderr(container.Stderr()),
		)
		if err != nil {
			return err
		}
		lintOptions := []bufcheck.LintOption{
			bufcheck.WithPluginConfigs(imageWithConfig.PluginConfigs()...),
		}
		if err := client.Lint(
			ctx,
			imageWithConfig.LintConfig(),
			imageWithConfig,
			lintOptions...,
		); err != nil {
			var fileAnnotationSet bufanalysis.FileAnnotationSet
			if errors.As(err, &fileAnnotationSet) {
				allFileAnnotations = append(allFileAnnotations, fileAnnotationSet.FileAnnotations()...)
			} else {
				return err
			}
		}
	}
	if len(allFileAnnotations) > 0 {
		allFileAnnotationSet := bufanalysis.NewFileAnnotationSet(allFileAnnotations...)
		if flags.ErrorFormat == "config-ignore-yaml" {
			if err := bufcli.PrintFileAnnotationSetLintConfigIgnoreYAMLV1(
				container.Stdout(),
				allFileAnnotationSet,
			); err != nil {
				return err
			}
		} else {
			if err := bufanalysis.PrintFileAnnotationSet(
				container.Stdout(),
				allFileAnnotationSet,
				flags.ErrorFormat,
			); err != nil {
				return err
			}
		}
		return bufctl.ErrFileAnnotation
	}
	return nil
}
