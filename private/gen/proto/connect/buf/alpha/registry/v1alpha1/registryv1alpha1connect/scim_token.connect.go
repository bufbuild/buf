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
// Source: buf/alpha/registry/v1alpha1/scim_token.proto

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
const _ = connect_go.IsAtLeastVersion0_1_0

const (
	// SCIMTokenServiceName is the fully-qualified name of the SCIMTokenService service.
	SCIMTokenServiceName = "buf.alpha.registry.v1alpha1.SCIMTokenService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// SCIMTokenServiceCreateSCIMTokenProcedure is the fully-qualified name of the SCIMTokenService's
	// CreateSCIMToken RPC.
	SCIMTokenServiceCreateSCIMTokenProcedure = "/buf.alpha.registry.v1alpha1.SCIMTokenService/CreateSCIMToken"
	// SCIMTokenServiceListSCIMTokensProcedure is the fully-qualified name of the SCIMTokenService's
	// ListSCIMTokens RPC.
	SCIMTokenServiceListSCIMTokensProcedure = "/buf.alpha.registry.v1alpha1.SCIMTokenService/ListSCIMTokens"
	// SCIMTokenServiceDeleteSCIMTokenProcedure is the fully-qualified name of the SCIMTokenService's
	// DeleteSCIMToken RPC.
	SCIMTokenServiceDeleteSCIMTokenProcedure = "/buf.alpha.registry.v1alpha1.SCIMTokenService/DeleteSCIMToken"
)

// SCIMTokenServiceClient is a client for the buf.alpha.registry.v1alpha1.SCIMTokenService service.
type SCIMTokenServiceClient interface {
	// CreateToken creates a new token suitable for authentication to the SCIM API.
	//
	// This method requires authentication.
	CreateSCIMToken(context.Context, *connect_go.Request[v1alpha1.CreateSCIMTokenRequest]) (*connect_go.Response[v1alpha1.CreateSCIMTokenResponse], error)
	// ListTokens lists all active SCIM tokens.
	//
	// This method requires authentication.
	ListSCIMTokens(context.Context, *connect_go.Request[v1alpha1.ListSCIMTokensRequest]) (*connect_go.Response[v1alpha1.ListSCIMTokensResponse], error)
	// DeleteToken deletes an existing token.
	//
	// This method requires authentication.
	DeleteSCIMToken(context.Context, *connect_go.Request[v1alpha1.DeleteSCIMTokenRequest]) (*connect_go.Response[v1alpha1.DeleteSCIMTokenResponse], error)
}

// NewSCIMTokenServiceClient constructs a client for the
// buf.alpha.registry.v1alpha1.SCIMTokenService service. By default, it uses the Connect protocol
// with the binary Protobuf Codec, asks for gzipped responses, and sends uncompressed requests. To
// use the gRPC or gRPC-Web protocols, supply the connect.WithGRPC() or connect.WithGRPCWeb()
// options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewSCIMTokenServiceClient(httpClient connect_go.HTTPClient, baseURL string, opts ...connect_go.ClientOption) SCIMTokenServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &sCIMTokenServiceClient{
		createSCIMToken: connect_go.NewClient[v1alpha1.CreateSCIMTokenRequest, v1alpha1.CreateSCIMTokenResponse](
			httpClient,
			baseURL+SCIMTokenServiceCreateSCIMTokenProcedure,
			opts...,
		),
		listSCIMTokens: connect_go.NewClient[v1alpha1.ListSCIMTokensRequest, v1alpha1.ListSCIMTokensResponse](
			httpClient,
			baseURL+SCIMTokenServiceListSCIMTokensProcedure,
			opts...,
		),
		deleteSCIMToken: connect_go.NewClient[v1alpha1.DeleteSCIMTokenRequest, v1alpha1.DeleteSCIMTokenResponse](
			httpClient,
			baseURL+SCIMTokenServiceDeleteSCIMTokenProcedure,
			opts...,
		),
	}
}

// sCIMTokenServiceClient implements SCIMTokenServiceClient.
type sCIMTokenServiceClient struct {
	createSCIMToken *connect_go.Client[v1alpha1.CreateSCIMTokenRequest, v1alpha1.CreateSCIMTokenResponse]
	listSCIMTokens  *connect_go.Client[v1alpha1.ListSCIMTokensRequest, v1alpha1.ListSCIMTokensResponse]
	deleteSCIMToken *connect_go.Client[v1alpha1.DeleteSCIMTokenRequest, v1alpha1.DeleteSCIMTokenResponse]
}

// CreateSCIMToken calls buf.alpha.registry.v1alpha1.SCIMTokenService.CreateSCIMToken.
func (c *sCIMTokenServiceClient) CreateSCIMToken(ctx context.Context, req *connect_go.Request[v1alpha1.CreateSCIMTokenRequest]) (*connect_go.Response[v1alpha1.CreateSCIMTokenResponse], error) {
	return c.createSCIMToken.CallUnary(ctx, req)
}

// ListSCIMTokens calls buf.alpha.registry.v1alpha1.SCIMTokenService.ListSCIMTokens.
func (c *sCIMTokenServiceClient) ListSCIMTokens(ctx context.Context, req *connect_go.Request[v1alpha1.ListSCIMTokensRequest]) (*connect_go.Response[v1alpha1.ListSCIMTokensResponse], error) {
	return c.listSCIMTokens.CallUnary(ctx, req)
}

// DeleteSCIMToken calls buf.alpha.registry.v1alpha1.SCIMTokenService.DeleteSCIMToken.
func (c *sCIMTokenServiceClient) DeleteSCIMToken(ctx context.Context, req *connect_go.Request[v1alpha1.DeleteSCIMTokenRequest]) (*connect_go.Response[v1alpha1.DeleteSCIMTokenResponse], error) {
	return c.deleteSCIMToken.CallUnary(ctx, req)
}

// SCIMTokenServiceHandler is an implementation of the buf.alpha.registry.v1alpha1.SCIMTokenService
// service.
type SCIMTokenServiceHandler interface {
	// CreateToken creates a new token suitable for authentication to the SCIM API.
	//
	// This method requires authentication.
	CreateSCIMToken(context.Context, *connect_go.Request[v1alpha1.CreateSCIMTokenRequest]) (*connect_go.Response[v1alpha1.CreateSCIMTokenResponse], error)
	// ListTokens lists all active SCIM tokens.
	//
	// This method requires authentication.
	ListSCIMTokens(context.Context, *connect_go.Request[v1alpha1.ListSCIMTokensRequest]) (*connect_go.Response[v1alpha1.ListSCIMTokensResponse], error)
	// DeleteToken deletes an existing token.
	//
	// This method requires authentication.
	DeleteSCIMToken(context.Context, *connect_go.Request[v1alpha1.DeleteSCIMTokenRequest]) (*connect_go.Response[v1alpha1.DeleteSCIMTokenResponse], error)
}

// NewSCIMTokenServiceHandler builds an HTTP handler from the service implementation. It returns the
// path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewSCIMTokenServiceHandler(svc SCIMTokenServiceHandler, opts ...connect_go.HandlerOption) (string, http.Handler) {
	mux := http.NewServeMux()
	mux.Handle(SCIMTokenServiceCreateSCIMTokenProcedure, connect_go.NewUnaryHandler(
		SCIMTokenServiceCreateSCIMTokenProcedure,
		svc.CreateSCIMToken,
		opts...,
	))
	mux.Handle(SCIMTokenServiceListSCIMTokensProcedure, connect_go.NewUnaryHandler(
		SCIMTokenServiceListSCIMTokensProcedure,
		svc.ListSCIMTokens,
		opts...,
	))
	mux.Handle(SCIMTokenServiceDeleteSCIMTokenProcedure, connect_go.NewUnaryHandler(
		SCIMTokenServiceDeleteSCIMTokenProcedure,
		svc.DeleteSCIMToken,
		opts...,
	))
	return "/buf.alpha.registry.v1alpha1.SCIMTokenService/", mux
}

// UnimplementedSCIMTokenServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedSCIMTokenServiceHandler struct{}

func (UnimplementedSCIMTokenServiceHandler) CreateSCIMToken(context.Context, *connect_go.Request[v1alpha1.CreateSCIMTokenRequest]) (*connect_go.Response[v1alpha1.CreateSCIMTokenResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.SCIMTokenService.CreateSCIMToken is not implemented"))
}

func (UnimplementedSCIMTokenServiceHandler) ListSCIMTokens(context.Context, *connect_go.Request[v1alpha1.ListSCIMTokensRequest]) (*connect_go.Response[v1alpha1.ListSCIMTokensResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.SCIMTokenService.ListSCIMTokens is not implemented"))
}

func (UnimplementedSCIMTokenServiceHandler) DeleteSCIMToken(context.Context, *connect_go.Request[v1alpha1.DeleteSCIMTokenRequest]) (*connect_go.Response[v1alpha1.DeleteSCIMTokenResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.SCIMTokenService.DeleteSCIMToken is not implemented"))
}
