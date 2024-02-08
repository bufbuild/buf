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

package push

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/bufpkg/bufcas/bufcasalpha"
	"github.com/bufbuild/buf/private/bufpkg/buflock"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	tagFlagName              = "tag"
	tagFlagShortName         = "t"
	draftFlagName            = "draft"
	branchFlagName           = "branch"
	errorFormatFlagName      = "error-format"
	disableSymlinksFlagName  = "disable-symlinks"
	createFlagName           = "create"
	createVisibilityFlagName = "create-visibility"
	// deprecated
	trackFlagName = "track"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <source>",
		Short: "Push a module to a registry",
		Long:  bufcli.GetSourceLong(`the source to push`),
		Args:  cobra.MaximumNArgs(1),
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
	Tags             []string
	Branch           string
	Draft            string
	ErrorFormat      string
	DisableSymlinks  bool
	Create           bool
	CreateVisibility string
	// Deprecated
	Tracks []string
	// special
	InputHashtag string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	bufcli.BindInputHashtag(flagSet, &f.InputHashtag)
	bufcli.BindDisableSymlinks(flagSet, &f.DisableSymlinks, disableSymlinksFlagName)
	bufcli.BindCreateVisibility(flagSet, &f.CreateVisibility, createVisibilityFlagName, createFlagName)
	flagSet.StringSliceVarP(
		&f.Tags,
		tagFlagName,
		tagFlagShortName,
		nil,
		fmt.Sprintf(
			"Create a tag for the pushed commit. Multiple tags are created if specified multiple times. Cannot be used together with --%s",
			draftFlagName,
		),
	)
	flagSet.StringVar(
		&f.Draft,
		draftFlagName,
		"",
		fmt.Sprintf(
			"Make the pushed commit a draft with the specified name. Cannot be used together with --%s (-%s) or --%s",
			tagFlagName,
			tagFlagShortName,
			branchFlagName,
		),
	)
	flagSet.StringVar(
		&f.Branch,
		branchFlagName,
		"",
		fmt.Sprintf(
			"Push a commit to a branch with the specified name. Cannot be used together with --%s (-%s) or --%s",
			tagFlagName,
			tagFlagShortName,
			draftFlagName,
		),
	)
	flagSet.StringVar(
		&f.ErrorFormat,
		errorFormatFlagName,
		"text",
		fmt.Sprintf(
			"The format for build errors printed to stderr. Must be one of %s",
			stringutil.SliceToString(bufanalysis.AllFormatStrings),
		),
	)
	flagSet.BoolVar(
		&f.Create,
		createFlagName,
		false,
		fmt.Sprintf("Create the repository if it does not exist. Must set a visibility using --%s", createVisibilityFlagName),
	)
	flagSet.StringSliceVar(
		&f.Tracks,
		trackFlagName,
		nil,
		"Do not use. This flag never had any effect",
	)
	_ = flagSet.MarkHidden(trackFlagName)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) (retErr error) {
	if len(flags.Tracks) > 0 {
		return appcmd.NewInvalidArgumentErrorf("--%s has never had any effect, do not use.", trackFlagName)
	}
	if err := bufcli.ValidateErrorFormatFlag(flags.ErrorFormat, errorFormatFlagName); err != nil {
		return err
	}
	if flags.Draft != "" && flags.Branch != "" {
		return appcmd.NewInvalidArgumentErrorf("--%s and --%s cannot be used together.", draftFlagName, branchFlagName)
	}
	if len(flags.Tags) > 0 && flags.Draft != "" {
		return appcmd.NewInvalidArgumentErrorf("--%s (-%s) and --%s cannot be used together.", tagFlagName, tagFlagShortName, draftFlagName)
	}
	// For now, we are restricting branch behavior to be exactly the same as drafts, and thus
	// cannot be used in conjunction with tags.
	if len(flags.Tags) > 0 && flags.Branch != "" {
		return appcmd.NewInvalidArgumentErrorf("--%s (-%s) and --%s cannot be used together.", tagFlagName, tagFlagShortName, branchFlagName)
	}
	if flags.CreateVisibility != "" {
		if !flags.Create {
			return appcmd.NewInvalidArgumentErrorf("Cannot set --%s without --%s.", createVisibilityFlagName, createFlagName)
		}
		// We re-parse below as needed, but do not return an appcmd.NewInvalidArgumentError below as
		// we expect validation to be handled here.
		if _, err := bufcli.VisibilityFlagToVisibility(flags.CreateVisibility); err != nil {
			return appcmd.NewInvalidArgumentError(err.Error())
		}
	} else if flags.Create {
		return appcmd.NewInvalidArgumentErrorf("--%s is required if --%s is set.", createVisibilityFlagName, createFlagName)
	}
	source, err := bufcli.GetInputValue(container, flags.InputHashtag, ".")
	if err != nil {
		return err
	}
	storageosProvider := bufcli.NewStorageosProvider(flags.DisableSymlinks)
	runner := command.NewRunner()
	// We are pushing to the BSR, this module has to be independently buildable
	// given the configuration it has without any enclosing workspace.
	sourceBucket, sourceConfig, err := bufcli.BucketAndConfigForSource(
		ctx,
		container.Logger(),
		container,
		storageosProvider,
		runner,
		source,
	)
	if err != nil {
		return err
	}
	if err := buflock.CheckDeprecatedDigests(ctx, container.Logger(), sourceBucket); err != nil {
		return err
	}
	moduleIdentity := sourceConfig.ModuleIdentity
	builtModule, err := bufmodulebuild.NewModuleBucketBuilder().BuildForBucket(
		ctx,
		sourceBucket,
		sourceConfig.Build,
	)
	if err != nil {
		return err
	}
	modulePin, err := pushOrCreate(ctx, container, moduleIdentity, builtModule, flags)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeAlreadyExists {
			if _, err := container.Stderr().Write(
				[]byte(fmt.Sprintf("%s\n", err.Error())),
			); err != nil {
				return err
			}
			return nil
		}
		return err
	}
	if modulePin == nil {
		return errors.New("Missing local module pin in the registry's response.")
	}
	if _, err := container.Stdout().Write([]byte(modulePin.Commit + "\n")); err != nil {
		return err
	}
	return nil
}

func pushOrCreate(
	ctx context.Context,
	container appflag.Container,
	moduleIdentity bufmoduleref.ModuleIdentity,
	builtModule *bufmodulebuild.BuiltModule,
	flags *flags,
) (*registryv1alpha1.LocalModulePin, error) {
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return nil, err
	}
	modulePin, err := push(ctx, container, clientConfig, moduleIdentity, builtModule, flags)
	if err != nil {
		// We rely on Push* returning a NotFound error to denote the repository is not created.
		// This technically could be a NotFound error for some other entity than the repository
		// in question, however if it is, then this Create call will just fail as the repository
		// is already created, and there is no side effect. The 99% case is that a NotFound
		// error is because the repository does not exist, and we want to avoid having to do
		// a GetRepository RPC call for every call to push --create.
		if flags.Create && connect.CodeOf(err) == connect.CodeNotFound {
			if err := create(ctx, container, clientConfig, moduleIdentity, flags); err != nil {
				return nil, err
			}
			return push(ctx, container, clientConfig, moduleIdentity, builtModule, flags)
		}
		return nil, err
	}
	return modulePin, nil
}

func push(
	ctx context.Context,
	container appflag.Container,
	clientConfig *connectclient.Config,
	moduleIdentity bufmoduleref.ModuleIdentity,
	builtModule *bufmodulebuild.BuiltModule,
	flags *flags,
) (*registryv1alpha1.LocalModulePin, error) {
	service := connectclient.Make(clientConfig, moduleIdentity.Remote(), registryv1alpha1connect.NewPushServiceClient)
	fileSet, err := bufcas.NewFileSetForBucket(ctx, builtModule.Bucket)
	if err != nil {
		return nil, err
	}
	protoManifestBlob, protoBlobs, err := bufcasalpha.FileSetToAlphaManifestBlobAndBlobs(fileSet)
	if err != nil {
		return nil, err
	}
	draftOrBranchName := flags.Draft
	if draftOrBranchName == "" {
		// If draft is not set, then we we set the draft name to branch.
		draftOrBranchName = flags.Branch
	}
	resp, err := service.PushManifestAndBlobs(
		ctx,
		connect.NewRequest(&registryv1alpha1.PushManifestAndBlobsRequest{
			Owner:      moduleIdentity.Owner(),
			Repository: moduleIdentity.Repository(),
			Manifest:   protoManifestBlob,
			Blobs:      protoBlobs,
			Tags:       flags.Tags,
			DraftName:  draftOrBranchName,
		}),
	)
	if err != nil {
		return nil, err
	}
	return resp.Msg.LocalModulePin, nil
}

func create(
	ctx context.Context,
	container appflag.Container,
	clientConfig *connectclient.Config,
	moduleIdentity bufmoduleref.ModuleIdentity,
	flags *flags,
) error {
	service := connectclient.Make(clientConfig, moduleIdentity.Remote(), registryv1alpha1connect.NewRepositoryServiceClient)
	visiblity, err := bufcli.VisibilityFlagToVisibility(flags.CreateVisibility)
	if err != nil {
		return err
	}
	fullName := moduleIdentity.Owner() + "/" + moduleIdentity.Repository()
	_, err = service.CreateRepositoryByFullName(
		ctx,
		connect.NewRequest(&registryv1alpha1.CreateRepositoryByFullNameRequest{
			FullName:   fullName,
			Visibility: visiblity,
		}),
	)
	if err != nil && connect.CodeOf(err) == connect.CodeAlreadyExists {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("Expected repository %s to be missing but found the repository to already exist", fullName))
	}
	return err
}
