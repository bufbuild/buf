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

// Code generated by protoc-gen-go-api. DO NOT EDIT.

package registryv1alpha1api

import (
	context "context"
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
)

type RepositoryTrackService interface {
	// CreateRepositoryTrack creates a new repository track.
	CreateRepositoryTrack(
		ctx context.Context,
		repositoryId string,
		name string,
	) (repositoryTrack *v1alpha1.RepositoryTrack, err error)
	// ListRepositoryTracks lists the repository tracks associated with a repository.
	ListRepositoryTracks(
		ctx context.Context,
		repositoryId string,
		pageSize uint32,
		pageToken string,
		reverse bool,
	) (repositoryTracks []*v1alpha1.RepositoryTrack, nextPageToken string, err error)
	// DeleteRepositoryTrackByName deletes a repository track by name.
	DeleteRepositoryTrackByName(
		ctx context.Context,
		ownerName string,
		repositoryName string,
		name string,
	) (err error)
	// GetRepositoryTrackByName gets a repository track by name.
	GetRepositoryTrackByName(
		ctx context.Context,
		ownerName string,
		repositoryName string,
		name string,
	) (repositoryTrack *v1alpha1.RepositoryTrack, err error)
	// ListRepositoryTracksByRepositoryCommit lists the repository tracks associated with a repository commit.
	ListRepositoryTracksByRepositoryCommit(
		ctx context.Context,
		repositoryId string,
		commit string,
		pageSize uint32,
		pageToken string,
		reverse bool,
	) (repositoryTracks []*v1alpha1.RepositoryTrack, nextPageToken string, err error)
}
