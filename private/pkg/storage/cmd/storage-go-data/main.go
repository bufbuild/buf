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

package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/storage/storagegodata"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	programName = "storage-go-data"
	pkgFlagName = "package"
)

func main() {
	appcmd.Main(context.Background(), newCommand())
}

func newCommand() *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:  fmt.Sprintf("%s path/to/dir", programName),
		Args: cobra.ExactArgs(1),
		Run: func(ctx context.Context, container app.Container) error {
			return run(ctx, container, flags)
		},
		BindFlags: flags.Bind,
	}
}

type flags struct {
	Pkg string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(
		&f.Pkg,
		pkgFlagName,
		"",
		"The name of the generated package.",
	)
}

func run(ctx context.Context, container app.Container, flags *flags) error {
	dirPath := container.Arg(0)
	readWriteBucket, err := storageos.NewProvider(storageos.ProviderWithSymlinks()).NewReadWriteBucket(container.Arg(0))
	if err != nil {
		return err
	}
	packageName := flags.Pkg
	if packageName == "" {
		packageName = filepath.Base(dirPath)
	}
	golangFileData, err := storagegodata.Build(
		ctx,
		readWriteBucket,
		storagegodata.BuildWithPackageName(packageName),
		storagegodata.BuildWithProgramName(programName),
	)
	if err != nil {
		return err
	}
	_, err = container.Stdout().Write(golangFileData)
	return err
}
