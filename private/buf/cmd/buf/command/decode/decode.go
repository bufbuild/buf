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
	sourceFlagName      = "source"
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
		Use:   name + " <descriptor>",
		Short: "Decode binary descriptors with a source reference.",
		Long: `The first argument is the serialized descriptor to decode.
If no argument is specified, defaults to stdin.`,
		Args: cobra.ExactArgs(0),
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
	Source      string
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
		&f.Source,
		sourceFlagName,
		"",
		"The source that defines the serialized descriptor (e.g. buf.build/acme/weather)",
	)
	flagSet.StringVar(
		&f.Type,
		typeFlagName,
		"",
		"The fully-qualified type name of the serialized descriptor (e.g. acme.weather.v1.Units)",
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
	descriptorBytes, err := io.ReadAll(container.Stdin())
	if err != nil {
		return err
	}
	if len(descriptorBytes) == 0 {
		return fmt.Errorf("stdin is required as the input")
	}
	protoSource, protoType, err := parseSourceAndType(ctx, flags.Source, flags.Type)
	if err != nil {
		return err
	}
	registryProvider, err := bufcli.NewRegistryProvider(ctx, container)
	if err != nil {
		return err
	}
	image, err := bufcli.NewImageForSource(ctx, container, registryProvider, protoSource, flags.ErrorFormat)
	if err != nil {
		return err
	}
	message, err := bufreflect.NewMessage(ctx, image, protoType)
	if err != nil {
		return err
	}
	if err := protoencoding.NewWireUnmarshaler(nil).Unmarshal(descriptorBytes, message); err != nil {
		return err
	}
	return bufcli.NewWireProtoEncodingWriter(
		container.Logger(),
	).PutMessage(
		ctx,
		container,
		message,
		flags.Output,
	)
}

// parseSourceAndType returns the moduleReference and typeName from the source and type provided by the user.
// When source is not provided, we assume the type is a fully-qualified path to the type and try to parse it.
// When source and type are redundantly specified (e.g. buf.build/acme/weather and buf.build/acme/weather#weather.v1.Units),
// we are not handling it and just let it fail.
func parseSourceAndType(
	ctx context.Context,
	flagSource string,
	flagType string,
) (moduleReference string, typeName string, _ error) {
	if flagSource != "" && flagType != "" {
		return flagSource, flagType, nil
	}
	if flagType == "" {
		return "", "", appcmd.NewInvalidArgumentError("type is required")
	}
	moduleReference, typeName, err := bufreflect.ParseFullyQualifiedPath(flagType)
	if err != nil {
		return "", "", appcmd.NewInvalidArgumentErrorf("if source is not provided, the type need to be a fully-qualified path that includes the module reference, failed to parse the type: %v", err)
	}
	return moduleReference, typeName, nil
}
