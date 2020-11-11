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

	"github.com/bufbuild/buf/internal/buf/bufanalysis"
	"github.com/bufbuild/buf/internal/buf/bufcli"
	"github.com/bufbuild/buf/internal/buf/bufconfig"
	"github.com/bufbuild/buf/internal/buf/buffetch"
	"github.com/bufbuild/buf/internal/buf/bufgen"
	"github.com/bufbuild/buf/internal/pkg/app/appcmd"
	"github.com/bufbuild/buf/internal/pkg/app/appflag"
	"github.com/bufbuild/buf/internal/pkg/stringutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	templateFlagName            = "template"
	baseOutDirPathFlagName      = "output"
	baseOutDirPathFlagShortName = "o"
	errorFormatFlagName         = "error-format"
	filesFlagName               = "file"
	configFlagName              = "config"

	// deprecated
	inputFlagName = "input"
	// deprecated
	inputConfigFlagName = "input-config"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appflag.Builder,
	moduleResolverReaderProvider bufcli.ModuleResolverReaderProvider,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <input>",
		Short: "Generate stubs for protoc plugins using a template.",
		Long: `This command uses a template file of the shape:

version: v1beta1         # required
plugins:
  - name: go             # required
    out: gen/go          # required
    opt: plugins=grpc    # optional
    path: custom-gen-go  # optional
  - name java
    out: gen/java

As an example, here's a typical "buf.gen.yaml" go and grpc, assuming
"protoc-gen-go" and "protoc-gen-go-grpc" are on your "$PATH":

version: v1beta1
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
$ buf generate --template '{"version":"v1beta1","plugins":[{"name":"go","out":"gen/go"}]}'

# download the repository, compile it, and generate per the bar.yaml template
$ buf generate --template bar.yaml https://github.com/foo/bar.git

# generate to the bar/ directory, prepending bar/ to the out directives in the template
$ buf generate --template bar.yaml -o bar https://github.com/foo/bar.git

The paths in the template and the -o flag will be interpreted as relative to your
current directory, so you can place your template files anywhere.

Plugins are invoked in the order they are specified in the template, but each plugin
has a per-directory parallel invocation, with results from each invocation combined
before writing the result. This is equivalent behavior to "buf protoc --by_dir".
`,
		Args: cobra.MaximumNArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appflag.Container) error {
				return run(ctx, container, flags, moduleResolverReaderProvider)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	Template       string
	BaseOutDirPath string
	ErrorFormat    string
	Files          []string
	Config         string

	// deprecated
	Input string
	// deprecated
	InputConfig string
	// special
	InputHashtag string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	bufcli.BindInputHashtag(flagSet, &f.InputHashtag)
	flagSet.StringVar(
		&f.Template,
		templateFlagName,
		"buf.gen.yaml",
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
	flagSet.StringSliceVar(
		&f.Files,
		filesFlagName,
		nil,
		`Limit to specific files. This is an advanced feature and is not recommended.`,
	)
	flagSet.StringVar(
		&f.Config,
		configFlagName,
		"",
		`The config file or data to use.`,
	)

	// deprecated
	flagSet.StringVar(
		&f.Input,
		inputFlagName,
		"",
		fmt.Sprintf(
			`The source or image to generate for. Must be one of format %s.`,
			buffetch.AllFormatsString,
		),
	)
	_ = flagSet.MarkDeprecated(
		inputFlagName,
		`input as the first argument instead.`+bufcli.FlagDeprecationMessageSuffix,
	)
	_ = flagSet.MarkHidden(inputFlagName)
	// deprecated
	flagSet.StringVar(
		&f.InputConfig,
		inputConfigFlagName,
		"",
		`The config file or data to use.`,
	)
	_ = flagSet.MarkDeprecated(
		inputConfigFlagName,
		fmt.Sprintf("use --%s instead.%s", configFlagName, bufcli.FlagDeprecationMessageSuffix),
	)
	_ = flagSet.MarkHidden(inputConfigFlagName)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
	moduleResolverReaderProvider bufcli.ModuleResolverReaderProvider,
) (retErr error) {
	logger := container.Logger()
	input, err := bufcli.GetInputValue(container, flags.InputHashtag, flags.Input, inputFlagName, ".")
	if err != nil {
		return err
	}
	inputConfig, err := bufcli.GetFlagOrDeprecatedFlag(
		flags.Config,
		configFlagName,
		flags.InputConfig,
		inputConfigFlagName,
	)
	if err != nil {
		return err
	}
	ref, err := buffetch.NewRefParser(logger).GetRef(ctx, input)
	if err != nil {
		return err
	}
	moduleResolver, err := moduleResolverReaderProvider.GetModuleResolver(ctx, container)
	if err != nil {
		return err
	}
	moduleReader, err := moduleResolverReaderProvider.GetModuleReader(ctx, container)
	if err != nil {
		return err
	}
	genConfig, err := bufgen.ReadConfig(flags.Template)
	if err != nil {
		return err
	}
	imageConfig, fileAnnotations, err := bufcli.NewWireImageConfigReader(
		logger,
		bufconfig.NewProvider(logger),
		moduleResolver,
		moduleReader,
	).GetImageConfig(
		ctx,
		container,
		ref,
		inputConfig,
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
	return bufgen.NewGenerator(logger).Generate(
		ctx,
		container,
		genConfig,
		imageConfig.Image(),
		bufgen.GenerateWithBaseOutDirPath(flags.BaseOutDirPath),
	)
}
