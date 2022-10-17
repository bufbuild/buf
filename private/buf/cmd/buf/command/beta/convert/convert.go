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

$ buf beta convert <input> --type=<type> --from=<payload> --to=<output>

<input> is the same input as any other buf command. 
It can be a local .proto file, binary output of "buf build", bsr module or local buf module.
e.g.
$ buf beta convert example.proto --type=Foo.proto --from=payload.json --to=output.bin

# Other examples

# All of <input>, "--from" and "to" accept formatting options

$ buf beta convert example.proto#format=bin --type=buf.Foo --from=payload#format=json --to=out#format=json

# Both <input> and "--from" accept stdin redirecting

$ buf beta convert <(buf build -o -)#format=bin --type=foo.Bar --from=<(echo "{\"one\":\"55\"}")#format=json

# Redirect from stdin to --from

$ echo "{\"one\":\"55\"}" | buf beta convert buf.proto --type buf.Foo --from -#format=json

# Redirect from stdin to <input>

$ buf build -o - | buf beta convert -#format=bin --type buf.Foo --from=payload.json

# Use a module on the bsr

buf beta convert buf.build/<org>/<repo> --type buf.Foo --from=payload.json
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
	input, err := bufcli.GetInputValue(container, flags.InputHashtag, "")
	if err != nil {
		return err
	}
	if wkpath := wktToPath(flags.Type); input == "" && wkpath != "" {
		input = wkpath
	}
	image, err := bufcli.NewImageForSource(
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
	if err != nil {
		return err
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

// wktToPath returns the import path of the proto file for a well known type.
func wktToPath(fulltype string) string {
	switch fulltype {
	case "google.protobuf.Timestamp":
		return "google/protobuf/timestamp.proto"
	case "google.protobuf.FieldMask":
		return "google/protobuf/field_mask.proto"
	case "google.protobuf.Api":
		return "google/protobuf/api.proto"
	case "google.protobuf.Method":
		return "google/protobuf/api.proto"
	case "google.protobuf.Mixin":
		return "google/protobuf/api.proto"
	case "google.protobuf.Duration":
		return "google/protobuf/duration.proto"
	case "google.protobuf.Struct":
		return "google/protobuf/struct.proto"
	case "google.protobuf.Value":
		return "google/protobuf/struct.proto"
	case "google.protobuf.ListValue":
		return "google/protobuf/struct.proto"
	case "google.protobuf.DoubleValue":
		return "google/protobuf/wrappers.proto"
	case "google.protobuf.FloatValue":
		return "google/protobuf/wrappers.proto"
	case "google.protobuf.Int64Value":
		return "google/protobuf/wrappers.proto"
	case "google.protobuf.UInt64Value":
		return "google/protobuf/wrappers.proto"
	case "google.protobuf.Int32Value":
		return "google/protobuf/wrappers.proto"
	case "google.protobuf.UInt32Value":
		return "google/protobuf/wrappers.proto"
	case "google.protobuf.BoolValue":
		return "google/protobuf/wrappers.proto"
	case "google.protobuf.StringValue":
		return "google/protobuf/wrappers.proto"
	case "google.protobuf.BytesValue":
		return "google/protobuf/wrappers.proto"
	case "google.protobuf.SourceContext":
		return "google/protobuf/source_context.proto"
	case "google.protobuf.Any":
		return "google/protobuf/any.proto"
	case "google.protobuf.Type":
		return "google/protobuf/type.proto"
	case "google.protobuf.Field":
		return "google/protobuf/type.proto"
	case "google.protobuf.Enum":
		return "google/protobuf/type.proto"
	case "google.protobuf.EnumValue":
		return "google/protobuf/type.proto"
	case "google.protobuf.Option":
		return "google/protobuf/type.proto"
	case "google.protobuf.Empty ":
		return "google/protobuf/empty.proto"
	case "google.protobuf.Version":
		return "google/protobuf/compiler/plugin.proto"
	case "google.protobuf.CodeGeneratorRequest":
		return "google/protobuf/compiler/plugin.proto"
	case "google.protobuf.CodeGeneratorResponse":
		return "google/protobuf/compiler/plugin.proto"
	case "google.protobuf.FileDescriptorSet":
		return "google/protobuf/descriptor.proto"
	case "google.protobuf.FileDescriptorProto":
		return "google/protobuf/descriptor.proto"
	case "google.protobuf.DescriptorProto":
		return "google/protobuf/descriptor.proto"
	case "google.protobuf.ExtensionRangeOptions":
		return "google/protobuf/descriptor.proto"
	case "google.protobuf.FieldDescriptorProto":
		return "google/protobuf/descriptor.proto"
	case "google.protobuf.OneofDescriptorProto":
		return "google/protobuf/descriptor.proto"
	case "google.protobuf.EnumDescriptorProto":
		return "google/protobuf/descriptor.proto"
	case "google.protobuf.EnumValueDescriptorProto":
		return "google/protobuf/descriptor.proto"
	case "google.protobuf.ServiceDescriptorProto":
		return "google/protobuf/descriptor.proto"
	case "google.protobuf.MethodDescriptorProto":
		return "google/protobuf/descriptor.proto"
	case "google.protobuf.FileOptions":
		return "google/protobuf/descriptor.proto"
	case "google.protobuf.MessageOptions":
		return "google/protobuf/descriptor.proto"
	case "google.protobuf.FieldOptions":
		return "google/protobuf/descriptor.proto"
	case "google.protobuf.OneofOptions":
		return "google/protobuf/descriptor.proto"
	case "google.protobuf.EnumOptions":
		return "google/protobuf/descriptor.proto"
	case "google.protobuf.EnumValueOptions":
		return "google/protobuf/descriptor.proto"
	case "google.protobuf.ServiceOptions":
		return "google/protobuf/descriptor.proto"
	case "google.protobuf.MethodOptions":
		return "google/protobuf/descriptor.proto"
	case "google.protobuf.UninterpretedOption":
		return "google/protobuf/descriptor.proto"
	case "google.protobuf.SourceCodeInfo":
		return "google/protobuf/descriptor.proto"
	case "google.protobuf.GeneratedCodeInfo":
		return "google/protobuf/descriptor.proto"
	}
	return ""
}
