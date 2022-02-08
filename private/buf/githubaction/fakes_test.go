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

package githubaction

import (
	"context"
	"testing"

	"github.com/bufbuild/buf/private/gen/proto/api/buf/alpha/registry/v1alpha1/registryv1alpha1api"
	"github.com/bufbuild/buf/private/gen/proto/apiclient/buf/alpha/registry/v1alpha1/registryv1alpha1apiclient"
	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
)

type fakePushService struct {
	registryv1alpha1api.PushService
	t *testing.T

	push func(
		ctx context.Context,
		owner string,
		repository string,
		branch string,
		module *modulev1alpha1.Module,
		tags []string,
		tracks []string,
	) (localModulePin *registryv1alpha1.LocalModulePin, err error)
}

func (f *fakePushService) Push(
	ctx context.Context,
	owner string,
	repository string,
	branch string,
	module *modulev1alpha1.Module,
	tags []string,
	tracks []string,
) (localModulePin *registryv1alpha1.LocalModulePin, err error) {
	if f.push == nil {
		f.t.Error("unexpected call to Push")
		return nil, nil
	}
	return f.push(ctx, owner, repository, branch, module, tags, tracks)
}

type fakeReferenceService struct {
	registryv1alpha1api.ReferenceService
	t *testing.T

	getReferenceByName func(
		ctx context.Context,
		name string,
		owner string,
		repositoryName string,
	) (reference *registryv1alpha1.Reference, err error)
}

func (f *fakeReferenceService) GetReferenceByName(
	ctx context.Context,
	name string,
	owner string,
	repositoryName string,
) (reference *registryv1alpha1.Reference, err error) {
	if f.getReferenceByName == nil {
		f.t.Error("unexpected call to GetReferenceByName")
		return nil, nil
	}
	return f.getReferenceByName(ctx, name, owner, repositoryName)
}

type fakeRepositoryTrackCommitService struct {
	registryv1alpha1api.RepositoryTrackCommitService
	t *testing.T

	getRepositoryTrackCommitByRepositoryCommit func(
		ctx context.Context,
		repositoryTrackId string,
		repositoryCommitId string,
	) (repositoryTrackCommit *registryv1alpha1.RepositoryTrackCommit, err error)

	createRepositoryTrackCommit func(
		ctx context.Context,
		repositoryTrackId string,
		repositoryCommit string,
	) (repositoryTrackCommit *registryv1alpha1.RepositoryTrackCommit, err error)
}

func (f *fakeRepositoryTrackCommitService) GetRepositoryTrackCommitByRepositoryCommit(
	ctx context.Context,
	repositoryTrackID string,
	repositoryCommitID string,
) (repositoryTrackCommit *registryv1alpha1.RepositoryTrackCommit, err error) {
	if f.getRepositoryTrackCommitByRepositoryCommit == nil {
		f.t.Error("unexpected call to GetRepositoryTrackCommitByRepositoryCommit")
		return nil, nil
	}
	return f.getRepositoryTrackCommitByRepositoryCommit(ctx, repositoryTrackID, repositoryCommitID)
}

func (f *fakeRepositoryTrackCommitService) CreateRepositoryTrackCommit(
	ctx context.Context,
	repositoryTrackID string,
	repositoryCommit string,
) (repositoryTrackCommit *registryv1alpha1.RepositoryTrackCommit, err error) {
	if f.createRepositoryTrackCommit == nil {
		f.t.Error("unexpected call to CreateRepositoryTrackCommit")
		return nil, nil
	}
	return f.createRepositoryTrackCommit(ctx, repositoryTrackID, repositoryCommit)
}

type fakeRepositoryCommitService struct {
	registryv1alpha1api.RepositoryCommitService
	t *testing.T

	getRepositoryCommitByReference func(
		ctx context.Context,
		repositoryOwner string,
		repositoryName string,
		reference string,
	) (*registryv1alpha1.RepositoryCommit, error)
}

func (f *fakeRepositoryCommitService) GetRepositoryCommitByReference(
	ctx context.Context,
	repositoryOwner string,
	repositoryName string,
	reference string,
) (repositoryCommit *registryv1alpha1.RepositoryCommit, err error) {
	if f.getRepositoryCommitByReference == nil {
		f.t.Error("unexpected call to GetRepositoryCommitByReference")
		return nil, nil
	}
	return f.getRepositoryCommitByReference(ctx, repositoryOwner, repositoryName, reference)
}

type fakeRegistryProvider struct {
	registryv1alpha1apiclient.Provider
	t                            *testing.T
	pushService                  fakePushService
	referenceService             fakeReferenceService
	repositoryTrackCommitService fakeRepositoryTrackCommitService
	repositoryCommitService      fakeRepositoryCommitService
}

func (f fakeRegistryProvider) NewPushService(context.Context, string) (registryv1alpha1api.PushService, error) {
	if f.pushService.t == nil {
		f.pushService.t = f.t
	}
	return &f.pushService, nil
}

func (f fakeRegistryProvider) NewReferenceService(
	context.Context,
	string,
) (registryv1alpha1api.ReferenceService, error) {
	if f.referenceService.t == nil {
		f.referenceService.t = f.t
	}
	return &f.referenceService, nil
}

func (f fakeRegistryProvider) NewRepositoryTrackCommitService(
	context.Context,
	string,
) (registryv1alpha1api.RepositoryTrackCommitService, error) {
	if f.repositoryTrackCommitService.t == nil {
		f.repositoryTrackCommitService.t = f.t
	}
	return &f.repositoryTrackCommitService, nil
}

func (f fakeRegistryProvider) NewRepositoryCommitService(
	context.Context,
	string,
) (registryv1alpha1api.RepositoryCommitService, error) {
	if f.repositoryCommitService.t == nil {
		f.repositoryCommitService.t = f.t
	}
	return &f.repositoryCommitService, nil
}
