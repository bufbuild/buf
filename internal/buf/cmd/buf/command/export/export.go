// Copyright 2020-2021 Buf Technologies, Inc.
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

package export

import (
	"context"
	"errors"
	"os"

	"github.com/bufbuild/buf/internal/buf/bufcli"
	"github.com/bufbuild/buf/internal/buf/buffetch"
	"github.com/bufbuild/buf/internal/buf/bufmodule"
	"github.com/bufbuild/buf/internal/buf/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/internal/pkg/app/appcmd"
	"github.com/bufbuild/buf/internal/pkg/app/appflag"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/multierr"
)

const (
	excludeImportsFlagName = "exclude-imports"
	pathsFlagName          = "path"
	outputFlagName         = "output"
	outputFlagShortName    = "o"
	configFlagName         = "config"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appflag.Builder,
	deprecated string,
	hidden bool,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:        name + " <input>",
		Short:      "Export the files from the input location.",
		Long:       bufcli.GetInputLong(`the source or module to export`),
		Args:       cobra.MaximumNArgs(1),
		Deprecated: deprecated,
		Hidden:     hidden,
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
	ExcludeImports bool
	Paths          []string
	Output         string
	Config         string

	// special
	InputHashtag string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	bufcli.BindInputHashtag(flagSet, &f.InputHashtag)
	bufcli.BindExcludeImports(flagSet, &f.ExcludeImports, excludeImportsFlagName)
	bufcli.BindPaths(flagSet, &f.Paths, pathsFlagName)
	flagSet.StringVarP(
		&f.Output,
		outputFlagName,
		outputFlagShortName,
		"",
		`The directory to write the files to.`,
	)
	_ = cobra.MarkFlagRequired(flagSet, outputFlagName)
	flagSet.StringVar(
		&f.Config,
		configFlagName,
		"",
		`The config file or data to use.`,
	)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) error {
	input, err := bufcli.GetInputValue(container, flags.InputHashtag, "", "", ".")
	if err != nil {
		return err
	}
	sourceOrModuleRef, err := buffetch.NewRefParser(container.Logger()).GetSourceOrModuleRef(ctx, input)
	if err != nil {
		return err
	}
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	registryProvider, err := bufcli.NewRegistryProvider(ctx, container)
	if err != nil {
		return err
	}
	moduleReader, err := bufcli.NewModuleReaderAndCreateCacheDirs(container, registryProvider)
	if err != nil {
		return err
	}
	moduleConfigReader, err := bufcli.NewWireModuleConfigReaderForModuleReader(
		container,
		storageosProvider,
		registryProvider,
		moduleReader,
	)
	if err != nil {
		return err
	}
	moduleConfigs, err := moduleConfigReader.GetModuleConfigs(
		ctx,
		container,
		sourceOrModuleRef,
		flags.Config,
		flags.Paths,
		false,
	)
	if err != nil {
		return err
	}
	moduleFileSetBuilder := bufmodulebuild.NewModuleFileSetBuilder(
		container.Logger(),
		moduleReader,
	)
	moduleFileSets := make([]bufmodule.ModuleFileSet, len(moduleConfigs))
	for i, moduleConfig := range moduleConfigs {
		moduleFileSet, err := moduleFileSetBuilder.Build(
			ctx,
			moduleConfig.Module(),
			bufmodulebuild.WithWorkspace(moduleConfig.Workspace()),
		)
		if err != nil {
			return err
		}
		moduleFileSets[i] = moduleFileSet
	}
	if err := os.MkdirAll(flags.Output, 0755); err != nil {
		return err
	}
	readWriteBucket, err := storageosProvider.NewReadWriteBucket(
		flags.Output,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return err
	}
	fileInfosFunc := bufmodule.ModuleFileSet.AllFileInfos
	// if we filtered on some paths, only use the targets
	if len(flags.Paths) > 0 {
		fileInfosFunc = func(
			moduleFileSet bufmodule.ModuleFileSet,
			ctx context.Context,
		) ([]bufmodule.FileInfo, error) {
			return moduleFileSet.TargetFileInfos(ctx)
		}
	}
	writtenPaths := make(map[string]struct{})
	for _, moduleFileSet := range moduleFileSets {
		fileInfos, err := fileInfosFunc(moduleFileSet, ctx)
		if err != nil {
			return err
		}
		for _, fileInfo := range fileInfos {
			path := fileInfo.Path()
			if _, ok := writtenPaths[path]; ok {
				continue
			}
			if flags.ExcludeImports && fileInfo.IsImport() {
				continue
			}
			moduleFile, err := moduleFileSet.GetModuleFile(ctx, path)
			if err != nil {
				return err
			}
			if err := storage.CopyReadObject(ctx, readWriteBucket, moduleFile); err != nil {
				return multierr.Append(err, moduleFile.Close())
			}
			if err := moduleFile.Close(); err != nil {
				return err
			}
			writtenPaths[path] = struct{}{}
		}
	}
	if len(writtenPaths) == 0 {
		return errors.New("no .proto target files found")
	}
	return nil
}
