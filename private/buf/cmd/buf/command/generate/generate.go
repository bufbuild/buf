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
	"os"
	"path/filepath"
	"strings"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/buf/bufgen"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufwasm"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
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
	migrateFlagName             = "migrate"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appflag.SubCommandBuilder,
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
	Migrate         bool
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
	flagSet.BoolVar(
		&f.Migrate,
		migrateFlagName,
		false,
		"Migrate the generation template to the latest version",
	)
	_ = flagSet.MarkDeprecated(typeDeprecatedFlagName, fmt.Sprintf("Use --%s instead", typeFlagName))
	_ = flagSet.MarkHidden(typeDeprecatedFlagName)
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
	input, err := bufcli.GetInputValue(container, flags.InputHashtag, "")
	if err != nil {
		return err
	}
	var storageosProvider storageos.Provider
	if flags.DisableSymlinks {
		storageosProvider = storageos.NewProvider()
	} else {
		storageosProvider = storageos.NewProvider(storageos.ProviderWithSymlinks())
	}
	controller, err := bufcli.NewController(
		container,
		bufctl.WithDisableSymlinks(flags.DisableSymlinks),
		bufctl.WithFileAnnotationErrorFormat(flags.ErrorFormat),
	)
	if err != nil {
		return err
	}
	wasmPluginExecutor, err := bufwasm.NewPluginExecutor(
		filepath.Join(container.CacheDirPath(), bufcli.WASMCompilationCacheDir))
	if err != nil {
		return err
	}
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	var bufGenYAMLFile bufconfig.BufGenYAMLFile
	templatePathExtension := filepath.Ext(flags.Template)
	switch {
	case flags.Template == "":
		bucket, err := storageosProvider.NewReadWriteBucket(".", storageos.ReadWriteBucketWithSymlinksIfSupported())
		if err != nil {
			return err
		}
		bufGenYAMLFile, err = bufconfig.GetBufGenYAMLFileForPrefix(ctx, bucket, ".")
		if err != nil {
			return err
		}
		if flags.Migrate && bufGenYAMLFile.FileVersion() != bufconfig.FileVersionV2 {
			migratedBufGenYAMLFile, err := getBufGenYAMLFileWithFlagEquivalence(
				ctx,
				logger,
				bufGenYAMLFile,
				input,
				*flags,
			)
			if err != nil {
				return err
			}
			if err := bufconfig.PutBufGenYAMLFileForPrefix(ctx, bucket, ".", migratedBufGenYAMLFile); err != nil {
				return err
			}
			// TODO: perhaps print a message
		}
	case templatePathExtension == ".yaml" || templatePathExtension == ".yml" || templatePathExtension == ".json":
		// We should not read from a bucket at "." because this path can jump context.
		configFile, err := os.Open(flags.Template)
		if err != nil {
			return err
		}
		bufGenYAMLFile, err = bufconfig.ReadBufGenYAMLFile(configFile)
		if err != nil {
			return err
		}
		if flags.Migrate && bufGenYAMLFile.FileVersion() != bufconfig.FileVersionV2 {
			migratedBufGenYAMLFile, err := getBufGenYAMLFileWithFlagEquivalence(
				ctx,
				logger,
				bufGenYAMLFile,
				input,
				*flags,
			)
			if err != nil {
				return err
			}
			if err := bufconfig.WriteBufGenYAMLFile(configFile, migratedBufGenYAMLFile); err != nil {
				return err
			}
			// TODO: perhaps print a message
		}
	default:
		bufGenYAMLFile, err = bufconfig.ReadBufGenYAMLFile(strings.NewReader(flags.Template))
		if err != nil {
			return err
		}
		if flags.Migrate && bufGenYAMLFile.FileVersion() != bufconfig.FileVersionV2 {
			return fmt.Errorf(
				"invalid template: %q, migration can only apply to a file on disk with extension .yaml, .yml or .json",
				flags.Template,
			)
		}
	}
	images, err := getInputImages(
		ctx,
		logger,
		controller,
		input,
		bufGenYAMLFile,
		flags.Config,
		flags.Paths,
		flags.ExcludePaths,
		flags.Types,
	)
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
	return bufgen.NewGenerator(
		logger,
		storageosProvider,
		command.NewRunner(),
		wasmPluginExecutor,
		clientConfig,
	).Generate(
		ctx,
		container,
		bufGenYAMLFile.GenerateConfig(),
		images,
		generateOptions...,
	)
}

func getInputImages(
	ctx context.Context,
	logger *zap.Logger,
	controller bufctl.Controller,
	inputSpecified string,
	bufGenYAMLFile bufconfig.BufGenYAMLFile,
	moduleConfigOverride string,
	includePathsOverride []string,
	excludePathsOverride []string,
	includeTypesOverride []string,
) ([]bufimage.Image, error) {
	var inputImages []bufimage.Image
	// If input is specified on the command line, we use that. If input is not
	// specified on the command line, but the config has no inputs, use the default input.
	if inputSpecified != "" || len(bufGenYAMLFile.InputConfigs()) == 0 {
		input := "."
		if inputSpecified != "" {
			input = inputSpecified
		}
		var includeTypes []string
		if typesConfig := bufGenYAMLFile.GenerateConfig().GenerateTypeConfig(); typesConfig != nil {
			includeTypes = typesConfig.IncludeTypes()
		}
		if len(includeTypesOverride) > 0 {
			includeTypes = includeTypesOverride
		}
		inputImage, err := controller.GetImage(
			ctx,
			input,
			bufctl.WithConfigOverride(moduleConfigOverride),
			bufctl.WithTargetPaths(includePathsOverride, excludePathsOverride),
			bufctl.WithImageTypes(includeTypes),
		)
		if err != nil {
			return nil, err
		}
		inputImages = []bufimage.Image{inputImage}
	} else {
		for _, inputConfig := range bufGenYAMLFile.InputConfigs() {
			includePaths := inputConfig.IncludePaths()
			if len(includePathsOverride) > 0 {
				includePaths = includePathsOverride
			}
			excludePaths := inputConfig.ExcludePaths()
			if len(excludePathsOverride) > 0 {
				excludePaths = excludePathsOverride
			}
			// In V2 we do not need to look at generateTypeConfig.IncludeTypes()
			// because it is always nil.
			// TODO: document the above in godoc
			includeTypes := inputConfig.IncludeTypes()
			if len(includeTypesOverride) > 0 {
				includeTypes = includeTypesOverride
			}
			inputImage, err := controller.GetImageForInputConfig(
				ctx,
				inputConfig,
				bufctl.WithConfigOverride(moduleConfigOverride),
				bufctl.WithTargetPaths(includePaths, excludePaths),
				bufctl.WithImageTypes(includeTypes),
			)
			if err != nil {
				return nil, err
			}
			inputImages = append(inputImages, inputImage)
		}
	}
	return inputImages, nil
}

// getBufGenYAMLFileWithFlagEquivalence returns a buf gen yaml file the same as
// the given one, except that it is overriden by flags.
// This is called only for migration. Input will be used regardless if it's empty.
func getBufGenYAMLFileWithFlagEquivalence(
	ctx context.Context,
	logger *zap.Logger,
	bufGenYAMLFile bufconfig.BufGenYAMLFile,
	input string,
	flags flags,
) (bufconfig.BufGenYAMLFile, error) {
	if input == "" {
		input = "."
	}
	inputConfig, err := buffetch.GetInputConfigForString(
		ctx,
		buffetch.NewRefParser(logger),
		input,
	)
	if err != nil {
		return nil, err
	}
	var includeTypes []string
	if bufGenYAMLFile.GenerateConfig().GenerateTypeConfig() != nil {
		includeTypes = bufGenYAMLFile.GenerateConfig().GenerateTypeConfig().IncludeTypes()
	}
	if len(flags.Types) > 0 {
		includeTypes = flags.Types
	}
	inputConfig, err = bufconfig.NewInputConfigWithTargets(
		inputConfig,
		flags.Paths,
		flags.ExcludePaths,
		includeTypes,
	)
	if err != nil {
		return nil, err
	}
	pluginConfigs, err := slicesext.MapError(
		bufGenYAMLFile.GenerateConfig().GeneratePluginConfigs(),
		func(pluginConfig bufconfig.GeneratePluginConfig) (bufconfig.GeneratePluginConfig, error) {
			return bufconfig.NewGeneratePluginWithIncludeImportsAndWKT(
				pluginConfig,
				flags.IncludeImports,
				flags.IncludeWKT,
			)
		},
	)
	if err != nil {
		return nil, err
	}
	generateConfig, err := bufconfig.NewGenerateConfig(
		pluginConfigs,
		bufGenYAMLFile.GenerateConfig().GenerateManagedConfig(),
		nil,
	)
	if err != nil {
		return nil, err
	}
	return bufconfig.NewBufGenYAMLFile(
		bufconfig.FileVersionV2,
		generateConfig,
		[]bufconfig.InputConfig{inputConfig},
	), nil
}
