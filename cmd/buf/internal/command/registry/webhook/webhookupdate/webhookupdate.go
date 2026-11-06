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

package webhookupdate

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
	nameFlagName                = "name"
	idFlagName                  = "id"
	newNameFlagName             = "new-name"
	urlFlagName                 = "url"
	eventFlagName               = "event"
	signingSecretFileFlagName   = "signing-secret-file"
	removeSigningSecretFlagName = "remove-signing-secret"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " " + internal.ScopeArgUsage,
		Short: "Update a BSR webhook",
		Long:  internal.ScopeArgLong + "\n\nThe scope of a webhook cannot be changed.",
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
	Name                string
	ID                  string
	NewName             *string
	URL                 *string
	Events              []string
	SigningSecretFile   string
	RemoveSigningSecret bool
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
	bufcli.BindStringPointer(
		flagSet,
		newNameFlagName,
		&f.NewName,
		"The new name for the webhook, unique within the scope",
	)
	bufcli.BindStringPointer(
		flagSet,
		urlFlagName,
		&f.URL,
		"The new URL that events are delivered to",
	)
	flagSet.StringSliceVar(
		&f.Events,
		eventFlagName,
		nil,
		internal.EventFlagUsage("The event type to subscribe to, replacing the existing event types"),
	)
	flagSet.StringVar(
		&f.SigningSecretFile,
		signingSecretFileFlagName,
		"",
		`The path to a file containing the new secret used to sign deliveries, or "-" to read from stdin. Trailing newlines are removed`,
	)
	flagSet.BoolVar(
		&f.RemoveSigningSecret,
		removeSigningSecretFlagName,
		false,
		"Remove the signing secret, so that deliveries are not signed",
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
	webhookEventTypes, err := internal.ParseWebhookEventTypes(flags.Events)
	if err != nil {
		return appcmd.WrapInvalidArgumentError(err)
	}
	if flags.SigningSecretFile != "" && flags.RemoveSigningSecret {
		return appcmd.NewInvalidArgumentErrorf(
			"cannot set both --%s and --%s",
			signingSecretFileFlagName,
			removeSigningSecretFlagName,
		)
	}
	var signingSecret []byte
	switch {
	case flags.SigningSecretFile != "":
		signingSecret, err = internal.ReadSigningSecretFile(container, flags.SigningSecretFile)
		if err != nil {
			return err
		}
	case flags.RemoveSigningSecret:
		// A set but empty secret removes the existing secret.
		signingSecret = []byte{}
	}
	if flags.NewName == nil && flags.URL == nil && len(webhookEventTypes) == 0 && signingSecret == nil {
		return appcmd.NewInvalidArgumentErrorf(
			"at least one of --%s, --%s, --%s, --%s, or --%s is required",
			newNameFlagName,
			urlFlagName,
			eventFlagName,
			signingSecretFileFlagName,
			removeSigningSecretFlagName,
		)
	}
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	webhookServiceClient := bufregistryapiwebhook.NewClientProvider(clientConfig).V1WebhookServiceClient(registry)
	if _, err := webhookServiceClient.UpdateWebhooks(
		ctx,
		&webhookv1.UpdateWebhooksRequest{
			Values: []*webhookv1.UpdateWebhooksRequest_Value{
				{
					WebhookRef:    webhookRef,
					Url:           flags.URL,
					EventTypes:    webhookEventTypes,
					Name:          flags.NewName,
					SigningSecret: signingSecret,
				},
			},
		},
	); err != nil {
		switch connect.CodeOf(err) {
		case connect.CodeNotFound:
			return bufcli.NewWebhookNotFoundError(cmp.Or(flags.Name, flags.ID), container.Arg(0))
		case connect.CodeAlreadyExists:
			if flags.NewName != nil {
				return bufcli.NewWebhookNameAlreadyExistsError(*flags.NewName, container.Arg(0))
			}
		}
		return err
	}
	if _, err := fmt.Fprintln(container.Stdout(), "Webhook updated."); err != nil {
		return syserror.Wrap(err)
	}
	return nil
}
