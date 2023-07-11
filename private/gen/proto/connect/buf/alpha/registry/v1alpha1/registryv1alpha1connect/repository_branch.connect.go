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
const _ = connect_go.IsAtLeastVersion1_7_0

const (
	// RepositoryBranchServiceName is the fully-qualified name of the RepositoryBranchService service.
	RepositoryBranchServiceName = "buf.alpha.registry.v1alpha1.RepositoryBranchService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// RepositoryBranchServiceListRepositoryBranchesProcedure is the fully-qualified name of the
	// RepositoryBranchService's ListRepositoryBranches RPC.
	RepositoryBranchServiceListRepositoryBranchesProcedure = "/buf.alpha.registry.v1alpha1.RepositoryBranchService/ListRepositoryBranches"
	// RepositoryBranchServiceGetDefaultBranchProcedure is the fully-qualified name of the
	// RepositoryBranchService's GetDefaultBranch RPC.
	RepositoryBranchServiceGetDefaultBranchProcedure = "/buf.alpha.registry.v1alpha1.RepositoryBranchService/GetDefaultBranch"
	// RepositoryBranchServiceListRepositoryNonDefaultBranchesProcedure is the fully-qualified name of
	// the RepositoryBranchService's ListRepositoryNonDefaultBranches RPC.
	RepositoryBranchServiceListRepositoryNonDefaultBranchesProcedure = "/buf.alpha.registry.v1alpha1.RepositoryBranchService/ListRepositoryNonDefaultBranches"
)

// RepositoryBranchServiceClient is a client for the
// buf.alpha.registry.v1alpha1.RepositoryBranchService service.
type RepositoryBranchServiceClient interface {
	// ListRepositoryBranchs lists the repository branches associated with a Repository.
	ListRepositoryBranches(context.Context, *connect_go.Request[v1alpha1.ListRepositoryBranchesRequest]) (*connect_go.Response[v1alpha1.ListRepositoryBranchesResponse], error)
	// GetDefaultBranch returns the branch name that is mapped to the main/BSR_HEAD.
	GetDefaultBranch(context.Context, *connect_go.Request[v1alpha1.GetDefaultBranchRequest]) (*connect_go.Response[v1alpha1.GetDefaultBranchResponse], error)
	// ListRepositoryNonDefaultBranches returns a paginated list of non-default branches in the BSR.
	ListRepositoryNonDefaultBranches(context.Context, *connect_go.Request[v1alpha1.ListRepositoryNonDefaultBranchesRequest]) (*connect_go.Response[v1alpha1.ListRepositoryNonDefaultBranchesResponse], error)
}

// NewRepositoryBranchServiceClient constructs a client for the
// buf.alpha.registry.v1alpha1.RepositoryBranchService service. By default, it uses the Connect
// protocol with the binary Protobuf Codec, asks for gzipped responses, and sends uncompressed
// requests. To use the gRPC or gRPC-Web protocols, supply the connect.WithGRPC() or
// connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewRepositoryBranchServiceClient(httpClient connect_go.HTTPClient, baseURL string, opts ...connect_go.ClientOption) RepositoryBranchServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &repositoryBranchServiceClient{
		listRepositoryBranches: connect_go.NewClient[v1alpha1.ListRepositoryBranchesRequest, v1alpha1.ListRepositoryBranchesResponse](
			httpClient,
			baseURL+RepositoryBranchServiceListRepositoryBranchesProcedure,
			connect_go.WithIdempotency(connect_go.IdempotencyNoSideEffects),
			connect_go.WithClientOptions(opts...),
		),
		getDefaultBranch: connect_go.NewClient[v1alpha1.GetDefaultBranchRequest, v1alpha1.GetDefaultBranchResponse](
			httpClient,
			baseURL+RepositoryBranchServiceGetDefaultBranchProcedure,
			connect_go.WithIdempotency(connect_go.IdempotencyNoSideEffects),
			connect_go.WithClientOptions(opts...),
		),
		listRepositoryNonDefaultBranches: connect_go.NewClient[v1alpha1.ListRepositoryNonDefaultBranchesRequest, v1alpha1.ListRepositoryNonDefaultBranchesResponse](
			httpClient,
			baseURL+RepositoryBranchServiceListRepositoryNonDefaultBranchesProcedure,
			connect_go.WithIdempotency(connect_go.IdempotencyNoSideEffects),
			connect_go.WithClientOptions(opts...),
		),
	}
}

// repositoryBranchServiceClient implements RepositoryBranchServiceClient.
type repositoryBranchServiceClient struct {
	listRepositoryBranches           *connect_go.Client[v1alpha1.ListRepositoryBranchesRequest, v1alpha1.ListRepositoryBranchesResponse]
	getDefaultBranch                 *connect_go.Client[v1alpha1.GetDefaultBranchRequest, v1alpha1.GetDefaultBranchResponse]
	listRepositoryNonDefaultBranches *connect_go.Client[v1alpha1.ListRepositoryNonDefaultBranchesRequest, v1alpha1.ListRepositoryNonDefaultBranchesResponse]
}

// ListRepositoryBranches calls
// buf.alpha.registry.v1alpha1.RepositoryBranchService.ListRepositoryBranches.
func (c *repositoryBranchServiceClient) ListRepositoryBranches(ctx context.Context, req *connect_go.Request[v1alpha1.ListRepositoryBranchesRequest]) (*connect_go.Response[v1alpha1.ListRepositoryBranchesResponse], error) {
	return c.listRepositoryBranches.CallUnary(ctx, req)
}

// GetDefaultBranch calls buf.alpha.registry.v1alpha1.RepositoryBranchService.GetDefaultBranch.
func (c *repositoryBranchServiceClient) GetDefaultBranch(ctx context.Context, req *connect_go.Request[v1alpha1.GetDefaultBranchRequest]) (*connect_go.Response[v1alpha1.GetDefaultBranchResponse], error) {
	return c.getDefaultBranch.CallUnary(ctx, req)
}

// ListRepositoryNonDefaultBranches calls
// buf.alpha.registry.v1alpha1.RepositoryBranchService.ListRepositoryNonDefaultBranches.
func (c *repositoryBranchServiceClient) ListRepositoryNonDefaultBranches(ctx context.Context, req *connect_go.Request[v1alpha1.ListRepositoryNonDefaultBranchesRequest]) (*connect_go.Response[v1alpha1.ListRepositoryNonDefaultBranchesResponse], error) {
	return c.listRepositoryNonDefaultBranches.CallUnary(ctx, req)
}

// RepositoryBranchServiceHandler is an implementation of the
// buf.alpha.registry.v1alpha1.RepositoryBranchService service.
type RepositoryBranchServiceHandler interface {
	// ListRepositoryBranchs lists the repository branches associated with a Repository.
	ListRepositoryBranches(context.Context, *connect_go.Request[v1alpha1.ListRepositoryBranchesRequest]) (*connect_go.Response[v1alpha1.ListRepositoryBranchesResponse], error)
	// GetDefaultBranch returns the branch name that is mapped to the main/BSR_HEAD.
	GetDefaultBranch(context.Context, *connect_go.Request[v1alpha1.GetDefaultBranchRequest]) (*connect_go.Response[v1alpha1.GetDefaultBranchResponse], error)
	// ListRepositoryNonDefaultBranches returns a paginated list of non-default branches in the BSR.
	ListRepositoryNonDefaultBranches(context.Context, *connect_go.Request[v1alpha1.ListRepositoryNonDefaultBranchesRequest]) (*connect_go.Response[v1alpha1.ListRepositoryNonDefaultBranchesResponse], error)
}

// NewRepositoryBranchServiceHandler builds an HTTP handler from the service implementation. It
// returns the path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewRepositoryBranchServiceHandler(svc RepositoryBranchServiceHandler, opts ...connect_go.HandlerOption) (string, http.Handler) {
	repositoryBranchServiceListRepositoryBranchesHandler := connect_go.NewUnaryHandler(
		RepositoryBranchServiceListRepositoryBranchesProcedure,
		svc.ListRepositoryBranches,
		connect_go.WithIdempotency(connect_go.IdempotencyNoSideEffects),
		connect_go.WithHandlerOptions(opts...),
	)
	repositoryBranchServiceGetDefaultBranchHandler := connect_go.NewUnaryHandler(
		RepositoryBranchServiceGetDefaultBranchProcedure,
		svc.GetDefaultBranch,
		connect_go.WithIdempotency(connect_go.IdempotencyNoSideEffects),
		connect_go.WithHandlerOptions(opts...),
	)
	repositoryBranchServiceListRepositoryNonDefaultBranchesHandler := connect_go.NewUnaryHandler(
		RepositoryBranchServiceListRepositoryNonDefaultBranchesProcedure,
		svc.ListRepositoryNonDefaultBranches,
		connect_go.WithIdempotency(connect_go.IdempotencyNoSideEffects),
		connect_go.WithHandlerOptions(opts...),
	)
	return "/buf.alpha.registry.v1alpha1.RepositoryBranchService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case RepositoryBranchServiceListRepositoryBranchesProcedure:
			repositoryBranchServiceListRepositoryBranchesHandler.ServeHTTP(w, r)
		case RepositoryBranchServiceGetDefaultBranchProcedure:
			repositoryBranchServiceGetDefaultBranchHandler.ServeHTTP(w, r)
		case RepositoryBranchServiceListRepositoryNonDefaultBranchesProcedure:
			repositoryBranchServiceListRepositoryNonDefaultBranchesHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedRepositoryBranchServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedRepositoryBranchServiceHandler struct{}

func (UnimplementedRepositoryBranchServiceHandler) ListRepositoryBranches(context.Context, *connect_go.Request[v1alpha1.ListRepositoryBranchesRequest]) (*connect_go.Response[v1alpha1.ListRepositoryBranchesResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.RepositoryBranchService.ListRepositoryBranches is not implemented"))
}

func (UnimplementedRepositoryBranchServiceHandler) GetDefaultBranch(context.Context, *connect_go.Request[v1alpha1.GetDefaultBranchRequest]) (*connect_go.Response[v1alpha1.GetDefaultBranchResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.RepositoryBranchService.GetDefaultBranch is not implemented"))
}

func (UnimplementedRepositoryBranchServiceHandler) ListRepositoryNonDefaultBranches(context.Context, *connect_go.Request[v1alpha1.ListRepositoryNonDefaultBranchesRequest]) (*connect_go.Response[v1alpha1.ListRepositoryNonDefaultBranchesResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.RepositoryBranchService.ListRepositoryNonDefaultBranches is not implemented"))
}
