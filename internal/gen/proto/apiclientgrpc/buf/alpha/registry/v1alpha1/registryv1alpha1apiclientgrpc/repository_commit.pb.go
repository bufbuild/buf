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

// Code generated by protoc-gen-go-apiclientgrpc. DO NOT EDIT.

package registryv1alpha1apiclientgrpc

import (
	context "context"
	v1alpha1 "github.com/bufbuild/buf/internal/gen/proto/go/buf/alpha/registry/v1alpha1"
	zap "go.uber.org/zap"
)

type repositoryCommitService struct {
	logger          *zap.Logger
	client          v1alpha1.RepositoryCommitServiceClient
	contextModifier func(context.Context) context.Context
}

// ListRepositoryCommits lists the repository commits associated with a repository branch.
func (s *repositoryCommitService) ListRepositoryCommits(
	ctx context.Context,
	repositoryId string,
	repositoryBranchName string,
	pageSize uint32,
	pageToken string,
	reverse bool,
) (repositoryCommits []*v1alpha1.RepositoryCommit, nextPageToken string, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.ListRepositoryCommits(
		ctx,
		&v1alpha1.ListRepositoryCommitsRequest{
			RepositoryId:         repositoryId,
			RepositoryBranchName: repositoryBranchName,
			PageSize:             pageSize,
			PageToken:            pageToken,
			Reverse:              reverse,
		},
	)
	if err != nil {
		return nil, "", err
	}
	return response.RepositoryCommits, response.NextPageToken, nil
}
