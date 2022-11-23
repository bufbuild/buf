// Copyright 2020-2022 Buf Technologies, Inc.
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

package getschema

import (
	"context"
	"encoding/json"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/bufbuild/connect-go"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	elementNamesFlagName            = "elementNames"
	includeWellKnownImportsFlagName = "includeWellKnownImports"
)

// NewCommand returns a new Command
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	f := newFlags()
	return &appcmd.Command{
		Use:   name + " <remote/owner/repository>",
		Short: "Get a filtered schema from the registry",
		Args:  cobra.ExactArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appflag.Container) error {
				return run(ctx, container, f)
			},
			bufcli.NewErrorInterceptor(),
		),
		BindFlags: f.Bind,
	}
}

func run(ctx context.Context, container appflag.Container, f *flags) error {
	bufcli.WarnAlphaCommand(ctx, container)
	moduleReferenceArg := container.Arg(0)
	if moduleReferenceArg == "" {
		return appcmd.NewInvalidArgumentError("repository is required")
	}
	moduleReference, err := bufmoduleref.ModuleReferenceForString(moduleReferenceArg)
	if err != nil {
		return appcmd.NewInvalidArgumentError(err.Error())
	}
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	service := connectclient.Make(
		clientConfig,
		moduleReference.Remote(),
		registryv1alpha1connect.NewSchemaServiceClient,
	)
	schemaResponse, err := service.GetSchema(
		ctx,
		connect.NewRequest(&registryv1alpha1.GetSchemaRequest{
			Module: &registryv1alpha1.LocalModuleReference{
				Owner:      moduleReference.Owner(),
				Repository: moduleReference.Repository(),
				Reference:  moduleReference.Reference(),
			},
			ElementNames:            f.elementNames,
			IncludeWellKnownImports: f.includeWellKnownImports,
		}),
	)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return bufcli.NewRepositoryNotFoundError(container.Arg(0))
		}
		return err
	}
	return json.NewEncoder(container.Stdout()).Encode(schemaResponse.Msg)
}

type flags struct {
	elementNames            []string
	includeWellKnownImports bool
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringSliceVar(
		&f.elementNames,
		elementNamesFlagName,
		[]string{},
		"The names may be type names (messages or enums), service names, or method names. All names must be fully-qualified. If any name is unknown, the request will fail and no schema will be returned.",
	)
	flagSet.BoolVar(
		&f.includeWellKnownImports,
		includeWellKnownImportsFlagName,
		false,
		"If true, well-known imports will be included the returned set of files. If false or not present, these files will omitted from the response",
	)
}
