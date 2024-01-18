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

package registrylogin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufconnect"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/bufbuild/buf/private/pkg/netrc"
	"github.com/spf13/pflag"
)

const (
	usernameFlagName   = "username"
	tokenStdinFlagName = "token-stdin"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <domain>",
		Short: `Log in to the Buf Schema Registry`,
		Long: fmt.Sprintf(`This prompts for your BSR username and a BSR token and updates your %s file with these credentials.
The <domain> argument will default to buf.build if not specified.`, netrc.Filename),
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
	Username   string
	TokenStdin bool
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(
		&f.Username,
		usernameFlagName,
		"",
		"The username to use. This command prompts for a username by default",
	)
	flagSet.BoolVar(
		&f.TokenStdin,
		tokenStdinFlagName,
		false,
		"Read the token from stdin. This command prompts for a token by default",
	)
}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) error {
	// If a user sends a SIGINT to buf, the top-level application context is
	// cancelled and signal masks are reset. However, during an interactive
	// login the context is not respected; for example, it takes two SIGINTs
	// to interrupt the process.

	// Ideally we could just trigger an I/O timeout by setting the deadline on
	// stdin, but when stdin is connected to a terminal the underlying fd is in
	// blocking mode making it ineligible. As changing the mode of stdin is
	// dangerous, this change takes an alternate approach of simply returning
	// early.

	// Note that this does not gracefully handle the case where the terminal is
	// in no-echo mode, as is the case when prompting for a password
	// interactively.
	errC := make(chan error, 1)
	go func() {
		errC <- inner(ctx, container, flags)
		close(errC)
	}()
	select {
	case err := <-errC:
		return err
	case <-ctx.Done():
		ctxErr := ctx.Err()
		// Otherwise we will print "Failure: context canceled".
		if errors.Is(ctxErr, context.Canceled) {
			// Otherwise the next terminal line will be on the same line as the
			// last output from buf.
			if _, err := fmt.Fprintln(container.Stdout()); err != nil {
				return err
			}
			return nil
		}
		return ctxErr
	}
}

func inner(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) error {
	remote := bufconnect.DefaultRemote
	if container.NumArgs() == 1 {
		remote = container.Arg(0)
	}
	// Do not print unless we are prompting
	if flags.Username == "" && !flags.TokenStdin {
		if _, err := fmt.Fprintf(
			container.Stdout(),
			"Log in with your Buf Schema Registry username. If you don't have a username, create one at https://%s.\n\n",
			remote,
		); err != nil {
			return err
		}
	}
	username := flags.Username
	if username == "" {
		var err error
		username, err = bufcli.PromptUser(container, "Username: ")
		if err != nil {
			if errors.Is(err, bufcli.ErrNotATTY) {
				return errors.New("cannot perform an interactive login from a non-TTY device")
			}
			return err
		}
	}
	var token string
	if flags.TokenStdin {
		data, err := io.ReadAll(container.Stdin())
		if err != nil {
			return err
		}
		token = string(data)
	} else {
		var err error
		token, err = bufcli.PromptUserForPassword(container, "Token: ")
		if err != nil {
			if errors.Is(err, bufcli.ErrNotATTY) {
				return errors.New("cannot perform an interactive login from a non-TTY device")
			}
			return err
		}
	}
	// Remove leading and trailing spaces from user-supplied token to avoid
	// common input errors such as trailing new lines, as-is the case of using
	// echo vs echo -n.
	token = strings.TrimSpace(token)
	if token == "" {
		return errors.New("token cannot be empty string")
	}
	clientConfig, err := bufcli.NewConnectClientConfigWithToken(container, token)
	if err != nil {
		return err
	}
	authnService := connectclient.Make(clientConfig, remote, registryv1alpha1connect.NewAuthnServiceClient)
	resp, err := authnService.GetCurrentUser(ctx, connect.NewRequest(&registryv1alpha1.GetCurrentUserRequest{}))
	if err != nil {
		if connectErr := new(connect.Error); errors.As(err, &connectErr) && connectErr.Code() == connect.CodeUnavailable {
			return connectErr
		}
		// We don't want to use the default error from wrapError here if the error
		// an unauthenticated error.
		return errors.New("invalid token provided")
	}
	user := resp.Msg.User
	if user == nil {
		return errors.New("no user found for provided token")
	}
	if user.Username != username {
		return errors.New("the username associated with the provided token does not match the provided username")
	}
	if err := netrc.PutMachines(
		container,
		netrc.NewMachine(
			remote,
			username,
			token,
		),
	); err != nil {
		return err
	}
	if _, err := netrc.DeleteMachineForName(container, "go."+remote); err != nil {
		return err
	}
	netrcFilePath, err := netrc.GetFilePath(container)
	if err != nil {
		return err
	}
	loggedInMessage := fmt.Sprintf("Credentials saved to %s.\n", netrcFilePath)
	// Unless we did not prompt at all, print a newline first
	if flags.Username == "" || !flags.TokenStdin {
		loggedInMessage = "\n" + loggedInMessage
	}
	if _, err := container.Stdout().Write([]byte(loggedInMessage)); err != nil {
		return err
	}
	return nil
}
