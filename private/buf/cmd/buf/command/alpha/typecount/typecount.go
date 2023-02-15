// Copyright 2020-2023 Buf Technologies, Inc.
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

package typecount

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"google.golang.org/protobuf/types/descriptorpb"
)

const (
	errorFormatFlagName     = "error-format"
	configFlagName          = "config"
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
		Short: "Count messages, enums, and methods",
		Long:  bufcli.GetInputLong(`the source or module to get the count for`),
		Args:  cobra.MaximumNArgs(1),
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
	Config          string
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
		&f.Config,
		configFlagName,
		"",
		`The file or data to use to use for configuration`,
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
	image, err := bufcli.NewImageForSource(
		ctx,
		container,
		input,
		flags.ErrorFormat,
		flags.DisableSymlinks,
		flags.Config,
		nil,
		nil,
		false,
		true,
	)
	if err != nil {
		return err
	}
	_, err = container.Stdout().Write([]byte(fmt.Sprintf("%d\n", countForImage(image))))
	return err
}

func countForImage(image bufimage.Image) int {
	count := 0
	for _, imageFile := range image.Files() {
		fileDescriptor := imageFile.FileDescriptor()
		for _, descriptorProto := range fileDescriptor.GetMessageType() {
			count += countForDescriptorProto(descriptorProto)
		}
		count += len(fileDescriptor.GetEnumType())
		for _, serviceDescriptorProto := range fileDescriptor.GetService() {
			count += len(serviceDescriptorProto.GetMethod())
		}
	}
	return count
}

func countForDescriptorProto(descriptorProto *descriptorpb.DescriptorProto) int {
	count := 1
	for _, nestedDescriptorProto := range descriptorProto.GetNestedType() {
		count += countForDescriptorProto(nestedDescriptorProto)
	}
	count += len(descriptorProto.GetEnumType())
	return count
}
