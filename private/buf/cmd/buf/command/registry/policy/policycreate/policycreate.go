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

package policycreate

import (
	"context"
	"fmt"

	ownerv1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/owner/v1"
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

const (
	formatFlagName      = "format"
	visibilityFlagName  = "visibility"
	defaultLabeFlagName = "default-label-name"

	defaultDefaultLabel = "main"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <remote/owner/policy>",
		Short: "Create a BSR policy",
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
	Format       string
	Visibility   string
	DefaultLabel string
	Type         string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	bufcli.BindVisibility(flagSet, &f.Visibility, visibilityFlagName, false)
	flagSet.StringVar(
		&f.Format,
		formatFlagName,
		bufprint.FormatText.String(),
		fmt.Sprintf(`The output format to use. Must be one of %s`, bufprint.AllFormatsString),
	)
	flagSet.StringVar(
		&f.DefaultLabel,
		defaultLabeFlagName,
		defaultDefaultLabel,
		"The default label name of the module",
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
	visibility, err := bufcli.VisibilityFlagToPolicyVisibilityAllowUnspecified(flags.Visibility)
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
	policyServiceClient := bufregistryapipolicy.NewClientProvider(clientConfig).
		V1Beta1PolicyServiceClient(policyFullName.Registry())

	policyResponse, err := policyServiceClient.CreatePolicies(ctx, connect.NewRequest(
		&policyv1beta1.CreatePoliciesRequest{
			Values: []*policyv1beta1.CreatePoliciesRequest_Value{
				{
					OwnerRef: &ownerv1.OwnerRef{
						Value: &ownerv1.OwnerRef_Name{
							Name: policyFullName.Owner(),
						},
					},
					Name:       policyFullName.Name(),
					Visibility: visibility,
				},
			},
		},
	))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeAlreadyExists {
			return bufcli.NewPolicyNameAlreadyExistsError(policyFullName.String())
		}
		return err
	}
	policys := policyResponse.Msg.Policies
	if len(policys) != 1 {
		return syserror.Newf("unexpected number of policies returned from server: %d", len(policys))
	}
	if format == bufprint.FormatText {
		_, err = fmt.Fprintf(container.Stdout(), "Created %s.\n", policyFullName)
		if err != nil {
			return syserror.Wrap(err)
		}
		return nil
	}
	return bufprint.PrintNames(
		container.Stdout(),
		format,
		bufprint.NewPolicyEntity(policys[0], policyFullName),
	)
}
