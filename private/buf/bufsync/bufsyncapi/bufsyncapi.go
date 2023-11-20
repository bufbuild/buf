package bufsyncapi

import (
	"github.com/bufbuild/buf/private/buf/bufsync"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/git"
	"go.uber.org/zap"
)

// NewHandle returns a new bufsync.Handler that handles requests by communicating with a BSR instance.
func NewHandler(
	logger *zap.Logger,
	container appflag.Container,
	repo git.Repository,
	createWithVisibility string,
	syncServiceClientFactory SyncServiceClientFactory,
	referenceServiceClientFactory ReferenceServiceClientFactory,
	repositoryServiceClientFactory RepositoryServiceClientFactory,
	repositoryBranchServiceClientFactory RepositoryBranchServiceClientFactory,
	repositoryTagServiceClientFactory RepositoryTagServiceClientFactory,
	repositoryCommitServiceClientFactory RepositoryCommitServiceClientFactory,
) bufsync.Handler {
	return newSyncHandler(
		logger,
		container,
		repo,
		createWithVisibility,
		syncServiceClientFactory,
		referenceServiceClientFactory,
		repositoryServiceClientFactory,
		repositoryBranchServiceClientFactory,
		repositoryTagServiceClientFactory,
		repositoryCommitServiceClientFactory,
	)
}
