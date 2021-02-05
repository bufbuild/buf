// Copyright 2020-2021 Buf Technologies, Inc.
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

	"github.com/bufbuild/buf/internal/buf/bufanalysis"
	"github.com/bufbuild/buf/internal/buf/bufcli"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/pkg/app/appcmd"
	"github.com/bufbuild/buf/internal/pkg/app/appflag"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/bufbuild/buf/internal/pkg/stringutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	branchFlagName      = "branch"
	branchFlagShortName = "b"
	errorFormatFlagName = "error-format"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appflag.Builder,
	moduleResolverReaderProvider bufcli.ModuleResolverReaderProvider,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <input>",
		Short: "Push a module to a registry.",
		Long:  bufcli.GetInputLong(`the source or module to push`),
		Args:  cobra.MaximumNArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appflag.Container) error {
				return run(ctx, container, flags, moduleResolverReaderProvider)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	Branch      string
	ErrorFormat string
	// special
	InputHashtag string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	bufcli.BindInputHashtag(flagSet, &f.InputHashtag)
	flagSet.StringVarP(
		&f.Branch,
		branchFlagName,
		branchFlagShortName,
		bufmodule.MainBranch,
		`The branch to push to.`,
	)
	flagSet.StringVar(
		&f.ErrorFormat,
		errorFormatFlagName,
		"text",
		fmt.Sprintf(
			"The format for build errors, printed to stderr. Must be one of %s.",
			stringutil.SliceToString(bufanalysis.AllFormatStrings),
		),
	)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
	moduleResolverReaderProvider bufcli.ModuleResolverReaderProvider,
) (retErr error) {
	if flags.Branch == "" {
		return bufcli.NewFlagIsRequiredError(branchFlagName)
	}
	source, err := bufcli.GetInputValue(container, flags.InputHashtag, "", "", ".")
	if err != nil {
		return err
	}
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	module, moduleIdentity, err := bufcli.ReadModule(
		ctx,
		container,
		storageosProvider,
		source,
	)
	if err != nil {
		return err
	}
	ctx, err = bufcli.WithHeaders(ctx, container, moduleIdentity.Remote())
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
	localModulePin, err := service.Push(
		ctx,
		moduleIdentity.Owner(),
		moduleIdentity.Repository(),
		flags.Branch,
		protoModule,
	)
	if err != nil {
		return bufcli.NewRPCError("push", moduleIdentity.Remote(), err)
	}
	if _, err := container.Stdout().Write([]byte(localModulePin.Commit + "\n")); err != nil {
		return err
	}
	return nil
}
