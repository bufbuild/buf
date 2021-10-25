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
// - protoc             v3.19.0
// source: buf/alpha/registry/v1alpha1/organization.proto

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

// OrganizationServiceClient is the client API for OrganizationService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type OrganizationServiceClient interface {
	// GetOrganization gets a organization by ID.
	GetOrganization(ctx context.Context, in *GetOrganizationRequest, opts ...grpc.CallOption) (*GetOrganizationResponse, error)
	// GetOrganizationByName gets a organization by name.
	GetOrganizationByName(ctx context.Context, in *GetOrganizationByNameRequest, opts ...grpc.CallOption) (*GetOrganizationByNameResponse, error)
	// ListOrganizations lists all organizations.
	ListOrganizations(ctx context.Context, in *ListOrganizationsRequest, opts ...grpc.CallOption) (*ListOrganizationsResponse, error)
	// ListUserOrganizations lists all organizations a user is member of.
	ListUserOrganizations(ctx context.Context, in *ListUserOrganizationsRequest, opts ...grpc.CallOption) (*ListUserOrganizationsResponse, error)
	// CreateOrganization creates a new organization.
	CreateOrganization(ctx context.Context, in *CreateOrganizationRequest, opts ...grpc.CallOption) (*CreateOrganizationResponse, error)
	// UpdateOrganizationName updates a organization's name.
	UpdateOrganizationName(ctx context.Context, in *UpdateOrganizationNameRequest, opts ...grpc.CallOption) (*UpdateOrganizationNameResponse, error)
	// UpdateOrganizationNameByName updates a organization's name by name.
	UpdateOrganizationNameByName(ctx context.Context, in *UpdateOrganizationNameByNameRequest, opts ...grpc.CallOption) (*UpdateOrganizationNameByNameResponse, error)
	// DeleteOrganization deletes a organization.
	DeleteOrganization(ctx context.Context, in *DeleteOrganizationRequest, opts ...grpc.CallOption) (*DeleteOrganizationResponse, error)
	// DeleteOrganizationByName deletes a organization by name.
	DeleteOrganizationByName(ctx context.Context, in *DeleteOrganizationByNameRequest, opts ...grpc.CallOption) (*DeleteOrganizationByNameResponse, error)
	// AddOrganizationBaseRepositoryScope adds a base repository scope to an organization by ID.
	AddOrganizationBaseRepositoryScope(ctx context.Context, in *AddOrganizationBaseRepositoryScopeRequest, opts ...grpc.CallOption) (*AddOrganizationBaseRepositoryScopeResponse, error)
	// AddOrganizationMember add a role to an user in the organization.
	AddOrganizationMember(ctx context.Context, in *AddOrganizationMemberRequest, opts ...grpc.CallOption) (*AddOrganizationMemberResponse, error)
	// UpdateOrganizationMember update the user's membership information in the organization.
	UpdateOrganizationMember(ctx context.Context, in *UpdateOrganizationMemberRequest, opts ...grpc.CallOption) (*UpdateOrganizationMemberResponse, error)
	// RemoveOrganizationMember remove the role of an user in the organization.
	RemoveOrganizationMember(ctx context.Context, in *RemoveOrganizationMemberRequest, opts ...grpc.CallOption) (*RemoveOrganizationMemberResponse, error)
	// LeaveOrganization remove the role of the current user in the organization.
	LeaveOrganization(ctx context.Context, in *LeaveOrganizationRequest, opts ...grpc.CallOption) (*LeaveOrganizationResponse, error)
}

type organizationServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewOrganizationServiceClient(cc grpc.ClientConnInterface) OrganizationServiceClient {
	return &organizationServiceClient{cc}
}

func (c *organizationServiceClient) GetOrganization(ctx context.Context, in *GetOrganizationRequest, opts ...grpc.CallOption) (*GetOrganizationResponse, error) {
	out := new(GetOrganizationResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.OrganizationService/GetOrganization", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *organizationServiceClient) GetOrganizationByName(ctx context.Context, in *GetOrganizationByNameRequest, opts ...grpc.CallOption) (*GetOrganizationByNameResponse, error) {
	out := new(GetOrganizationByNameResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.OrganizationService/GetOrganizationByName", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *organizationServiceClient) ListOrganizations(ctx context.Context, in *ListOrganizationsRequest, opts ...grpc.CallOption) (*ListOrganizationsResponse, error) {
	out := new(ListOrganizationsResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.OrganizationService/ListOrganizations", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *organizationServiceClient) ListUserOrganizations(ctx context.Context, in *ListUserOrganizationsRequest, opts ...grpc.CallOption) (*ListUserOrganizationsResponse, error) {
	out := new(ListUserOrganizationsResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.OrganizationService/ListUserOrganizations", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *organizationServiceClient) CreateOrganization(ctx context.Context, in *CreateOrganizationRequest, opts ...grpc.CallOption) (*CreateOrganizationResponse, error) {
	out := new(CreateOrganizationResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.OrganizationService/CreateOrganization", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *organizationServiceClient) UpdateOrganizationName(ctx context.Context, in *UpdateOrganizationNameRequest, opts ...grpc.CallOption) (*UpdateOrganizationNameResponse, error) {
	out := new(UpdateOrganizationNameResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.OrganizationService/UpdateOrganizationName", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *organizationServiceClient) UpdateOrganizationNameByName(ctx context.Context, in *UpdateOrganizationNameByNameRequest, opts ...grpc.CallOption) (*UpdateOrganizationNameByNameResponse, error) {
	out := new(UpdateOrganizationNameByNameResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.OrganizationService/UpdateOrganizationNameByName", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *organizationServiceClient) DeleteOrganization(ctx context.Context, in *DeleteOrganizationRequest, opts ...grpc.CallOption) (*DeleteOrganizationResponse, error) {
	out := new(DeleteOrganizationResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.OrganizationService/DeleteOrganization", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *organizationServiceClient) DeleteOrganizationByName(ctx context.Context, in *DeleteOrganizationByNameRequest, opts ...grpc.CallOption) (*DeleteOrganizationByNameResponse, error) {
	out := new(DeleteOrganizationByNameResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.OrganizationService/DeleteOrganizationByName", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *organizationServiceClient) AddOrganizationBaseRepositoryScope(ctx context.Context, in *AddOrganizationBaseRepositoryScopeRequest, opts ...grpc.CallOption) (*AddOrganizationBaseRepositoryScopeResponse, error) {
	out := new(AddOrganizationBaseRepositoryScopeResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.OrganizationService/AddOrganizationBaseRepositoryScope", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *organizationServiceClient) AddOrganizationMember(ctx context.Context, in *AddOrganizationMemberRequest, opts ...grpc.CallOption) (*AddOrganizationMemberResponse, error) {
	out := new(AddOrganizationMemberResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.OrganizationService/AddOrganizationMember", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *organizationServiceClient) UpdateOrganizationMember(ctx context.Context, in *UpdateOrganizationMemberRequest, opts ...grpc.CallOption) (*UpdateOrganizationMemberResponse, error) {
	out := new(UpdateOrganizationMemberResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.OrganizationService/UpdateOrganizationMember", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *organizationServiceClient) RemoveOrganizationMember(ctx context.Context, in *RemoveOrganizationMemberRequest, opts ...grpc.CallOption) (*RemoveOrganizationMemberResponse, error) {
	out := new(RemoveOrganizationMemberResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.OrganizationService/RemoveOrganizationMember", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *organizationServiceClient) LeaveOrganization(ctx context.Context, in *LeaveOrganizationRequest, opts ...grpc.CallOption) (*LeaveOrganizationResponse, error) {
	out := new(LeaveOrganizationResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.OrganizationService/LeaveOrganization", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// OrganizationServiceServer is the server API for OrganizationService service.
// All implementations should embed UnimplementedOrganizationServiceServer
// for forward compatibility
type OrganizationServiceServer interface {
	// GetOrganization gets a organization by ID.
	GetOrganization(context.Context, *GetOrganizationRequest) (*GetOrganizationResponse, error)
	// GetOrganizationByName gets a organization by name.
	GetOrganizationByName(context.Context, *GetOrganizationByNameRequest) (*GetOrganizationByNameResponse, error)
	// ListOrganizations lists all organizations.
	ListOrganizations(context.Context, *ListOrganizationsRequest) (*ListOrganizationsResponse, error)
	// ListUserOrganizations lists all organizations a user is member of.
	ListUserOrganizations(context.Context, *ListUserOrganizationsRequest) (*ListUserOrganizationsResponse, error)
	// CreateOrganization creates a new organization.
	CreateOrganization(context.Context, *CreateOrganizationRequest) (*CreateOrganizationResponse, error)
	// UpdateOrganizationName updates a organization's name.
	UpdateOrganizationName(context.Context, *UpdateOrganizationNameRequest) (*UpdateOrganizationNameResponse, error)
	// UpdateOrganizationNameByName updates a organization's name by name.
	UpdateOrganizationNameByName(context.Context, *UpdateOrganizationNameByNameRequest) (*UpdateOrganizationNameByNameResponse, error)
	// DeleteOrganization deletes a organization.
	DeleteOrganization(context.Context, *DeleteOrganizationRequest) (*DeleteOrganizationResponse, error)
	// DeleteOrganizationByName deletes a organization by name.
	DeleteOrganizationByName(context.Context, *DeleteOrganizationByNameRequest) (*DeleteOrganizationByNameResponse, error)
	// AddOrganizationBaseRepositoryScope adds a base repository scope to an organization by ID.
	AddOrganizationBaseRepositoryScope(context.Context, *AddOrganizationBaseRepositoryScopeRequest) (*AddOrganizationBaseRepositoryScopeResponse, error)
	// AddOrganizationMember add a role to an user in the organization.
	AddOrganizationMember(context.Context, *AddOrganizationMemberRequest) (*AddOrganizationMemberResponse, error)
	// UpdateOrganizationMember update the user's membership information in the organization.
	UpdateOrganizationMember(context.Context, *UpdateOrganizationMemberRequest) (*UpdateOrganizationMemberResponse, error)
	// RemoveOrganizationMember remove the role of an user in the organization.
	RemoveOrganizationMember(context.Context, *RemoveOrganizationMemberRequest) (*RemoveOrganizationMemberResponse, error)
	// LeaveOrganization remove the role of the current user in the organization.
	LeaveOrganization(context.Context, *LeaveOrganizationRequest) (*LeaveOrganizationResponse, error)
}

// UnimplementedOrganizationServiceServer should be embedded to have forward compatible implementations.
type UnimplementedOrganizationServiceServer struct {
}

func (UnimplementedOrganizationServiceServer) GetOrganization(context.Context, *GetOrganizationRequest) (*GetOrganizationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetOrganization not implemented")
}
func (UnimplementedOrganizationServiceServer) GetOrganizationByName(context.Context, *GetOrganizationByNameRequest) (*GetOrganizationByNameResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetOrganizationByName not implemented")
}
func (UnimplementedOrganizationServiceServer) ListOrganizations(context.Context, *ListOrganizationsRequest) (*ListOrganizationsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListOrganizations not implemented")
}
func (UnimplementedOrganizationServiceServer) ListUserOrganizations(context.Context, *ListUserOrganizationsRequest) (*ListUserOrganizationsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListUserOrganizations not implemented")
}
func (UnimplementedOrganizationServiceServer) CreateOrganization(context.Context, *CreateOrganizationRequest) (*CreateOrganizationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateOrganization not implemented")
}
func (UnimplementedOrganizationServiceServer) UpdateOrganizationName(context.Context, *UpdateOrganizationNameRequest) (*UpdateOrganizationNameResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateOrganizationName not implemented")
}
func (UnimplementedOrganizationServiceServer) UpdateOrganizationNameByName(context.Context, *UpdateOrganizationNameByNameRequest) (*UpdateOrganizationNameByNameResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateOrganizationNameByName not implemented")
}
func (UnimplementedOrganizationServiceServer) DeleteOrganization(context.Context, *DeleteOrganizationRequest) (*DeleteOrganizationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteOrganization not implemented")
}
func (UnimplementedOrganizationServiceServer) DeleteOrganizationByName(context.Context, *DeleteOrganizationByNameRequest) (*DeleteOrganizationByNameResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteOrganizationByName not implemented")
}
func (UnimplementedOrganizationServiceServer) AddOrganizationBaseRepositoryScope(context.Context, *AddOrganizationBaseRepositoryScopeRequest) (*AddOrganizationBaseRepositoryScopeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddOrganizationBaseRepositoryScope not implemented")
}
func (UnimplementedOrganizationServiceServer) AddOrganizationMember(context.Context, *AddOrganizationMemberRequest) (*AddOrganizationMemberResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddOrganizationMember not implemented")
}
func (UnimplementedOrganizationServiceServer) UpdateOrganizationMember(context.Context, *UpdateOrganizationMemberRequest) (*UpdateOrganizationMemberResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateOrganizationMember not implemented")
}
func (UnimplementedOrganizationServiceServer) RemoveOrganizationMember(context.Context, *RemoveOrganizationMemberRequest) (*RemoveOrganizationMemberResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RemoveOrganizationMember not implemented")
}
func (UnimplementedOrganizationServiceServer) LeaveOrganization(context.Context, *LeaveOrganizationRequest) (*LeaveOrganizationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method LeaveOrganization not implemented")
}

// UnsafeOrganizationServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to OrganizationServiceServer will
// result in compilation errors.
type UnsafeOrganizationServiceServer interface {
	mustEmbedUnimplementedOrganizationServiceServer()
}

func RegisterOrganizationServiceServer(s grpc.ServiceRegistrar, srv OrganizationServiceServer) {
	s.RegisterService(&OrganizationService_ServiceDesc, srv)
}

func _OrganizationService_GetOrganization_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetOrganizationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrganizationServiceServer).GetOrganization(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.OrganizationService/GetOrganization",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrganizationServiceServer).GetOrganization(ctx, req.(*GetOrganizationRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _OrganizationService_GetOrganizationByName_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetOrganizationByNameRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrganizationServiceServer).GetOrganizationByName(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.OrganizationService/GetOrganizationByName",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrganizationServiceServer).GetOrganizationByName(ctx, req.(*GetOrganizationByNameRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _OrganizationService_ListOrganizations_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListOrganizationsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrganizationServiceServer).ListOrganizations(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.OrganizationService/ListOrganizations",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrganizationServiceServer).ListOrganizations(ctx, req.(*ListOrganizationsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _OrganizationService_ListUserOrganizations_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListUserOrganizationsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrganizationServiceServer).ListUserOrganizations(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.OrganizationService/ListUserOrganizations",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrganizationServiceServer).ListUserOrganizations(ctx, req.(*ListUserOrganizationsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _OrganizationService_CreateOrganization_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateOrganizationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrganizationServiceServer).CreateOrganization(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.OrganizationService/CreateOrganization",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrganizationServiceServer).CreateOrganization(ctx, req.(*CreateOrganizationRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _OrganizationService_UpdateOrganizationName_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateOrganizationNameRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrganizationServiceServer).UpdateOrganizationName(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.OrganizationService/UpdateOrganizationName",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrganizationServiceServer).UpdateOrganizationName(ctx, req.(*UpdateOrganizationNameRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _OrganizationService_UpdateOrganizationNameByName_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateOrganizationNameByNameRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrganizationServiceServer).UpdateOrganizationNameByName(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.OrganizationService/UpdateOrganizationNameByName",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrganizationServiceServer).UpdateOrganizationNameByName(ctx, req.(*UpdateOrganizationNameByNameRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _OrganizationService_DeleteOrganization_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteOrganizationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrganizationServiceServer).DeleteOrganization(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.OrganizationService/DeleteOrganization",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrganizationServiceServer).DeleteOrganization(ctx, req.(*DeleteOrganizationRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _OrganizationService_DeleteOrganizationByName_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteOrganizationByNameRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrganizationServiceServer).DeleteOrganizationByName(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.OrganizationService/DeleteOrganizationByName",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrganizationServiceServer).DeleteOrganizationByName(ctx, req.(*DeleteOrganizationByNameRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _OrganizationService_AddOrganizationBaseRepositoryScope_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AddOrganizationBaseRepositoryScopeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrganizationServiceServer).AddOrganizationBaseRepositoryScope(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.OrganizationService/AddOrganizationBaseRepositoryScope",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrganizationServiceServer).AddOrganizationBaseRepositoryScope(ctx, req.(*AddOrganizationBaseRepositoryScopeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _OrganizationService_AddOrganizationMember_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AddOrganizationMemberRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrganizationServiceServer).AddOrganizationMember(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.OrganizationService/AddOrganizationMember",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrganizationServiceServer).AddOrganizationMember(ctx, req.(*AddOrganizationMemberRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _OrganizationService_UpdateOrganizationMember_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateOrganizationMemberRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrganizationServiceServer).UpdateOrganizationMember(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.OrganizationService/UpdateOrganizationMember",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrganizationServiceServer).UpdateOrganizationMember(ctx, req.(*UpdateOrganizationMemberRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _OrganizationService_RemoveOrganizationMember_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RemoveOrganizationMemberRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrganizationServiceServer).RemoveOrganizationMember(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.OrganizationService/RemoveOrganizationMember",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrganizationServiceServer).RemoveOrganizationMember(ctx, req.(*RemoveOrganizationMemberRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _OrganizationService_LeaveOrganization_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(LeaveOrganizationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrganizationServiceServer).LeaveOrganization(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.OrganizationService/LeaveOrganization",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrganizationServiceServer).LeaveOrganization(ctx, req.(*LeaveOrganizationRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// OrganizationService_ServiceDesc is the grpc.ServiceDesc for OrganizationService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var OrganizationService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "buf.alpha.registry.v1alpha1.OrganizationService",
	HandlerType: (*OrganizationServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetOrganization",
			Handler:    _OrganizationService_GetOrganization_Handler,
		},
		{
			MethodName: "GetOrganizationByName",
			Handler:    _OrganizationService_GetOrganizationByName_Handler,
		},
		{
			MethodName: "ListOrganizations",
			Handler:    _OrganizationService_ListOrganizations_Handler,
		},
		{
			MethodName: "ListUserOrganizations",
			Handler:    _OrganizationService_ListUserOrganizations_Handler,
		},
		{
			MethodName: "CreateOrganization",
			Handler:    _OrganizationService_CreateOrganization_Handler,
		},
		{
			MethodName: "UpdateOrganizationName",
			Handler:    _OrganizationService_UpdateOrganizationName_Handler,
		},
		{
			MethodName: "UpdateOrganizationNameByName",
			Handler:    _OrganizationService_UpdateOrganizationNameByName_Handler,
		},
		{
			MethodName: "DeleteOrganization",
			Handler:    _OrganizationService_DeleteOrganization_Handler,
		},
		{
			MethodName: "DeleteOrganizationByName",
			Handler:    _OrganizationService_DeleteOrganizationByName_Handler,
		},
		{
			MethodName: "AddOrganizationBaseRepositoryScope",
			Handler:    _OrganizationService_AddOrganizationBaseRepositoryScope_Handler,
		},
		{
			MethodName: "AddOrganizationMember",
			Handler:    _OrganizationService_AddOrganizationMember_Handler,
		},
		{
			MethodName: "UpdateOrganizationMember",
			Handler:    _OrganizationService_UpdateOrganizationMember_Handler,
		},
		{
			MethodName: "RemoveOrganizationMember",
			Handler:    _OrganizationService_RemoveOrganizationMember_Handler,
		},
		{
			MethodName: "LeaveOrganization",
			Handler:    _OrganizationService_LeaveOrganization_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "buf/alpha/registry/v1alpha1/organization.proto",
}
