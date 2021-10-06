// Copyright 2020-2021 Buf Technologies, Inc.
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

package bufapirepository

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/gen/proto/apiclient/buf/alpha/registry/v1alpha1/registryv1alpha1apiclient"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
)

type repositoryReader struct {
	repositoryServiceProvider registryv1alpha1apiclient.RepositoryServiceProvider
}

func newRepositoryReader(repositoryServiceProvider registryv1alpha1apiclient.RepositoryServiceProvider) *repositoryReader {
	return &repositoryReader{
		repositoryServiceProvider: repositoryServiceProvider,
	}
}

func (r *repositoryReader) GetRepository(ctx context.Context, modulePin bufmoduleref.ModulePin) (*registryv1alpha1.Repository, error) {
	repositoryService, err := r.repositoryServiceProvider.NewRepositoryService(ctx, modulePin.Remote())
	if err != nil {
		return nil, err
	}
	repository, err := repositoryService.GetRepositoryByFullName(ctx, fmt.Sprintf("%s/%s", modulePin.Owner(), modulePin.Repository()))
	if err != nil {
		return nil, err
	}
	return repository, nil
}
