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

package build

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimageutil"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	asFileDescriptorSetFlagName = "as-file-descriptor-set"
	errorFormatFlagName         = "error-format"
	excludeImportsFlagName      = "exclude-imports"
	excludeSourceInfoFlagName   = "exclude-source-info"
	pathsFlagName               = "path"
	outputFlagName              = "output"
	outputFlagShortName         = "o"
	configFlagName              = "config"
	excludePathsFlagName        = "exclude-path"
	disableSymlinksFlagName     = "disable-symlinks"
	watchFlagName               = "watch"
	watchFlagShortName          = "w"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <input>",
		Short: "Build Protobuf files into a Buf image",
		Long:  bufcli.GetInputLong(`the source or module to build or image to convert`),
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
	AsFileDescriptorSet bool
	ErrorFormat         string
	ExcludeImports      bool
	ExcludeSourceInfo   bool
	Paths               []string
	Output              string
	Config              string
	ExcludePaths        []string
	DisableSymlinks     bool
	Types               []string
	Watch               bool
	// special
	InputHashtag string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	bufcli.BindInputHashtag(flagSet, &f.InputHashtag)
	bufcli.BindAsFileDescriptorSet(flagSet, &f.AsFileDescriptorSet, asFileDescriptorSetFlagName)
	bufcli.BindExcludeImports(flagSet, &f.ExcludeImports, excludeImportsFlagName)
	bufcli.BindExcludeSourceInfo(flagSet, &f.ExcludeSourceInfo, excludeSourceInfoFlagName)
	bufcli.BindPaths(flagSet, &f.Paths, pathsFlagName)
	bufcli.BindExcludePaths(flagSet, &f.ExcludePaths, excludePathsFlagName)
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
	flagSet.StringVarP(
		&f.Output,
		outputFlagName,
		outputFlagShortName,
		app.DevNullFilePath,
		fmt.Sprintf(
			`The output location for the built image. Must be one of format %s`,
			buffetch.ImageFormatsString,
		),
	)
	flagSet.StringVar(
		&f.Config,
		configFlagName,
		"",
		`The file or data to use to use for configuration`,
	)
	flagSet.StringSliceVar(
		&f.Types,
		"type",
		nil,
		"The types (message, enum, service) that should be included in this image. When specified, the resulting image will only include descriptors to describe the requested types",
	)
	flagSet.BoolVarP(
		&f.Watch,
		watchFlagName,
		watchFlagShortName,
		false,
		"Watch.",
	)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) (retErr error) {
	if flags.Output == "" {
		return appcmd.NewInvalidArgumentErrorf("required flag %q not set", outputFlagName)
	}
	if err := bufcli.ValidateErrorFormatFlag(flags.ErrorFormat, errorFormatFlagName); err != nil {
		return err
	}
	input, err := bufcli.GetInputValue(container, flags.InputHashtag, ".")
	if err != nil {
		return err
	}
	ref, err := buffetch.NewRefParser(container.Logger()).GetRef(ctx, input)
	if err != nil {
		return err
	}
	if !flags.Watch {
		return runForRef(ctx, container, flags, ref)
	}
	refPath := ref.ExternalPath()
	if refPath == "" {
		return appcmd.NewInvalidArgumentErrorf("cannot watch %q", refPath)
	}
	return bufcli.WatchProtoFiles(
		ctx,
		refPath,
		func(err error) {
			if err != nil {
				container.Stderr().Write([]byte(err.Error() + "\n"))
			}
		},
		func() error {
			return runForRef(ctx, container, flags, ref)
		},
	)
}

func runForRef(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
	ref buffetch.Ref,
) error {
	image, err := bufcli.NewImageForSource(
		ctx,
		container,
		ref,
		flags.ErrorFormat,
		flags.DisableSymlinks,
		flags.Config,
		flags.Paths,
		flags.ExcludePaths, // we exclude these paths
		false,
		flags.ExcludeSourceInfo,
	)
	if err != nil {
		return err
	}
	imageRef, err := buffetch.NewImageRefParser(container.Logger()).GetImageRef(ctx, flags.Output)
	if err != nil {
		return fmt.Errorf("--%s: %v", outputFlagName, err)
	}
	if len(flags.Types) > 0 {
		image, err = bufimageutil.ImageFilteredByTypes(image, flags.Types...)
		if err != nil {
			return err
		}
	}
	return bufcli.NewWireImageWriter(
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
