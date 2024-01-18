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

package tagcreate

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufprint"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const formatFlagName = "format"

// NewCommand returns a new Command
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <buf.build/owner/repository:commit> <tag>",
		Short: "Create a tag for a specified commit",
		Args:  cobra.ExactArgs(2),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appflag.Container) error {
				return run(ctx, container, flags)
			},
			bufcli.NewErrorInterceptor(),
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
	container appflag.Container,
	flags *flags,
) error {
	bufcli.WarnBetaCommand(ctx, container)
	moduleReference, err := bufmoduleref.ModuleReferenceForString(
		container.Arg(0),
	)
	if err != nil {
		return appcmd.NewInvalidArgumentError(err.Error())
	}
	if !bufmoduleref.IsCommitModuleReference(moduleReference) {
		return fmt.Errorf("commit is required, but a tag was given: %q", container.Arg(0))
	}
	format, err := bufprint.ParseFormat(flags.Format)
	if err != nil {
		return appcmd.NewInvalidArgumentError(err.Error())
	}

	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	repositoryService := connectclient.Make(
		clientConfig,
		moduleReference.Remote(),
		registryv1alpha1connect.NewRepositoryServiceClient,
	)
	repositoryTagService := connectclient.Make(
		clientConfig,
		moduleReference.Remote(),
		registryv1alpha1connect.NewRepositoryTagServiceClient,
	)
	resp, err := repositoryService.GetRepositoryByFullName(ctx,
		connect.NewRequest(&registryv1alpha1.GetRepositoryByFullNameRequest{
			FullName: moduleReference.Owner() + "/" + moduleReference.Repository(),
		}),
	)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return bufcli.NewRepositoryNotFoundError(moduleReference.Remote() + "/" + moduleReference.Owner() + "/" + moduleReference.Repository())
		}
		return err
	}
	tag := container.Arg(1)
	commit := moduleReference.Reference()
	tagResp, err := repositoryTagService.CreateRepositoryTag(
		ctx,
		connect.NewRequest(&registryv1alpha1.CreateRepositoryTagRequest{
			RepositoryId: resp.Msg.Repository.Id,
			Name:         tag,
			CommitName:   commit,
		}),
	)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeAlreadyExists {
			return bufcli.NewTagOrDraftNameAlreadyExistsError(tag)
		}
		if connect.CodeOf(err) == connect.CodeNotFound {
			return bufcli.NewModuleReferenceNotFoundError(moduleReference)
		}
		return err
	}
	return bufprint.NewRepositoryTagPrinter(container.Stdout()).PrintRepositoryTag(ctx, format, tagResp.Msg.RepositoryTag)
}
