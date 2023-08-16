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
// Source: buf/alpha/registry/v1alpha1/resolve.proto

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
const _ = connect.IsAtLeastVersion1_7_0

const (
	// ResolveServiceName is the fully-qualified name of the ResolveService service.
	ResolveServiceName = "buf.alpha.registry.v1alpha1.ResolveService"
	// LocalResolveServiceName is the fully-qualified name of the LocalResolveService service.
	LocalResolveServiceName = "buf.alpha.registry.v1alpha1.LocalResolveService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// ResolveServiceGetModulePinsProcedure is the fully-qualified name of the ResolveService's
	// GetModulePins RPC.
	ResolveServiceGetModulePinsProcedure = "/buf.alpha.registry.v1alpha1.ResolveService/GetModulePins"
	// ResolveServiceGetGoVersionProcedure is the fully-qualified name of the ResolveService's
	// GetGoVersion RPC.
	ResolveServiceGetGoVersionProcedure = "/buf.alpha.registry.v1alpha1.ResolveService/GetGoVersion"
	// ResolveServiceGetSwiftVersionProcedure is the fully-qualified name of the ResolveService's
	// GetSwiftVersion RPC.
	ResolveServiceGetSwiftVersionProcedure = "/buf.alpha.registry.v1alpha1.ResolveService/GetSwiftVersion"
	// ResolveServiceGetMavenVersionProcedure is the fully-qualified name of the ResolveService's
	// GetMavenVersion RPC.
	ResolveServiceGetMavenVersionProcedure = "/buf.alpha.registry.v1alpha1.ResolveService/GetMavenVersion"
	// ResolveServiceGetNPMVersionProcedure is the fully-qualified name of the ResolveService's
	// GetNPMVersion RPC.
	ResolveServiceGetNPMVersionProcedure = "/buf.alpha.registry.v1alpha1.ResolveService/GetNPMVersion"
	// LocalResolveServiceGetLocalModulePinsProcedure is the fully-qualified name of the
	// LocalResolveService's GetLocalModulePins RPC.
	LocalResolveServiceGetLocalModulePinsProcedure = "/buf.alpha.registry.v1alpha1.LocalResolveService/GetLocalModulePins"
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
	GetModulePins(context.Context, *connect.Request[v1alpha1.GetModulePinsRequest]) (*connect.Response[v1alpha1.GetModulePinsResponse], error)
	// GetGoVersion resolves the given plugin and module references to a version.
	GetGoVersion(context.Context, *connect.Request[v1alpha1.GetGoVersionRequest]) (*connect.Response[v1alpha1.GetGoVersionResponse], error)
	// GetSwiftVersion resolves the given plugin and module references to a version.
	GetSwiftVersion(context.Context, *connect.Request[v1alpha1.GetSwiftVersionRequest]) (*connect.Response[v1alpha1.GetSwiftVersionResponse], error)
	// GetMavenVersion resolves the given plugin and module references to a version.
	GetMavenVersion(context.Context, *connect.Request[v1alpha1.GetMavenVersionRequest]) (*connect.Response[v1alpha1.GetMavenVersionResponse], error)
	// GetNPMVersion resolves the given plugin and module references to a version.
	GetNPMVersion(context.Context, *connect.Request[v1alpha1.GetNPMVersionRequest]) (*connect.Response[v1alpha1.GetNPMVersionResponse], error)
}

// NewResolveServiceClient constructs a client for the buf.alpha.registry.v1alpha1.ResolveService
// service. By default, it uses the Connect protocol with the binary Protobuf Codec, asks for
// gzipped responses, and sends uncompressed requests. To use the gRPC or gRPC-Web protocols, supply
// the connect.WithGRPC() or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewResolveServiceClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) ResolveServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &resolveServiceClient{
		getModulePins: connect.NewClient[v1alpha1.GetModulePinsRequest, v1alpha1.GetModulePinsResponse](
			httpClient,
			baseURL+ResolveServiceGetModulePinsProcedure,
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
		getGoVersion: connect.NewClient[v1alpha1.GetGoVersionRequest, v1alpha1.GetGoVersionResponse](
			httpClient,
			baseURL+ResolveServiceGetGoVersionProcedure,
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
		getSwiftVersion: connect.NewClient[v1alpha1.GetSwiftVersionRequest, v1alpha1.GetSwiftVersionResponse](
			httpClient,
			baseURL+ResolveServiceGetSwiftVersionProcedure,
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
		getMavenVersion: connect.NewClient[v1alpha1.GetMavenVersionRequest, v1alpha1.GetMavenVersionResponse](
			httpClient,
			baseURL+ResolveServiceGetMavenVersionProcedure,
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
		getNPMVersion: connect.NewClient[v1alpha1.GetNPMVersionRequest, v1alpha1.GetNPMVersionResponse](
			httpClient,
			baseURL+ResolveServiceGetNPMVersionProcedure,
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
	}
}

// resolveServiceClient implements ResolveServiceClient.
type resolveServiceClient struct {
	getModulePins   *connect.Client[v1alpha1.GetModulePinsRequest, v1alpha1.GetModulePinsResponse]
	getGoVersion    *connect.Client[v1alpha1.GetGoVersionRequest, v1alpha1.GetGoVersionResponse]
	getSwiftVersion *connect.Client[v1alpha1.GetSwiftVersionRequest, v1alpha1.GetSwiftVersionResponse]
	getMavenVersion *connect.Client[v1alpha1.GetMavenVersionRequest, v1alpha1.GetMavenVersionResponse]
	getNPMVersion   *connect.Client[v1alpha1.GetNPMVersionRequest, v1alpha1.GetNPMVersionResponse]
}

// GetModulePins calls buf.alpha.registry.v1alpha1.ResolveService.GetModulePins.
func (c *resolveServiceClient) GetModulePins(ctx context.Context, req *connect.Request[v1alpha1.GetModulePinsRequest]) (*connect.Response[v1alpha1.GetModulePinsResponse], error) {
	return c.getModulePins.CallUnary(ctx, req)
}

// GetGoVersion calls buf.alpha.registry.v1alpha1.ResolveService.GetGoVersion.
func (c *resolveServiceClient) GetGoVersion(ctx context.Context, req *connect.Request[v1alpha1.GetGoVersionRequest]) (*connect.Response[v1alpha1.GetGoVersionResponse], error) {
	return c.getGoVersion.CallUnary(ctx, req)
}

// GetSwiftVersion calls buf.alpha.registry.v1alpha1.ResolveService.GetSwiftVersion.
func (c *resolveServiceClient) GetSwiftVersion(ctx context.Context, req *connect.Request[v1alpha1.GetSwiftVersionRequest]) (*connect.Response[v1alpha1.GetSwiftVersionResponse], error) {
	return c.getSwiftVersion.CallUnary(ctx, req)
}

// GetMavenVersion calls buf.alpha.registry.v1alpha1.ResolveService.GetMavenVersion.
func (c *resolveServiceClient) GetMavenVersion(ctx context.Context, req *connect.Request[v1alpha1.GetMavenVersionRequest]) (*connect.Response[v1alpha1.GetMavenVersionResponse], error) {
	return c.getMavenVersion.CallUnary(ctx, req)
}

// GetNPMVersion calls buf.alpha.registry.v1alpha1.ResolveService.GetNPMVersion.
func (c *resolveServiceClient) GetNPMVersion(ctx context.Context, req *connect.Request[v1alpha1.GetNPMVersionRequest]) (*connect.Response[v1alpha1.GetNPMVersionResponse], error) {
	return c.getNPMVersion.CallUnary(ctx, req)
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
	GetModulePins(context.Context, *connect.Request[v1alpha1.GetModulePinsRequest]) (*connect.Response[v1alpha1.GetModulePinsResponse], error)
	// GetGoVersion resolves the given plugin and module references to a version.
	GetGoVersion(context.Context, *connect.Request[v1alpha1.GetGoVersionRequest]) (*connect.Response[v1alpha1.GetGoVersionResponse], error)
	// GetSwiftVersion resolves the given plugin and module references to a version.
	GetSwiftVersion(context.Context, *connect.Request[v1alpha1.GetSwiftVersionRequest]) (*connect.Response[v1alpha1.GetSwiftVersionResponse], error)
	// GetMavenVersion resolves the given plugin and module references to a version.
	GetMavenVersion(context.Context, *connect.Request[v1alpha1.GetMavenVersionRequest]) (*connect.Response[v1alpha1.GetMavenVersionResponse], error)
	// GetNPMVersion resolves the given plugin and module references to a version.
	GetNPMVersion(context.Context, *connect.Request[v1alpha1.GetNPMVersionRequest]) (*connect.Response[v1alpha1.GetNPMVersionResponse], error)
}

// NewResolveServiceHandler builds an HTTP handler from the service implementation. It returns the
// path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewResolveServiceHandler(svc ResolveServiceHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	resolveServiceGetModulePinsHandler := connect.NewUnaryHandler(
		ResolveServiceGetModulePinsProcedure,
		svc.GetModulePins,
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	resolveServiceGetGoVersionHandler := connect.NewUnaryHandler(
		ResolveServiceGetGoVersionProcedure,
		svc.GetGoVersion,
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	resolveServiceGetSwiftVersionHandler := connect.NewUnaryHandler(
		ResolveServiceGetSwiftVersionProcedure,
		svc.GetSwiftVersion,
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	resolveServiceGetMavenVersionHandler := connect.NewUnaryHandler(
		ResolveServiceGetMavenVersionProcedure,
		svc.GetMavenVersion,
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	resolveServiceGetNPMVersionHandler := connect.NewUnaryHandler(
		ResolveServiceGetNPMVersionProcedure,
		svc.GetNPMVersion,
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	return "/buf.alpha.registry.v1alpha1.ResolveService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case ResolveServiceGetModulePinsProcedure:
			resolveServiceGetModulePinsHandler.ServeHTTP(w, r)
		case ResolveServiceGetGoVersionProcedure:
			resolveServiceGetGoVersionHandler.ServeHTTP(w, r)
		case ResolveServiceGetSwiftVersionProcedure:
			resolveServiceGetSwiftVersionHandler.ServeHTTP(w, r)
		case ResolveServiceGetMavenVersionProcedure:
			resolveServiceGetMavenVersionHandler.ServeHTTP(w, r)
		case ResolveServiceGetNPMVersionProcedure:
			resolveServiceGetNPMVersionHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedResolveServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedResolveServiceHandler struct{}

func (UnimplementedResolveServiceHandler) GetModulePins(context.Context, *connect.Request[v1alpha1.GetModulePinsRequest]) (*connect.Response[v1alpha1.GetModulePinsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.ResolveService.GetModulePins is not implemented"))
}

func (UnimplementedResolveServiceHandler) GetGoVersion(context.Context, *connect.Request[v1alpha1.GetGoVersionRequest]) (*connect.Response[v1alpha1.GetGoVersionResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.ResolveService.GetGoVersion is not implemented"))
}

func (UnimplementedResolveServiceHandler) GetSwiftVersion(context.Context, *connect.Request[v1alpha1.GetSwiftVersionRequest]) (*connect.Response[v1alpha1.GetSwiftVersionResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.ResolveService.GetSwiftVersion is not implemented"))
}

func (UnimplementedResolveServiceHandler) GetMavenVersion(context.Context, *connect.Request[v1alpha1.GetMavenVersionRequest]) (*connect.Response[v1alpha1.GetMavenVersionResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.ResolveService.GetMavenVersion is not implemented"))
}

func (UnimplementedResolveServiceHandler) GetNPMVersion(context.Context, *connect.Request[v1alpha1.GetNPMVersionRequest]) (*connect.Response[v1alpha1.GetNPMVersionResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.ResolveService.GetNPMVersion is not implemented"))
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
	GetLocalModulePins(context.Context, *connect.Request[v1alpha1.GetLocalModulePinsRequest]) (*connect.Response[v1alpha1.GetLocalModulePinsResponse], error)
}

// NewLocalResolveServiceClient constructs a client for the
// buf.alpha.registry.v1alpha1.LocalResolveService service. By default, it uses the Connect protocol
// with the binary Protobuf Codec, asks for gzipped responses, and sends uncompressed requests. To
// use the gRPC or gRPC-Web protocols, supply the connect.WithGRPC() or connect.WithGRPCWeb()
// options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewLocalResolveServiceClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) LocalResolveServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &localResolveServiceClient{
		getLocalModulePins: connect.NewClient[v1alpha1.GetLocalModulePinsRequest, v1alpha1.GetLocalModulePinsResponse](
			httpClient,
			baseURL+LocalResolveServiceGetLocalModulePinsProcedure,
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
	}
}

// localResolveServiceClient implements LocalResolveServiceClient.
type localResolveServiceClient struct {
	getLocalModulePins *connect.Client[v1alpha1.GetLocalModulePinsRequest, v1alpha1.GetLocalModulePinsResponse]
}

// GetLocalModulePins calls buf.alpha.registry.v1alpha1.LocalResolveService.GetLocalModulePins.
func (c *localResolveServiceClient) GetLocalModulePins(ctx context.Context, req *connect.Request[v1alpha1.GetLocalModulePinsRequest]) (*connect.Response[v1alpha1.GetLocalModulePinsResponse], error) {
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
	GetLocalModulePins(context.Context, *connect.Request[v1alpha1.GetLocalModulePinsRequest]) (*connect.Response[v1alpha1.GetLocalModulePinsResponse], error)
}

// NewLocalResolveServiceHandler builds an HTTP handler from the service implementation. It returns
// the path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewLocalResolveServiceHandler(svc LocalResolveServiceHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	localResolveServiceGetLocalModulePinsHandler := connect.NewUnaryHandler(
		LocalResolveServiceGetLocalModulePinsProcedure,
		svc.GetLocalModulePins,
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	return "/buf.alpha.registry.v1alpha1.LocalResolveService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case LocalResolveServiceGetLocalModulePinsProcedure:
			localResolveServiceGetLocalModulePinsHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedLocalResolveServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedLocalResolveServiceHandler struct{}

func (UnimplementedLocalResolveServiceHandler) GetLocalModulePins(context.Context, *connect.Request[v1alpha1.GetLocalModulePinsRequest]) (*connect.Response[v1alpha1.GetLocalModulePinsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.LocalResolveService.GetLocalModulePins is not implemented"))
}
