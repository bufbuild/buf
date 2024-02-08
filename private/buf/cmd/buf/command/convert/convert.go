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
	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimageutil"
	"github.com/bufbuild/buf/private/gen/data/datawkt"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	errorFormatFlagName     = "error-format"
	typeFlagName            = "type"
	fromFlagName            = "from"
	outputFlagName          = "to"
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
	ErrorFormat     string
	Type            string
	From            string
	To              string
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
		outputFlagName,
		"-",
		fmt.Sprintf(
			`The output location of the conversion. Supported formats are %s`,
			buffetch.MessageFormatsString,
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
	fromMessageRef, err := buffetch.NewMessageRefParser(
		container.Logger(),
		buffetch.MessageRefParserWithDefaultMessageEncoding(
			buffetch.MessageEncodingBinpb,
		),
	).GetMessageRef(ctx, flags.From)
	if err != nil {
		return fmt.Errorf("--%s: %v", outputFlagName, err)
	}
	storageosProvider := bufcli.NewStorageosProvider(flags.DisableSymlinks)
	runner := command.NewRunner()
	message, err := bufcli.NewWireProtoEncodingReader(
		container.Logger(),
		storageosProvider,
		runner,
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
	toMessageRef, err := buffetch.NewMessageRefParser(
		container.Logger(),
		buffetch.MessageRefParserWithDefaultMessageEncoding(
			defaultToEncoding,
		),
	).GetMessageRef(ctx, flags.To)
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
		toMessageRef,
	)
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
