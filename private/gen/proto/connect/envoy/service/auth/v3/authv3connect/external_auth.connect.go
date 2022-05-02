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
// Source: envoy/service/auth/v3/external_auth.proto

package authv3connect

import (
	context "context"
	errors "errors"
	connect_go "github.com/bufbuild/connect-go"
	v3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
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
	// AuthorizationName is the fully-qualified name of the Authorization service.
	AuthorizationName = "envoy.service.auth.v3.Authorization"
)

// AuthorizationClient is a client for the envoy.service.auth.v3.Authorization service.
type AuthorizationClient interface {
	// Performs authorization check based on the attributes associated with the
	// incoming request, and returns status `OK` or not `OK`.
	Check(context.Context, *connect_go.Request[v3.CheckRequest]) (*connect_go.Response[v3.CheckResponse], error)
}

// NewAuthorizationClient constructs a client for the envoy.service.auth.v3.Authorization service.
// By default, it uses the binary Protobuf Codec, asks for gzipped responses, and sends uncompressed
// requests. It doesn't have a default protocol; you must supply either the connect.WithGRPC() or
// connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewAuthorizationClient(httpClient connect_go.HTTPClient, baseURL string, opts ...connect_go.ClientOption) AuthorizationClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &authorizationClient{
		check: connect_go.NewClient[v3.CheckRequest, v3.CheckResponse](
			httpClient,
			baseURL+"/envoy.service.auth.v3.Authorization/Check",
			opts...,
		),
	}
}

// authorizationClient implements AuthorizationClient.
type authorizationClient struct {
	check *connect_go.Client[v3.CheckRequest, v3.CheckResponse]
}

// Check calls envoy.service.auth.v3.Authorization.Check.
func (c *authorizationClient) Check(ctx context.Context, req *connect_go.Request[v3.CheckRequest]) (*connect_go.Response[v3.CheckResponse], error) {
	return c.check.CallUnary(ctx, req)
}

// AuthorizationHandler is an implementation of the envoy.service.auth.v3.Authorization service.
type AuthorizationHandler interface {
	// Performs authorization check based on the attributes associated with the
	// incoming request, and returns status `OK` or not `OK`.
	Check(context.Context, *connect_go.Request[v3.CheckRequest]) (*connect_go.Response[v3.CheckResponse], error)
}

// NewAuthorizationHandler builds an HTTP handler from the service implementation. It returns the
// path on which to mount the handler and the handler itself.
//
// By default, handlers support the gRPC and gRPC-Web protocols with the binary Protobuf and JSON
// codecs.
func NewAuthorizationHandler(svc AuthorizationHandler, opts ...connect_go.HandlerOption) (string, http.Handler) {
	mux := http.NewServeMux()
	mux.Handle("/envoy.service.auth.v3.Authorization/Check", connect_go.NewUnaryHandler(
		"/envoy.service.auth.v3.Authorization/Check",
		svc.Check,
		opts...,
	))
	return "/envoy.service.auth.v3.Authorization/", mux
}

// UnimplementedAuthorizationHandler returns CodeUnimplemented from all methods.
type UnimplementedAuthorizationHandler struct{}

func (UnimplementedAuthorizationHandler) Check(context.Context, *connect_go.Request[v3.CheckRequest]) (*connect_go.Response[v3.CheckResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("envoy.service.auth.v3.Authorization.Check is not implemented"))
}
