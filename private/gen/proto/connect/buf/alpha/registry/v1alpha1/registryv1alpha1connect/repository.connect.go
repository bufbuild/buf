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

// Code generated by protoc-gen-connect-go. DO NOT EDIT.
//
// Source: buf/alpha/registry/v1alpha1/repository.proto

package registryv1alpha1connect

import (
	context "context"
	errors "errors"
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	connect_go "github.com/bufbuild/connect-go"
	http "net/http"
	strings "strings"
)

// This is a compile-time assertion to ensure that this generated file and the connect package are
// compatible. If you get a compiler error that this constant is not defined, this code was
// generated with a version of connect newer than the one compiled into your binary. You can fix the
// problem by either regenerating this code with an older version of connect or updating the connect
// version compiled into your binary.
const _ = connect_go.IsAtLeastVersion0_1_0

const (
	// RepositoryServiceName is the fully-qualified name of the RepositoryService service.
	RepositoryServiceName = "buf.alpha.registry.v1alpha1.RepositoryService"
)

// RepositoryServiceClient is a client for the buf.alpha.registry.v1alpha1.RepositoryService
// service.
type RepositoryServiceClient interface {
	// GetRepository gets a repository by ID.
	GetRepository(context.Context, *connect_go.Request[v1alpha1.GetRepositoryRequest]) (*connect_go.Response[v1alpha1.GetRepositoryResponse], error)
	// GetRepositoryByFullName gets a repository by full name.
	GetRepositoryByFullName(context.Context, *connect_go.Request[v1alpha1.GetRepositoryByFullNameRequest]) (*connect_go.Response[v1alpha1.GetRepositoryByFullNameResponse], error)
	// ListRepositories lists all repositories.
	ListRepositories(context.Context, *connect_go.Request[v1alpha1.ListRepositoriesRequest]) (*connect_go.Response[v1alpha1.ListRepositoriesResponse], error)
	// ListUserRepositories lists all repositories belonging to a user.
	ListUserRepositories(context.Context, *connect_go.Request[v1alpha1.ListUserRepositoriesRequest]) (*connect_go.Response[v1alpha1.ListUserRepositoriesResponse], error)
	// ListRepositoriesUserCanAccess lists all repositories a user can access.
	ListRepositoriesUserCanAccess(context.Context, *connect_go.Request[v1alpha1.ListRepositoriesUserCanAccessRequest]) (*connect_go.Response[v1alpha1.ListRepositoriesUserCanAccessResponse], error)
	// ListOrganizationRepositories lists all repositories for an organization.
	ListOrganizationRepositories(context.Context, *connect_go.Request[v1alpha1.ListOrganizationRepositoriesRequest]) (*connect_go.Response[v1alpha1.ListOrganizationRepositoriesResponse], error)
	// CreateRepositoryByFullName creates a new repository by full name.
	CreateRepositoryByFullName(context.Context, *connect_go.Request[v1alpha1.CreateRepositoryByFullNameRequest]) (*connect_go.Response[v1alpha1.CreateRepositoryByFullNameResponse], error)
	// DeleteRepository deletes a repository.
	DeleteRepository(context.Context, *connect_go.Request[v1alpha1.DeleteRepositoryRequest]) (*connect_go.Response[v1alpha1.DeleteRepositoryResponse], error)
	// DeleteRepositoryByFullName deletes a repository by full name.
	DeleteRepositoryByFullName(context.Context, *connect_go.Request[v1alpha1.DeleteRepositoryByFullNameRequest]) (*connect_go.Response[v1alpha1.DeleteRepositoryByFullNameResponse], error)
	// DeprecateRepositoryByName deprecates the repository.
	DeprecateRepositoryByName(context.Context, *connect_go.Request[v1alpha1.DeprecateRepositoryByNameRequest]) (*connect_go.Response[v1alpha1.DeprecateRepositoryByNameResponse], error)
	// UndeprecateRepositoryByName makes the repository not deprecated and removes any deprecation_message.
	UndeprecateRepositoryByName(context.Context, *connect_go.Request[v1alpha1.UndeprecateRepositoryByNameRequest]) (*connect_go.Response[v1alpha1.UndeprecateRepositoryByNameResponse], error)
	// GetRepositoriesByFullName gets repositories by full name. Response order is unspecified.
	// Errors if any of the repositories don't exist or the caller does not have access to any of the repositories.
	GetRepositoriesByFullName(context.Context, *connect_go.Request[v1alpha1.GetRepositoriesByFullNameRequest]) (*connect_go.Response[v1alpha1.GetRepositoriesByFullNameResponse], error)
	// SetRepositoryContributor sets the role of a user in the repository.
	SetRepositoryContributor(context.Context, *connect_go.Request[v1alpha1.SetRepositoryContributorRequest]) (*connect_go.Response[v1alpha1.SetRepositoryContributorResponse], error)
	// ListRepositoryContributors returns the list of contributors that has an explicit role against the repository.
	// This does not include users who have implicit roles against the repository, unless they have also been
	// assigned a role explicitly.
	ListRepositoryContributors(context.Context, *connect_go.Request[v1alpha1.ListRepositoryContributorsRequest]) (*connect_go.Response[v1alpha1.ListRepositoryContributorsResponse], error)
	// GetRepositoryContributor returns the contributor information of a user in a repository.
	GetRepositoryContributor(context.Context, *connect_go.Request[v1alpha1.GetRepositoryContributorRequest]) (*connect_go.Response[v1alpha1.GetRepositoryContributorResponse], error)
	// GetRepositorySettings gets the settings of a repository.
	GetRepositorySettings(context.Context, *connect_go.Request[v1alpha1.GetRepositorySettingsRequest]) (*connect_go.Response[v1alpha1.GetRepositorySettingsResponse], error)
	// UpdateRepositorySettingsByName updates the settings of a repository.
	UpdateRepositorySettingsByName(context.Context, *connect_go.Request[v1alpha1.UpdateRepositorySettingsByNameRequest]) (*connect_go.Response[v1alpha1.UpdateRepositorySettingsByNameResponse], error)
	// GetRepositoriesMetadata gets the metadata of the repositories in the request, the length of repositories in the
	// request should match the length of the metadata in the response, and the order of repositories in the response
	// should match the order of the metadata in the request.
	GetRepositoriesMetadata(context.Context, *connect_go.Request[v1alpha1.GetRepositoriesMetadataRequest]) (*connect_go.Response[v1alpha1.GetRepositoriesMetadataResponse], error)
}

// NewRepositoryServiceClient constructs a client for the
// buf.alpha.registry.v1alpha1.RepositoryService service. By default, it uses the Connect protocol
// with the binary Protobuf Codec, asks for gzipped responses, and sends uncompressed requests. To
// use the gRPC or gRPC-Web protocols, supply the connect.WithGRPC() or connect.WithGRPCWeb()
// options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewRepositoryServiceClient(httpClient connect_go.HTTPClient, baseURL string, opts ...connect_go.ClientOption) RepositoryServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &repositoryServiceClient{
		getRepository: connect_go.NewClient[v1alpha1.GetRepositoryRequest, v1alpha1.GetRepositoryResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.RepositoryService/GetRepository",
			opts...,
		),
		getRepositoryByFullName: connect_go.NewClient[v1alpha1.GetRepositoryByFullNameRequest, v1alpha1.GetRepositoryByFullNameResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.RepositoryService/GetRepositoryByFullName",
			opts...,
		),
		listRepositories: connect_go.NewClient[v1alpha1.ListRepositoriesRequest, v1alpha1.ListRepositoriesResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.RepositoryService/ListRepositories",
			opts...,
		),
		listUserRepositories: connect_go.NewClient[v1alpha1.ListUserRepositoriesRequest, v1alpha1.ListUserRepositoriesResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.RepositoryService/ListUserRepositories",
			opts...,
		),
		listRepositoriesUserCanAccess: connect_go.NewClient[v1alpha1.ListRepositoriesUserCanAccessRequest, v1alpha1.ListRepositoriesUserCanAccessResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.RepositoryService/ListRepositoriesUserCanAccess",
			opts...,
		),
		listOrganizationRepositories: connect_go.NewClient[v1alpha1.ListOrganizationRepositoriesRequest, v1alpha1.ListOrganizationRepositoriesResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.RepositoryService/ListOrganizationRepositories",
			opts...,
		),
		createRepositoryByFullName: connect_go.NewClient[v1alpha1.CreateRepositoryByFullNameRequest, v1alpha1.CreateRepositoryByFullNameResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.RepositoryService/CreateRepositoryByFullName",
			opts...,
		),
		deleteRepository: connect_go.NewClient[v1alpha1.DeleteRepositoryRequest, v1alpha1.DeleteRepositoryResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.RepositoryService/DeleteRepository",
			opts...,
		),
		deleteRepositoryByFullName: connect_go.NewClient[v1alpha1.DeleteRepositoryByFullNameRequest, v1alpha1.DeleteRepositoryByFullNameResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.RepositoryService/DeleteRepositoryByFullName",
			opts...,
		),
		deprecateRepositoryByName: connect_go.NewClient[v1alpha1.DeprecateRepositoryByNameRequest, v1alpha1.DeprecateRepositoryByNameResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.RepositoryService/DeprecateRepositoryByName",
			opts...,
		),
		undeprecateRepositoryByName: connect_go.NewClient[v1alpha1.UndeprecateRepositoryByNameRequest, v1alpha1.UndeprecateRepositoryByNameResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.RepositoryService/UndeprecateRepositoryByName",
			opts...,
		),
		getRepositoriesByFullName: connect_go.NewClient[v1alpha1.GetRepositoriesByFullNameRequest, v1alpha1.GetRepositoriesByFullNameResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.RepositoryService/GetRepositoriesByFullName",
			opts...,
		),
		setRepositoryContributor: connect_go.NewClient[v1alpha1.SetRepositoryContributorRequest, v1alpha1.SetRepositoryContributorResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.RepositoryService/SetRepositoryContributor",
			opts...,
		),
		listRepositoryContributors: connect_go.NewClient[v1alpha1.ListRepositoryContributorsRequest, v1alpha1.ListRepositoryContributorsResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.RepositoryService/ListRepositoryContributors",
			opts...,
		),
		getRepositoryContributor: connect_go.NewClient[v1alpha1.GetRepositoryContributorRequest, v1alpha1.GetRepositoryContributorResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.RepositoryService/GetRepositoryContributor",
			opts...,
		),
		getRepositorySettings: connect_go.NewClient[v1alpha1.GetRepositorySettingsRequest, v1alpha1.GetRepositorySettingsResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.RepositoryService/GetRepositorySettings",
			opts...,
		),
		updateRepositorySettingsByName: connect_go.NewClient[v1alpha1.UpdateRepositorySettingsByNameRequest, v1alpha1.UpdateRepositorySettingsByNameResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.RepositoryService/UpdateRepositorySettingsByName",
			opts...,
		),
		getRepositoriesMetadata: connect_go.NewClient[v1alpha1.GetRepositoriesMetadataRequest, v1alpha1.GetRepositoriesMetadataResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.RepositoryService/GetRepositoriesMetadata",
			opts...,
		),
	}
}

// repositoryServiceClient implements RepositoryServiceClient.
type repositoryServiceClient struct {
	getRepository                  *connect_go.Client[v1alpha1.GetRepositoryRequest, v1alpha1.GetRepositoryResponse]
	getRepositoryByFullName        *connect_go.Client[v1alpha1.GetRepositoryByFullNameRequest, v1alpha1.GetRepositoryByFullNameResponse]
	listRepositories               *connect_go.Client[v1alpha1.ListRepositoriesRequest, v1alpha1.ListRepositoriesResponse]
	listUserRepositories           *connect_go.Client[v1alpha1.ListUserRepositoriesRequest, v1alpha1.ListUserRepositoriesResponse]
	listRepositoriesUserCanAccess  *connect_go.Client[v1alpha1.ListRepositoriesUserCanAccessRequest, v1alpha1.ListRepositoriesUserCanAccessResponse]
	listOrganizationRepositories   *connect_go.Client[v1alpha1.ListOrganizationRepositoriesRequest, v1alpha1.ListOrganizationRepositoriesResponse]
	createRepositoryByFullName     *connect_go.Client[v1alpha1.CreateRepositoryByFullNameRequest, v1alpha1.CreateRepositoryByFullNameResponse]
	deleteRepository               *connect_go.Client[v1alpha1.DeleteRepositoryRequest, v1alpha1.DeleteRepositoryResponse]
	deleteRepositoryByFullName     *connect_go.Client[v1alpha1.DeleteRepositoryByFullNameRequest, v1alpha1.DeleteRepositoryByFullNameResponse]
	deprecateRepositoryByName      *connect_go.Client[v1alpha1.DeprecateRepositoryByNameRequest, v1alpha1.DeprecateRepositoryByNameResponse]
	undeprecateRepositoryByName    *connect_go.Client[v1alpha1.UndeprecateRepositoryByNameRequest, v1alpha1.UndeprecateRepositoryByNameResponse]
	getRepositoriesByFullName      *connect_go.Client[v1alpha1.GetRepositoriesByFullNameRequest, v1alpha1.GetRepositoriesByFullNameResponse]
	setRepositoryContributor       *connect_go.Client[v1alpha1.SetRepositoryContributorRequest, v1alpha1.SetRepositoryContributorResponse]
	listRepositoryContributors     *connect_go.Client[v1alpha1.ListRepositoryContributorsRequest, v1alpha1.ListRepositoryContributorsResponse]
	getRepositoryContributor       *connect_go.Client[v1alpha1.GetRepositoryContributorRequest, v1alpha1.GetRepositoryContributorResponse]
	getRepositorySettings          *connect_go.Client[v1alpha1.GetRepositorySettingsRequest, v1alpha1.GetRepositorySettingsResponse]
	updateRepositorySettingsByName *connect_go.Client[v1alpha1.UpdateRepositorySettingsByNameRequest, v1alpha1.UpdateRepositorySettingsByNameResponse]
	getRepositoriesMetadata        *connect_go.Client[v1alpha1.GetRepositoriesMetadataRequest, v1alpha1.GetRepositoriesMetadataResponse]
}

// GetRepository calls buf.alpha.registry.v1alpha1.RepositoryService.GetRepository.
func (c *repositoryServiceClient) GetRepository(ctx context.Context, req *connect_go.Request[v1alpha1.GetRepositoryRequest]) (*connect_go.Response[v1alpha1.GetRepositoryResponse], error) {
	return c.getRepository.CallUnary(ctx, req)
}

// GetRepositoryByFullName calls
// buf.alpha.registry.v1alpha1.RepositoryService.GetRepositoryByFullName.
func (c *repositoryServiceClient) GetRepositoryByFullName(ctx context.Context, req *connect_go.Request[v1alpha1.GetRepositoryByFullNameRequest]) (*connect_go.Response[v1alpha1.GetRepositoryByFullNameResponse], error) {
	return c.getRepositoryByFullName.CallUnary(ctx, req)
}

// ListRepositories calls buf.alpha.registry.v1alpha1.RepositoryService.ListRepositories.
func (c *repositoryServiceClient) ListRepositories(ctx context.Context, req *connect_go.Request[v1alpha1.ListRepositoriesRequest]) (*connect_go.Response[v1alpha1.ListRepositoriesResponse], error) {
	return c.listRepositories.CallUnary(ctx, req)
}

// ListUserRepositories calls buf.alpha.registry.v1alpha1.RepositoryService.ListUserRepositories.
func (c *repositoryServiceClient) ListUserRepositories(ctx context.Context, req *connect_go.Request[v1alpha1.ListUserRepositoriesRequest]) (*connect_go.Response[v1alpha1.ListUserRepositoriesResponse], error) {
	return c.listUserRepositories.CallUnary(ctx, req)
}

// ListRepositoriesUserCanAccess calls
// buf.alpha.registry.v1alpha1.RepositoryService.ListRepositoriesUserCanAccess.
func (c *repositoryServiceClient) ListRepositoriesUserCanAccess(ctx context.Context, req *connect_go.Request[v1alpha1.ListRepositoriesUserCanAccessRequest]) (*connect_go.Response[v1alpha1.ListRepositoriesUserCanAccessResponse], error) {
	return c.listRepositoriesUserCanAccess.CallUnary(ctx, req)
}

// ListOrganizationRepositories calls
// buf.alpha.registry.v1alpha1.RepositoryService.ListOrganizationRepositories.
func (c *repositoryServiceClient) ListOrganizationRepositories(ctx context.Context, req *connect_go.Request[v1alpha1.ListOrganizationRepositoriesRequest]) (*connect_go.Response[v1alpha1.ListOrganizationRepositoriesResponse], error) {
	return c.listOrganizationRepositories.CallUnary(ctx, req)
}

// CreateRepositoryByFullName calls
// buf.alpha.registry.v1alpha1.RepositoryService.CreateRepositoryByFullName.
func (c *repositoryServiceClient) CreateRepositoryByFullName(ctx context.Context, req *connect_go.Request[v1alpha1.CreateRepositoryByFullNameRequest]) (*connect_go.Response[v1alpha1.CreateRepositoryByFullNameResponse], error) {
	return c.createRepositoryByFullName.CallUnary(ctx, req)
}

// DeleteRepository calls buf.alpha.registry.v1alpha1.RepositoryService.DeleteRepository.
func (c *repositoryServiceClient) DeleteRepository(ctx context.Context, req *connect_go.Request[v1alpha1.DeleteRepositoryRequest]) (*connect_go.Response[v1alpha1.DeleteRepositoryResponse], error) {
	return c.deleteRepository.CallUnary(ctx, req)
}

// DeleteRepositoryByFullName calls
// buf.alpha.registry.v1alpha1.RepositoryService.DeleteRepositoryByFullName.
func (c *repositoryServiceClient) DeleteRepositoryByFullName(ctx context.Context, req *connect_go.Request[v1alpha1.DeleteRepositoryByFullNameRequest]) (*connect_go.Response[v1alpha1.DeleteRepositoryByFullNameResponse], error) {
	return c.deleteRepositoryByFullName.CallUnary(ctx, req)
}

// DeprecateRepositoryByName calls
// buf.alpha.registry.v1alpha1.RepositoryService.DeprecateRepositoryByName.
func (c *repositoryServiceClient) DeprecateRepositoryByName(ctx context.Context, req *connect_go.Request[v1alpha1.DeprecateRepositoryByNameRequest]) (*connect_go.Response[v1alpha1.DeprecateRepositoryByNameResponse], error) {
	return c.deprecateRepositoryByName.CallUnary(ctx, req)
}

// UndeprecateRepositoryByName calls
// buf.alpha.registry.v1alpha1.RepositoryService.UndeprecateRepositoryByName.
func (c *repositoryServiceClient) UndeprecateRepositoryByName(ctx context.Context, req *connect_go.Request[v1alpha1.UndeprecateRepositoryByNameRequest]) (*connect_go.Response[v1alpha1.UndeprecateRepositoryByNameResponse], error) {
	return c.undeprecateRepositoryByName.CallUnary(ctx, req)
}

// GetRepositoriesByFullName calls
// buf.alpha.registry.v1alpha1.RepositoryService.GetRepositoriesByFullName.
func (c *repositoryServiceClient) GetRepositoriesByFullName(ctx context.Context, req *connect_go.Request[v1alpha1.GetRepositoriesByFullNameRequest]) (*connect_go.Response[v1alpha1.GetRepositoriesByFullNameResponse], error) {
	return c.getRepositoriesByFullName.CallUnary(ctx, req)
}

// SetRepositoryContributor calls
// buf.alpha.registry.v1alpha1.RepositoryService.SetRepositoryContributor.
func (c *repositoryServiceClient) SetRepositoryContributor(ctx context.Context, req *connect_go.Request[v1alpha1.SetRepositoryContributorRequest]) (*connect_go.Response[v1alpha1.SetRepositoryContributorResponse], error) {
	return c.setRepositoryContributor.CallUnary(ctx, req)
}

// ListRepositoryContributors calls
// buf.alpha.registry.v1alpha1.RepositoryService.ListRepositoryContributors.
func (c *repositoryServiceClient) ListRepositoryContributors(ctx context.Context, req *connect_go.Request[v1alpha1.ListRepositoryContributorsRequest]) (*connect_go.Response[v1alpha1.ListRepositoryContributorsResponse], error) {
	return c.listRepositoryContributors.CallUnary(ctx, req)
}

// GetRepositoryContributor calls
// buf.alpha.registry.v1alpha1.RepositoryService.GetRepositoryContributor.
func (c *repositoryServiceClient) GetRepositoryContributor(ctx context.Context, req *connect_go.Request[v1alpha1.GetRepositoryContributorRequest]) (*connect_go.Response[v1alpha1.GetRepositoryContributorResponse], error) {
	return c.getRepositoryContributor.CallUnary(ctx, req)
}

// GetRepositorySettings calls buf.alpha.registry.v1alpha1.RepositoryService.GetRepositorySettings.
func (c *repositoryServiceClient) GetRepositorySettings(ctx context.Context, req *connect_go.Request[v1alpha1.GetRepositorySettingsRequest]) (*connect_go.Response[v1alpha1.GetRepositorySettingsResponse], error) {
	return c.getRepositorySettings.CallUnary(ctx, req)
}

// UpdateRepositorySettingsByName calls
// buf.alpha.registry.v1alpha1.RepositoryService.UpdateRepositorySettingsByName.
func (c *repositoryServiceClient) UpdateRepositorySettingsByName(ctx context.Context, req *connect_go.Request[v1alpha1.UpdateRepositorySettingsByNameRequest]) (*connect_go.Response[v1alpha1.UpdateRepositorySettingsByNameResponse], error) {
	return c.updateRepositorySettingsByName.CallUnary(ctx, req)
}

// GetRepositoriesMetadata calls
// buf.alpha.registry.v1alpha1.RepositoryService.GetRepositoriesMetadata.
func (c *repositoryServiceClient) GetRepositoriesMetadata(ctx context.Context, req *connect_go.Request[v1alpha1.GetRepositoriesMetadataRequest]) (*connect_go.Response[v1alpha1.GetRepositoriesMetadataResponse], error) {
	return c.getRepositoriesMetadata.CallUnary(ctx, req)
}

// RepositoryServiceHandler is an implementation of the
// buf.alpha.registry.v1alpha1.RepositoryService service.
type RepositoryServiceHandler interface {
	// GetRepository gets a repository by ID.
	GetRepository(context.Context, *connect_go.Request[v1alpha1.GetRepositoryRequest]) (*connect_go.Response[v1alpha1.GetRepositoryResponse], error)
	// GetRepositoryByFullName gets a repository by full name.
	GetRepositoryByFullName(context.Context, *connect_go.Request[v1alpha1.GetRepositoryByFullNameRequest]) (*connect_go.Response[v1alpha1.GetRepositoryByFullNameResponse], error)
	// ListRepositories lists all repositories.
	ListRepositories(context.Context, *connect_go.Request[v1alpha1.ListRepositoriesRequest]) (*connect_go.Response[v1alpha1.ListRepositoriesResponse], error)
	// ListUserRepositories lists all repositories belonging to a user.
	ListUserRepositories(context.Context, *connect_go.Request[v1alpha1.ListUserRepositoriesRequest]) (*connect_go.Response[v1alpha1.ListUserRepositoriesResponse], error)
	// ListRepositoriesUserCanAccess lists all repositories a user can access.
	ListRepositoriesUserCanAccess(context.Context, *connect_go.Request[v1alpha1.ListRepositoriesUserCanAccessRequest]) (*connect_go.Response[v1alpha1.ListRepositoriesUserCanAccessResponse], error)
	// ListOrganizationRepositories lists all repositories for an organization.
	ListOrganizationRepositories(context.Context, *connect_go.Request[v1alpha1.ListOrganizationRepositoriesRequest]) (*connect_go.Response[v1alpha1.ListOrganizationRepositoriesResponse], error)
	// CreateRepositoryByFullName creates a new repository by full name.
	CreateRepositoryByFullName(context.Context, *connect_go.Request[v1alpha1.CreateRepositoryByFullNameRequest]) (*connect_go.Response[v1alpha1.CreateRepositoryByFullNameResponse], error)
	// DeleteRepository deletes a repository.
	DeleteRepository(context.Context, *connect_go.Request[v1alpha1.DeleteRepositoryRequest]) (*connect_go.Response[v1alpha1.DeleteRepositoryResponse], error)
	// DeleteRepositoryByFullName deletes a repository by full name.
	DeleteRepositoryByFullName(context.Context, *connect_go.Request[v1alpha1.DeleteRepositoryByFullNameRequest]) (*connect_go.Response[v1alpha1.DeleteRepositoryByFullNameResponse], error)
	// DeprecateRepositoryByName deprecates the repository.
	DeprecateRepositoryByName(context.Context, *connect_go.Request[v1alpha1.DeprecateRepositoryByNameRequest]) (*connect_go.Response[v1alpha1.DeprecateRepositoryByNameResponse], error)
	// UndeprecateRepositoryByName makes the repository not deprecated and removes any deprecation_message.
	UndeprecateRepositoryByName(context.Context, *connect_go.Request[v1alpha1.UndeprecateRepositoryByNameRequest]) (*connect_go.Response[v1alpha1.UndeprecateRepositoryByNameResponse], error)
	// GetRepositoriesByFullName gets repositories by full name. Response order is unspecified.
	// Errors if any of the repositories don't exist or the caller does not have access to any of the repositories.
	GetRepositoriesByFullName(context.Context, *connect_go.Request[v1alpha1.GetRepositoriesByFullNameRequest]) (*connect_go.Response[v1alpha1.GetRepositoriesByFullNameResponse], error)
	// SetRepositoryContributor sets the role of a user in the repository.
	SetRepositoryContributor(context.Context, *connect_go.Request[v1alpha1.SetRepositoryContributorRequest]) (*connect_go.Response[v1alpha1.SetRepositoryContributorResponse], error)
	// ListRepositoryContributors returns the list of contributors that has an explicit role against the repository.
	// This does not include users who have implicit roles against the repository, unless they have also been
	// assigned a role explicitly.
	ListRepositoryContributors(context.Context, *connect_go.Request[v1alpha1.ListRepositoryContributorsRequest]) (*connect_go.Response[v1alpha1.ListRepositoryContributorsResponse], error)
	// GetRepositoryContributor returns the contributor information of a user in a repository.
	GetRepositoryContributor(context.Context, *connect_go.Request[v1alpha1.GetRepositoryContributorRequest]) (*connect_go.Response[v1alpha1.GetRepositoryContributorResponse], error)
	// GetRepositorySettings gets the settings of a repository.
	GetRepositorySettings(context.Context, *connect_go.Request[v1alpha1.GetRepositorySettingsRequest]) (*connect_go.Response[v1alpha1.GetRepositorySettingsResponse], error)
	// UpdateRepositorySettingsByName updates the settings of a repository.
	UpdateRepositorySettingsByName(context.Context, *connect_go.Request[v1alpha1.UpdateRepositorySettingsByNameRequest]) (*connect_go.Response[v1alpha1.UpdateRepositorySettingsByNameResponse], error)
	// GetRepositoriesMetadata gets the metadata of the repositories in the request, the length of repositories in the
	// request should match the length of the metadata in the response, and the order of repositories in the response
	// should match the order of the metadata in the request.
	GetRepositoriesMetadata(context.Context, *connect_go.Request[v1alpha1.GetRepositoriesMetadataRequest]) (*connect_go.Response[v1alpha1.GetRepositoriesMetadataResponse], error)
}

// NewRepositoryServiceHandler builds an HTTP handler from the service implementation. It returns
// the path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewRepositoryServiceHandler(svc RepositoryServiceHandler, opts ...connect_go.HandlerOption) (string, http.Handler) {
	mux := http.NewServeMux()
	mux.Handle("/buf.alpha.registry.v1alpha1.RepositoryService/GetRepository", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.RepositoryService/GetRepository",
		svc.GetRepository,
		opts...,
	))
	mux.Handle("/buf.alpha.registry.v1alpha1.RepositoryService/GetRepositoryByFullName", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.RepositoryService/GetRepositoryByFullName",
		svc.GetRepositoryByFullName,
		opts...,
	))
	mux.Handle("/buf.alpha.registry.v1alpha1.RepositoryService/ListRepositories", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.RepositoryService/ListRepositories",
		svc.ListRepositories,
		opts...,
	))
	mux.Handle("/buf.alpha.registry.v1alpha1.RepositoryService/ListUserRepositories", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.RepositoryService/ListUserRepositories",
		svc.ListUserRepositories,
		opts...,
	))
	mux.Handle("/buf.alpha.registry.v1alpha1.RepositoryService/ListRepositoriesUserCanAccess", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.RepositoryService/ListRepositoriesUserCanAccess",
		svc.ListRepositoriesUserCanAccess,
		opts...,
	))
	mux.Handle("/buf.alpha.registry.v1alpha1.RepositoryService/ListOrganizationRepositories", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.RepositoryService/ListOrganizationRepositories",
		svc.ListOrganizationRepositories,
		opts...,
	))
	mux.Handle("/buf.alpha.registry.v1alpha1.RepositoryService/CreateRepositoryByFullName", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.RepositoryService/CreateRepositoryByFullName",
		svc.CreateRepositoryByFullName,
		opts...,
	))
	mux.Handle("/buf.alpha.registry.v1alpha1.RepositoryService/DeleteRepository", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.RepositoryService/DeleteRepository",
		svc.DeleteRepository,
		opts...,
	))
	mux.Handle("/buf.alpha.registry.v1alpha1.RepositoryService/DeleteRepositoryByFullName", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.RepositoryService/DeleteRepositoryByFullName",
		svc.DeleteRepositoryByFullName,
		opts...,
	))
	mux.Handle("/buf.alpha.registry.v1alpha1.RepositoryService/DeprecateRepositoryByName", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.RepositoryService/DeprecateRepositoryByName",
		svc.DeprecateRepositoryByName,
		opts...,
	))
	mux.Handle("/buf.alpha.registry.v1alpha1.RepositoryService/UndeprecateRepositoryByName", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.RepositoryService/UndeprecateRepositoryByName",
		svc.UndeprecateRepositoryByName,
		opts...,
	))
	mux.Handle("/buf.alpha.registry.v1alpha1.RepositoryService/GetRepositoriesByFullName", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.RepositoryService/GetRepositoriesByFullName",
		svc.GetRepositoriesByFullName,
		opts...,
	))
	mux.Handle("/buf.alpha.registry.v1alpha1.RepositoryService/SetRepositoryContributor", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.RepositoryService/SetRepositoryContributor",
		svc.SetRepositoryContributor,
		opts...,
	))
	mux.Handle("/buf.alpha.registry.v1alpha1.RepositoryService/ListRepositoryContributors", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.RepositoryService/ListRepositoryContributors",
		svc.ListRepositoryContributors,
		opts...,
	))
	mux.Handle("/buf.alpha.registry.v1alpha1.RepositoryService/GetRepositoryContributor", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.RepositoryService/GetRepositoryContributor",
		svc.GetRepositoryContributor,
		opts...,
	))
	mux.Handle("/buf.alpha.registry.v1alpha1.RepositoryService/GetRepositorySettings", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.RepositoryService/GetRepositorySettings",
		svc.GetRepositorySettings,
		opts...,
	))
	mux.Handle("/buf.alpha.registry.v1alpha1.RepositoryService/UpdateRepositorySettingsByName", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.RepositoryService/UpdateRepositorySettingsByName",
		svc.UpdateRepositorySettingsByName,
		opts...,
	))
	mux.Handle("/buf.alpha.registry.v1alpha1.RepositoryService/GetRepositoriesMetadata", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.RepositoryService/GetRepositoriesMetadata",
		svc.GetRepositoriesMetadata,
		opts...,
	))
	return "/buf.alpha.registry.v1alpha1.RepositoryService/", mux
}

// UnimplementedRepositoryServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedRepositoryServiceHandler struct{}

func (UnimplementedRepositoryServiceHandler) GetRepository(context.Context, *connect_go.Request[v1alpha1.GetRepositoryRequest]) (*connect_go.Response[v1alpha1.GetRepositoryResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.RepositoryService.GetRepository is not implemented"))
}

func (UnimplementedRepositoryServiceHandler) GetRepositoryByFullName(context.Context, *connect_go.Request[v1alpha1.GetRepositoryByFullNameRequest]) (*connect_go.Response[v1alpha1.GetRepositoryByFullNameResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.RepositoryService.GetRepositoryByFullName is not implemented"))
}

func (UnimplementedRepositoryServiceHandler) ListRepositories(context.Context, *connect_go.Request[v1alpha1.ListRepositoriesRequest]) (*connect_go.Response[v1alpha1.ListRepositoriesResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.RepositoryService.ListRepositories is not implemented"))
}

func (UnimplementedRepositoryServiceHandler) ListUserRepositories(context.Context, *connect_go.Request[v1alpha1.ListUserRepositoriesRequest]) (*connect_go.Response[v1alpha1.ListUserRepositoriesResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.RepositoryService.ListUserRepositories is not implemented"))
}

func (UnimplementedRepositoryServiceHandler) ListRepositoriesUserCanAccess(context.Context, *connect_go.Request[v1alpha1.ListRepositoriesUserCanAccessRequest]) (*connect_go.Response[v1alpha1.ListRepositoriesUserCanAccessResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.RepositoryService.ListRepositoriesUserCanAccess is not implemented"))
}

func (UnimplementedRepositoryServiceHandler) ListOrganizationRepositories(context.Context, *connect_go.Request[v1alpha1.ListOrganizationRepositoriesRequest]) (*connect_go.Response[v1alpha1.ListOrganizationRepositoriesResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.RepositoryService.ListOrganizationRepositories is not implemented"))
}

func (UnimplementedRepositoryServiceHandler) CreateRepositoryByFullName(context.Context, *connect_go.Request[v1alpha1.CreateRepositoryByFullNameRequest]) (*connect_go.Response[v1alpha1.CreateRepositoryByFullNameResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.RepositoryService.CreateRepositoryByFullName is not implemented"))
}

func (UnimplementedRepositoryServiceHandler) DeleteRepository(context.Context, *connect_go.Request[v1alpha1.DeleteRepositoryRequest]) (*connect_go.Response[v1alpha1.DeleteRepositoryResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.RepositoryService.DeleteRepository is not implemented"))
}

func (UnimplementedRepositoryServiceHandler) DeleteRepositoryByFullName(context.Context, *connect_go.Request[v1alpha1.DeleteRepositoryByFullNameRequest]) (*connect_go.Response[v1alpha1.DeleteRepositoryByFullNameResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.RepositoryService.DeleteRepositoryByFullName is not implemented"))
}

func (UnimplementedRepositoryServiceHandler) DeprecateRepositoryByName(context.Context, *connect_go.Request[v1alpha1.DeprecateRepositoryByNameRequest]) (*connect_go.Response[v1alpha1.DeprecateRepositoryByNameResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.RepositoryService.DeprecateRepositoryByName is not implemented"))
}

func (UnimplementedRepositoryServiceHandler) UndeprecateRepositoryByName(context.Context, *connect_go.Request[v1alpha1.UndeprecateRepositoryByNameRequest]) (*connect_go.Response[v1alpha1.UndeprecateRepositoryByNameResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.RepositoryService.UndeprecateRepositoryByName is not implemented"))
}

func (UnimplementedRepositoryServiceHandler) GetRepositoriesByFullName(context.Context, *connect_go.Request[v1alpha1.GetRepositoriesByFullNameRequest]) (*connect_go.Response[v1alpha1.GetRepositoriesByFullNameResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.RepositoryService.GetRepositoriesByFullName is not implemented"))
}

func (UnimplementedRepositoryServiceHandler) SetRepositoryContributor(context.Context, *connect_go.Request[v1alpha1.SetRepositoryContributorRequest]) (*connect_go.Response[v1alpha1.SetRepositoryContributorResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.RepositoryService.SetRepositoryContributor is not implemented"))
}

func (UnimplementedRepositoryServiceHandler) ListRepositoryContributors(context.Context, *connect_go.Request[v1alpha1.ListRepositoryContributorsRequest]) (*connect_go.Response[v1alpha1.ListRepositoryContributorsResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.RepositoryService.ListRepositoryContributors is not implemented"))
}

func (UnimplementedRepositoryServiceHandler) GetRepositoryContributor(context.Context, *connect_go.Request[v1alpha1.GetRepositoryContributorRequest]) (*connect_go.Response[v1alpha1.GetRepositoryContributorResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.RepositoryService.GetRepositoryContributor is not implemented"))
}

func (UnimplementedRepositoryServiceHandler) GetRepositorySettings(context.Context, *connect_go.Request[v1alpha1.GetRepositorySettingsRequest]) (*connect_go.Response[v1alpha1.GetRepositorySettingsResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.RepositoryService.GetRepositorySettings is not implemented"))
}

func (UnimplementedRepositoryServiceHandler) UpdateRepositorySettingsByName(context.Context, *connect_go.Request[v1alpha1.UpdateRepositorySettingsByNameRequest]) (*connect_go.Response[v1alpha1.UpdateRepositorySettingsByNameResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.RepositoryService.UpdateRepositorySettingsByName is not implemented"))
}

func (UnimplementedRepositoryServiceHandler) GetRepositoriesMetadata(context.Context, *connect_go.Request[v1alpha1.GetRepositoriesMetadataRequest]) (*connect_go.Response[v1alpha1.GetRepositoriesMetadataResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.RepositoryService.GetRepositoriesMetadata is not implemented"))
}
