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

package plugincommitaddlabel

import (
	"context"
	"fmt"

	pluginv1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/plugin/v1beta1"
	"buf.build/go/app/appcmd"
	"buf.build/go/app/appext"
	"buf.build/go/standard/xslices"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufprint"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/bufpkg/bufregistryapi/bufregistryapiplugin"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"github.com/spf13/pflag"
)

const (
	formatFlagName = "format"
	labelsFlagName = "label"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
	deprecated string,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:        name + " <remote/owner/plugin:commit> --label <label>",
		Short:      "Add labels to a commit",
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
	Labels []string
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
	flagSet.StringSliceVar(
		&f.Labels,
		labelsFlagName,
		nil,
		"The labels to add to the commit. Must have at least one",
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
	if pluginRef.Ref() == "" {
		return appcmd.NewInvalidArgumentError("commit is required")
	}
	commitID := pluginRef.Ref()
	if _, err := uuidutil.FromDashless(commitID); err != nil {
		return appcmd.NewInvalidArgumentErrorf("invalid commit: %w", err)
	}
	labels := flags.Labels
	if len(labels) == 0 {
		return appcmd.NewInvalidArgumentError("must create at least one label")
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
	labelServiceClient := pluginClientProvider.V1Beta1LabelServiceClient(pluginRef.FullName().Registry())
	requestValues := xslices.Map(labels, func(label string) *pluginv1beta1.CreateOrUpdateLabelsRequest_Value {
		return &pluginv1beta1.CreateOrUpdateLabelsRequest_Value{
			LabelRef: &pluginv1beta1.LabelRef{
				Value: &pluginv1beta1.LabelRef_Name_{
					Name: &pluginv1beta1.LabelRef_Name{
						Owner:  pluginRef.FullName().Owner(),
						Plugin: pluginRef.FullName().Name(),
						Label:  label,
					},
				},
			},
			CommitId: commitID,
		}
	})
	resp, err := labelServiceClient.CreateOrUpdateLabels(
		ctx,
		connect.NewRequest(
			&pluginv1beta1.CreateOrUpdateLabelsRequest{
				Values: requestValues,
			},
		),
	)
	if err != nil {
		// Not explicitly handling error with connect.CodeNotFound as
		// it can be Plugin or Commit not found error. May be caused by
		// a misformatted ID.
		return err
	}
	return bufprint.PrintNames(
		container.Stdout(),
		format,
		xslices.Map(resp.Msg.Labels, func(label *pluginv1beta1.Label) bufprint.Entity {
			return bufprint.NewLabelEntity(label, pluginRef.FullName())
		})...,
	)
}
