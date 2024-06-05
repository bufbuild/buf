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

package archive

import (
	"context"

	v1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/buf/cmd/buf/command/beta/registry/internal"
	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/spf13/pflag"
)

const (
	labelFlagName = "label"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <input> --label <label>",
		Short: "Archive one or more labels for the given input",
		Args:  appcmd.MaximumNArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	Labels []string
	// special
	InputHashtag string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	bufcli.BindInputHashtag(flagSet, &f.InputHashtag)
	flagSet.StringSliceVar(
		&f.Labels,
		labelFlagName,
		nil,
		"Label(s) to archive. Must have at least one.",
	)
}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) error {
	if len(flags.Labels) < 1 {
		return appcmd.NewInvalidArgumentError("must archive at least one label")
	}
	controller, err := bufcli.NewController(container)
	if err != nil {
		return err
	}
	source, err := bufcli.GetInputValue(container, flags.InputHashtag, ".")
	if err != nil {
		return err
	}
	workspace, err := controller.GetWorkspace(
		ctx,
		source,
		// If we do not set this configuration, we would allow users to run `buf beta registry archive`
		// against a v1 workspace. This is somewhat safe, since archiving labels does not need
		// to happen in any special order, however, it would create an inconsistency with
		// `buf push`, where we do have that constraint.
		bufctl.WithIgnoreAndDisallowV1BufWorkYAMLs(),
	)
	if err != nil {
		return err
	}
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	registryToModuleFullNames := map[string][]bufmodule.ModuleFullName{}
	for _, module := range workspace.Modules() {
		if !module.IsTarget() {
			continue
		}
		if moduleFullName := module.ModuleFullName(); moduleFullName != nil {
			moduleFullNames, ok := registryToModuleFullNames[moduleFullName.Registry()]
			if !ok {
				registryToModuleFullNames[moduleFullName.Registry()] = []bufmodule.ModuleFullName{moduleFullName}
				continue
			}
			registryToModuleFullNames[moduleFullName.Registry()] = append(moduleFullNames, moduleFullName)
		}
	}
	for registry, moduleFullNames := range registryToModuleFullNames {
		labelServiceClient := bufapi.NewClientProvider(clientConfig).V1LabelServiceClient(registry)
		// ArchiveLabelsResponse is empty.
		if _, err := labelServiceClient.ArchiveLabels(
			ctx,
			connect.NewRequest(
				&v1.ArchiveLabelsRequest{
					LabelRefs: internal.GetLabelRefsForModuleFullNamesAndLabels(moduleFullNames, flags.Labels),
				},
			),
		); err != nil {
			return err
		}
	}
	return nil
}
