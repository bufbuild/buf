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

package labelget

import (
	"context"
	"fmt"

	modulev1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufprint"
	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/spf13/pflag"
)

const formatFlagName = "format"

// NewCommand returns a new Command
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <buf.build/owner/repository:label>",
		Short: "Get label details",
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
	bufcli.WarnBetaCommand(ctx, container)
	moduleRef, err := bufmodule.ParseModuleRef(container.Arg(0))
	if err != nil {
		return appcmd.NewInvalidArgumentError(err.Error())
	}
	format, err := bufprint.ParseFormat(flags.Format)
	if err != nil {
		return appcmd.NewInvalidArgumentError(err.Error())
	}
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	clientProvider := bufapi.NewClientProvider(clientConfig)
	labelServiceClient := clientProvider.V1LabelServiceClient(moduleRef.ModuleFullName().Registry())
	resp, err := labelServiceClient.GetLabels(
		ctx,
		connect.NewRequest(
			&modulev1.GetLabelsRequest{
				LabelRefs: []*modulev1.LabelRef{
					{
						Value: &modulev1.LabelRef_Name_{
							Name: &modulev1.LabelRef_Name{
								Owner:  moduleRef.ModuleFullName().Owner(),
								Module: moduleRef.ModuleFullName().Name(),
								Label:  moduleRef.Ref(),
							},
						},
					},
				},
			},
		),
	)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return bufcli.NewLabelNotFoundError(moduleRef)
		}
		return err
	}
	labels := resp.Msg.Labels
	if len(labels) != 1 {
		return syserror.Newf("expect 1 label from response, got %d", len(labels))
	}
	return bufprint.NewRepositoryLabelPrinter(container.Stdout()).PrintRepositoryLabel(ctx, format, labels[0])
}
