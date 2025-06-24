// Copyright 2020-2025 Buf Technologies, Inc.
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

	"buf.build/go/app/appcmd"
	"buf.build/go/app/appext"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufprint"
	"github.com/bufbuild/buf/private/bufpkg/bufconnect"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/bufbuild/buf/private/pkg/netext"
	"github.com/spf13/pflag"
)

const (
	formatFlagName = "format"

	loginCommand = "buf registry login"
)

// NewCommand returns a new Command.
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

type flags struct {
	Format string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
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
			return fmt.Errorf("Not currently logged in for %s.", remote)
		}
		return err
	}
	user := currentUserResponse.Msg.GetUser()
	if user == nil {
		return fmt.Errorf(
			`No user is logged in to %s. Run %q to refresh your credentials. If you have the %s environment variable set, ensure that the token is valid.`,
			remote,
			loginCommandForRemote(remote),
			bufconnect.TokenEnvKey,
		)
	}
	format, err := bufprint.ParseFormat(flags.Format)
	if err != nil {
		return appcmd.WrapInvalidArgumentError(err)
	}
	// ParseFormat always expects a format that is either text or json, otherwise it returns
	// an error, so do not need a default case for this switch.
	switch format {
	case bufprint.FormatText:
		_, err = fmt.Fprintf(container.Stdout(), "Logged in as %s.\n", user.GetUsername())
		return err
	case bufprint.FormatJSON:
		return bufprint.PrintEntity(
			container.Stdout(),
			format,
			bufprint.NewUserEntity(user),
		)
	}
	return nil
}

// loginCommandForRemote returns the login command for the given remote,
// the default remote is excluded in the command.
func loginCommandForRemote(remote string) string {
	if remote == bufconnect.DefaultRemote {
		return loginCommand
	}
	return fmt.Sprintf("%s %s", loginCommand, remote)
}
