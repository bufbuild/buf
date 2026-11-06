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

package webhookping

import (
	"cmp"
	"context"
	"fmt"
	"time"

	webhookv1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/webhook/v1"
	"buf.build/go/app/appcmd"
	"buf.build/go/app/appext"
	"connectrpc.com/connect/v2"
	"github.com/bufbuild/buf/cmd/buf/internal/command/registry/webhook/internal"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufprint"
	"github.com/bufbuild/buf/private/bufpkg/bufregistryapi/bufregistryapiwebhook"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	nameFlagName              = "name"
	idFlagName                = "id"
	urlFlagName               = "url"
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
		Short: "Send a ping event to a BSR webhook",
		Long: internal.ScopeArgLong + `

To ping an existing webhook, set --name or --id. Set --url to deliver the ping
to a different URL using the webhook's signing secret.

To ping a URL before creating a webhook, set --url without --name or --id. Set
--signing-secret-file to sign the ping.

The command fails if the ping is not delivered successfully.`,
		Args: appcmd.ExactArgs(1),
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
	ID                string
	URL               *string
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
		"The name of an existing webhook within the scope",
	)
	flagSet.StringVar(
		&f.ID,
		idFlagName,
		"",
		"The id of an existing webhook",
	)
	bufcli.BindStringPointer(
		flagSet,
		urlFlagName,
		&f.URL,
		"The URL to deliver the ping to. Required if neither --name nor --id is set",
	)
	flagSet.StringVar(
		&f.SigningSecretFile,
		signingSecretFileFlagName,
		"",
		`The path to a file containing the secret used to sign the ping, or "-" to read from stdin. Trailing newlines are removed. Only used if neither --name nor --id is set`,
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
	format, err := bufprint.ParseFormat(flags.Format)
	if err != nil {
		return appcmd.WrapInvalidArgumentError(err)
	}
	request, err := newPingWebhookRequest(container, flags, webhookScopeRef)
	if err != nil {
		return err
	}
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	webhookServiceClient := bufregistryapiwebhook.NewClientProvider(clientConfig).V1WebhookServiceClient(registry)
	resp, err := webhookServiceClient.PingWebhook(ctx, request)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound && request.GetExisting() != nil {
			return bufcli.NewWebhookNotFoundError(cmp.Or(flags.Name, flags.ID), container.Arg(0))
		}
		return err
	}
	duration := resp.GetDuration().AsDuration().Round(time.Millisecond)
	switch format {
	case bufprint.FormatText:
		if resp.GetErrorMessage() == "" {
			if _, err := fmt.Fprintf(
				container.Stdout(),
				"Ping delivered: HTTP %d in %s.\n",
				resp.GetHttpStatusCode(),
				duration,
			); err != nil {
				return syserror.Wrap(err)
			}
		}
	case bufprint.FormatJSON:
		data, err := protoencoding.NewJSONMarshaler(nil, protoencoding.JSONMarshalerWithIndent()).Marshal(resp)
		if err != nil {
			return err
		}
		if _, err := container.Stdout().Write(append(data, '\n')); err != nil {
			return syserror.Wrap(err)
		}
	default:
		return syserror.Newf("unknown format: %s", format)
	}
	if resp.GetErrorMessage() != "" {
		return fmt.Errorf(
			"ping failed after %s with HTTP status %d: %s",
			duration,
			resp.GetHttpStatusCode(),
			resp.GetErrorMessage(),
		)
	}
	return nil
}

func newPingWebhookRequest(
	container appext.Container,
	flags *flags,
	webhookScopeRef *webhookv1.WebhookScopeRef,
) (*webhookv1.PingWebhookRequest, error) {
	if flags.Name != "" || flags.ID != "" {
		if flags.SigningSecretFile != "" {
			return nil, appcmd.NewInvalidArgumentErrorf(
				"--%s cannot be set when pinging an existing webhook with --%s or --%s",
				signingSecretFileFlagName,
				nameFlagName,
				idFlagName,
			)
		}
		webhookRef, err := bufcli.NewWebhookRef(webhookScopeRef, flags.Name, flags.ID)
		if err != nil {
			return nil, appcmd.WrapInvalidArgumentError(err)
		}
		return &webhookv1.PingWebhookRequest{
			Value: &webhookv1.PingWebhookRequest_Existing_{
				Existing: &webhookv1.PingWebhookRequest_Existing{
					WebhookRef: webhookRef,
					Url:        flags.URL,
				},
			},
		}, nil
	}
	if flags.URL == nil {
		return nil, appcmd.NewInvalidArgumentErrorf(
			"one of --%s, --%s, or --%s is required",
			nameFlagName,
			idFlagName,
			urlFlagName,
		)
	}
	var signingSecret []byte
	if flags.SigningSecretFile != "" {
		var err error
		signingSecret, err = internal.ReadSigningSecretFile(container, flags.SigningSecretFile)
		if err != nil {
			return nil, err
		}
	}
	return &webhookv1.PingWebhookRequest{
		Value: &webhookv1.PingWebhookRequest_Draft_{
			Draft: &webhookv1.PingWebhookRequest_Draft{
				ScopeRef:      webhookScopeRef,
				Url:           *flags.URL,
				SigningSecret: signingSecret,
			},
		},
	}, nil
}
