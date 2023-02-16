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

package stats

import (
	"context"
	"encoding/json"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/protostat"
	"github.com/bufbuild/buf/private/pkg/protostat/protostatos"
	"github.com/bufbuild/buf/private/pkg/protostat/protostatstorage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/spf13/pflag"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " files...",
		Short: "Get statistics for a list of Protobuf files",
		Long: `The input to this command is different than other buf commands:

- If no arguments are provided, this command searches for all .proto files under the current
  directory and uses them as inputs. This may include vendored files, test files, etc.
- If you want to check specific files, input them as arguments.

Examples:

Use all .proto files under the current directory:

    $ buf alpha stats

Use all .proto files in the module in the ./proto directory:

    $ buf alpha stats $(buf ls-files)

Use all .proto files in the current directory that do not include "foo" in the name:

    $ buf alpha stats $(find . -name '*.proto' | grep -v foo )`,
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appflag.Container) error {
				return run(ctx, container, flags)
			},
			bufcli.NewErrorInterceptor(),
		),
		BindFlags: flags.Bind,
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
	var fileWalker protostat.FileWalker
	if container.NumArgs() == 0 {
		storageosProvider := bufcli.NewStorageosProvider(false)
		readWriteBucket, err := storageosProvider.NewReadWriteBucket(
			".",
			storageos.ReadWriteBucketWithSymlinksIfSupported(),
		)
		if err != nil {
			return err
		}
		fileWalker = protostatstorage.NewFileWalker(readWriteBucket)
	} else {
		fileWalker = protostatos.NewFileWalker(app.Args(container)...)
	}
	stats, err := protostat.GetStats(ctx, fileWalker)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return err
	}
	_, err = container.Stdout().Write(append(data, []byte("\n")...))
	return err
}
