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
// Source: buf/alpha/registry/v1alpha1/resource.proto

package registryv1alpha1connect

import (
	context "context"
	errors "errors"
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	connect_go "connectrpc.com/connect"
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
	// ResourceServiceName is the fully-qualified name of the ResourceService service.
	ResourceServiceName = "buf.alpha.registry.v1alpha1.ResourceService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// ResourceServiceGetResourceByNameProcedure is the fully-qualified name of the ResourceService's
	// GetResourceByName RPC.
	ResourceServiceGetResourceByNameProcedure = "/buf.alpha.registry.v1alpha1.ResourceService/GetResourceByName"
)

// ResourceServiceClient is a client for the buf.alpha.registry.v1alpha1.ResourceService service.
type ResourceServiceClient interface {
	// GetResourceByName takes a resource name and returns the
	// resource either as a repository or a plugin.
	GetResourceByName(context.Context, *connect_go.Request[v1alpha1.GetResourceByNameRequest]) (*connect_go.Response[v1alpha1.GetResourceByNameResponse], error)
}

// NewResourceServiceClient constructs a client for the buf.alpha.registry.v1alpha1.ResourceService
// service. By default, it uses the Connect protocol with the binary Protobuf Codec, asks for
// gzipped responses, and sends uncompressed requests. To use the gRPC or gRPC-Web protocols, supply
// the connect.WithGRPC() or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewResourceServiceClient(httpClient connect_go.HTTPClient, baseURL string, opts ...connect_go.ClientOption) ResourceServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &resourceServiceClient{
		getResourceByName: connect_go.NewClient[v1alpha1.GetResourceByNameRequest, v1alpha1.GetResourceByNameResponse](
			httpClient,
			baseURL+ResourceServiceGetResourceByNameProcedure,
			connect_go.WithIdempotency(connect_go.IdempotencyNoSideEffects),
			connect_go.WithClientOptions(opts...),
		),
	}
}

// resourceServiceClient implements ResourceServiceClient.
type resourceServiceClient struct {
	getResourceByName *connect_go.Client[v1alpha1.GetResourceByNameRequest, v1alpha1.GetResourceByNameResponse]
}

// GetResourceByName calls buf.alpha.registry.v1alpha1.ResourceService.GetResourceByName.
func (c *resourceServiceClient) GetResourceByName(ctx context.Context, req *connect_go.Request[v1alpha1.GetResourceByNameRequest]) (*connect_go.Response[v1alpha1.GetResourceByNameResponse], error) {
	return c.getResourceByName.CallUnary(ctx, req)
}

// ResourceServiceHandler is an implementation of the buf.alpha.registry.v1alpha1.ResourceService
// service.
type ResourceServiceHandler interface {
	// GetResourceByName takes a resource name and returns the
	// resource either as a repository or a plugin.
	GetResourceByName(context.Context, *connect_go.Request[v1alpha1.GetResourceByNameRequest]) (*connect_go.Response[v1alpha1.GetResourceByNameResponse], error)
}

// NewResourceServiceHandler builds an HTTP handler from the service implementation. It returns the
// path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewResourceServiceHandler(svc ResourceServiceHandler, opts ...connect_go.HandlerOption) (string, http.Handler) {
	resourceServiceGetResourceByNameHandler := connect_go.NewUnaryHandler(
		ResourceServiceGetResourceByNameProcedure,
		svc.GetResourceByName,
		connect_go.WithIdempotency(connect_go.IdempotencyNoSideEffects),
		connect_go.WithHandlerOptions(opts...),
	)
	return "/buf.alpha.registry.v1alpha1.ResourceService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case ResourceServiceGetResourceByNameProcedure:
			resourceServiceGetResourceByNameHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedResourceServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedResourceServiceHandler struct{}

func (UnimplementedResourceServiceHandler) GetResourceByName(context.Context, *connect_go.Request[v1alpha1.GetResourceByNameRequest]) (*connect_go.Response[v1alpha1.GetResourceByNameResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.ResourceService.GetResourceByName is not implemented"))
}
