// Copyright 2020-2023 Buf Technologies, Inc.
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
// buf/alpha/registry/v1alpha1/generate.proto is a deprecated file.

package registryv1alpha1connect

import (
	v1alpha1 "buf.build/gen/go/bufbuild/buf/protocolbuffers/go/buf/alpha/registry/v1alpha1"
	context "context"
	errors "errors"
	connect_go "github.com/bufbuild/connect-go"
	http "net/http"
	strings "strings"
)

// This is a compile-time assertion to ensure that this generated file and the connect package are
// compatible. If you get a compiler error that this constant is not defined, this code was
// generated with a version of connect newer than the one compiled into your binary. You can fix the
// problem by either regenerating this code with an older version of connect or updating the connect
// version compiled into your binary.
const _ = connect_go.IsAtLeastVersion0_1_0

const (
	// GenerateServiceName is the fully-qualified name of the GenerateService service.
	GenerateServiceName = "buf.alpha.registry.v1alpha1.GenerateService"
)

// GenerateServiceClient is a client for the buf.alpha.registry.v1alpha1.GenerateService service.
type GenerateServiceClient interface {
	// GeneratePlugins generates an array of files given the provided
	// module reference and plugin version and option tuples. No attempt
	// is made at merging insertion points.
	GeneratePlugins(context.Context, *connect_go.Request[v1alpha1.GeneratePluginsRequest]) (*connect_go.Response[v1alpha1.GeneratePluginsResponse], error)
	// GenerateTemplate generates an array of files given the provided
	// module reference and template version.
	GenerateTemplate(context.Context, *connect_go.Request[v1alpha1.GenerateTemplateRequest]) (*connect_go.Response[v1alpha1.GenerateTemplateResponse], error)
}

// NewGenerateServiceClient constructs a client for the buf.alpha.registry.v1alpha1.GenerateService
// service. By default, it uses the Connect protocol with the binary Protobuf Codec, asks for
// gzipped responses, and sends uncompressed requests. To use the gRPC or gRPC-Web protocols, supply
// the connect.WithGRPC() or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewGenerateServiceClient(httpClient connect_go.HTTPClient, baseURL string, opts ...connect_go.ClientOption) GenerateServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &generateServiceClient{
		generatePlugins: connect_go.NewClient[v1alpha1.GeneratePluginsRequest, v1alpha1.GeneratePluginsResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.GenerateService/GeneratePlugins",
			opts...,
		),
		generateTemplate: connect_go.NewClient[v1alpha1.GenerateTemplateRequest, v1alpha1.GenerateTemplateResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.GenerateService/GenerateTemplate",
			opts...,
		),
	}
}

// generateServiceClient implements GenerateServiceClient.
type generateServiceClient struct {
	generatePlugins  *connect_go.Client[v1alpha1.GeneratePluginsRequest, v1alpha1.GeneratePluginsResponse]
	generateTemplate *connect_go.Client[v1alpha1.GenerateTemplateRequest, v1alpha1.GenerateTemplateResponse]
}

// GeneratePlugins calls buf.alpha.registry.v1alpha1.GenerateService.GeneratePlugins.
func (c *generateServiceClient) GeneratePlugins(ctx context.Context, req *connect_go.Request[v1alpha1.GeneratePluginsRequest]) (*connect_go.Response[v1alpha1.GeneratePluginsResponse], error) {
	return c.generatePlugins.CallUnary(ctx, req)
}

// GenerateTemplate calls buf.alpha.registry.v1alpha1.GenerateService.GenerateTemplate.
func (c *generateServiceClient) GenerateTemplate(ctx context.Context, req *connect_go.Request[v1alpha1.GenerateTemplateRequest]) (*connect_go.Response[v1alpha1.GenerateTemplateResponse], error) {
	return c.generateTemplate.CallUnary(ctx, req)
}

// GenerateServiceHandler is an implementation of the buf.alpha.registry.v1alpha1.GenerateService
// service.
type GenerateServiceHandler interface {
	// GeneratePlugins generates an array of files given the provided
	// module reference and plugin version and option tuples. No attempt
	// is made at merging insertion points.
	GeneratePlugins(context.Context, *connect_go.Request[v1alpha1.GeneratePluginsRequest]) (*connect_go.Response[v1alpha1.GeneratePluginsResponse], error)
	// GenerateTemplate generates an array of files given the provided
	// module reference and template version.
	GenerateTemplate(context.Context, *connect_go.Request[v1alpha1.GenerateTemplateRequest]) (*connect_go.Response[v1alpha1.GenerateTemplateResponse], error)
}

// NewGenerateServiceHandler builds an HTTP handler from the service implementation. It returns the
// path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewGenerateServiceHandler(svc GenerateServiceHandler, opts ...connect_go.HandlerOption) (string, http.Handler) {
	mux := http.NewServeMux()
	mux.Handle("/buf.alpha.registry.v1alpha1.GenerateService/GeneratePlugins", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.GenerateService/GeneratePlugins",
		svc.GeneratePlugins,
		opts...,
	))
	mux.Handle("/buf.alpha.registry.v1alpha1.GenerateService/GenerateTemplate", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.GenerateService/GenerateTemplate",
		svc.GenerateTemplate,
		opts...,
	))
	return "/buf.alpha.registry.v1alpha1.GenerateService/", mux
}

// UnimplementedGenerateServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedGenerateServiceHandler struct{}

func (UnimplementedGenerateServiceHandler) GeneratePlugins(context.Context, *connect_go.Request[v1alpha1.GeneratePluginsRequest]) (*connect_go.Response[v1alpha1.GeneratePluginsResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.GenerateService.GeneratePlugins is not implemented"))
}

func (UnimplementedGenerateServiceHandler) GenerateTemplate(context.Context, *connect_go.Request[v1alpha1.GenerateTemplateRequest]) (*connect_go.Response[v1alpha1.GenerateTemplateResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.GenerateService.GenerateTemplate is not implemented"))
}
