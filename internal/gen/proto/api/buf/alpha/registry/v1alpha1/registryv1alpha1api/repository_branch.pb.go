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

// Code generated by protoc-gen-go-api. DO NOT EDIT.

package registryv1alpha1api

import (
	context "context"
	v1alpha1 "github.com/bufbuild/buf/internal/gen/proto/go/buf/alpha/registry/v1alpha1"
)

// RepositoryBranchService is the Repository branch service.
// All methods on the Repository branch service require authentication.
type RepositoryBranchService interface {
	// CreateRepositoryBranch creates a new repository branch.
	CreateRepositoryBranch(
		ctx context.Context,
		repositoryId string,
		name string,
		parentBranch string,
	) (repositoryBranch *v1alpha1.RepositoryBranch, err error)
	// ListRepositoryBranches lists the repository branches associated with a Repository.
	ListRepositoryBranches(
		ctx context.Context,
		repositoryId string,
		pageSize uint32,
		pageToken string,
		reverse bool,
	) (repositoryBranches []*v1alpha1.RepositoryBranch, nextPageToken string, err error)
}
