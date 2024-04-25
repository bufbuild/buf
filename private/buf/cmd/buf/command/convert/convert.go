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

package convert

import (
	"context"
	"errors"
	"fmt"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufconvert"
	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimageutil"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/gen/data/datawkt"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/buf/private/pkg/tracing"
	"github.com/spf13/pflag"
)

const (
	errorFormatFlagName     = "error-format"
	typeFlagName            = "type"
	fromFlagName            = "from"
	toFlagName              = "to"
	validateFlagName        = "validate"
	disableSymlinksFlagName = "disable-symlinks"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <input>",
		Short: "Convert a message between binary, text, or JSON",
		Long: `
Use an input proto to interpret a proto/json message and convert it to a different format.

Examples:

    $ buf convert <input> --type=<type> --from=<payload> --to=<output>

The <input> can be a local .proto file, binary output of "buf build", bsr module or local buf module:

    $ buf convert example.proto --type=Foo.proto --from=payload.json --to=output.binpb

All of <input>, "--from" and "to" accept formatting options:

    $ buf convert example.proto#format=binpb --type=buf.Foo --from=payload#format=json --to=out#format=json

Both <input> and "--from" accept stdin redirecting:

    $ buf convert <(buf build -o -)#format=binpb --type=foo.Bar --from=<(echo "{\"one\":\"55\"}")#format=json

Redirect from stdin to --from:

    $ echo "{\"one\":\"55\"}" | buf convert buf.proto --type buf.Foo --from -#format=json

Redirect from stdin to <input>:

    $ buf build -o - | buf convert -#format=binpb --type buf.Foo --from=payload.json

Use a module on the bsr:

    $ buf convert <buf.build/owner/repository> --type buf.Foo --from=payload.json
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
	ErrorFormat     string
	Type            string
	From            string
	To              string
	Validate        bool
	DisableSymlinks bool

	// special
	InputHashtag string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	bufcli.BindInputHashtag(flagSet, &f.InputHashtag)
	bufcli.BindDisableSymlinks(flagSet, &f.DisableSymlinks, disableSymlinksFlagName)
	flagSet.StringVar(
		&f.ErrorFormat,
		errorFormatFlagName,
		"text",
		fmt.Sprintf(
			"The format for build errors printed to stderr. Must be one of %s",
			stringutil.SliceToString(bufanalysis.AllFormatStrings),
		),
	)
	flagSet.StringVar(
		&f.Type,
		typeFlagName,
		"",
		`The full type name of the message within the input (e.g. acme.weather.v1.Units)`,
	)
	flagSet.StringVar(
		&f.From,
		fromFlagName,
		"-",
		fmt.Sprintf(
			`The location of the payload to be converted. Supported formats are %s`,
			buffetch.MessageFormatsString,
		),
	)
	flagSet.StringVar(
		&f.To,
		toFlagName,
		"-",
		fmt.Sprintf(
			`The output location of the conversion. Supported formats are %s`,
			buffetch.MessageFormatsString,
		),
	)
	flagSet.BoolVar(
		&f.Validate,
		validateFlagName,
		false,
		fmt.Sprintf(
			`Validate the message specified with --%s by applying protovalidate rules to it. See https://github.com/bufbuild/protovalidate for more details.`,
			fromFlagName,
		),
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
		bufctl.WithFileAnnotationErrorFormat(flags.ErrorFormat),
	)
	if err != nil {
		return err
	}
	schemaImage, schemaImageErr := controller.GetImage(
		ctx,
		input,
	)
	var resolveWellKnownType bool
	// only resolve wkts if input was not set.
	if container.NumArgs() == 0 {
		if schemaImageErr != nil {
			resolveWellKnownType = true
		}
		if schemaImage != nil {
			_, filterErr := bufimageutil.ImageFilteredByTypes(schemaImage, flags.Type)
			if errors.Is(filterErr, bufimageutil.ErrImageFilterTypeNotFound) {
				resolveWellKnownType = true
			}
		}
	}
	if resolveWellKnownType {
		if _, ok := datawkt.MessageFilePath(flags.Type); ok {
			var wktErr error
			schemaImage, wktErr = wellKnownTypeImage(
				ctx,
				tracing.NewTracer(container.Tracer()),
				flags.Type,
			)
			if wktErr != nil {
				return wktErr
			}
		}
	}
	if schemaImageErr != nil && schemaImage == nil {
		return schemaImageErr
	}
	// We can't correctly convert anything that uses message-set wire
	// format. So we prevent that by having the resolver return an error
	// if asked to resolve any type that uses it.
	schemaImage = bufconvert.ImageWithoutMessageSetWireFormatResolution(schemaImage)
	var fromFunctionOptions []bufctl.FunctionOption
	if flags.Validate {
		fromFunctionOptions = append(fromFunctionOptions, bufctl.WithMessageValidation())
	}
	fromMessage, fromMessageEncoding, err := controller.GetMessage(
		ctx,
		schemaImage,
		flags.From,
		flags.Type,
		buffetch.MessageEncodingBinpb,
		fromFunctionOptions...,
	)
	if err != nil {
		return fmt.Errorf("--%s: %w", fromFlagName, err)
	}
	defaultToMessageEncoding, err := inverseEncoding(fromMessageEncoding)
	if err != nil {
		return err
	}
	if err := controller.PutMessage(
		ctx,
		schemaImage,
		flags.To,
		fromMessage,
		defaultToMessageEncoding,
	); err != nil {
		return fmt.Errorf("--%s: %w", toFlagName, err)
	}
	return nil
}

// inverseEncoding returns the opposite encoding of the provided encoding,
// which will be the default output encoding for a given payload encoding.
func inverseEncoding(encoding buffetch.MessageEncoding) (buffetch.MessageEncoding, error) {
	switch encoding {
	case buffetch.MessageEncodingBinpb:
		return buffetch.MessageEncodingJSON, nil
	case buffetch.MessageEncodingJSON:
		return buffetch.MessageEncodingBinpb, nil
	case buffetch.MessageEncodingTxtpb:
		return buffetch.MessageEncodingBinpb, nil
	case buffetch.MessageEncodingYAML:
		return buffetch.MessageEncodingBinpb, nil
	default:
		return 0, fmt.Errorf("unknown message encoding %v", encoding)
	}
}

// wellKnownTypeImage returns an Image with just the given WKT type name (google.protobuf.Duration for example).
func wellKnownTypeImage(
	ctx context.Context,
	tracer tracing.Tracer,
	wellKnownTypeName string,
) (bufimage.Image, error) {
	moduleSetBuilder := bufmodule.NewModuleSetBuilder(ctx, tracer, bufmodule.NopModuleDataProvider, bufmodule.NopCommitProvider)
	moduleSetBuilder.AddLocalModule(
		datawkt.ReadBucket,
		".",
		true,
	)
	moduleSet, err := moduleSetBuilder.Build()
	if err != nil {
		return nil, err
	}
	image, err := bufimage.BuildImage(
		ctx,
		tracer,
		bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles(moduleSet),
	)
	if err != nil {
		return nil, err
	}
	return bufimageutil.ImageFilteredByTypes(image, wellKnownTypeName)
}
