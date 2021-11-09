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
// - protoc             (unknown)
// source: buf/alpha/registry/v1alpha1/download.proto

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

// DownloadServiceClient is the client API for DownloadService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type DownloadServiceClient interface {
	// Download downloads.
	Download(ctx context.Context, in *DownloadRequest, opts ...grpc.CallOption) (*DownloadResponse, error)
}

type downloadServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewDownloadServiceClient(cc grpc.ClientConnInterface) DownloadServiceClient {
	return &downloadServiceClient{cc}
}

func (c *downloadServiceClient) Download(ctx context.Context, in *DownloadRequest, opts ...grpc.CallOption) (*DownloadResponse, error) {
	out := new(DownloadResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.DownloadService/Download", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DownloadServiceServer is the server API for DownloadService service.
// All implementations should embed UnimplementedDownloadServiceServer
// for forward compatibility
type DownloadServiceServer interface {
	// Download downloads.
	Download(context.Context, *DownloadRequest) (*DownloadResponse, error)
}

// UnimplementedDownloadServiceServer should be embedded to have forward compatible implementations.
type UnimplementedDownloadServiceServer struct {
}

func (UnimplementedDownloadServiceServer) Download(context.Context, *DownloadRequest) (*DownloadResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Download not implemented")
}

// UnsafeDownloadServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to DownloadServiceServer will
// result in compilation errors.
type UnsafeDownloadServiceServer interface {
	mustEmbedUnimplementedDownloadServiceServer()
}

func RegisterDownloadServiceServer(s grpc.ServiceRegistrar, srv DownloadServiceServer) {
	s.RegisterService(&DownloadService_ServiceDesc, srv)
}

func _DownloadService_Download_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DownloadRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DownloadServiceServer).Download(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.DownloadService/Download",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DownloadServiceServer).Download(ctx, req.(*DownloadRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// DownloadService_ServiceDesc is the grpc.ServiceDesc for DownloadService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var DownloadService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "buf.alpha.registry.v1alpha1.DownloadService",
	HandlerType: (*DownloadServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Download",
			Handler:    _DownloadService_Download_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "buf/alpha/registry/v1alpha1/download.proto",
}
