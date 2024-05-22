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

package export

import (
	"context"
	"errors"
	"io/fs"
	"os"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/gen/data/datawkt"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/syserror"
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
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <source>",
		Short: "Export proto files from one location to another",
		Long: bufcli.GetSourceOrModuleLong(`the source or module to export`) + `

Examples:

Export proto files in <source> to an output directory.

    $ buf export <source> --output=<output-dir>

Export current directory to another local directory.

    $ buf export . --output=<output-dir>

Export the latest remote module to a local directory.

    $ buf export <buf.build/owner/repository> --output=<output-dir>

Export a specific version of a remote module to a local directory.

    $ buf export <buf.build/owner/repository:ref> --output=<output-dir>

Export a git repo to a local directory.

    $ buf export https://github.com/owner/repository.git --output=<output-dir>
`,
		Args: appcmd.MaximumNArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container, flags)
			},
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
		`The output directory for exported files`,
	)
	_ = appcmd.MarkFlagRequired(flagSet, outputFlagName)
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
) error {
	input, err := bufcli.GetInputValue(container, flags.InputHashtag, ".")
	if err != nil {
		return err
	}
	controller, err := bufcli.NewController(
		container,
		bufctl.WithDisableSymlinks(flags.DisableSymlinks),
	)
	if err != nil {
		return err
	}
	workspace, err := controller.GetWorkspace(
		ctx,
		input,
		bufctl.WithTargetPaths(flags.Paths, flags.ExcludePaths),
		bufctl.WithConfigOverride(flags.Config),
	)
	if err != nil {
		return err
	}
	moduleReadBucket := bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles(workspace)

	if err := os.MkdirAll(flags.Output, 0755); err != nil {
		return err
	}
	var options []storageos.ProviderOption
	if !flags.DisableSymlinks {
		options = append(options, storageos.ProviderWithSymlinks())
	}
	readWriteBucket, err := storageos.NewProvider(options...).NewReadWriteBucket(
		flags.Output,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return err
	}

	// In the case where we are excluding imports, we are allowing users to specify an input
	// that may not have resolved imports (https://github.com/bufbuild/buf/issues/3002).
	// Thus we do not need to build the image, and instead we can return the non-import files
	// from the workspace.
	if flags.ExcludeImports {
		if err := moduleReadBucket.WalkFileInfos(
			ctx,
			func(fileInfo bufmodule.FileInfo) error {
				moduleFile, err := moduleReadBucket.GetFile(ctx, fileInfo.Path())
				if err != nil {
					return syserror.Wrap(err)
				}
				if err := storage.CopyReadObject(ctx, readWriteBucket, moduleFile); err != nil {
					return multierr.Append(err, moduleFile.Close())
				}
				return moduleFile.Close()
			},
			bufmodule.WalkFileInfosWithOnlyTargetFiles(),
		); err != nil {
			return err
		}
		return nil
	}

	image, err := controller.GetImageForWorkspace(
		ctx,
		workspace,
		bufctl.WithImageExcludeSourceInfo(true),
		bufctl.WithImageExcludeImports(flags.ExcludeImports),
	)
	if err != nil {
		return err
	}
	imageFiles := image.Files()
	if len(imageFiles) == 0 {
		return errors.New("no .proto target files found")
	}
	for _, imageFile := range image.Files() {
		moduleFile, err := moduleReadBucket.GetFile(ctx, imageFile.Path())
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) && datawkt.Exists(imageFile.Path()) {
				// Images include all imports, including WKTs. WKTs may or may not exist as part of the Workspace. They are implicitly
				// added to Images if they are not present in a Module or its dependencies. However, we want to make sure that
				// we still export them if they were part of a Module, or were part of an explicit dependency (for example,
				// buf.build/protocolbuffers/wellknowntypes).
				//
				// This is the only case where a file may exist in the Image but not in the Workspace. Any other case where a file
				// does not exist is a system error.
				continue
			}
			return syserror.Wrap(err)
		}
		if err := storage.CopyReadObject(ctx, readWriteBucket, moduleFile); err != nil {
			return multierr.Append(err, moduleFile.Close())
		}
		if err := moduleFile.Close(); err != nil {
			return err
		}
	}
	return nil
}
