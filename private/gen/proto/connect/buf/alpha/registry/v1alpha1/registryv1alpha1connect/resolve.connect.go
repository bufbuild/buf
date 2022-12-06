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
// Source: buf/alpha/registry/v1alpha1/resolve.proto

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
	// ResolveServiceName is the fully-qualified name of the ResolveService service.
	ResolveServiceName = "buf.alpha.registry.v1alpha1.ResolveService"
	// LocalResolveServiceName is the fully-qualified name of the LocalResolveService service.
	LocalResolveServiceName = "buf.alpha.registry.v1alpha1.LocalResolveService"
)

// ResolveServiceClient is a client for the buf.alpha.registry.v1alpha1.ResolveService service.
type ResolveServiceClient interface {
	// GetModulePins finds all the latest digests and respective dependencies of
	// the provided module references and picks a set of distinct modules pins.
	//
	// Note that module references with commits should still be passed to this function
	// to make sure this function can do dependency resolution.
	//
	// This function also deals with tiebreaking what ModulePin wins for the same repository.
	GetModulePins(context.Context, *connect_go.Request[v1alpha1.GetModulePinsRequest]) (*connect_go.Response[v1alpha1.GetModulePinsResponse], error)
}

// NewResolveServiceClient constructs a client for the buf.alpha.registry.v1alpha1.ResolveService
// service. By default, it uses the Connect protocol with the binary Protobuf Codec, asks for
// gzipped responses, and sends uncompressed requests. To use the gRPC or gRPC-Web protocols, supply
// the connect.WithGRPC() or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewResolveServiceClient(httpClient connect_go.HTTPClient, baseURL string, opts ...connect_go.ClientOption) ResolveServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &resolveServiceClient{
		getModulePins: connect_go.NewClient[v1alpha1.GetModulePinsRequest, v1alpha1.GetModulePinsResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.ResolveService/GetModulePins",
			opts...,
		),
	}
}

// resolveServiceClient implements ResolveServiceClient.
type resolveServiceClient struct {
	getModulePins *connect_go.Client[v1alpha1.GetModulePinsRequest, v1alpha1.GetModulePinsResponse]
}

// GetModulePins calls buf.alpha.registry.v1alpha1.ResolveService.GetModulePins.
func (c *resolveServiceClient) GetModulePins(ctx context.Context, req *connect_go.Request[v1alpha1.GetModulePinsRequest]) (*connect_go.Response[v1alpha1.GetModulePinsResponse], error) {
	return c.getModulePins.CallUnary(ctx, req)
}

// ResolveServiceHandler is an implementation of the buf.alpha.registry.v1alpha1.ResolveService
// service.
type ResolveServiceHandler interface {
	// GetModulePins finds all the latest digests and respective dependencies of
	// the provided module references and picks a set of distinct modules pins.
	//
	// Note that module references with commits should still be passed to this function
	// to make sure this function can do dependency resolution.
	//
	// This function also deals with tiebreaking what ModulePin wins for the same repository.
	GetModulePins(context.Context, *connect_go.Request[v1alpha1.GetModulePinsRequest]) (*connect_go.Response[v1alpha1.GetModulePinsResponse], error)
}

// NewResolveServiceHandler builds an HTTP handler from the service implementation. It returns the
// path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewResolveServiceHandler(svc ResolveServiceHandler, opts ...connect_go.HandlerOption) (string, http.Handler) {
	mux := http.NewServeMux()
	mux.Handle("/buf.alpha.registry.v1alpha1.ResolveService/GetModulePins", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.ResolveService/GetModulePins",
		svc.GetModulePins,
		opts...,
	))
	return "/buf.alpha.registry.v1alpha1.ResolveService/", mux
}

// UnimplementedResolveServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedResolveServiceHandler struct{}

func (UnimplementedResolveServiceHandler) GetModulePins(context.Context, *connect_go.Request[v1alpha1.GetModulePinsRequest]) (*connect_go.Response[v1alpha1.GetModulePinsResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.ResolveService.GetModulePins is not implemented"))
}

// LocalResolveServiceClient is a client for the buf.alpha.registry.v1alpha1.LocalResolveService
// service.
type LocalResolveServiceClient interface {
	// GetLocalModulePins gets the latest pins for the specified local module references.
	// It also includes all of the modules transitive dependencies for the specified references.
	//
	// We want this for two reasons:
	//
	//  1. It makes it easy to say "we know we're looking for owner/repo on this specific remote".
	//     While we could just do this in GetModulePins by being aware of what our remote is
	//     (something we probably still need to know, DNS problems aside, which are more
	//     theoretical), this helps.
	//  2. Having a separate method makes us able to say "do not make decisions about what
	//     wins between competing pins for the same repo". This should only be done in
	//     GetModulePins, not in this function, i.e. only done at the top level.
	GetLocalModulePins(context.Context, *connect_go.Request[v1alpha1.GetLocalModulePinsRequest]) (*connect_go.Response[v1alpha1.GetLocalModulePinsResponse], error)
}

// NewLocalResolveServiceClient constructs a client for the
// buf.alpha.registry.v1alpha1.LocalResolveService service. By default, it uses the Connect protocol
// with the binary Protobuf Codec, asks for gzipped responses, and sends uncompressed requests. To
// use the gRPC or gRPC-Web protocols, supply the connect.WithGRPC() or connect.WithGRPCWeb()
// options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewLocalResolveServiceClient(httpClient connect_go.HTTPClient, baseURL string, opts ...connect_go.ClientOption) LocalResolveServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &localResolveServiceClient{
		getLocalModulePins: connect_go.NewClient[v1alpha1.GetLocalModulePinsRequest, v1alpha1.GetLocalModulePinsResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.LocalResolveService/GetLocalModulePins",
			opts...,
		),
	}
}

// localResolveServiceClient implements LocalResolveServiceClient.
type localResolveServiceClient struct {
	getLocalModulePins *connect_go.Client[v1alpha1.GetLocalModulePinsRequest, v1alpha1.GetLocalModulePinsResponse]
}

// GetLocalModulePins calls buf.alpha.registry.v1alpha1.LocalResolveService.GetLocalModulePins.
func (c *localResolveServiceClient) GetLocalModulePins(ctx context.Context, req *connect_go.Request[v1alpha1.GetLocalModulePinsRequest]) (*connect_go.Response[v1alpha1.GetLocalModulePinsResponse], error) {
	return c.getLocalModulePins.CallUnary(ctx, req)
}

// LocalResolveServiceHandler is an implementation of the
// buf.alpha.registry.v1alpha1.LocalResolveService service.
type LocalResolveServiceHandler interface {
	// GetLocalModulePins gets the latest pins for the specified local module references.
	// It also includes all of the modules transitive dependencies for the specified references.
	//
	// We want this for two reasons:
	//
	//  1. It makes it easy to say "we know we're looking for owner/repo on this specific remote".
	//     While we could just do this in GetModulePins by being aware of what our remote is
	//     (something we probably still need to know, DNS problems aside, which are more
	//     theoretical), this helps.
	//  2. Having a separate method makes us able to say "do not make decisions about what
	//     wins between competing pins for the same repo". This should only be done in
	//     GetModulePins, not in this function, i.e. only done at the top level.
	GetLocalModulePins(context.Context, *connect_go.Request[v1alpha1.GetLocalModulePinsRequest]) (*connect_go.Response[v1alpha1.GetLocalModulePinsResponse], error)
}

// NewLocalResolveServiceHandler builds an HTTP handler from the service implementation. It returns
// the path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewLocalResolveServiceHandler(svc LocalResolveServiceHandler, opts ...connect_go.HandlerOption) (string, http.Handler) {
	mux := http.NewServeMux()
	mux.Handle("/buf.alpha.registry.v1alpha1.LocalResolveService/GetLocalModulePins", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.LocalResolveService/GetLocalModulePins",
		svc.GetLocalModulePins,
		opts...,
	))
	return "/buf.alpha.registry.v1alpha1.LocalResolveService/", mux
}

// UnimplementedLocalResolveServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedLocalResolveServiceHandler struct{}

func (UnimplementedLocalResolveServiceHandler) GetLocalModulePins(context.Context, *connect_go.Request[v1alpha1.GetLocalModulePinsRequest]) (*connect_go.Response[v1alpha1.GetLocalModulePinsResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.LocalResolveService.GetLocalModulePins is not implemented"))
}
