// Copyright 2020 Buf Technologies, Inc.
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

	"github.com/bufbuild/buf/internal/buf/buffetch"
	imageinternal "github.com/bufbuild/buf/internal/buf/cmd/buf/command/image/internal"
	"github.com/bufbuild/buf/internal/buf/cmd/internal"
	"github.com/bufbuild/buf/internal/pkg/app/appcmd"
	"github.com/bufbuild/buf/internal/pkg/app/appflag"
	"github.com/bufbuild/buf/internal/pkg/app/applog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	asFileDescriptorSetFlagName = "as-file-descriptor-set"
	excludeImportsFlagName      = "exclude-imports"
	excludeSourceInfoFlagName   = "exclude-source-info"
	filesFlagName               = "file"
	outputFlagName              = "output"
	outputFlagShortName         = "o"
	imageFlagName               = "image"
	imageFlagShortName          = "i"
)

// NewCommand returns a new Command
func NewCommand(name string, builder appflag.Builder) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name,
		Short: "Convert the input Image to an output Image with the specified format and filters.",
		Args:  cobra.NoArgs,
		Run: builder.NewRunFunc(
			func(ctx context.Context, container applog.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	AsFileDescriptorSet bool
	ExcludeImports      bool
	ExcludeSourceInfo   bool
	Files               []string
	Image               string
	Output              string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	imageinternal.BindAsFileDescriptorSet(flagSet, &f.AsFileDescriptorSet, asFileDescriptorSetFlagName)
	imageinternal.BindExcludeImports(flagSet, &f.ExcludeImports, excludeImportsFlagName)
	imageinternal.BindExcludeSourceInfo(flagSet, &f.ExcludeSourceInfo, excludeSourceInfoFlagName)
	imageinternal.BindFiles(flagSet, &f.Files, filesFlagName)
	flagSet.StringVarP(
		&f.Image,
		imageFlagName,
		imageFlagShortName,
		"",
		fmt.Sprintf(
			`The image to convert. Must be one of format %s.`,
			buffetch.ImageFormatsString,
		),
	)
	flagSet.StringVarP(
		&f.Output,
		outputFlagName,
		outputFlagShortName,
		"",
		fmt.Sprintf(
			`Required. The location to write the image to. Must be one of format %s.`,
			buffetch.ImageFormatsString,
		),
	)
}

func run(ctx context.Context, container applog.Container, flags *flags) (retErr error) {
	internal.WarnExperimental(container)
	if flags.Output == "" {
		return fmt.Errorf("--%s is required", outputFlagName)
	}
	imageRef, err := buffetch.NewImageRefParser(container.Logger()).GetImageRef(ctx, flags.Image)
	if err != nil {
		return fmt.Errorf("--%s: %v", imageFlagName, err)
	}
	image, err := internal.NewBufwireImageReader(
		container.Logger(),
	).GetImage(
		ctx,
		container,
		imageRef,
		flags.Files,
		false,
		flags.ExcludeSourceInfo,
	)
	if err != nil {
		return err
	}
	imageRef, err = buffetch.NewImageRefParser(container.Logger()).GetImageRef(ctx, flags.Output)
	if err != nil {
		return fmt.Errorf("--%s: %v", outputFlagName, err)
	}
	return internal.NewBufwireImageWriter(
		container.Logger(),
	).PutImage(
		ctx,
		container,
		imageRef,
		image,
		flags.AsFileDescriptorSet,
		flags.ExcludeImports,
	)
}
