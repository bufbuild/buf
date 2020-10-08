// Copyright 2020 Buf Technologies, Inc.
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

package generate

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bufbuild/buf/internal/buf/bufanalysis"
	"github.com/bufbuild/buf/internal/buf/bufcli"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufimage"
	"github.com/bufbuild/buf/internal/buf/buffetch"
	"github.com/bufbuild/buf/internal/pkg/app/appcmd"
	"github.com/bufbuild/buf/internal/pkg/app/appflag"
	"github.com/bufbuild/buf/internal/pkg/app/appproto"
	"github.com/bufbuild/buf/internal/pkg/app/appproto/appprotoexec"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/bufbuild/buf/internal/pkg/stringutil"
	"github.com/bufbuild/buf/internal/pkg/thread"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	errorFormatFlagName = "error-format"
	filesFlagName       = "file"
	inputFlagName       = "input"
	inputConfigFlagName = "input-config"
	pluginNameFlagName  = "plugin"
	pluginOutFlagName   = "plugin-out"
	pluginOptFlagName   = "plugin-opt"
	pluginPathFlagName  = "plugin-path"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appflag.Builder,
	moduleResolverReaderProvider bufcli.ModuleResolverReaderProvider,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name,
		Short: "Generate stubs for a plugin.",
		Args:  cobra.NoArgs,
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appflag.Container) error {
				return run(ctx, container, flags, moduleResolverReaderProvider)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	ErrorFormat string
	Files       []string
	Input       string
	InputConfig string
	PluginName  string
	PluginOut   string
	PluginOpt   []string
	PluginPath  string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(
		&f.ErrorFormat,
		errorFormatFlagName,
		"text",
		fmt.Sprintf(
			"The format for build errors, printed to stderr. Must be one of %s.",
			stringutil.SliceToString(bufanalysis.AllFormatStrings),
		),
	)
	flagSet.StringSliceVar(
		&f.Files,
		filesFlagName,
		nil,
		`Limit to specific files. This is an advanced feature and is not recommended.`,
	)
	flagSet.StringVar(
		&f.Input,
		inputFlagName,
		".",
		fmt.Sprintf(
			`The source or image to generate from. Must be one of format %s.`,
			buffetch.AllFormatsString,
		),
	)
	flagSet.StringVar(
		&f.InputConfig,
		inputConfigFlagName,
		"",
		`The config file or data to use.`,
	)
	flagSet.StringVar(
		&f.PluginName,
		pluginNameFlagName,
		"",
		`The plugin to use. By default, protoc-gen-PLUGIN must be on your $PATH.`,
	)
	flagSet.StringVar(
		&f.PluginOut,
		pluginOutFlagName,
		"",
		`The output directory.`,
	)
	flagSet.StringSliceVar(
		&f.PluginOpt,
		pluginOptFlagName,
		nil,
		`The options to use.`,
	)
	flagSet.StringVar(
		&f.PluginPath,
		pluginPathFlagName,
		"",
		`The path to the plugin binary. By default, uses protoc-gen-PLUGIN on your $PATH.`,
	)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
	moduleResolverReaderProvider bufcli.ModuleResolverReaderProvider,
) (retErr error) {
	if flags.PluginName == "" {
		return appcmd.NewInvalidArgumentErrorf("--%s is required", pluginNameFlagName)
	}
	if flags.PluginOut == "" {
		return appcmd.NewInvalidArgumentErrorf("--%s is required", pluginOutFlagName)
	}
	ref, err := buffetch.NewRefParser(container.Logger()).GetRef(ctx, flags.Input)
	if err != nil {
		return fmt.Errorf("--%s: %v", inputFlagName, err)
	}
	moduleResolver, err := moduleResolverReaderProvider.GetModuleResolver(ctx, container)
	if err != nil {
		return err
	}
	moduleReader, err := moduleResolverReaderProvider.GetModuleReader(ctx, container)
	if err != nil {
		return err
	}
	env, fileAnnotations, err := bufcli.NewWireEnvReader(
		container.Logger(),
		inputConfigFlagName,
		moduleResolver,
		moduleReader,
	).GetEnv(
		ctx,
		container,
		ref,
		flags.InputConfig,
		flags.Files, // we filter on files
		false,       // input files must exist
		false,       // we must include source info for generation
	)
	if err != nil {
		return err
	}
	if len(fileAnnotations) > 0 {
		if err := bufanalysis.PrintFileAnnotations(container.Stderr(), fileAnnotations, flags.ErrorFormat); err != nil {
			return err
		}
		return errors.New("")
	}
	images, err := bufimage.ImageByDir(env.Image())
	if err != nil {
		return err
	}
	readWriteBucket, err := storageos.NewReadWriteBucket(flags.PluginOut)
	if err != nil {
		return err
	}
	var handlerOptions []appprotoexec.HandlerOption
	if flags.PluginPath != "" {
		handlerOptions = append(handlerOptions, appprotoexec.HandlerWithPluginPath(flags.PluginPath))
	}
	handler, err := appprotoexec.NewHandler(
		container.Logger(),
		flags.PluginName,
		handlerOptions...,
	)
	if err != nil {
		return err
	}
	executor := appproto.NewExecutor(container.Logger(), handler)
	jobs := make([]func() error, len(images))
	for i, image := range images {
		image := image
		jobs[i] = func() error {
			return executor.Execute(
				ctx,
				container,
				readWriteBucket,
				bufimage.ImageToCodeGeneratorRequest(
					image,
					strings.Join(flags.PluginOpt, ","),
				),
			)
		}
	}
	return thread.Parallelize(jobs...)
}
