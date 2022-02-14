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

package decode

import (
	"context"
	"fmt"
	"io"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufreflect"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	errorFormatFlagName = "error-format"
	typeFlagName        = "type"
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
		Use:   name,
		Short: "Decode binary serialized message with a source reference.",
		Long: `The first argument is the source that defines the serialized message (e.g. buf.build/acme/weather).
If no argument is specified, the type provided need to be a fully-qualified path to the type (e.g. buf.build/acme/weather#acme.weather.v1.Units).`,
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
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(
		&f.ErrorFormat,
		errorFormatFlagName,
		"text",
		fmt.Sprintf(
			"The format for build errors, printed to stderr. Must be one of %s.",
			stringutil.SliceToString(bufanalysis.AllFormatStrings),
		),
	)
	flagSet.StringVar(
		&f.Type,
		typeFlagName,
		"",
		`The fully-qualified type name of the serialized message (e.g. acme.weather.v1.Units)
Alternatively, this can be a fully-qualified path to the type (e.g. buf.build/acme/weather#acme.weather.v1.Units) without providing the source`,
	)
	flagSet.StringVarP(
		&f.Output,
		outputFlagName,
		outputFlagShortName,
		"-",
		// TODO: If we ever support other formats (e.g. prototext), we will need
		// to build a buffetch.ProtoEncodingRefParser.
		`The location to write the decoded result to. This is always JSON for now.`,
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
	messageBytes, err := io.ReadAll(container.Stdin())
	if err != nil {
		return err
	}
	if len(messageBytes) == 0 {
		return fmt.Errorf("stdin is required as the input")
	}
	input, err := bufcli.GetInputValue(container, "", "")
	if err != nil {
		return err
	}
	protoSource, protoType, err := bufcli.ParseSourceAndType(ctx, input, flags.Type)
	if err != nil {
		return err
	}
	image, err := bufcli.NewImageForSource(
		ctx,
		container,
		protoSource,
		flags.ErrorFormat,
		false,
		"",
		nil,
		nil,
		false,
		false,
	)
	if err != nil {
		return err
	}
	message, err := bufreflect.NewMessage(ctx, image, protoType)
	if err != nil {
		return err
	}
	if err := protoencoding.NewWireUnmarshaler(nil).Unmarshal(messageBytes, message); err != nil {
		return err
	}
	return bufcli.NewWireProtoEncodingWriter(
		container.Logger(),
	).PutMessage(
		ctx,
		container,
		image,
		message,
		flags.Output,
	)
}
