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
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/bandeps"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	name = "bandeps"

	configFileFlagName      = "config-file"
	configFileFlagShortName = "f"

	timeout = 120 * time.Second
)

func main() {
	appcmd.Main(context.Background(), newCommand())
}

func newCommand() *appcmd.Command {
	builder := appflag.NewBuilder(
		name,
		appflag.BuilderWithTimeout(timeout),
		appflag.BuilderWithTracing(),
	)
	flags := newFlags()
	return &appcmd.Command{
		Use: name,
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appflag.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindPersistentFlags: builder.BindRoot,
		BindFlags:           flags.Bind,
	}
}

type flags struct {
	ConfigFile string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVarP(
		&f.ConfigFile,
		configFileFlagName,
		configFileFlagShortName,
		"",
		"The config file to use.",
	)
	_ = cobra.MarkFlagRequired(flagSet, configFileFlagName)
}

func run(ctx context.Context, container appflag.Container, flags *flags) error {
	configData, err := os.ReadFile(flags.ConfigFile)
	if err != nil {
		return err
	}
	var externalConfig bandeps.ExternalConfig
	if err := encoding.UnmarshalJSONOrYAMLStrict(configData, &externalConfig); err != nil {
		return err
	}
	violations, err := bandeps.NewChecker(
		container.Logger(),
		command.NewRunner(),
	).Check(
		ctx,
		container,
		externalConfig,
	)
	if err != nil {
		return err
	}
	if len(violations) > 0 {
		for _, violation := range violations {
			if _, err := fmt.Fprintln(container.Stdout(), violation.String()); err != nil {
				return err
			}
		}
		return errors.New("")
	}
	return nil
}
