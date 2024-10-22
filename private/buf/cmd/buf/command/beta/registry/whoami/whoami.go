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

package whoami

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufconnect"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/bufbuild/buf/private/pkg/netext"
	"github.com/spf13/pflag"
)

func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <domain>",
		Short: `Check if you are logged in to the Buf Schema Registry`,
		Long: `This command checks if you are currently logged into the Buf Schema Registry at the provided <domain>.
The <domain> argument will default to buf.build if not specified.`,
		Args: appcmd.MaximumNArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct{}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSEt *pflag.FlagSet) {}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) error {
	remote := bufconnect.DefaultRemote
	if container.NumArgs() == 1 {
		remote = container.Arg(0)
		if _, err := netext.ValidateHostname(remote); err != nil {
			return err
		}
	}
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	authnService := connectclient.Make(clientConfig, remote, registryv1alpha1connect.NewAuthnServiceClient)
	currentUserResponse, err := authnService.GetCurrentUser(ctx, connect.NewRequest(&registryv1alpha1.GetCurrentUserRequest{}))
	if err != nil {
		if connectErr := new(connect.Error); errors.As(err, &connectErr) && connectErr.Code() == connect.CodeUnauthenticated {
			return fmt.Errorf("Not currently logged in for %s", remote)
		}
		return err
	}
	user := currentUserResponse.Msg.User
	if user == nil {
		return errors.New("No user found for provided token")
	}
	_, err = fmt.Fprintf(container.Stdout(), "Logged in as %s.\n", user.Username)
	return err
}
