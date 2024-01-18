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
	"github.com/bufbuild/buf/private/buf/bufsync"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/git"
	"go.uber.org/zap"
)

// NewHandle returns a new bufsync.Handler that handles requests by communicating with a BSR instance.
func NewHandler(
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
