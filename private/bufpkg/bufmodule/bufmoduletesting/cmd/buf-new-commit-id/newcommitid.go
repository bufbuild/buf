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
	"fmt"
	"time"

	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"github.com/spf13/pflag"
)

const (
	name         = "buf-new-commit-id"
	typeFlagName = "type"
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
		Use:   name,
		Short: "Produce a new commit ID for testing.",
		Long: fmt.Sprintf(`If --%s v1 is specified, a dashless commit ID is produced.
If --%s v2 is specified, a dashful commit ID is produced.`, typeFlagName, typeFlagName),
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
	Type string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(
		&f.Type,
		typeFlagName,
		"v2",
		"The commit ID type, either v1 or v2.",
	)
}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) error {
	commitID, err := uuidutil.New()
	if err != nil {
		return err
	}
	var commitIDString string
	switch flags.Type {
	case "v1":
		commitIDString, err = uuidutil.ToDashless(commitID)
		if err != nil {
			return err
		}
	case "v2":
		commitIDString = commitID.String()
	default:
		return appcmd.NewInvalidArgumentErrorf("invalid value for --%s: %q", typeFlagName, flags.Type)
	}
	_, err = container.Stdout().Write([]byte(commitIDString + "\n"))
	return err
}
