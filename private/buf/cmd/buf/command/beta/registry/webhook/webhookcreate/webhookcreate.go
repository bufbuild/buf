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

package webhookcreate

import (
	"context"
	"encoding/json"
	"fmt"

	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/spf13/pflag"
)

const (
	ownerFlagName        = "owner"
	repositoryFlagName   = "repository"
	callbackURLFlagName  = "callback-url"
	webhookEventFlagName = "event"
	remoteFlagName       = "remote"
)

// NewCommand returns a new Command
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name,
		Short: "Create a repository webhook",
		Args:  appcmd.ExactArgs(0),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	WebhookEvent   string
	OwnerName      string
	RepositoryName string
	CallbackURL    string
	Remote         string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(
		&f.WebhookEvent,
		webhookEventFlagName,
		"",
		"The event type to create a webhook for. The proto enum string value is used for this input (e.g. 'WEBHOOK_EVENT_REPOSITORY_PUSH')",
	)
	_ = appcmd.MarkFlagRequired(flagSet, webhookEventFlagName)
	flagSet.StringVar(
		&f.OwnerName,
		ownerFlagName,
		"",
		`The owner name of the repository to create a webhook for`,
	)
	_ = appcmd.MarkFlagRequired(flagSet, ownerFlagName)
	flagSet.StringVar(
		&f.RepositoryName,
		repositoryFlagName,
		"",
		"The repository name to create a webhook for",
	)
	_ = appcmd.MarkFlagRequired(flagSet, repositoryFlagName)
	flagSet.StringVar(
		&f.CallbackURL,
		callbackURLFlagName,
		"",
		"The url for the webhook to callback to on a given event",
	)
	_ = appcmd.MarkFlagRequired(flagSet, callbackURLFlagName)
	flagSet.StringVar(
		&f.Remote,
		remoteFlagName,
		"",
		"The remote of the repository the created webhook will belong to",
	)
	_ = appcmd.MarkFlagRequired(flagSet, remoteFlagName)
}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) error {
	bufcli.WarnBetaCommand(ctx, container)
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	service := connectclient.Make(clientConfig, flags.Remote, registryv1alpha1connect.NewWebhookServiceClient)
	event, ok := registryv1alpha1.WebhookEvent_value[flags.WebhookEvent]
	if !ok || event == int32(registryv1alpha1.WebhookEvent_WEBHOOK_EVENT_UNSPECIFIED) {
		return fmt.Errorf("webhook event must be specified")
	}
	resp, err := service.CreateWebhook(
		ctx,
		connect.NewRequest(
			&registryv1alpha1.CreateWebhookRequest{
				WebhookEvent:   registryv1alpha1.WebhookEvent(event),
				OwnerName:      flags.OwnerName,
				RepositoryName: flags.RepositoryName,
				CallbackUrl:    flags.CallbackURL,
			},
		),
	)
	if err != nil {
		return err
	}
	webhookJSON, err := json.MarshalIndent(resp.Msg.Webhook, "", "\t")
	if err != nil {
		return err
	}
	// Ignore errors for writing to stdout.
	_, _ = container.Stdout().Write(webhookJSON)
	return nil
}
