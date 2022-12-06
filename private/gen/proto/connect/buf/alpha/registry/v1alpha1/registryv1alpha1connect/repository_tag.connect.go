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

// Code generated by protoc-gen-connect-go. DO NOT EDIT.
//
// Source: buf/alpha/registry/v1alpha1/repository_tag.proto

package registryv1alpha1connect

import (
	context "context"
	errors "errors"
	http "net/http"
	strings "strings"

	v1alpha1 "buf.build/gen/go/bufbuild/buf/protocolbuffers/go/buf/alpha/registry/v1alpha1"
	connect_go "github.com/bufbuild/connect-go"
)

// This is a compile-time assertion to ensure that this generated file and the connect package are
// compatible. If you get a compiler error that this constant is not defined, this code was
// generated with a version of connect newer than the one compiled into your binary. You can fix the
// problem by either regenerating this code with an older version of connect or updating the connect
// version compiled into your binary.
const _ = connect_go.IsAtLeastVersion0_1_0

const (
	// RepositoryTagServiceName is the fully-qualified name of the RepositoryTagService service.
	RepositoryTagServiceName = "buf.alpha.registry.v1alpha1.RepositoryTagService"
)

// RepositoryTagServiceClient is a client for the buf.alpha.registry.v1alpha1.RepositoryTagService
// service.
type RepositoryTagServiceClient interface {
	// CreateRepositoryTag creates a new repository tag.
	CreateRepositoryTag(context.Context, *connect_go.Request[v1alpha1.CreateRepositoryTagRequest]) (*connect_go.Response[v1alpha1.CreateRepositoryTagResponse], error)
	// ListRepositoryTags lists the repository tags associated with a Repository.
	ListRepositoryTags(context.Context, *connect_go.Request[v1alpha1.ListRepositoryTagsRequest]) (*connect_go.Response[v1alpha1.ListRepositoryTagsResponse], error)
}

// NewRepositoryTagServiceClient constructs a client for the
// buf.alpha.registry.v1alpha1.RepositoryTagService service. By default, it uses the Connect
// protocol with the binary Protobuf Codec, asks for gzipped responses, and sends uncompressed
// requests. To use the gRPC or gRPC-Web protocols, supply the connect.WithGRPC() or
// connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewRepositoryTagServiceClient(httpClient connect_go.HTTPClient, baseURL string, opts ...connect_go.ClientOption) RepositoryTagServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &repositoryTagServiceClient{
		createRepositoryTag: connect_go.NewClient[v1alpha1.CreateRepositoryTagRequest, v1alpha1.CreateRepositoryTagResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.RepositoryTagService/CreateRepositoryTag",
			opts...,
		),
		listRepositoryTags: connect_go.NewClient[v1alpha1.ListRepositoryTagsRequest, v1alpha1.ListRepositoryTagsResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.RepositoryTagService/ListRepositoryTags",
			opts...,
		),
	}
}

// repositoryTagServiceClient implements RepositoryTagServiceClient.
type repositoryTagServiceClient struct {
	createRepositoryTag *connect_go.Client[v1alpha1.CreateRepositoryTagRequest, v1alpha1.CreateRepositoryTagResponse]
	listRepositoryTags  *connect_go.Client[v1alpha1.ListRepositoryTagsRequest, v1alpha1.ListRepositoryTagsResponse]
}

// CreateRepositoryTag calls buf.alpha.registry.v1alpha1.RepositoryTagService.CreateRepositoryTag.
func (c *repositoryTagServiceClient) CreateRepositoryTag(ctx context.Context, req *connect_go.Request[v1alpha1.CreateRepositoryTagRequest]) (*connect_go.Response[v1alpha1.CreateRepositoryTagResponse], error) {
	return c.createRepositoryTag.CallUnary(ctx, req)
}

// ListRepositoryTags calls buf.alpha.registry.v1alpha1.RepositoryTagService.ListRepositoryTags.
func (c *repositoryTagServiceClient) ListRepositoryTags(ctx context.Context, req *connect_go.Request[v1alpha1.ListRepositoryTagsRequest]) (*connect_go.Response[v1alpha1.ListRepositoryTagsResponse], error) {
	return c.listRepositoryTags.CallUnary(ctx, req)
}

// RepositoryTagServiceHandler is an implementation of the
// buf.alpha.registry.v1alpha1.RepositoryTagService service.
type RepositoryTagServiceHandler interface {
	// CreateRepositoryTag creates a new repository tag.
	CreateRepositoryTag(context.Context, *connect_go.Request[v1alpha1.CreateRepositoryTagRequest]) (*connect_go.Response[v1alpha1.CreateRepositoryTagResponse], error)
	// ListRepositoryTags lists the repository tags associated with a Repository.
	ListRepositoryTags(context.Context, *connect_go.Request[v1alpha1.ListRepositoryTagsRequest]) (*connect_go.Response[v1alpha1.ListRepositoryTagsResponse], error)
}

// NewRepositoryTagServiceHandler builds an HTTP handler from the service implementation. It returns
// the path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewRepositoryTagServiceHandler(svc RepositoryTagServiceHandler, opts ...connect_go.HandlerOption) (string, http.Handler) {
	mux := http.NewServeMux()
	mux.Handle("/buf.alpha.registry.v1alpha1.RepositoryTagService/CreateRepositoryTag", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.RepositoryTagService/CreateRepositoryTag",
		svc.CreateRepositoryTag,
		opts...,
	))
	mux.Handle("/buf.alpha.registry.v1alpha1.RepositoryTagService/ListRepositoryTags", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.RepositoryTagService/ListRepositoryTags",
		svc.ListRepositoryTags,
		opts...,
	))
	return "/buf.alpha.registry.v1alpha1.RepositoryTagService/", mux
}

// UnimplementedRepositoryTagServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedRepositoryTagServiceHandler struct{}

func (UnimplementedRepositoryTagServiceHandler) CreateRepositoryTag(context.Context, *connect_go.Request[v1alpha1.CreateRepositoryTagRequest]) (*connect_go.Response[v1alpha1.CreateRepositoryTagResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.RepositoryTagService.CreateRepositoryTag is not implemented"))
}

func (UnimplementedRepositoryTagServiceHandler) ListRepositoryTags(context.Context, *connect_go.Request[v1alpha1.ListRepositoryTagsRequest]) (*connect_go.Response[v1alpha1.ListRepositoryTagsResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.RepositoryTagService.ListRepositoryTags is not implemented"))
}
