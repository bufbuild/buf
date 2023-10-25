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
// Source: buf/registry/owner/v1beta1/owner_service.proto

package ownerv1beta1connect

import (
	connect "connectrpc.com/connect"
	context "context"
	errors "errors"
	v1beta1 "github.com/bufbuild/buf/private/gen/proto/go/buf/registry/owner/v1beta1"
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
	// OwnerServiceName is the fully-qualified name of the OwnerService service.
	OwnerServiceName = "buf.registry.owner.v1beta1.OwnerService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// OwnerServiceGetOwnersProcedure is the fully-qualified name of the OwnerService's GetOwners RPC.
	OwnerServiceGetOwnersProcedure = "/buf.registry.owner.v1beta1.OwnerService/GetOwners"
)

// OwnerServiceClient is a client for the buf.registry.owner.v1beta1.OwnerService service.
type OwnerServiceClient interface {
	// Get Users or Organizations by id or name.
	GetOwners(context.Context, *connect.Request[v1beta1.GetOwnersRequest]) (*connect.Response[v1beta1.GetOwnersResponse], error)
}

// NewOwnerServiceClient constructs a client for the buf.registry.owner.v1beta1.OwnerService
// service. By default, it uses the Connect protocol with the binary Protobuf Codec, asks for
// gzipped responses, and sends uncompressed requests. To use the gRPC or gRPC-Web protocols, supply
// the connect.WithGRPC() or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewOwnerServiceClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) OwnerServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &ownerServiceClient{
		getOwners: connect.NewClient[v1beta1.GetOwnersRequest, v1beta1.GetOwnersResponse](
			httpClient,
			baseURL+OwnerServiceGetOwnersProcedure,
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
	}
}

// ownerServiceClient implements OwnerServiceClient.
type ownerServiceClient struct {
	getOwners *connect.Client[v1beta1.GetOwnersRequest, v1beta1.GetOwnersResponse]
}

// GetOwners calls buf.registry.owner.v1beta1.OwnerService.GetOwners.
func (c *ownerServiceClient) GetOwners(ctx context.Context, req *connect.Request[v1beta1.GetOwnersRequest]) (*connect.Response[v1beta1.GetOwnersResponse], error) {
	return c.getOwners.CallUnary(ctx, req)
}

// OwnerServiceHandler is an implementation of the buf.registry.owner.v1beta1.OwnerService service.
type OwnerServiceHandler interface {
	// Get Users or Organizations by id or name.
	GetOwners(context.Context, *connect.Request[v1beta1.GetOwnersRequest]) (*connect.Response[v1beta1.GetOwnersResponse], error)
}

// NewOwnerServiceHandler builds an HTTP handler from the service implementation. It returns the
// path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewOwnerServiceHandler(svc OwnerServiceHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	ownerServiceGetOwnersHandler := connect.NewUnaryHandler(
		OwnerServiceGetOwnersProcedure,
		svc.GetOwners,
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	return "/buf.registry.owner.v1beta1.OwnerService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case OwnerServiceGetOwnersProcedure:
			ownerServiceGetOwnersHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedOwnerServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedOwnerServiceHandler struct{}

func (UnimplementedOwnerServiceHandler) GetOwners(context.Context, *connect.Request[v1beta1.GetOwnersRequest]) (*connect.Response[v1beta1.GetOwnersResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.registry.owner.v1beta1.OwnerService.GetOwners is not implemented"))
}
