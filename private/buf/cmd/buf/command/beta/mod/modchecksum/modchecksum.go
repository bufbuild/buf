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

package modchecksum

import (
	"context"
	"sort"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/manifest"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	disableSymlinksFlagName = "disable-symlinks"
)

// NewCommand returns a new Command
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:       name + " <directory>",
		Short:     "Get module checksum",
		Args:      cobra.MaximumNArgs(1),
		BindFlags: flags.Bind,
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appflag.Container) error {
				return run(ctx, container, flags)
			}),
	}
}

type flags struct {
	DisableSymlinks bool
	InputHashtag    string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	bufcli.BindInputHashtag(flagSet, &f.InputHashtag)
	bufcli.BindDisableSymlinks(flagSet, &f.DisableSymlinks, disableSymlinksFlagName)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) error {
	bufcli.WarnBetaCommand(ctx, container)
	source, err := bufcli.GetInputValue(container, flags.InputHashtag, ".")
	if err != nil {
		return err
	}
	storageosProvider := bufcli.NewStorageosProvider(flags.DisableSymlinks)
	runner := command.NewRunner()
	// We are pushing to the BSR, this module has to be independently buildable
	// given the configuration it has without any enclosing workspace.
	sourceBucket, _, err := bufcli.BucketAndConfigForSource(
		ctx,
		container.Logger(),
		container,
		storageosProvider,
		runner,
		source,
	)
	if err != nil {
		return err
	}
	m, _, err := manifest.NewFromBucket(ctx, sourceBucket)
	if err != nil {
		return err
	}
	manifestBlob, err := m.Blob()
	if err != nil {
		return err
	}
	if container.Logger().Level().Enabled(zapcore.DebugLevel) {
		paths := m.Paths()
		sort.Strings(paths)
		for _, path := range paths {
			digest, ok := m.DigestFor(path)
			if !ok {
				continue
			}
			container.Logger().Debug("entry", zap.String("path", path), zap.String("digest", digest.String()))
		}
	}
	if _, err := container.Stdout().Write([]byte(manifestBlob.Digest().String() + "\n")); err != nil {
		return err
	}
	return nil
}
