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

package protoc

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/buf/bufprotoc"
	"github.com/bufbuild/buf/private/buf/bufprotopluginexec"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimageutil"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufprotoplugin"
	"github.com/bufbuild/buf/private/bufpkg/bufprotoplugin/bufprotopluginos"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/tracing"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flagsBuilder := newFlagsBuilder()
	return &appcmd.Command{
		Use:   name + " <proto_file1> <proto_file2> ...",
		Short: "High-performance protoc replacement",
		Long: `This command replaces protoc using Buf's internal compiler.

The implementation is in progress. Although it outperforms mainline protoc,
it hasn't yet been optimized.

This protoc replacement is currently stable but should be considered a preview.

Additional flags:

      --(.*)_out:                   Run the named plugin.
      --(.*)_opt:                   Options for the named plugin.
      @filename:                    Parse arguments from the given filename.`,
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				env, err := flagsBuilder.Build(app.Args(container))
				if err != nil {
					return err
				}
				return run(ctx, container, env)
			},
		),
		BindFlags:     flagsBuilder.Bind,
		NormalizeFlag: flagsBuilder.Normalize,
		Version: fmt.Sprintf(
			"%v.%v-buf",
			// DefaultVersion has an extra major version that corresponds to
			// backwards-compatibility level of C++ runtime. The actual version
			// of the compiler is just the minor and patch versions.
			bufprotopluginexec.DefaultVersion.GetMinor(),
			bufprotopluginexec.DefaultVersion.GetPatch(),
		),
	}
}

func run(
	ctx context.Context,
	container appext.Container,
	env *env,
) (retErr error) {
	runner := command.NewRunner()
	logger := container.Logger()
	tracer := tracing.NewTracer(container.Tracer())
	ctx, span := tracer.Start(ctx, tracing.WithErr(&retErr))
	defer span.End()

	if env.PrintFreeFieldNumbers && len(env.PluginNameToPluginInfo) > 0 {
		return fmt.Errorf("cannot call --%s and plugins at the same time", printFreeFieldNumbersFlagName)
	}
	if env.PrintFreeFieldNumbers && env.Output != "" {
		return fmt.Errorf("cannot call --%s and --%s at the same time", printFreeFieldNumbersFlagName, outputFlagName)
	}
	if len(env.PluginNameToPluginInfo) > 0 && env.Output != "" {
		return fmt.Errorf("cannot call --%s and plugins at the same time", outputFlagName)
	}

	if checkedEntry := logger.Check(zapcore.DebugLevel, "env"); checkedEntry != nil {
		checkedEntry.Write(
			zap.Any("flags", env.flags),
			zap.Any("plugins", env.PluginNameToPluginInfo),
		)
	}

	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	moduleSet, err := bufprotoc.NewModuleSetForProtoc(
		ctx,
		tracer,
		storageosProvider,
		env.IncludeDirPaths,
		env.FilePaths,
	)
	if err != nil {
		return err
	}
	var buildOptions []bufimage.BuildImageOption
	// we always need source code info if we are doing generation
	if len(env.PluginNameToPluginInfo) == 0 && !env.IncludeSourceInfo {
		buildOptions = append(buildOptions, bufimage.WithExcludeSourceCodeInfo())
	}
	image, err := bufimage.BuildImage(
		ctx,
		tracer,
		bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles(moduleSet),
		buildOptions...,
	)
	if err != nil {
		var fileAnnotationSet bufanalysis.FileAnnotationSet
		if errors.As(err, &fileAnnotationSet) {
			if err := bufanalysis.PrintFileAnnotationSet(
				container.Stderr(),
				fileAnnotationSet,
				env.ErrorFormat,
			); err != nil {
				return err
			}
			// we do this even though we're in protoc compatibility mode as we just need to do non-zero
			// but this also makes us consistent with the rest of buf
			return bufctl.ErrFileAnnotation
		}
		return err
	}
	if env.PrintFreeFieldNumbers {
		fileInfos, err := bufmodule.GetTargetFileInfos(
			ctx,
			bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles(
				moduleSet,
			),
		)
		if err != nil {
			return err
		}
		var filePaths []string
		for _, fileInfo := range fileInfos {
			filePaths = append(filePaths, fileInfo.Path())
		}
		s, err := bufimageutil.FreeMessageRangeStrings(ctx, filePaths, image)
		if err != nil {
			return err
		}
		if _, err := container.Stdout().Write([]byte(strings.Join(s, "\n") + "\n")); err != nil {
			return err
		}
		return nil
	}
	if len(env.PluginNameToPluginInfo) > 0 {
		images := []bufimage.Image{image}
		if env.ByDir {
			f := func() (retErr error) {
				_, span := tracer.Start(ctx, tracing.WithErr(&retErr))
				defer span.End()
				images, err = bufimage.ImageByDir(image)
				return err
			}
			if err := f(); err != nil {
				return err
			}
		}
		pluginResponses := make([]*bufprotoplugin.PluginResponse, 0, len(env.PluginNamesSortedByOutIndex))
		for _, pluginName := range env.PluginNamesSortedByOutIndex {
			pluginInfo, ok := env.PluginNameToPluginInfo[pluginName]
			if !ok {
				return fmt.Errorf("no value in PluginNamesToPluginInfo for %q", pluginName)
			}
			response, err := executePlugin(
				ctx,
				logger,
				tracer,
				storageosProvider,
				runner,
				container,
				images,
				pluginName,
				pluginInfo,
			)
			if err != nil {
				return err
			}
			pluginResponses = append(pluginResponses, bufprotoplugin.NewPluginResponse(response, pluginName, pluginInfo.Out))
		}
		if err := bufprotoplugin.ValidatePluginResponses(pluginResponses); err != nil {
			return err
		}
		responseWriter := bufprotopluginos.NewResponseWriter(
			logger,
			storageosProvider,
		)
		for _, pluginResponse := range pluginResponses {
			pluginInfo, ok := env.PluginNameToPluginInfo[pluginResponse.PluginName]
			if !ok {
				return fmt.Errorf("no value in PluginNamesToPluginInfo for %q", pluginResponse.PluginName)
			}
			if err := responseWriter.AddResponse(
				ctx,
				pluginResponse.Response,
				pluginInfo.Out,
			); err != nil {
				return err
			}
		}
		if err := responseWriter.Close(); err != nil {
			return err
		}
		return nil
	}
	if env.Output == "" {
		return appcmd.NewInvalidArgumentErrorf("required flag %q not set", outputFlagName)
	}
	controller, err := bufcli.NewController(container)
	if err != nil {
		return err
	}
	return controller.PutImage(
		ctx,
		env.Output,
		image,
		// Actually redundant with bufimage.BuildImageOptions right now.
		bufctl.WithImageExcludeSourceInfo(!env.IncludeSourceInfo),
		bufctl.WithImageExcludeImports(!env.IncludeImports),
		bufctl.WithImageAsFileDescriptorSet(true),
	)
}
