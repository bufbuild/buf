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

package encode

import (
	"context"
	"fmt"
	"io"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/bufpkg/bufreflect"
	reflectv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/reflect/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
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
		Short: "Encode the given descriptor into binary with a source reference.",
		Long: `The first argument is the descriptor to encode.
If no argument is specified, defaults to stdin.`,
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
		"The source containing the definition of the descriptor. If this is a module reference, it will be encoded in the descriptor",
	)
	flagSet.StringVar(
		&f.Type,
		typeFlagName,
		"",
		"The fully-qualified type name encoded in the descriptor (e.g. acme.weather.v1.Units)",
	)
	flagSet.StringVarP(
		&f.Output,
		outputFlagName,
		outputFlagShortName,
		"",
		fmt.Sprintf(
			`The location to write the encoded result to. Must be one of format %s.`,
			`[bin]`, // TODO: Do we need to support other formats?
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
	if flags.Source == "" {
		return appcmd.NewInvalidArgumentErrorf("--source is required")
	}
	if flags.Type == "" {
		return appcmd.NewInvalidArgumentErrorf("--type name is required")
	}
	if err := bufreflect.ValidateTypeName(flags.Type); err != nil {
		return appcmd.NewInvalidArgumentError(err.Error())
	}
	registryProvider, err := bufcli.NewRegistryProvider(ctx, container)
	if err != nil {
		return err
	}
	// TODO: There's probably a better way to distinguish between stdin and positional parameters.
	// We're normally able to do all of this within bufconfig, supporting multiple input formats
	// for the --config flag and whatnot.
	//
	// All of the following use cases will eventually need to be supported.
	//
	//  $ buf encode descriptor.json -o descriptor.bin
	//  $ buf encode <$json_bytes> -o descriptor.bin
	//  $ echo <$json_bytes> | buf encode -o descriptor.bin
	//
	var jsonBytes []byte
	if container.NumArgs() > 0 {
		jsonBytes = []byte(container.Arg(0))
	} else {
		jsonBytes, err = io.ReadAll(container.Stdin())
		if err != nil {
			return err
		}
	}
	image, err := bufcli.NewImageForSource(ctx, container, registryProvider, flags.Source, flags.ErrorFormat)
	if err != nil {
		return err
	}
	message, err := bufreflect.NewMessage(ctx, image, flags.Type)
	if err != nil {
		return err
	}
	if err := protojson.Unmarshal(jsonBytes, message); err != nil {
		return err
	}
	moduleReference, err := bufmoduleref.ModuleReferenceForString(flags.Source)
	if err == nil {
		// We can ignore this error because the source isn't required to be a valid ModuleReference.
		//
		// If it is, it will be included in the serialized descriptor. Otherwise, the result will
		// omit DescriptorInfo.
		if !bufmoduleref.IsCommitModuleReference(moduleReference) {
			// TODO: We either need to support generic references (e.g. tracks and tags),
			// or we need to resolve the commit associated with this reference before
			// it's attached here. It should probably be the latter so that we always
			// have a resolved commit in the message.
			return appcmd.NewInvalidArgumentErrorf("--source as a module reference must be a commit")
		}
	}
	bytes, err := marshalMessage(message, moduleReference, flags.Type)
	if err != nil {
		return err
	}
	if _, err := container.Stdout().Write(bytes); err != nil {
		return err
	}
	return nil
}

// marshalMessage marshals the given message into bytes, and appends the
// bytes containing the DescriptorInfo associated with the ModuleReference,
// if any.
func marshalMessage(
	message proto.Message,
	moduleReference bufmoduleref.ModuleReference,
	typeName string,
) ([]byte, error) {
	bytes, err := proto.Marshal(message)
	if err != nil {
		return nil, err
	}
	if moduleReference == nil {
		return bytes, nil
	}
	descriptorInfoBytes, err := proto.Marshal(
		&reflectv1alpha1.Reflector{
			DescriptorInfo: &reflectv1alpha1.DescriptorInfo{
				ModuleInfo: &reflectv1alpha1.ModuleInfo{
					Name: &reflectv1alpha1.ModuleName{
						Remote:     moduleReference.Remote(),
						Owner:      moduleReference.Owner(),
						Repository: moduleReference.Repository(),
					},
					Commit: moduleReference.Reference(),
				},
				TypeName: typeName,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return append(bytes, descriptorInfoBytes...), nil
}
