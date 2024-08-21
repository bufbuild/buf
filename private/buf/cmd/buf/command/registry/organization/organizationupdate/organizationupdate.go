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

package organizationupdate

import (
	"context"
	"fmt"

	ownerv1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/owner/v1"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/spf13/pflag"
)

const (
	descriptionFlagName = "description"
	urlFlagName         = "url"
)

// NewCommand returns a new Command
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <remote/organization>",
		Short: "Update a BSR organization",
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
	url         *string
	description *string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	bufcli.BindStringPointer(
		flagSet,
		descriptionFlagName,
		&f.description,
		"The new description for the organization",
	)
	bufcli.BindStringPointer(
		flagSet,
		urlFlagName,
		&f.url,
		"The new URL for the organization",
	)
}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) error {
	moduleOwner, err := bufcli.ParseModuleOwner(container.Arg(0))
	if err != nil {
		return appcmd.WrapInvalidArgumentError(err)
	}
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	clientProvider := bufapi.NewClientProvider(clientConfig)
	organizationServiceClient := clientProvider.V1OrganizationServiceClient(moduleOwner.Registry())
	if _, err := organizationServiceClient.UpdateOrganizations(
		ctx,
		connect.NewRequest(
			&ownerv1.UpdateOrganizationsRequest{
				Values: []*ownerv1.UpdateOrganizationsRequest_Value{
					{
						OrganizationRef: &ownerv1.OrganizationRef{
							Value: &ownerv1.OrganizationRef_Name{
								Name: moduleOwner.Owner(),
							},
						},
						Description: flags.description,
						Url:         flags.url,
					},
				},
			},
		),
	); err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return bufcli.NewOrganizationNotFoundError(container.Arg(0))
		}
		return err
	}
	if _, err := fmt.Fprintln(container.Stdout(), "Organization updated."); err != nil {
		return syserror.Wrap(err)
	}
	return nil
}
