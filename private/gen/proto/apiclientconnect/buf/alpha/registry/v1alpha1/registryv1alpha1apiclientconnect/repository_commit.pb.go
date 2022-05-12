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
	registryv1alpha1connect "github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	connect_go "github.com/bufbuild/connect-go"
	zap "go.uber.org/zap"
)

type repositoryCommitServiceClient struct {
	logger          *zap.Logger
	client          registryv1alpha1connect.RepositoryCommitServiceClient
	contextModifier func(context.Context) context.Context
}

func newRepositoryCommitServiceClient(
	httpClient connect_go.HTTPClient,
	address string,
	contextModifier func(context.Context) context.Context,
	options ...connect_go.ClientOption,
) registryv1alpha1api.RepositoryCommitService {
	return &repositoryCommitServiceClient{
		client: registryv1alpha1connect.NewRepositoryCommitServiceClient(
			httpClient,
			address,
			options...,
		),
		contextModifier: contextModifier,
	}
}

// ListRepositoryCommitsByBranch lists the repository commits associated
// with a repository branch on a repository, ordered by their create time.
func (s *repositoryCommitServiceClient) ListRepositoryCommitsByBranch(
	ctx context.Context,
	repositoryOwner string,
	repositoryName string,
	repositoryBranchName string,
	pageSize uint32,
	pageToken string,
	reverse bool,
) (repositoryCommits []*v1alpha1.RepositoryCommit, nextPageToken string, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.ListRepositoryCommitsByBranch(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.ListRepositoryCommitsByBranchRequest{
				RepositoryOwner:      repositoryOwner,
				RepositoryName:       repositoryName,
				RepositoryBranchName: repositoryBranchName,
				PageSize:             pageSize,
				PageToken:            pageToken,
				Reverse:              reverse,
			}),
	)
	if err != nil {
		return nil, "", err
	}
	return response.Msg.RepositoryCommits, response.Msg.NextPageToken, nil
}

// ListRepositoryCommitsByReference returns repository commits up-to and including
// the provided reference.
func (s *repositoryCommitServiceClient) ListRepositoryCommitsByReference(
	ctx context.Context,
	repositoryOwner string,
	repositoryName string,
	reference string,
	pageSize uint32,
	pageToken string,
	reverse bool,
) (repositoryCommits []*v1alpha1.RepositoryCommit, nextPageToken string, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.ListRepositoryCommitsByReference(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.ListRepositoryCommitsByReferenceRequest{
				RepositoryOwner: repositoryOwner,
				RepositoryName:  repositoryName,
				Reference:       reference,
				PageSize:        pageSize,
				PageToken:       pageToken,
				Reverse:         reverse,
			}),
	)
	if err != nil {
		return nil, "", err
	}
	return response.Msg.RepositoryCommits, response.Msg.NextPageToken, nil
}

// ListRepositoryCommitsOnTrack returns repository commits up-to and including
// the provided reference.
func (s *repositoryCommitServiceClient) ListRepositoryCommitsOnTrack(
	ctx context.Context,
	repositoryOwner string,
	repositoryName string,
	repositoryTrackName string,
	reference string,
	pageSize uint32,
	pageToken string,
	reverse bool,
) (repositoryCommits []*v1alpha1.RepositoryCommit, nextPageToken string, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.ListRepositoryCommitsOnTrack(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.ListRepositoryCommitsOnTrackRequest{
				RepositoryOwner:     repositoryOwner,
				RepositoryName:      repositoryName,
				RepositoryTrackName: repositoryTrackName,
				Reference:           reference,
				PageSize:            pageSize,
				PageToken:           pageToken,
				Reverse:             reverse,
			}),
	)
	if err != nil {
		return nil, "", err
	}
	return response.Msg.RepositoryCommits, response.Msg.NextPageToken, nil
}

// GetRepositoryCommitByReference returns the repository commit matching
// the provided reference, if it exists.
func (s *repositoryCommitServiceClient) GetRepositoryCommitByReference(
	ctx context.Context,
	repositoryOwner string,
	repositoryName string,
	reference string,
) (repositoryCommit *v1alpha1.RepositoryCommit, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.GetRepositoryCommitByReference(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.GetRepositoryCommitByReferenceRequest{
				RepositoryOwner: repositoryOwner,
				RepositoryName:  repositoryName,
				Reference:       reference,
			}),
	)
	if err != nil {
		return nil, err
	}
	return response.Msg.RepositoryCommit, nil
}

// GetRepositoryCommitBySequenceId returns the repository commit matching
// the provided sequence ID and branch, if it exists.
func (s *repositoryCommitServiceClient) GetRepositoryCommitBySequenceId(
	ctx context.Context,
	repositoryOwner string,
	repositoryName string,
	repositoryBranchName string,
	commitSequenceId int64,
) (repositoryCommit *v1alpha1.RepositoryCommit, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.GetRepositoryCommitBySequenceId(
		ctx,
		connect_go.NewRequest(
			&v1alpha1.GetRepositoryCommitBySequenceIdRequest{
				RepositoryOwner:      repositoryOwner,
				RepositoryName:       repositoryName,
				RepositoryBranchName: repositoryBranchName,
				CommitSequenceId:     commitSequenceId,
			}),
	)
	if err != nil {
		return nil, err
	}
	return response.Msg.RepositoryCommit, nil
}
