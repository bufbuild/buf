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

package bufsyncapi

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufsync"
	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/bufpkg/bufcas/bufcasalpha"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/storage"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type SyncServiceClientFactory func(address string) registryv1alpha1connect.SyncServiceClient
type ReferenceServiceClientFactory func(address string) registryv1alpha1connect.ReferenceServiceClient
type RepositoryServiceClientFactory func(address string) registryv1alpha1connect.RepositoryServiceClient
type RepositoryBranchServiceClientFactory func(address string) registryv1alpha1connect.RepositoryBranchServiceClient
type RepositoryTagServiceClientFactory func(address string) registryv1alpha1connect.RepositoryTagServiceClient
type RepositoryCommitServiceClientFactory func(address string) registryv1alpha1connect.RepositoryCommitServiceClient

type syncHandler struct {
	logger               *zap.Logger
	container            appflag.Container
	repo                 git.Repository
	createWithVisibility *registryv1alpha1.Visibility

	syncServiceClientFactory             SyncServiceClientFactory
	referenceServiceClientFactory        ReferenceServiceClientFactory
	repositoryServiceClientFactory       RepositoryServiceClientFactory
	repositoryBranchServiceClientFactory RepositoryBranchServiceClientFactory
	repositoryTagServiceClientFactory    RepositoryTagServiceClientFactory
	repositoryCommitServiceClientFactory RepositoryCommitServiceClientFactory

	moduleIdentityToRepositoryIDCache  map[string]string
	moduleIdentityToDefaultBranchCache map[string]string
	existingModuleIdentityCache        map[string]struct{}
}

func newSyncHandler(
	logger *zap.Logger,
	container appflag.Container,
	repo git.Repository,
	createWithVisibility *registryv1alpha1.Visibility,
	syncServiceClientFactory SyncServiceClientFactory,
	referenceServiceClientFactory ReferenceServiceClientFactory,
	repositoryServiceClientFactory RepositoryServiceClientFactory,
	repositoryBranchServiceClientFactory RepositoryBranchServiceClientFactory,
	repositoryTagServiceClientFactory RepositoryTagServiceClientFactory,
	repositoryCommitServiceClientFactory RepositoryCommitServiceClientFactory,
) bufsync.Handler {
	return &syncHandler{
		logger:                               logger,
		container:                            container,
		repo:                                 repo,
		createWithVisibility:                 createWithVisibility,
		moduleIdentityToRepositoryIDCache:    make(map[string]string),
		moduleIdentityToDefaultBranchCache:   make(map[string]string),
		existingModuleIdentityCache:          make(map[string]struct{}),
		syncServiceClientFactory:             syncServiceClientFactory,
		referenceServiceClientFactory:        referenceServiceClientFactory,
		repositoryServiceClientFactory:       repositoryServiceClientFactory,
		repositoryBranchServiceClientFactory: repositoryBranchServiceClientFactory,
		repositoryTagServiceClientFactory:    repositoryTagServiceClientFactory,
		repositoryCommitServiceClientFactory: repositoryCommitServiceClientFactory,
	}
}

func (h *syncHandler) ResolveSyncPoint(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
	branchName string,
) (git.Hash, error) {
	service := h.syncServiceClientFactory(moduleIdentity.Remote())
	syncPoint, err := service.GetGitSyncPoint(ctx, connect.NewRequest(&registryv1alpha1.GetGitSyncPointRequest{
		Owner:      moduleIdentity.Owner(),
		Repository: moduleIdentity.Repository(),
		Branch:     branchName,
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
	service := h.referenceServiceClientFactory(moduleIdentity.Remote())
	res, err := service.GetReferenceByName(ctx, connect.NewRequest(&registryv1alpha1.GetReferenceByNameRequest{
		Owner:          moduleIdentity.Owner(),
		RepositoryName: moduleIdentity.Repository(),
		Name:           hash.Hex(),
	}))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			// Repo is not created, or reference does not exist anywhere. Either way, false.
			return false, nil
		}
		return false, fmt.Errorf("get reference by name: %w", err)
	}
	return res.Msg.Reference.GetVcsCommit() != nil, nil
}

func (h *syncHandler) IsGitCommitSyncedToBranch(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
	branchName string,
	hash git.Hash,
) (bool, error) {
	repositoryID, err := h.getRepositoryID(ctx, moduleIdentity)
	if err != nil {
		return false, err
	}
	service := h.repositoryBranchServiceClientFactory(moduleIdentity.Remote())
	var nextPageToken string
	for {
		res, err := service.ListRepositoryBranchesByReference(ctx, connect.NewRequest(&registryv1alpha1.ListRepositoryBranchesByReferenceRequest{
			RepositoryId: repositoryID,
			PageToken:    nextPageToken,
			PageSize:     10,
			Reference: &registryv1alpha1.ListRepositoryBranchesByReferenceRequest_VcsCommitHash{
				VcsCommitHash: hash.Hex(),
			},
		}))
		if err != nil {
			if connect.CodeOf(err) == connect.CodeNotFound {
				// Repo is not created
				return false, nil
			}
			return false, fmt.Errorf("list repository branch by reference: %w", err)
		}
		for _, branch := range res.Msg.RepositoryBranches {
			if branch.Name == branchName {
				return true, nil
			}
		}
		if res.Msg.NextPageToken == "" {
			break
		}
		nextPageToken = res.Msg.NextPageToken
	}
	return false, nil
}

func (h *syncHandler) SyncModuleTags(
	ctx context.Context,
	moduleTags bufsync.ModuleTags,
) error {
	for _, commit := range moduleTags.TaggedCommitsToSync() {
		repositoryID, err := h.getRepositoryID(ctx, moduleTags.TargetModuleIdentity())
		if err != nil {
			return err
		}
		referenceService := h.referenceServiceClientFactory(moduleTags.TargetModuleIdentity().Remote())
		repositoryTagService := h.repositoryTagServiceClientFactory(moduleTags.TargetModuleIdentity().Remote())
		commitRes, err := referenceService.GetReferenceByName(ctx, connect.NewRequest(&registryv1alpha1.GetReferenceByNameRequest{
			Owner:          moduleTags.TargetModuleIdentity().Owner(),
			RepositoryName: moduleTags.TargetModuleIdentity().Repository(),
			Name:           commit.Commit().Hash().Hex(),
		}))
		if err != nil {
			if connect.CodeOf(err) == connect.CodeNotFound {
				return fmt.Errorf(
					"git commit %q is not known to module %q",
					commit.Commit().Hash(),
					moduleTags.TargetModuleIdentity().IdentityString(),
				)
			}
			return fmt.Errorf("get reference by name %q: %w", commit.Commit().Hash(), err)
		}
		if commitRes.Msg.Reference.GetVcsCommit() == nil {
			return fmt.Errorf(
				"git commit %q is not synced to module %q",
				commit.Commit().Hash(),
				moduleTags.TargetModuleIdentity().IdentityString(),
			)
		}
		for _, tag := range commit.Tags() {
			tagExists, err := h.bsrTagExists(ctx, repositoryTagService, repositoryID, tag)
			if err != nil {
				return fmt.Errorf("determine if tag %q exists: %w", tag, err)
			}
			if !tagExists {
				_, err := repositoryTagService.CreateRepositoryTag(ctx, connect.NewRequest(&registryv1alpha1.CreateRepositoryTagRequest{
					RepositoryId: repositoryID,
					Name:         tag,
					CommitName:   commitRes.Msg.Reference.GetVcsCommit().CommitName,
				}))
				if err != nil {
					return fmt.Errorf("create new tag %q on module %q: %w", tag, moduleTags.TargetModuleIdentity().IdentityString(), err)
				}
			} else {
				// TODO: don't do this unless we need to
				_, err := repositoryTagService.UpdateRepositoryTag(ctx, connect.NewRequest(&registryv1alpha1.UpdateRepositoryTagRequest{
					RepositoryId: repositoryID,
					Name:         tag,
					CommitName:   &commitRes.Msg.Reference.GetVcsCommit().CommitName,
				}))
				if err != nil {
					return fmt.Errorf("update existing tag %q on module %q: %w", tag, moduleTags.TargetModuleIdentity().IdentityString(), err)
				}
			}
		}
	}
	return nil
}

func (h *syncHandler) SyncModuleBranch(ctx context.Context, moduleBranch bufsync.ModuleBranch) error {
	for _, moduleCommit := range moduleBranch.CommitsToSync() {
		bucket, err := moduleCommit.Bucket(ctx)
		if err != nil {
			return fmt.Errorf("read bucket for commit %q: %w", moduleCommit.Commit().Hash(), err)
		}
		if h.createWithVisibility != nil {
			if err := h.createRepository(ctx, moduleBranch.TargetModuleIdentity()); err != nil {
				return fmt.Errorf("create repo %s: %w", moduleBranch.TargetModuleIdentity(), err)
			}
		}
		syncPoint, err := h.syncCommitModule(
			ctx,
			moduleCommit.Commit(),
			moduleBranch.BranchName(),
			moduleCommit.Tags(),
			moduleBranch.TargetModuleIdentity(),
			bucket,
		)
		if err != nil {
			// We failed to sync. We fail hard on this because the error may be recoverable
			// (i.e., the BSR may be down) and we should re-attempt this commit.
			return fmt.Errorf(
				"sync module %s at branch %s commit %s directory %s: %w",
				moduleBranch.TargetModuleIdentity().IdentityString(),
				moduleBranch.BranchName(),
				moduleCommit.Commit().Hash(),
				moduleBranch.Directory(),
				err,
			)
		}
		syncMsg := fmt.Sprintf(
			// from local                                        -> to remote
			// <module-directory>:<git-branch>:<git-commit-hash> -> <module-identity>:<bsr-commit-name>
			"%s:%s:%s -> %s:%s\n",
			moduleBranch.Directory(), moduleBranch.BranchName(), moduleCommit.Commit().Hash().Hex(),
			moduleBranch.TargetModuleIdentity().IdentityString(), syncPoint.BsrCommitName,
		)
		if _, err := h.container.Stderr().Write([]byte(syncMsg)); err != nil {
			return fmt.Errorf("write %q to stderr: %w", syncMsg, err)
		}
	}
	return nil
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
	return h.IsReleaseBranch(ctx, moduleIdentity, branchName)
}

func (h *syncHandler) IsReleaseBranch(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
	branchName string,
) (bool, error) {
	// We cache a repository's release branch even though it can change because it's _extremely_ unlikely that it changes.
	cacheKey := moduleIdentity.IdentityString()
	if _, ok := h.moduleIdentityToDefaultBranchCache[cacheKey]; !ok {
		service := h.repositoryServiceClientFactory(moduleIdentity.Remote())
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
	repositoryID, err := h.getRepositoryID(ctx, moduleIdentity)
	if err != nil {
		return nil, err
	}
	service := h.repositoryBranchServiceClientFactory(moduleIdentity.Remote())
	branchRes, err := service.GetRepositoryBranch(ctx, connect.NewRequest(&registryv1alpha1.GetRepositoryBranchRequest{
		RepositoryId: repositoryID,
		Name:         branchName,
	}))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return nil, nil
		}
		return nil, err
	}
	commitName := branchRes.Msg.Branch.LatestCommitName
	if commitName == "" {
		return nil, nil // branch has no commits on it
	}
	commitService := h.repositoryCommitServiceClientFactory(moduleIdentity.Remote())
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

func (h *syncHandler) GetReleaseHead(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
) (*registryv1alpha1.RepositoryCommit, error) {
	commitService := h.repositoryCommitServiceClientFactory(moduleIdentity.Remote())
	res, err := commitService.GetRepositoryCommitByReference(ctx, connect.NewRequest(&registryv1alpha1.GetRepositoryCommitByReferenceRequest{
		RepositoryOwner: moduleIdentity.Owner(),
		RepositoryName:  moduleIdentity.Repository(),
		Reference:       bufmoduleref.Main,
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
		return false, err
	}
	service := h.repositoryBranchServiceClientFactory(moduleIdentity.Remote())
	branchRes, err := service.GetRepositoryBranch(ctx, connect.NewRequest(&registryv1alpha1.GetRepositoryBranchRequest{
		RepositoryId: repositoryID,
		Name:         branchName,
	}))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return false, nil
		}
		return false, err
	}
	return branchRes.Msg.Branch.LastUpdateGitCommitHash != "", nil
}

func (h *syncHandler) getRepositoryID(ctx context.Context, moduleIdentity bufmoduleref.ModuleIdentity) (string, error) {
	if repositoryID, hit := h.moduleIdentityToRepositoryIDCache[moduleIdentity.IdentityString()]; hit {
		return repositoryID, nil
	}
	repoService := h.repositoryServiceClientFactory(moduleIdentity.Remote())
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

func (h *syncHandler) syncCommitModule(
	ctx context.Context,
	commit git.Commit,
	branchName string,
	tags []string,
	moduleIdentity bufmoduleref.ModuleIdentity,
	moduleBucket storage.ReadBucket,
) (*registryv1alpha1.GitSyncPoint, error) {
	service := h.syncServiceClientFactory(moduleIdentity.Remote())
	fileSet, err := bufcas.NewFileSetForBucket(ctx, moduleBucket)
	if err != nil {
		return nil, err
	}
	protoManifestBlob, protoBlobs, err := bufcasalpha.FileSetToAlphaManifestBlobAndBlobs(fileSet)
	if err != nil {
		return nil, err
	}
	resp, err := service.SyncGitCommit(ctx, connect.NewRequest(&registryv1alpha1.SyncGitCommitRequest{
		Owner:      moduleIdentity.Owner(),
		Repository: moduleIdentity.Repository(),
		Manifest:   protoManifestBlob,
		Blobs:      protoBlobs,
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

func (h *syncHandler) createRepository(
	ctx context.Context,
	moduleIdentity bufmoduleref.ModuleIdentity,
) error {
	if _, alreadyExists := h.existingModuleIdentityCache[moduleIdentity.IdentityString()]; alreadyExists {
		return nil
	}
	service := h.repositoryServiceClientFactory(moduleIdentity.Remote())
	fullName := moduleIdentity.Owner() + "/" + moduleIdentity.Repository()
	_, err := service.CreateRepositoryByFullName(
		ctx,
		connect.NewRequest(&registryv1alpha1.CreateRepositoryByFullNameRequest{
			FullName:   fullName,
			Visibility: *h.createWithVisibility,
		}),
	)
	if err != nil && connect.CodeOf(err) != connect.CodeAlreadyExists {
		return err
	}
	// if created successfully or if it already existed, cache it
	h.existingModuleIdentityCache[moduleIdentity.IdentityString()] = struct{}{}
	return nil
}
