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
// source: buf/alpha/registry/v1alpha1/studio.proto

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

// StudioServiceClient is the client API for StudioService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type StudioServiceClient interface {
	// ListPresetAgents returns a list of preset agents in the server.
	ListPresetAgents(ctx context.Context, in *ListPresetAgentsRequest, opts ...grpc.CallOption) (*ListPresetAgentsResponse, error)
	// SetPresetAgents set the list of preset agents in the server.
	SetPresetAgents(ctx context.Context, in *SetPresetAgentsRequest, opts ...grpc.CallOption) (*SetPresetAgentsResponse, error)
}

type studioServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewStudioServiceClient(cc grpc.ClientConnInterface) StudioServiceClient {
	return &studioServiceClient{cc}
}

func (c *studioServiceClient) ListPresetAgents(ctx context.Context, in *ListPresetAgentsRequest, opts ...grpc.CallOption) (*ListPresetAgentsResponse, error) {
	out := new(ListPresetAgentsResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.StudioService/ListPresetAgents", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *studioServiceClient) SetPresetAgents(ctx context.Context, in *SetPresetAgentsRequest, opts ...grpc.CallOption) (*SetPresetAgentsResponse, error) {
	out := new(SetPresetAgentsResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.StudioService/SetPresetAgents", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// StudioServiceServer is the server API for StudioService service.
// All implementations should embed UnimplementedStudioServiceServer
// for forward compatibility
type StudioServiceServer interface {
	// ListPresetAgents returns a list of preset agents in the server.
	ListPresetAgents(context.Context, *ListPresetAgentsRequest) (*ListPresetAgentsResponse, error)
	// SetPresetAgents set the list of preset agents in the server.
	SetPresetAgents(context.Context, *SetPresetAgentsRequest) (*SetPresetAgentsResponse, error)
}

// UnimplementedStudioServiceServer should be embedded to have forward compatible implementations.
type UnimplementedStudioServiceServer struct {
}

func (UnimplementedStudioServiceServer) ListPresetAgents(context.Context, *ListPresetAgentsRequest) (*ListPresetAgentsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListPresetAgents not implemented")
}
func (UnimplementedStudioServiceServer) SetPresetAgents(context.Context, *SetPresetAgentsRequest) (*SetPresetAgentsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetPresetAgents not implemented")
}

// UnsafeStudioServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to StudioServiceServer will
// result in compilation errors.
type UnsafeStudioServiceServer interface {
	mustEmbedUnimplementedStudioServiceServer()
}

func RegisterStudioServiceServer(s grpc.ServiceRegistrar, srv StudioServiceServer) {
	s.RegisterService(&StudioService_ServiceDesc, srv)
}

func _StudioService_ListPresetAgents_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListPresetAgentsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(StudioServiceServer).ListPresetAgents(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.StudioService/ListPresetAgents",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(StudioServiceServer).ListPresetAgents(ctx, req.(*ListPresetAgentsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _StudioService_SetPresetAgents_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetPresetAgentsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(StudioServiceServer).SetPresetAgents(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.StudioService/SetPresetAgents",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(StudioServiceServer).SetPresetAgents(ctx, req.(*SetPresetAgentsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// StudioService_ServiceDesc is the grpc.ServiceDesc for StudioService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var StudioService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "buf.alpha.registry.v1alpha1.StudioService",
	HandlerType: (*StudioServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ListPresetAgents",
			Handler:    _StudioService_ListPresetAgents_Handler,
		},
		{
			MethodName: "SetPresetAgents",
			Handler:    _StudioService_SetPresetAgents_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "buf/alpha/registry/v1alpha1/studio.proto",
}
