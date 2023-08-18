// Copyright 2020-2023 Buf Technologies, Inc.
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
// Source: buf/alpha/registry/v1alpha1/authz.proto

package registryv1alpha1connect

import (
	connect "connectrpc.com/connect"
	context "context"
	errors "errors"
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	http "net/http"
	strings "strings"
)

// This is a compile-time assertion to ensure that this generated file and the connect package are
// compatible. If you get a compiler error that this constant is not defined, this code was
// generated with a version of connect newer than the one compiled into your binary. You can fix the
// problem by either regenerating this code with an older version of connect or updating the connect
// version compiled into your binary.
const _ = connect.IsAtLeastVersion1_7_0

const (
	// AuthzServiceName is the fully-qualified name of the AuthzService service.
	AuthzServiceName = "buf.alpha.registry.v1alpha1.AuthzService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// AuthzServiceUserCanCreateOrganizationRepositoryProcedure is the fully-qualified name of the
	// AuthzService's UserCanCreateOrganizationRepository RPC.
	AuthzServiceUserCanCreateOrganizationRepositoryProcedure = "/buf.alpha.registry.v1alpha1.AuthzService/UserCanCreateOrganizationRepository"
	// AuthzServiceUserCanSeeRepositorySettingsProcedure is the fully-qualified name of the
	// AuthzService's UserCanSeeRepositorySettings RPC.
	AuthzServiceUserCanSeeRepositorySettingsProcedure = "/buf.alpha.registry.v1alpha1.AuthzService/UserCanSeeRepositorySettings"
	// AuthzServiceUserCanSeeOrganizationSettingsProcedure is the fully-qualified name of the
	// AuthzService's UserCanSeeOrganizationSettings RPC.
	AuthzServiceUserCanSeeOrganizationSettingsProcedure = "/buf.alpha.registry.v1alpha1.AuthzService/UserCanSeeOrganizationSettings"
	// AuthzServiceUserCanAddOrganizationMemberProcedure is the fully-qualified name of the
	// AuthzService's UserCanAddOrganizationMember RPC.
	AuthzServiceUserCanAddOrganizationMemberProcedure = "/buf.alpha.registry.v1alpha1.AuthzService/UserCanAddOrganizationMember"
	// AuthzServiceUserCanUpdateOrganizationMemberProcedure is the fully-qualified name of the
	// AuthzService's UserCanUpdateOrganizationMember RPC.
	AuthzServiceUserCanUpdateOrganizationMemberProcedure = "/buf.alpha.registry.v1alpha1.AuthzService/UserCanUpdateOrganizationMember"
	// AuthzServiceUserCanRemoveOrganizationMemberProcedure is the fully-qualified name of the
	// AuthzService's UserCanRemoveOrganizationMember RPC.
	AuthzServiceUserCanRemoveOrganizationMemberProcedure = "/buf.alpha.registry.v1alpha1.AuthzService/UserCanRemoveOrganizationMember"
	// AuthzServiceUserCanDeleteOrganizationProcedure is the fully-qualified name of the AuthzService's
	// UserCanDeleteOrganization RPC.
	AuthzServiceUserCanDeleteOrganizationProcedure = "/buf.alpha.registry.v1alpha1.AuthzService/UserCanDeleteOrganization"
	// AuthzServiceUserCanDeleteRepositoryProcedure is the fully-qualified name of the AuthzService's
	// UserCanDeleteRepository RPC.
	AuthzServiceUserCanDeleteRepositoryProcedure = "/buf.alpha.registry.v1alpha1.AuthzService/UserCanDeleteRepository"
	// AuthzServiceUserCanDeleteUserProcedure is the fully-qualified name of the AuthzService's
	// UserCanDeleteUser RPC.
	AuthzServiceUserCanDeleteUserProcedure = "/buf.alpha.registry.v1alpha1.AuthzService/UserCanDeleteUser"
	// AuthzServiceUserCanSeeServerAdminPanelProcedure is the fully-qualified name of the AuthzService's
	// UserCanSeeServerAdminPanel RPC.
	AuthzServiceUserCanSeeServerAdminPanelProcedure = "/buf.alpha.registry.v1alpha1.AuthzService/UserCanSeeServerAdminPanel"
	// AuthzServiceUserCanManageRepositoryContributorsProcedure is the fully-qualified name of the
	// AuthzService's UserCanManageRepositoryContributors RPC.
	AuthzServiceUserCanManageRepositoryContributorsProcedure = "/buf.alpha.registry.v1alpha1.AuthzService/UserCanManageRepositoryContributors"
)

// AuthzServiceClient is a client for the buf.alpha.registry.v1alpha1.AuthzService service.
type AuthzServiceClient interface {
	// UserCanCreateOrganizationRepository returns whether the user is authorized
	// to create repositories in an organization.
	UserCanCreateOrganizationRepository(context.Context, *connect.Request[v1alpha1.UserCanCreateOrganizationRepositoryRequest]) (*connect.Response[v1alpha1.UserCanCreateOrganizationRepositoryResponse], error)
	// UserCanSeeRepositorySettings returns whether the user is authorized
	// to see repository settings.
	UserCanSeeRepositorySettings(context.Context, *connect.Request[v1alpha1.UserCanSeeRepositorySettingsRequest]) (*connect.Response[v1alpha1.UserCanSeeRepositorySettingsResponse], error)
	// UserCanSeeOrganizationSettings returns whether the user is authorized
	// to see organization settings.
	UserCanSeeOrganizationSettings(context.Context, *connect.Request[v1alpha1.UserCanSeeOrganizationSettingsRequest]) (*connect.Response[v1alpha1.UserCanSeeOrganizationSettingsResponse], error)
	// UserCanAddOrganizationMember returns whether the user is authorized to add
	// any members to the organization and the list of roles they can add.
	UserCanAddOrganizationMember(context.Context, *connect.Request[v1alpha1.UserCanAddOrganizationMemberRequest]) (*connect.Response[v1alpha1.UserCanAddOrganizationMemberResponse], error)
	// UserCanUpdateOrganizationMember returns whether the user is authorized to update
	// any members' membership information in the organization and the list of roles they can update.
	UserCanUpdateOrganizationMember(context.Context, *connect.Request[v1alpha1.UserCanUpdateOrganizationMemberRequest]) (*connect.Response[v1alpha1.UserCanUpdateOrganizationMemberResponse], error)
	// UserCanRemoveOrganizationMember returns whether the user is authorized to remove
	// any members from the organization and the list of roles they can remove.
	UserCanRemoveOrganizationMember(context.Context, *connect.Request[v1alpha1.UserCanRemoveOrganizationMemberRequest]) (*connect.Response[v1alpha1.UserCanRemoveOrganizationMemberResponse], error)
	// UserCanDeleteOrganization returns whether the user is authorized
	// to delete an organization.
	UserCanDeleteOrganization(context.Context, *connect.Request[v1alpha1.UserCanDeleteOrganizationRequest]) (*connect.Response[v1alpha1.UserCanDeleteOrganizationResponse], error)
	// UserCanDeleteRepository returns whether the user is authorized
	// to delete a repository.
	UserCanDeleteRepository(context.Context, *connect.Request[v1alpha1.UserCanDeleteRepositoryRequest]) (*connect.Response[v1alpha1.UserCanDeleteRepositoryResponse], error)
	// UserCanDeleteUser returns whether the user is authorized
	// to delete a user.
	UserCanDeleteUser(context.Context, *connect.Request[v1alpha1.UserCanDeleteUserRequest]) (*connect.Response[v1alpha1.UserCanDeleteUserResponse], error)
	// UserCanSeeServerAdminPanel returns whether the user is authorized
	// to see server admin panel.
	UserCanSeeServerAdminPanel(context.Context, *connect.Request[v1alpha1.UserCanSeeServerAdminPanelRequest]) (*connect.Response[v1alpha1.UserCanSeeServerAdminPanelResponse], error)
	// UserCanManageRepositoryContributors returns whether the user is authorized to manage
	// any contributors to the repository and the list of roles they can manage.
	UserCanManageRepositoryContributors(context.Context, *connect.Request[v1alpha1.UserCanManageRepositoryContributorsRequest]) (*connect.Response[v1alpha1.UserCanManageRepositoryContributorsResponse], error)
}

// NewAuthzServiceClient constructs a client for the buf.alpha.registry.v1alpha1.AuthzService
// service. By default, it uses the Connect protocol with the binary Protobuf Codec, asks for
// gzipped responses, and sends uncompressed requests. To use the gRPC or gRPC-Web protocols, supply
// the connect.WithGRPC() or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewAuthzServiceClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) AuthzServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &authzServiceClient{
		userCanCreateOrganizationRepository: connect.NewClient[v1alpha1.UserCanCreateOrganizationRepositoryRequest, v1alpha1.UserCanCreateOrganizationRepositoryResponse](
			httpClient,
			baseURL+AuthzServiceUserCanCreateOrganizationRepositoryProcedure,
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
		userCanSeeRepositorySettings: connect.NewClient[v1alpha1.UserCanSeeRepositorySettingsRequest, v1alpha1.UserCanSeeRepositorySettingsResponse](
			httpClient,
			baseURL+AuthzServiceUserCanSeeRepositorySettingsProcedure,
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
		userCanSeeOrganizationSettings: connect.NewClient[v1alpha1.UserCanSeeOrganizationSettingsRequest, v1alpha1.UserCanSeeOrganizationSettingsResponse](
			httpClient,
			baseURL+AuthzServiceUserCanSeeOrganizationSettingsProcedure,
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
		userCanAddOrganizationMember: connect.NewClient[v1alpha1.UserCanAddOrganizationMemberRequest, v1alpha1.UserCanAddOrganizationMemberResponse](
			httpClient,
			baseURL+AuthzServiceUserCanAddOrganizationMemberProcedure,
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
		userCanUpdateOrganizationMember: connect.NewClient[v1alpha1.UserCanUpdateOrganizationMemberRequest, v1alpha1.UserCanUpdateOrganizationMemberResponse](
			httpClient,
			baseURL+AuthzServiceUserCanUpdateOrganizationMemberProcedure,
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
		userCanRemoveOrganizationMember: connect.NewClient[v1alpha1.UserCanRemoveOrganizationMemberRequest, v1alpha1.UserCanRemoveOrganizationMemberResponse](
			httpClient,
			baseURL+AuthzServiceUserCanRemoveOrganizationMemberProcedure,
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
		userCanDeleteOrganization: connect.NewClient[v1alpha1.UserCanDeleteOrganizationRequest, v1alpha1.UserCanDeleteOrganizationResponse](
			httpClient,
			baseURL+AuthzServiceUserCanDeleteOrganizationProcedure,
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
		userCanDeleteRepository: connect.NewClient[v1alpha1.UserCanDeleteRepositoryRequest, v1alpha1.UserCanDeleteRepositoryResponse](
			httpClient,
			baseURL+AuthzServiceUserCanDeleteRepositoryProcedure,
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
		userCanDeleteUser: connect.NewClient[v1alpha1.UserCanDeleteUserRequest, v1alpha1.UserCanDeleteUserResponse](
			httpClient,
			baseURL+AuthzServiceUserCanDeleteUserProcedure,
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
		userCanSeeServerAdminPanel: connect.NewClient[v1alpha1.UserCanSeeServerAdminPanelRequest, v1alpha1.UserCanSeeServerAdminPanelResponse](
			httpClient,
			baseURL+AuthzServiceUserCanSeeServerAdminPanelProcedure,
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
		userCanManageRepositoryContributors: connect.NewClient[v1alpha1.UserCanManageRepositoryContributorsRequest, v1alpha1.UserCanManageRepositoryContributorsResponse](
			httpClient,
			baseURL+AuthzServiceUserCanManageRepositoryContributorsProcedure,
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
	}
}

// authzServiceClient implements AuthzServiceClient.
type authzServiceClient struct {
	userCanCreateOrganizationRepository *connect.Client[v1alpha1.UserCanCreateOrganizationRepositoryRequest, v1alpha1.UserCanCreateOrganizationRepositoryResponse]
	userCanSeeRepositorySettings        *connect.Client[v1alpha1.UserCanSeeRepositorySettingsRequest, v1alpha1.UserCanSeeRepositorySettingsResponse]
	userCanSeeOrganizationSettings      *connect.Client[v1alpha1.UserCanSeeOrganizationSettingsRequest, v1alpha1.UserCanSeeOrganizationSettingsResponse]
	userCanAddOrganizationMember        *connect.Client[v1alpha1.UserCanAddOrganizationMemberRequest, v1alpha1.UserCanAddOrganizationMemberResponse]
	userCanUpdateOrganizationMember     *connect.Client[v1alpha1.UserCanUpdateOrganizationMemberRequest, v1alpha1.UserCanUpdateOrganizationMemberResponse]
	userCanRemoveOrganizationMember     *connect.Client[v1alpha1.UserCanRemoveOrganizationMemberRequest, v1alpha1.UserCanRemoveOrganizationMemberResponse]
	userCanDeleteOrganization           *connect.Client[v1alpha1.UserCanDeleteOrganizationRequest, v1alpha1.UserCanDeleteOrganizationResponse]
	userCanDeleteRepository             *connect.Client[v1alpha1.UserCanDeleteRepositoryRequest, v1alpha1.UserCanDeleteRepositoryResponse]
	userCanDeleteUser                   *connect.Client[v1alpha1.UserCanDeleteUserRequest, v1alpha1.UserCanDeleteUserResponse]
	userCanSeeServerAdminPanel          *connect.Client[v1alpha1.UserCanSeeServerAdminPanelRequest, v1alpha1.UserCanSeeServerAdminPanelResponse]
	userCanManageRepositoryContributors *connect.Client[v1alpha1.UserCanManageRepositoryContributorsRequest, v1alpha1.UserCanManageRepositoryContributorsResponse]
}

// UserCanCreateOrganizationRepository calls
// buf.alpha.registry.v1alpha1.AuthzService.UserCanCreateOrganizationRepository.
func (c *authzServiceClient) UserCanCreateOrganizationRepository(ctx context.Context, req *connect.Request[v1alpha1.UserCanCreateOrganizationRepositoryRequest]) (*connect.Response[v1alpha1.UserCanCreateOrganizationRepositoryResponse], error) {
	return c.userCanCreateOrganizationRepository.CallUnary(ctx, req)
}

// UserCanSeeRepositorySettings calls
// buf.alpha.registry.v1alpha1.AuthzService.UserCanSeeRepositorySettings.
func (c *authzServiceClient) UserCanSeeRepositorySettings(ctx context.Context, req *connect.Request[v1alpha1.UserCanSeeRepositorySettingsRequest]) (*connect.Response[v1alpha1.UserCanSeeRepositorySettingsResponse], error) {
	return c.userCanSeeRepositorySettings.CallUnary(ctx, req)
}

// UserCanSeeOrganizationSettings calls
// buf.alpha.registry.v1alpha1.AuthzService.UserCanSeeOrganizationSettings.
func (c *authzServiceClient) UserCanSeeOrganizationSettings(ctx context.Context, req *connect.Request[v1alpha1.UserCanSeeOrganizationSettingsRequest]) (*connect.Response[v1alpha1.UserCanSeeOrganizationSettingsResponse], error) {
	return c.userCanSeeOrganizationSettings.CallUnary(ctx, req)
}

// UserCanAddOrganizationMember calls
// buf.alpha.registry.v1alpha1.AuthzService.UserCanAddOrganizationMember.
func (c *authzServiceClient) UserCanAddOrganizationMember(ctx context.Context, req *connect.Request[v1alpha1.UserCanAddOrganizationMemberRequest]) (*connect.Response[v1alpha1.UserCanAddOrganizationMemberResponse], error) {
	return c.userCanAddOrganizationMember.CallUnary(ctx, req)
}

// UserCanUpdateOrganizationMember calls
// buf.alpha.registry.v1alpha1.AuthzService.UserCanUpdateOrganizationMember.
func (c *authzServiceClient) UserCanUpdateOrganizationMember(ctx context.Context, req *connect.Request[v1alpha1.UserCanUpdateOrganizationMemberRequest]) (*connect.Response[v1alpha1.UserCanUpdateOrganizationMemberResponse], error) {
	return c.userCanUpdateOrganizationMember.CallUnary(ctx, req)
}

// UserCanRemoveOrganizationMember calls
// buf.alpha.registry.v1alpha1.AuthzService.UserCanRemoveOrganizationMember.
func (c *authzServiceClient) UserCanRemoveOrganizationMember(ctx context.Context, req *connect.Request[v1alpha1.UserCanRemoveOrganizationMemberRequest]) (*connect.Response[v1alpha1.UserCanRemoveOrganizationMemberResponse], error) {
	return c.userCanRemoveOrganizationMember.CallUnary(ctx, req)
}

// UserCanDeleteOrganization calls
// buf.alpha.registry.v1alpha1.AuthzService.UserCanDeleteOrganization.
func (c *authzServiceClient) UserCanDeleteOrganization(ctx context.Context, req *connect.Request[v1alpha1.UserCanDeleteOrganizationRequest]) (*connect.Response[v1alpha1.UserCanDeleteOrganizationResponse], error) {
	return c.userCanDeleteOrganization.CallUnary(ctx, req)
}

// UserCanDeleteRepository calls buf.alpha.registry.v1alpha1.AuthzService.UserCanDeleteRepository.
func (c *authzServiceClient) UserCanDeleteRepository(ctx context.Context, req *connect.Request[v1alpha1.UserCanDeleteRepositoryRequest]) (*connect.Response[v1alpha1.UserCanDeleteRepositoryResponse], error) {
	return c.userCanDeleteRepository.CallUnary(ctx, req)
}

// UserCanDeleteUser calls buf.alpha.registry.v1alpha1.AuthzService.UserCanDeleteUser.
func (c *authzServiceClient) UserCanDeleteUser(ctx context.Context, req *connect.Request[v1alpha1.UserCanDeleteUserRequest]) (*connect.Response[v1alpha1.UserCanDeleteUserResponse], error) {
	return c.userCanDeleteUser.CallUnary(ctx, req)
}

// UserCanSeeServerAdminPanel calls
// buf.alpha.registry.v1alpha1.AuthzService.UserCanSeeServerAdminPanel.
func (c *authzServiceClient) UserCanSeeServerAdminPanel(ctx context.Context, req *connect.Request[v1alpha1.UserCanSeeServerAdminPanelRequest]) (*connect.Response[v1alpha1.UserCanSeeServerAdminPanelResponse], error) {
	return c.userCanSeeServerAdminPanel.CallUnary(ctx, req)
}

// UserCanManageRepositoryContributors calls
// buf.alpha.registry.v1alpha1.AuthzService.UserCanManageRepositoryContributors.
func (c *authzServiceClient) UserCanManageRepositoryContributors(ctx context.Context, req *connect.Request[v1alpha1.UserCanManageRepositoryContributorsRequest]) (*connect.Response[v1alpha1.UserCanManageRepositoryContributorsResponse], error) {
	return c.userCanManageRepositoryContributors.CallUnary(ctx, req)
}

// AuthzServiceHandler is an implementation of the buf.alpha.registry.v1alpha1.AuthzService service.
type AuthzServiceHandler interface {
	// UserCanCreateOrganizationRepository returns whether the user is authorized
	// to create repositories in an organization.
	UserCanCreateOrganizationRepository(context.Context, *connect.Request[v1alpha1.UserCanCreateOrganizationRepositoryRequest]) (*connect.Response[v1alpha1.UserCanCreateOrganizationRepositoryResponse], error)
	// UserCanSeeRepositorySettings returns whether the user is authorized
	// to see repository settings.
	UserCanSeeRepositorySettings(context.Context, *connect.Request[v1alpha1.UserCanSeeRepositorySettingsRequest]) (*connect.Response[v1alpha1.UserCanSeeRepositorySettingsResponse], error)
	// UserCanSeeOrganizationSettings returns whether the user is authorized
	// to see organization settings.
	UserCanSeeOrganizationSettings(context.Context, *connect.Request[v1alpha1.UserCanSeeOrganizationSettingsRequest]) (*connect.Response[v1alpha1.UserCanSeeOrganizationSettingsResponse], error)
	// UserCanAddOrganizationMember returns whether the user is authorized to add
	// any members to the organization and the list of roles they can add.
	UserCanAddOrganizationMember(context.Context, *connect.Request[v1alpha1.UserCanAddOrganizationMemberRequest]) (*connect.Response[v1alpha1.UserCanAddOrganizationMemberResponse], error)
	// UserCanUpdateOrganizationMember returns whether the user is authorized to update
	// any members' membership information in the organization and the list of roles they can update.
	UserCanUpdateOrganizationMember(context.Context, *connect.Request[v1alpha1.UserCanUpdateOrganizationMemberRequest]) (*connect.Response[v1alpha1.UserCanUpdateOrganizationMemberResponse], error)
	// UserCanRemoveOrganizationMember returns whether the user is authorized to remove
	// any members from the organization and the list of roles they can remove.
	UserCanRemoveOrganizationMember(context.Context, *connect.Request[v1alpha1.UserCanRemoveOrganizationMemberRequest]) (*connect.Response[v1alpha1.UserCanRemoveOrganizationMemberResponse], error)
	// UserCanDeleteOrganization returns whether the user is authorized
	// to delete an organization.
	UserCanDeleteOrganization(context.Context, *connect.Request[v1alpha1.UserCanDeleteOrganizationRequest]) (*connect.Response[v1alpha1.UserCanDeleteOrganizationResponse], error)
	// UserCanDeleteRepository returns whether the user is authorized
	// to delete a repository.
	UserCanDeleteRepository(context.Context, *connect.Request[v1alpha1.UserCanDeleteRepositoryRequest]) (*connect.Response[v1alpha1.UserCanDeleteRepositoryResponse], error)
	// UserCanDeleteUser returns whether the user is authorized
	// to delete a user.
	UserCanDeleteUser(context.Context, *connect.Request[v1alpha1.UserCanDeleteUserRequest]) (*connect.Response[v1alpha1.UserCanDeleteUserResponse], error)
	// UserCanSeeServerAdminPanel returns whether the user is authorized
	// to see server admin panel.
	UserCanSeeServerAdminPanel(context.Context, *connect.Request[v1alpha1.UserCanSeeServerAdminPanelRequest]) (*connect.Response[v1alpha1.UserCanSeeServerAdminPanelResponse], error)
	// UserCanManageRepositoryContributors returns whether the user is authorized to manage
	// any contributors to the repository and the list of roles they can manage.
	UserCanManageRepositoryContributors(context.Context, *connect.Request[v1alpha1.UserCanManageRepositoryContributorsRequest]) (*connect.Response[v1alpha1.UserCanManageRepositoryContributorsResponse], error)
}

// NewAuthzServiceHandler builds an HTTP handler from the service implementation. It returns the
// path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewAuthzServiceHandler(svc AuthzServiceHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	authzServiceUserCanCreateOrganizationRepositoryHandler := connect.NewUnaryHandler(
		AuthzServiceUserCanCreateOrganizationRepositoryProcedure,
		svc.UserCanCreateOrganizationRepository,
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	authzServiceUserCanSeeRepositorySettingsHandler := connect.NewUnaryHandler(
		AuthzServiceUserCanSeeRepositorySettingsProcedure,
		svc.UserCanSeeRepositorySettings,
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	authzServiceUserCanSeeOrganizationSettingsHandler := connect.NewUnaryHandler(
		AuthzServiceUserCanSeeOrganizationSettingsProcedure,
		svc.UserCanSeeOrganizationSettings,
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	authzServiceUserCanAddOrganizationMemberHandler := connect.NewUnaryHandler(
		AuthzServiceUserCanAddOrganizationMemberProcedure,
		svc.UserCanAddOrganizationMember,
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	authzServiceUserCanUpdateOrganizationMemberHandler := connect.NewUnaryHandler(
		AuthzServiceUserCanUpdateOrganizationMemberProcedure,
		svc.UserCanUpdateOrganizationMember,
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	authzServiceUserCanRemoveOrganizationMemberHandler := connect.NewUnaryHandler(
		AuthzServiceUserCanRemoveOrganizationMemberProcedure,
		svc.UserCanRemoveOrganizationMember,
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	authzServiceUserCanDeleteOrganizationHandler := connect.NewUnaryHandler(
		AuthzServiceUserCanDeleteOrganizationProcedure,
		svc.UserCanDeleteOrganization,
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	authzServiceUserCanDeleteRepositoryHandler := connect.NewUnaryHandler(
		AuthzServiceUserCanDeleteRepositoryProcedure,
		svc.UserCanDeleteRepository,
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	authzServiceUserCanDeleteUserHandler := connect.NewUnaryHandler(
		AuthzServiceUserCanDeleteUserProcedure,
		svc.UserCanDeleteUser,
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	authzServiceUserCanSeeServerAdminPanelHandler := connect.NewUnaryHandler(
		AuthzServiceUserCanSeeServerAdminPanelProcedure,
		svc.UserCanSeeServerAdminPanel,
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	authzServiceUserCanManageRepositoryContributorsHandler := connect.NewUnaryHandler(
		AuthzServiceUserCanManageRepositoryContributorsProcedure,
		svc.UserCanManageRepositoryContributors,
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	return "/buf.alpha.registry.v1alpha1.AuthzService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case AuthzServiceUserCanCreateOrganizationRepositoryProcedure:
			authzServiceUserCanCreateOrganizationRepositoryHandler.ServeHTTP(w, r)
		case AuthzServiceUserCanSeeRepositorySettingsProcedure:
			authzServiceUserCanSeeRepositorySettingsHandler.ServeHTTP(w, r)
		case AuthzServiceUserCanSeeOrganizationSettingsProcedure:
			authzServiceUserCanSeeOrganizationSettingsHandler.ServeHTTP(w, r)
		case AuthzServiceUserCanAddOrganizationMemberProcedure:
			authzServiceUserCanAddOrganizationMemberHandler.ServeHTTP(w, r)
		case AuthzServiceUserCanUpdateOrganizationMemberProcedure:
			authzServiceUserCanUpdateOrganizationMemberHandler.ServeHTTP(w, r)
		case AuthzServiceUserCanRemoveOrganizationMemberProcedure:
			authzServiceUserCanRemoveOrganizationMemberHandler.ServeHTTP(w, r)
		case AuthzServiceUserCanDeleteOrganizationProcedure:
			authzServiceUserCanDeleteOrganizationHandler.ServeHTTP(w, r)
		case AuthzServiceUserCanDeleteRepositoryProcedure:
			authzServiceUserCanDeleteRepositoryHandler.ServeHTTP(w, r)
		case AuthzServiceUserCanDeleteUserProcedure:
			authzServiceUserCanDeleteUserHandler.ServeHTTP(w, r)
		case AuthzServiceUserCanSeeServerAdminPanelProcedure:
			authzServiceUserCanSeeServerAdminPanelHandler.ServeHTTP(w, r)
		case AuthzServiceUserCanManageRepositoryContributorsProcedure:
			authzServiceUserCanManageRepositoryContributorsHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedAuthzServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedAuthzServiceHandler struct{}

func (UnimplementedAuthzServiceHandler) UserCanCreateOrganizationRepository(context.Context, *connect.Request[v1alpha1.UserCanCreateOrganizationRepositoryRequest]) (*connect.Response[v1alpha1.UserCanCreateOrganizationRepositoryResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.AuthzService.UserCanCreateOrganizationRepository is not implemented"))
}

func (UnimplementedAuthzServiceHandler) UserCanSeeRepositorySettings(context.Context, *connect.Request[v1alpha1.UserCanSeeRepositorySettingsRequest]) (*connect.Response[v1alpha1.UserCanSeeRepositorySettingsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.AuthzService.UserCanSeeRepositorySettings is not implemented"))
}

func (UnimplementedAuthzServiceHandler) UserCanSeeOrganizationSettings(context.Context, *connect.Request[v1alpha1.UserCanSeeOrganizationSettingsRequest]) (*connect.Response[v1alpha1.UserCanSeeOrganizationSettingsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.AuthzService.UserCanSeeOrganizationSettings is not implemented"))
}

func (UnimplementedAuthzServiceHandler) UserCanAddOrganizationMember(context.Context, *connect.Request[v1alpha1.UserCanAddOrganizationMemberRequest]) (*connect.Response[v1alpha1.UserCanAddOrganizationMemberResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.AuthzService.UserCanAddOrganizationMember is not implemented"))
}

func (UnimplementedAuthzServiceHandler) UserCanUpdateOrganizationMember(context.Context, *connect.Request[v1alpha1.UserCanUpdateOrganizationMemberRequest]) (*connect.Response[v1alpha1.UserCanUpdateOrganizationMemberResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.AuthzService.UserCanUpdateOrganizationMember is not implemented"))
}

func (UnimplementedAuthzServiceHandler) UserCanRemoveOrganizationMember(context.Context, *connect.Request[v1alpha1.UserCanRemoveOrganizationMemberRequest]) (*connect.Response[v1alpha1.UserCanRemoveOrganizationMemberResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.AuthzService.UserCanRemoveOrganizationMember is not implemented"))
}

func (UnimplementedAuthzServiceHandler) UserCanDeleteOrganization(context.Context, *connect.Request[v1alpha1.UserCanDeleteOrganizationRequest]) (*connect.Response[v1alpha1.UserCanDeleteOrganizationResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.AuthzService.UserCanDeleteOrganization is not implemented"))
}

func (UnimplementedAuthzServiceHandler) UserCanDeleteRepository(context.Context, *connect.Request[v1alpha1.UserCanDeleteRepositoryRequest]) (*connect.Response[v1alpha1.UserCanDeleteRepositoryResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.AuthzService.UserCanDeleteRepository is not implemented"))
}

func (UnimplementedAuthzServiceHandler) UserCanDeleteUser(context.Context, *connect.Request[v1alpha1.UserCanDeleteUserRequest]) (*connect.Response[v1alpha1.UserCanDeleteUserResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.AuthzService.UserCanDeleteUser is not implemented"))
}

func (UnimplementedAuthzServiceHandler) UserCanSeeServerAdminPanel(context.Context, *connect.Request[v1alpha1.UserCanSeeServerAdminPanelRequest]) (*connect.Response[v1alpha1.UserCanSeeServerAdminPanelResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.AuthzService.UserCanSeeServerAdminPanel is not implemented"))
}

func (UnimplementedAuthzServiceHandler) UserCanManageRepositoryContributors(context.Context, *connect.Request[v1alpha1.UserCanManageRepositoryContributorsRequest]) (*connect.Response[v1alpha1.UserCanManageRepositoryContributorsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.AuthzService.UserCanManageRepositoryContributors is not implemented"))
}
