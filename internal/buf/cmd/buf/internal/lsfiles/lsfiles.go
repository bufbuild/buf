// Copyright 2020 Buf Technologies, Inc.
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

package lsfiles

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/internal/buf/buffetch"
	"github.com/bufbuild/buf/internal/buf/cmd/internal"
	"github.com/bufbuild/buf/internal/pkg/app/appcmd"
	"github.com/bufbuild/buf/internal/pkg/app/appflag"
	"github.com/bufbuild/buf/internal/pkg/app/applog"
	"github.com/spf13/pflag"
)

const (
	inputFlagName  = "input"
	configFlagName = "input-config"
)

// NewCommand returns a new Command
func NewCommand(use string, builder appflag.Builder) *appcmd.Command {
	controller := newController()
	return &appcmd.Command{
		Use:   use,
		Short: "List all Protobuf files for the input location.",
		Run: builder.NewRunFunc(
			func(ctx context.Context, container applog.Container) error {
				return controller.Run(ctx, container)
			},
		),
		BindFlags: controller.Bind,
	}
}

func newController() *controller {
	return &controller{}
}

type controller struct {
	input                string
	config               string
	experimentalGitClone bool
}

func (c *controller) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(
		&c.input,
		inputFlagName,
		".",
		fmt.Sprintf(
			`The source or image to list the files from. Must be one of format %s.`,
			buffetch.AllFormatsString,
		),
	)
	flagSet.StringVar(
		&c.config,
		configFlagName,
		"",
		`The config file or data to use.`,
	)
	internal.BindExperimentalGitClone(flagSet, &c.experimentalGitClone)
}

func (c *controller) Run(ctx context.Context, container applog.Container) (retErr error) {
	fileRefs, err := internal.NewBufwireEnvReader(
		container.Logger(),
		inputFlagName,
		configFlagName,
	).ListFiles(
		ctx,
		container,
		c.input,
		c.config,
	)
	if err != nil {
		return err
	}
	for _, fileRef := range fileRefs {
		if _, err := fmt.Fprintln(container.Stdout(), fileRef.ExternalPath()); err != nil {
			return err
		}
	}
	return nil
}
