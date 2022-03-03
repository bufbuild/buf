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
	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/bufpkg/bufrpc"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	errorFormatFlagName = "error-format"
	typeFlagName        = "type"
	inputFlagName       = "input"
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
		Use:   name + " <source>",
		Short: "Use a source reference to convert a binary or JSON serialized message supplied through stdin or the input flag.",
		Long: `The first argument is the source that defines the serialized message (like buf.build/acme/weather).
Alternatively, you can omit the source and specify a fully qualified path for the type using the --type option (like buf.build/acme/weather#acme.weather.v1.Units).`,
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
	Input       string
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
			"The format for build errors printed to stderr. Must be one of %s.",
			stringutil.SliceToString(bufanalysis.AllFormatStrings),
		),
	)
	flagSet.StringVar(
		&f.Type,
		typeFlagName,
		"",
		`The full type name of the serialized message (like acme.weather.v1.Units).
Alternatively, this can be a fully qualified path to the type without providing the source (like buf.build/acme/weather#acme.weather.v1.Units).`,
	)
	flagSet.StringVar(
		&f.Input,
		inputFlagName,
		"-",
		fmt.Sprintf(
			`The location to read the input message. Must be one of format %s.`,
			bufconvert.MessageEncodingFormatsString,
		),
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
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) error {
	if err := bufcli.ValidateErrorFormatFlag(flags.ErrorFormat, errorFormatFlagName); err != nil {
		return err
	}
	input, err := bufcli.GetInputValue(container, "", "")
	if err != nil {
		return err
	}
	source, typeName, err := bufcli.ParseSourceAndType(ctx, input, flags.Type)
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
	inputMessageRef, err := bufconvert.NewMessageEncodingRef(ctx, flags.Input, bufconvert.MessageEncodingBin)
	if err != nil {
		return fmt.Errorf("--%s: %v", outputFlagName, err)
	}
	messagePayload, err := bufcli.NewWireProtoEncodingReader(
		container.Logger(),
	).GetMessagePayload(
		ctx,
		container,
		inputMessageRef,
	)
	if err != nil {
		return err
	}
	defaultOutputEncoding, err := inverseEncoding(inputMessageRef.MessageEncoding())
	if err != nil {
		return err
	}
	outputMessageRef, err := bufconvert.NewMessageEncodingRef(ctx, flags.Output, defaultOutputEncoding)
	if err != nil {
		return fmt.Errorf("--%s: %v", outputFlagName, err)
	}
	remote, err := remoteForSource(ctx, container, source)
	if err != nil {
		return err
	}
	registryProvider, err := bufcli.NewRegistryProvider(ctx, container)
	if err != nil {
		return err
	}
	convertService, err := registryProvider.NewConvertService(ctx, remote)
	if err != nil {
		return err
	}
	requestFormat, err := messageEncodingToConvertFormat(inputMessageRef.MessageEncoding())
	if err != nil {
		return err
	}
	responseFormat, err := messageEncodingToConvertFormat(outputMessageRef.MessageEncoding())
	if err != nil {
		return err
	}
	payload, err := convertService.Convert(
		ctx,
		typeName,
		bufimage.ImageToProtoImage(image),
		messagePayload,
		requestFormat,
		responseFormat,
	)
	if err != nil {
		return err
	}
	return bufcli.NewWireProtoEncodingWriter(
		container.Logger(),
	).PutMessagePayload(
		ctx,
		container,
		payload,
		outputMessageRef,
	)
}

// inverseEncoding returns the opposite encoding of the provided encoding,
// which will be the default output encoding for a given input encoding.
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

// messageEncodingToConvertFormat parses a MessageEncoding to a ConvertFormat.
func messageEncodingToConvertFormat(encoding bufconvert.MessageEncoding) (registryv1alpha1.ConvertFormat, error) {
	switch encoding {
	case bufconvert.MessageEncodingBin:
		return registryv1alpha1.ConvertFormat_CONVERT_FORMAT_BIN, nil
	case bufconvert.MessageEncodingJSON:
		return registryv1alpha1.ConvertFormat_CONVERT_FORMAT_JSON, nil
	default:
		return 0, fmt.Errorf("unknown message encoding %v", encoding)
	}
}

// remoteForSource returns the module remote if the source is a module ref,
// and returns the default remote if it is other type of ref.
func remoteForSource(
	ctx context.Context,
	container appflag.Container,
	source string,
) (string, error) {
	ref, err := buffetch.NewRefParser(container.Logger(), buffetch.RefParserWithProtoFileRefAllowed()).GetRef(ctx, source)
	if err != nil {
		return "", err
	}
	switch ref.(type) {
	case buffetch.ModuleRef:
		moduleReference, err := bufmoduleref.ModuleReferenceForString(source)
		if err != nil {
			return "", appcmd.NewInvalidArgumentError(err.Error())
		}
		return moduleReference.Remote(), nil
	default:
		return bufrpc.DefaultRemote, nil
	}
}
