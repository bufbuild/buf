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

package policyinfo

import (
	"context"
	"fmt"

	policyv1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/policy/v1beta1"
	"buf.build/go/app/appcmd"
	"buf.build/go/app/appext"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufprint"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/bufpkg/bufregistryapi/bufregistryapipolicy"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/spf13/pflag"
)

const formatFlagName = "format"

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <remote/owner/policy>",
		Short: "Get a BSR policy",
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
	policyFullName, err := bufparse.ParseFullName(container.Arg(0))
	if err != nil {
		return appcmd.WrapInvalidArgumentError(err)
	}
	format, err := bufprint.ParseFormat(flags.Format)
	if err != nil {
		return appcmd.WrapInvalidArgumentError(err)
	}

	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	policieserviceClient := bufregistryapipolicy.NewClientProvider(clientConfig).
		V1Beta1PolicyServiceClient(policyFullName.Registry())

	policiesResponse, err := policieserviceClient.GetPolicies(ctx, connect.NewRequest(
		&policyv1beta1.GetPoliciesRequest{
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
	))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return bufcli.NewPolicyNotFoundError(container.Arg(0))
		}
		return err
	}
	policies := policiesResponse.Msg.Policies
	if len(policies) != 1 {
		return syserror.Newf("unexpected number of policies returned from server: %d", len(policies))
	}
	return bufprint.PrintEntity(
		container.Stdout(),
		format,
		bufprint.NewPolicyEntity(policies[0], policyFullName),
	)
}
