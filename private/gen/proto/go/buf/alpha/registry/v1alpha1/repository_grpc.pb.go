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

// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.1.0
// - protoc             v3.18.0
// source: buf/alpha/registry/v1alpha1/repository.proto

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

// RepositoryServiceClient is the client API for RepositoryService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type RepositoryServiceClient interface {
	// GetRepository gets a repository by ID.
	GetRepository(ctx context.Context, in *GetRepositoryRequest, opts ...grpc.CallOption) (*GetRepositoryResponse, error)
	// GetRepositoryByFullName gets a repository by full name.
	GetRepositoryByFullName(ctx context.Context, in *GetRepositoryByFullNameRequest, opts ...grpc.CallOption) (*GetRepositoryByFullNameResponse, error)
	// ListRepositories lists all repositories.
	ListRepositories(ctx context.Context, in *ListRepositoriesRequest, opts ...grpc.CallOption) (*ListRepositoriesResponse, error)
	// ListUserRepositories lists all repositories belonging to a user.
	ListUserRepositories(ctx context.Context, in *ListUserRepositoriesRequest, opts ...grpc.CallOption) (*ListUserRepositoriesResponse, error)
	// ListUserRepositories lists all repositories a user can access.
	ListRepositoriesUserCanAccess(ctx context.Context, in *ListRepositoriesUserCanAccessRequest, opts ...grpc.CallOption) (*ListRepositoriesUserCanAccessResponse, error)
	// ListOrganizationRepositories lists all repositories for an organization.
	ListOrganizationRepositories(ctx context.Context, in *ListOrganizationRepositoriesRequest, opts ...grpc.CallOption) (*ListOrganizationRepositoriesResponse, error)
	// CreateRepositoryByFullName creates a new repository by full name.
	CreateRepositoryByFullName(ctx context.Context, in *CreateRepositoryByFullNameRequest, opts ...grpc.CallOption) (*CreateRepositoryByFullNameResponse, error)
	// UpdateRepositoryName updates a repository's name.
	UpdateRepositoryName(ctx context.Context, in *UpdateRepositoryNameRequest, opts ...grpc.CallOption) (*UpdateRepositoryNameResponse, error)
	// UpdateRepositoryNameByFullName updates a repository's name by full name.
	UpdateRepositoryNameByFullName(ctx context.Context, in *UpdateRepositoryNameByFullNameRequest, opts ...grpc.CallOption) (*UpdateRepositoryNameByFullNameResponse, error)
	// UpdateRepositoryVisibility updates a repository's visibility.
	UpdateRepositoryVisibility(ctx context.Context, in *UpdateRepositoryVisibilityRequest, opts ...grpc.CallOption) (*UpdateRepositoryVisibilityResponse, error)
	// UpdateRepositoryVisibilityByName updates a repository's visibility by name.
	UpdateRepositoryVisibilityByName(ctx context.Context, in *UpdateRepositoryVisibilityByNameRequest, opts ...grpc.CallOption) (*UpdateRepositoryVisibilityByNameResponse, error)
	// DeleteRepository deletes a repository.
	DeleteRepository(ctx context.Context, in *DeleteRepositoryRequest, opts ...grpc.CallOption) (*DeleteRepositoryResponse, error)
	// DeleteRepositoryByFullName deletes a repository by full name.
	DeleteRepositoryByFullName(ctx context.Context, in *DeleteRepositoryByFullNameRequest, opts ...grpc.CallOption) (*DeleteRepositoryByFullNameResponse, error)
	// DeprecateRepository deprecates the repository
	DeprecateRepository(ctx context.Context, in *DeprecateRepositoryRequest, opts ...grpc.CallOption) (*DeprecateRepositoryResponse, error)
	// UndeprecateRepository makes the repository not deprecated and removes any deprecation_message
	UndeprecateRepository(ctx context.Context, in *UndeprecateRepositoryRequest, opts ...grpc.CallOption) (*UndeprecateRepositoryResponse, error)
}

type repositoryServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewRepositoryServiceClient(cc grpc.ClientConnInterface) RepositoryServiceClient {
	return &repositoryServiceClient{cc}
}

func (c *repositoryServiceClient) GetRepository(ctx context.Context, in *GetRepositoryRequest, opts ...grpc.CallOption) (*GetRepositoryResponse, error) {
	out := new(GetRepositoryResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.RepositoryService/GetRepository", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *repositoryServiceClient) GetRepositoryByFullName(ctx context.Context, in *GetRepositoryByFullNameRequest, opts ...grpc.CallOption) (*GetRepositoryByFullNameResponse, error) {
	out := new(GetRepositoryByFullNameResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.RepositoryService/GetRepositoryByFullName", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *repositoryServiceClient) ListRepositories(ctx context.Context, in *ListRepositoriesRequest, opts ...grpc.CallOption) (*ListRepositoriesResponse, error) {
	out := new(ListRepositoriesResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.RepositoryService/ListRepositories", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *repositoryServiceClient) ListUserRepositories(ctx context.Context, in *ListUserRepositoriesRequest, opts ...grpc.CallOption) (*ListUserRepositoriesResponse, error) {
	out := new(ListUserRepositoriesResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.RepositoryService/ListUserRepositories", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *repositoryServiceClient) ListRepositoriesUserCanAccess(ctx context.Context, in *ListRepositoriesUserCanAccessRequest, opts ...grpc.CallOption) (*ListRepositoriesUserCanAccessResponse, error) {
	out := new(ListRepositoriesUserCanAccessResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.RepositoryService/ListRepositoriesUserCanAccess", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *repositoryServiceClient) ListOrganizationRepositories(ctx context.Context, in *ListOrganizationRepositoriesRequest, opts ...grpc.CallOption) (*ListOrganizationRepositoriesResponse, error) {
	out := new(ListOrganizationRepositoriesResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.RepositoryService/ListOrganizationRepositories", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *repositoryServiceClient) CreateRepositoryByFullName(ctx context.Context, in *CreateRepositoryByFullNameRequest, opts ...grpc.CallOption) (*CreateRepositoryByFullNameResponse, error) {
	out := new(CreateRepositoryByFullNameResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.RepositoryService/CreateRepositoryByFullName", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *repositoryServiceClient) UpdateRepositoryName(ctx context.Context, in *UpdateRepositoryNameRequest, opts ...grpc.CallOption) (*UpdateRepositoryNameResponse, error) {
	out := new(UpdateRepositoryNameResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.RepositoryService/UpdateRepositoryName", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *repositoryServiceClient) UpdateRepositoryNameByFullName(ctx context.Context, in *UpdateRepositoryNameByFullNameRequest, opts ...grpc.CallOption) (*UpdateRepositoryNameByFullNameResponse, error) {
	out := new(UpdateRepositoryNameByFullNameResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.RepositoryService/UpdateRepositoryNameByFullName", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *repositoryServiceClient) UpdateRepositoryVisibility(ctx context.Context, in *UpdateRepositoryVisibilityRequest, opts ...grpc.CallOption) (*UpdateRepositoryVisibilityResponse, error) {
	out := new(UpdateRepositoryVisibilityResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.RepositoryService/UpdateRepositoryVisibility", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *repositoryServiceClient) UpdateRepositoryVisibilityByName(ctx context.Context, in *UpdateRepositoryVisibilityByNameRequest, opts ...grpc.CallOption) (*UpdateRepositoryVisibilityByNameResponse, error) {
	out := new(UpdateRepositoryVisibilityByNameResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.RepositoryService/UpdateRepositoryVisibilityByName", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *repositoryServiceClient) DeleteRepository(ctx context.Context, in *DeleteRepositoryRequest, opts ...grpc.CallOption) (*DeleteRepositoryResponse, error) {
	out := new(DeleteRepositoryResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.RepositoryService/DeleteRepository", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *repositoryServiceClient) DeleteRepositoryByFullName(ctx context.Context, in *DeleteRepositoryByFullNameRequest, opts ...grpc.CallOption) (*DeleteRepositoryByFullNameResponse, error) {
	out := new(DeleteRepositoryByFullNameResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.RepositoryService/DeleteRepositoryByFullName", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *repositoryServiceClient) DeprecateRepository(ctx context.Context, in *DeprecateRepositoryRequest, opts ...grpc.CallOption) (*DeprecateRepositoryResponse, error) {
	out := new(DeprecateRepositoryResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.RepositoryService/DeprecateRepository", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *repositoryServiceClient) UndeprecateRepository(ctx context.Context, in *UndeprecateRepositoryRequest, opts ...grpc.CallOption) (*UndeprecateRepositoryResponse, error) {
	out := new(UndeprecateRepositoryResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.RepositoryService/UndeprecateRepository", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RepositoryServiceServer is the server API for RepositoryService service.
// All implementations should embed UnimplementedRepositoryServiceServer
// for forward compatibility
type RepositoryServiceServer interface {
	// GetRepository gets a repository by ID.
	GetRepository(context.Context, *GetRepositoryRequest) (*GetRepositoryResponse, error)
	// GetRepositoryByFullName gets a repository by full name.
	GetRepositoryByFullName(context.Context, *GetRepositoryByFullNameRequest) (*GetRepositoryByFullNameResponse, error)
	// ListRepositories lists all repositories.
	ListRepositories(context.Context, *ListRepositoriesRequest) (*ListRepositoriesResponse, error)
	// ListUserRepositories lists all repositories belonging to a user.
	ListUserRepositories(context.Context, *ListUserRepositoriesRequest) (*ListUserRepositoriesResponse, error)
	// ListUserRepositories lists all repositories a user can access.
	ListRepositoriesUserCanAccess(context.Context, *ListRepositoriesUserCanAccessRequest) (*ListRepositoriesUserCanAccessResponse, error)
	// ListOrganizationRepositories lists all repositories for an organization.
	ListOrganizationRepositories(context.Context, *ListOrganizationRepositoriesRequest) (*ListOrganizationRepositoriesResponse, error)
	// CreateRepositoryByFullName creates a new repository by full name.
	CreateRepositoryByFullName(context.Context, *CreateRepositoryByFullNameRequest) (*CreateRepositoryByFullNameResponse, error)
	// UpdateRepositoryName updates a repository's name.
	UpdateRepositoryName(context.Context, *UpdateRepositoryNameRequest) (*UpdateRepositoryNameResponse, error)
	// UpdateRepositoryNameByFullName updates a repository's name by full name.
	UpdateRepositoryNameByFullName(context.Context, *UpdateRepositoryNameByFullNameRequest) (*UpdateRepositoryNameByFullNameResponse, error)
	// UpdateRepositoryVisibility updates a repository's visibility.
	UpdateRepositoryVisibility(context.Context, *UpdateRepositoryVisibilityRequest) (*UpdateRepositoryVisibilityResponse, error)
	// UpdateRepositoryVisibilityByName updates a repository's visibility by name.
	UpdateRepositoryVisibilityByName(context.Context, *UpdateRepositoryVisibilityByNameRequest) (*UpdateRepositoryVisibilityByNameResponse, error)
	// DeleteRepository deletes a repository.
	DeleteRepository(context.Context, *DeleteRepositoryRequest) (*DeleteRepositoryResponse, error)
	// DeleteRepositoryByFullName deletes a repository by full name.
	DeleteRepositoryByFullName(context.Context, *DeleteRepositoryByFullNameRequest) (*DeleteRepositoryByFullNameResponse, error)
	// DeprecateRepository deprecates the repository
	DeprecateRepository(context.Context, *DeprecateRepositoryRequest) (*DeprecateRepositoryResponse, error)
	// UndeprecateRepository makes the repository not deprecated and removes any deprecation_message
	UndeprecateRepository(context.Context, *UndeprecateRepositoryRequest) (*UndeprecateRepositoryResponse, error)
}

// UnimplementedRepositoryServiceServer should be embedded to have forward compatible implementations.
type UnimplementedRepositoryServiceServer struct {
}

func (UnimplementedRepositoryServiceServer) GetRepository(context.Context, *GetRepositoryRequest) (*GetRepositoryResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetRepository not implemented")
}
func (UnimplementedRepositoryServiceServer) GetRepositoryByFullName(context.Context, *GetRepositoryByFullNameRequest) (*GetRepositoryByFullNameResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetRepositoryByFullName not implemented")
}
func (UnimplementedRepositoryServiceServer) ListRepositories(context.Context, *ListRepositoriesRequest) (*ListRepositoriesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListRepositories not implemented")
}
func (UnimplementedRepositoryServiceServer) ListUserRepositories(context.Context, *ListUserRepositoriesRequest) (*ListUserRepositoriesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListUserRepositories not implemented")
}
func (UnimplementedRepositoryServiceServer) ListRepositoriesUserCanAccess(context.Context, *ListRepositoriesUserCanAccessRequest) (*ListRepositoriesUserCanAccessResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListRepositoriesUserCanAccess not implemented")
}
func (UnimplementedRepositoryServiceServer) ListOrganizationRepositories(context.Context, *ListOrganizationRepositoriesRequest) (*ListOrganizationRepositoriesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListOrganizationRepositories not implemented")
}
func (UnimplementedRepositoryServiceServer) CreateRepositoryByFullName(context.Context, *CreateRepositoryByFullNameRequest) (*CreateRepositoryByFullNameResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateRepositoryByFullName not implemented")
}
func (UnimplementedRepositoryServiceServer) UpdateRepositoryName(context.Context, *UpdateRepositoryNameRequest) (*UpdateRepositoryNameResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateRepositoryName not implemented")
}
func (UnimplementedRepositoryServiceServer) UpdateRepositoryNameByFullName(context.Context, *UpdateRepositoryNameByFullNameRequest) (*UpdateRepositoryNameByFullNameResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateRepositoryNameByFullName not implemented")
}
func (UnimplementedRepositoryServiceServer) UpdateRepositoryVisibility(context.Context, *UpdateRepositoryVisibilityRequest) (*UpdateRepositoryVisibilityResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateRepositoryVisibility not implemented")
}
func (UnimplementedRepositoryServiceServer) UpdateRepositoryVisibilityByName(context.Context, *UpdateRepositoryVisibilityByNameRequest) (*UpdateRepositoryVisibilityByNameResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateRepositoryVisibilityByName not implemented")
}
func (UnimplementedRepositoryServiceServer) DeleteRepository(context.Context, *DeleteRepositoryRequest) (*DeleteRepositoryResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteRepository not implemented")
}
func (UnimplementedRepositoryServiceServer) DeleteRepositoryByFullName(context.Context, *DeleteRepositoryByFullNameRequest) (*DeleteRepositoryByFullNameResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteRepositoryByFullName not implemented")
}
func (UnimplementedRepositoryServiceServer) DeprecateRepository(context.Context, *DeprecateRepositoryRequest) (*DeprecateRepositoryResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeprecateRepository not implemented")
}
func (UnimplementedRepositoryServiceServer) UndeprecateRepository(context.Context, *UndeprecateRepositoryRequest) (*UndeprecateRepositoryResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UndeprecateRepository not implemented")
}

// UnsafeRepositoryServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to RepositoryServiceServer will
// result in compilation errors.
type UnsafeRepositoryServiceServer interface {
	mustEmbedUnimplementedRepositoryServiceServer()
}

func RegisterRepositoryServiceServer(s grpc.ServiceRegistrar, srv RepositoryServiceServer) {
	s.RegisterService(&RepositoryService_ServiceDesc, srv)
}

func _RepositoryService_GetRepository_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetRepositoryRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RepositoryServiceServer).GetRepository(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.RepositoryService/GetRepository",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RepositoryServiceServer).GetRepository(ctx, req.(*GetRepositoryRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RepositoryService_GetRepositoryByFullName_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetRepositoryByFullNameRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RepositoryServiceServer).GetRepositoryByFullName(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.RepositoryService/GetRepositoryByFullName",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RepositoryServiceServer).GetRepositoryByFullName(ctx, req.(*GetRepositoryByFullNameRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RepositoryService_ListRepositories_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListRepositoriesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RepositoryServiceServer).ListRepositories(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.RepositoryService/ListRepositories",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RepositoryServiceServer).ListRepositories(ctx, req.(*ListRepositoriesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RepositoryService_ListUserRepositories_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListUserRepositoriesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RepositoryServiceServer).ListUserRepositories(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.RepositoryService/ListUserRepositories",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RepositoryServiceServer).ListUserRepositories(ctx, req.(*ListUserRepositoriesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RepositoryService_ListRepositoriesUserCanAccess_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListRepositoriesUserCanAccessRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RepositoryServiceServer).ListRepositoriesUserCanAccess(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.RepositoryService/ListRepositoriesUserCanAccess",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RepositoryServiceServer).ListRepositoriesUserCanAccess(ctx, req.(*ListRepositoriesUserCanAccessRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RepositoryService_ListOrganizationRepositories_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListOrganizationRepositoriesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RepositoryServiceServer).ListOrganizationRepositories(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.RepositoryService/ListOrganizationRepositories",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RepositoryServiceServer).ListOrganizationRepositories(ctx, req.(*ListOrganizationRepositoriesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RepositoryService_CreateRepositoryByFullName_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateRepositoryByFullNameRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RepositoryServiceServer).CreateRepositoryByFullName(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.RepositoryService/CreateRepositoryByFullName",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RepositoryServiceServer).CreateRepositoryByFullName(ctx, req.(*CreateRepositoryByFullNameRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RepositoryService_UpdateRepositoryName_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateRepositoryNameRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RepositoryServiceServer).UpdateRepositoryName(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.RepositoryService/UpdateRepositoryName",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RepositoryServiceServer).UpdateRepositoryName(ctx, req.(*UpdateRepositoryNameRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RepositoryService_UpdateRepositoryNameByFullName_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateRepositoryNameByFullNameRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RepositoryServiceServer).UpdateRepositoryNameByFullName(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.RepositoryService/UpdateRepositoryNameByFullName",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RepositoryServiceServer).UpdateRepositoryNameByFullName(ctx, req.(*UpdateRepositoryNameByFullNameRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RepositoryService_UpdateRepositoryVisibility_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateRepositoryVisibilityRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RepositoryServiceServer).UpdateRepositoryVisibility(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.RepositoryService/UpdateRepositoryVisibility",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RepositoryServiceServer).UpdateRepositoryVisibility(ctx, req.(*UpdateRepositoryVisibilityRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RepositoryService_UpdateRepositoryVisibilityByName_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateRepositoryVisibilityByNameRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RepositoryServiceServer).UpdateRepositoryVisibilityByName(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.RepositoryService/UpdateRepositoryVisibilityByName",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RepositoryServiceServer).UpdateRepositoryVisibilityByName(ctx, req.(*UpdateRepositoryVisibilityByNameRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RepositoryService_DeleteRepository_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteRepositoryRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RepositoryServiceServer).DeleteRepository(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.RepositoryService/DeleteRepository",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RepositoryServiceServer).DeleteRepository(ctx, req.(*DeleteRepositoryRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RepositoryService_DeleteRepositoryByFullName_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteRepositoryByFullNameRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RepositoryServiceServer).DeleteRepositoryByFullName(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.RepositoryService/DeleteRepositoryByFullName",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RepositoryServiceServer).DeleteRepositoryByFullName(ctx, req.(*DeleteRepositoryByFullNameRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RepositoryService_DeprecateRepository_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeprecateRepositoryRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RepositoryServiceServer).DeprecateRepository(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.RepositoryService/DeprecateRepository",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RepositoryServiceServer).DeprecateRepository(ctx, req.(*DeprecateRepositoryRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RepositoryService_UndeprecateRepository_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UndeprecateRepositoryRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RepositoryServiceServer).UndeprecateRepository(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.RepositoryService/UndeprecateRepository",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RepositoryServiceServer).UndeprecateRepository(ctx, req.(*UndeprecateRepositoryRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// RepositoryService_ServiceDesc is the grpc.ServiceDesc for RepositoryService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var RepositoryService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "buf.alpha.registry.v1alpha1.RepositoryService",
	HandlerType: (*RepositoryServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetRepository",
			Handler:    _RepositoryService_GetRepository_Handler,
		},
		{
			MethodName: "GetRepositoryByFullName",
			Handler:    _RepositoryService_GetRepositoryByFullName_Handler,
		},
		{
			MethodName: "ListRepositories",
			Handler:    _RepositoryService_ListRepositories_Handler,
		},
		{
			MethodName: "ListUserRepositories",
			Handler:    _RepositoryService_ListUserRepositories_Handler,
		},
		{
			MethodName: "ListRepositoriesUserCanAccess",
			Handler:    _RepositoryService_ListRepositoriesUserCanAccess_Handler,
		},
		{
			MethodName: "ListOrganizationRepositories",
			Handler:    _RepositoryService_ListOrganizationRepositories_Handler,
		},
		{
			MethodName: "CreateRepositoryByFullName",
			Handler:    _RepositoryService_CreateRepositoryByFullName_Handler,
		},
		{
			MethodName: "UpdateRepositoryName",
			Handler:    _RepositoryService_UpdateRepositoryName_Handler,
		},
		{
			MethodName: "UpdateRepositoryNameByFullName",
			Handler:    _RepositoryService_UpdateRepositoryNameByFullName_Handler,
		},
		{
			MethodName: "UpdateRepositoryVisibility",
			Handler:    _RepositoryService_UpdateRepositoryVisibility_Handler,
		},
		{
			MethodName: "UpdateRepositoryVisibilityByName",
			Handler:    _RepositoryService_UpdateRepositoryVisibilityByName_Handler,
		},
		{
			MethodName: "DeleteRepository",
			Handler:    _RepositoryService_DeleteRepository_Handler,
		},
		{
			MethodName: "DeleteRepositoryByFullName",
			Handler:    _RepositoryService_DeleteRepositoryByFullName_Handler,
		},
		{
			MethodName: "DeprecateRepository",
			Handler:    _RepositoryService_DeprecateRepository_Handler,
		},
		{
			MethodName: "UndeprecateRepository",
			Handler:    _RepositoryService_UndeprecateRepository_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "buf/alpha/registry/v1alpha1/repository.proto",
}
