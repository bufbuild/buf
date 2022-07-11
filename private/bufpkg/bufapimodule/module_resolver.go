// Copyright 2020-2022 Buf Technologies, Inc.
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

package bufapimodule

import (
	"context"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/gen/proto/apiclient/buf/alpha/registry/v1alpha1/registryv1alpha1apiclient"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/connect-go"
	"go.uber.org/zap"
)

type moduleResolver struct {
	logger                          *zap.Logger
	repositoryCommitServiceProvider registryv1alpha1apiclient.RepositoryCommitServiceProvider
}

func newModuleResolver(
	logger *zap.Logger,
	repositoryCommitServiceProvider registryv1alpha1apiclient.RepositoryCommitServiceProvider,
) *moduleResolver {
	return &moduleResolver{
		logger:                          logger,
		repositoryCommitServiceProvider: repositoryCommitServiceProvider,
	}
}

func (m *moduleResolver) GetModulePin(ctx context.Context, moduleReference bufmoduleref.ModuleReference) (bufmoduleref.ModulePin, error) {
	repositoryCommitService, err := m.repositoryCommitServiceProvider.NewRepositoryCommitService(ctx, moduleReference.Remote())
	if err != nil {
		return nil, err
	}
	repositoryCommit, err := repositoryCommitService.GetRepositoryCommitByReference(
		ctx,
		moduleReference.Owner(),
		moduleReference.Repository(),
		moduleReference.Reference(),
	)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			// Required by ModuleResolver interface spec
			return nil, storage.NewErrNotExist(moduleReference.String())
		}
		return nil, err
	}
	return bufmoduleref.NewModulePin(
		moduleReference.Remote(),
		moduleReference.Owner(),
		moduleReference.Repository(),
		bufmoduleref.Main,
		repositoryCommit.Name,
		repositoryCommit.CreateTime.AsTime(),
	)
}
