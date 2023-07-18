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
// Source: buf/alpha/registry/v1alpha1/admin.proto

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
const _ = connect_go.IsAtLeastVersion1_7_0

const (
	// AdminServiceName is the fully-qualified name of the AdminService service.
	AdminServiceName = "buf.alpha.registry.v1alpha1.AdminService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// AdminServiceForceDeleteUserProcedure is the fully-qualified name of the AdminService's
	// ForceDeleteUser RPC.
	AdminServiceForceDeleteUserProcedure = "/buf.alpha.registry.v1alpha1.AdminService/ForceDeleteUser"
	// AdminServiceUpdateUserVerificationStatusProcedure is the fully-qualified name of the
	// AdminService's UpdateUserVerificationStatus RPC.
	AdminServiceUpdateUserVerificationStatusProcedure = "/buf.alpha.registry.v1alpha1.AdminService/UpdateUserVerificationStatus"
	// AdminServiceUpdateOrganizationVerificationStatusProcedure is the fully-qualified name of the
	// AdminService's UpdateOrganizationVerificationStatus RPC.
	AdminServiceUpdateOrganizationVerificationStatusProcedure = "/buf.alpha.registry.v1alpha1.AdminService/UpdateOrganizationVerificationStatus"
	// AdminServiceCreateMachineUserProcedure is the fully-qualified name of the AdminService's
	// CreateMachineUser RPC.
	AdminServiceCreateMachineUserProcedure = "/buf.alpha.registry.v1alpha1.AdminService/CreateMachineUser"
	// AdminServiceGetBreakingChangePolicyProcedure is the fully-qualified name of the AdminService's
	// GetBreakingChangePolicy RPC.
	AdminServiceGetBreakingChangePolicyProcedure = "/buf.alpha.registry.v1alpha1.AdminService/GetBreakingChangePolicy"
	// AdminServiceUpdateBreakingChangePolicyProcedure is the fully-qualified name of the AdminService's
	// UpdateBreakingChangePolicy RPC.
	AdminServiceUpdateBreakingChangePolicyProcedure = "/buf.alpha.registry.v1alpha1.AdminService/UpdateBreakingChangePolicy"
	// AdminServiceGetUniquenessPolicyProcedure is the fully-qualified name of the AdminService's
	// GetUniquenessPolicy RPC.
	AdminServiceGetUniquenessPolicyProcedure = "/buf.alpha.registry.v1alpha1.AdminService/GetUniquenessPolicy"
	// AdminServiceUpdateUniquenessPolicyProcedure is the fully-qualified name of the AdminService's
	// UpdateUniquenessPolicy RPC.
	AdminServiceUpdateUniquenessPolicyProcedure = "/buf.alpha.registry.v1alpha1.AdminService/UpdateUniquenessPolicy"
	// AdminServiceListServerUniquenessCollisionsProcedure is the fully-qualified name of the
	// AdminService's ListServerUniquenessCollisions RPC.
	AdminServiceListServerUniquenessCollisionsProcedure = "/buf.alpha.registry.v1alpha1.AdminService/ListServerUniquenessCollisions"
	// AdminServiceGetAdminEmailsForOrganizationProcedure is the fully-qualified name of the
	// AdminService's GetAdminEmailsForOrganization RPC.
	AdminServiceGetAdminEmailsForOrganizationProcedure = "/buf.alpha.registry.v1alpha1.AdminService/GetAdminEmailsForOrganization"
)

// AdminServiceClient is a client for the buf.alpha.registry.v1alpha1.AdminService service.
type AdminServiceClient interface {
	// ForceDeleteUser forces to delete a user. Resources and organizations that are
	// solely owned by the user will also be deleted.
	ForceDeleteUser(context.Context, *connect_go.Request[v1alpha1.ForceDeleteUserRequest]) (*connect_go.Response[v1alpha1.ForceDeleteUserResponse], error)
	// Update a user's verification status.
	UpdateUserVerificationStatus(context.Context, *connect_go.Request[v1alpha1.UpdateUserVerificationStatusRequest]) (*connect_go.Response[v1alpha1.UpdateUserVerificationStatusResponse], error)
	// Update a organization's verification.
	UpdateOrganizationVerificationStatus(context.Context, *connect_go.Request[v1alpha1.UpdateOrganizationVerificationStatusRequest]) (*connect_go.Response[v1alpha1.UpdateOrganizationVerificationStatusResponse], error)
	// Create a new machine user on the server.
	CreateMachineUser(context.Context, *connect_go.Request[v1alpha1.CreateMachineUserRequest]) (*connect_go.Response[v1alpha1.CreateMachineUserResponse], error)
	// Get breaking change policy for the server.
	GetBreakingChangePolicy(context.Context, *connect_go.Request[v1alpha1.GetBreakingChangePolicyRequest]) (*connect_go.Response[v1alpha1.GetBreakingChangePolicyResponse], error)
	// Update breaking change policy for the server.
	UpdateBreakingChangePolicy(context.Context, *connect_go.Request[v1alpha1.UpdateBreakingChangePolicyRequest]) (*connect_go.Response[v1alpha1.UpdateBreakingChangePolicyResponse], error)
	// Get uniqueness policy for the server.
	GetUniquenessPolicy(context.Context, *connect_go.Request[v1alpha1.GetUniquenessPolicyRequest]) (*connect_go.Response[v1alpha1.GetUniquenessPolicyResponse], error)
	// Update uniqueness policy enforcement for the server.
	UpdateUniquenessPolicy(context.Context, *connect_go.Request[v1alpha1.UpdateUniquenessPolicyRequest]) (*connect_go.Response[v1alpha1.UpdateUniquenessPolicyResponse], error)
	// Get state of uniqueness collisions for the server
	ListServerUniquenessCollisions(context.Context, *connect_go.Request[v1alpha1.ListServerUniquenessCollisionsRequest]) (*connect_go.Response[v1alpha1.ListServerUniquenessCollisionsResponse], error)
	// GetOrganizationAdminEmails gets a list of emails for admins in the organization.
	GetAdminEmailsForOrganization(context.Context, *connect_go.Request[v1alpha1.GetAdminEmailsForOrganizationRequest]) (*connect_go.Response[v1alpha1.GetAdminEmailsForOrganizationResponse], error)
}

// NewAdminServiceClient constructs a client for the buf.alpha.registry.v1alpha1.AdminService
// service. By default, it uses the Connect protocol with the binary Protobuf Codec, asks for
// gzipped responses, and sends uncompressed requests. To use the gRPC or gRPC-Web protocols, supply
// the connect.WithGRPC() or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewAdminServiceClient(httpClient connect_go.HTTPClient, baseURL string, opts ...connect_go.ClientOption) AdminServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &adminServiceClient{
		forceDeleteUser: connect_go.NewClient[v1alpha1.ForceDeleteUserRequest, v1alpha1.ForceDeleteUserResponse](
			httpClient,
			baseURL+AdminServiceForceDeleteUserProcedure,
			connect_go.WithIdempotency(connect_go.IdempotencyIdempotent),
			connect_go.WithClientOptions(opts...),
		),
		updateUserVerificationStatus: connect_go.NewClient[v1alpha1.UpdateUserVerificationStatusRequest, v1alpha1.UpdateUserVerificationStatusResponse](
			httpClient,
			baseURL+AdminServiceUpdateUserVerificationStatusProcedure,
			opts...,
		),
		updateOrganizationVerificationStatus: connect_go.NewClient[v1alpha1.UpdateOrganizationVerificationStatusRequest, v1alpha1.UpdateOrganizationVerificationStatusResponse](
			httpClient,
			baseURL+AdminServiceUpdateOrganizationVerificationStatusProcedure,
			opts...,
		),
		createMachineUser: connect_go.NewClient[v1alpha1.CreateMachineUserRequest, v1alpha1.CreateMachineUserResponse](
			httpClient,
			baseURL+AdminServiceCreateMachineUserProcedure,
			connect_go.WithIdempotency(connect_go.IdempotencyIdempotent),
			connect_go.WithClientOptions(opts...),
		),
		getBreakingChangePolicy: connect_go.NewClient[v1alpha1.GetBreakingChangePolicyRequest, v1alpha1.GetBreakingChangePolicyResponse](
			httpClient,
			baseURL+AdminServiceGetBreakingChangePolicyProcedure,
			connect_go.WithIdempotency(connect_go.IdempotencyNoSideEffects),
			connect_go.WithClientOptions(opts...),
		),
		updateBreakingChangePolicy: connect_go.NewClient[v1alpha1.UpdateBreakingChangePolicyRequest, v1alpha1.UpdateBreakingChangePolicyResponse](
			httpClient,
			baseURL+AdminServiceUpdateBreakingChangePolicyProcedure,
			opts...,
		),
		getUniquenessPolicy: connect_go.NewClient[v1alpha1.GetUniquenessPolicyRequest, v1alpha1.GetUniquenessPolicyResponse](
			httpClient,
			baseURL+AdminServiceGetUniquenessPolicyProcedure,
			connect_go.WithIdempotency(connect_go.IdempotencyNoSideEffects),
			connect_go.WithClientOptions(opts...),
		),
		updateUniquenessPolicy: connect_go.NewClient[v1alpha1.UpdateUniquenessPolicyRequest, v1alpha1.UpdateUniquenessPolicyResponse](
			httpClient,
			baseURL+AdminServiceUpdateUniquenessPolicyProcedure,
			opts...,
		),
		listServerUniquenessCollisions: connect_go.NewClient[v1alpha1.ListServerUniquenessCollisionsRequest, v1alpha1.ListServerUniquenessCollisionsResponse](
			httpClient,
			baseURL+AdminServiceListServerUniquenessCollisionsProcedure,
			connect_go.WithIdempotency(connect_go.IdempotencyNoSideEffects),
			connect_go.WithClientOptions(opts...),
		),
		getAdminEmailsForOrganization: connect_go.NewClient[v1alpha1.GetAdminEmailsForOrganizationRequest, v1alpha1.GetAdminEmailsForOrganizationResponse](
			httpClient,
			baseURL+AdminServiceGetAdminEmailsForOrganizationProcedure,
			connect_go.WithIdempotency(connect_go.IdempotencyIdempotent),
			connect_go.WithClientOptions(opts...),
		),
	}
}

// adminServiceClient implements AdminServiceClient.
type adminServiceClient struct {
	forceDeleteUser                      *connect_go.Client[v1alpha1.ForceDeleteUserRequest, v1alpha1.ForceDeleteUserResponse]
	updateUserVerificationStatus         *connect_go.Client[v1alpha1.UpdateUserVerificationStatusRequest, v1alpha1.UpdateUserVerificationStatusResponse]
	updateOrganizationVerificationStatus *connect_go.Client[v1alpha1.UpdateOrganizationVerificationStatusRequest, v1alpha1.UpdateOrganizationVerificationStatusResponse]
	createMachineUser                    *connect_go.Client[v1alpha1.CreateMachineUserRequest, v1alpha1.CreateMachineUserResponse]
	getBreakingChangePolicy              *connect_go.Client[v1alpha1.GetBreakingChangePolicyRequest, v1alpha1.GetBreakingChangePolicyResponse]
	updateBreakingChangePolicy           *connect_go.Client[v1alpha1.UpdateBreakingChangePolicyRequest, v1alpha1.UpdateBreakingChangePolicyResponse]
	getUniquenessPolicy                  *connect_go.Client[v1alpha1.GetUniquenessPolicyRequest, v1alpha1.GetUniquenessPolicyResponse]
	updateUniquenessPolicy               *connect_go.Client[v1alpha1.UpdateUniquenessPolicyRequest, v1alpha1.UpdateUniquenessPolicyResponse]
	listServerUniquenessCollisions       *connect_go.Client[v1alpha1.ListServerUniquenessCollisionsRequest, v1alpha1.ListServerUniquenessCollisionsResponse]
	getAdminEmailsForOrganization        *connect_go.Client[v1alpha1.GetAdminEmailsForOrganizationRequest, v1alpha1.GetAdminEmailsForOrganizationResponse]
}

// ForceDeleteUser calls buf.alpha.registry.v1alpha1.AdminService.ForceDeleteUser.
func (c *adminServiceClient) ForceDeleteUser(ctx context.Context, req *connect_go.Request[v1alpha1.ForceDeleteUserRequest]) (*connect_go.Response[v1alpha1.ForceDeleteUserResponse], error) {
	return c.forceDeleteUser.CallUnary(ctx, req)
}

// UpdateUserVerificationStatus calls
// buf.alpha.registry.v1alpha1.AdminService.UpdateUserVerificationStatus.
func (c *adminServiceClient) UpdateUserVerificationStatus(ctx context.Context, req *connect_go.Request[v1alpha1.UpdateUserVerificationStatusRequest]) (*connect_go.Response[v1alpha1.UpdateUserVerificationStatusResponse], error) {
	return c.updateUserVerificationStatus.CallUnary(ctx, req)
}

// UpdateOrganizationVerificationStatus calls
// buf.alpha.registry.v1alpha1.AdminService.UpdateOrganizationVerificationStatus.
func (c *adminServiceClient) UpdateOrganizationVerificationStatus(ctx context.Context, req *connect_go.Request[v1alpha1.UpdateOrganizationVerificationStatusRequest]) (*connect_go.Response[v1alpha1.UpdateOrganizationVerificationStatusResponse], error) {
	return c.updateOrganizationVerificationStatus.CallUnary(ctx, req)
}

// CreateMachineUser calls buf.alpha.registry.v1alpha1.AdminService.CreateMachineUser.
func (c *adminServiceClient) CreateMachineUser(ctx context.Context, req *connect_go.Request[v1alpha1.CreateMachineUserRequest]) (*connect_go.Response[v1alpha1.CreateMachineUserResponse], error) {
	return c.createMachineUser.CallUnary(ctx, req)
}

// GetBreakingChangePolicy calls buf.alpha.registry.v1alpha1.AdminService.GetBreakingChangePolicy.
func (c *adminServiceClient) GetBreakingChangePolicy(ctx context.Context, req *connect_go.Request[v1alpha1.GetBreakingChangePolicyRequest]) (*connect_go.Response[v1alpha1.GetBreakingChangePolicyResponse], error) {
	return c.getBreakingChangePolicy.CallUnary(ctx, req)
}

// UpdateBreakingChangePolicy calls
// buf.alpha.registry.v1alpha1.AdminService.UpdateBreakingChangePolicy.
func (c *adminServiceClient) UpdateBreakingChangePolicy(ctx context.Context, req *connect_go.Request[v1alpha1.UpdateBreakingChangePolicyRequest]) (*connect_go.Response[v1alpha1.UpdateBreakingChangePolicyResponse], error) {
	return c.updateBreakingChangePolicy.CallUnary(ctx, req)
}

// GetUniquenessPolicy calls buf.alpha.registry.v1alpha1.AdminService.GetUniquenessPolicy.
func (c *adminServiceClient) GetUniquenessPolicy(ctx context.Context, req *connect_go.Request[v1alpha1.GetUniquenessPolicyRequest]) (*connect_go.Response[v1alpha1.GetUniquenessPolicyResponse], error) {
	return c.getUniquenessPolicy.CallUnary(ctx, req)
}

// UpdateUniquenessPolicy calls buf.alpha.registry.v1alpha1.AdminService.UpdateUniquenessPolicy.
func (c *adminServiceClient) UpdateUniquenessPolicy(ctx context.Context, req *connect_go.Request[v1alpha1.UpdateUniquenessPolicyRequest]) (*connect_go.Response[v1alpha1.UpdateUniquenessPolicyResponse], error) {
	return c.updateUniquenessPolicy.CallUnary(ctx, req)
}

// ListServerUniquenessCollisions calls
// buf.alpha.registry.v1alpha1.AdminService.ListServerUniquenessCollisions.
func (c *adminServiceClient) ListServerUniquenessCollisions(ctx context.Context, req *connect_go.Request[v1alpha1.ListServerUniquenessCollisionsRequest]) (*connect_go.Response[v1alpha1.ListServerUniquenessCollisionsResponse], error) {
	return c.listServerUniquenessCollisions.CallUnary(ctx, req)
}

// GetAdminEmailsForOrganization calls
// buf.alpha.registry.v1alpha1.AdminService.GetAdminEmailsForOrganization.
func (c *adminServiceClient) GetAdminEmailsForOrganization(ctx context.Context, req *connect_go.Request[v1alpha1.GetAdminEmailsForOrganizationRequest]) (*connect_go.Response[v1alpha1.GetAdminEmailsForOrganizationResponse], error) {
	return c.getAdminEmailsForOrganization.CallUnary(ctx, req)
}

// AdminServiceHandler is an implementation of the buf.alpha.registry.v1alpha1.AdminService service.
type AdminServiceHandler interface {
	// ForceDeleteUser forces to delete a user. Resources and organizations that are
	// solely owned by the user will also be deleted.
	ForceDeleteUser(context.Context, *connect_go.Request[v1alpha1.ForceDeleteUserRequest]) (*connect_go.Response[v1alpha1.ForceDeleteUserResponse], error)
	// Update a user's verification status.
	UpdateUserVerificationStatus(context.Context, *connect_go.Request[v1alpha1.UpdateUserVerificationStatusRequest]) (*connect_go.Response[v1alpha1.UpdateUserVerificationStatusResponse], error)
	// Update a organization's verification.
	UpdateOrganizationVerificationStatus(context.Context, *connect_go.Request[v1alpha1.UpdateOrganizationVerificationStatusRequest]) (*connect_go.Response[v1alpha1.UpdateOrganizationVerificationStatusResponse], error)
	// Create a new machine user on the server.
	CreateMachineUser(context.Context, *connect_go.Request[v1alpha1.CreateMachineUserRequest]) (*connect_go.Response[v1alpha1.CreateMachineUserResponse], error)
	// Get breaking change policy for the server.
	GetBreakingChangePolicy(context.Context, *connect_go.Request[v1alpha1.GetBreakingChangePolicyRequest]) (*connect_go.Response[v1alpha1.GetBreakingChangePolicyResponse], error)
	// Update breaking change policy for the server.
	UpdateBreakingChangePolicy(context.Context, *connect_go.Request[v1alpha1.UpdateBreakingChangePolicyRequest]) (*connect_go.Response[v1alpha1.UpdateBreakingChangePolicyResponse], error)
	// Get uniqueness policy for the server.
	GetUniquenessPolicy(context.Context, *connect_go.Request[v1alpha1.GetUniquenessPolicyRequest]) (*connect_go.Response[v1alpha1.GetUniquenessPolicyResponse], error)
	// Update uniqueness policy enforcement for the server.
	UpdateUniquenessPolicy(context.Context, *connect_go.Request[v1alpha1.UpdateUniquenessPolicyRequest]) (*connect_go.Response[v1alpha1.UpdateUniquenessPolicyResponse], error)
	// Get state of uniqueness collisions for the server
	ListServerUniquenessCollisions(context.Context, *connect_go.Request[v1alpha1.ListServerUniquenessCollisionsRequest]) (*connect_go.Response[v1alpha1.ListServerUniquenessCollisionsResponse], error)
	// GetOrganizationAdminEmails gets a list of emails for admins in the organization.
	GetAdminEmailsForOrganization(context.Context, *connect_go.Request[v1alpha1.GetAdminEmailsForOrganizationRequest]) (*connect_go.Response[v1alpha1.GetAdminEmailsForOrganizationResponse], error)
}

// NewAdminServiceHandler builds an HTTP handler from the service implementation. It returns the
// path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewAdminServiceHandler(svc AdminServiceHandler, opts ...connect_go.HandlerOption) (string, http.Handler) {
	adminServiceForceDeleteUserHandler := connect_go.NewUnaryHandler(
		AdminServiceForceDeleteUserProcedure,
		svc.ForceDeleteUser,
		connect_go.WithIdempotency(connect_go.IdempotencyIdempotent),
		connect_go.WithHandlerOptions(opts...),
	)
	adminServiceUpdateUserVerificationStatusHandler := connect_go.NewUnaryHandler(
		AdminServiceUpdateUserVerificationStatusProcedure,
		svc.UpdateUserVerificationStatus,
		opts...,
	)
	adminServiceUpdateOrganizationVerificationStatusHandler := connect_go.NewUnaryHandler(
		AdminServiceUpdateOrganizationVerificationStatusProcedure,
		svc.UpdateOrganizationVerificationStatus,
		opts...,
	)
	adminServiceCreateMachineUserHandler := connect_go.NewUnaryHandler(
		AdminServiceCreateMachineUserProcedure,
		svc.CreateMachineUser,
		connect_go.WithIdempotency(connect_go.IdempotencyIdempotent),
		connect_go.WithHandlerOptions(opts...),
	)
	adminServiceGetBreakingChangePolicyHandler := connect_go.NewUnaryHandler(
		AdminServiceGetBreakingChangePolicyProcedure,
		svc.GetBreakingChangePolicy,
		connect_go.WithIdempotency(connect_go.IdempotencyNoSideEffects),
		connect_go.WithHandlerOptions(opts...),
	)
	adminServiceUpdateBreakingChangePolicyHandler := connect_go.NewUnaryHandler(
		AdminServiceUpdateBreakingChangePolicyProcedure,
		svc.UpdateBreakingChangePolicy,
		opts...,
	)
	adminServiceGetUniquenessPolicyHandler := connect_go.NewUnaryHandler(
		AdminServiceGetUniquenessPolicyProcedure,
		svc.GetUniquenessPolicy,
		connect_go.WithIdempotency(connect_go.IdempotencyNoSideEffects),
		connect_go.WithHandlerOptions(opts...),
	)
	adminServiceUpdateUniquenessPolicyHandler := connect_go.NewUnaryHandler(
		AdminServiceUpdateUniquenessPolicyProcedure,
		svc.UpdateUniquenessPolicy,
		opts...,
	)
	adminServiceListServerUniquenessCollisionsHandler := connect_go.NewUnaryHandler(
		AdminServiceListServerUniquenessCollisionsProcedure,
		svc.ListServerUniquenessCollisions,
		connect_go.WithIdempotency(connect_go.IdempotencyNoSideEffects),
		connect_go.WithHandlerOptions(opts...),
	)
	adminServiceGetAdminEmailsForOrganizationHandler := connect_go.NewUnaryHandler(
		AdminServiceGetAdminEmailsForOrganizationProcedure,
		svc.GetAdminEmailsForOrganization,
		connect_go.WithIdempotency(connect_go.IdempotencyIdempotent),
		connect_go.WithHandlerOptions(opts...),
	)
	return "/buf.alpha.registry.v1alpha1.AdminService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case AdminServiceForceDeleteUserProcedure:
			adminServiceForceDeleteUserHandler.ServeHTTP(w, r)
		case AdminServiceUpdateUserVerificationStatusProcedure:
			adminServiceUpdateUserVerificationStatusHandler.ServeHTTP(w, r)
		case AdminServiceUpdateOrganizationVerificationStatusProcedure:
			adminServiceUpdateOrganizationVerificationStatusHandler.ServeHTTP(w, r)
		case AdminServiceCreateMachineUserProcedure:
			adminServiceCreateMachineUserHandler.ServeHTTP(w, r)
		case AdminServiceGetBreakingChangePolicyProcedure:
			adminServiceGetBreakingChangePolicyHandler.ServeHTTP(w, r)
		case AdminServiceUpdateBreakingChangePolicyProcedure:
			adminServiceUpdateBreakingChangePolicyHandler.ServeHTTP(w, r)
		case AdminServiceGetUniquenessPolicyProcedure:
			adminServiceGetUniquenessPolicyHandler.ServeHTTP(w, r)
		case AdminServiceUpdateUniquenessPolicyProcedure:
			adminServiceUpdateUniquenessPolicyHandler.ServeHTTP(w, r)
		case AdminServiceListServerUniquenessCollisionsProcedure:
			adminServiceListServerUniquenessCollisionsHandler.ServeHTTP(w, r)
		case AdminServiceGetAdminEmailsForOrganizationProcedure:
			adminServiceGetAdminEmailsForOrganizationHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedAdminServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedAdminServiceHandler struct{}

func (UnimplementedAdminServiceHandler) ForceDeleteUser(context.Context, *connect_go.Request[v1alpha1.ForceDeleteUserRequest]) (*connect_go.Response[v1alpha1.ForceDeleteUserResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.AdminService.ForceDeleteUser is not implemented"))
}

func (UnimplementedAdminServiceHandler) UpdateUserVerificationStatus(context.Context, *connect_go.Request[v1alpha1.UpdateUserVerificationStatusRequest]) (*connect_go.Response[v1alpha1.UpdateUserVerificationStatusResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.AdminService.UpdateUserVerificationStatus is not implemented"))
}

func (UnimplementedAdminServiceHandler) UpdateOrganizationVerificationStatus(context.Context, *connect_go.Request[v1alpha1.UpdateOrganizationVerificationStatusRequest]) (*connect_go.Response[v1alpha1.UpdateOrganizationVerificationStatusResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.AdminService.UpdateOrganizationVerificationStatus is not implemented"))
}

func (UnimplementedAdminServiceHandler) CreateMachineUser(context.Context, *connect_go.Request[v1alpha1.CreateMachineUserRequest]) (*connect_go.Response[v1alpha1.CreateMachineUserResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.AdminService.CreateMachineUser is not implemented"))
}

func (UnimplementedAdminServiceHandler) GetBreakingChangePolicy(context.Context, *connect_go.Request[v1alpha1.GetBreakingChangePolicyRequest]) (*connect_go.Response[v1alpha1.GetBreakingChangePolicyResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.AdminService.GetBreakingChangePolicy is not implemented"))
}

func (UnimplementedAdminServiceHandler) UpdateBreakingChangePolicy(context.Context, *connect_go.Request[v1alpha1.UpdateBreakingChangePolicyRequest]) (*connect_go.Response[v1alpha1.UpdateBreakingChangePolicyResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.AdminService.UpdateBreakingChangePolicy is not implemented"))
}

func (UnimplementedAdminServiceHandler) GetUniquenessPolicy(context.Context, *connect_go.Request[v1alpha1.GetUniquenessPolicyRequest]) (*connect_go.Response[v1alpha1.GetUniquenessPolicyResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.AdminService.GetUniquenessPolicy is not implemented"))
}

func (UnimplementedAdminServiceHandler) UpdateUniquenessPolicy(context.Context, *connect_go.Request[v1alpha1.UpdateUniquenessPolicyRequest]) (*connect_go.Response[v1alpha1.UpdateUniquenessPolicyResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.AdminService.UpdateUniquenessPolicy is not implemented"))
}

func (UnimplementedAdminServiceHandler) ListServerUniquenessCollisions(context.Context, *connect_go.Request[v1alpha1.ListServerUniquenessCollisionsRequest]) (*connect_go.Response[v1alpha1.ListServerUniquenessCollisionsResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.AdminService.ListServerUniquenessCollisions is not implemented"))
}

func (UnimplementedAdminServiceHandler) GetAdminEmailsForOrganization(context.Context, *connect_go.Request[v1alpha1.GetAdminEmailsForOrganizationRequest]) (*connect_go.Response[v1alpha1.GetAdminEmailsForOrganizationResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.AdminService.GetAdminEmailsForOrganization is not implemented"))
}
