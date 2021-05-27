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
// - protoc             v3.17.1
// source: buf/alpha/registry/v1alpha1/repository_branch.proto

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

// RepositoryBranchServiceClient is the client API for RepositoryBranchService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type RepositoryBranchServiceClient interface {
	// CreateRepositoryBranch creates a new repository branch.
	CreateRepositoryBranch(ctx context.Context, in *CreateRepositoryBranchRequest, opts ...grpc.CallOption) (*CreateRepositoryBranchResponse, error)
	// ListRepositoryBranches lists the repository branches associated with a Repository.
	ListRepositoryBranches(ctx context.Context, in *ListRepositoryBranchesRequest, opts ...grpc.CallOption) (*ListRepositoryBranchesResponse, error)
}

type repositoryBranchServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewRepositoryBranchServiceClient(cc grpc.ClientConnInterface) RepositoryBranchServiceClient {
	return &repositoryBranchServiceClient{cc}
}

func (c *repositoryBranchServiceClient) CreateRepositoryBranch(ctx context.Context, in *CreateRepositoryBranchRequest, opts ...grpc.CallOption) (*CreateRepositoryBranchResponse, error) {
	out := new(CreateRepositoryBranchResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.RepositoryBranchService/CreateRepositoryBranch", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *repositoryBranchServiceClient) ListRepositoryBranches(ctx context.Context, in *ListRepositoryBranchesRequest, opts ...grpc.CallOption) (*ListRepositoryBranchesResponse, error) {
	out := new(ListRepositoryBranchesResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.RepositoryBranchService/ListRepositoryBranches", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RepositoryBranchServiceServer is the server API for RepositoryBranchService service.
// All implementations should embed UnimplementedRepositoryBranchServiceServer
// for forward compatibility
type RepositoryBranchServiceServer interface {
	// CreateRepositoryBranch creates a new repository branch.
	CreateRepositoryBranch(context.Context, *CreateRepositoryBranchRequest) (*CreateRepositoryBranchResponse, error)
	// ListRepositoryBranches lists the repository branches associated with a Repository.
	ListRepositoryBranches(context.Context, *ListRepositoryBranchesRequest) (*ListRepositoryBranchesResponse, error)
}

// UnimplementedRepositoryBranchServiceServer should be embedded to have forward compatible implementations.
type UnimplementedRepositoryBranchServiceServer struct {
}

func (UnimplementedRepositoryBranchServiceServer) CreateRepositoryBranch(context.Context, *CreateRepositoryBranchRequest) (*CreateRepositoryBranchResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateRepositoryBranch not implemented")
}
func (UnimplementedRepositoryBranchServiceServer) ListRepositoryBranches(context.Context, *ListRepositoryBranchesRequest) (*ListRepositoryBranchesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListRepositoryBranches not implemented")
}

// UnsafeRepositoryBranchServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to RepositoryBranchServiceServer will
// result in compilation errors.
type UnsafeRepositoryBranchServiceServer interface {
	mustEmbedUnimplementedRepositoryBranchServiceServer()
}

func RegisterRepositoryBranchServiceServer(s grpc.ServiceRegistrar, srv RepositoryBranchServiceServer) {
	s.RegisterService(&RepositoryBranchService_ServiceDesc, srv)
}

func _RepositoryBranchService_CreateRepositoryBranch_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateRepositoryBranchRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RepositoryBranchServiceServer).CreateRepositoryBranch(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.RepositoryBranchService/CreateRepositoryBranch",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RepositoryBranchServiceServer).CreateRepositoryBranch(ctx, req.(*CreateRepositoryBranchRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RepositoryBranchService_ListRepositoryBranches_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListRepositoryBranchesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RepositoryBranchServiceServer).ListRepositoryBranches(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.RepositoryBranchService/ListRepositoryBranches",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RepositoryBranchServiceServer).ListRepositoryBranches(ctx, req.(*ListRepositoryBranchesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// RepositoryBranchService_ServiceDesc is the grpc.ServiceDesc for RepositoryBranchService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var RepositoryBranchService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "buf.alpha.registry.v1alpha1.RepositoryBranchService",
	HandlerType: (*RepositoryBranchServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CreateRepositoryBranch",
			Handler:    _RepositoryBranchService_CreateRepositoryBranch_Handler,
		},
		{
			MethodName: "ListRepositoryBranches",
			Handler:    _RepositoryBranchService_ListRepositoryBranches_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "buf/alpha/registry/v1alpha1/repository_branch.proto",
}
