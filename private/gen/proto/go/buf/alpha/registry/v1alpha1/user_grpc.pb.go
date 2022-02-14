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

// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             (unknown)
// source: buf/alpha/registry/v1alpha1/user.proto

package registryv1alpha1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// UserServiceClient is the client API for UserService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type UserServiceClient interface {
	// CreateUser creates a new user with the given username.
	CreateUser(ctx context.Context, in *CreateUserRequest, opts ...grpc.CallOption) (*CreateUserResponse, error)
	// GetUser gets a user by ID.
	GetUser(ctx context.Context, in *GetUserRequest, opts ...grpc.CallOption) (*GetUserResponse, error)
	// GetUserByUsername gets a user by username.
	GetUserByUsername(ctx context.Context, in *GetUserByUsernameRequest, opts ...grpc.CallOption) (*GetUserByUsernameResponse, error)
	// ListUsers lists all users.
	ListUsers(ctx context.Context, in *ListUsersRequest, opts ...grpc.CallOption) (*ListUsersResponse, error)
	// ListOrganizationUsers lists all users for an organization.
	// TODO: #663 move this to organization service
	ListOrganizationUsers(ctx context.Context, in *ListOrganizationUsersRequest, opts ...grpc.CallOption) (*ListOrganizationUsersResponse, error)
	// DeleteUser deletes a user.
	DeleteUser(ctx context.Context, in *DeleteUserRequest, opts ...grpc.CallOption) (*DeleteUserResponse, error)
	// Deactivate user deactivates a user.
	DeactivateUser(ctx context.Context, in *DeactivateUserRequest, opts ...grpc.CallOption) (*DeactivateUserResponse, error)
	// UpdateUserServerRole update the role of an user in the server.
	UpdateUserServerRole(ctx context.Context, in *UpdateUserServerRoleRequest, opts ...grpc.CallOption) (*UpdateUserServerRoleResponse, error)
	// CountUsers returns the number of users in the server by the user state provided.
	CountUsers(ctx context.Context, in *CountUsersRequest, opts ...grpc.CallOption) (*CountUsersResponse, error)
}

type userServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewUserServiceClient(cc grpc.ClientConnInterface) UserServiceClient {
	return &userServiceClient{cc}
}

func (c *userServiceClient) CreateUser(ctx context.Context, in *CreateUserRequest, opts ...grpc.CallOption) (*CreateUserResponse, error) {
	out := new(CreateUserResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.UserService/CreateUser", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userServiceClient) GetUser(ctx context.Context, in *GetUserRequest, opts ...grpc.CallOption) (*GetUserResponse, error) {
	out := new(GetUserResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.UserService/GetUser", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userServiceClient) GetUserByUsername(ctx context.Context, in *GetUserByUsernameRequest, opts ...grpc.CallOption) (*GetUserByUsernameResponse, error) {
	out := new(GetUserByUsernameResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.UserService/GetUserByUsername", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userServiceClient) ListUsers(ctx context.Context, in *ListUsersRequest, opts ...grpc.CallOption) (*ListUsersResponse, error) {
	out := new(ListUsersResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.UserService/ListUsers", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userServiceClient) ListOrganizationUsers(ctx context.Context, in *ListOrganizationUsersRequest, opts ...grpc.CallOption) (*ListOrganizationUsersResponse, error) {
	out := new(ListOrganizationUsersResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.UserService/ListOrganizationUsers", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userServiceClient) DeleteUser(ctx context.Context, in *DeleteUserRequest, opts ...grpc.CallOption) (*DeleteUserResponse, error) {
	out := new(DeleteUserResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.UserService/DeleteUser", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userServiceClient) DeactivateUser(ctx context.Context, in *DeactivateUserRequest, opts ...grpc.CallOption) (*DeactivateUserResponse, error) {
	out := new(DeactivateUserResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.UserService/DeactivateUser", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userServiceClient) UpdateUserServerRole(ctx context.Context, in *UpdateUserServerRoleRequest, opts ...grpc.CallOption) (*UpdateUserServerRoleResponse, error) {
	out := new(UpdateUserServerRoleResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.UserService/UpdateUserServerRole", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userServiceClient) CountUsers(ctx context.Context, in *CountUsersRequest, opts ...grpc.CallOption) (*CountUsersResponse, error) {
	out := new(CountUsersResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.UserService/CountUsers", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// UserServiceServer is the server API for UserService service.
// All implementations should embed UnimplementedUserServiceServer
// for forward compatibility
type UserServiceServer interface {
	// CreateUser creates a new user with the given username.
	CreateUser(context.Context, *CreateUserRequest) (*CreateUserResponse, error)
	// GetUser gets a user by ID.
	GetUser(context.Context, *GetUserRequest) (*GetUserResponse, error)
	// GetUserByUsername gets a user by username.
	GetUserByUsername(context.Context, *GetUserByUsernameRequest) (*GetUserByUsernameResponse, error)
	// ListUsers lists all users.
	ListUsers(context.Context, *ListUsersRequest) (*ListUsersResponse, error)
	// ListOrganizationUsers lists all users for an organization.
	// TODO: #663 move this to organization service
	ListOrganizationUsers(context.Context, *ListOrganizationUsersRequest) (*ListOrganizationUsersResponse, error)
	// DeleteUser deletes a user.
	DeleteUser(context.Context, *DeleteUserRequest) (*DeleteUserResponse, error)
	// Deactivate user deactivates a user.
	DeactivateUser(context.Context, *DeactivateUserRequest) (*DeactivateUserResponse, error)
	// UpdateUserServerRole update the role of an user in the server.
	UpdateUserServerRole(context.Context, *UpdateUserServerRoleRequest) (*UpdateUserServerRoleResponse, error)
	// CountUsers returns the number of users in the server by the user state provided.
	CountUsers(context.Context, *CountUsersRequest) (*CountUsersResponse, error)
}

// UnimplementedUserServiceServer should be embedded to have forward compatible implementations.
type UnimplementedUserServiceServer struct {
}

func (UnimplementedUserServiceServer) CreateUser(context.Context, *CreateUserRequest) (*CreateUserResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateUser not implemented")
}
func (UnimplementedUserServiceServer) GetUser(context.Context, *GetUserRequest) (*GetUserResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetUser not implemented")
}
func (UnimplementedUserServiceServer) GetUserByUsername(context.Context, *GetUserByUsernameRequest) (*GetUserByUsernameResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetUserByUsername not implemented")
}
func (UnimplementedUserServiceServer) ListUsers(context.Context, *ListUsersRequest) (*ListUsersResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListUsers not implemented")
}
func (UnimplementedUserServiceServer) ListOrganizationUsers(context.Context, *ListOrganizationUsersRequest) (*ListOrganizationUsersResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListOrganizationUsers not implemented")
}
func (UnimplementedUserServiceServer) DeleteUser(context.Context, *DeleteUserRequest) (*DeleteUserResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteUser not implemented")
}
func (UnimplementedUserServiceServer) DeactivateUser(context.Context, *DeactivateUserRequest) (*DeactivateUserResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeactivateUser not implemented")
}
func (UnimplementedUserServiceServer) UpdateUserServerRole(context.Context, *UpdateUserServerRoleRequest) (*UpdateUserServerRoleResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateUserServerRole not implemented")
}
func (UnimplementedUserServiceServer) CountUsers(context.Context, *CountUsersRequest) (*CountUsersResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CountUsers not implemented")
}

// UnsafeUserServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to UserServiceServer will
// result in compilation errors.
type UnsafeUserServiceServer interface {
	mustEmbedUnimplementedUserServiceServer()
}

func RegisterUserServiceServer(s grpc.ServiceRegistrar, srv UserServiceServer) {
	s.RegisterService(&UserService_ServiceDesc, srv)
}

func _UserService_CreateUser_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateUserRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServiceServer).CreateUser(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.UserService/CreateUser",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServiceServer).CreateUser(ctx, req.(*CreateUserRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _UserService_GetUser_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetUserRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServiceServer).GetUser(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.UserService/GetUser",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServiceServer).GetUser(ctx, req.(*GetUserRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _UserService_GetUserByUsername_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetUserByUsernameRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServiceServer).GetUserByUsername(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.UserService/GetUserByUsername",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServiceServer).GetUserByUsername(ctx, req.(*GetUserByUsernameRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _UserService_ListUsers_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListUsersRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServiceServer).ListUsers(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.UserService/ListUsers",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServiceServer).ListUsers(ctx, req.(*ListUsersRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _UserService_ListOrganizationUsers_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListOrganizationUsersRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServiceServer).ListOrganizationUsers(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.UserService/ListOrganizationUsers",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServiceServer).ListOrganizationUsers(ctx, req.(*ListOrganizationUsersRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _UserService_DeleteUser_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteUserRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServiceServer).DeleteUser(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.UserService/DeleteUser",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServiceServer).DeleteUser(ctx, req.(*DeleteUserRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _UserService_DeactivateUser_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeactivateUserRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServiceServer).DeactivateUser(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.UserService/DeactivateUser",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServiceServer).DeactivateUser(ctx, req.(*DeactivateUserRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _UserService_UpdateUserServerRole_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateUserServerRoleRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServiceServer).UpdateUserServerRole(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.UserService/UpdateUserServerRole",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServiceServer).UpdateUserServerRole(ctx, req.(*UpdateUserServerRoleRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _UserService_CountUsers_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CountUsersRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServiceServer).CountUsers(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.UserService/CountUsers",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServiceServer).CountUsers(ctx, req.(*CountUsersRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// UserService_ServiceDesc is the grpc.ServiceDesc for UserService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var UserService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "buf.alpha.registry.v1alpha1.UserService",
	HandlerType: (*UserServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CreateUser",
			Handler:    _UserService_CreateUser_Handler,
		},
		{
			MethodName: "GetUser",
			Handler:    _UserService_GetUser_Handler,
		},
		{
			MethodName: "GetUserByUsername",
			Handler:    _UserService_GetUserByUsername_Handler,
		},
		{
			MethodName: "ListUsers",
			Handler:    _UserService_ListUsers_Handler,
		},
		{
			MethodName: "ListOrganizationUsers",
			Handler:    _UserService_ListOrganizationUsers_Handler,
		},
		{
			MethodName: "DeleteUser",
			Handler:    _UserService_DeleteUser_Handler,
		},
		{
			MethodName: "DeactivateUser",
			Handler:    _UserService_DeactivateUser_Handler,
		},
		{
			MethodName: "UpdateUserServerRole",
			Handler:    _UserService_UpdateUserServerRole_Handler,
		},
		{
			MethodName: "CountUsers",
			Handler:    _UserService_CountUsers_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "buf/alpha/registry/v1alpha1/user.proto",
}
