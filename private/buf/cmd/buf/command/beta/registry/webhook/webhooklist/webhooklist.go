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

package webhooklist

import (
	"context"
	"encoding/json"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	ownerFlagName      = "owner"
	repositoryFlagName = "repository"
	remoteFlagName     = "remote"
)

// NewCommand returns a new Command
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name,
		Short: "List module webhooks.",
		Args:  cobra.ExactArgs(0),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appflag.Container) error {
				return run(ctx, container, flags)
			},
			bufcli.NewErrorInterceptor(),
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	OwnerName      string
	RepositoryName string
	Remote         string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(
		&f.OwnerName,
		ownerFlagName,
		"",
		`The owner name of the module to list webhooks for.`,
	)
	_ = cobra.MarkFlagRequired(flagSet, ownerFlagName)
	flagSet.StringVar(
		&f.RepositoryName,
		repositoryFlagName,
		"",
		"The module name to list webhooks for.",
	)
	_ = cobra.MarkFlagRequired(flagSet, repositoryFlagName)
	flagSet.StringVar(
		&f.Remote,
		remoteFlagName,
		"",
		"The remote of the owner and module to list webhooks for.",
	)
	_ = cobra.MarkFlagRequired(flagSet, remoteFlagName)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) error {
	bufcli.WarnBetaCommand(ctx, container)
	apiProvider, err := bufcli.NewRegistryProvider(ctx, container)
	if err != nil {
		return err
	}
	service, err := apiProvider.NewWebhookService(ctx, flags.Remote)
	if err != nil {
		return err
	}
	results, _, err := service.ListWebhooks(ctx, flags.RepositoryName, flags.OwnerName, "")
	if err != nil {
		return err
	}
	if results == nil {
		// Ignore errors for writing to stdout.
		_, _ = container.Stdout().Write([]byte("[]"))
	}
	response, err := json.MarshalIndent(results, "", "\t")
	if err != nil {
		return err
	}
	// Ignore errors for writing to stdout.
	_, _ = container.Stdout().Write(response)
	return nil
}
