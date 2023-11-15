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
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type syncHandler struct {
	logger               *zap.Logger
	clientConfig         *connectclient.Config
	container            appflag.Container
	repo                 git.Repository
	createWithVisibility string

	moduleIdentityToRepositoryIDCache  map[string]string
	moduleIdentityToDefaultBranchCache map[string]string
	syncGitCommitsCache                map[struct {
		hash   string
		branch string
	}]bool
}

func newSyncHandler(
	logger *zap.Logger,
	clientConfig *connectclient.Config,
	container appflag.Container,
	repo git.Repository,
	createWithVisibility string,
) bufsync.Handler {
	return &syncHandler{
		logger:                             logger,
		clientConfig:                       clientConfig,
		container:                          container,
		repo:                               repo,
		createWithVisibility:               createWithVisibility,
		moduleIdentityToRepositoryIDCache:  make(map[string]string),
		moduleIdentityToDefaultBranchCache: make(map[string]string),
		syncGitCommitsCache: make(map[struct {
			hash   string
			branch string
		}]bool),
	}
}

func (h *syncHandler) ResolveSyncPoint(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
	branch string,
) (git.Hash, error) {
	service := connectclient.Make(h.clientConfig, moduleIdentity.Remote(), registryv1alpha1connect.NewSyncServiceClient)
	syncPoint, err := service.GetGitSyncPoint(ctx, connect.NewRequest(&registryv1alpha1.GetGitSyncPointRequest{
		Owner:      moduleIdentity.Owner(),
		Repository: moduleIdentity.Repository(),
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

func (h *syncHandler) IsGitCommitSynced(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
	hash git.Hash,
) (bool, error) {
	cacheKey := struct {
		hash   string
		branch string
	}{hash: hash.Hex()}
	if _, hit := h.syncGitCommitsCache[cacheKey]; !hit {
		service := connectclient.Make(h.clientConfig, moduleIdentity.Remote(), registryv1alpha1connect.NewReferenceServiceClient)
		res, err := service.GetReferenceByName(ctx, connect.NewRequest(&registryv1alpha1.GetReferenceByNameRequest{
			Owner:          moduleIdentity.Owner(),
			RepositoryName: moduleIdentity.Repository(),
			Name:           hash.Hex(),
		}))
		if err != nil {
			if connect.CodeOf(err) == connect.CodeNotFound {
				// Repo is not created
				h.syncGitCommitsCache[cacheKey] = false
				return false, nil
			}
			return false, fmt.Errorf("get reference by name: %w", err)
		}
		h.syncGitCommitsCache[cacheKey] = res.Msg.Reference.GetVcsCommit() != nil
	}
	return h.syncGitCommitsCache[cacheKey], nil
}

func (h *syncHandler) IsGitCommitSyncedToBranch(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
	branch string,
	hash git.Hash,
) (bool, error) {
	cacheKey := struct {
		hash   string
		branch string
	}{hash: hash.Hex(), branch: branch}
	if _, hit := h.syncGitCommitsCache[cacheKey]; !hit {
		repositoryID, err := h.getRepositoryID(ctx, moduleIdentity)
		if err != nil {
			return false, err
		}
		service := connectclient.Make(h.clientConfig, moduleIdentity.Remote(), registryv1alpha1connect.NewRepositoryBranchServiceClient)
		res, err := service.ListRepositoryBranchesByReference(ctx, connect.NewRequest(&registryv1alpha1.ListRepositoryBranchesByReferenceRequest{
			RepositoryId: repositoryID,
			Reference: &registryv1alpha1.ListRepositoryBranchesByReferenceRequest_VcsCommitHash{
				VcsCommitHash: hash.Hex(),
			},
		}))
		if err != nil {
			if connect.CodeOf(err) == connect.CodeNotFound {
				// Repo is not created
				h.syncGitCommitsCache[cacheKey] = false
				return false, nil
			}
			return false, fmt.Errorf("list repository branch by reference: %w", err)
		}
		h.syncGitCommitsCache[cacheKey] = len(res.Msg.RepositoryBranches) > 0
	}
	return h.syncGitCommitsCache[cacheKey], nil
}

func (h *syncHandler) SyncModuleTags(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
	commitTags map[git.Hash][]string,
) error {
	repositoryID, err := h.getRepositoryID(ctx, moduleIdentity)
	if err != nil {
		return err
	}
	referenceService := connectclient.Make(h.clientConfig, moduleIdentity.Remote(), registryv1alpha1connect.NewReferenceServiceClient)
	tagService := connectclient.Make(h.clientConfig, moduleIdentity.Remote(), registryv1alpha1connect.NewRepositoryTagServiceClient)
	for commit, tags := range commitTags {
		commitRes, err := referenceService.GetReferenceByName(ctx, connect.NewRequest(&registryv1alpha1.GetReferenceByNameRequest{
			Owner:          moduleIdentity.Owner(),
			RepositoryName: moduleIdentity.Repository(),
			Name:           commit.Hex(),
		}))
		if err != nil {
			return fmt.Errorf("get reference by name %q: %w", commit, err)
		}
		if commitRes.Msg.Reference.GetVcsCommit() == nil {
			return fmt.Errorf("git commit %q not synced to module %q", commit, moduleIdentity.IdentityString())
		}
		for _, tag := range tags {
			tagExists, err := h.bsrTagExists(ctx, tagService, repositoryID, tag)
			if err != nil {
				return fmt.Errorf("determine if tag %q exists: %w", tag, err)
			}
			if !tagExists {
				_, err := tagService.CreateRepositoryTag(ctx, connect.NewRequest(&registryv1alpha1.CreateRepositoryTagRequest{
					RepositoryId: repositoryID,
					Name:         tag,
					CommitName:   commitRes.Msg.Reference.GetVcsCommit().CommitName,
				}))
				if err != nil {
					return fmt.Errorf("create new tag %q on module %q: %w", tag, moduleIdentity.IdentityString(), err)
				}
			} else {
				_, err := tagService.UpdateRepositoryTag(ctx, connect.NewRequest(&registryv1alpha1.UpdateRepositoryTagRequest{
					RepositoryId: repositoryID,
					Name:         tag,
					CommitName:   &commitRes.Msg.Reference.GetVcsCommit().CommitName,
				}))
				if err != nil {
					return fmt.Errorf("update existing tag %q on module %q: %w", tag, moduleIdentity.IdentityString(), err)
				}
			}
		}
	}
	return nil
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

func (h *syncHandler) InvalidBSRSyncPoint(
	moduleIdentity bufmoduleref.ModuleIdentity,
	branchName string,
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
				syncPoint.Hex(), branchName, moduleIdentity.IdentityString(),
			)
		}
		h.logger.Warn(
			"last synced git commit not found in the git repo for a non-default branch",
			zap.String("module", moduleIdentity.IdentityString()),
			zap.String("branch", branchName),
			zap.String("last synced git commit", syncPoint.Hex()),
		)
		return nil
	}
	// Other error, let's abort sync.
	return fmt.Errorf(
		"invalid sync point %q for branch %q in module %q: %w",
		syncPoint.Hex(), branchName, moduleIdentity.IdentityString(), err,
	)
}

func (h *syncHandler) IsProtectedBranch(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
	branchName string,
) (bool, error) {
	// If the branch is the Git default branch, protect it.
	if branchName == h.repo.DefaultBranch() {
		return true, nil
	}
	// Otherwise the only other protected branch is the Repository's default (release) branch.
	cacheKey := moduleIdentity.IdentityString()
	if _, ok := h.moduleIdentityToDefaultBranchCache[cacheKey]; !ok {
		service := connectclient.Make(h.clientConfig, moduleIdentity.Remote(), registryv1alpha1connect.NewRepositoryServiceClient)
		res, err := service.GetRepositoryByFullName(ctx, connect.NewRequest(&registryv1alpha1.GetRepositoryByFullNameRequest{
			FullName: moduleIdentity.Owner() + "/" + moduleIdentity.Repository(),
		}))
		if err != nil {
			if connect.CodeOf(err) == connect.CodeNotFound {
				// Repo not created, no branch is protected because no branches exist. We cache this
				// because it shouldn't change during the lifetime of sync.
				h.moduleIdentityToDefaultBranchCache[cacheKey] = ""
			}
			return false, fmt.Errorf("load repository %q: %w", cacheKey, err)
		}
		h.moduleIdentityToDefaultBranchCache[cacheKey] = res.Msg.Repository.DefaultBranch
	}
	return branchName == h.moduleIdentityToDefaultBranchCache[cacheKey], nil
}

func (h *syncHandler) GetBranchHead(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
	branchName string,
) (*registryv1alpha1.RepositoryCommit, error) {
	service := connectclient.Make(h.clientConfig, moduleIdentity.Remote(), registryv1alpha1connect.NewReferenceServiceClient)
	refRes, err := service.GetReferenceByName(ctx, connect.NewRequest(&registryv1alpha1.GetReferenceByNameRequest{
		Owner:          moduleIdentity.Owner(),
		RepositoryName: moduleIdentity.Repository(),
		Name:           branchName,
	}))
	if err != nil {
		return nil, err
	}
	if refRes.Msg.GetReference().GetBranch() == nil {
		return nil, fmt.Errorf("reference %q did not resolve to a branch", branchName)
	}
	commitName := refRes.Msg.GetReference().GetBranch().GetLatestCommitName()
	if commitName == "" {
		return nil, fmt.Errorf("branch %q has no commits on it", branchName)
	}
	commitService := connectclient.Make(h.clientConfig, moduleIdentity.Remote(), registryv1alpha1connect.NewRepositoryCommitServiceClient)
	res, err := commitService.GetRepositoryCommitByReference(ctx, connect.NewRequest(&registryv1alpha1.GetRepositoryCommitByReferenceRequest{
		RepositoryOwner: moduleIdentity.Owner(),
		RepositoryName:  moduleIdentity.Repository(),
		Reference:       commitName,
	}))
	if err != nil {
		return nil, err
	}
	return res.Msg.RepositoryCommit, nil
}

func (h *syncHandler) IsBranchSynced(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
	branchName string,
) (bool, error) {
	repositoryID, err := h.getRepositoryID(ctx, moduleIdentity)
	if err != nil {
		return false, nil
	}
	service := connectclient.Make(h.clientConfig, moduleIdentity.Remote(), registryv1alpha1connect.NewRepositoryBranchServiceClient)
	branchRes, err := service.GetRepositoryBranch(ctx, connect.NewRequest(&registryv1alpha1.GetRepositoryBranchRequest{
		RepositoryId: repositoryID,
		Name:         branchName,
	}))
	if err != nil {
		return false, err
	}
	return branchRes.Msg.Branch.LastUpdateGitCommitHash != "", nil
}

func (h *syncHandler) getRepositoryID(ctx context.Context, moduleIdentity bufmoduleref.ModuleIdentity) (string, error) {
	if _, hit := h.moduleIdentityToRepositoryIDCache[moduleIdentity.IdentityString()]; !hit {
		repoService := connectclient.Make(h.clientConfig, moduleIdentity.Remote(), registryv1alpha1connect.NewRepositoryServiceClient)
		if repoRes, err := repoService.GetRepositoryByFullName(ctx, connect.NewRequest(&registryv1alpha1.GetRepositoryByFullNameRequest{
			FullName: moduleIdentity.Owner() + "/" + moduleIdentity.Repository(),
		})); err != nil {
			if connect.CodeOf(err) == connect.CodeNotFound {
				return "", fmt.Errorf("repository for module %q does not exist", moduleIdentity.IdentityString())
			}
			return "", fmt.Errorf("get repository for module identity: %w", err)
		} else {
			h.moduleIdentityToRepositoryIDCache[moduleIdentity.IdentityString()] = repoRes.Msg.Repository.Id
		}
	}
	return h.moduleIdentityToRepositoryIDCache[moduleIdentity.IdentityString()], nil
}

func (h *syncHandler) bsrTagExists(
	ctx context.Context,
	client registryv1alpha1connect.RepositoryTagServiceClient,
	repositoryID string,
	tagName string,
) (bool, error) {
	_, err := client.GetRepositoryTag(ctx, connect.NewRequest(&registryv1alpha1.GetRepositoryTagRequest{
		RepositoryId: repositoryID,
		Name:         tagName,
	}))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (h *syncHandler) pushOrCreate(
	ctx context.Context,
	commit git.Commit,
	branchName string,
	tags []string,
	moduleIdentity bufmoduleref.ModuleIdentity,
	moduleBucket storage.ReadBucket,
) (*registryv1alpha1.GitSyncPoint, error) {
	modulePin, err := h.push(
		ctx,
		commit,
		branchName,
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
				branchName,
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
	branchName string,
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
		Branch:     branchName,
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
