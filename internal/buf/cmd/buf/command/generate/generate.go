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
	inputFlagName               = "input"
	inputConfigFlagName         = "input-config"
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
		Short: "Generate stubs for protoc plugins using a template.",
		Long: `This command uses a template file of the shape:

version: v1beta1         # required
plugins:
  - name: go             # required
    out: gen/go          # required
    out: plugins=grpc    # optional
    path: custom-gen-go  # optional
  - name java
    out: gen/java

By default, buf generate will look for a file of this shape named
"buf.gen.yaml" in your current directory. This can be thought of as a template
for the set of plugins you want to invoke. Then, call with:

# uses buf.gen.yaml as template, current directory as input
$ buf generate

# same as the defaults
$ buf generate --input . --template buf.gen.yaml

# --template also takes YAML or JSON data as input, so it can be used without a file
$ buf generate --input . --template '{"version":"v1beta1","plugins":[{"name":"go","out":"gen/go"}]}'

# download the repository, compile it, and generate per the bar.yaml template
$ buf generate --input https://github.com/foo/bar.git --template bar.yaml

# generate to the bar/ directory, prepending bar/ to the out directives in the template
$ buf generate --input https://github.com/foo/bar.git --template bar.yaml -o bar

The paths in the template and the -o flag will be interpreted as relative to your
current directory, so you can place your template files anywhere.

Plugins are invoked in the order they are specified in the template, but each plugin
has a per-directory parallel invocation, with results from each invocation combined
before writing the result. This is equivalent behavior to "buf protoc --by_dir".
`,
		Args: cobra.NoArgs,
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
	Input          string
	InputConfig    string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
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
		&f.Input,
		inputFlagName,
		".",
		fmt.Sprintf(
			`The source or image to generate for. Must be one of format %s.`,
			buffetch.AllFormatsString,
		),
	)
	flagSet.StringVar(
		&f.InputConfig,
		inputConfigFlagName,
		"",
		`The source or image config file or data to use.`,
	)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
	moduleResolverReaderProvider bufcli.ModuleResolverReaderProvider,
) (retErr error) {
	logger := container.Logger()
	ref, err := buffetch.NewRefParser(logger).GetRef(ctx, flags.Input)
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
	genConfig, err := bufgen.ReadConfig(flags.Template)
	if err != nil {
		return err
	}
	env, fileAnnotations, err := bufcli.NewWireEnvReader(
		logger,
		bufconfig.NewProvider(logger),
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
	return bufgen.NewGenerator(logger).Generate(
		ctx,
		container,
		genConfig,
		env.Image(),
		bufgen.GenerateWithBaseOutDirPath(flags.BaseOutDirPath),
	)
}
