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

package commitaddlabel

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
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"github.com/spf13/pflag"
)

const (
	formatFlagName = "format"
	labelsFlagName = "label"
)

// NewCommand returns a new Command
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <remote/owner/module:commit> --label <label>",
		Short: "Add labels to a commit",
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
	moduleRef, err := bufmodule.ParseModuleRef(container.Arg(0))
	if err != nil {
		return appcmd.WrapInvalidArgumentError(err)
	}
	if moduleRef.Ref() == "" {
		return appcmd.NewInvalidArgumentError("commit is required")
	}
	commitID := moduleRef.Ref()
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
	clientProvider := bufapi.NewClientProvider(clientConfig)
	labelServiceClient := clientProvider.V1LabelServiceClient(moduleRef.ModuleFullName().Registry())
	requestValues := slicesext.Map(labels, func(label string) *modulev1.CreateOrUpdateLabelsRequest_Value {
		return &modulev1.CreateOrUpdateLabelsRequest_Value{
			LabelRef: &modulev1.LabelRef{
				Value: &modulev1.LabelRef_Name_{
					Name: &modulev1.LabelRef_Name{
						Owner:  moduleRef.ModuleFullName().Owner(),
						Module: moduleRef.ModuleFullName().Name(),
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
			&modulev1.CreateOrUpdateLabelsRequest{
				Values: requestValues,
			},
		),
	)
	if err != nil {
		// Not explicitly handling error with connect.CodeNotFound as it can be repository not found or commit not found.
		// It can also be a misformatted commit ID error.
		return err
	}
	if format == bufprint.FormatText {
		for _, label := range resp.Msg.Labels {
			fmt.Fprintf(container.Stdout(), "%s:%s\n", moduleRef.ModuleFullName(), label.Name)
		}
		return nil
	}
	return bufprint.PrintNames(
		container.Stdout(),
		format,
		slicesext.Map(resp.Msg.Labels, func(label *modulev1.Label) bufprint.Entity {
			return bufprint.NewLabelEntity(label, moduleRef.ModuleFullName())
		})...,
	)
}
