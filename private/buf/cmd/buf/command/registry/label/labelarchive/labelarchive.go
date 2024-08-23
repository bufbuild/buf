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

package labelarchive

import (
	"context"
	"fmt"

	modulev1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/spf13/pflag"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <remote/owner/module:label>",
		Short: "Archive a module label",
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
	moduleRef, err := bufmodule.ParseModuleRef(container.Arg(0))
	if err != nil {
		return appcmd.WrapInvalidArgumentError(err)
	}
	labelName := moduleRef.Ref()
	if labelName == "" {
		return appcmd.NewInvalidArgumentError("label is required")
	}
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	moduleFullName := moduleRef.ModuleFullName()
	labelServiceClient := bufapi.NewClientProvider(clientConfig).V1LabelServiceClient(moduleFullName.Registry())
	// ArchiveLabelsResponse is empty.
	if _, err := labelServiceClient.ArchiveLabels(
		ctx,
		connect.NewRequest(
			&modulev1.ArchiveLabelsRequest{
				LabelRefs: []*modulev1.LabelRef{
					{
						Value: &modulev1.LabelRef_Name_{
							Name: &modulev1.LabelRef_Name{
								Owner:  moduleFullName.Owner(),
								Module: moduleFullName.Name(),
								Label:  labelName,
							},
						},
					},
				},
			},
		),
	); err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return bufcli.NewLabelNotFoundError(moduleRef)
		}
		return err
	}
	_, err = fmt.Fprintf(container.Stdout(), "Archived %s.\n", moduleRef)
	return err
}
