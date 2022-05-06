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

// Code generated by protoc-gen-go-apiclientconnect. DO NOT EDIT.

package registryv1alpha1apiclientconnect

import (
	context "context"
	registryv1alpha1api "github.com/bufbuild/buf/private/gen/proto/api/buf/alpha/registry/v1alpha1/registryv1alpha1api"
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	connect_go "github.com/bufbuild/connect-go"
	zap "go.uber.org/zap"
)

type repositoryBranchService struct {
	logger          *zap.Logger
	client          registryv1alpha1api.RepositoryBranchService
	contextModifier func(context.Context) context.Context
}

// CreateRepositoryBranch creates a new repository branch.
func (s *repositoryBranchService) CreateRepositoryBranch(
	ctx context.Context,
	repositoryId string,
	name string,
	parentBranch string,
) (repositoryBranch *v1alpha1.RepositoryBranch, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.CreateRepositoryBranch(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.CreateRepositoryBranchRequest{
				RepositoryId: repositoryId,
				Name:         name,
				ParentBranch: parentBranch,
			}),
	)
	if err != nil {
		return nil, err
	}
	return response.Msg.RepositoryBranch, nil
}

// ListRepositoryBranches lists the repository branches associated with a Repository.
func (s *repositoryBranchService) ListRepositoryBranches(
	ctx context.Context,
	repositoryId string,
	pageSize uint32,
	pageToken string,
	reverse bool,
) (repositoryBranches []*v1alpha1.RepositoryBranch, nextPageToken string, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.ListRepositoryBranches(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.ListRepositoryBranchesRequest{
				RepositoryId: repositoryId,
				PageSize:     pageSize,
				PageToken:    pageToken,
				Reverse:      reverse,
			}),
	)
	if err != nil {
		return nil, "", err
	}
	return response.Msg.RepositoryBranches, response.Msg.NextPageToken, nil
}
