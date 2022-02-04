// Copyright 2020-2022 Buf Technologies, Inc.
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

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/multierr"
)

const (
	excludeImportsFlagName  = "exclude-imports"
	pathsFlagName           = "path"
	outputFlagName          = "output"
	outputFlagShortName     = "o"
	configFlagName          = "config"
	excludePathsFlagName    = "exclude-path"
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
		Short: "Export the files from the input location to an output location.",
		Long:  bufcli.GetInputLong(`the source or module to export`),
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
	ExcludeImports  bool
	Paths           []string
	Output          string
	Config          string
	ExcludePaths    []string
	DisableSymlinks bool

	// special
	InputHashtag string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	bufcli.BindDisableSymlinks(flagSet, &f.DisableSymlinks, disableSymlinksFlagName)
	bufcli.BindInputHashtag(flagSet, &f.InputHashtag)
	bufcli.BindExcludeImports(flagSet, &f.ExcludeImports, excludeImportsFlagName)
	bufcli.BindPaths(flagSet, &f.Paths, pathsFlagName)
	bufcli.BindExcludePaths(flagSet, &f.ExcludePaths, excludePathsFlagName)
	flagSet.StringVarP(
		&f.Output,
		outputFlagName,
		outputFlagShortName,
		"",
		`The output directory for exported files.`,
	)
	_ = cobra.MarkFlagRequired(flagSet, outputFlagName)
	flagSet.StringVar(
		&f.Config,
		configFlagName,
		"",
		`The file or data to use for configuration.`,
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
	sourceOrModuleRef, err := buffetch.NewRefParser(container.Logger(), buffetch.RefParserWithProtoFileRefAllowed()).GetSourceOrModuleRef(ctx, input)
	if err != nil {
		return err
	}
	storageosProvider := bufcli.NewStorageosProvider(flags.DisableSymlinks)
	runner := command.NewRunner()
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
		runner,
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
		flags.ExcludePaths,
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
	// There are two cases where we need an image to filter the output:
	//   1) the input is a proto file reference
	//   2) ensuring that we are including the relevant imports
	//
	// In the first scenario, the imageConfigReader returns imageCongfigs that handle the filtering
	// for the proto file ref.
	//
	// To handle imports for all other references, unless we are excluding imports, we only want
	// to export those imports that are actually used. To figure this out, we build an image of images
	// and use the fact that something is in an image to determine if it is actually used.
	var images []bufimage.Image
	var mergedImage bufimage.Image
	_, isProtoFileRef := sourceOrModuleRef.(buffetch.ProtoFileRef)
	// We gate on flags.ExcludeImports/buffetch.ProtoFileRef so that we don't waste time building if the
	// result of the build is not relevant.
	if !flags.ExcludeImports {
		imageBuilder := bufimagebuild.NewBuilder(container.Logger())
		for _, moduleFileSet := range moduleFileSets {
			targetFileInfos, err := moduleFileSet.TargetFileInfos(ctx)
			if err != nil {
				return err
			}
			if len(targetFileInfos) == 0 {
				// This ModuleFileSet doesn't have any targets, so we shouldn't build
				// an image for it.
				continue
			}
			image, fileAnnotations, err := imageBuilder.Build(
				ctx,
				moduleFileSet,
				bufimagebuild.WithExcludeSourceCodeInfo(),
			)
			if err != nil {
				return err
			}
			if len(fileAnnotations) > 0 {
				// stderr since we do output to stdout potentially
				if err := bufanalysis.PrintFileAnnotations(
					container.Stderr(),
					fileAnnotations,
					bufanalysis.FormatText.String(),
				); err != nil {
					return err
				}
				return bufcli.ErrFileAnnotation
			}
			images = append(images, image)
		}
	} else if isProtoFileRef {
		// If the reference is a ProtoFileRef, we need to resolve the image for the reference,
		// since the image config reader distills down the reference to the file and its dependencies,
		// and also handles the #include_package_files option.
		imageConfigReader, err := bufcli.NewWireImageConfigReader(
			container,
			storageosProvider,
			runner,
			registryProvider,
		)
		if err != nil {
			return err
		}
		imageConfigs, fileAnnotations, err := imageConfigReader.GetImageConfigs(
			ctx,
			container,
			sourceOrModuleRef,
			flags.Config,
			flags.Paths,
			flags.ExcludePaths,
			false,
			true, // SourceCodeInfo is not needed here for outputting the source code
		)
		if err != nil {
			return err
		}
		if len(fileAnnotations) > 0 {
			if err := bufanalysis.PrintFileAnnotations(
				container.Stderr(),
				fileAnnotations,
				bufanalysis.FormatText.String(),
			); err != nil {
				return err
			}
		}
		for _, imageConfig := range imageConfigs {
			images = append(images, imageConfig.Image())
		}
	}
	mergedImage, err = bufimage.MergeImages(images...)
	if err != nil {
		return err
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
	// If we filtered on some paths, only use the targets.
	// Otherwise, we want to print everything, including potentially imports.
	// We can have duplicates across the ModuleFileSets and that's OK - they will
	// only be written once via writtenPaths, and we aren't building here.
	if len(flags.Paths) > 0 {
		fileInfosFunc = func(
			moduleFileSet bufmodule.ModuleFileSet,
			ctx context.Context,
		) ([]bufmoduleref.FileInfo, error) {
			return moduleFileSet.TargetFileInfos(ctx)
		}
	}
	writtenPaths := make(map[string]struct{})
	for _, moduleFileSet := range moduleFileSets {
		// If the reference was a proto file reference, we will use the image files as the basis
		// for outputting source files.
		if isProtoFileRef {
			for _, protoFileRefImageFile := range mergedImage.Files() {
				path := protoFileRefImageFile.Path()
				if _, ok := writtenPaths[path]; ok {
					continue
				}
				if flags.ExcludeImports && protoFileRefImageFile.IsImport() {
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
			if len(writtenPaths) == 0 {
				return errors.New("no .proto target files found")
			}
			return nil
		}
		fileInfos, err := fileInfosFunc(moduleFileSet, ctx)
		if err != nil {
			return err
		}
		for _, fileInfo := range fileInfos {
			path := fileInfo.Path()
			if _, ok := writtenPaths[path]; ok {
				continue
			}
			// If the file is not an import in some ModuleFileSet, it will
			// eventually be written via the iteration over moduleFileSets.
			if fileInfo.IsImport() {
				if flags.ExcludeImports {
					// Exclude imports, don't output here
					continue
				} else if mergedImage.GetFile(path) == nil {
					// We check the merged image to see if the path exists. If it does,
					// we use this import, so we want to output the file. If it doesn't,
					// continue.
					continue
				}
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
