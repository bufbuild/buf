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

package bufsyncapi

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufsync"
	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/bufpkg/bufcas/bufcasalpha"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app/appext"
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
	container            appext.Container
	repo                 git.Repository
	createWithVisibility *registryv1alpha1.Visibility

	syncServiceClientFactory             SyncServiceClientFactory
	referenceServiceClientFactory        ReferenceServiceClientFactory
	repositoryServiceClientFactory       RepositoryServiceClientFactory
	repositoryBranchServiceClientFactory RepositoryBranchServiceClientFactory
	repositoryTagServiceClientFactory    RepositoryTagServiceClientFactory
	repositoryCommitServiceClientFactory RepositoryCommitServiceClientFactory

	moduleFullNameToRepositoryIDCache  map[string]string
	moduleFullNameToDefaultBranchCache map[string]string
	existingModuleFullNameCache        map[string]struct{}
}

func newSyncHandler(
	logger *zap.Logger,
	container appext.Container,
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
		moduleFullNameToRepositoryIDCache:    make(map[string]string),
		moduleFullNameToDefaultBranchCache:   make(map[string]string),
		existingModuleFullNameCache:          make(map[string]struct{}),
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
	moduleFullName bufmodule.ModuleFullName,
	branchName string,
) (git.Hash, error) {
	service := h.syncServiceClientFactory(moduleFullName.Registry())
	syncPoint, err := service.GetGitSyncPoint(ctx, connect.NewRequest(&registryv1alpha1.GetGitSyncPointRequest{
		Owner:      moduleFullName.Owner(),
		Repository: moduleFullName.Name(),
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
	moduleFullName bufmodule.ModuleFullName,
	hash git.Hash,
) (bool, error) {
	service := h.referenceServiceClientFactory(moduleFullName.Registry())
	res, err := service.GetReferenceByName(ctx, connect.NewRequest(&registryv1alpha1.GetReferenceByNameRequest{
		Owner:          moduleFullName.Owner(),
		RepositoryName: moduleFullName.Name(),
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
	moduleFullName bufmodule.ModuleFullName,
	branchName string,
	hash git.Hash,
) (bool, error) {
	repositoryID, err := h.getRepositoryID(ctx, moduleFullName)
	if err != nil {
		return false, err
	}
	service := h.repositoryBranchServiceClientFactory(moduleFullName.Registry())
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
		repositoryID, err := h.getRepositoryID(ctx, moduleTags.TargetModuleFullName())
		if err != nil {
			return err
		}
		referenceService := h.referenceServiceClientFactory(moduleTags.TargetModuleFullName().Registry())
		repositoryTagService := h.repositoryTagServiceClientFactory(moduleTags.TargetModuleFullName().Registry())
		commitRes, err := referenceService.GetReferenceByName(ctx, connect.NewRequest(&registryv1alpha1.GetReferenceByNameRequest{
			Owner:          moduleTags.TargetModuleFullName().Owner(),
			RepositoryName: moduleTags.TargetModuleFullName().Name(),
			Name:           commit.Commit().Hash().Hex(),
		}))
		if err != nil {
			if connect.CodeOf(err) == connect.CodeNotFound {
				return fmt.Errorf(
					"git commit %q is not known to module %q",
					commit.Commit().Hash(),
					moduleTags.TargetModuleFullName().String(),
				)
			}
			return fmt.Errorf("get reference by name %q: %w", commit.Commit().Hash(), err)
		}
		if commitRes.Msg.Reference.GetVcsCommit() == nil {
			return fmt.Errorf(
				"git commit %q is not synced to module %q",
				commit.Commit().Hash(),
				moduleTags.TargetModuleFullName().String(),
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
					return fmt.Errorf("create new tag %q on module %q: %w", tag, moduleTags.TargetModuleFullName().String(), err)
				}
			} else {
				// TODO: don't do this unless we need to
				_, err := repositoryTagService.UpdateRepositoryTag(ctx, connect.NewRequest(&registryv1alpha1.UpdateRepositoryTagRequest{
					RepositoryId: repositoryID,
					Name:         tag,
					CommitName:   &commitRes.Msg.Reference.GetVcsCommit().CommitName,
				}))
				if err != nil {
					return fmt.Errorf("update existing tag %q on module %q: %w", tag, moduleTags.TargetModuleFullName().String(), err)
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
			if err := h.createRepository(ctx, moduleBranch.TargetModuleFullName()); err != nil {
				return fmt.Errorf("create repo %s: %w", moduleBranch.TargetModuleFullName().String(), err)
			}
		}
		syncPoint, err := h.syncCommitModule(
			ctx,
			moduleCommit.Commit(),
			moduleBranch.BranchName(),
			moduleCommit.Tags(),
			moduleBranch.TargetModuleFullName(),
			bucket,
		)
		if err != nil {
			// We failed to sync. We fail hard on this because the error may be recoverable
			// (i.e., the BSR may be down) and we should re-attempt this commit.
			return fmt.Errorf(
				"sync module %s at branch %s commit %s directory %s: %w",
				moduleBranch.TargetModuleFullName().String(),
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
			moduleBranch.TargetModuleFullName().String(), syncPoint.BsrCommitName,
		)
		if _, err := h.container.Stderr().Write([]byte(syncMsg)); err != nil {
			return fmt.Errorf("write %q to stderr: %w", syncMsg, err)
		}
	}
	return nil
}

func (h *syncHandler) IsProtectedBranch(
	ctx context.Context,
	moduleFullName bufmodule.ModuleFullName,
	branchName string,
) (bool, error) {
	// If the branch is the Git default branch, protect it.
	if branchName == h.repo.DefaultBranch() {
		return true, nil
	}
	return h.IsReleaseBranch(ctx, moduleFullName, branchName)
}

func (h *syncHandler) IsReleaseBranch(
	ctx context.Context,
	moduleFullName bufmodule.ModuleFullName,
	branchName string,
) (bool, error) {
	// We cache a repository's release branch even though it can change because it's _extremely_ unlikely that it changes.
	cacheKey := moduleFullName.String()
	if _, ok := h.moduleFullNameToDefaultBranchCache[cacheKey]; !ok {
		service := h.repositoryServiceClientFactory(moduleFullName.Registry())
		res, err := service.GetRepositoryByFullName(ctx, connect.NewRequest(&registryv1alpha1.GetRepositoryByFullNameRequest{
			FullName: moduleFullName.Owner() + "/" + moduleFullName.Name(),
		}))
		if err != nil {
			if connect.CodeOf(err) == connect.CodeNotFound {
				// Repo not created, no branch is protected because no branches exist. We cache this
				// because it shouldn't change during the lifetime of sync.
				h.moduleFullNameToDefaultBranchCache[cacheKey] = ""
			}
			return false, fmt.Errorf("load repository %q: %w", cacheKey, err)
		}
		h.moduleFullNameToDefaultBranchCache[cacheKey] = res.Msg.Repository.DefaultBranch
	}
	return branchName == h.moduleFullNameToDefaultBranchCache[cacheKey], nil
}

func (h *syncHandler) GetBranchHead(
	ctx context.Context,
	moduleFullName bufmodule.ModuleFullName,
	branchName string,
) (*registryv1alpha1.RepositoryCommit, error) {
	repositoryID, err := h.getRepositoryID(ctx, moduleFullName)
	if err != nil {
		return nil, err
	}
	service := h.repositoryBranchServiceClientFactory(moduleFullName.Registry())
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
	commitService := h.repositoryCommitServiceClientFactory(moduleFullName.Registry())
	res, err := commitService.GetRepositoryCommitByReference(ctx, connect.NewRequest(&registryv1alpha1.GetRepositoryCommitByReferenceRequest{
		RepositoryOwner: moduleFullName.Owner(),
		RepositoryName:  moduleFullName.Name(),
		Reference:       commitName,
	}))
	if err != nil {
		return nil, err
	}
	return res.Msg.RepositoryCommit, nil
}

func (h *syncHandler) GetReleaseHead(
	ctx context.Context,
	moduleFullName bufmodule.ModuleFullName,
) (*registryv1alpha1.RepositoryCommit, error) {
	commitService := h.repositoryCommitServiceClientFactory(moduleFullName.Registry())
	res, err := commitService.GetRepositoryCommitByReference(ctx, connect.NewRequest(&registryv1alpha1.GetRepositoryCommitByReferenceRequest{
		RepositoryOwner: moduleFullName.Owner(),
		RepositoryName:  moduleFullName.Name(),
		Reference:       bufmoduleref.Main,
	}))
	if err != nil {
		return nil, err
	}
	return res.Msg.RepositoryCommit, nil
}

func (h *syncHandler) IsBranchSynced(
	ctx context.Context,
	moduleFullName bufmodule.ModuleFullName,
	branchName string,
) (bool, error) {
	repositoryID, err := h.getRepositoryID(ctx, moduleFullName)
	if err != nil {
		return false, err
	}
	service := h.repositoryBranchServiceClientFactory(moduleFullName.Registry())
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

func (h *syncHandler) getRepositoryID(ctx context.Context, moduleFullName bufmodule.ModuleFullName) (string, error) {
	if repositoryID, hit := h.moduleFullNameToRepositoryIDCache[moduleFullName.String()]; hit {
		return repositoryID, nil
	}
	repoService := h.repositoryServiceClientFactory(moduleFullName.Registry())
	if repoRes, err := repoService.GetRepositoryByFullName(ctx, connect.NewRequest(&registryv1alpha1.GetRepositoryByFullNameRequest{
		FullName: moduleFullName.Owner() + "/" + moduleFullName.Name(),
	})); err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return "", fmt.Errorf("repository for module %q does not exist", moduleFullName.String())
		}
		return "", fmt.Errorf("get repository for module identity: %w", err)
	} else {
		h.moduleFullNameToRepositoryIDCache[moduleFullName.String()] = repoRes.Msg.Repository.Id
	}
	return h.moduleFullNameToRepositoryIDCache[moduleFullName.String()], nil
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
	moduleFullName bufmodule.ModuleFullName,
	moduleBucket storage.ReadBucket,
) (*registryv1alpha1.GitSyncPoint, error) {
	service := h.syncServiceClientFactory(moduleFullName.Registry())
	fileSet, err := bufcas.NewFileSetForBucket(ctx, moduleBucket)
	if err != nil {
		return nil, err
	}
	protoManifestBlob, protoBlobs, err := bufcas.FileSetToProtoManifestBlobAndBlobs(fileSet)
	if err != nil {
		return nil, err
	}
	resp, err := service.SyncGitCommit(ctx, connect.NewRequest(&registryv1alpha1.SyncGitCommitRequest{
		Owner:      moduleFullName.Owner(),
		Repository: moduleFullName.Name(),
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

func (h *syncHandler) createRepository(
	ctx context.Context,
	moduleFullName bufmodule.ModuleFullName,
) error {
	if _, alreadyExists := h.existingModuleFullNameCache[moduleFullName.String()]; alreadyExists {
		return nil
	}
	service := h.repositoryServiceClientFactory(moduleFullName.Registry())
	fullName := moduleFullName.Owner() + "/" + moduleFullName.Name()
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
	h.existingModuleFullNameCache[moduleFullName.String()] = struct{}{}
	return nil
}
