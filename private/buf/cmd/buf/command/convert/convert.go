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

package convert

import (
	"context"
	"errors"
	"fmt"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufconvert"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimageutil"
	"github.com/bufbuild/buf/private/gen/data/datawkt"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	errorFormatFlagName = "error-format"
	typeFlagName        = "type"
	fromFlagName        = "from"
	outputFlagName      = "to"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <input>",
		Short: "Convert a message from binary to JSON or vice versa",
		Long: `
Use an input proto to interpret a proto/json message and convert it to a different format.

The simplest form is:

$ buf convert <input> --type=<type> --from=<payload> --to=<output>

<input> is the same input as any other buf command. 
It can be a local .proto file, binary output of "buf build", bsr module or local buf module.
e.g.
$ buf convert example.proto --type=Foo.proto --from=payload.json --to=output.bin

# Other examples

# All of <input>, "--from" and "to" accept formatting options

$ buf convert example.proto#format=bin --type=buf.Foo --from=payload#format=json --to=out#format=json

# Both <input> and "--from" accept stdin redirecting

$ buf convert <(buf build -o -)#format=bin --type=foo.Bar --from=<(echo "{\"one\":\"55\"}")#format=json

# Redirect from stdin to --from

$ echo "{\"one\":\"55\"}" | buf convert buf.proto --type buf.Foo --from -#format=json

# Redirect from stdin to <input>

$ buf build -o - | buf convert -#format=bin --type buf.Foo --from=payload.json

# Use a module on the bsr

buf convert buf.build/<org>/<repo> --type buf.Foo --from=payload.json
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
	ErrorFormat string
	Type        string
	From        string
	To          string

	// special
	InputHashtag string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	bufcli.BindInputHashtag(flagSet, &f.InputHashtag)
	flagSet.StringVar(
		&f.ErrorFormat,
		errorFormatFlagName,
		"text",
		fmt.Sprintf(
			"The format for build errors printed to stderr. Must be one of %s.",
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
			`The location of the payload to be converted. Supported formats are %s.`,
			bufconvert.MessageEncodingFormatsString,
		),
	)
	flagSet.StringVar(
		&f.To,
		outputFlagName,
		"-",
		fmt.Sprintf(
			`The output location of the conversion. Supported formats are %s.`,
			bufconvert.MessageEncodingFormatsString,
		),
	)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) error {
	if err := bufcli.ValidateErrorFormatFlag(flags.ErrorFormat, errorFormatFlagName); err != nil {
		return err
	}
	input, err := bufcli.GetInputValue(container, flags.InputHashtag, ".")
	if err != nil {
		return err
	}
	image, inputErr := bufcli.NewImageForSource(
		ctx,
		container,
		input,
		flags.ErrorFormat,
		false, // disableSymlinks
		"",    // configOverride
		nil,   // externalDirOrFilePaths
		nil,   // externalExcludeDirOrFilePaths
		false, // externalDirOrFilePathsAllowNotExist
		false, // excludeSourceCodeInfo
	)
	var resolveWellKnownType bool
	// only resolve wkts if input was not set.
	if container.NumArgs() == 0 {
		if inputErr != nil {
			resolveWellKnownType = true
		}
		if image != nil {
			_, filterErr := bufimageutil.ImageFilteredByTypes(image, flags.Type)
			if errors.Is(filterErr, bufimageutil.ErrImageFilterTypeNotFound) {
				resolveWellKnownType = true
			}
		}
	}
	if resolveWellKnownType {
		if _, ok := datawkt.MessageFilePath(flags.Type); ok {
			var wktErr error
			image, wktErr = bufcli.WellKnownTypeImage(ctx, container.Logger(), flags.Type)
			if wktErr != nil {
				return wktErr
			}
		}
	}
	if inputErr != nil && image == nil {
		return inputErr
	}
	fromMessageRef, err := bufconvert.NewMessageEncodingRef(ctx, flags.From, bufconvert.MessageEncodingBin)
	if err != nil {
		return fmt.Errorf("--%s: %v", outputFlagName, err)
	}
	message, err := bufcli.NewWireProtoEncodingReader(
		container.Logger(),
	).GetMessage(
		ctx,
		container,
		image,
		flags.Type,
		fromMessageRef,
	)
	if err != nil {
		return err
	}
	defaultToEncoding, err := inverseEncoding(fromMessageRef.MessageEncoding())
	if err != nil {
		return err
	}
	outputMessageRef, err := bufconvert.NewMessageEncodingRef(ctx, flags.To, defaultToEncoding)
	if err != nil {
		return fmt.Errorf("--%s: %v", outputFlagName, err)
	}
	return bufcli.NewWireProtoEncodingWriter(
		container.Logger(),
	).PutMessage(
		ctx,
		container,
		image,
		message,
		outputMessageRef,
	)
}

// inverseEncoding returns the opposite encoding of the provided encoding,
// which will be the default output encoding for a given payload encoding.
func inverseEncoding(encoding bufconvert.MessageEncoding) (bufconvert.MessageEncoding, error) {
	switch encoding {
	case bufconvert.MessageEncodingBin:
		return bufconvert.MessageEncodingJSON, nil
	case bufconvert.MessageEncodingJSON:
		return bufconvert.MessageEncodingBin, nil
	default:
		return 0, fmt.Errorf("unknown message encoding %v", encoding)
	}
}
