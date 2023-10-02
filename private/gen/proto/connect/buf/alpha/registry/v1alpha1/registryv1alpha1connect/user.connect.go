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
// Source: buf/alpha/registry/v1alpha1/user.proto

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
	// UserServiceName is the fully-qualified name of the UserService service.
	UserServiceName = "buf.alpha.registry.v1alpha1.UserService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// UserServiceCreateUserProcedure is the fully-qualified name of the UserService's CreateUser RPC.
	UserServiceCreateUserProcedure = "/buf.alpha.registry.v1alpha1.UserService/CreateUser"
	// UserServiceGetUserProcedure is the fully-qualified name of the UserService's GetUser RPC.
	UserServiceGetUserProcedure = "/buf.alpha.registry.v1alpha1.UserService/GetUser"
	// UserServiceGetUserByUsernameProcedure is the fully-qualified name of the UserService's
	// GetUserByUsername RPC.
	UserServiceGetUserByUsernameProcedure = "/buf.alpha.registry.v1alpha1.UserService/GetUserByUsername"
	// UserServiceListUsersProcedure is the fully-qualified name of the UserService's ListUsers RPC.
	UserServiceListUsersProcedure = "/buf.alpha.registry.v1alpha1.UserService/ListUsers"
	// UserServiceListOrganizationUsersProcedure is the fully-qualified name of the UserService's
	// ListOrganizationUsers RPC.
	UserServiceListOrganizationUsersProcedure = "/buf.alpha.registry.v1alpha1.UserService/ListOrganizationUsers"
	// UserServiceDeleteUserProcedure is the fully-qualified name of the UserService's DeleteUser RPC.
	UserServiceDeleteUserProcedure = "/buf.alpha.registry.v1alpha1.UserService/DeleteUser"
	// UserServiceDeactivateUserProcedure is the fully-qualified name of the UserService's
	// DeactivateUser RPC.
	UserServiceDeactivateUserProcedure = "/buf.alpha.registry.v1alpha1.UserService/DeactivateUser"
	// UserServiceUpdateUserServerRoleProcedure is the fully-qualified name of the UserService's
	// UpdateUserServerRole RPC.
	UserServiceUpdateUserServerRoleProcedure = "/buf.alpha.registry.v1alpha1.UserService/UpdateUserServerRole"
	// UserServiceCountUsersProcedure is the fully-qualified name of the UserService's CountUsers RPC.
	UserServiceCountUsersProcedure = "/buf.alpha.registry.v1alpha1.UserService/CountUsers"
	// UserServiceUpdateUserSettingsProcedure is the fully-qualified name of the UserService's
	// UpdateUserSettings RPC.
	UserServiceUpdateUserSettingsProcedure = "/buf.alpha.registry.v1alpha1.UserService/UpdateUserSettings"
	// UserServiceGetUserPluginPreferencesProcedure is the fully-qualified name of the UserService's
	// GetUserPluginPreferences RPC.
	UserServiceGetUserPluginPreferencesProcedure = "/buf.alpha.registry.v1alpha1.UserService/GetUserPluginPreferences"
	// UserServiceUpdateUserPluginPreferenceProcedure is the fully-qualified name of the UserService's
	// UpdateUserPluginPreference RPC.
	UserServiceUpdateUserPluginPreferenceProcedure = "/buf.alpha.registry.v1alpha1.UserService/UpdateUserPluginPreference"
)

// UserServiceClient is a client for the buf.alpha.registry.v1alpha1.UserService service.
type UserServiceClient interface {
	// CreateUser creates a new user with the given username.
	CreateUser(context.Context, *connect.Request[v1alpha1.CreateUserRequest]) (*connect.Response[v1alpha1.CreateUserResponse], error)
	// GetUser gets a user by ID.
	GetUser(context.Context, *connect.Request[v1alpha1.GetUserRequest]) (*connect.Response[v1alpha1.GetUserResponse], error)
	// GetUserByUsername gets a user by username.
	GetUserByUsername(context.Context, *connect.Request[v1alpha1.GetUserByUsernameRequest]) (*connect.Response[v1alpha1.GetUserByUsernameResponse], error)
	// ListUsers lists all users.
	ListUsers(context.Context, *connect.Request[v1alpha1.ListUsersRequest]) (*connect.Response[v1alpha1.ListUsersResponse], error)
	// ListOrganizationUsers lists all users for an organization.
	// TODO: #663 move this to organization service
	ListOrganizationUsers(context.Context, *connect.Request[v1alpha1.ListOrganizationUsersRequest]) (*connect.Response[v1alpha1.ListOrganizationUsersResponse], error)
	// DeleteUser deletes a user.
	DeleteUser(context.Context, *connect.Request[v1alpha1.DeleteUserRequest]) (*connect.Response[v1alpha1.DeleteUserResponse], error)
	// Deactivate user deactivates a user.
	DeactivateUser(context.Context, *connect.Request[v1alpha1.DeactivateUserRequest]) (*connect.Response[v1alpha1.DeactivateUserResponse], error)
	// UpdateUserServerRole update the role of an user in the server.
	UpdateUserServerRole(context.Context, *connect.Request[v1alpha1.UpdateUserServerRoleRequest]) (*connect.Response[v1alpha1.UpdateUserServerRoleResponse], error)
	// CountUsers returns the number of users in the server by the user state provided.
	CountUsers(context.Context, *connect.Request[v1alpha1.CountUsersRequest]) (*connect.Response[v1alpha1.CountUsersResponse], error)
	// UpdateUserSettings update the user settings including description.
	UpdateUserSettings(context.Context, *connect.Request[v1alpha1.UpdateUserSettingsRequest]) (*connect.Response[v1alpha1.UpdateUserSettingsResponse], error)
	// GetUserPluginPreferences gets plugin preferences for the user.
	GetUserPluginPreferences(context.Context, *connect.Request[v1alpha1.GetUserPluginPreferencesRequest]) (*connect.Response[v1alpha1.GetUserPluginPreferencesResponse], error)
	// UpdateUserPluginPreference updates the user plugin preferences.
	UpdateUserPluginPreference(context.Context, *connect.Request[v1alpha1.UpdateUserPluginPreferenceRequest]) (*connect.Response[v1alpha1.UpdateUserPluginPreferenceResponse], error)
}

// NewUserServiceClient constructs a client for the buf.alpha.registry.v1alpha1.UserService service.
// By default, it uses the Connect protocol with the binary Protobuf Codec, asks for gzipped
// responses, and sends uncompressed requests. To use the gRPC or gRPC-Web protocols, supply the
// connect.WithGRPC() or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewUserServiceClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) UserServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &userServiceClient{
		createUser: connect.NewClient[v1alpha1.CreateUserRequest, v1alpha1.CreateUserResponse](
			httpClient,
			baseURL+UserServiceCreateUserProcedure,
			connect.WithIdempotency(connect.IdempotencyIdempotent),
			connect.WithClientOptions(opts...),
		),
		getUser: connect.NewClient[v1alpha1.GetUserRequest, v1alpha1.GetUserResponse](
			httpClient,
			baseURL+UserServiceGetUserProcedure,
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
		getUserByUsername: connect.NewClient[v1alpha1.GetUserByUsernameRequest, v1alpha1.GetUserByUsernameResponse](
			httpClient,
			baseURL+UserServiceGetUserByUsernameProcedure,
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
		listUsers: connect.NewClient[v1alpha1.ListUsersRequest, v1alpha1.ListUsersResponse](
			httpClient,
			baseURL+UserServiceListUsersProcedure,
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
		listOrganizationUsers: connect.NewClient[v1alpha1.ListOrganizationUsersRequest, v1alpha1.ListOrganizationUsersResponse](
			httpClient,
			baseURL+UserServiceListOrganizationUsersProcedure,
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
		deleteUser: connect.NewClient[v1alpha1.DeleteUserRequest, v1alpha1.DeleteUserResponse](
			httpClient,
			baseURL+UserServiceDeleteUserProcedure,
			connect.WithIdempotency(connect.IdempotencyIdempotent),
			connect.WithClientOptions(opts...),
		),
		deactivateUser: connect.NewClient[v1alpha1.DeactivateUserRequest, v1alpha1.DeactivateUserResponse](
			httpClient,
			baseURL+UserServiceDeactivateUserProcedure,
			connect.WithIdempotency(connect.IdempotencyIdempotent),
			connect.WithClientOptions(opts...),
		),
		updateUserServerRole: connect.NewClient[v1alpha1.UpdateUserServerRoleRequest, v1alpha1.UpdateUserServerRoleResponse](
			httpClient,
			baseURL+UserServiceUpdateUserServerRoleProcedure,
			opts...,
		),
		countUsers: connect.NewClient[v1alpha1.CountUsersRequest, v1alpha1.CountUsersResponse](
			httpClient,
			baseURL+UserServiceCountUsersProcedure,
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
		updateUserSettings: connect.NewClient[v1alpha1.UpdateUserSettingsRequest, v1alpha1.UpdateUserSettingsResponse](
			httpClient,
			baseURL+UserServiceUpdateUserSettingsProcedure,
			opts...,
		),
		getUserPluginPreferences: connect.NewClient[v1alpha1.GetUserPluginPreferencesRequest, v1alpha1.GetUserPluginPreferencesResponse](
			httpClient,
			baseURL+UserServiceGetUserPluginPreferencesProcedure,
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
		updateUserPluginPreference: connect.NewClient[v1alpha1.UpdateUserPluginPreferenceRequest, v1alpha1.UpdateUserPluginPreferenceResponse](
			httpClient,
			baseURL+UserServiceUpdateUserPluginPreferenceProcedure,
			opts...,
		),
	}
}

// userServiceClient implements UserServiceClient.
type userServiceClient struct {
	createUser                 *connect.Client[v1alpha1.CreateUserRequest, v1alpha1.CreateUserResponse]
	getUser                    *connect.Client[v1alpha1.GetUserRequest, v1alpha1.GetUserResponse]
	getUserByUsername          *connect.Client[v1alpha1.GetUserByUsernameRequest, v1alpha1.GetUserByUsernameResponse]
	listUsers                  *connect.Client[v1alpha1.ListUsersRequest, v1alpha1.ListUsersResponse]
	listOrganizationUsers      *connect.Client[v1alpha1.ListOrganizationUsersRequest, v1alpha1.ListOrganizationUsersResponse]
	deleteUser                 *connect.Client[v1alpha1.DeleteUserRequest, v1alpha1.DeleteUserResponse]
	deactivateUser             *connect.Client[v1alpha1.DeactivateUserRequest, v1alpha1.DeactivateUserResponse]
	updateUserServerRole       *connect.Client[v1alpha1.UpdateUserServerRoleRequest, v1alpha1.UpdateUserServerRoleResponse]
	countUsers                 *connect.Client[v1alpha1.CountUsersRequest, v1alpha1.CountUsersResponse]
	updateUserSettings         *connect.Client[v1alpha1.UpdateUserSettingsRequest, v1alpha1.UpdateUserSettingsResponse]
	getUserPluginPreferences   *connect.Client[v1alpha1.GetUserPluginPreferencesRequest, v1alpha1.GetUserPluginPreferencesResponse]
	updateUserPluginPreference *connect.Client[v1alpha1.UpdateUserPluginPreferenceRequest, v1alpha1.UpdateUserPluginPreferenceResponse]
}

// CreateUser calls buf.alpha.registry.v1alpha1.UserService.CreateUser.
func (c *userServiceClient) CreateUser(ctx context.Context, req *connect.Request[v1alpha1.CreateUserRequest]) (*connect.Response[v1alpha1.CreateUserResponse], error) {
	return c.createUser.CallUnary(ctx, req)
}

// GetUser calls buf.alpha.registry.v1alpha1.UserService.GetUser.
func (c *userServiceClient) GetUser(ctx context.Context, req *connect.Request[v1alpha1.GetUserRequest]) (*connect.Response[v1alpha1.GetUserResponse], error) {
	return c.getUser.CallUnary(ctx, req)
}

// GetUserByUsername calls buf.alpha.registry.v1alpha1.UserService.GetUserByUsername.
func (c *userServiceClient) GetUserByUsername(ctx context.Context, req *connect.Request[v1alpha1.GetUserByUsernameRequest]) (*connect.Response[v1alpha1.GetUserByUsernameResponse], error) {
	return c.getUserByUsername.CallUnary(ctx, req)
}

// ListUsers calls buf.alpha.registry.v1alpha1.UserService.ListUsers.
func (c *userServiceClient) ListUsers(ctx context.Context, req *connect.Request[v1alpha1.ListUsersRequest]) (*connect.Response[v1alpha1.ListUsersResponse], error) {
	return c.listUsers.CallUnary(ctx, req)
}

// ListOrganizationUsers calls buf.alpha.registry.v1alpha1.UserService.ListOrganizationUsers.
func (c *userServiceClient) ListOrganizationUsers(ctx context.Context, req *connect.Request[v1alpha1.ListOrganizationUsersRequest]) (*connect.Response[v1alpha1.ListOrganizationUsersResponse], error) {
	return c.listOrganizationUsers.CallUnary(ctx, req)
}

// DeleteUser calls buf.alpha.registry.v1alpha1.UserService.DeleteUser.
func (c *userServiceClient) DeleteUser(ctx context.Context, req *connect.Request[v1alpha1.DeleteUserRequest]) (*connect.Response[v1alpha1.DeleteUserResponse], error) {
	return c.deleteUser.CallUnary(ctx, req)
}

// DeactivateUser calls buf.alpha.registry.v1alpha1.UserService.DeactivateUser.
func (c *userServiceClient) DeactivateUser(ctx context.Context, req *connect.Request[v1alpha1.DeactivateUserRequest]) (*connect.Response[v1alpha1.DeactivateUserResponse], error) {
	return c.deactivateUser.CallUnary(ctx, req)
}

// UpdateUserServerRole calls buf.alpha.registry.v1alpha1.UserService.UpdateUserServerRole.
func (c *userServiceClient) UpdateUserServerRole(ctx context.Context, req *connect.Request[v1alpha1.UpdateUserServerRoleRequest]) (*connect.Response[v1alpha1.UpdateUserServerRoleResponse], error) {
	return c.updateUserServerRole.CallUnary(ctx, req)
}

// CountUsers calls buf.alpha.registry.v1alpha1.UserService.CountUsers.
func (c *userServiceClient) CountUsers(ctx context.Context, req *connect.Request[v1alpha1.CountUsersRequest]) (*connect.Response[v1alpha1.CountUsersResponse], error) {
	return c.countUsers.CallUnary(ctx, req)
}

// UpdateUserSettings calls buf.alpha.registry.v1alpha1.UserService.UpdateUserSettings.
func (c *userServiceClient) UpdateUserSettings(ctx context.Context, req *connect.Request[v1alpha1.UpdateUserSettingsRequest]) (*connect.Response[v1alpha1.UpdateUserSettingsResponse], error) {
	return c.updateUserSettings.CallUnary(ctx, req)
}

// GetUserPluginPreferences calls buf.alpha.registry.v1alpha1.UserService.GetUserPluginPreferences.
func (c *userServiceClient) GetUserPluginPreferences(ctx context.Context, req *connect.Request[v1alpha1.GetUserPluginPreferencesRequest]) (*connect.Response[v1alpha1.GetUserPluginPreferencesResponse], error) {
	return c.getUserPluginPreferences.CallUnary(ctx, req)
}

// UpdateUserPluginPreference calls
// buf.alpha.registry.v1alpha1.UserService.UpdateUserPluginPreference.
func (c *userServiceClient) UpdateUserPluginPreference(ctx context.Context, req *connect.Request[v1alpha1.UpdateUserPluginPreferenceRequest]) (*connect.Response[v1alpha1.UpdateUserPluginPreferenceResponse], error) {
	return c.updateUserPluginPreference.CallUnary(ctx, req)
}

// UserServiceHandler is an implementation of the buf.alpha.registry.v1alpha1.UserService service.
type UserServiceHandler interface {
	// CreateUser creates a new user with the given username.
	CreateUser(context.Context, *connect.Request[v1alpha1.CreateUserRequest]) (*connect.Response[v1alpha1.CreateUserResponse], error)
	// GetUser gets a user by ID.
	GetUser(context.Context, *connect.Request[v1alpha1.GetUserRequest]) (*connect.Response[v1alpha1.GetUserResponse], error)
	// GetUserByUsername gets a user by username.
	GetUserByUsername(context.Context, *connect.Request[v1alpha1.GetUserByUsernameRequest]) (*connect.Response[v1alpha1.GetUserByUsernameResponse], error)
	// ListUsers lists all users.
	ListUsers(context.Context, *connect.Request[v1alpha1.ListUsersRequest]) (*connect.Response[v1alpha1.ListUsersResponse], error)
	// ListOrganizationUsers lists all users for an organization.
	// TODO: #663 move this to organization service
	ListOrganizationUsers(context.Context, *connect.Request[v1alpha1.ListOrganizationUsersRequest]) (*connect.Response[v1alpha1.ListOrganizationUsersResponse], error)
	// DeleteUser deletes a user.
	DeleteUser(context.Context, *connect.Request[v1alpha1.DeleteUserRequest]) (*connect.Response[v1alpha1.DeleteUserResponse], error)
	// Deactivate user deactivates a user.
	DeactivateUser(context.Context, *connect.Request[v1alpha1.DeactivateUserRequest]) (*connect.Response[v1alpha1.DeactivateUserResponse], error)
	// UpdateUserServerRole update the role of an user in the server.
	UpdateUserServerRole(context.Context, *connect.Request[v1alpha1.UpdateUserServerRoleRequest]) (*connect.Response[v1alpha1.UpdateUserServerRoleResponse], error)
	// CountUsers returns the number of users in the server by the user state provided.
	CountUsers(context.Context, *connect.Request[v1alpha1.CountUsersRequest]) (*connect.Response[v1alpha1.CountUsersResponse], error)
	// UpdateUserSettings update the user settings including description.
	UpdateUserSettings(context.Context, *connect.Request[v1alpha1.UpdateUserSettingsRequest]) (*connect.Response[v1alpha1.UpdateUserSettingsResponse], error)
	// GetUserPluginPreferences gets plugin preferences for the user.
	GetUserPluginPreferences(context.Context, *connect.Request[v1alpha1.GetUserPluginPreferencesRequest]) (*connect.Response[v1alpha1.GetUserPluginPreferencesResponse], error)
	// UpdateUserPluginPreference updates the user plugin preferences.
	UpdateUserPluginPreference(context.Context, *connect.Request[v1alpha1.UpdateUserPluginPreferenceRequest]) (*connect.Response[v1alpha1.UpdateUserPluginPreferenceResponse], error)
}

// NewUserServiceHandler builds an HTTP handler from the service implementation. It returns the path
// on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewUserServiceHandler(svc UserServiceHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	userServiceCreateUserHandler := connect.NewUnaryHandler(
		UserServiceCreateUserProcedure,
		svc.CreateUser,
		connect.WithIdempotency(connect.IdempotencyIdempotent),
		connect.WithHandlerOptions(opts...),
	)
	userServiceGetUserHandler := connect.NewUnaryHandler(
		UserServiceGetUserProcedure,
		svc.GetUser,
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	userServiceGetUserByUsernameHandler := connect.NewUnaryHandler(
		UserServiceGetUserByUsernameProcedure,
		svc.GetUserByUsername,
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	userServiceListUsersHandler := connect.NewUnaryHandler(
		UserServiceListUsersProcedure,
		svc.ListUsers,
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	userServiceListOrganizationUsersHandler := connect.NewUnaryHandler(
		UserServiceListOrganizationUsersProcedure,
		svc.ListOrganizationUsers,
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	userServiceDeleteUserHandler := connect.NewUnaryHandler(
		UserServiceDeleteUserProcedure,
		svc.DeleteUser,
		connect.WithIdempotency(connect.IdempotencyIdempotent),
		connect.WithHandlerOptions(opts...),
	)
	userServiceDeactivateUserHandler := connect.NewUnaryHandler(
		UserServiceDeactivateUserProcedure,
		svc.DeactivateUser,
		connect.WithIdempotency(connect.IdempotencyIdempotent),
		connect.WithHandlerOptions(opts...),
	)
	userServiceUpdateUserServerRoleHandler := connect.NewUnaryHandler(
		UserServiceUpdateUserServerRoleProcedure,
		svc.UpdateUserServerRole,
		opts...,
	)
	userServiceCountUsersHandler := connect.NewUnaryHandler(
		UserServiceCountUsersProcedure,
		svc.CountUsers,
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	userServiceUpdateUserSettingsHandler := connect.NewUnaryHandler(
		UserServiceUpdateUserSettingsProcedure,
		svc.UpdateUserSettings,
		opts...,
	)
	userServiceGetUserPluginPreferencesHandler := connect.NewUnaryHandler(
		UserServiceGetUserPluginPreferencesProcedure,
		svc.GetUserPluginPreferences,
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	userServiceUpdateUserPluginPreferenceHandler := connect.NewUnaryHandler(
		UserServiceUpdateUserPluginPreferenceProcedure,
		svc.UpdateUserPluginPreference,
		opts...,
	)
	return "/buf.alpha.registry.v1alpha1.UserService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case UserServiceCreateUserProcedure:
			userServiceCreateUserHandler.ServeHTTP(w, r)
		case UserServiceGetUserProcedure:
			userServiceGetUserHandler.ServeHTTP(w, r)
		case UserServiceGetUserByUsernameProcedure:
			userServiceGetUserByUsernameHandler.ServeHTTP(w, r)
		case UserServiceListUsersProcedure:
			userServiceListUsersHandler.ServeHTTP(w, r)
		case UserServiceListOrganizationUsersProcedure:
			userServiceListOrganizationUsersHandler.ServeHTTP(w, r)
		case UserServiceDeleteUserProcedure:
			userServiceDeleteUserHandler.ServeHTTP(w, r)
		case UserServiceDeactivateUserProcedure:
			userServiceDeactivateUserHandler.ServeHTTP(w, r)
		case UserServiceUpdateUserServerRoleProcedure:
			userServiceUpdateUserServerRoleHandler.ServeHTTP(w, r)
		case UserServiceCountUsersProcedure:
			userServiceCountUsersHandler.ServeHTTP(w, r)
		case UserServiceUpdateUserSettingsProcedure:
			userServiceUpdateUserSettingsHandler.ServeHTTP(w, r)
		case UserServiceGetUserPluginPreferencesProcedure:
			userServiceGetUserPluginPreferencesHandler.ServeHTTP(w, r)
		case UserServiceUpdateUserPluginPreferenceProcedure:
			userServiceUpdateUserPluginPreferenceHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedUserServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedUserServiceHandler struct{}

func (UnimplementedUserServiceHandler) CreateUser(context.Context, *connect.Request[v1alpha1.CreateUserRequest]) (*connect.Response[v1alpha1.CreateUserResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.UserService.CreateUser is not implemented"))
}

func (UnimplementedUserServiceHandler) GetUser(context.Context, *connect.Request[v1alpha1.GetUserRequest]) (*connect.Response[v1alpha1.GetUserResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.UserService.GetUser is not implemented"))
}

func (UnimplementedUserServiceHandler) GetUserByUsername(context.Context, *connect.Request[v1alpha1.GetUserByUsernameRequest]) (*connect.Response[v1alpha1.GetUserByUsernameResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.UserService.GetUserByUsername is not implemented"))
}

func (UnimplementedUserServiceHandler) ListUsers(context.Context, *connect.Request[v1alpha1.ListUsersRequest]) (*connect.Response[v1alpha1.ListUsersResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.UserService.ListUsers is not implemented"))
}

func (UnimplementedUserServiceHandler) ListOrganizationUsers(context.Context, *connect.Request[v1alpha1.ListOrganizationUsersRequest]) (*connect.Response[v1alpha1.ListOrganizationUsersResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.UserService.ListOrganizationUsers is not implemented"))
}

func (UnimplementedUserServiceHandler) DeleteUser(context.Context, *connect.Request[v1alpha1.DeleteUserRequest]) (*connect.Response[v1alpha1.DeleteUserResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.UserService.DeleteUser is not implemented"))
}

func (UnimplementedUserServiceHandler) DeactivateUser(context.Context, *connect.Request[v1alpha1.DeactivateUserRequest]) (*connect.Response[v1alpha1.DeactivateUserResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.UserService.DeactivateUser is not implemented"))
}

func (UnimplementedUserServiceHandler) UpdateUserServerRole(context.Context, *connect.Request[v1alpha1.UpdateUserServerRoleRequest]) (*connect.Response[v1alpha1.UpdateUserServerRoleResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.UserService.UpdateUserServerRole is not implemented"))
}

func (UnimplementedUserServiceHandler) CountUsers(context.Context, *connect.Request[v1alpha1.CountUsersRequest]) (*connect.Response[v1alpha1.CountUsersResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.UserService.CountUsers is not implemented"))
}

func (UnimplementedUserServiceHandler) UpdateUserSettings(context.Context, *connect.Request[v1alpha1.UpdateUserSettingsRequest]) (*connect.Response[v1alpha1.UpdateUserSettingsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.UserService.UpdateUserSettings is not implemented"))
}

func (UnimplementedUserServiceHandler) GetUserPluginPreferences(context.Context, *connect.Request[v1alpha1.GetUserPluginPreferencesRequest]) (*connect.Response[v1alpha1.GetUserPluginPreferencesResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.UserService.GetUserPluginPreferences is not implemented"))
}

func (UnimplementedUserServiceHandler) UpdateUserPluginPreference(context.Context, *connect.Request[v1alpha1.UpdateUserPluginPreferenceRequest]) (*connect.Response[v1alpha1.UpdateUserPluginPreferenceResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.UserService.UpdateUserPluginPreference is not implemented"))
}
