// Copyright 2020-2024 Buf Technologies, Inc.
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
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/slogapp"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/spf13/pflag"
)

const (
	name               = "buf-digest"
	digestTypeFlagName = "digest-type"
)

func main() {
	appcmd.Main(context.Background(), newCommand())
}

func newCommand() *appcmd.Command {
	builder := appext.NewBuilder(
		name,
		appext.BuilderWithTimeout(120*time.Second),
		appext.BuilderWithLoggerProvider(slogapp.LoggerProvider),
	)
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <path/to/module1> <path/to/module2> ...",
		Short: "Produce a digest for a set of self-contained modules.",
		Long: `This is a low-level command used to generate digests for modules on disk.
The modules given must be self-contained, i.e. all dependencies must be represented.
This is not intended to be used outside of development of the buf codebase.

If a given module contains a v1 buf.yaml and/or buf.lock, they will be included
in the B4 digest calculations.`,
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags:           flags.Bind,
		BindPersistentFlags: builder.BindRoot,
	}
}

type flags struct {
	DigestType string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(
		&f.DigestType,
		digestTypeFlagName,
		bufmodule.DigestTypeB5.String(),
		fmt.Sprintf(
			"The digest type. Must be one of %s",
			stringutil.SliceToString(slicesext.Map(bufmodule.AllDigestTypes, bufmodule.DigestType.String)),
		),
	)
}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) error {
	digestType, err := bufmodule.ParseDigestType(flags.DigestType)
	if err != nil {
		return appcmd.NewInvalidArgumentErrorf("--%s: %w", digestTypeFlagName, err)
	}
	dirPaths := app.Args(container)
	if len(dirPaths) == 0 {
		dirPaths = []string{"."}
	}
	moduleSetBuilder := bufmodule.NewModuleSetBuilder(ctx, container.Logger(), bufmodule.NopModuleDataProvider, bufmodule.NopCommitProvider)
	storageosProvider := storageos.NewProvider()
	for _, dirPath := range dirPaths {
		bucket, err := storageosProvider.NewReadWriteBucket(dirPath)
		if err != nil {
			return err
		}
		v1BufYAMLObjectData, err := bufconfig.GetBufYAMLV1Beta1OrV1ObjectDataForPrefix(ctx, bucket, ".")
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return err
			}
		}
		v1BufLockObjectData, err := bufconfig.GetBufLockV1Beta1OrV1ObjectDataForPrefix(ctx, bucket, ".")
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return err
			}
		}
		moduleSetBuilder.AddLocalModule(
			bucket,
			dirPath,
			true,
			bufmodule.LocalModuleWithV1Beta1OrV1BufYAMLObjectData(v1BufYAMLObjectData),
			bufmodule.LocalModuleWithV1Beta1OrV1BufLockObjectData(v1BufLockObjectData),
		)
	}
	moduleSet, err := moduleSetBuilder.Build()
	if err != nil {
		return err
	}
	for _, module := range moduleSet.Modules() {
		digest, err := module.Digest(digestType)
		if err != nil {
			return err
		}
		if _, err := container.Stdout().Write([]byte(module.OpaqueID() + " " + digest.String() + "\n")); err != nil {
			return err
		}
	}
	return nil
}
