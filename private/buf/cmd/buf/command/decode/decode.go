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
	"errors"
	"fmt"
	"io"
	"path"
	"strings"

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
	"google.golang.org/protobuf/types/known/anypb"
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
		"",
		fmt.Sprintf(
			`The location to write the decoded result to. Must be one of format %s.`,
			`[json]`, // TODO: We need to support other formats.
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
	registryProvider, err := bufcli.NewRegistryProvider(ctx, container)
	if err != nil {
		return err
	}
	// TODO: There's probably a better way to distinguish between stdin and positional parameters.
	// We're normally able to do all of this within buffetch, supporting multiple input formats
	// and whatnot.
	//
	// All of the following use cases will eventually need to be supported.
	//
	//  $ buf decode descriptor.bin | jq
	//  $ buf decode <$bytes> | jq
	//  $ echo <$bytes> | buf decode | jq
	//
	var descriptorBytes []byte
	if container.NumArgs() > 0 {
		descriptorBytes = []byte(container.Arg(0))
	} else {
		descriptorBytes, err = io.ReadAll(container.Stdin())
		if err != nil {
			return err
		}
	}
	typeInfo, err := getTypeInfo(
		ctx,
		descriptorBytes,
		flags.Source,
		flags.Type,
	)
	if err != nil {
		return err
	}
	image, err := bufcli.NewImageForSource(ctx, container, registryProvider, typeInfo.Source, flags.ErrorFormat)
	if err != nil {
		return err
	}
	message, err := bufreflect.NewMessage(ctx, image, typeInfo.TypeName)
	if err != nil {
		return err
	}
	if err := proto.Unmarshal(typeInfo.DescriptorBytes, message); err != nil {
		return err
	}
	marshaler := protojson.MarshalOptions{
		Indent:       "  ",
		AllowPartial: true,
	}
	jsonBytes, err := marshaler.Marshal(message)
	if err != nil {
		return err
	}
	if _, err := container.Stdout().Write(jsonBytes); err != nil {
		return err
	}
	return nil
}

// getTypeInfo resolves the source and type name from the given descriptor or user-provided
// name, if any.
//
// The user-provided values always take precedence over the encoded values. The only exception
// is if the descriptorBytes represent an proto.Any - in that case the proto.Any's value
// overrides the user-provided descriptorBytes (as expected).
func getTypeInfo(
	ctx context.Context,
	descriptorBytes []byte,
	source string,
	typeName string,
) (*typeInfo, error) {
	typeInfo, err := getTypeInfoForDescriptor(descriptorBytes)
	if err != nil {
		return nil, err
	}
	if typeInfo != nil {
		if source == "" {
			// The user-provided source takes precedence over the descriptor.
			source = typeInfo.Source
		}
		if typeName == "" {
			// The user-provided typeName takes precedence over the descriptor.
			typeName = typeInfo.TypeName
		}
		if len(typeInfo.DescriptorBytes) > 0 {
			// The descriptor bytes attached to the typeInfo are the only thing that take
			// precedence over the user-provided type. This is particularly relevant for
			// the encoded proto.Any.
			descriptorBytes = typeInfo.DescriptorBytes
		}
	}
	if source == "" {
		return nil, fmt.Errorf("a source could not be resolved; specify one with --%s", sourceFlagName)
	}
	if typeName == "" {
		return nil, fmt.Errorf("a type name could not be resolved; specify one with --%s", typeFlagName)
	}
	if err := bufreflect.ValidateTypeName(typeName); err != nil {
		return nil, err
	}
	return newTypeInfo(
		source,
		typeName,
		descriptorBytes,
	), nil
}

// getTypeInfoForDescriptor returns the module reference and type name associated with the
// DescriptorInfo embedded in the descriptor, if any. Otherwise, this function checks if
// the given descriptor is an proto.Any and attempts to extract a valid module reference from
// its TypeURL. If neither of these attempts are successful, a nil typeInfo is returned, which
// can still be used if the user provided a valid image source.
func getTypeInfoForDescriptor(
	descriptorBytes []byte,
) (*typeInfo, error) {
	// First check if the descriptor embeds the DescriptorInfo itself.
	reflector := new(reflectv1alpha1.Reflector)
	if err := proto.Unmarshal(descriptorBytes, reflector); err == nil {
		if reflector.GetDescriptorInfo() != nil {
			return typeInfoForDescriptorInfo(
				reflector.GetDescriptorInfo(),
				descriptorBytes,
			)
		}
	}
	// This message doesn't include DescriptorInfo, so we'll see if it's a proto.Any.
	any := new(anypb.Any)
	if err := proto.Unmarshal(descriptorBytes, any); err == nil {
		if any.GetTypeUrl() != "" && any.GetValue() != nil {
			return typeInfoForAny(any)
		}
	}
	return nil, nil
}

// typeInfoForDescriptorInfo parses the DescriptorInfo into a module reference and type name, validating that
// they're well-formed.
func typeInfoForDescriptorInfo(
	descriptorInfo *reflectv1alpha1.DescriptorInfo,
	descriptorBytes []byte,
) (*typeInfo, error) {
	moduleReference, err := moduleReferenceForModuleInfo(descriptorInfo.GetModuleInfo())
	if err != nil {
		return nil, err
	}
	typeName := descriptorInfo.GetTypeName()
	if err := bufreflect.ValidateTypeName(typeName); err != nil {
		return nil, err
	}
	return newTypeInfo(
		moduleReference.String(),
		typeName,
		descriptorBytes,
	), nil
}

// typeInfoForAny parses the proto.Any into a module reference and type name, validating that
// they're well-formed.
func typeInfoForAny(
	any *anypb.Any,
) (*typeInfo, error) {
	moduleReference, typeName, err := parseFullyQualifiedTypeName(any.GetTypeUrl())
	if err != nil {
		return nil, err
	}
	return newTypeInfo(
		moduleReference.String(),
		typeName,
		any.GetValue(),
	), nil
}

// parseFullyQualifiedTypeName parses the given fully qualified type name (e.g. buf.build/acme/weather/acme.weather.v1.Units)
// into a module reference and type name.
func parseFullyQualifiedTypeName(
	fullyQualifiedTypeName string,
) (bufmoduleref.ModuleReference, string, error) {
	split := strings.Split(fullyQualifiedTypeName, "/")
	if len(split) != 4 {
		return nil, "", appcmd.NewInvalidArgumentErrorf(
			"fully qualified type name %q is invalid: must be in the form remote/owner/repository/type",
			fullyQualifiedTypeName,
		)
	}
	moduleReference, err := bufmoduleref.ModuleReferenceForString(path.Join(split[:len(split)-1]...))
	if err != nil {
		return nil, "", appcmd.NewInvalidArgumentError(err.Error())
	}
	typeName := split[3]
	if err := bufreflect.ValidateTypeName(typeName); err != nil {
		return nil, "", appcmd.NewInvalidArgumentError(err.Error())
	}
	return moduleReference, typeName, nil
}

// moduleReferenceForModuleInfo maps the given moduleInfo into a module reference.
func moduleReferenceForModuleInfo(
	moduleInfo *reflectv1alpha1.ModuleInfo,
) (bufmoduleref.ModuleReference, error) {
	if moduleInfo == nil {
		return nil, errors.New("found a nil ModuleInfo in the descriptor's DescriptorInfo")
	}
	moduleName := moduleInfo.GetName()
	if moduleName == nil {
		return nil, errors.New("found a nil ModuleName in the descriptor's ModuleInfo")
	}
	return bufmoduleref.NewModuleReference(
		moduleName.GetRemote(),
		moduleName.GetOwner(),
		moduleName.GetRepository(),
		moduleInfo.GetCommit(),
	)
}

// typeInfo holds all of the information related to the encoded descriptor.
type typeInfo struct {
	Source          string
	TypeName        string
	DescriptorBytes []byte
}

// newTypeInfo constructs a new typeInfo.
func newTypeInfo(
	source string,
	typeName string,
	descriptorBytes []byte,
) *typeInfo {
	return &typeInfo{
		Source:          source,
		TypeName:        typeName,
		DescriptorBytes: descriptorBytes,
	}
}
