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

package policylabelarchive

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
	"github.com/spf13/pflag"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
	deprecated string,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:        name + " <remote/owner/policy:label>",
		Short:      "Archive a policy label",
		Args:       appcmd.ExactArgs(1),
		Deprecated: deprecated,
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
}

func run(
	ctx context.Context,
	container appext.Container,
	_ *flags,
) error {
	policyRef, err := bufparse.ParseRef(container.Arg(0))
	if err != nil {
		return appcmd.WrapInvalidArgumentError(err)
	}
	labelName := policyRef.Ref()
	if labelName == "" {
		return appcmd.NewInvalidArgumentError("label is required")
	}
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	policyFullName := policyRef.FullName()
	labelServiceClient := bufregistryapipolicy.NewClientProvider(clientConfig).V1Beta1LabelServiceClient(policyFullName.Registry())
	// ArchiveLabelsResponse is empty.
	if _, err := labelServiceClient.ArchiveLabels(
		ctx,
		connect.NewRequest(
			&policyv1beta1.ArchiveLabelsRequest{
				LabelRefs: []*policyv1beta1.LabelRef{
					{
						Value: &policyv1beta1.LabelRef_Name_{
							Name: &policyv1beta1.LabelRef_Name{
								Owner:  policyFullName.Owner(),
								Policy: policyFullName.Name(),
								Label:  labelName,
							},
						},
					},
				},
			},
		),
	); err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return bufcli.NewLabelNotFoundError(policyRef)
		}
		return err
	}
	_, err = fmt.Fprintf(container.Stdout(), "Archived %s.\n", policyRef)
	return err
}
