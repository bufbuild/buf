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

// RepositoryService is the Repository service.
type RepositoryService interface {
	// GetRepository gets a repository by ID.
	GetRepository(ctx context.Context, id string) (repository *v1alpha1.Repository, err error)
	// GetRepositoryByFullName gets a repository by full name.
	GetRepositoryByFullName(ctx context.Context, fullName string) (repository *v1alpha1.Repository, err error)
	// ListRepositories lists all repositories.
	ListRepositories(
		ctx context.Context,
		pageSize uint32,
		pageToken string,
		reverse bool,
	) (repositories []*v1alpha1.Repository, nextPageToken string, err error)
	// ListUserRepositories lists all repositories belonging to a user.
	ListUserRepositories(
		ctx context.Context,
		userId string,
		pageSize uint32,
		pageToken string,
		reverse bool,
	) (repositories []*v1alpha1.Repository, nextPageToken string, err error)
	// ListUserRepositories lists all repositories a user can access.
	ListRepositoriesUserCanAccess(
		ctx context.Context,
		pageSize uint32,
		pageToken string,
		reverse bool,
	) (repositories []*v1alpha1.Repository, nextPageToken string, err error)
	// ListOrganizationRepositories lists all repositories for an organization.
	ListOrganizationRepositories(
		ctx context.Context,
		organizationId string,
		pageSize uint32,
		pageToken string,
		reverse bool,
	) (repositories []*v1alpha1.Repository, nextPageToken string, err error)
	// CreateRepositoryByFullName creates a new repository by full name.
	CreateRepositoryByFullName(
		ctx context.Context,
		fullName string,
		visibility v1alpha1.Visibility,
	) (repository *v1alpha1.Repository, err error)
	// DeleteRepository deletes a repository.
	DeleteRepository(ctx context.Context, id string) (err error)
	// DeleteRepositoryByFullName deletes a repository by full name.
	DeleteRepositoryByFullName(ctx context.Context, fullName string) (err error)
	// DeprecateRepositoryByName deprecates the repository.
	DeprecateRepositoryByName(
		ctx context.Context,
		ownerName string,
		repositoryName string,
		deprecationMessage string,
	) (repository *v1alpha1.Repository, err error)
	// UndeprecateRepositoryByName makes the repository not deprecated and removes any deprecation_message.
	UndeprecateRepositoryByName(
		ctx context.Context,
		ownerName string,
		repositoryName string,
	) (repository *v1alpha1.Repository, err error)
	// GetRepositoriesByFullName gets repositories by full name. Response order is unspecified.
	// Errors if any of the repositories don't exist or the caller does not have access to any of the repositories.
	GetRepositoriesByFullName(ctx context.Context, fullNames []string) (repositories []*v1alpha1.Repository, err error)
	// SetRepositoryContributor sets the role of a user in the repository.
	SetRepositoryContributor(
		ctx context.Context,
		repositoryId string,
		userId string,
		repositoryRole v1alpha1.RepositoryRole,
	) (err error)
	// ListRepositoryContributors returns the list of contributors that has an explicit role against the repository.
	// This does not include users who have implicit roles against the repository, unless they have also been
	// assigned a role explicitly.
	ListRepositoryContributors(
		ctx context.Context,
		repositoryId string,
		pageSize uint32,
		pageToken string,
		reverse bool,
	) (users []*v1alpha1.RepositoryContributor, nextPageToken string, err error)
}
