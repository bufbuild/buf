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

package reposync

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufsync"
	"github.com/bufbuild/buf/private/buf/bufsync/bufsyncapi"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage/storagegit"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	errorFormatFlagName      = "error-format"
	moduleFlagName           = "module"
	createFlagName           = "create"
	createVisibilityFlagName = "create-visibility"
	allBranchesFlagName      = "all-branches"
	remoteFlagName           = "remote"
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
		Long: "Sync commits in a Git repository to a registry in topological order. " +
			"Local commits in the default and current branch are processed. " +
			fmt.Sprintf("Syncing only commits pushed to a specific remote is possible using --%s flag. ", remoteFlagName) +
			fmt.Sprintf("Syncing all branches is possible using --%s flag. ", allBranchesFlagName) +
			"By default a single module at the root of the repository is assumed, " +
			fmt.Sprintf("for specific module paths use the --%s flag. ", moduleFlagName) +
			"This command needs to be run at the root of the Git repository.",
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
	AllBranches      bool
	Remote           string
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
	flagSet.StringSliceVar(
		&f.Modules,
		moduleFlagName,
		nil,
		"The module(s) to sync to the BSR. This value can be just the module directory, or you can "+
			"also have define a module identity override in the format <module-directory>:<module-name>, "+
			"where the <module-name> is the module's fully qualified name (FQN) destination as defined in "+
			"https://buf.build/docs/bsr/module/manage/#how-modules-are-defined. If a module identity "+
			"override is not passed, the sync destination of the remote module is read from the 'name' "+
			"field in your 'buf.yaml' at the HEAD commit of each branch. By default this command attempts "+
			"to sync a single module located at the root directory of the Git repository.",
	)
	bufcli.BindCreateVisibility(flagSet, &f.CreateVisibility, createVisibilityFlagName, createFlagName)
	flagSet.BoolVar(
		&f.Create,
		createFlagName,
		false,
		fmt.Sprintf("Create the BSR repository if it does not exist. Must set a visibility using --%s", createVisibilityFlagName),
	)
	flagSet.BoolVar(
		&f.AllBranches,
		allBranchesFlagName,
		false,
		"Sync all Git branches and not only the default and checked out one. "+
			"Order of sync for git branches is as follows: First, it syncs the default branch (read "+
			"from 'refs/remotes/origin/HEAD'), and then all the rest of the branches in "+
			"lexicographical order. "+
			fmt.Sprintf("You can use --%s to only consider remote branches.", remoteFlagName),
	)
	flagSet.StringVar(
		&f.Remote,
		remoteFlagName,
		"",
		"The name of the Git remote to sync. If this flag is passed, only commits pushed to this remote are processed.",
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
	var createWithVisibility *registryv1alpha1.Visibility
	if flags.CreateVisibility != "" {
		if !flags.Create {
			return appcmd.NewInvalidArgumentErrorf("Cannot set --%s without --%s.", createVisibilityFlagName, createFlagName)
		}
		// We re-parse below as needed, but do not return an appcmd.NewInvalidArgumentError below as
		// we expect validation to be handled here.
		if parsed, err := bufcli.VisibilityFlagToVisibility(flags.CreateVisibility); err != nil {
			return appcmd.NewInvalidArgumentError(err.Error())
		} else {
			createWithVisibility = &parsed
		}
	} else if flags.Create {
		return appcmd.NewInvalidArgumentErrorf("--%s is required if --%s is set.", createVisibilityFlagName, createFlagName)
	}
	return sync(
		ctx,
		container,
		flags.Modules,
		// No need to pass `flags.Create`, this is not empty iff `flags.Create`
		createWithVisibility,
		flags.AllBranches,
		flags.Remote,
	)
}

func sync(
	ctx context.Context,
	container appflag.Container,
	modules []string, // moduleDir(:moduleIdentityOverride)
	createWithVisibility *registryv1alpha1.Visibility,
	allBranches bool,
	remoteName string,
) error {
	// Assume that this command is run from the repository root. If not, `OpenRepository` will return
	// a dir not found error.
	repo, err := git.OpenRepository(ctx, git.DotGitDir, command.NewRunner())
	if err != nil {
		return fmt.Errorf("open repository: %w", err)
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
	syncerOptions := []bufsync.SyncerOption{
		bufsync.SyncerWithGitRemote(remoteName),
	}
	if allBranches {
		syncerOptions = append(syncerOptions, bufsync.SyncerWithAllBranches())
	}
	if len(modules) == 0 {
		// default behavior, if no modules are passed, a single module at the root of the repo is
		// assumed.
		modules = []string{"."}
	}
	modulesDirsWithOverrides := make(map[string]struct{})
	for _, module := range modules {
		if len(module) == 0 {
			return errors.New("empty module")
		}
		colon := strings.IndexRune(module, ':')
		if colon == -1 {
			// no module override was passed, we can pass the module directory alone and continue
			syncerOptions = append(syncerOptions, bufsync.SyncerWithModule(module, nil))
			continue
		}
		moduleIdentityOverride, err := bufmoduleref.ModuleIdentityForString(module[colon+1:])
		if err != nil {
			return fmt.Errorf("module %s invalid module identity: %w", module, err)
		}
		moduleDir := normalpath.Normalize(module[:colon])
		syncerOptions = append(syncerOptions, bufsync.SyncerWithModule(moduleDir, moduleIdentityOverride))
		modulesDirsWithOverrides[moduleDir] = struct{}{}
	}
	syncer, err := bufsync.NewSyncer(
		container.Logger(),
		repo,
		storageProvider,
		bufsyncapi.NewHandler(
			container.Logger(),
			container,
			repo,
			createWithVisibility,
			func(address string) registryv1alpha1connect.SyncServiceClient {
				return connectclient.Make(clientConfig, address, registryv1alpha1connect.NewSyncServiceClient)
			},
			func(address string) registryv1alpha1connect.ReferenceServiceClient {
				return connectclient.Make(clientConfig, address, registryv1alpha1connect.NewReferenceServiceClient)
			},
			func(address string) registryv1alpha1connect.RepositoryServiceClient {
				return connectclient.Make(clientConfig, address, registryv1alpha1connect.NewRepositoryServiceClient)
			},
			func(address string) registryv1alpha1connect.RepositoryBranchServiceClient {
				return connectclient.Make(clientConfig, address, registryv1alpha1connect.NewRepositoryBranchServiceClient)
			},
			func(address string) registryv1alpha1connect.RepositoryTagServiceClient {
				return connectclient.Make(clientConfig, address, registryv1alpha1connect.NewRepositoryTagServiceClient)
			},
			func(address string) registryv1alpha1connect.RepositoryCommitServiceClient {
				return connectclient.Make(clientConfig, address, registryv1alpha1connect.NewRepositoryCommitServiceClient)
			},
		),
		syncerOptions...,
	)
	if err != nil {
		return fmt.Errorf("new syncer: %w", err)
	}
	return syncer.Sync(ctx)
}
