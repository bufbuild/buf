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

package push

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/rpc"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	trackFlagName           = "track"
	tagFlagName             = "tag"
	tagFlagShortName        = "t"
	errorFormatFlagName     = "error-format"
	disableSymlinksFlagName = "disable-symlinks"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <source>",
		Short: "Push a module to a registry.",
		Long:  bufcli.GetSourceLong(`the source to push`),
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
	Tracks          []string
	Tags            []string
	ErrorFormat     string
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
	flagSet.StringSliceVar(
		&f.Tracks,
		trackFlagName,
		nil,
		"Append the pushed module to this track. Multiple tracks are appended if specified multiple times.",
	)
	flagSet.StringSliceVarP(
		&f.Tags,
		tagFlagName,
		tagFlagShortName,
		nil,
		"Create a tag for the pushed commit. Multiple tags are created if specified multiple times.",
	)
	flagSet.StringVar(
		&f.ErrorFormat,
		errorFormatFlagName,
		"text",
		fmt.Sprintf(
			"The format for build errors printed to stderr. Must be one of %s.",
			stringutil.SliceToString(bufanalysis.AllFormatStrings),
		),
	)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) (retErr error) {
	if err := bufcli.ValidateErrorFormatFlag(flags.ErrorFormat, errorFormatFlagName); err != nil {
		return err
	}
	source, err := bufcli.GetInputValue(container, flags.InputHashtag, ".")
	if err != nil {
		return err
	}
	storageosProvider := bufcli.NewStorageosProvider(flags.DisableSymlinks)
	runner := command.NewRunner()
	// We are pushing to the BSR, this module has to be independently buildable
	// given the configuration it has without any enclosing workspace.
	module, moduleIdentity, err := bufcli.ReadModuleWithWorkspacesDisabled(
		ctx,
		container,
		storageosProvider,
		runner,
		source,
	)
	if err != nil {
		return err
	}
	protoModule, err := bufmodule.ModuleToProtoModule(ctx, module)
	if err != nil {
		return err
	}
	apiProvider, err := bufcli.NewRegistryProvider(ctx, container)
	if err != nil {
		return err
	}
	service, err := apiProvider.NewPushService(ctx, moduleIdentity.Remote())
	if err != nil {
		return err
	}
	tracks := flags.Tracks
	if tracks == nil {
		tracks = []string{bufmoduleref.MainTrack}
	}
	localModulePin, err := service.Push(
		ctx,
		moduleIdentity.Owner(),
		moduleIdentity.Repository(),
		"",
		protoModule,
		flags.Tags,
		tracks,
	)
	if err != nil {
		if rpc.GetErrorCode(err) == rpc.ErrorCodeAlreadyExists {
			if _, err := container.Stderr().Write(
				[]byte("The latest commit has the same content; not creating a new commit.\n"),
			); err != nil {
				return err
			}
			return nil
		}
		return err
	}
	if _, err := container.Stdout().Write([]byte(localModulePin.Commit + "\n")); err != nil {
		return err
	}
	return nil
}
