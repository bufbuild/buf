// Copyright 2020-2026 Buf Technologies, Inc.
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
	"cmp"
	"context"
	"fmt"

	webhookv1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/webhook/v1"
	"buf.build/go/app/appcmd"
	"buf.build/go/app/appext"
	"connectrpc.com/connect/v2"
	"github.com/bufbuild/buf/cmd/buf/internal/command/registry/webhook/internal"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufregistryapi/bufregistryapiwebhook"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/spf13/pflag"
)

const (
	nameFlagName  = "name"
	idFlagName    = "id"
	forceFlagName = "force"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " " + internal.ScopeArgUsage,
		Short: "Delete a BSR webhook",
		Long:  internal.ScopeArgLong,
		Args:  appcmd.ExactArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	Name  string
	ID    string
	Force bool
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(
		&f.Name,
		nameFlagName,
		"",
		fmt.Sprintf("The name of the webhook within the scope. Exactly one of --%s or --%s is required", nameFlagName, idFlagName),
	)
	flagSet.StringVar(
		&f.ID,
		idFlagName,
		"",
		fmt.Sprintf("The id of the webhook. Exactly one of --%s or --%s is required", nameFlagName, idFlagName),
	)
	flagSet.BoolVar(
		&f.Force,
		forceFlagName,
		false,
		"Force deletion without confirming. Use with caution",
	)
}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) error {
	registry, webhookScopeRef, err := bufcli.ParseWebhookScope(container.Arg(0))
	if err != nil {
		return appcmd.WrapInvalidArgumentError(err)
	}
	webhookRef, err := bufcli.NewWebhookRef(webhookScopeRef, flags.Name, flags.ID)
	if err != nil {
		return appcmd.WrapInvalidArgumentError(err)
	}
	nameOrID := cmp.Or(flags.Name, flags.ID)
	if !flags.Force {
		if err := bufcli.PromptUserForDelete(container, "webhook", nameOrID); err != nil {
			return err
		}
	}
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	webhookServiceClient := bufregistryapiwebhook.NewClientProvider(clientConfig).V1WebhookServiceClient(registry)
	if _, err := webhookServiceClient.DeleteWebhooks(
		ctx,
		&webhookv1.DeleteWebhooksRequest{
			WebhookRefs: []*webhookv1.WebhookRef{webhookRef},
		},
	); err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return bufcli.NewWebhookNotFoundError(nameOrID, container.Arg(0))
		}
		return err
	}
	if _, err := fmt.Fprintln(container.Stdout(), "Webhook deleted."); err != nil {
		return syserror.Wrap(err)
	}
	return nil
}
