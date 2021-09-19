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
// source: buf/alpha/registry/v1alpha1/authz.proto

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

// AuthzServiceClient is the client API for AuthzService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type AuthzServiceClient interface {
	// UserCanAddUserOrganizationScopes returns whether the user is authorized
	// to remove user scopes from an organization.
	UserCanAddUserOrganizationScopes(ctx context.Context, in *UserCanAddUserOrganizationScopesRequest, opts ...grpc.CallOption) (*UserCanAddUserOrganizationScopesResponse, error)
	// UserCanRemoveUserOrganizationScopes returns whether the user is authorized
	// to remove user scopes from an organization.
	UserCanRemoveUserOrganizationScopes(ctx context.Context, in *UserCanRemoveUserOrganizationScopesRequest, opts ...grpc.CallOption) (*UserCanRemoveUserOrganizationScopesResponse, error)
	// UserCanCreateOrganizationRepository returns whether the user is authorized
	// to create repositories in an organization.
	UserCanCreateOrganizationRepository(ctx context.Context, in *UserCanCreateOrganizationRepositoryRequest, opts ...grpc.CallOption) (*UserCanCreateOrganizationRepositoryResponse, error)
	// UserCanCreateOrganizationTeam returns whether the user is authorized
	// to create teams in an organization.
	UserCanCreateOrganizationTeam(ctx context.Context, in *UserCanCreateOrganizationTeamRequest, opts ...grpc.CallOption) (*UserCanCreateOrganizationTeamResponse, error)
	// UserCanListOrganizationTeams returns whether the user is authorized
	// to list teams in an organization.
	UserCanListOrganizationTeams(ctx context.Context, in *UserCanListOrganizationTeamsRequest, opts ...grpc.CallOption) (*UserCanListOrganizationTeamsResponse, error)
	// UserCanSeeRepositorySettings returns whether the user is authorized
	// to see repository settings.
	UserCanSeeRepositorySettings(ctx context.Context, in *UserCanSeeRepositorySettingsRequest, opts ...grpc.CallOption) (*UserCanSeeRepositorySettingsResponse, error)
	// UserCanSeeOrganizationSettings returns whether the user is authorized
	// to see organization settings.
	UserCanSeeOrganizationSettings(ctx context.Context, in *UserCanSeeOrganizationSettingsRequest, opts ...grpc.CallOption) (*UserCanSeeOrganizationSettingsResponse, error)
	// UserCanReadPlugin returns whether the user has read access to the specified plugin.
	UserCanReadPlugin(ctx context.Context, in *UserCanReadPluginRequest, opts ...grpc.CallOption) (*UserCanReadPluginResponse, error)
	// UserCanCreatePluginVersion returns whether the user is authorized
	// to create a plugin version under the specified plugin.
	UserCanCreatePluginVersion(ctx context.Context, in *UserCanCreatePluginVersionRequest, opts ...grpc.CallOption) (*UserCanCreatePluginVersionResponse, error)
	// UserCanCreateTemplateVersion returns whether the user is authorized
	// to create a template version under the specified template.
	UserCanCreateTemplateVersion(ctx context.Context, in *UserCanCreateTemplateVersionRequest, opts ...grpc.CallOption) (*UserCanCreateTemplateVersionResponse, error)
	// UserCanCreateOrganizationPlugin returns whether the user is authorized to create
	// a plugin in an organization.
	UserCanCreateOrganizationPlugin(ctx context.Context, in *UserCanCreateOrganizationPluginRequest, opts ...grpc.CallOption) (*UserCanCreateOrganizationPluginResponse, error)
	// UserCanCreateOrganizationPlugin returns whether the user is authorized to create
	// a template in an organization.
	UserCanCreateOrganizationTemplate(ctx context.Context, in *UserCanCreateOrganizationTemplateRequest, opts ...grpc.CallOption) (*UserCanCreateOrganizationTemplateResponse, error)
	// UserCanSeePluginSettings returns whether the user is authorized
	// to see plugin settings.
	UserCanSeePluginSettings(ctx context.Context, in *UserCanSeePluginSettingsRequest, opts ...grpc.CallOption) (*UserCanSeePluginSettingsResponse, error)
	// UserCanSeeTemplateSettings returns whether the user is authorized
	// to see template settings.
	UserCanSeeTemplateSettings(ctx context.Context, in *UserCanSeeTemplateSettingsRequest, opts ...grpc.CallOption) (*UserCanSeeTemplateSettingsResponse, error)
}

type authzServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewAuthzServiceClient(cc grpc.ClientConnInterface) AuthzServiceClient {
	return &authzServiceClient{cc}
}

func (c *authzServiceClient) UserCanAddUserOrganizationScopes(ctx context.Context, in *UserCanAddUserOrganizationScopesRequest, opts ...grpc.CallOption) (*UserCanAddUserOrganizationScopesResponse, error) {
	out := new(UserCanAddUserOrganizationScopesResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.AuthzService/UserCanAddUserOrganizationScopes", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *authzServiceClient) UserCanRemoveUserOrganizationScopes(ctx context.Context, in *UserCanRemoveUserOrganizationScopesRequest, opts ...grpc.CallOption) (*UserCanRemoveUserOrganizationScopesResponse, error) {
	out := new(UserCanRemoveUserOrganizationScopesResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.AuthzService/UserCanRemoveUserOrganizationScopes", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *authzServiceClient) UserCanCreateOrganizationRepository(ctx context.Context, in *UserCanCreateOrganizationRepositoryRequest, opts ...grpc.CallOption) (*UserCanCreateOrganizationRepositoryResponse, error) {
	out := new(UserCanCreateOrganizationRepositoryResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.AuthzService/UserCanCreateOrganizationRepository", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *authzServiceClient) UserCanCreateOrganizationTeam(ctx context.Context, in *UserCanCreateOrganizationTeamRequest, opts ...grpc.CallOption) (*UserCanCreateOrganizationTeamResponse, error) {
	out := new(UserCanCreateOrganizationTeamResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.AuthzService/UserCanCreateOrganizationTeam", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *authzServiceClient) UserCanListOrganizationTeams(ctx context.Context, in *UserCanListOrganizationTeamsRequest, opts ...grpc.CallOption) (*UserCanListOrganizationTeamsResponse, error) {
	out := new(UserCanListOrganizationTeamsResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.AuthzService/UserCanListOrganizationTeams", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *authzServiceClient) UserCanSeeRepositorySettings(ctx context.Context, in *UserCanSeeRepositorySettingsRequest, opts ...grpc.CallOption) (*UserCanSeeRepositorySettingsResponse, error) {
	out := new(UserCanSeeRepositorySettingsResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.AuthzService/UserCanSeeRepositorySettings", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *authzServiceClient) UserCanSeeOrganizationSettings(ctx context.Context, in *UserCanSeeOrganizationSettingsRequest, opts ...grpc.CallOption) (*UserCanSeeOrganizationSettingsResponse, error) {
	out := new(UserCanSeeOrganizationSettingsResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.AuthzService/UserCanSeeOrganizationSettings", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *authzServiceClient) UserCanReadPlugin(ctx context.Context, in *UserCanReadPluginRequest, opts ...grpc.CallOption) (*UserCanReadPluginResponse, error) {
	out := new(UserCanReadPluginResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.AuthzService/UserCanReadPlugin", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *authzServiceClient) UserCanCreatePluginVersion(ctx context.Context, in *UserCanCreatePluginVersionRequest, opts ...grpc.CallOption) (*UserCanCreatePluginVersionResponse, error) {
	out := new(UserCanCreatePluginVersionResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.AuthzService/UserCanCreatePluginVersion", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *authzServiceClient) UserCanCreateTemplateVersion(ctx context.Context, in *UserCanCreateTemplateVersionRequest, opts ...grpc.CallOption) (*UserCanCreateTemplateVersionResponse, error) {
	out := new(UserCanCreateTemplateVersionResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.AuthzService/UserCanCreateTemplateVersion", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *authzServiceClient) UserCanCreateOrganizationPlugin(ctx context.Context, in *UserCanCreateOrganizationPluginRequest, opts ...grpc.CallOption) (*UserCanCreateOrganizationPluginResponse, error) {
	out := new(UserCanCreateOrganizationPluginResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.AuthzService/UserCanCreateOrganizationPlugin", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *authzServiceClient) UserCanCreateOrganizationTemplate(ctx context.Context, in *UserCanCreateOrganizationTemplateRequest, opts ...grpc.CallOption) (*UserCanCreateOrganizationTemplateResponse, error) {
	out := new(UserCanCreateOrganizationTemplateResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.AuthzService/UserCanCreateOrganizationTemplate", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *authzServiceClient) UserCanSeePluginSettings(ctx context.Context, in *UserCanSeePluginSettingsRequest, opts ...grpc.CallOption) (*UserCanSeePluginSettingsResponse, error) {
	out := new(UserCanSeePluginSettingsResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.AuthzService/UserCanSeePluginSettings", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *authzServiceClient) UserCanSeeTemplateSettings(ctx context.Context, in *UserCanSeeTemplateSettingsRequest, opts ...grpc.CallOption) (*UserCanSeeTemplateSettingsResponse, error) {
	out := new(UserCanSeeTemplateSettingsResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.AuthzService/UserCanSeeTemplateSettings", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// AuthzServiceServer is the server API for AuthzService service.
// All implementations should embed UnimplementedAuthzServiceServer
// for forward compatibility
type AuthzServiceServer interface {
	// UserCanAddUserOrganizationScopes returns whether the user is authorized
	// to remove user scopes from an organization.
	UserCanAddUserOrganizationScopes(context.Context, *UserCanAddUserOrganizationScopesRequest) (*UserCanAddUserOrganizationScopesResponse, error)
	// UserCanRemoveUserOrganizationScopes returns whether the user is authorized
	// to remove user scopes from an organization.
	UserCanRemoveUserOrganizationScopes(context.Context, *UserCanRemoveUserOrganizationScopesRequest) (*UserCanRemoveUserOrganizationScopesResponse, error)
	// UserCanCreateOrganizationRepository returns whether the user is authorized
	// to create repositories in an organization.
	UserCanCreateOrganizationRepository(context.Context, *UserCanCreateOrganizationRepositoryRequest) (*UserCanCreateOrganizationRepositoryResponse, error)
	// UserCanCreateOrganizationTeam returns whether the user is authorized
	// to create teams in an organization.
	UserCanCreateOrganizationTeam(context.Context, *UserCanCreateOrganizationTeamRequest) (*UserCanCreateOrganizationTeamResponse, error)
	// UserCanListOrganizationTeams returns whether the user is authorized
	// to list teams in an organization.
	UserCanListOrganizationTeams(context.Context, *UserCanListOrganizationTeamsRequest) (*UserCanListOrganizationTeamsResponse, error)
	// UserCanSeeRepositorySettings returns whether the user is authorized
	// to see repository settings.
	UserCanSeeRepositorySettings(context.Context, *UserCanSeeRepositorySettingsRequest) (*UserCanSeeRepositorySettingsResponse, error)
	// UserCanSeeOrganizationSettings returns whether the user is authorized
	// to see organization settings.
	UserCanSeeOrganizationSettings(context.Context, *UserCanSeeOrganizationSettingsRequest) (*UserCanSeeOrganizationSettingsResponse, error)
	// UserCanReadPlugin returns whether the user has read access to the specified plugin.
	UserCanReadPlugin(context.Context, *UserCanReadPluginRequest) (*UserCanReadPluginResponse, error)
	// UserCanCreatePluginVersion returns whether the user is authorized
	// to create a plugin version under the specified plugin.
	UserCanCreatePluginVersion(context.Context, *UserCanCreatePluginVersionRequest) (*UserCanCreatePluginVersionResponse, error)
	// UserCanCreateTemplateVersion returns whether the user is authorized
	// to create a template version under the specified template.
	UserCanCreateTemplateVersion(context.Context, *UserCanCreateTemplateVersionRequest) (*UserCanCreateTemplateVersionResponse, error)
	// UserCanCreateOrganizationPlugin returns whether the user is authorized to create
	// a plugin in an organization.
	UserCanCreateOrganizationPlugin(context.Context, *UserCanCreateOrganizationPluginRequest) (*UserCanCreateOrganizationPluginResponse, error)
	// UserCanCreateOrganizationPlugin returns whether the user is authorized to create
	// a template in an organization.
	UserCanCreateOrganizationTemplate(context.Context, *UserCanCreateOrganizationTemplateRequest) (*UserCanCreateOrganizationTemplateResponse, error)
	// UserCanSeePluginSettings returns whether the user is authorized
	// to see plugin settings.
	UserCanSeePluginSettings(context.Context, *UserCanSeePluginSettingsRequest) (*UserCanSeePluginSettingsResponse, error)
	// UserCanSeeTemplateSettings returns whether the user is authorized
	// to see template settings.
	UserCanSeeTemplateSettings(context.Context, *UserCanSeeTemplateSettingsRequest) (*UserCanSeeTemplateSettingsResponse, error)
}

// UnimplementedAuthzServiceServer should be embedded to have forward compatible implementations.
type UnimplementedAuthzServiceServer struct {
}

func (UnimplementedAuthzServiceServer) UserCanAddUserOrganizationScopes(context.Context, *UserCanAddUserOrganizationScopesRequest) (*UserCanAddUserOrganizationScopesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UserCanAddUserOrganizationScopes not implemented")
}
func (UnimplementedAuthzServiceServer) UserCanRemoveUserOrganizationScopes(context.Context, *UserCanRemoveUserOrganizationScopesRequest) (*UserCanRemoveUserOrganizationScopesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UserCanRemoveUserOrganizationScopes not implemented")
}
func (UnimplementedAuthzServiceServer) UserCanCreateOrganizationRepository(context.Context, *UserCanCreateOrganizationRepositoryRequest) (*UserCanCreateOrganizationRepositoryResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UserCanCreateOrganizationRepository not implemented")
}
func (UnimplementedAuthzServiceServer) UserCanCreateOrganizationTeam(context.Context, *UserCanCreateOrganizationTeamRequest) (*UserCanCreateOrganizationTeamResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UserCanCreateOrganizationTeam not implemented")
}
func (UnimplementedAuthzServiceServer) UserCanListOrganizationTeams(context.Context, *UserCanListOrganizationTeamsRequest) (*UserCanListOrganizationTeamsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UserCanListOrganizationTeams not implemented")
}
func (UnimplementedAuthzServiceServer) UserCanSeeRepositorySettings(context.Context, *UserCanSeeRepositorySettingsRequest) (*UserCanSeeRepositorySettingsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UserCanSeeRepositorySettings not implemented")
}
func (UnimplementedAuthzServiceServer) UserCanSeeOrganizationSettings(context.Context, *UserCanSeeOrganizationSettingsRequest) (*UserCanSeeOrganizationSettingsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UserCanSeeOrganizationSettings not implemented")
}
func (UnimplementedAuthzServiceServer) UserCanReadPlugin(context.Context, *UserCanReadPluginRequest) (*UserCanReadPluginResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UserCanReadPlugin not implemented")
}
func (UnimplementedAuthzServiceServer) UserCanCreatePluginVersion(context.Context, *UserCanCreatePluginVersionRequest) (*UserCanCreatePluginVersionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UserCanCreatePluginVersion not implemented")
}
func (UnimplementedAuthzServiceServer) UserCanCreateTemplateVersion(context.Context, *UserCanCreateTemplateVersionRequest) (*UserCanCreateTemplateVersionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UserCanCreateTemplateVersion not implemented")
}
func (UnimplementedAuthzServiceServer) UserCanCreateOrganizationPlugin(context.Context, *UserCanCreateOrganizationPluginRequest) (*UserCanCreateOrganizationPluginResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UserCanCreateOrganizationPlugin not implemented")
}
func (UnimplementedAuthzServiceServer) UserCanCreateOrganizationTemplate(context.Context, *UserCanCreateOrganizationTemplateRequest) (*UserCanCreateOrganizationTemplateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UserCanCreateOrganizationTemplate not implemented")
}
func (UnimplementedAuthzServiceServer) UserCanSeePluginSettings(context.Context, *UserCanSeePluginSettingsRequest) (*UserCanSeePluginSettingsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UserCanSeePluginSettings not implemented")
}
func (UnimplementedAuthzServiceServer) UserCanSeeTemplateSettings(context.Context, *UserCanSeeTemplateSettingsRequest) (*UserCanSeeTemplateSettingsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UserCanSeeTemplateSettings not implemented")
}

// UnsafeAuthzServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to AuthzServiceServer will
// result in compilation errors.
type UnsafeAuthzServiceServer interface {
	mustEmbedUnimplementedAuthzServiceServer()
}

func RegisterAuthzServiceServer(s grpc.ServiceRegistrar, srv AuthzServiceServer) {
	s.RegisterService(&AuthzService_ServiceDesc, srv)
}

func _AuthzService_UserCanAddUserOrganizationScopes_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UserCanAddUserOrganizationScopesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AuthzServiceServer).UserCanAddUserOrganizationScopes(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.AuthzService/UserCanAddUserOrganizationScopes",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AuthzServiceServer).UserCanAddUserOrganizationScopes(ctx, req.(*UserCanAddUserOrganizationScopesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AuthzService_UserCanRemoveUserOrganizationScopes_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UserCanRemoveUserOrganizationScopesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AuthzServiceServer).UserCanRemoveUserOrganizationScopes(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.AuthzService/UserCanRemoveUserOrganizationScopes",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AuthzServiceServer).UserCanRemoveUserOrganizationScopes(ctx, req.(*UserCanRemoveUserOrganizationScopesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AuthzService_UserCanCreateOrganizationRepository_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UserCanCreateOrganizationRepositoryRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AuthzServiceServer).UserCanCreateOrganizationRepository(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.AuthzService/UserCanCreateOrganizationRepository",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AuthzServiceServer).UserCanCreateOrganizationRepository(ctx, req.(*UserCanCreateOrganizationRepositoryRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AuthzService_UserCanCreateOrganizationTeam_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UserCanCreateOrganizationTeamRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AuthzServiceServer).UserCanCreateOrganizationTeam(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.AuthzService/UserCanCreateOrganizationTeam",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AuthzServiceServer).UserCanCreateOrganizationTeam(ctx, req.(*UserCanCreateOrganizationTeamRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AuthzService_UserCanListOrganizationTeams_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UserCanListOrganizationTeamsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AuthzServiceServer).UserCanListOrganizationTeams(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.AuthzService/UserCanListOrganizationTeams",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AuthzServiceServer).UserCanListOrganizationTeams(ctx, req.(*UserCanListOrganizationTeamsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AuthzService_UserCanSeeRepositorySettings_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UserCanSeeRepositorySettingsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AuthzServiceServer).UserCanSeeRepositorySettings(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.AuthzService/UserCanSeeRepositorySettings",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AuthzServiceServer).UserCanSeeRepositorySettings(ctx, req.(*UserCanSeeRepositorySettingsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AuthzService_UserCanSeeOrganizationSettings_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UserCanSeeOrganizationSettingsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AuthzServiceServer).UserCanSeeOrganizationSettings(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.AuthzService/UserCanSeeOrganizationSettings",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AuthzServiceServer).UserCanSeeOrganizationSettings(ctx, req.(*UserCanSeeOrganizationSettingsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AuthzService_UserCanReadPlugin_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UserCanReadPluginRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AuthzServiceServer).UserCanReadPlugin(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.AuthzService/UserCanReadPlugin",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AuthzServiceServer).UserCanReadPlugin(ctx, req.(*UserCanReadPluginRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AuthzService_UserCanCreatePluginVersion_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UserCanCreatePluginVersionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AuthzServiceServer).UserCanCreatePluginVersion(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.AuthzService/UserCanCreatePluginVersion",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AuthzServiceServer).UserCanCreatePluginVersion(ctx, req.(*UserCanCreatePluginVersionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AuthzService_UserCanCreateTemplateVersion_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UserCanCreateTemplateVersionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AuthzServiceServer).UserCanCreateTemplateVersion(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.AuthzService/UserCanCreateTemplateVersion",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AuthzServiceServer).UserCanCreateTemplateVersion(ctx, req.(*UserCanCreateTemplateVersionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AuthzService_UserCanCreateOrganizationPlugin_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UserCanCreateOrganizationPluginRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AuthzServiceServer).UserCanCreateOrganizationPlugin(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.AuthzService/UserCanCreateOrganizationPlugin",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AuthzServiceServer).UserCanCreateOrganizationPlugin(ctx, req.(*UserCanCreateOrganizationPluginRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AuthzService_UserCanCreateOrganizationTemplate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UserCanCreateOrganizationTemplateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AuthzServiceServer).UserCanCreateOrganizationTemplate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.AuthzService/UserCanCreateOrganizationTemplate",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AuthzServiceServer).UserCanCreateOrganizationTemplate(ctx, req.(*UserCanCreateOrganizationTemplateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AuthzService_UserCanSeePluginSettings_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UserCanSeePluginSettingsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AuthzServiceServer).UserCanSeePluginSettings(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.AuthzService/UserCanSeePluginSettings",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AuthzServiceServer).UserCanSeePluginSettings(ctx, req.(*UserCanSeePluginSettingsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AuthzService_UserCanSeeTemplateSettings_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UserCanSeeTemplateSettingsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AuthzServiceServer).UserCanSeeTemplateSettings(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.AuthzService/UserCanSeeTemplateSettings",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AuthzServiceServer).UserCanSeeTemplateSettings(ctx, req.(*UserCanSeeTemplateSettingsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// AuthzService_ServiceDesc is the grpc.ServiceDesc for AuthzService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var AuthzService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "buf.alpha.registry.v1alpha1.AuthzService",
	HandlerType: (*AuthzServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "UserCanAddUserOrganizationScopes",
			Handler:    _AuthzService_UserCanAddUserOrganizationScopes_Handler,
		},
		{
			MethodName: "UserCanRemoveUserOrganizationScopes",
			Handler:    _AuthzService_UserCanRemoveUserOrganizationScopes_Handler,
		},
		{
			MethodName: "UserCanCreateOrganizationRepository",
			Handler:    _AuthzService_UserCanCreateOrganizationRepository_Handler,
		},
		{
			MethodName: "UserCanCreateOrganizationTeam",
			Handler:    _AuthzService_UserCanCreateOrganizationTeam_Handler,
		},
		{
			MethodName: "UserCanListOrganizationTeams",
			Handler:    _AuthzService_UserCanListOrganizationTeams_Handler,
		},
		{
			MethodName: "UserCanSeeRepositorySettings",
			Handler:    _AuthzService_UserCanSeeRepositorySettings_Handler,
		},
		{
			MethodName: "UserCanSeeOrganizationSettings",
			Handler:    _AuthzService_UserCanSeeOrganizationSettings_Handler,
		},
		{
			MethodName: "UserCanReadPlugin",
			Handler:    _AuthzService_UserCanReadPlugin_Handler,
		},
		{
			MethodName: "UserCanCreatePluginVersion",
			Handler:    _AuthzService_UserCanCreatePluginVersion_Handler,
		},
		{
			MethodName: "UserCanCreateTemplateVersion",
			Handler:    _AuthzService_UserCanCreateTemplateVersion_Handler,
		},
		{
			MethodName: "UserCanCreateOrganizationPlugin",
			Handler:    _AuthzService_UserCanCreateOrganizationPlugin_Handler,
		},
		{
			MethodName: "UserCanCreateOrganizationTemplate",
			Handler:    _AuthzService_UserCanCreateOrganizationTemplate_Handler,
		},
		{
			MethodName: "UserCanSeePluginSettings",
			Handler:    _AuthzService_UserCanSeePluginSettings_Handler,
		},
		{
			MethodName: "UserCanSeeTemplateSettings",
			Handler:    _AuthzService_UserCanSeeTemplateSettings_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "buf/alpha/registry/v1alpha1/authz.proto",
}
