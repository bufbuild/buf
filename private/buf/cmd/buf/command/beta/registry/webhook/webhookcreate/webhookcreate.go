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

package webhookcreate

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bufbuild/buf/private/buf/bufcli"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	ownerFlagName        = "owner"
	repositoryFlagName   = "repository"
	callbackURLFlagName  = "callback_url"
	webhookEventFlagName = "event"
)

// NewCommand returns a new Command
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <source>",
		Short: "Create a repository webhook.",
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
	WebhookEvent   string
	OwnerName      string
	RepositoryName string
	CallbackURL    string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(
		&f.WebhookEvent,
		webhookEventFlagName,
		"",
		"The event type to create a webhook for. The proto enum string value is used for this input (e.g. 'WEBHOOK_EVENT_REPOSITORY_PUSH').",
	)
	_ = cobra.MarkFlagRequired(flagSet, webhookEventFlagName)
	flagSet.StringVar(
		&f.OwnerName,
		ownerFlagName,
		"",
		`The owner name of the repository to create a webhook for.`,
	)
	_ = cobra.MarkFlagRequired(flagSet, ownerFlagName)
	flagSet.StringVar(
		&f.RepositoryName,
		repositoryFlagName,
		"",
		"The repository name to create a webhook for.",
	)
	_ = cobra.MarkFlagRequired(flagSet, repositoryFlagName)
	flagSet.StringVar(
		&f.CallbackURL,
		callbackURLFlagName,
		"",
		"The url for the webhook to callback to on a given event.",
	)
	_ = cobra.MarkFlagRequired(flagSet, callbackURLFlagName)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) error {
	bufcli.WarnBetaCommand(ctx, container)
	input, err := bufcli.GetInputValue(container, "", ".")
	if err != nil {
		return err
	}
	storageosProvider := bufcli.NewStorageosProvider(false)
	runner := command.NewRunner()
	_, moduleIdentity, err := bufcli.ReadModuleWithWorkspacesDisabled(
		ctx,
		container,
		storageosProvider,
		runner,
		input,
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
	value, ok := registryv1alpha1.WebhookEvent_value[flags.WebhookEvent]
	if !ok || value == int32(registryv1alpha1.WebhookEvent_WEBHOOK_EVENT_UNSPECIFIED) {
		return fmt.Errorf("webhook event must be specified")
	}
	event := registryv1alpha1.WebhookEvent(value)
	createWebhook, err := service.CreateWebhook(
		ctx,
		event,
		flags.OwnerName,
		flags.RepositoryName,
		flags.CallbackURL,
	)
	if err != nil {
		return err
	}
	createWebhookResponse, err := json.MarshalIndent(createWebhook, "", "\t")
	if err != nil {
		return err
	}
	// Ignore errors for writing to stdout.
	_, _ = container.Stdout().Write(createWebhookResponse)
	return nil
}
