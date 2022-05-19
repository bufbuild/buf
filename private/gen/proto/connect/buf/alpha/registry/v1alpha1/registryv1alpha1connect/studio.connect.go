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
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	connect_go "github.com/bufbuild/connect-go"
	http "net/http"
	strings "strings"
)

// This is a compile-time assertion to ensure that this generated file and the connect package are
// compatible. If you get a compiler error that this constant is not defined, this code was
// generated with a version of connect newer than the one compiled into your binary. You can fix the
// problem by either regenerating this code with an older version of connect or updating the connect
// version compiled into your binary.
const _ = connect_go.IsAtLeastVersion0_0_1

const (
	// StudioServiceName is the fully-qualified name of the StudioService service.
	StudioServiceName = "buf.alpha.registry.v1alpha1.StudioService"
)

// StudioServiceClient is a client for the buf.alpha.registry.v1alpha1.StudioService service.
type StudioServiceClient interface {
	// ListPresetAgents returns a list of preset agents in the server.
	ListPresetAgents(context.Context, *connect_go.Request[v1alpha1.ListPresetAgentsRequest]) (*connect_go.Response[v1alpha1.ListPresetAgentsResponse], error)
	// SetPresetAgents set the list of preset agents in the server.
	SetPresetAgents(context.Context, *connect_go.Request[v1alpha1.SetPresetAgentsRequest]) (*connect_go.Response[v1alpha1.SetPresetAgentsResponse], error)
}

// NewStudioServiceClient constructs a client for the buf.alpha.registry.v1alpha1.StudioService
// service. By default, it uses the binary Protobuf Codec, asks for gzipped responses, and sends
// uncompressed requests. It doesn't have a default protocol; you must supply either the
// connect.WithGRPC() or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewStudioServiceClient(httpClient connect_go.HTTPClient, baseURL string, opts ...connect_go.ClientOption) StudioServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &studioServiceClient{
		listPresetAgents: connect_go.NewClient[v1alpha1.ListPresetAgentsRequest, v1alpha1.ListPresetAgentsResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.StudioService/ListPresetAgents",
			opts...,
		),
		setPresetAgents: connect_go.NewClient[v1alpha1.SetPresetAgentsRequest, v1alpha1.SetPresetAgentsResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.StudioService/SetPresetAgents",
			opts...,
		),
	}
}

// studioServiceClient implements StudioServiceClient.
type studioServiceClient struct {
	listPresetAgents *connect_go.Client[v1alpha1.ListPresetAgentsRequest, v1alpha1.ListPresetAgentsResponse]
	setPresetAgents  *connect_go.Client[v1alpha1.SetPresetAgentsRequest, v1alpha1.SetPresetAgentsResponse]
}

// ListPresetAgents calls buf.alpha.registry.v1alpha1.StudioService.ListPresetAgents.
func (c *studioServiceClient) ListPresetAgents(ctx context.Context, req *connect_go.Request[v1alpha1.ListPresetAgentsRequest]) (*connect_go.Response[v1alpha1.ListPresetAgentsResponse], error) {
	return c.listPresetAgents.CallUnary(ctx, req)
}

// SetPresetAgents calls buf.alpha.registry.v1alpha1.StudioService.SetPresetAgents.
func (c *studioServiceClient) SetPresetAgents(ctx context.Context, req *connect_go.Request[v1alpha1.SetPresetAgentsRequest]) (*connect_go.Response[v1alpha1.SetPresetAgentsResponse], error) {
	return c.setPresetAgents.CallUnary(ctx, req)
}

// StudioServiceHandler is an implementation of the buf.alpha.registry.v1alpha1.StudioService
// service.
type StudioServiceHandler interface {
	// ListPresetAgents returns a list of preset agents in the server.
	ListPresetAgents(context.Context, *connect_go.Request[v1alpha1.ListPresetAgentsRequest]) (*connect_go.Response[v1alpha1.ListPresetAgentsResponse], error)
	// SetPresetAgents set the list of preset agents in the server.
	SetPresetAgents(context.Context, *connect_go.Request[v1alpha1.SetPresetAgentsRequest]) (*connect_go.Response[v1alpha1.SetPresetAgentsResponse], error)
}

// NewStudioServiceHandler builds an HTTP handler from the service implementation. It returns the
// path on which to mount the handler and the handler itself.
//
// By default, handlers support the gRPC and gRPC-Web protocols with the binary Protobuf and JSON
// codecs.
func NewStudioServiceHandler(svc StudioServiceHandler, opts ...connect_go.HandlerOption) (string, http.Handler) {
	mux := http.NewServeMux()
	mux.Handle("/buf.alpha.registry.v1alpha1.StudioService/ListPresetAgents", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.StudioService/ListPresetAgents",
		svc.ListPresetAgents,
		opts...,
	))
	mux.Handle("/buf.alpha.registry.v1alpha1.StudioService/SetPresetAgents", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.StudioService/SetPresetAgents",
		svc.SetPresetAgents,
		opts...,
	))
	return "/buf.alpha.registry.v1alpha1.StudioService/", mux
}

// UnimplementedStudioServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedStudioServiceHandler struct{}

func (UnimplementedStudioServiceHandler) ListPresetAgents(context.Context, *connect_go.Request[v1alpha1.ListPresetAgentsRequest]) (*connect_go.Response[v1alpha1.ListPresetAgentsResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.StudioService.ListPresetAgents is not implemented"))
}

func (UnimplementedStudioServiceHandler) SetPresetAgents(context.Context, *connect_go.Request[v1alpha1.SetPresetAgentsRequest]) (*connect_go.Response[v1alpha1.SetPresetAgentsResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.StudioService.SetPresetAgents is not implemented"))
}
