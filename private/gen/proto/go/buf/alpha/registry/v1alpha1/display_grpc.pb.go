// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             (unknown)
// source: buf/alpha/registry/v1alpha1/display.proto

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

// DisplayServiceClient is the client API for DisplayService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type DisplayServiceClient interface {
	// DisplayOrganizationElements returns which organization elements should be displayed to the user.
	DisplayOrganizationElements(ctx context.Context, in *DisplayOrganizationElementsRequest, opts ...grpc.CallOption) (*DisplayOrganizationElementsResponse, error)
	// DisplayRepositoryElements returns which repository elements should be displayed to the user.
	DisplayRepositoryElements(ctx context.Context, in *DisplayRepositoryElementsRequest, opts ...grpc.CallOption) (*DisplayRepositoryElementsResponse, error)
	// DisplayPluginElements returns which plugin elements should be displayed to the user.
	DisplayPluginElements(ctx context.Context, in *DisplayPluginElementsRequest, opts ...grpc.CallOption) (*DisplayPluginElementsResponse, error)
	// DisplayTemplateElements returns which template elements should be displayed to the user.
	DisplayTemplateElements(ctx context.Context, in *DisplayTemplateElementsRequest, opts ...grpc.CallOption) (*DisplayTemplateElementsResponse, error)
	// DisplayUserElements returns which user elements should be displayed to the user.
	DisplayUserElements(ctx context.Context, in *DisplayUserElementsRequest, opts ...grpc.CallOption) (*DisplayUserElementsResponse, error)
	// DisplayServerElements returns which server elements should be displayed to the user.
	DisplayServerElements(ctx context.Context, in *DisplayServerElementsRequest, opts ...grpc.CallOption) (*DisplayServerElementsResponse, error)
	// ListManageableRepositoryRoles returns which roles should be displayed
	// to the user when they are managing contributors on the repository.
	ListManageableRepositoryRoles(ctx context.Context, in *ListManageableRepositoryRolesRequest, opts ...grpc.CallOption) (*ListManageableRepositoryRolesResponse, error)
	// ListManageableUserRepositoryRoles returns which roles should be displayed
	// to the user when they are managing a specific contributor on the repository.
	ListManageableUserRepositoryRoles(ctx context.Context, in *ListManageableUserRepositoryRolesRequest, opts ...grpc.CallOption) (*ListManageableUserRepositoryRolesResponse, error)
	// ListManageablePluginRoles returns which roles should be displayed
	// to the user when they are managing contributors on the plugin.
	ListManageablePluginRoles(ctx context.Context, in *ListManageablePluginRolesRequest, opts ...grpc.CallOption) (*ListManageablePluginRolesResponse, error)
	// ListManageableUserPluginRoles returns which roles should be displayed
	// to the user when they are managing a specific contributor on the plugin.
	ListManageableUserPluginRoles(ctx context.Context, in *ListManageableUserPluginRolesRequest, opts ...grpc.CallOption) (*ListManageableUserPluginRolesResponse, error)
	// ListManageableTemplateRoles returns which roles should be displayed
	// to the user when they are managing contributors on the template.
	ListManageableTemplateRoles(ctx context.Context, in *ListManageableTemplateRolesRequest, opts ...grpc.CallOption) (*ListManageableTemplateRolesResponse, error)
	// ListManageableUserTemplateRoles returns which roles should be displayed
	// to the user when they are managing a specific contributor on the template.
	ListManageableUserTemplateRoles(ctx context.Context, in *ListManageableUserTemplateRolesRequest, opts ...grpc.CallOption) (*ListManageableUserTemplateRolesResponse, error)
}

type displayServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewDisplayServiceClient(cc grpc.ClientConnInterface) DisplayServiceClient {
	return &displayServiceClient{cc}
}

func (c *displayServiceClient) DisplayOrganizationElements(ctx context.Context, in *DisplayOrganizationElementsRequest, opts ...grpc.CallOption) (*DisplayOrganizationElementsResponse, error) {
	out := new(DisplayOrganizationElementsResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.DisplayService/DisplayOrganizationElements", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *displayServiceClient) DisplayRepositoryElements(ctx context.Context, in *DisplayRepositoryElementsRequest, opts ...grpc.CallOption) (*DisplayRepositoryElementsResponse, error) {
	out := new(DisplayRepositoryElementsResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.DisplayService/DisplayRepositoryElements", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *displayServiceClient) DisplayPluginElements(ctx context.Context, in *DisplayPluginElementsRequest, opts ...grpc.CallOption) (*DisplayPluginElementsResponse, error) {
	out := new(DisplayPluginElementsResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.DisplayService/DisplayPluginElements", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *displayServiceClient) DisplayTemplateElements(ctx context.Context, in *DisplayTemplateElementsRequest, opts ...grpc.CallOption) (*DisplayTemplateElementsResponse, error) {
	out := new(DisplayTemplateElementsResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.DisplayService/DisplayTemplateElements", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *displayServiceClient) DisplayUserElements(ctx context.Context, in *DisplayUserElementsRequest, opts ...grpc.CallOption) (*DisplayUserElementsResponse, error) {
	out := new(DisplayUserElementsResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.DisplayService/DisplayUserElements", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *displayServiceClient) DisplayServerElements(ctx context.Context, in *DisplayServerElementsRequest, opts ...grpc.CallOption) (*DisplayServerElementsResponse, error) {
	out := new(DisplayServerElementsResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.DisplayService/DisplayServerElements", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *displayServiceClient) ListManageableRepositoryRoles(ctx context.Context, in *ListManageableRepositoryRolesRequest, opts ...grpc.CallOption) (*ListManageableRepositoryRolesResponse, error) {
	out := new(ListManageableRepositoryRolesResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.DisplayService/ListManageableRepositoryRoles", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *displayServiceClient) ListManageableUserRepositoryRoles(ctx context.Context, in *ListManageableUserRepositoryRolesRequest, opts ...grpc.CallOption) (*ListManageableUserRepositoryRolesResponse, error) {
	out := new(ListManageableUserRepositoryRolesResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.DisplayService/ListManageableUserRepositoryRoles", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *displayServiceClient) ListManageablePluginRoles(ctx context.Context, in *ListManageablePluginRolesRequest, opts ...grpc.CallOption) (*ListManageablePluginRolesResponse, error) {
	out := new(ListManageablePluginRolesResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.DisplayService/ListManageablePluginRoles", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *displayServiceClient) ListManageableUserPluginRoles(ctx context.Context, in *ListManageableUserPluginRolesRequest, opts ...grpc.CallOption) (*ListManageableUserPluginRolesResponse, error) {
	out := new(ListManageableUserPluginRolesResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.DisplayService/ListManageableUserPluginRoles", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *displayServiceClient) ListManageableTemplateRoles(ctx context.Context, in *ListManageableTemplateRolesRequest, opts ...grpc.CallOption) (*ListManageableTemplateRolesResponse, error) {
	out := new(ListManageableTemplateRolesResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.DisplayService/ListManageableTemplateRoles", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *displayServiceClient) ListManageableUserTemplateRoles(ctx context.Context, in *ListManageableUserTemplateRolesRequest, opts ...grpc.CallOption) (*ListManageableUserTemplateRolesResponse, error) {
	out := new(ListManageableUserTemplateRolesResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.DisplayService/ListManageableUserTemplateRoles", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DisplayServiceServer is the server API for DisplayService service.
// All implementations should embed UnimplementedDisplayServiceServer
// for forward compatibility
type DisplayServiceServer interface {
	// DisplayOrganizationElements returns which organization elements should be displayed to the user.
	DisplayOrganizationElements(context.Context, *DisplayOrganizationElementsRequest) (*DisplayOrganizationElementsResponse, error)
	// DisplayRepositoryElements returns which repository elements should be displayed to the user.
	DisplayRepositoryElements(context.Context, *DisplayRepositoryElementsRequest) (*DisplayRepositoryElementsResponse, error)
	// DisplayPluginElements returns which plugin elements should be displayed to the user.
	DisplayPluginElements(context.Context, *DisplayPluginElementsRequest) (*DisplayPluginElementsResponse, error)
	// DisplayTemplateElements returns which template elements should be displayed to the user.
	DisplayTemplateElements(context.Context, *DisplayTemplateElementsRequest) (*DisplayTemplateElementsResponse, error)
	// DisplayUserElements returns which user elements should be displayed to the user.
	DisplayUserElements(context.Context, *DisplayUserElementsRequest) (*DisplayUserElementsResponse, error)
	// DisplayServerElements returns which server elements should be displayed to the user.
	DisplayServerElements(context.Context, *DisplayServerElementsRequest) (*DisplayServerElementsResponse, error)
	// ListManageableRepositoryRoles returns which roles should be displayed
	// to the user when they are managing contributors on the repository.
	ListManageableRepositoryRoles(context.Context, *ListManageableRepositoryRolesRequest) (*ListManageableRepositoryRolesResponse, error)
	// ListManageableUserRepositoryRoles returns which roles should be displayed
	// to the user when they are managing a specific contributor on the repository.
	ListManageableUserRepositoryRoles(context.Context, *ListManageableUserRepositoryRolesRequest) (*ListManageableUserRepositoryRolesResponse, error)
	// ListManageablePluginRoles returns which roles should be displayed
	// to the user when they are managing contributors on the plugin.
	ListManageablePluginRoles(context.Context, *ListManageablePluginRolesRequest) (*ListManageablePluginRolesResponse, error)
	// ListManageableUserPluginRoles returns which roles should be displayed
	// to the user when they are managing a specific contributor on the plugin.
	ListManageableUserPluginRoles(context.Context, *ListManageableUserPluginRolesRequest) (*ListManageableUserPluginRolesResponse, error)
	// ListManageableTemplateRoles returns which roles should be displayed
	// to the user when they are managing contributors on the template.
	ListManageableTemplateRoles(context.Context, *ListManageableTemplateRolesRequest) (*ListManageableTemplateRolesResponse, error)
	// ListManageableUserTemplateRoles returns which roles should be displayed
	// to the user when they are managing a specific contributor on the template.
	ListManageableUserTemplateRoles(context.Context, *ListManageableUserTemplateRolesRequest) (*ListManageableUserTemplateRolesResponse, error)
}

// UnimplementedDisplayServiceServer should be embedded to have forward compatible implementations.
type UnimplementedDisplayServiceServer struct {
}

func (UnimplementedDisplayServiceServer) DisplayOrganizationElements(context.Context, *DisplayOrganizationElementsRequest) (*DisplayOrganizationElementsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DisplayOrganizationElements not implemented")
}
func (UnimplementedDisplayServiceServer) DisplayRepositoryElements(context.Context, *DisplayRepositoryElementsRequest) (*DisplayRepositoryElementsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DisplayRepositoryElements not implemented")
}
func (UnimplementedDisplayServiceServer) DisplayPluginElements(context.Context, *DisplayPluginElementsRequest) (*DisplayPluginElementsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DisplayPluginElements not implemented")
}
func (UnimplementedDisplayServiceServer) DisplayTemplateElements(context.Context, *DisplayTemplateElementsRequest) (*DisplayTemplateElementsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DisplayTemplateElements not implemented")
}
func (UnimplementedDisplayServiceServer) DisplayUserElements(context.Context, *DisplayUserElementsRequest) (*DisplayUserElementsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DisplayUserElements not implemented")
}
func (UnimplementedDisplayServiceServer) DisplayServerElements(context.Context, *DisplayServerElementsRequest) (*DisplayServerElementsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DisplayServerElements not implemented")
}
func (UnimplementedDisplayServiceServer) ListManageableRepositoryRoles(context.Context, *ListManageableRepositoryRolesRequest) (*ListManageableRepositoryRolesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListManageableRepositoryRoles not implemented")
}
func (UnimplementedDisplayServiceServer) ListManageableUserRepositoryRoles(context.Context, *ListManageableUserRepositoryRolesRequest) (*ListManageableUserRepositoryRolesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListManageableUserRepositoryRoles not implemented")
}
func (UnimplementedDisplayServiceServer) ListManageablePluginRoles(context.Context, *ListManageablePluginRolesRequest) (*ListManageablePluginRolesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListManageablePluginRoles not implemented")
}
func (UnimplementedDisplayServiceServer) ListManageableUserPluginRoles(context.Context, *ListManageableUserPluginRolesRequest) (*ListManageableUserPluginRolesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListManageableUserPluginRoles not implemented")
}
func (UnimplementedDisplayServiceServer) ListManageableTemplateRoles(context.Context, *ListManageableTemplateRolesRequest) (*ListManageableTemplateRolesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListManageableTemplateRoles not implemented")
}
func (UnimplementedDisplayServiceServer) ListManageableUserTemplateRoles(context.Context, *ListManageableUserTemplateRolesRequest) (*ListManageableUserTemplateRolesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListManageableUserTemplateRoles not implemented")
}

// UnsafeDisplayServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to DisplayServiceServer will
// result in compilation errors.
type UnsafeDisplayServiceServer interface {
	mustEmbedUnimplementedDisplayServiceServer()
}

func RegisterDisplayServiceServer(s grpc.ServiceRegistrar, srv DisplayServiceServer) {
	s.RegisterService(&DisplayService_ServiceDesc, srv)
}

func _DisplayService_DisplayOrganizationElements_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DisplayOrganizationElementsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DisplayServiceServer).DisplayOrganizationElements(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.DisplayService/DisplayOrganizationElements",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DisplayServiceServer).DisplayOrganizationElements(ctx, req.(*DisplayOrganizationElementsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DisplayService_DisplayRepositoryElements_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DisplayRepositoryElementsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DisplayServiceServer).DisplayRepositoryElements(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.DisplayService/DisplayRepositoryElements",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DisplayServiceServer).DisplayRepositoryElements(ctx, req.(*DisplayRepositoryElementsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DisplayService_DisplayPluginElements_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DisplayPluginElementsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DisplayServiceServer).DisplayPluginElements(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.DisplayService/DisplayPluginElements",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DisplayServiceServer).DisplayPluginElements(ctx, req.(*DisplayPluginElementsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DisplayService_DisplayTemplateElements_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DisplayTemplateElementsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DisplayServiceServer).DisplayTemplateElements(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.DisplayService/DisplayTemplateElements",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DisplayServiceServer).DisplayTemplateElements(ctx, req.(*DisplayTemplateElementsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DisplayService_DisplayUserElements_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DisplayUserElementsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DisplayServiceServer).DisplayUserElements(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.DisplayService/DisplayUserElements",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DisplayServiceServer).DisplayUserElements(ctx, req.(*DisplayUserElementsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DisplayService_DisplayServerElements_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DisplayServerElementsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DisplayServiceServer).DisplayServerElements(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.DisplayService/DisplayServerElements",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DisplayServiceServer).DisplayServerElements(ctx, req.(*DisplayServerElementsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DisplayService_ListManageableRepositoryRoles_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListManageableRepositoryRolesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DisplayServiceServer).ListManageableRepositoryRoles(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.DisplayService/ListManageableRepositoryRoles",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DisplayServiceServer).ListManageableRepositoryRoles(ctx, req.(*ListManageableRepositoryRolesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DisplayService_ListManageableUserRepositoryRoles_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListManageableUserRepositoryRolesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DisplayServiceServer).ListManageableUserRepositoryRoles(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.DisplayService/ListManageableUserRepositoryRoles",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DisplayServiceServer).ListManageableUserRepositoryRoles(ctx, req.(*ListManageableUserRepositoryRolesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DisplayService_ListManageablePluginRoles_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListManageablePluginRolesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DisplayServiceServer).ListManageablePluginRoles(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.DisplayService/ListManageablePluginRoles",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DisplayServiceServer).ListManageablePluginRoles(ctx, req.(*ListManageablePluginRolesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DisplayService_ListManageableUserPluginRoles_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListManageableUserPluginRolesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DisplayServiceServer).ListManageableUserPluginRoles(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.DisplayService/ListManageableUserPluginRoles",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DisplayServiceServer).ListManageableUserPluginRoles(ctx, req.(*ListManageableUserPluginRolesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DisplayService_ListManageableTemplateRoles_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListManageableTemplateRolesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DisplayServiceServer).ListManageableTemplateRoles(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.DisplayService/ListManageableTemplateRoles",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DisplayServiceServer).ListManageableTemplateRoles(ctx, req.(*ListManageableTemplateRolesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DisplayService_ListManageableUserTemplateRoles_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListManageableUserTemplateRolesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DisplayServiceServer).ListManageableUserTemplateRoles(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.DisplayService/ListManageableUserTemplateRoles",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DisplayServiceServer).ListManageableUserTemplateRoles(ctx, req.(*ListManageableUserTemplateRolesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// DisplayService_ServiceDesc is the grpc.ServiceDesc for DisplayService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var DisplayService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "buf.alpha.registry.v1alpha1.DisplayService",
	HandlerType: (*DisplayServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "DisplayOrganizationElements",
			Handler:    _DisplayService_DisplayOrganizationElements_Handler,
		},
		{
			MethodName: "DisplayRepositoryElements",
			Handler:    _DisplayService_DisplayRepositoryElements_Handler,
		},
		{
			MethodName: "DisplayPluginElements",
			Handler:    _DisplayService_DisplayPluginElements_Handler,
		},
		{
			MethodName: "DisplayTemplateElements",
			Handler:    _DisplayService_DisplayTemplateElements_Handler,
		},
		{
			MethodName: "DisplayUserElements",
			Handler:    _DisplayService_DisplayUserElements_Handler,
		},
		{
			MethodName: "DisplayServerElements",
			Handler:    _DisplayService_DisplayServerElements_Handler,
		},
		{
			MethodName: "ListManageableRepositoryRoles",
			Handler:    _DisplayService_ListManageableRepositoryRoles_Handler,
		},
		{
			MethodName: "ListManageableUserRepositoryRoles",
			Handler:    _DisplayService_ListManageableUserRepositoryRoles_Handler,
		},
		{
			MethodName: "ListManageablePluginRoles",
			Handler:    _DisplayService_ListManageablePluginRoles_Handler,
		},
		{
			MethodName: "ListManageableUserPluginRoles",
			Handler:    _DisplayService_ListManageableUserPluginRoles_Handler,
		},
		{
			MethodName: "ListManageableTemplateRoles",
			Handler:    _DisplayService_ListManageableTemplateRoles_Handler,
		},
		{
			MethodName: "ListManageableUserTemplateRoles",
			Handler:    _DisplayService_ListManageableUserTemplateRoles_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "buf/alpha/registry/v1alpha1/display.proto",
}
