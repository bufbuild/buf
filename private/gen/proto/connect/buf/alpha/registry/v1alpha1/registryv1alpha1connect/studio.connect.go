// Copyright 2020-2025 Buf Technologies, Inc.
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
	connect "connectrpc.com/connect"
	context "context"
	errors "errors"
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	http "net/http"
	strings "strings"
)

// This is a compile-time assertion to ensure that this generated file and the connect package are
// compatible. If you get a compiler error that this constant is not defined, this code was
// generated with a version of connect newer than the one compiled into your binary. You can fix the
// problem by either regenerating this code with an older version of connect or updating the connect
// version compiled into your binary.
const _ = connect.IsAtLeastVersion1_13_0

const (
	// StudioServiceName is the fully-qualified name of the StudioService service.
	StudioServiceName = "buf.alpha.registry.v1alpha1.StudioService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// StudioServiceListStudioAgentPresetsProcedure is the fully-qualified name of the StudioService's
	// ListStudioAgentPresets RPC.
	StudioServiceListStudioAgentPresetsProcedure = "/buf.alpha.registry.v1alpha1.StudioService/ListStudioAgentPresets"
	// StudioServiceSetStudioAgentPresetsProcedure is the fully-qualified name of the StudioService's
	// SetStudioAgentPresets RPC.
	StudioServiceSetStudioAgentPresetsProcedure = "/buf.alpha.registry.v1alpha1.StudioService/SetStudioAgentPresets"
)

// StudioServiceClient is a client for the buf.alpha.registry.v1alpha1.StudioService service.
type StudioServiceClient interface {
	// ListStudioAgentPresets returns a list of agent presets in the server.
	ListStudioAgentPresets(context.Context, *connect.Request[v1alpha1.ListStudioAgentPresetsRequest]) (*connect.Response[v1alpha1.ListStudioAgentPresetsResponse], error)
	// SetStudioAgentPresets sets the list of agent presets in the server.
	SetStudioAgentPresets(context.Context, *connect.Request[v1alpha1.SetStudioAgentPresetsRequest]) (*connect.Response[v1alpha1.SetStudioAgentPresetsResponse], error)
}

// NewStudioServiceClient constructs a client for the buf.alpha.registry.v1alpha1.StudioService
// service. By default, it uses the Connect protocol with the binary Protobuf Codec, asks for
// gzipped responses, and sends uncompressed requests. To use the gRPC or gRPC-Web protocols, supply
// the connect.WithGRPC() or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewStudioServiceClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) StudioServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	studioServiceMethods := v1alpha1.File_buf_alpha_registry_v1alpha1_studio_proto.Services().ByName("StudioService").Methods()
	return &studioServiceClient{
		listStudioAgentPresets: connect.NewClient[v1alpha1.ListStudioAgentPresetsRequest, v1alpha1.ListStudioAgentPresetsResponse](
			httpClient,
			baseURL+StudioServiceListStudioAgentPresetsProcedure,
			connect.WithSchema(studioServiceMethods.ByName("ListStudioAgentPresets")),
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
		setStudioAgentPresets: connect.NewClient[v1alpha1.SetStudioAgentPresetsRequest, v1alpha1.SetStudioAgentPresetsResponse](
			httpClient,
			baseURL+StudioServiceSetStudioAgentPresetsProcedure,
			connect.WithSchema(studioServiceMethods.ByName("SetStudioAgentPresets")),
			connect.WithClientOptions(opts...),
		),
	}
}

// studioServiceClient implements StudioServiceClient.
type studioServiceClient struct {
	listStudioAgentPresets *connect.Client[v1alpha1.ListStudioAgentPresetsRequest, v1alpha1.ListStudioAgentPresetsResponse]
	setStudioAgentPresets  *connect.Client[v1alpha1.SetStudioAgentPresetsRequest, v1alpha1.SetStudioAgentPresetsResponse]
}

// ListStudioAgentPresets calls buf.alpha.registry.v1alpha1.StudioService.ListStudioAgentPresets.
func (c *studioServiceClient) ListStudioAgentPresets(ctx context.Context, req *connect.Request[v1alpha1.ListStudioAgentPresetsRequest]) (*connect.Response[v1alpha1.ListStudioAgentPresetsResponse], error) {
	return c.listStudioAgentPresets.CallUnary(ctx, req)
}

// SetStudioAgentPresets calls buf.alpha.registry.v1alpha1.StudioService.SetStudioAgentPresets.
func (c *studioServiceClient) SetStudioAgentPresets(ctx context.Context, req *connect.Request[v1alpha1.SetStudioAgentPresetsRequest]) (*connect.Response[v1alpha1.SetStudioAgentPresetsResponse], error) {
	return c.setStudioAgentPresets.CallUnary(ctx, req)
}

// StudioServiceHandler is an implementation of the buf.alpha.registry.v1alpha1.StudioService
// service.
type StudioServiceHandler interface {
	// ListStudioAgentPresets returns a list of agent presets in the server.
	ListStudioAgentPresets(context.Context, *connect.Request[v1alpha1.ListStudioAgentPresetsRequest]) (*connect.Response[v1alpha1.ListStudioAgentPresetsResponse], error)
	// SetStudioAgentPresets sets the list of agent presets in the server.
	SetStudioAgentPresets(context.Context, *connect.Request[v1alpha1.SetStudioAgentPresetsRequest]) (*connect.Response[v1alpha1.SetStudioAgentPresetsResponse], error)
}

// NewStudioServiceHandler builds an HTTP handler from the service implementation. It returns the
// path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewStudioServiceHandler(svc StudioServiceHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	studioServiceMethods := v1alpha1.File_buf_alpha_registry_v1alpha1_studio_proto.Services().ByName("StudioService").Methods()
	studioServiceListStudioAgentPresetsHandler := connect.NewUnaryHandler(
		StudioServiceListStudioAgentPresetsProcedure,
		svc.ListStudioAgentPresets,
		connect.WithSchema(studioServiceMethods.ByName("ListStudioAgentPresets")),
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	studioServiceSetStudioAgentPresetsHandler := connect.NewUnaryHandler(
		StudioServiceSetStudioAgentPresetsProcedure,
		svc.SetStudioAgentPresets,
		connect.WithSchema(studioServiceMethods.ByName("SetStudioAgentPresets")),
		connect.WithHandlerOptions(opts...),
	)
	return "/buf.alpha.registry.v1alpha1.StudioService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case StudioServiceListStudioAgentPresetsProcedure:
			studioServiceListStudioAgentPresetsHandler.ServeHTTP(w, r)
		case StudioServiceSetStudioAgentPresetsProcedure:
			studioServiceSetStudioAgentPresetsHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedStudioServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedStudioServiceHandler struct{}

func (UnimplementedStudioServiceHandler) ListStudioAgentPresets(context.Context, *connect.Request[v1alpha1.ListStudioAgentPresetsRequest]) (*connect.Response[v1alpha1.ListStudioAgentPresetsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.StudioService.ListStudioAgentPresets is not implemented"))
}

func (UnimplementedStudioServiceHandler) SetStudioAgentPresets(context.Context, *connect.Request[v1alpha1.SetStudioAgentPresetsRequest]) (*connect.Response[v1alpha1.SetStudioAgentPresetsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.StudioService.SetStudioAgentPresets is not implemented"))
}
