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

package generate

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/buf/bufgen"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	templateFlagName            = "template"
	baseOutDirPathFlagName      = "output"
	baseOutDirPathFlagShortName = "o"
	errorFormatFlagName         = "error-format"
	configFlagName              = "config"
	pathsFlagName               = "path"
	includeImportsFlagName      = "include-imports"
	includeWKTFlagName          = "include-wkt"
	excludePathsFlagName        = "exclude-path"
	disableSymlinksFlagName     = "disable-symlinks"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <input>",
		Short: "Generate stubs for protoc plugins using a template.",
		Long: `This command uses a template file of the shape:

# The version of the generation template.
# Required.
# The valid values are v1beta1, v1.
version: v1
# The plugins to run. One of "name" and "remote" is required.
plugins:
    # The name of the plugin.
    # By default, buf generate will look for a binary named protoc-gen-NAME on your $PATH.
    # Alternatively, use a remote reference:
    # remote: buf.build/protocolbuffers/plugins/go:v1.27.0-1
  - name: go
    # The the relative output directory.
    # Required.
    out: gen/go
    # Any options to provide to the plugin.
    # This can be either a single string or a list of strings.
    # Optional.
    opt: paths=source_relative
    # The custom path to the plugin binary, if not protoc-gen-NAME on your $PATH.
    # Optional, and exclusive with "remote".
    path: custom-gen-go
    # The generation strategy to use. There are two options:
    #
    # 1. "directory"
    #
    #   This will result in buf splitting the input files by directory, and making separate plugin
    #   invocations in parallel. This is roughly the concurrent equivalent of:
    #
    #     for dir in $(find . -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq); do
    #       protoc -I . $(find "${dir}" -name '*.proto')
    #     done
    #
    #   Almost every Protobuf plugin either requires this, or works with this,
    #   and this is the recommended and default value.
    #
    # 2. "all"
    #
    #   This will result in buf making a single plugin invocation with all input files.
    #   This is roughly the equivalent of:
    #
    #     protoc -I . $(find . -name '*.proto')
    #
    #   This is needed for certain plugins that expect all files to be given at once.
    #
    # If omitted, "directory" is used. Most users should not need to set this option.
    # Optional.
    strategy: directory
  - name: java
    out: gen/java
    # Use the plugin hosted at buf.build/protocolbuffers/plugins/python at version v3.17.0-1.
    # If version is omitted, uses the latest version of the plugin.
  - remote: buf.build/protocolbuffers/plugins/python:v3.17.0-1
    out: gen/python

As an example, here's a typical "buf.gen.yaml" go and grpc, assuming
"protoc-gen-go" and "protoc-gen-go-grpc" are on your "$PATH":

version: v1
plugins:
  - name: go
    out: gen/go
    opt: paths=source_relative
  - name: go-grpc
    out: gen/go
    opt: paths=source_relative,require_unimplemented_servers=false

By default, buf generate will look for a file of this shape named
"buf.gen.yaml" in your current directory. This can be thought of as a template
for the set of plugins you want to invoke.

The first argument is the source, module, or image to generate from.
If no argument is specified, defaults to ".".

Call with:

# uses buf.gen.yaml as template, current directory as input
$ buf generate

# same as the defaults (template of "buf.gen.yaml", current directory as input)
$ buf generate --template buf.gen.yaml .

# --template also takes YAML or JSON data as input, so it can be used without a file
$ buf generate --template '{"version":"v1","plugins":[{"name":"go","out":"gen/go"}]}'

# download the repository and generate code stubs per the bar.yaml template
$ buf generate --template bar.yaml https://github.com/foo/bar.git

# generate to the bar/ directory, prepending bar/ to the out directives in the template
$ buf generate --template bar.yaml -o bar https://github.com/foo/bar.git

The paths in the template and the -o flag will be interpreted as relative to your
current directory, so you can place your template files anywhere.

If you only want to generate stubs for a subset of your input, you can do so via the --path flag:

# Only generate for the files in the directories proto/foo and proto/bar
$ buf generate --path proto/foo --path proto/bar

# Only generate for the files proto/foo/foo.proto and proto/foo/bar.proto
$ buf generate --path proto/foo/foo.proto --path proto/foo/bar.proto

# Only generate for the files in the directory proto/foo on your GitHub repository
$ buf generate --template buf.gen.yaml https://github.com/foo/bar.git --path proto/foo

Note that all paths must be contained within the same module. For example, if you have a
module in "proto", you cannot specify "--path proto", however "--path proto/foo" is allowed
as "proto/foo" is contained within "proto".

Plugins are invoked in the order they are specified in the template, but each plugin
has a per-directory parallel invocation, with results from each invocation combined
before writing the result.

Insertion points are processed in the order the plugins are specified in the template.
`,
		Args: cobra.MaximumNArgs(1),
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
	Template        string
	BaseOutDirPath  string
	ErrorFormat     string
	Files           []string
	Config          string
	Paths           []string
	IncludeImports  bool
	IncludeWKT      bool
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
	bufcli.BindPaths(flagSet, &f.Paths, pathsFlagName)
	bufcli.BindExcludePaths(flagSet, &f.ExcludePaths, excludePathsFlagName)
	flagSet.BoolVar(
		&f.IncludeImports,
		includeImportsFlagName,
		false,
		"Also generate all imports except for Well-Known Types.",
	)
	flagSet.BoolVar(
		&f.IncludeWKT,
		includeWKTFlagName,
		false,
		fmt.Sprintf(
			"Also generate Well-Known Types. Cannot be set without --%s.",
			includeImportsFlagName,
		),
	)
	flagSet.StringVar(
		&f.Template,
		templateFlagName,
		"",
		`The generation template file or data to use. Must be in either YAML or JSON format.`,
	)
	flagSet.StringVarP(
		&f.BaseOutDirPath,
		baseOutDirPathFlagName,
		baseOutDirPathFlagShortName,
		".",
		`The base directory to generate to. This is prepended to the out directories in the generation template.`,
	)
	flagSet.StringVar(
		&f.ErrorFormat,
		errorFormatFlagName,
		"text",
		fmt.Sprintf(
			"The format for build errors, printed to stderr. Must be one of %s.",
			stringutil.SliceToString(bufanalysis.AllFormatStrings),
		),
	)
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
) (retErr error) {
	logger := container.Logger()
	if flags.IncludeWKT && !flags.IncludeImports {
		// You need to set --include-imports if you set --include-wkt, which isn’t great. The alternative is to have
		// --include-wkt implicitly set --include-imports, but this could be surprising. Or we could rename
		// --include-wkt to --include-imports-and/with-wkt. But the summary is that the flag only makes sense
		// in the context of including imports.
		return appcmd.NewInvalidArgumentErrorf("Cannot set --%s without --%s", includeWKTFlagName, includeImportsFlagName)
	}
	if err := bufcli.ValidateErrorFormatFlag(flags.ErrorFormat, errorFormatFlagName); err != nil {
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
	readWriteBucket, err := storageosProvider.NewReadWriteBucket(
		".",
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return err
	}
	genConfig, err := bufgen.ReadConfig(
		ctx,
		bufgen.NewProvider(logger),
		readWriteBucket,
		bufgen.ReadConfigWithOverride(flags.Template),
	)
	if err != nil {
		return err
	}
	registryProvider, err := bufcli.NewRegistryProvider(ctx, container)
	if err != nil {
		return err
	}
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
		ref,
		flags.Config,
		flags.Paths,        // we filter on files
		flags.ExcludePaths, // we exclude these paths
		false,              // input files must exist
		false,              // we must include source info for generation
	)
	if err != nil {
		return err
	}
	if len(fileAnnotations) > 0 {
		if err := bufanalysis.PrintFileAnnotations(container.Stderr(), fileAnnotations, flags.ErrorFormat); err != nil {
			return err
		}
		return bufcli.ErrFileAnnotation
	}
	images := make([]bufimage.Image, 0, len(imageConfigs))
	for _, imageConfig := range imageConfigs {
		images = append(images, imageConfig.Image())
	}
	image, err := bufimage.MergeImages(images...)
	if err != nil {
		return err
	}
	generateOptions := []bufgen.GenerateOption{
		bufgen.GenerateWithBaseOutDirPath(flags.BaseOutDirPath),
	}
	if flags.IncludeImports {
		generateOptions = append(
			generateOptions,
			bufgen.GenerateWithIncludeImports(),
		)
	}
	if flags.IncludeWKT {
		generateOptions = append(
			generateOptions,
			bufgen.GenerateWithIncludeWellKnownTypes(),
		)
	}
	return bufgen.NewGenerator(
		logger,
		storageosProvider,
		runner,
		registryProvider,
	).Generate(
		ctx,
		container,
		genConfig,
		image,
		generateOptions...,
	)
}
