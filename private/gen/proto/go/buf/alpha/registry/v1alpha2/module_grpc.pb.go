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
// source: buf/alpha/registry/v1alpha2/module.proto

package registryv1alpha2

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

// RemoteModuleServiceClient is the client API for RemoteModuleService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type RemoteModuleServiceClient interface {
	// GetRemoteModule gets a module by ID.
	GetRemoteModule(ctx context.Context, in *GetRemoteModuleRequest, opts ...grpc.CallOption) (*GetRemoteModuleResponse, error)
	// GetRemoteModuleByFullName gets a module by full name.
	GetRemoteModuleByFullName(ctx context.Context, in *GetRemoteModuleByFullNameRequest, opts ...grpc.CallOption) (*GetRemoteModuleByFullNameResponse, error)
	// ListRemoteModules lists all modules.
	ListRemoteModules(ctx context.Context, in *ListRemoteModulesRequest, opts ...grpc.CallOption) (*ListRemoteModulesResponse, error)
	// ListUserRemoteModules lists all modules belonging to a user.
	ListUserRemoteModules(ctx context.Context, in *ListUserRemoteModulesRequest, opts ...grpc.CallOption) (*ListUserRemoteModulesResponse, error)
	// ListRemoteModulesUserCanAccess lists all modules a user can access.
	ListRemoteModulesUserCanAccess(ctx context.Context, in *ListRemoteModulesUserCanAccessRequest, opts ...grpc.CallOption) (*ListRemoteModulesUserCanAccessResponse, error)
	// ListOrganizationRemoteModules lists all modules for an organization.
	ListOrganizationRemoteModules(ctx context.Context, in *ListOrganizationRemoteModulesRequest, opts ...grpc.CallOption) (*ListOrganizationRemoteModulesResponse, error)
	// CreateRemoteModuleByFullName creates a new module by full name.
	CreateRemoteModuleByFullName(ctx context.Context, in *CreateRemoteModuleByFullNameRequest, opts ...grpc.CallOption) (*CreateRemoteModuleByFullNameResponse, error)
	// DeleteRemoteModule deletes a module.
	DeleteRemoteModule(ctx context.Context, in *DeleteRemoteModuleRequest, opts ...grpc.CallOption) (*DeleteRemoteModuleResponse, error)
	// DeleteRemoteModuleByFullName deletes a module by full name.
	DeleteRemoteModuleByFullName(ctx context.Context, in *DeleteRemoteModuleByFullNameRequest, opts ...grpc.CallOption) (*DeleteRemoteModuleByFullNameResponse, error)
	// DeprecateRemoteModuleByName deprecates the module.
	DeprecateRemoteModuleByName(ctx context.Context, in *DeprecateRemoteModuleByNameRequest, opts ...grpc.CallOption) (*DeprecateRemoteModuleByNameResponse, error)
	// UndeprecateRemoteModuleByName makes the module not deprecated and removes any deprecation_message.
	UndeprecateRemoteModuleByName(ctx context.Context, in *UndeprecateRemoteModuleByNameRequest, opts ...grpc.CallOption) (*UndeprecateRemoteModuleByNameResponse, error)
	// GetRemoteModulesByFullName gets modules by full name. Response order is unspecified.
	// Errors if any of the modules don't exist or the caller does not have access to any of the modules.
	GetRemoteModulesByFullName(ctx context.Context, in *GetRemoteModulesByFullNameRequest, opts ...grpc.CallOption) (*GetRemoteModulesByFullNameResponse, error)
	// SetRemoteModuleContributor sets the role of a user in the module.
	SetRemoteModuleContributor(ctx context.Context, in *SetRemoteModuleContributorRequest, opts ...grpc.CallOption) (*SetRemoteModuleContributorResponse, error)
	// ListRemoteModuleContributors returns the list of contributors that has an explicit role against the module.
	// This does not include users who have implicit roles against the module, unless they have also been
	// assigned a role explicitly.
	ListRemoteModuleContributors(ctx context.Context, in *ListRemoteModuleContributorsRequest, opts ...grpc.CallOption) (*ListRemoteModuleContributorsResponse, error)
	// GetRemoteModuleContributor returns the contributor information of a user in a module.
	GetRemoteModuleContributor(ctx context.Context, in *GetRemoteModuleContributorRequest, opts ...grpc.CallOption) (*GetRemoteModuleContributorResponse, error)
	// GetRemoteModuleSettings gets the settings of a module.
	GetRemoteModuleSettings(ctx context.Context, in *GetRemoteModuleSettingsRequest, opts ...grpc.CallOption) (*GetRemoteModuleSettingsResponse, error)
	// UpdateRemoteModuleSettingsByName updates the settings of a module.
	UpdateRemoteModuleSettingsByName(ctx context.Context, in *UpdateRemoteModuleSettingsByNameRequest, opts ...grpc.CallOption) (*UpdateRemoteModuleSettingsByNameResponse, error)
}

type remoteModuleServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewRemoteModuleServiceClient(cc grpc.ClientConnInterface) RemoteModuleServiceClient {
	return &remoteModuleServiceClient{cc}
}

func (c *remoteModuleServiceClient) GetRemoteModule(ctx context.Context, in *GetRemoteModuleRequest, opts ...grpc.CallOption) (*GetRemoteModuleResponse, error) {
	out := new(GetRemoteModuleResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha2.RemoteModuleService/GetRemoteModule", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *remoteModuleServiceClient) GetRemoteModuleByFullName(ctx context.Context, in *GetRemoteModuleByFullNameRequest, opts ...grpc.CallOption) (*GetRemoteModuleByFullNameResponse, error) {
	out := new(GetRemoteModuleByFullNameResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha2.RemoteModuleService/GetRemoteModuleByFullName", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *remoteModuleServiceClient) ListRemoteModules(ctx context.Context, in *ListRemoteModulesRequest, opts ...grpc.CallOption) (*ListRemoteModulesResponse, error) {
	out := new(ListRemoteModulesResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha2.RemoteModuleService/ListRemoteModules", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *remoteModuleServiceClient) ListUserRemoteModules(ctx context.Context, in *ListUserRemoteModulesRequest, opts ...grpc.CallOption) (*ListUserRemoteModulesResponse, error) {
	out := new(ListUserRemoteModulesResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha2.RemoteModuleService/ListUserRemoteModules", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *remoteModuleServiceClient) ListRemoteModulesUserCanAccess(ctx context.Context, in *ListRemoteModulesUserCanAccessRequest, opts ...grpc.CallOption) (*ListRemoteModulesUserCanAccessResponse, error) {
	out := new(ListRemoteModulesUserCanAccessResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha2.RemoteModuleService/ListRemoteModulesUserCanAccess", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *remoteModuleServiceClient) ListOrganizationRemoteModules(ctx context.Context, in *ListOrganizationRemoteModulesRequest, opts ...grpc.CallOption) (*ListOrganizationRemoteModulesResponse, error) {
	out := new(ListOrganizationRemoteModulesResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha2.RemoteModuleService/ListOrganizationRemoteModules", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *remoteModuleServiceClient) CreateRemoteModuleByFullName(ctx context.Context, in *CreateRemoteModuleByFullNameRequest, opts ...grpc.CallOption) (*CreateRemoteModuleByFullNameResponse, error) {
	out := new(CreateRemoteModuleByFullNameResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha2.RemoteModuleService/CreateRemoteModuleByFullName", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *remoteModuleServiceClient) DeleteRemoteModule(ctx context.Context, in *DeleteRemoteModuleRequest, opts ...grpc.CallOption) (*DeleteRemoteModuleResponse, error) {
	out := new(DeleteRemoteModuleResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha2.RemoteModuleService/DeleteRemoteModule", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *remoteModuleServiceClient) DeleteRemoteModuleByFullName(ctx context.Context, in *DeleteRemoteModuleByFullNameRequest, opts ...grpc.CallOption) (*DeleteRemoteModuleByFullNameResponse, error) {
	out := new(DeleteRemoteModuleByFullNameResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha2.RemoteModuleService/DeleteRemoteModuleByFullName", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *remoteModuleServiceClient) DeprecateRemoteModuleByName(ctx context.Context, in *DeprecateRemoteModuleByNameRequest, opts ...grpc.CallOption) (*DeprecateRemoteModuleByNameResponse, error) {
	out := new(DeprecateRemoteModuleByNameResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha2.RemoteModuleService/DeprecateRemoteModuleByName", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *remoteModuleServiceClient) UndeprecateRemoteModuleByName(ctx context.Context, in *UndeprecateRemoteModuleByNameRequest, opts ...grpc.CallOption) (*UndeprecateRemoteModuleByNameResponse, error) {
	out := new(UndeprecateRemoteModuleByNameResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha2.RemoteModuleService/UndeprecateRemoteModuleByName", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *remoteModuleServiceClient) GetRemoteModulesByFullName(ctx context.Context, in *GetRemoteModulesByFullNameRequest, opts ...grpc.CallOption) (*GetRemoteModulesByFullNameResponse, error) {
	out := new(GetRemoteModulesByFullNameResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha2.RemoteModuleService/GetRemoteModulesByFullName", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *remoteModuleServiceClient) SetRemoteModuleContributor(ctx context.Context, in *SetRemoteModuleContributorRequest, opts ...grpc.CallOption) (*SetRemoteModuleContributorResponse, error) {
	out := new(SetRemoteModuleContributorResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha2.RemoteModuleService/SetRemoteModuleContributor", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *remoteModuleServiceClient) ListRemoteModuleContributors(ctx context.Context, in *ListRemoteModuleContributorsRequest, opts ...grpc.CallOption) (*ListRemoteModuleContributorsResponse, error) {
	out := new(ListRemoteModuleContributorsResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha2.RemoteModuleService/ListRemoteModuleContributors", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *remoteModuleServiceClient) GetRemoteModuleContributor(ctx context.Context, in *GetRemoteModuleContributorRequest, opts ...grpc.CallOption) (*GetRemoteModuleContributorResponse, error) {
	out := new(GetRemoteModuleContributorResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha2.RemoteModuleService/GetRemoteModuleContributor", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *remoteModuleServiceClient) GetRemoteModuleSettings(ctx context.Context, in *GetRemoteModuleSettingsRequest, opts ...grpc.CallOption) (*GetRemoteModuleSettingsResponse, error) {
	out := new(GetRemoteModuleSettingsResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha2.RemoteModuleService/GetRemoteModuleSettings", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *remoteModuleServiceClient) UpdateRemoteModuleSettingsByName(ctx context.Context, in *UpdateRemoteModuleSettingsByNameRequest, opts ...grpc.CallOption) (*UpdateRemoteModuleSettingsByNameResponse, error) {
	out := new(UpdateRemoteModuleSettingsByNameResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha2.RemoteModuleService/UpdateRemoteModuleSettingsByName", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RemoteModuleServiceServer is the server API for RemoteModuleService service.
// All implementations should embed UnimplementedRemoteModuleServiceServer
// for forward compatibility
type RemoteModuleServiceServer interface {
	// GetRemoteModule gets a module by ID.
	GetRemoteModule(context.Context, *GetRemoteModuleRequest) (*GetRemoteModuleResponse, error)
	// GetRemoteModuleByFullName gets a module by full name.
	GetRemoteModuleByFullName(context.Context, *GetRemoteModuleByFullNameRequest) (*GetRemoteModuleByFullNameResponse, error)
	// ListRemoteModules lists all modules.
	ListRemoteModules(context.Context, *ListRemoteModulesRequest) (*ListRemoteModulesResponse, error)
	// ListUserRemoteModules lists all modules belonging to a user.
	ListUserRemoteModules(context.Context, *ListUserRemoteModulesRequest) (*ListUserRemoteModulesResponse, error)
	// ListRemoteModulesUserCanAccess lists all modules a user can access.
	ListRemoteModulesUserCanAccess(context.Context, *ListRemoteModulesUserCanAccessRequest) (*ListRemoteModulesUserCanAccessResponse, error)
	// ListOrganizationRemoteModules lists all modules for an organization.
	ListOrganizationRemoteModules(context.Context, *ListOrganizationRemoteModulesRequest) (*ListOrganizationRemoteModulesResponse, error)
	// CreateRemoteModuleByFullName creates a new module by full name.
	CreateRemoteModuleByFullName(context.Context, *CreateRemoteModuleByFullNameRequest) (*CreateRemoteModuleByFullNameResponse, error)
	// DeleteRemoteModule deletes a module.
	DeleteRemoteModule(context.Context, *DeleteRemoteModuleRequest) (*DeleteRemoteModuleResponse, error)
	// DeleteRemoteModuleByFullName deletes a module by full name.
	DeleteRemoteModuleByFullName(context.Context, *DeleteRemoteModuleByFullNameRequest) (*DeleteRemoteModuleByFullNameResponse, error)
	// DeprecateRemoteModuleByName deprecates the module.
	DeprecateRemoteModuleByName(context.Context, *DeprecateRemoteModuleByNameRequest) (*DeprecateRemoteModuleByNameResponse, error)
	// UndeprecateRemoteModuleByName makes the module not deprecated and removes any deprecation_message.
	UndeprecateRemoteModuleByName(context.Context, *UndeprecateRemoteModuleByNameRequest) (*UndeprecateRemoteModuleByNameResponse, error)
	// GetRemoteModulesByFullName gets modules by full name. Response order is unspecified.
	// Errors if any of the modules don't exist or the caller does not have access to any of the modules.
	GetRemoteModulesByFullName(context.Context, *GetRemoteModulesByFullNameRequest) (*GetRemoteModulesByFullNameResponse, error)
	// SetRemoteModuleContributor sets the role of a user in the module.
	SetRemoteModuleContributor(context.Context, *SetRemoteModuleContributorRequest) (*SetRemoteModuleContributorResponse, error)
	// ListRemoteModuleContributors returns the list of contributors that has an explicit role against the module.
	// This does not include users who have implicit roles against the module, unless they have also been
	// assigned a role explicitly.
	ListRemoteModuleContributors(context.Context, *ListRemoteModuleContributorsRequest) (*ListRemoteModuleContributorsResponse, error)
	// GetRemoteModuleContributor returns the contributor information of a user in a module.
	GetRemoteModuleContributor(context.Context, *GetRemoteModuleContributorRequest) (*GetRemoteModuleContributorResponse, error)
	// GetRemoteModuleSettings gets the settings of a module.
	GetRemoteModuleSettings(context.Context, *GetRemoteModuleSettingsRequest) (*GetRemoteModuleSettingsResponse, error)
	// UpdateRemoteModuleSettingsByName updates the settings of a module.
	UpdateRemoteModuleSettingsByName(context.Context, *UpdateRemoteModuleSettingsByNameRequest) (*UpdateRemoteModuleSettingsByNameResponse, error)
}

// UnimplementedRemoteModuleServiceServer should be embedded to have forward compatible implementations.
type UnimplementedRemoteModuleServiceServer struct {
}

func (UnimplementedRemoteModuleServiceServer) GetRemoteModule(context.Context, *GetRemoteModuleRequest) (*GetRemoteModuleResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetRemoteModule not implemented")
}
func (UnimplementedRemoteModuleServiceServer) GetRemoteModuleByFullName(context.Context, *GetRemoteModuleByFullNameRequest) (*GetRemoteModuleByFullNameResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetRemoteModuleByFullName not implemented")
}
func (UnimplementedRemoteModuleServiceServer) ListRemoteModules(context.Context, *ListRemoteModulesRequest) (*ListRemoteModulesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListRemoteModules not implemented")
}
func (UnimplementedRemoteModuleServiceServer) ListUserRemoteModules(context.Context, *ListUserRemoteModulesRequest) (*ListUserRemoteModulesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListUserRemoteModules not implemented")
}
func (UnimplementedRemoteModuleServiceServer) ListRemoteModulesUserCanAccess(context.Context, *ListRemoteModulesUserCanAccessRequest) (*ListRemoteModulesUserCanAccessResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListRemoteModulesUserCanAccess not implemented")
}
func (UnimplementedRemoteModuleServiceServer) ListOrganizationRemoteModules(context.Context, *ListOrganizationRemoteModulesRequest) (*ListOrganizationRemoteModulesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListOrganizationRemoteModules not implemented")
}
func (UnimplementedRemoteModuleServiceServer) CreateRemoteModuleByFullName(context.Context, *CreateRemoteModuleByFullNameRequest) (*CreateRemoteModuleByFullNameResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateRemoteModuleByFullName not implemented")
}
func (UnimplementedRemoteModuleServiceServer) DeleteRemoteModule(context.Context, *DeleteRemoteModuleRequest) (*DeleteRemoteModuleResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteRemoteModule not implemented")
}
func (UnimplementedRemoteModuleServiceServer) DeleteRemoteModuleByFullName(context.Context, *DeleteRemoteModuleByFullNameRequest) (*DeleteRemoteModuleByFullNameResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteRemoteModuleByFullName not implemented")
}
func (UnimplementedRemoteModuleServiceServer) DeprecateRemoteModuleByName(context.Context, *DeprecateRemoteModuleByNameRequest) (*DeprecateRemoteModuleByNameResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeprecateRemoteModuleByName not implemented")
}
func (UnimplementedRemoteModuleServiceServer) UndeprecateRemoteModuleByName(context.Context, *UndeprecateRemoteModuleByNameRequest) (*UndeprecateRemoteModuleByNameResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UndeprecateRemoteModuleByName not implemented")
}
func (UnimplementedRemoteModuleServiceServer) GetRemoteModulesByFullName(context.Context, *GetRemoteModulesByFullNameRequest) (*GetRemoteModulesByFullNameResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetRemoteModulesByFullName not implemented")
}
func (UnimplementedRemoteModuleServiceServer) SetRemoteModuleContributor(context.Context, *SetRemoteModuleContributorRequest) (*SetRemoteModuleContributorResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetRemoteModuleContributor not implemented")
}
func (UnimplementedRemoteModuleServiceServer) ListRemoteModuleContributors(context.Context, *ListRemoteModuleContributorsRequest) (*ListRemoteModuleContributorsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListRemoteModuleContributors not implemented")
}
func (UnimplementedRemoteModuleServiceServer) GetRemoteModuleContributor(context.Context, *GetRemoteModuleContributorRequest) (*GetRemoteModuleContributorResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetRemoteModuleContributor not implemented")
}
func (UnimplementedRemoteModuleServiceServer) GetRemoteModuleSettings(context.Context, *GetRemoteModuleSettingsRequest) (*GetRemoteModuleSettingsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetRemoteModuleSettings not implemented")
}
func (UnimplementedRemoteModuleServiceServer) UpdateRemoteModuleSettingsByName(context.Context, *UpdateRemoteModuleSettingsByNameRequest) (*UpdateRemoteModuleSettingsByNameResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateRemoteModuleSettingsByName not implemented")
}

// UnsafeRemoteModuleServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to RemoteModuleServiceServer will
// result in compilation errors.
type UnsafeRemoteModuleServiceServer interface {
	mustEmbedUnimplementedRemoteModuleServiceServer()
}

func RegisterRemoteModuleServiceServer(s grpc.ServiceRegistrar, srv RemoteModuleServiceServer) {
	s.RegisterService(&RemoteModuleService_ServiceDesc, srv)
}

func _RemoteModuleService_GetRemoteModule_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetRemoteModuleRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RemoteModuleServiceServer).GetRemoteModule(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha2.RemoteModuleService/GetRemoteModule",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RemoteModuleServiceServer).GetRemoteModule(ctx, req.(*GetRemoteModuleRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RemoteModuleService_GetRemoteModuleByFullName_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetRemoteModuleByFullNameRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RemoteModuleServiceServer).GetRemoteModuleByFullName(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha2.RemoteModuleService/GetRemoteModuleByFullName",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RemoteModuleServiceServer).GetRemoteModuleByFullName(ctx, req.(*GetRemoteModuleByFullNameRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RemoteModuleService_ListRemoteModules_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListRemoteModulesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RemoteModuleServiceServer).ListRemoteModules(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha2.RemoteModuleService/ListRemoteModules",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RemoteModuleServiceServer).ListRemoteModules(ctx, req.(*ListRemoteModulesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RemoteModuleService_ListUserRemoteModules_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListUserRemoteModulesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RemoteModuleServiceServer).ListUserRemoteModules(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha2.RemoteModuleService/ListUserRemoteModules",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RemoteModuleServiceServer).ListUserRemoteModules(ctx, req.(*ListUserRemoteModulesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RemoteModuleService_ListRemoteModulesUserCanAccess_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListRemoteModulesUserCanAccessRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RemoteModuleServiceServer).ListRemoteModulesUserCanAccess(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha2.RemoteModuleService/ListRemoteModulesUserCanAccess",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RemoteModuleServiceServer).ListRemoteModulesUserCanAccess(ctx, req.(*ListRemoteModulesUserCanAccessRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RemoteModuleService_ListOrganizationRemoteModules_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListOrganizationRemoteModulesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RemoteModuleServiceServer).ListOrganizationRemoteModules(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha2.RemoteModuleService/ListOrganizationRemoteModules",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RemoteModuleServiceServer).ListOrganizationRemoteModules(ctx, req.(*ListOrganizationRemoteModulesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RemoteModuleService_CreateRemoteModuleByFullName_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateRemoteModuleByFullNameRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RemoteModuleServiceServer).CreateRemoteModuleByFullName(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha2.RemoteModuleService/CreateRemoteModuleByFullName",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RemoteModuleServiceServer).CreateRemoteModuleByFullName(ctx, req.(*CreateRemoteModuleByFullNameRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RemoteModuleService_DeleteRemoteModule_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteRemoteModuleRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RemoteModuleServiceServer).DeleteRemoteModule(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha2.RemoteModuleService/DeleteRemoteModule",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RemoteModuleServiceServer).DeleteRemoteModule(ctx, req.(*DeleteRemoteModuleRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RemoteModuleService_DeleteRemoteModuleByFullName_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteRemoteModuleByFullNameRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RemoteModuleServiceServer).DeleteRemoteModuleByFullName(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha2.RemoteModuleService/DeleteRemoteModuleByFullName",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RemoteModuleServiceServer).DeleteRemoteModuleByFullName(ctx, req.(*DeleteRemoteModuleByFullNameRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RemoteModuleService_DeprecateRemoteModuleByName_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeprecateRemoteModuleByNameRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RemoteModuleServiceServer).DeprecateRemoteModuleByName(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha2.RemoteModuleService/DeprecateRemoteModuleByName",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RemoteModuleServiceServer).DeprecateRemoteModuleByName(ctx, req.(*DeprecateRemoteModuleByNameRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RemoteModuleService_UndeprecateRemoteModuleByName_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UndeprecateRemoteModuleByNameRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RemoteModuleServiceServer).UndeprecateRemoteModuleByName(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha2.RemoteModuleService/UndeprecateRemoteModuleByName",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RemoteModuleServiceServer).UndeprecateRemoteModuleByName(ctx, req.(*UndeprecateRemoteModuleByNameRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RemoteModuleService_GetRemoteModulesByFullName_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetRemoteModulesByFullNameRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RemoteModuleServiceServer).GetRemoteModulesByFullName(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha2.RemoteModuleService/GetRemoteModulesByFullName",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RemoteModuleServiceServer).GetRemoteModulesByFullName(ctx, req.(*GetRemoteModulesByFullNameRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RemoteModuleService_SetRemoteModuleContributor_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetRemoteModuleContributorRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RemoteModuleServiceServer).SetRemoteModuleContributor(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha2.RemoteModuleService/SetRemoteModuleContributor",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RemoteModuleServiceServer).SetRemoteModuleContributor(ctx, req.(*SetRemoteModuleContributorRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RemoteModuleService_ListRemoteModuleContributors_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListRemoteModuleContributorsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RemoteModuleServiceServer).ListRemoteModuleContributors(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha2.RemoteModuleService/ListRemoteModuleContributors",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RemoteModuleServiceServer).ListRemoteModuleContributors(ctx, req.(*ListRemoteModuleContributorsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RemoteModuleService_GetRemoteModuleContributor_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetRemoteModuleContributorRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RemoteModuleServiceServer).GetRemoteModuleContributor(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha2.RemoteModuleService/GetRemoteModuleContributor",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RemoteModuleServiceServer).GetRemoteModuleContributor(ctx, req.(*GetRemoteModuleContributorRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RemoteModuleService_GetRemoteModuleSettings_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetRemoteModuleSettingsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RemoteModuleServiceServer).GetRemoteModuleSettings(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha2.RemoteModuleService/GetRemoteModuleSettings",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RemoteModuleServiceServer).GetRemoteModuleSettings(ctx, req.(*GetRemoteModuleSettingsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RemoteModuleService_UpdateRemoteModuleSettingsByName_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateRemoteModuleSettingsByNameRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RemoteModuleServiceServer).UpdateRemoteModuleSettingsByName(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha2.RemoteModuleService/UpdateRemoteModuleSettingsByName",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RemoteModuleServiceServer).UpdateRemoteModuleSettingsByName(ctx, req.(*UpdateRemoteModuleSettingsByNameRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// RemoteModuleService_ServiceDesc is the grpc.ServiceDesc for RemoteModuleService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var RemoteModuleService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "buf.alpha.registry.v1alpha2.RemoteModuleService",
	HandlerType: (*RemoteModuleServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetRemoteModule",
			Handler:    _RemoteModuleService_GetRemoteModule_Handler,
		},
		{
			MethodName: "GetRemoteModuleByFullName",
			Handler:    _RemoteModuleService_GetRemoteModuleByFullName_Handler,
		},
		{
			MethodName: "ListRemoteModules",
			Handler:    _RemoteModuleService_ListRemoteModules_Handler,
		},
		{
			MethodName: "ListUserRemoteModules",
			Handler:    _RemoteModuleService_ListUserRemoteModules_Handler,
		},
		{
			MethodName: "ListRemoteModulesUserCanAccess",
			Handler:    _RemoteModuleService_ListRemoteModulesUserCanAccess_Handler,
		},
		{
			MethodName: "ListOrganizationRemoteModules",
			Handler:    _RemoteModuleService_ListOrganizationRemoteModules_Handler,
		},
		{
			MethodName: "CreateRemoteModuleByFullName",
			Handler:    _RemoteModuleService_CreateRemoteModuleByFullName_Handler,
		},
		{
			MethodName: "DeleteRemoteModule",
			Handler:    _RemoteModuleService_DeleteRemoteModule_Handler,
		},
		{
			MethodName: "DeleteRemoteModuleByFullName",
			Handler:    _RemoteModuleService_DeleteRemoteModuleByFullName_Handler,
		},
		{
			MethodName: "DeprecateRemoteModuleByName",
			Handler:    _RemoteModuleService_DeprecateRemoteModuleByName_Handler,
		},
		{
			MethodName: "UndeprecateRemoteModuleByName",
			Handler:    _RemoteModuleService_UndeprecateRemoteModuleByName_Handler,
		},
		{
			MethodName: "GetRemoteModulesByFullName",
			Handler:    _RemoteModuleService_GetRemoteModulesByFullName_Handler,
		},
		{
			MethodName: "SetRemoteModuleContributor",
			Handler:    _RemoteModuleService_SetRemoteModuleContributor_Handler,
		},
		{
			MethodName: "ListRemoteModuleContributors",
			Handler:    _RemoteModuleService_ListRemoteModuleContributors_Handler,
		},
		{
			MethodName: "GetRemoteModuleContributor",
			Handler:    _RemoteModuleService_GetRemoteModuleContributor_Handler,
		},
		{
			MethodName: "GetRemoteModuleSettings",
			Handler:    _RemoteModuleService_GetRemoteModuleSettings_Handler,
		},
		{
			MethodName: "UpdateRemoteModuleSettingsByName",
			Handler:    _RemoteModuleService_UpdateRemoteModuleSettingsByName_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "buf/alpha/registry/v1alpha2/module.proto",
}
