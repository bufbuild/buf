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
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"github.com/gofrs/uuid/v5"
	"github.com/spf13/pflag"
)

const (
	name = "buf-commit-id-to-dashless"
)

func main() {
	appcmd.Main(context.Background(), newCommand())
}

func newCommand() *appcmd.Command {
	builder := appext.NewBuilder(
		name,
		appext.BuilderWithTimeout(120*time.Second),
		appext.BuilderWithTracing(),
	)
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + "commit_id",
		Short: "Convert a commit ID to dashless.",
		Args:  appcmd.ExactArgs(1),
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
	commitID, err := uuid.FromString(container.Arg(0))
	if err != nil {
		return err
	}
	commitIDString := uuidutil.ToDashless(commitID)
	_, err = container.Stdout().Write([]byte(commitIDString + "\n"))
	return err
}
