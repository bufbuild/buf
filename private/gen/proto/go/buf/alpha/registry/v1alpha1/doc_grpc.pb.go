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
// - protoc             v3.18.1
// source: buf/alpha/registry/v1alpha1/doc.proto

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

// DocServiceClient is the client API for DocService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type DocServiceClient interface {
	// GetSourceDirectoryInfo retrieves the directory and file structure for the
	// given owner, repository and reference.
	//
	// The purpose of this is to get a representation of the file tree for a given
	// module to enable exploring the module by navigating through its contents.
	GetSourceDirectoryInfo(ctx context.Context, in *GetSourceDirectoryInfoRequest, opts ...grpc.CallOption) (*GetSourceDirectoryInfoResponse, error)
	// GetSourceFile retrieves the source contents for the given owner, repository,
	// reference, and path.
	GetSourceFile(ctx context.Context, in *GetSourceFileRequest, opts ...grpc.CallOption) (*GetSourceFileResponse, error)
	// GetModulePackages retrieves the list of packages for the module based on the given
	// owner, repository, and reference.
	GetModulePackages(ctx context.Context, in *GetModulePackagesRequest, opts ...grpc.CallOption) (*GetModulePackagesResponse, error)
	// GetModuleDocumentation retrieves the documentation for module based on the given
	// owner, repository, and reference.
	GetModuleDocumentation(ctx context.Context, in *GetModuleDocumentationRequest, opts ...grpc.CallOption) (*GetModuleDocumentationResponse, error)
	// GetPackageDocumentation retrieves a a slice of documentation structures
	// for the given owner, repository, reference, and package name.
	GetPackageDocumentation(ctx context.Context, in *GetPackageDocumentationRequest, opts ...grpc.CallOption) (*GetPackageDocumentationResponse, error)
}

type docServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewDocServiceClient(cc grpc.ClientConnInterface) DocServiceClient {
	return &docServiceClient{cc}
}

func (c *docServiceClient) GetSourceDirectoryInfo(ctx context.Context, in *GetSourceDirectoryInfoRequest, opts ...grpc.CallOption) (*GetSourceDirectoryInfoResponse, error) {
	out := new(GetSourceDirectoryInfoResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.DocService/GetSourceDirectoryInfo", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *docServiceClient) GetSourceFile(ctx context.Context, in *GetSourceFileRequest, opts ...grpc.CallOption) (*GetSourceFileResponse, error) {
	out := new(GetSourceFileResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.DocService/GetSourceFile", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *docServiceClient) GetModulePackages(ctx context.Context, in *GetModulePackagesRequest, opts ...grpc.CallOption) (*GetModulePackagesResponse, error) {
	out := new(GetModulePackagesResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.DocService/GetModulePackages", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *docServiceClient) GetModuleDocumentation(ctx context.Context, in *GetModuleDocumentationRequest, opts ...grpc.CallOption) (*GetModuleDocumentationResponse, error) {
	out := new(GetModuleDocumentationResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.DocService/GetModuleDocumentation", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *docServiceClient) GetPackageDocumentation(ctx context.Context, in *GetPackageDocumentationRequest, opts ...grpc.CallOption) (*GetPackageDocumentationResponse, error) {
	out := new(GetPackageDocumentationResponse)
	err := c.cc.Invoke(ctx, "/buf.alpha.registry.v1alpha1.DocService/GetPackageDocumentation", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DocServiceServer is the server API for DocService service.
// All implementations should embed UnimplementedDocServiceServer
// for forward compatibility
type DocServiceServer interface {
	// GetSourceDirectoryInfo retrieves the directory and file structure for the
	// given owner, repository and reference.
	//
	// The purpose of this is to get a representation of the file tree for a given
	// module to enable exploring the module by navigating through its contents.
	GetSourceDirectoryInfo(context.Context, *GetSourceDirectoryInfoRequest) (*GetSourceDirectoryInfoResponse, error)
	// GetSourceFile retrieves the source contents for the given owner, repository,
	// reference, and path.
	GetSourceFile(context.Context, *GetSourceFileRequest) (*GetSourceFileResponse, error)
	// GetModulePackages retrieves the list of packages for the module based on the given
	// owner, repository, and reference.
	GetModulePackages(context.Context, *GetModulePackagesRequest) (*GetModulePackagesResponse, error)
	// GetModuleDocumentation retrieves the documentation for module based on the given
	// owner, repository, and reference.
	GetModuleDocumentation(context.Context, *GetModuleDocumentationRequest) (*GetModuleDocumentationResponse, error)
	// GetPackageDocumentation retrieves a a slice of documentation structures
	// for the given owner, repository, reference, and package name.
	GetPackageDocumentation(context.Context, *GetPackageDocumentationRequest) (*GetPackageDocumentationResponse, error)
}

// UnimplementedDocServiceServer should be embedded to have forward compatible implementations.
type UnimplementedDocServiceServer struct {
}

func (UnimplementedDocServiceServer) GetSourceDirectoryInfo(context.Context, *GetSourceDirectoryInfoRequest) (*GetSourceDirectoryInfoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetSourceDirectoryInfo not implemented")
}
func (UnimplementedDocServiceServer) GetSourceFile(context.Context, *GetSourceFileRequest) (*GetSourceFileResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetSourceFile not implemented")
}
func (UnimplementedDocServiceServer) GetModulePackages(context.Context, *GetModulePackagesRequest) (*GetModulePackagesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetModulePackages not implemented")
}
func (UnimplementedDocServiceServer) GetModuleDocumentation(context.Context, *GetModuleDocumentationRequest) (*GetModuleDocumentationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetModuleDocumentation not implemented")
}
func (UnimplementedDocServiceServer) GetPackageDocumentation(context.Context, *GetPackageDocumentationRequest) (*GetPackageDocumentationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetPackageDocumentation not implemented")
}

// UnsafeDocServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to DocServiceServer will
// result in compilation errors.
type UnsafeDocServiceServer interface {
	mustEmbedUnimplementedDocServiceServer()
}

func RegisterDocServiceServer(s grpc.ServiceRegistrar, srv DocServiceServer) {
	s.RegisterService(&DocService_ServiceDesc, srv)
}

func _DocService_GetSourceDirectoryInfo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetSourceDirectoryInfoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DocServiceServer).GetSourceDirectoryInfo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.DocService/GetSourceDirectoryInfo",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DocServiceServer).GetSourceDirectoryInfo(ctx, req.(*GetSourceDirectoryInfoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DocService_GetSourceFile_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetSourceFileRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DocServiceServer).GetSourceFile(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.DocService/GetSourceFile",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DocServiceServer).GetSourceFile(ctx, req.(*GetSourceFileRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DocService_GetModulePackages_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetModulePackagesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DocServiceServer).GetModulePackages(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.DocService/GetModulePackages",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DocServiceServer).GetModulePackages(ctx, req.(*GetModulePackagesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DocService_GetModuleDocumentation_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetModuleDocumentationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DocServiceServer).GetModuleDocumentation(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.DocService/GetModuleDocumentation",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DocServiceServer).GetModuleDocumentation(ctx, req.(*GetModuleDocumentationRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DocService_GetPackageDocumentation_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetPackageDocumentationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DocServiceServer).GetPackageDocumentation(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buf.alpha.registry.v1alpha1.DocService/GetPackageDocumentation",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DocServiceServer).GetPackageDocumentation(ctx, req.(*GetPackageDocumentationRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// DocService_ServiceDesc is the grpc.ServiceDesc for DocService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var DocService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "buf.alpha.registry.v1alpha1.DocService",
	HandlerType: (*DocServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetSourceDirectoryInfo",
			Handler:    _DocService_GetSourceDirectoryInfo_Handler,
		},
		{
			MethodName: "GetSourceFile",
			Handler:    _DocService_GetSourceFile_Handler,
		},
		{
			MethodName: "GetModulePackages",
			Handler:    _DocService_GetModulePackages_Handler,
		},
		{
			MethodName: "GetModuleDocumentation",
			Handler:    _DocService_GetModuleDocumentation_Handler,
		},
		{
			MethodName: "GetPackageDocumentation",
			Handler:    _DocService_GetPackageDocumentation_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "buf/alpha/registry/v1alpha1/doc.proto",
}
