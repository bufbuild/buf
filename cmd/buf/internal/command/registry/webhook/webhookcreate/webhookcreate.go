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

package webhookcreate

import (
	"context"
	"fmt"

	webhookv1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/webhook/v1"
	"buf.build/go/app/appcmd"
	"buf.build/go/app/appext"
	"connectrpc.com/connect/v2"
	"github.com/bufbuild/buf/cmd/buf/internal/command/registry/webhook/internal"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufprint"
	"github.com/bufbuild/buf/private/bufpkg/bufregistryapi/bufregistryapiwebhook"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	nameFlagName              = "name"
	urlFlagName               = "url"
	eventFlagName             = "event"
	signingSecretFileFlagName = "signing-secret-file"
	formatFlagName            = "format"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " " + internal.ScopeArgUsage,
		Short: "Create a BSR webhook",
		Long:  internal.ScopeArgLong,
		Args:  appcmd.ExactArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags: flags.Bind,
		ModifyCobra: func(cmd *cobra.Command) error {
			return bufcli.RegisterFlagCompletionOutputFormat(cmd, formatFlagName)
		},
	}
}

type flags struct {
	Name              string
	URL               string
	Events            []string
	SigningSecretFile string
	Format            string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(
		&f.Name,
		nameFlagName,
		"",
		"The name of the webhook, unique within the scope",
	)
	_ = appcmd.MarkFlagRequired(flagSet, nameFlagName)
	flagSet.StringVar(
		&f.URL,
		urlFlagName,
		"",
		"The URL that events are delivered to",
	)
	_ = appcmd.MarkFlagRequired(flagSet, urlFlagName)
	flagSet.StringSliceVar(
		&f.Events,
		eventFlagName,
		nil,
		internal.EventFlagUsage("The event type to subscribe to"),
	)
	_ = appcmd.MarkFlagRequired(flagSet, eventFlagName)
	flagSet.StringVar(
		&f.SigningSecretFile,
		signingSecretFileFlagName,
		"",
		`The path to a file containing the secret used to sign deliveries, or "-" to read from stdin. Trailing newlines are removed. If not set, deliveries are not signed`,
	)
	flagSet.StringVar(
		&f.Format,
		formatFlagName,
		bufprint.FormatText.String(),
		fmt.Sprintf(`The output format to use. Must be one of %s`, bufprint.AllFormatsString),
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
	webhookEventTypes, err := internal.ParseWebhookEventTypes(flags.Events)
	if err != nil {
		return appcmd.WrapInvalidArgumentError(err)
	}
	format, err := bufprint.ParseFormat(flags.Format)
	if err != nil {
		return appcmd.WrapInvalidArgumentError(err)
	}
	var signingSecret []byte
	if flags.SigningSecretFile != "" {
		signingSecret, err = internal.ReadSigningSecretFile(container, flags.SigningSecretFile)
		if err != nil {
			return err
		}
	}
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	webhookServiceClient := bufregistryapiwebhook.NewClientProvider(clientConfig).V1WebhookServiceClient(registry)
	resp, err := webhookServiceClient.CreateWebhooks(
		ctx,
		&webhookv1.CreateWebhooksRequest{
			Values: []*webhookv1.CreateWebhooksRequest_Value{
				{
					Name:          flags.Name,
					ScopeRef:      webhookScopeRef,
					EventTypes:    webhookEventTypes,
					Url:           flags.URL,
					SigningSecret: signingSecret,
				},
			},
		},
	)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeAlreadyExists {
			return bufcli.NewWebhookNameAlreadyExistsError(flags.Name, container.Arg(0))
		}
		return err
	}
	webhooks := resp.GetWebhooks()
	if len(webhooks) != 1 {
		return syserror.Newf("unexpected number of webhooks created by server: %d", len(webhooks))
	}
	if format == bufprint.FormatText {
		if _, err := fmt.Fprintf(container.Stdout(), "Created webhook %s in %s.\n", flags.Name, container.Arg(0)); err != nil {
			return syserror.Wrap(err)
		}
		return nil
	}
	return bufprint.PrintNames(
		container.Stdout(),
		format,
		bufprint.NewWebhookEntity(webhooks[0], container.Arg(0)),
	)
}
