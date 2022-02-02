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
// - protoc-gen-go-grpc v1.1.0
// - protoc             (unknown)
// source: buf/alpha/registry/v1alpha1/repository_track.proto

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

// RepositoryTrackServiceClient is the client API for RepositoryTrackService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type RepositoryTrackServiceClient interface {
	// CreateRepositoryTrack creates a new repository track.
	CreateRepositoryTrack(ctx context.Context, in *CreateRepositoryTrackRequest, opts ...grpc.CallOption) (*CreateRepositoryTrackResponse, error)
	// ListRepositoryTracks lists the repository tracks associated with a repository.
	ListRepositoryTracks(ctx context.Context, in *ListRepositoryTracksRequest, opts ...grpc.CallOption) (*ListRepositoryTracksResponse, error)
	// DeleteRepositoryTrackByName deletes a repository track by name.
	DeleteRepositoryTrackByName(ctx context.Context, in *DeleteRepositoryTrackByNameRequest, opts ...grpc.CallOption) (*DeleteRepositoryTrackByNameResponse, error)
	// GetRepositoryTrackByName gets a repository track by name.
	GetRepositoryTrackByName(ctx context.Context, in *GetRepositoryTrackByNameRequest, opts ...grpc.CallOption) (*GetRepositoryTrackByNameResponse, error)
}

type repositoryTrackServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewRepositoryTrackServiceClient(cc grpc.ClientConnInterface) RepositoryTrackServiceClient {
	return &repositoryTrackServiceClient{cc}
}

func (c *repositoryTrackServiceClient) CreateRepositoryTrack(ctx context.Context, in *CreateRepositoryTrackRequest, opts ...grpc.CallOption) (*CreateRepositoryTrackResponse, error) {
	out := new(CreateRepositoryTrackResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.RepositoryTrackService/CreateRepositoryTrack", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *repositoryTrackServiceClient) ListRepositoryTracks(ctx context.Context, in *ListRepositoryTracksRequest, opts ...grpc.CallOption) (*ListRepositoryTracksResponse, error) {
	out := new(ListRepositoryTracksResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.RepositoryTrackService/ListRepositoryTracks", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *repositoryTrackServiceClient) DeleteRepositoryTrackByName(ctx context.Context, in *DeleteRepositoryTrackByNameRequest, opts ...grpc.CallOption) (*DeleteRepositoryTrackByNameResponse, error) {
	out := new(DeleteRepositoryTrackByNameResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.RepositoryTrackService/DeleteRepositoryTrackByName", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *repositoryTrackServiceClient) GetRepositoryTrackByName(ctx context.Context, in *GetRepositoryTrackByNameRequest, opts ...grpc.CallOption) (*GetRepositoryTrackByNameResponse, error) {
	out := new(GetRepositoryTrackByNameResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.RepositoryTrackService/GetRepositoryTrackByName", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RepositoryTrackServiceServer is the server API for RepositoryTrackService service.
// All implementations should embed UnimplementedRepositoryTrackServiceServer
// for forward compatibility
type RepositoryTrackServiceServer interface {
	// CreateRepositoryTrack creates a new repository track.
	CreateRepositoryTrack(context.Context, *CreateRepositoryTrackRequest) (*CreateRepositoryTrackResponse, error)
	// ListRepositoryTracks lists the repository tracks associated with a repository.
	ListRepositoryTracks(context.Context, *ListRepositoryTracksRequest) (*ListRepositoryTracksResponse, error)
	// DeleteRepositoryTrackByName deletes a repository track by name.
	DeleteRepositoryTrackByName(context.Context, *DeleteRepositoryTrackByNameRequest) (*DeleteRepositoryTrackByNameResponse, error)
	// GetRepositoryTrackByName gets a repository track by name.
	GetRepositoryTrackByName(context.Context, *GetRepositoryTrackByNameRequest) (*GetRepositoryTrackByNameResponse, error)
}

// UnimplementedRepositoryTrackServiceServer should be embedded to have forward compatible implementations.
type UnimplementedRepositoryTrackServiceServer struct {
}

func (UnimplementedRepositoryTrackServiceServer) CreateRepositoryTrack(context.Context, *CreateRepositoryTrackRequest) (*CreateRepositoryTrackResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateRepositoryTrack not implemented")
}
func (UnimplementedRepositoryTrackServiceServer) ListRepositoryTracks(context.Context, *ListRepositoryTracksRequest) (*ListRepositoryTracksResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListRepositoryTracks not implemented")
}
func (UnimplementedRepositoryTrackServiceServer) DeleteRepositoryTrackByName(context.Context, *DeleteRepositoryTrackByNameRequest) (*DeleteRepositoryTrackByNameResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteRepositoryTrackByName not implemented")
}
func (UnimplementedRepositoryTrackServiceServer) GetRepositoryTrackByName(context.Context, *GetRepositoryTrackByNameRequest) (*GetRepositoryTrackByNameResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetRepositoryTrackByName not implemented")
}

// UnsafeRepositoryTrackServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to RepositoryTrackServiceServer will
// result in compilation errors.
type UnsafeRepositoryTrackServiceServer interface {
	mustEmbedUnimplementedRepositoryTrackServiceServer()
}

func RegisterRepositoryTrackServiceServer(s grpc.ServiceRegistrar, srv RepositoryTrackServiceServer) {
	s.RegisterService(&RepositoryTrackService_ServiceDesc, srv)
}

func _RepositoryTrackService_CreateRepositoryTrack_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateRepositoryTrackRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RepositoryTrackServiceServer).CreateRepositoryTrack(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.RepositoryTrackService/CreateRepositoryTrack",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RepositoryTrackServiceServer).CreateRepositoryTrack(ctx, req.(*CreateRepositoryTrackRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RepositoryTrackService_ListRepositoryTracks_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListRepositoryTracksRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RepositoryTrackServiceServer).ListRepositoryTracks(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.RepositoryTrackService/ListRepositoryTracks",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RepositoryTrackServiceServer).ListRepositoryTracks(ctx, req.(*ListRepositoryTracksRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RepositoryTrackService_DeleteRepositoryTrackByName_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteRepositoryTrackByNameRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RepositoryTrackServiceServer).DeleteRepositoryTrackByName(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.RepositoryTrackService/DeleteRepositoryTrackByName",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RepositoryTrackServiceServer).DeleteRepositoryTrackByName(ctx, req.(*DeleteRepositoryTrackByNameRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RepositoryTrackService_GetRepositoryTrackByName_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetRepositoryTrackByNameRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RepositoryTrackServiceServer).GetRepositoryTrackByName(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.RepositoryTrackService/GetRepositoryTrackByName",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RepositoryTrackServiceServer).GetRepositoryTrackByName(ctx, req.(*GetRepositoryTrackByNameRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// RepositoryTrackService_ServiceDesc is the grpc.ServiceDesc for RepositoryTrackService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var RepositoryTrackService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "buf.alpha.registry.v1alpha1.RepositoryTrackService",
	HandlerType: (*RepositoryTrackServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CreateRepositoryTrack",
			Handler:    _RepositoryTrackService_CreateRepositoryTrack_Handler,
		},
		{
			MethodName: "ListRepositoryTracks",
			Handler:    _RepositoryTrackService_ListRepositoryTracks_Handler,
		},
		{
			MethodName: "DeleteRepositoryTrackByName",
			Handler:    _RepositoryTrackService_DeleteRepositoryTrackByName_Handler,
		},
		{
			MethodName: "GetRepositoryTrackByName",
			Handler:    _RepositoryTrackService_GetRepositoryTrackByName_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "buf/alpha/registry/v1alpha1/repository_track.proto",
}
