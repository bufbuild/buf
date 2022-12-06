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
// Source: buf/alpha/registry/v1alpha1/studio.proto

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
	// StudioServiceName is the fully-qualified name of the StudioService service.
	StudioServiceName = "buf.alpha.registry.v1alpha1.StudioService"
)

// StudioServiceClient is a client for the buf.alpha.registry.v1alpha1.StudioService service.
type StudioServiceClient interface {
	// ListStudioAgentPresets returns a list of agent presets in the server.
	ListStudioAgentPresets(context.Context, *connect_go.Request[v1alpha1.ListStudioAgentPresetsRequest]) (*connect_go.Response[v1alpha1.ListStudioAgentPresetsResponse], error)
	// SetStudioAgentPresets sets the list of agent presets in the server.
	SetStudioAgentPresets(context.Context, *connect_go.Request[v1alpha1.SetStudioAgentPresetsRequest]) (*connect_go.Response[v1alpha1.SetStudioAgentPresetsResponse], error)
}

// NewStudioServiceClient constructs a client for the buf.alpha.registry.v1alpha1.StudioService
// service. By default, it uses the Connect protocol with the binary Protobuf Codec, asks for
// gzipped responses, and sends uncompressed requests. To use the gRPC or gRPC-Web protocols, supply
// the connect.WithGRPC() or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewStudioServiceClient(httpClient connect_go.HTTPClient, baseURL string, opts ...connect_go.ClientOption) StudioServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &studioServiceClient{
		listStudioAgentPresets: connect_go.NewClient[v1alpha1.ListStudioAgentPresetsRequest, v1alpha1.ListStudioAgentPresetsResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.StudioService/ListStudioAgentPresets",
			opts...,
		),
		setStudioAgentPresets: connect_go.NewClient[v1alpha1.SetStudioAgentPresetsRequest, v1alpha1.SetStudioAgentPresetsResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.StudioService/SetStudioAgentPresets",
			opts...,
		),
	}
}

// studioServiceClient implements StudioServiceClient.
type studioServiceClient struct {
	listStudioAgentPresets *connect_go.Client[v1alpha1.ListStudioAgentPresetsRequest, v1alpha1.ListStudioAgentPresetsResponse]
	setStudioAgentPresets  *connect_go.Client[v1alpha1.SetStudioAgentPresetsRequest, v1alpha1.SetStudioAgentPresetsResponse]
}

// ListStudioAgentPresets calls buf.alpha.registry.v1alpha1.StudioService.ListStudioAgentPresets.
func (c *studioServiceClient) ListStudioAgentPresets(ctx context.Context, req *connect_go.Request[v1alpha1.ListStudioAgentPresetsRequest]) (*connect_go.Response[v1alpha1.ListStudioAgentPresetsResponse], error) {
	return c.listStudioAgentPresets.CallUnary(ctx, req)
}

// SetStudioAgentPresets calls buf.alpha.registry.v1alpha1.StudioService.SetStudioAgentPresets.
func (c *studioServiceClient) SetStudioAgentPresets(ctx context.Context, req *connect_go.Request[v1alpha1.SetStudioAgentPresetsRequest]) (*connect_go.Response[v1alpha1.SetStudioAgentPresetsResponse], error) {
	return c.setStudioAgentPresets.CallUnary(ctx, req)
}

// StudioServiceHandler is an implementation of the buf.alpha.registry.v1alpha1.StudioService
// service.
type StudioServiceHandler interface {
	// ListStudioAgentPresets returns a list of agent presets in the server.
	ListStudioAgentPresets(context.Context, *connect_go.Request[v1alpha1.ListStudioAgentPresetsRequest]) (*connect_go.Response[v1alpha1.ListStudioAgentPresetsResponse], error)
	// SetStudioAgentPresets sets the list of agent presets in the server.
	SetStudioAgentPresets(context.Context, *connect_go.Request[v1alpha1.SetStudioAgentPresetsRequest]) (*connect_go.Response[v1alpha1.SetStudioAgentPresetsResponse], error)
}

// NewStudioServiceHandler builds an HTTP handler from the service implementation. It returns the
// path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewStudioServiceHandler(svc StudioServiceHandler, opts ...connect_go.HandlerOption) (string, http.Handler) {
	mux := http.NewServeMux()
	mux.Handle("/buf.alpha.registry.v1alpha1.StudioService/ListStudioAgentPresets", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.StudioService/ListStudioAgentPresets",
		svc.ListStudioAgentPresets,
		opts...,
	))
	mux.Handle("/buf.alpha.registry.v1alpha1.StudioService/SetStudioAgentPresets", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.StudioService/SetStudioAgentPresets",
		svc.SetStudioAgentPresets,
		opts...,
	))
	return "/buf.alpha.registry.v1alpha1.StudioService/", mux
}

// UnimplementedStudioServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedStudioServiceHandler struct{}

func (UnimplementedStudioServiceHandler) ListStudioAgentPresets(context.Context, *connect_go.Request[v1alpha1.ListStudioAgentPresetsRequest]) (*connect_go.Response[v1alpha1.ListStudioAgentPresetsResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.StudioService.ListStudioAgentPresets is not implemented"))
}

func (UnimplementedStudioServiceHandler) SetStudioAgentPresets(context.Context, *connect_go.Request[v1alpha1.SetStudioAgentPresetsRequest]) (*connect_go.Response[v1alpha1.SetStudioAgentPresetsResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.StudioService.SetStudioAgentPresets is not implemented"))
}
