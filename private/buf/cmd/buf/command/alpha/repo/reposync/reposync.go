// Copyright 2020-2023 Buf Technologies, Inc.
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

package reposync

import (
	"context"
	"fmt"
	"strings"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufsync"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufmanifest"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/manifest"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagegit"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/connect-go"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	errorFormatFlagName      = "error-format"
	moduleFlagName           = "module"
	createFlagName           = "create"
	createVisibilityFlagName = "create-visibility"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name,
		Short: "Sync a Git repository to a registry",
		Long: "Sync a Git repository's commits to a registry in topological order. " +
			"Only commits belonging to the 'origin' remote are processed, which means that " +
			"commits must be pushed to a remote. " +
			"Only modules specified via '--module' are synced.",
		Args: cobra.NoArgs,
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
	ErrorFormat      string
	Modules          []string
	Create           bool
	CreateVisibility string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(
		&f.ErrorFormat,
		errorFormatFlagName,
		"text",
		fmt.Sprintf(
			"The format for build errors printed to stderr. Must be one of %s",
			stringutil.SliceToString(bufanalysis.AllFormatStrings),
		),
	)
	flagSet.StringArrayVar(
		&f.Modules,
		moduleFlagName,
		nil,
		"The module(s) to sync to the BSR; you can provide a module override in the format "+
			"<module>:<module-identity>",
	)
	bufcli.BindCreateVisibility(flagSet, &f.CreateVisibility, createVisibilityFlagName, createFlagName)
	flagSet.BoolVar(
		&f.Create,
		createFlagName,
		false,
		fmt.Sprintf("Create the repository if it does not exist. Must set a visibility using --%s", createVisibilityFlagName),
	)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) (retErr error) {
	if err := bufcli.ValidateErrorFormatFlag(flags.ErrorFormat, errorFormatFlagName); err != nil {
		return err
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
	return sync(
		ctx,
		container,
		flags.Modules,
		// No need to pass `flags.Create`, this is not empty iff `flags.Create`
		flags.CreateVisibility,
	)
}

func sync(
	ctx context.Context,
	container appflag.Container,
	modules []string,
	createWithVisibility string,
) error {
	// Assume that this command is run from the repository root. If not, `OpenRepository` will return
	// a dir not found error.
	repo, err := git.OpenRepository(git.DotGitDir, command.NewRunner())
	if err != nil {
		return err
	}
	defer repo.Close()
	storageProvider := storagegit.NewProvider(
		repo.Objects(),
		storagegit.ProviderWithSymlinks(),
	)
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return fmt.Errorf("create connect client %w", err)
	}
	var syncerOptions []bufsync.SyncerOption
	for _, module := range modules {
		var moduleIdentityOverride bufmoduleref.ModuleIdentity
		if colon := strings.IndexRune(module, ':'); colon != -1 {
			moduleIdentityOverride, err = bufmoduleref.ModuleIdentityForString(module[colon+1:])
			if err != nil {
				return err
			}
			module = normalpath.Normalize(module[:colon])
		}
		syncerOptions = append(syncerOptions, bufsync.SyncerWithModule(module, moduleIdentityOverride))
	}
	syncer, err := bufsync.NewSyncer(
		container.Logger(),
		repo,
		storageProvider,
		syncerOptions...,
	)
	if err != nil {
		return err
	}
	return syncer.Sync(ctx, func(ctx context.Context, commit bufsync.ModuleCommit) error {
		pin, err := pushOrCreate(
			ctx,
			clientConfig,
			repo,
			commit.Commit(),
			commit.Branch(),
			commit.Tags(),
			commit.Identity(),
			commit.Bucket(),
			createWithVisibility,
		)
		if err != nil {
			if connect.CodeOf(err) == connect.CodeAlreadyExists {
				// Module has identical content. The BSR has already created the relevant labels
				// for us, so we can simply carry on.
				return nil
			}
			// We failed to push. We fail hard on this because the error may be recoverable
			// (i.e., the BSR may be down) and we should re-attempt this commit.
			return fmt.Errorf(
				"failed to push %s at %s: %w",
				commit.Identity().IdentityString(),
				commit.Commit().Hash(),
				err,
			)
		}
		_, err = container.Stderr().Write([]byte(
			fmt.Sprintf("%s:%s\n", commit.Identity().IdentityString(), pin.Commit)),
		)
		return err
	})
}

func pushOrCreate(
	ctx context.Context,
	clientConfig *connectclient.Config,
	repo git.Repository,
	commit git.Commit,
	branch string,
	tags []string,
	moduleIdentity bufmoduleref.ModuleIdentity,
	moduleBucket storage.ReadBucket,
	createWithVisibility string,
) (*registryv1alpha1.LocalModulePin, error) {
	modulePin, err := push(
		ctx,
		clientConfig,
		repo,
		commit,
		branch,
		tags,
		moduleIdentity,
		moduleBucket,
	)
	if err != nil {
		// We rely on Push* returning a NotFound error to denote the repository is not created.
		// This technically could be a NotFound error for some other entity than the repository
		// in question, however if it is, then this Create call will just fail as the repository
		// is already created, and there is no side effect. The 99% case is that a NotFound
		// error is because the repository does not exist, and we want to avoid having to do
		// a GetRepository RPC call for every call to push --create.
		if createWithVisibility != "" && connect.CodeOf(err) == connect.CodeNotFound {
			if err := create(ctx, clientConfig, moduleIdentity, createWithVisibility); err != nil {
				return nil, err
			}
			return push(
				ctx,
				clientConfig,
				repo,
				commit,
				branch,
				tags,
				moduleIdentity,
				moduleBucket,
			)
		}
		return nil, err
	}
	return modulePin, nil
}

func push(
	ctx context.Context,
	clientConfig *connectclient.Config,
	repo git.Repository,
	commit git.Commit,
	branch string,
	tags []string,
	moduleIdentity bufmoduleref.ModuleIdentity,
	moduleBucket storage.ReadBucket,
) (*registryv1alpha1.LocalModulePin, error) {
	service := connectclient.Make(clientConfig, moduleIdentity.Remote(), registryv1alpha1connect.NewPushServiceClient)
	m, blobSet, err := manifest.NewFromBucket(ctx, moduleBucket)
	if err != nil {
		return nil, err
	}
	bucketManifest, blobs, err := bufmanifest.ToProtoManifestAndBlobs(ctx, m, blobSet)
	if err != nil {
		return nil, err
	}
	if repo.BaseBranch() == branch {
		// We are pushing a commit on the base branch of this repository.
		// The BSR represents the base track as "main", and this is not configurable
		// per module.
		branch = bufmoduleref.Main
	}
	resp, err := service.PushManifestAndBlobs(
		ctx,
		connect.NewRequest(&registryv1alpha1.PushManifestAndBlobsRequest{
			Owner:      moduleIdentity.Owner(),
			Repository: moduleIdentity.Repository(),
			Manifest:   bucketManifest,
			Blobs:      blobs,
			GitMetadata: &registryv1alpha1.GitCommitMetadata{
				Hash: commit.Hash().Hex(),
				Branches: []string{
					branch,
				},
				Tags: tags,
				Author: &registryv1alpha1.GitIdentity{
					Name:  commit.Author().Name(),
					Email: commit.Author().Email(),
					Time:  timestamppb.New(commit.Author().Timestamp()),
				},
				Commiter: &registryv1alpha1.GitIdentity{
					Name:  commit.Committer().Name(),
					Email: commit.Committer().Email(),
					Time:  timestamppb.New(commit.Committer().Timestamp()),
				},
			},
		}),
	)
	if err != nil {
		return nil, err
	}
	return resp.Msg.LocalModulePin, nil
}

func create(
	ctx context.Context,
	clientConfig *connectclient.Config,
	moduleIdentity bufmoduleref.ModuleIdentity,
	visibility string,
) error {
	service := connectclient.Make(clientConfig, moduleIdentity.Remote(), registryv1alpha1connect.NewRepositoryServiceClient)
	visiblity, err := bufcli.VisibilityFlagToVisibility(visibility)
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
		return connect.NewError(connect.CodeInternal, fmt.Errorf("expected repository %s to be missing but found the repository to already exist", fullName))
	}
	return err
}
