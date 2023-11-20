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

package main

import (
	"context"
	"time"

	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
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
		Use:   name + " <path/to/module1> <path/to/module2> ...",
		Short: "Produce a digest for a set of self-contained modules.",
		Long: `This is a low-level command used to generate digests for modules on disk.
The modules given must be self-contained, i.e. all dependencies must be represented.
This is not intended to be used outside of development of the buf codebase.`,
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appflag.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags:           flags.Bind,
		BindPersistentFlags: builder.BindRoot,
	}
}

type flags struct{}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) error {
	dirPaths := app.Args(container)
	if len(dirPaths) == 0 {
		dirPaths = []string{"."}
	}
	moduleSetBuilder := bufmodule.NewModuleSetBuilder(ctx, bufmodule.NopModuleDataProvider)
	storageosProvider := storageos.NewProvider()
	for _, dirPath := range dirPaths {
		bucket, err := storageosProvider.NewReadWriteBucket(dirPath)
		if err != nil {
			return err
		}
		moduleSetBuilder.AddLocalModule(bucket, dirPath, true)
	}
	moduleSet, err := moduleSetBuilder.Build()
	if err != nil {
		return err
	}
	for _, module := range moduleSet.Modules() {
		digest, err := module.Digest()
		if err != nil {
			return err
		}
		if _, err := container.Stdout().Write([]byte(module.OpaqueID() + " " + digest.String() + "\n")); err != nil {
			return err
		}
	}
	return nil
}
