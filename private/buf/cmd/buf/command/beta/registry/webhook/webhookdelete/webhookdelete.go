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

package webhookdelete

import (
	"context"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	webhookIDFlagName = "id"
	remoteFlagName    = "remote"
)

// NewCommand returns a new Command
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name,
		Short: "Delete a repository webhook.",
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
	WebhookID string
	Remote    string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(
		&f.WebhookID,
		webhookIDFlagName,
		"",
		"The webhook ID to delete.",
	)
	_ = cobra.MarkFlagRequired(flagSet, webhookIDFlagName)
	flagSet.StringVar(
		&f.Remote,
		remoteFlagName,
		"",
		"The remote of the repository the webhook ID belongs to.",
	)
	_ = cobra.MarkFlagRequired(flagSet, remoteFlagName)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) error {
	bufcli.WarnBetaCommand(ctx, container)
	storageosProvider := bufcli.NewStorageosProvider(false)
	runner := command.NewRunner()
	_, moduleIdentity, err := bufcli.ReadModuleWithWorkspacesDisabled(
		ctx,
		container,
		storageosProvider,
		runner,
		flags.Remote,
	)
	if err != nil {
		return err
	}
	apiProvider, err := bufcli.NewRegistryProvider(ctx, container)
	if err != nil {
		return err
	}
	remote := moduleIdentity.Remote()
	service, err := apiProvider.NewWebhookService(ctx, remote)
	if err != nil {
		return err
	}
	if err := service.DeleteWebhook(ctx, flags.WebhookID); err != nil {
		return err
	}
	return nil
}
