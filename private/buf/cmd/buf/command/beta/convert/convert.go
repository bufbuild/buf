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
	"fmt"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufconvert"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	errorFormatFlagName = "error-format"
	typeFlagName        = "type"
	schemaFlagName      = "schema"
	schemaFlagShortName = "s"
	outputFlagName      = "output"
	outputFlagShortName = "o"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <payload>",
		Short: "Convert a binary or JSON serialized payloads using the schema supplied",
		Long: `The first argument is the serialized message payload. 
The --schema flag is either a .proto, image, or buf module and the --type flag specifies the type within the schema.`,
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
	Output      string
	Schema      string

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
		`The full type name of the serialized payload (like acme.weather.v1.Units) within the schema input.`,
	)
	flagSet.StringVarP(
		&f.Output,
		outputFlagName,
		outputFlagShortName,
		"-",
		fmt.Sprintf(
			`The location to write the converted result to. Must be one of format %s.`,
			bufconvert.MessageEncodingFormatsString,
		),
	)

	flagSet.StringVarP(
		&f.Schema,
		schemaFlagName,
		schemaFlagShortName,
		".",
		`Source, module or image to use as descriptor for converting message.
Must be one of format [bin,dir,git,json,mod,protofile,tar,zip].`,
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
	source, typeName, err := bufcli.ParseInputAndType(ctx, flags.Schema, flags.Type)
	if err != nil {
		return err
	}
	image, err := bufcli.NewImageForSource(
		ctx,
		container,
		source,
		flags.ErrorFormat,
		false, // disableSymlinks
		"",    // configOverride
		nil,   // externalDirOrFilePaths
		nil,   // externalExcludeDirOrFilePaths
		false, // externalDirOrFilePathsAllowNotExist
		false, // excludeSourceCodeInfo
	)
	if err != nil {
		return err
	}
	payload, err := bufcli.GetInputValue(container, flags.InputHashtag, "-")
	if err != nil {
		return err
	}
	payloadMessageRef, err := bufconvert.NewMessageEncodingRef(ctx, payload, bufconvert.MessageEncodingBin)
	if err != nil {
		return fmt.Errorf("--%s: %v", outputFlagName, err)
	}
	message, err := bufcli.NewWireProtoEncodingReader(
		container.Logger(),
	).GetMessage(
		ctx,
		container,
		image,
		typeName,
		payloadMessageRef,
	)
	if err != nil {
		return err
	}
	defaultOutputEncoding, err := inverseEncoding(payloadMessageRef.MessageEncoding())
	if err != nil {
		return err
	}
	outputMessageRef, err := bufconvert.NewMessageEncodingRef(ctx, flags.Output, defaultOutputEncoding)
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
