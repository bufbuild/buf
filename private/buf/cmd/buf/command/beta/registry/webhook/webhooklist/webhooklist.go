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

package webhooklist

import (
	"context"
	"encoding/json"

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
	ownerFlagName      = "owner"
	repositoryFlagName = "repository"
	remoteFlagName     = "remote"
)

// NewCommand returns a new Command
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name,
		Short: "List repository webhooks",
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
		`The owner name of the repository to list webhooks for`,
	)
	_ = appcmd.MarkFlagRequired(flagSet, ownerFlagName)
	flagSet.StringVar(
		&f.RepositoryName,
		repositoryFlagName,
		"",
		"The repository name to list webhooks for.",
	)
	_ = appcmd.MarkFlagRequired(flagSet, repositoryFlagName)
	flagSet.StringVar(
		&f.Remote,
		remoteFlagName,
		"",
		"The remote of the owner and repository to list webhooks for",
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
	resp, err := service.ListWebhooks(
		ctx,
		connect.NewRequest(
			&registryv1alpha1.ListWebhooksRequest{
				RepositoryName: flags.RepositoryName,
				OwnerName:      flags.OwnerName,
				// TODO FUTURE: this should probably be in a loop so we can get page token from
				//   response and query for the next page
			},
		),
	)
	if err != nil {
		return err
	}
	if resp.Msg.Webhooks == nil {
		// Ignore errors for writing to stdout.
		_, _ = container.Stdout().Write([]byte("[]"))
		return nil
	}
	webhooksJSON, err := json.MarshalIndent(resp.Msg.Webhooks, "", "\t")
	if err != nil {
		return err
	}
	// Ignore errors for writing to stdout.
	_, _ = container.Stdout().Write(webhooksJSON)
	return nil
}
