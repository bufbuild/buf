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

package pluginlabelinfo

import (
	"context"
	"fmt"

	pluginv1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/plugin/v1beta1"
	"buf.build/go/app/appcmd"
	"buf.build/go/app/appext"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufprint"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/bufpkg/bufregistryapi/bufregistryapiplugin"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/spf13/pflag"
)

const formatFlagName = "format"

// NewCommand returns a new Command
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
	deprecated string,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:        name + " <remote/owner/plugin:label>",
		Short:      "Show label information",
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
	pluginRef, err := bufparse.ParseRef(container.Arg(0))
	if err != nil {
		return appcmd.WrapInvalidArgumentError(err)
	}
	labelName := pluginRef.Ref()
	if labelName == "" {
		return appcmd.NewInvalidArgumentError("label is required")
	}
	format, err := bufprint.ParseFormat(flags.Format)
	if err != nil {
		return appcmd.WrapInvalidArgumentError(err)
	}
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	pluginClientProvider := bufregistryapiplugin.NewClientProvider(clientConfig)
	pluginFullName := pluginRef.FullName()
	labelServiceClient := pluginClientProvider.V1Beta1LabelServiceClient(pluginFullName.Registry())
	resp, err := labelServiceClient.GetLabels(
		ctx,
		connect.NewRequest(
			&pluginv1beta1.GetLabelsRequest{
				LabelRefs: []*pluginv1beta1.LabelRef{
					{
						Value: &pluginv1beta1.LabelRef_Name_{
							Name: &pluginv1beta1.LabelRef_Name{
								Owner:  pluginFullName.Owner(),
								Plugin: pluginFullName.Name(),
								Label:  labelName,
							},
						},
					},
				},
			},
		),
	)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return bufcli.NewLabelNotFoundError(pluginRef)
		}
		return err
	}
	labels := resp.Msg.Labels
	if len(labels) != 1 {
		return syserror.Newf("expect 1 label from response, got %d", len(labels))
	}
	return bufprint.PrintEntity(
		container.Stdout(),
		format,
		bufprint.NewLabelEntity(labels[0], pluginFullName),
	)
}
