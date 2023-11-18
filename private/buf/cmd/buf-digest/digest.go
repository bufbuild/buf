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

package digest

import (
	"context"
	"fmt"
	"time"

	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	name        = "buf-digest"
	depFlagName = "dep"
)

func main() {
	appcmd.Main(context.Background(), newCommand())
}

func newCommand() *appcmd.Command {
	builder := appflag.NewBuilder(
		name,
		appflag.BuilderWithTimeout(120*time.Second),
		appflag.BuilderWithTracing(),
	)
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <path/to/module>",
		Short: "Produce a digest for a module and its dependencies",
		Long: `This is a low-level command used to generate digest for modules on disk and their on-disk dependencies.

This is not intended to be used outside of development of the buf codebase.`,
		Args: cobra.ExactArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appflag.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags:           flags.Bind,
		BindPersistentFlags: builder.BindRoot,
	}
}

type flags struct {
	Deps []string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringSliceVar(
		&f.Deps,
		depFlagName,
		nil,
		`The path to a dependency on disk. This may or may not be an actual dependency of the input module - if it is not, it will not be used to calculate the digest`,
	)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) error {
	moduleSetBuilder := bufmodule.NewModuleSetBuilder(ctx, bufmodule.NopModuleDataProvider)
	storageosProvider := storageos.NewProvider()
	bucket, err := storageosProvider.NewReadWriteBucket(container.Arg(0))
	if err != nil {
		return err
	}
	moduleSetBuilder.AddLocalModule(bucket, container.Arg(0), true)
	for _, dep := range flags.Deps {
		bucket, err := storageosProvider.NewReadWriteBucket(dep)
		if err != nil {
			return err
		}
		moduleSetBuilder.AddLocalModule(bucket, dep, false)
	}
	moduleSet, err := moduleSetBuilder.Build()
	if err != nil {
		return err
	}
	module := moduleSet.GetModuleForOpaqueID(container.Arg(0))
	if module == nil {
		return fmt.Errorf("could not get module by opaque ID %s", container.Arg(0))
	}
	digest, err := module.Digest()
	if err != nil {
		return err
	}
	_, err = container.Stdout().Write([]byte(digest.String() + "\n"))
	return err
}
