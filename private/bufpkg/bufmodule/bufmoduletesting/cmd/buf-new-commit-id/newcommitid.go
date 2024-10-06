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
	"time"

	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/slogapp"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"github.com/spf13/pflag"
)

const (
	name = "buf-new-commit-id"
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
		Use:   name,
		Short: "Produce a new commit ID for testing.",
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
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
	container appext.Container,
	flags *flags,
) error {
	commitID, err := uuidutil.New()
	if err != nil {
		return err
	}
	_, err = container.Stdout().Write([]byte(uuidutil.ToDashless(commitID) + "\n"))
	return err
}
