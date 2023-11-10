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
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufsync"
	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/bufpkg/bufcas/bufcasalpha"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type syncHandler struct {
	logger                          *zap.Logger
	clientConfig                    *connectclient.Config
	container                       appflag.Container
	repo                            git.Repository
	createWithVisibility            string
	modulesDirsWithIdentityOverride map[string]struct{}
}

func newSyncHandler(
	logger *zap.Logger,
	clientConfig *connectclient.Config,
	container appflag.Container,
	repo git.Repository,
	createWithVisibility string,
	modulesDirsWithIdentityOverride map[string]struct{},
) bufsync.Handler {
	return &syncHandler{
		logger:                          logger,
		clientConfig:                    clientConfig,
		container:                       container,
		repo:                            repo,
		createWithVisibility:            createWithVisibility,
		modulesDirsWithIdentityOverride: modulesDirsWithIdentityOverride,
	}
}

func (h *syncHandler) ResolveSyncPoint(ctx context.Context, module bufmoduleref.ModuleIdentity, branch string) (git.Hash, error) {
	service := connectclient.Make(h.clientConfig, module.Remote(), registryv1alpha1connect.NewSyncServiceClient)
	syncPoint, err := service.GetGitSyncPoint(ctx, connect.NewRequest(&registryv1alpha1.GetGitSyncPointRequest{
		Owner:      module.Owner(),
		Repository: module.Repository(),
		Branch:     branch,
	}))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			// No syncpoint
			return nil, nil
		}
		return nil, fmt.Errorf("get git sync point: %w", err)
	}
	hash, err := git.NewHashFromHex(syncPoint.Msg.GetSyncPoint().GitCommitHash)
	if err != nil {
		return nil, fmt.Errorf(
			"invalid sync point from BSR %q: %w",
			syncPoint.Msg.GetSyncPoint().GetGitCommitHash(),
			err,
		)
	}
	return hash, nil
}

func (h *syncHandler) CheckSyncedGitCommits(ctx context.Context, module bufmoduleref.ModuleIdentity, commitHashes map[string]struct{}) (map[string]struct{}, error) {
	service := connectclient.Make(h.clientConfig, module.Remote(), registryv1alpha1connect.NewLabelServiceClient)
	res, err := service.GetLabelsInNamespace(ctx, connect.NewRequest(&registryv1alpha1.GetLabelsInNamespaceRequest{
		RepositoryOwner: module.Owner(),
		RepositoryName:  module.Repository(),
		LabelNamespace:  registryv1alpha1.LabelNamespace_LABEL_NAMESPACE_GIT_COMMIT,
		LabelNames:      stringutil.MapToSlice(commitHashes),
	}))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			// Repo is not created
			return nil, nil
		}
		return nil, fmt.Errorf("get labels in namespace: %w", err)
	}
	syncedHashes := make(map[string]struct{})
	for _, label := range res.Msg.Labels {
		syncedHash := label.LabelName.Name
		if _, expected := commitHashes[syncedHash]; !expected {
			return nil, fmt.Errorf("received unexpected synced hash %q, expected %v", syncedHash, commitHashes)
		}
		syncedHashes[syncedHash] = struct{}{}
	}
	return syncedHashes, nil
}

func (h *syncHandler) GetModuleReleaseBranch(ctx context.Context, module bufmoduleref.ModuleIdentity) (string, error) {
	service := connectclient.Make(h.clientConfig, module.Remote(), registryv1alpha1connect.NewRepositoryServiceClient)
	res, err := service.GetRepositoryByFullName(ctx, connect.NewRequest(&registryv1alpha1.GetRepositoryByFullNameRequest{
		FullName: module.Owner() + "/" + module.Repository(),
	}))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			// Repo is not created
			return "", bufsync.ErrModuleDoesNotExist
		}
		return "", fmt.Errorf("get repository by full name %q: %w", module.IdentityString(), err)
	}
	return res.Msg.Repository.DefaultBranch, nil
}

func (h *syncHandler) BackfillTags(
	ctx context.Context,
	module bufmoduleref.ModuleIdentity,
	alreadySyncedHash git.Hash,
	author git.Ident,
	committer git.Ident,
	tags []string,
) (string, error) {
	service := connectclient.Make(h.clientConfig, module.Remote(), registryv1alpha1connect.NewSyncServiceClient)
	res, err := service.AttachGitTags(ctx, connect.NewRequest(&registryv1alpha1.AttachGitTagsRequest{
		Owner:      module.Owner(),
		Repository: module.Repository(),
		Hash:       alreadySyncedHash.Hex(),
		Author: &registryv1alpha1.GitIdentity{
			Name:  author.Name(),
			Email: author.Email(),
			Time:  timestamppb.New(author.Timestamp()),
		},
		Committer: &registryv1alpha1.GitIdentity{
			Name:  committer.Name(),
			Email: committer.Email(),
			Time:  timestamppb.New(committer.Timestamp()),
		},
		Tags: tags,
	}))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			// Repo is not created
			return "", bufsync.ErrModuleDoesNotExist
		}
		return "", fmt.Errorf("attach git tags to module %q: %w", module.IdentityString(), err)
	}
	return res.Msg.GetBsrCommitName(), nil
}

func (h *syncHandler) SyncModuleCommit(ctx context.Context, moduleCommit bufsync.ModuleCommit) error {
	syncPoint, err := h.pushOrCreate(
		ctx,
		moduleCommit.Commit(),
		moduleCommit.Branch(),
		moduleCommit.Tags(),
		moduleCommit.Identity(),
		moduleCommit.Bucket(),
	)
	if err != nil {
		// We failed to push. We fail hard on this because the error may be recoverable
		// (i.e., the BSR may be down) and we should re-attempt this commit.
		return fmt.Errorf(
			"failed to push or create %s at %s: %w",
			moduleCommit.Identity().IdentityString(),
			moduleCommit.Commit().Hash(),
			err,
		)
	}
	_, err = h.container.Stderr().Write([]byte(
		// from local                                        -> to remote
		// <module-directory>:<git-branch>:<git-commit-hash> -> <module-identity>:<bsr-commit-name>
		fmt.Sprintf(
			"%s:%s:%s -> %s:%s\n",
			moduleCommit.Directory(), moduleCommit.Branch(), moduleCommit.Commit().Hash().Hex(),
			moduleCommit.Identity().IdentityString(), syncPoint.BsrCommitName,
		)),
	)
	return err
}

func (h *syncHandler) HandleReadModuleError(err *bufsync.ReadModuleError) bufsync.LookbackDecisionCode {
	switch err.Code() {
	case bufsync.ReadModuleErrorCodeModuleNotFound,
		bufsync.ReadModuleErrorCodeInvalidModuleConfig,
		bufsync.ReadModuleErrorCodeBuildModule:
		// if the module cannot be found, has an invalid config, or cannot build, we can just skip the
		// commit.
		return bufsync.LookbackDecisionCodeSkip
	case bufsync.ReadModuleErrorCodeUnnamedModule,
		bufsync.ReadModuleErrorCodeUnexpectedName:
		// if the module has an unexpected or no name, we should override the module identity only if it
		// was passed explicitly as an identity override, otherwise skip the commit.
		if _, hasExplicitOverride := h.modulesDirsWithIdentityOverride[err.ModuleDir()]; hasExplicitOverride {
			return bufsync.LookbackDecisionCodeOverride
		}
		return bufsync.LookbackDecisionCodeSkip
	}
	// any unhandled scenarios? just fail the sync
	return bufsync.LookbackDecisionCodeFail
}

func (h *syncHandler) InvalidBSRSyncPoint(
	module bufmoduleref.ModuleIdentity,
	branch string,
	syncPoint git.Hash,
	isGitDefaultBranch bool,
	err error,
) error {
	// The most likely culprit for an invalid sync point is a rebase, where the last known commit has
	// been garbage collected. In this case, let's present a better error message.
	//
	// This is not trivial scenario if the branch that's been rebased is a long-lived branch (like
	// main) whose artifacts are consumed by other branches, as we may fail to sync those commits if
	// we continue.
	//
	// For now we simply error if this happens in the default branch, and WARN+skip for the other
	// branches. We may want to provide a flag in the future for forcing sync to continue despite
	// this.
	if errors.Is(err, git.ErrObjectNotFound) {
		if isGitDefaultBranch {
			return fmt.Errorf(
				"last synced git commit %q for default branch %q in module %q is not found in the git repo, did you rebase or reset your default branch?",
				syncPoint.Hex(), branch, module.IdentityString(),
			)
		}
		h.logger.Warn(
			"last synced git commit not found in the git repo for a non-default branch",
			zap.String("module", module.IdentityString()),
			zap.String("branch", branch),
			zap.String("last synced git commit", syncPoint.Hex()),
		)
		return nil
	}
	// Other error, let's abort sync.
	return fmt.Errorf(
		"invalid sync point %q for branch %q in module %q: %w",
		syncPoint.Hex(), branch, module.IdentityString(), err,
	)
}

func (h *syncHandler) pushOrCreate(
	ctx context.Context,
	commit git.Commit,
	branch string,
	tags []string,
	moduleIdentity bufmoduleref.ModuleIdentity,
	moduleBucket storage.ReadBucket,
) (*registryv1alpha1.GitSyncPoint, error) {
	modulePin, err := h.push(
		ctx,
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
		if h.createWithVisibility != "" && connect.CodeOf(err) == connect.CodeNotFound {
			if err := h.create(ctx, moduleIdentity); err != nil {
				return nil, fmt.Errorf("create repo: %w", err)
			}
			return h.push(
				ctx,
				commit,
				branch,
				tags,
				moduleIdentity,
				moduleBucket,
			)
		}
		return nil, fmt.Errorf("push: %w", err)
	}
	return modulePin, nil
}

func (h *syncHandler) push(
	ctx context.Context,
	commit git.Commit,
	branch string,
	tags []string,
	moduleIdentity bufmoduleref.ModuleIdentity,
	moduleBucket storage.ReadBucket,
) (*registryv1alpha1.GitSyncPoint, error) {
	service := connectclient.Make(h.clientConfig, moduleIdentity.Remote(), registryv1alpha1connect.NewSyncServiceClient)
	fileSet, err := bufcas.NewFileSetForBucket(ctx, moduleBucket)
	if err != nil {
		return nil, err
	}
	protoManifestBlob, protoBlobs, err := bufcas.FileSetToProtoManifestBlobAndBlobs(fileSet)
	if err != nil {
		return nil, err
	}
	resp, err := service.SyncGitCommit(ctx, connect.NewRequest(&registryv1alpha1.SyncGitCommitRequest{
		Owner:      moduleIdentity.Owner(),
		Repository: moduleIdentity.Repository(),
		Manifest:   bufcasalpha.BlobToAlpha(protoManifestBlob),
		Blobs:      bufcasalpha.BlobsToAlpha(protoBlobs),
		Hash:       commit.Hash().Hex(),
		Branch:     branch,
		Tags:       tags,
		Author: &registryv1alpha1.GitIdentity{
			Name:  commit.Author().Name(),
			Email: commit.Author().Email(),
			Time:  timestamppb.New(commit.Author().Timestamp()),
		},
		Committer: &registryv1alpha1.GitIdentity{
			Name:  commit.Committer().Name(),
			Email: commit.Committer().Email(),
			Time:  timestamppb.New(commit.Committer().Timestamp()),
		},
	}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.SyncPoint, nil
}

func (h *syncHandler) create(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
) error {
	service := connectclient.Make(h.clientConfig, moduleIdentity.Remote(), registryv1alpha1connect.NewRepositoryServiceClient)
	visiblity, err := bufcli.VisibilityFlagToVisibility(h.createWithVisibility)
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
