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

package modexport

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/internal/buf/bufcli"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/buf/buffetch"
	"github.com/bufbuild/buf/internal/pkg/app/appcmd"
	"github.com/bufbuild/buf/internal/pkg/app/appflag"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	outputFlagName      = "output"
	outputFlagShortName = "o"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appflag.Builder,
	moduleResolverReaderProvider bufcli.ModuleResolverReaderProvider,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <module_name>",
		Short: "Export a module to a directory.",
		Args:  cobra.ExactArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appflag.Container) error {
				return run(ctx, container, flags, moduleResolverReaderProvider)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	Output string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVarP(
		&f.Output,
		outputFlagName,
		outputFlagShortName,
		"",
		"Required. The location to export the module to. Must be a local directory.",
	)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
	moduleResolverReaderProvider bufcli.ModuleResolverReaderProvider,
) error {
	if flags.Output == "" {
		return appcmd.NewInvalidArgumentErrorf("--%s is required", outputFlagName)
	}
	moduleRef, err := buffetch.NewModuleRefParser(
		container.Logger(),
	).GetModuleRef(
		ctx,
		container.Arg(0),
	)
	if err != nil {
		return bufcli.NewModuleRefError(container.Arg(0))
	}
	moduleResolver, err := moduleResolverReaderProvider.GetModuleResolver(ctx, container)
	if err != nil {
		return err
	}
	moduleReader, err := moduleResolverReaderProvider.GetModuleReader(ctx, container)
	if err != nil {
		return err
	}
	moduleIdentity, err := bufmodule.ModuleReferenceForString(container.Arg(0))
	if err != nil {
		return err
	}
	ctx, err = bufcli.WithHeaders(ctx, container, moduleIdentity.Remote())
	if err != nil {
		return err
	}
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	module, err := buffetch.NewModuleFetcher(
		container.Logger(),
		storageosProvider,
		moduleResolver,
		moduleReader,
	).GetModule(
		ctx,
		container,
		moduleRef,
	)
	if err != nil {
		return fmt.Errorf("could not resolve module %s: %v", container.Arg(0), err)
	}
	writeBucket, err := storageosProvider.NewReadWriteBucket(
		normalpath.Normalize(flags.Output),
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return fmt.Errorf("failed to export module files into %s: %v", flags.Output, err)
	}
	// note that this only writes sources and the buf.lock file
	if err := bufmodule.ModuleToBucket(ctx, module, writeBucket); err != nil {
		return bufcli.NewInternalError(err)
	}
	return nil
}
