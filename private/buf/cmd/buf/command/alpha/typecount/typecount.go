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
	"encoding/json"
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
	stats := newStats()
	countForImage(stats, image)
	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return err
	}
	_, err = container.Stdout().Write(append(data, []byte("\n")...))
	return err
}

func countForImage(stats *stats, image bufimage.Image) {
	for _, imageFile := range image.Files() {
		fileDescriptor := imageFile.FileDescriptor()
		for _, descriptorProto := range fileDescriptor.GetMessageType() {
			countForDescriptorProto(stats, descriptorProto)
		}
		stats.NumEnums += len(fileDescriptor.GetEnumType())
		for _, serviceDescriptorProto := range fileDescriptor.GetService() {
			stats.NumMethods += len(serviceDescriptorProto.GetMethod())
		}
	}
}

func countForDescriptorProto(stats *stats, descriptorProto *descriptorpb.DescriptorProto) {
	stats.NumMessages++
	for _, nestedDescriptorProto := range descriptorProto.GetNestedType() {
		countForDescriptorProto(stats, nestedDescriptorProto)
	}
	stats.NumEnums += len(descriptorProto.GetEnumType())
}

type stats struct {
	NumMessages int `json:"num_messages,omitempty" yaml:"num_messages,omitempty"`
	NumEnums    int `json:"num_enums,omitempty" yaml:"num_enums,omitempty"`
	NumMethods  int `json:"num_methods,omitempty" yaml:"num_methods,omitempty"`
}

func newStats() *stats {
	return &stats{}
}
