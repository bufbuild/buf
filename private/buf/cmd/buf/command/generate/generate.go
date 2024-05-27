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

package generate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/buf/bufgen"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/buf/private/pkg/tracing"
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
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <input>",
		Short: "Generate code with protoc plugins",
		Long: `This command uses a template file of the shape:

    # buf.gen.yaml
    # The version of the generation template.
    # The valid values are v1beta1, v1 and v2.
    # Required.
    version: v2
    # The plugins to run.
    # Required.
    plugins:
        # Use the plugin hosted at buf.build/protocolbuffers/go at version v1.28.1.
        # If version is omitted, uses the latest version of the plugin.
        # One of "remote", "local" and "protoc_builtin" is required.
      - remote: buf.build/protocolbuffers/go:v1.28.1
        # The relative output directory.
        # Required.
        out: gen/go
        # The revision of the remote plugin to use, a sequence number that Buf
        # increments when rebuilding or repackaging the plugin.
        revision: 4
        # Any options to provide to the plugin.
        # This can be either a single string or a list of strings.
        # Optional.
        opt: paths=source_relative
        # Whether to generate code for imported files as well.
        # Optional.
        include_imports: false
        # Whether to generate code for the well-known types.
        # Optional.
        include_wkt: false

        # The name of a local plugin if discoverable in "${PATH}" or its path in the file system.
      - local: protoc-gen-es
        out: gen/es
        include_imports: true
        include_wkt: true

        # The full invocation of a local plugin can be specified as a list.
      - local: ["go", "run", "path/to/plugin.go"]
        out: gen/plugin
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
        #   This is also the only strategy for remote plugins.
        #
        # If omitted, "directory" is used. Most users should not need to set this option.
        # Optional.
        strategy: directory

        # "protoc_builtin" specifies a plugin that comes with protoc, without the "protoc-gen-" prefix.
      - protoc_builtin: java
        out: gen/java
        # Path to protoc. If not specified, the protoc installation in "${PATH}" is used.
        # Optional.
        protoc_path: path/to/protoc

    # Managed mode modifies file options and/or field options on the fly.
    managed:
      # Enables managed mode.
      enabled: true

      # Each override rule specifies an option, the value for this option and
      # optionally the files/fields for which the override is applied.
      #
      # The accepted file options are:
      #  - java_package
      #  - java_package_prefix
      #  - java_package_suffix
      #  - java_multiple_files
      #  - java_outer_classname
      #  - java_string_check_utf8
      #  - go_package
      #  - go_package_prefix
      #  - optimize_for
      #  - csharp_namespace
      #  - csharp_namespace_prefix
      #  - ruby_package
      #  - ruby_package_suffix
      #  - objc_class_prefix
      #  - php_namespace
      #  - php_metadata_namespace
      #  - php_metadata_namespace_suffix
      #  - cc_enable_arenas
      #
      # An override rule can apply to a field option.
      # The accepted field options are:
      #  - jstype
      #
      # If multple overrides for the same option apply to a file or field,
      # the last rule takes effect.
      # Optional.
      override:
          # Sets "go_package_prefix" to "foo/bar/baz" for all files.
        - file_option: go_package_prefix
          value: foo/bar/baz

          # Sets "java_package_prefix" to "net.foo" for files in "buf.build/foo/bar".
        - file_option: java_package_prefix
          value: net.foo
          module: buf.build/foo/bar

          # Sets "java_package_prefix" to "dev" for "file.proto".
          # This overrides the value "net.foo" for "file.proto" from the previous rule.
        - file_option: java_package_prefix
          value: dev
          module: buf.build/foo/bar
          path: file.proto

          # Sets "go_package" to "x/y/z" for all files in directory "x/y/z".
        - file_option: go_package
          value: foo/bar/baz
          path: x/y/z

          # Sets a field's "jstype" to "JS_NORMAL".
        - field_option: jstype
          value: JS_STRING
          field: foo.v1.Bar.baz

      # Disables managed mode under certain conditions.
      # Takes precedence over "overrides".
      # Optional.
      disable:
          # Do not modify any options for files in this module.
        - module: buf.build/googleapis/googleapis

          # Do not modify any options for this file.
        - module: buf.build/googleapis/googleapis
          path: foo/bar/file.proto

          # Do not modify "java_multiple_files" for any file
        - file_option: java_multiple_files

          # Do not modify "csharp_namespace" for files in this module.
        - module: buf.build/acme/weather
          file_option: csharp_namespace

    # The inputs to generate code for.
    # The inputs here are ignored if an input is specified as a command line argument.
    # Each input is one of "directory", "git_repo", "module", "tarball", "zip_archive",
    # "proto_file", "binary_image", "json_image", "text_image" and "yaml_image".
    # Optional.
    inputs:
        # The path to a directory.
      - directory: x/y/z

        # The URL of a Git repository.
      - git_repo: https://github.com/acme/weather.git
        # The branch to clone.
        # Optional.
        branch: dev
        # The subdirectory in the repository to use.
        # Optional.
        subdir: proto
        # How deep of a clone to perform.
        # Optional.
        depth: 30

        # The URL of a BSR module.
      - module: buf.build/acme/weather
        # Only generate code for these types.
        # Optional.
        types:
          - "foo.v1.User"
          - "foo.v1.UserService"
        # Only generate code for files in these paths.
        # If empty, include all paths.
        paths:
          - a/b/c
          - a/b/d
        # Do not generate code for files in these paths.
        exclude_paths:
          - a/b/c/x.proto
          - a/b/d/y.proto

        # The URL or path to a tarball.
      - tarball: a/b/x.tar.gz
        # The relative path within the archive to use as the base directory.
        # Optional.
        subdir: proto

        # The compression scheme, derived from the file extension if unspecified.
        # ".tgz" and ".tar.gz" extensions automatically use Gzip.
        # ".tar.zst" automatically uses Zstandard.
        # Optional.
        compression: gzip

        # Reads at the relative path and strips some number of components.
        # Optional.
        strip_components: 2

        # The URL or path to a zip archive.
      - zip_archive: https://github.com/googleapis/googleapis/archive/master.zip
        # The number of directories to strip.
        # Optional.
        strip_components: 1

        # The path to a specific proto file.
      - proto_file: foo/bar/baz.proto
        # Whether to generate code for files in the same package as well, default to false.
        # Optional.
        include_package_files: true

        # A Buf image in binary format.
        # Other image formats are "yaml_image", "text_image" and "json_image".
      - binary_image: image.binpb.gz
        # The compression scheme of the image file, derived from file extension if unspecified.
        # Optional.
        compression: gzip

As an example, here's a typical "buf.gen.yaml" go and grpc, assuming
"protoc-gen-go" and "protoc-gen-go-grpc" are on your "$PATH":

    # buf.gen.yaml
    version: v2
    plugins:
      - local: protoc-gen-go
        out: gen/go
        opt: paths=source_relative
      - local: protoc-gen-go-grpc
        out: gen/go
        opt:
          - paths=source_relative
          - require_unimplemented_servers=false

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

    $ buf generate --template '{"version":"v2","plugins":[{"local":"protoc-gen-go","out":"gen/go"}]}'

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
	Template               string
	BaseOutDirPath         string
	ErrorFormat            string
	Files                  []string
	Config                 string
	Paths                  []string
	IncludeImportsOverride *bool
	IncludeWKTOverride     *bool
	ExcludePaths           []string
	DisableSymlinks        bool
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
	bindBoolPointer(
		flagSet,
		includeImportsFlagName,
		&f.IncludeImportsOverride,
		"Also generate all imports except for Well-Known Types",
	)
	bindBoolPointer(
		flagSet,
		includeWKTFlagName,
		&f.IncludeWKTOverride,
		fmt.Sprintf(
			"Also generate Well-Known Types. Cannot be set to true without setting --%s to true",
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
	container appext.Container,
	flags *flags,
) (retErr error) {
	logger := container.Logger()
	if flags.IncludeWKTOverride != nil &&
		*flags.IncludeWKTOverride &&
		(flags.IncludeImportsOverride == nil || !*flags.IncludeImportsOverride) {
		// You need to set --include-imports to true if you set --include-wkt to true, which isn’t great.
		// The alternative is to have --include-wkt implicitly set --include-imports, but this could be surprising.
		// Or we could rename --include-wkt to --include-imports-and/with-wkt. But the summary is that the flag
		// only makes sense in the context of including imports.
		return appcmd.NewInvalidArgumentErrorf("Cannot set --%s to true without setting --%s to true", includeWKTFlagName, includeImportsFlagName)
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
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	bufGenYAMLFile, err := readBufGenYAMLFile(ctx, storageosProvider, flags.Template)
	if err != nil {
		return err
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
	if flags.IncludeImportsOverride != nil {
		generateOptions = append(
			generateOptions,
			bufgen.GenerateWithIncludeImportsOverride(*flags.IncludeImportsOverride),
		)
	}
	if flags.IncludeWKTOverride != nil {
		generateOptions = append(
			generateOptions,
			bufgen.GenerateWithIncludeWellKnownTypesOverride(*flags.IncludeWKTOverride),
		)
	}
	return bufgen.NewGenerator(
		logger,
		tracing.NewTracer(container.Tracer()),
		storageosProvider,
		command.NewRunner(),
		clientConfig,
	).Generate(
		ctx,
		container,
		bufGenYAMLFile.GenerateConfig(),
		images,
		generateOptions...,
	)
}

func readBufGenYAMLFile(
	ctx context.Context,
	storageosProvider storageos.Provider,
	templatePath string,
) (bufconfig.BufGenYAMLFile, error) {
	templatePathExtension := filepath.Ext(templatePath)
	switch {
	case templatePath == "":
		bucket, err := storageosProvider.NewReadWriteBucket(".", storageos.ReadWriteBucketWithSymlinksIfSupported())
		if err != nil {
			return nil, err
		}
		return bufconfig.GetBufGenYAMLFileForPrefix(ctx, bucket, ".")
	case templatePathExtension == ".yaml" || templatePathExtension == ".yml" || templatePathExtension == ".json":
		// We should not read from a bucket at "." because this path can jump context.
		configFile, err := os.Open(templatePath)
		if err != nil {
			return nil, err
		}
		defer configFile.Close()
		return bufconfig.ReadBufGenYAMLFile(configFile)
	default:
		return bufconfig.ReadBufGenYAMLFile(strings.NewReader(templatePath))
	}
}

func getInputImages(
	ctx context.Context,
	logger *zap.Logger,
	controller bufctl.Controller,
	inputSpecified string,
	bufGenYAMLFile bufconfig.BufGenYAMLFile,
	moduleConfigOverride string,
	targetPathsOverride []string,
	excludePathsOverride []string,
	includeTypesOverride []string,
) ([]bufimage.Image, error) {
	// If input is specified on the command line, we use that. If input is not
	// specified on the command line, use the default input.
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
			bufctl.WithTargetPaths(targetPathsOverride, excludePathsOverride),
			bufctl.WithImageTypes(includeTypes),
		)
		if err != nil {
			return nil, err
		}
		return []bufimage.Image{inputImage}, nil
	}
	var inputImages []bufimage.Image
	for _, inputConfig := range bufGenYAMLFile.InputConfigs() {
		targetPaths := inputConfig.TargetPaths()
		if len(targetPathsOverride) > 0 {
			targetPaths = targetPathsOverride
		}
		excludePaths := inputConfig.ExcludePaths()
		if len(excludePathsOverride) > 0 {
			excludePaths = excludePathsOverride
		}
		// In V2 we do not need to look at generateTypeConfig.IncludeTypes()
		// because it is always nil.
		includeTypes := inputConfig.IncludeTypes()
		if len(includeTypesOverride) > 0 {
			includeTypes = includeTypesOverride
		}
		inputImage, err := controller.GetImageForInputConfig(
			ctx,
			inputConfig,
			bufctl.WithConfigOverride(moduleConfigOverride),
			bufctl.WithTargetPaths(targetPaths, excludePaths),
			bufctl.WithImageTypes(includeTypes),
		)
		if err != nil {
			return nil, err
		}
		inputImages = append(inputImages, inputImage)
	}
	return inputImages, nil
}

// TODO FUTURE: where does this belong? A flagsext package?
// value must not be nil.
func bindBoolPointer(flagSet *pflag.FlagSet, name string, value **bool, usage string) {
	flag := flagSet.VarPF(
		&boolPointerValue{
			valuePointer: value,
		},
		name,
		"",
		usage,
	)
	flag.NoOptDefVal = "true"
}

// Implements pflag.Value.
type boolPointerValue struct {
	// This must not be nil at construction time.
	valuePointer **bool
}

func (b *boolPointerValue) Type() string {
	// From the CLI users' perspective, this is just a bool.
	return "bool"
}

func (b *boolPointerValue) String() string {
	if *b.valuePointer == nil {
		// From the CLI users' perspective, this is just false.
		return "false"
	}
	return strconv.FormatBool(**b.valuePointer)
}

func (b *boolPointerValue) Set(value string) error {
	parsedValue, err := strconv.ParseBool(value)
	if err != nil {
		return err
	}
	*b.valuePointer = &parsedValue
	return nil
}
