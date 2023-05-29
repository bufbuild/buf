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

package bufinit

import (
	"context"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufinit"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	excludePathsFlagName = "exclude-path"
)

// NewCommand returns a new init Command.
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " [path/to/dir]",
		Short: "Initializes buf configuration.",
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
	ExcludePaths []string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	bufcli.BindExcludePaths(flagSet, &f.ExcludePaths, excludePathsFlagName)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) error {
	var dirPath string
	switch numArgs := container.NumArgs(); numArgs {
	case 0:
		dirPath = "."
	case 1:
		dirPath = container.Arg(0)
	default:
		return appcmd.NewInvalidArgumentErrorf("only 1 argument allowed but %d arguments specified", numArgs)
	}
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	readWriteBucket, err := storageosProvider.NewReadWriteBucket(
		dirPath,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return err
	}
	return bufinit.NewInitializer(container.Logger()).Initialize(
		ctx,
		readWriteBucket,
		bufinit.InitializeWithExcludePaths(flags.ExcludePaths...),
	)
}
