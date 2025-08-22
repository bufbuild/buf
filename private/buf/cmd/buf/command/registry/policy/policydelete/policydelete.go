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

package policydelete

import (
	"context"
	"fmt"

	policyv1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/policy/v1beta1"
	"buf.build/go/app/appcmd"
	"buf.build/go/app/appext"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/bufpkg/bufregistryapi/bufregistryapipolicy"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/spf13/pflag"
)

const forceFlagName = "force"

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <remote/owner/policy>",
		Short: "Delete a BSR policy",
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
	Force bool
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
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
	policyFullName, err := bufparse.ParseFullName(container.Arg(0))
	if err != nil {
		return appcmd.WrapInvalidArgumentError(err)
	}
	if !flags.Force {
		if err := bufcli.PromptUserForDelete(container, "entity", policyFullName.Name()); err != nil {
			return err
		}
	}
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	policyServiceClient := bufregistryapipolicy.NewClientProvider(clientConfig).
		V1Beta1PolicyServiceClient(policyFullName.Registry())

	if _, err := policyServiceClient.DeletePolicies(ctx, connect.NewRequest(
		&policyv1beta1.DeletePoliciesRequest{
			PolicyRefs: []*policyv1beta1.PolicyRef{
				{
					Value: &policyv1beta1.PolicyRef_Name_{
						Name: &policyv1beta1.PolicyRef_Name{
							Owner:  policyFullName.Owner(),
							Policy: policyFullName.Name(),
						},
					},
				},
			},
		},
	)); err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return bufcli.NewPolicyNotFoundError(container.Arg(0))
		}
		return err
	}
	if _, err := fmt.Fprintf(container.Stdout(), "Deleted %s.\n", policyFullName); err != nil {
		return syserror.Wrap(err)
	}
	return nil
}
