// Code generated by protoc-gen-connect-go. DO NOT EDIT.
//
// Source: buf/alpha/registry/v1alpha1/resolve.proto

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
// service. By default, it uses the binary Protobuf Codec, asks for gzipped responses, and sends
// uncompressed requests. It doesn't have a default protocol; you must supply either the
// connect.WithGRPC() or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the gRPC server (for example,
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
// By default, handlers support the gRPC and gRPC-Web protocols with the binary Protobuf and JSON
// codecs.
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
	// 1. It makes it easy to say "we know we're looking for owner/repo on this specific remote".
	//    While we could just do this in GetModulePins by being aware of what our remote is
	//    (something we probably still need to know, DNS problems aside, which are more
	//    theoretical), this helps.
	// 2. Having a separate method makes us able to say "do not make decisions about what
	//    wins between competing pins for the same repo". This should only be done in
	//    GetModulePins, not in this function, i.e. only done at the top level.
	GetLocalModulePins(context.Context, *connect_go.Request[v1alpha1.GetLocalModulePinsRequest]) (*connect_go.Response[v1alpha1.GetLocalModulePinsResponse], error)
}

// NewLocalResolveServiceClient constructs a client for the
// buf.alpha.registry.v1alpha1.LocalResolveService service. By default, it uses the binary Protobuf
// Codec, asks for gzipped responses, and sends uncompressed requests. It doesn't have a default
// protocol; you must supply either the connect.WithGRPC() or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the gRPC server (for example,
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
	// 1. It makes it easy to say "we know we're looking for owner/repo on this specific remote".
	//    While we could just do this in GetModulePins by being aware of what our remote is
	//    (something we probably still need to know, DNS problems aside, which are more
	//    theoretical), this helps.
	// 2. Having a separate method makes us able to say "do not make decisions about what
	//    wins between competing pins for the same repo". This should only be done in
	//    GetModulePins, not in this function, i.e. only done at the top level.
	GetLocalModulePins(context.Context, *connect_go.Request[v1alpha1.GetLocalModulePinsRequest]) (*connect_go.Response[v1alpha1.GetLocalModulePinsResponse], error)
}

// NewLocalResolveServiceHandler builds an HTTP handler from the service implementation. It returns
// the path on which to mount the handler and the handler itself.
//
// By default, handlers support the gRPC and gRPC-Web protocols with the binary Protobuf and JSON
// codecs.
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
