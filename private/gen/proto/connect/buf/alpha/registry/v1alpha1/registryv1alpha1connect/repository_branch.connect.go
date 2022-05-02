// Code generated by protoc-gen-connect-go. DO NOT EDIT.
//
// Source: buf/alpha/registry/v1alpha1/repository_branch.proto

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
	// RepositoryBranchServiceName is the fully-qualified name of the RepositoryBranchService service.
	RepositoryBranchServiceName = "buf.alpha.registry.v1alpha1.RepositoryBranchService"
)

// RepositoryBranchServiceClient is a client for the
// buf.alpha.registry.v1alpha1.RepositoryBranchService service.
type RepositoryBranchServiceClient interface {
	// CreateRepositoryBranch creates a new repository branch.
	CreateRepositoryBranch(context.Context, *connect_go.Request[v1alpha1.CreateRepositoryBranchRequest]) (*connect_go.Response[v1alpha1.CreateRepositoryBranchResponse], error)
	// ListRepositoryBranches lists the repository branches associated with a Repository.
	ListRepositoryBranches(context.Context, *connect_go.Request[v1alpha1.ListRepositoryBranchesRequest]) (*connect_go.Response[v1alpha1.ListRepositoryBranchesResponse], error)
}

// NewRepositoryBranchServiceClient constructs a client for the
// buf.alpha.registry.v1alpha1.RepositoryBranchService service. By default, it uses the binary
// Protobuf Codec, asks for gzipped responses, and sends uncompressed requests. It doesn't have a
// default protocol; you must supply either the connect.WithGRPC() or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewRepositoryBranchServiceClient(httpClient connect_go.HTTPClient, baseURL string, opts ...connect_go.ClientOption) RepositoryBranchServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &repositoryBranchServiceClient{
		createRepositoryBranch: connect_go.NewClient[v1alpha1.CreateRepositoryBranchRequest, v1alpha1.CreateRepositoryBranchResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.RepositoryBranchService/CreateRepositoryBranch",
			opts...,
		),
		listRepositoryBranches: connect_go.NewClient[v1alpha1.ListRepositoryBranchesRequest, v1alpha1.ListRepositoryBranchesResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.RepositoryBranchService/ListRepositoryBranches",
			opts...,
		),
	}
}

// repositoryBranchServiceClient implements RepositoryBranchServiceClient.
type repositoryBranchServiceClient struct {
	createRepositoryBranch *connect_go.Client[v1alpha1.CreateRepositoryBranchRequest, v1alpha1.CreateRepositoryBranchResponse]
	listRepositoryBranches *connect_go.Client[v1alpha1.ListRepositoryBranchesRequest, v1alpha1.ListRepositoryBranchesResponse]
}

// CreateRepositoryBranch calls
// buf.alpha.registry.v1alpha1.RepositoryBranchService.CreateRepositoryBranch.
func (c *repositoryBranchServiceClient) CreateRepositoryBranch(ctx context.Context, req *connect_go.Request[v1alpha1.CreateRepositoryBranchRequest]) (*connect_go.Response[v1alpha1.CreateRepositoryBranchResponse], error) {
	return c.createRepositoryBranch.CallUnary(ctx, req)
}

// ListRepositoryBranches calls
// buf.alpha.registry.v1alpha1.RepositoryBranchService.ListRepositoryBranches.
func (c *repositoryBranchServiceClient) ListRepositoryBranches(ctx context.Context, req *connect_go.Request[v1alpha1.ListRepositoryBranchesRequest]) (*connect_go.Response[v1alpha1.ListRepositoryBranchesResponse], error) {
	return c.listRepositoryBranches.CallUnary(ctx, req)
}

// RepositoryBranchServiceHandler is an implementation of the
// buf.alpha.registry.v1alpha1.RepositoryBranchService service.
type RepositoryBranchServiceHandler interface {
	// CreateRepositoryBranch creates a new repository branch.
	CreateRepositoryBranch(context.Context, *connect_go.Request[v1alpha1.CreateRepositoryBranchRequest]) (*connect_go.Response[v1alpha1.CreateRepositoryBranchResponse], error)
	// ListRepositoryBranches lists the repository branches associated with a Repository.
	ListRepositoryBranches(context.Context, *connect_go.Request[v1alpha1.ListRepositoryBranchesRequest]) (*connect_go.Response[v1alpha1.ListRepositoryBranchesResponse], error)
}

// NewRepositoryBranchServiceHandler builds an HTTP handler from the service implementation. It
// returns the path on which to mount the handler and the handler itself.
//
// By default, handlers support the gRPC and gRPC-Web protocols with the binary Protobuf and JSON
// codecs.
func NewRepositoryBranchServiceHandler(svc RepositoryBranchServiceHandler, opts ...connect_go.HandlerOption) (string, http.Handler) {
	mux := http.NewServeMux()
	mux.Handle("/buf.alpha.registry.v1alpha1.RepositoryBranchService/CreateRepositoryBranch", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.RepositoryBranchService/CreateRepositoryBranch",
		svc.CreateRepositoryBranch,
		opts...,
	))
	mux.Handle("/buf.alpha.registry.v1alpha1.RepositoryBranchService/ListRepositoryBranches", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.RepositoryBranchService/ListRepositoryBranches",
		svc.ListRepositoryBranches,
		opts...,
	))
	return "/buf.alpha.registry.v1alpha1.RepositoryBranchService/", mux
}

// UnimplementedRepositoryBranchServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedRepositoryBranchServiceHandler struct{}

func (UnimplementedRepositoryBranchServiceHandler) CreateRepositoryBranch(context.Context, *connect_go.Request[v1alpha1.CreateRepositoryBranchRequest]) (*connect_go.Response[v1alpha1.CreateRepositoryBranchResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.RepositoryBranchService.CreateRepositoryBranch is not implemented"))
}

func (UnimplementedRepositoryBranchServiceHandler) ListRepositoryBranches(context.Context, *connect_go.Request[v1alpha1.ListRepositoryBranchesRequest]) (*connect_go.Response[v1alpha1.ListRepositoryBranchesResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.RepositoryBranchService.ListRepositoryBranches is not implemented"))
}
