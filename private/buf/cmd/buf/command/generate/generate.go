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

package generate

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/buf/bufgen"
	"github.com/bufbuild/buf/private/buf/bufgen/bufgenv1"
	"github.com/bufbuild/buf/private/buf/bufgen/bufgenv2"
	"github.com/bufbuild/buf/private/buf/bufwire"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimageutil"
	"github.com/bufbuild/buf/private/bufpkg/bufwasm"
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
	typeFlagName                = "type"
	typeDeprecatedFlagName      = "include-types"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <input>",
		Short: "Generate code with protoc plugins",
		Long: `This command uses a template file of the shape:

    # buf.gen.yaml
    # The version of the generation template.
    # Required.
    # The valid values are v1beta1, v1.
    version: v1
    # The plugins to run. "plugin" is required.
    plugins:
        # The name of the plugin.
        # By default, buf generate will look for a binary named protoc-gen-NAME on your $PATH.
        # Alternatively, use a remote plugin:
        # plugin: buf.build/protocolbuffers/go:v1.28.1
      - plugin: go
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
      - plugin: java
        out: gen/java
        # Use the plugin hosted at buf.build/protocolbuffers/python at version v21.9.
        # If version is omitted, uses the latest version of the plugin.
      - plugin: buf.build/protocolbuffers/python:v21.9
        out: gen/python

As an example, here's a typical "buf.gen.yaml" go and grpc, assuming
"protoc-gen-go" and "protoc-gen-go-grpc" are on your "$PATH":

    # buf.gen.yaml
    version: v1
    plugins:
      - plugin: go
        out: gen/go
        opt: paths=source_relative
      - plugin: go-grpc
        out: gen/go
        opt: paths=source_relative,require_unimplemented_servers=false

By default, buf generate will look for a file of this shape named
"buf.gen.yaml" in your current directory. This can be thought of as a template
for the set of plugins you want to invoke.

The first argument is the source, module, or image to generate from.
Defaults to "." if no argument is specified.

Use buf.gen.yaml as template, current directory as input:

    $ buf generate

Same as the defaults (template of "buf.gen.yaml", current directory as input):

    $ buf generate --template buf.gen.yaml .

The --template flag also takes YAML or JSON data as input, so it can be used without a file:

    $ buf generate --template '{"version":"v1","plugins":[{"plugin":"go","out":"gen/go"}]}'

Download the repository and generate code stubs per the bar.yaml template:

    $ buf generate --template bar.yaml https://github.com/foo/bar.git

Generate to the bar/ directory, prepending bar/ to the out directives in the template:

    $ buf generate --template bar.yaml -o bar https://github.com/foo/bar.git

The paths in the template and the -o flag will be interpreted as relative to the
current directory, so you can place your template files anywhere.

If you only want to generate stubs for a subset of your input, you can do so via the --path. e.g.

Only generate for the files in the directories proto/foo and proto/bar:

    $ buf generate --path proto/foo --path proto/bar

Only generate for the files proto/foo/foo.proto and proto/foo/bar.proto:

    $ buf generate --path proto/foo/foo.proto --path proto/foo/bar.proto

Only generate for the files in the directory proto/foo on your git repository:

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
	// We may be able to bind two flags to one string slice but I don't
	// want to find out what will break if we do.
	Types           []string
	TypesDeprecated []string
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
		"Also generate all imports except for Well-Known Types",
	)
	flagSet.BoolVar(
		&f.IncludeWKT,
		includeWKTFlagName,
		false,
		fmt.Sprintf(
			"Also generate Well-Known Types. Cannot be set without --%s",
			includeImportsFlagName,
		),
	)
	flagSet.StringVar(
		&f.Template,
		templateFlagName,
		"",
		`The generation template file or data to use. Must be in either YAML or JSON format`,
	)
	flagSet.StringVarP(
		&f.BaseOutDirPath,
		baseOutDirPathFlagName,
		baseOutDirPathFlagShortName,
		".",
		`The base directory to generate to. This is prepended to the out directories in the generation template`,
	)
	flagSet.StringVar(
		&f.ErrorFormat,
		errorFormatFlagName,
		"text",
		fmt.Sprintf(
			"The format for build errors, printed to stderr. Must be one of %s",
			stringutil.SliceToString(bufanalysis.AllFormatStrings),
		),
	)
	flagSet.StringVar(
		&f.Config,
		configFlagName,
		"",
		`The buf.yaml file or data to use for configuration`,
	)
	flagSet.StringSliceVar(
		&f.Types,
		typeFlagName,
		nil,
		"The types (package, message, enum, extension, service, method) that should be included in this image. When specified, the resulting image will only include descriptors to describe the requested types. Flag usage overrides buf.gen.yaml",
	)
	flagSet.StringSliceVar(
		&f.TypesDeprecated,
		typeDeprecatedFlagName,
		nil,
		"The types (package, message, enum, extension, service, method) that should be included in this image. When specified, the resulting image will only include descriptors to describe the requested types. Flag usage overrides buf.gen.yaml",
	)
	_ = flagSet.MarkDeprecated(typeDeprecatedFlagName, fmt.Sprintf("Use --%s instead", typeFlagName))
	_ = flagSet.MarkHidden(typeDeprecatedFlagName)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) (retErr error) {
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
	storageosProvider := bufcli.NewStorageosProvider(flags.DisableSymlinks)
	readWriteBucket, err := storageosProvider.NewReadWriteBucket(
		".",
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return err
	}
	logger := container.Logger()
	configDataProvider := bufgen.NewConfigDataProvider(logger)
	configVersion, err := bufgen.ReadConfigVersion(
		ctx,
		logger,
		configDataProvider,
		readWriteBucket,
		bufgen.ReadConfigWithOverride(flags.Template),
	)
	if err != nil {
		return err
	}
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

	var includedTypesFromCLI []string
	if len(flags.Types) > 0 || len(flags.TypesDeprecated) > 0 {
		// command-line flags take precedence
		includedTypesFromCLI = append(flags.Types, flags.TypesDeprecated...)
	}

	var (
		inputImages   []bufimage.Image
		imageModifier bufimagemodify.Modifier
		plugins       []bufgen.PluginConfig
	)
	switch configVersion {
	case bufgen.V2Version:
		genConfigV2, err := bufgenv2.ReadConfigV2(
			ctx,
			logger,
			configDataProvider,
			readWriteBucket,
			bufgen.ReadConfigWithOverride(flags.Template),
		)
		if err != nil {
			return err
		}
		// TODO: implement managed mode
		imageModifier = nopModifier{}
		plugins = genConfigV2.Plugins
		if !bufcli.IsInputSpecified(container, flags.InputHashtag) {
			inputRef, err := getInputRefFromCLI(
				ctx,
				container,
				flags.InputHashtag,
			)
			if err != nil {
				return err
			}
			inputImage, err := getInputImage(
				ctx,
				container,
				inputRef,
				imageConfigReader,
				flags.Config,
				flags.Paths,
				flags.ExcludePaths,
				flags.ErrorFormat,
				includedTypesFromCLI,
			)
			if err != nil {
				return err
			}
			inputImages = append(inputImages, inputImage)
			break
		}
		for _, inputConfig := range genConfigV2.Inputs {
			includePaths := inputConfig.IncludePaths
			if len(flags.Paths) > 0 {
				includePaths = flags.Paths
			}
			excludePaths := inputConfig.ExcludePaths
			if len(flags.ExcludePaths) > 0 {
				excludePaths = flags.ExcludePaths
			}
			includedTypes := inputConfig.Types
			if len(includedTypesFromCLI) > 0 {
				includedTypes = includedTypesFromCLI
			}
			inputImage, err := getInputImage(
				ctx,
				container,
				inputConfig.InputRef,
				imageConfigReader,
				flags.Config,
				includePaths,
				excludePaths,
				flags.ErrorFormat,
				includedTypes,
			)
			if err != nil {
				return err
			}
			inputImages = append(inputImages, inputImage)
		}
	case bufgen.V1Version, bufgen.V1Beta1Version:
		genConfigV1, err := bufgenv1.ReadConfigV1(
			ctx,
			logger,
			configDataProvider,
			readWriteBucket,
			bufgen.ReadConfigWithOverride(flags.Template),
		)
		if err != nil {
			return err
		}
		if imageModifier, err = bufgenv1.NewModifier(
			logger,
			genConfigV1,
		); err != nil {
			return err
		}
		plugins = genConfigV1.PluginConfigs
		inputRef, err := getInputRefFromCLI(
			ctx,
			container,
			flags.InputHashtag,
		)
		if err != nil {
			return err
		}
		var includedTypes []string
		if typesConfig := genConfigV1.TypesConfig; typesConfig != nil {
			includedTypes = typesConfig.Include
		}
		if len(includedTypesFromCLI) > 0 {
			includedTypes = includedTypesFromCLI
		}
		inputImage, err := getInputImage(
			ctx,
			container,
			inputRef,
			imageConfigReader,
			flags.Config,
			flags.Paths,
			flags.ExcludePaths,
			flags.ErrorFormat,
			includedTypes,
		)
		if err != nil {
			return err
		}
		inputImages = append(inputImages, inputImage)
	default:
		return fmt.Errorf(`no version set. Please add "version: %s"`, bufgen.V2Version)
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
	wasmEnabled, err := bufcli.IsAlphaWASMEnabled(container)
	if err != nil {
		return err
	}
	if wasmEnabled {
		generateOptions = append(
			generateOptions,
			bufgen.GenerateWithWASMEnabled(),
		)
	}
	wasmPluginExecutor, err := bufwasm.NewPluginExecutor(
		filepath.Join(
			container.CacheDirPath(),
			bufcli.WASMCompilationCacheDir,
		),
	)
	if err != nil {
		return err
	}
	generator := bufgen.NewGenerator(
		logger,
		storageosProvider,
		runner,
		wasmPluginExecutor,
		clientConfig,
	)
	for _, image := range inputImages {
		if err := imageModifier.Modify(
			ctx,
			image,
		); err != nil {
			return err
		}
		if err := generator.Generate(
			ctx,
			container,
			plugins,
			image,
			generateOptions...,
		); err != nil {
			return err
		}
	}
	return nil
}

func getInputRefFromCLI(
	ctx context.Context,
	container appflag.Container,
	inputHashtag string,
) (buffetch.Ref, error) {
	input, err := bufcli.GetInputValue(container, inputHashtag, ".")
	if err != nil {
		return nil, err
	}
	return buffetch.NewRefParser(container.Logger()).GetRef(ctx, input)
}

func getInputImage(
	ctx context.Context,
	container appflag.Container,
	ref buffetch.Ref,
	imageConfigReader bufwire.ImageConfigReader,
	configLocationOverride string,
	includedPaths []string,
	excludedPaths []string,
	errorFormat string,
	includedTypes []string,
) (bufimage.Image, error) {
	imageConfigs, fileAnnotations, err := imageConfigReader.GetImageConfigs(
		ctx,
		container,
		ref,
		configLocationOverride, // flags.Config,
		includedPaths,          // flags.Paths,            // we filter on files
		excludedPaths,          // flags.ExcludePaths,     // we exclude these paths
		false,                  // input files must exist
		false,                  // we must include source info for generation
	)
	if err != nil {
		return nil, err
	}
	if len(fileAnnotations) > 0 {
		if err := bufanalysis.PrintFileAnnotations(container.Stderr(), fileAnnotations, errorFormat); err != nil {
			return nil, err
		}
		return nil, bufcli.ErrFileAnnotation
	}
	images := make([]bufimage.Image, 0, len(imageConfigs))
	for _, imageConfig := range imageConfigs {
		images = append(images, imageConfig.Image())
	}
	image, err := bufimage.MergeImages(images...)
	if err != nil {
		return nil, err
	}
	if len(includedTypes) > 0 {
		image, err = bufimageutil.ImageFilteredByTypes(image, includedTypes...)
		if err != nil {
			return nil, err
		}
	}
	return image, nil
}

// A placeholder for V2 managed mode image modifier.
// TODO: remove this
type nopModifier struct{}

func (a nopModifier) Modify(
	ctx context.Context,
	image bufimage.Image,
) error {
	return nil
}
