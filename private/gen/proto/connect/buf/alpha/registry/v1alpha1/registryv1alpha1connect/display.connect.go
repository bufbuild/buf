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
// Source: buf/alpha/registry/v1alpha1/display.proto

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
	// DisplayServiceName is the fully-qualified name of the DisplayService service.
	DisplayServiceName = "buf.alpha.registry.v1alpha1.DisplayService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// DisplayServiceDisplayOrganizationElementsProcedure is the fully-qualified name of the
	// DisplayService's DisplayOrganizationElements RPC.
	DisplayServiceDisplayOrganizationElementsProcedure = "/buf.alpha.registry.v1alpha1.DisplayService/DisplayOrganizationElements"
	// DisplayServiceDisplayRepositoryElementsProcedure is the fully-qualified name of the
	// DisplayService's DisplayRepositoryElements RPC.
	DisplayServiceDisplayRepositoryElementsProcedure = "/buf.alpha.registry.v1alpha1.DisplayService/DisplayRepositoryElements"
	// DisplayServiceDisplayPluginElementsProcedure is the fully-qualified name of the DisplayService's
	// DisplayPluginElements RPC.
	DisplayServiceDisplayPluginElementsProcedure = "/buf.alpha.registry.v1alpha1.DisplayService/DisplayPluginElements"
	// DisplayServiceDisplayUserElementsProcedure is the fully-qualified name of the DisplayService's
	// DisplayUserElements RPC.
	DisplayServiceDisplayUserElementsProcedure = "/buf.alpha.registry.v1alpha1.DisplayService/DisplayUserElements"
	// DisplayServiceDisplayServerElementsProcedure is the fully-qualified name of the DisplayService's
	// DisplayServerElements RPC.
	DisplayServiceDisplayServerElementsProcedure = "/buf.alpha.registry.v1alpha1.DisplayService/DisplayServerElements"
	// DisplayServiceDisplayOwnerEntitledElementsProcedure is the fully-qualified name of the
	// DisplayService's DisplayOwnerEntitledElements RPC.
	DisplayServiceDisplayOwnerEntitledElementsProcedure = "/buf.alpha.registry.v1alpha1.DisplayService/DisplayOwnerEntitledElements"
	// DisplayServiceDisplayRepositoryEntitledElementsProcedure is the fully-qualified name of the
	// DisplayService's DisplayRepositoryEntitledElements RPC.
	DisplayServiceDisplayRepositoryEntitledElementsProcedure = "/buf.alpha.registry.v1alpha1.DisplayService/DisplayRepositoryEntitledElements"
	// DisplayServiceListManageableRepositoryRolesProcedure is the fully-qualified name of the
	// DisplayService's ListManageableRepositoryRoles RPC.
	DisplayServiceListManageableRepositoryRolesProcedure = "/buf.alpha.registry.v1alpha1.DisplayService/ListManageableRepositoryRoles"
	// DisplayServiceListManageableUserRepositoryRolesProcedure is the fully-qualified name of the
	// DisplayService's ListManageableUserRepositoryRoles RPC.
	DisplayServiceListManageableUserRepositoryRolesProcedure = "/buf.alpha.registry.v1alpha1.DisplayService/ListManageableUserRepositoryRoles"
)

// DisplayServiceClient is a client for the buf.alpha.registry.v1alpha1.DisplayService service.
type DisplayServiceClient interface {
	// DisplayOrganizationElements returns which organization elements should be displayed to the user.
	DisplayOrganizationElements(context.Context, *connect.Request[v1alpha1.DisplayOrganizationElementsRequest]) (*connect.Response[v1alpha1.DisplayOrganizationElementsResponse], error)
	// DisplayRepositoryElements returns which repository elements should be displayed to the user.
	DisplayRepositoryElements(context.Context, *connect.Request[v1alpha1.DisplayRepositoryElementsRequest]) (*connect.Response[v1alpha1.DisplayRepositoryElementsResponse], error)
	// DisplayPluginElements returns which plugin elements should be displayed to the user.
	DisplayPluginElements(context.Context, *connect.Request[v1alpha1.DisplayPluginElementsRequest]) (*connect.Response[v1alpha1.DisplayPluginElementsResponse], error)
	// DisplayUserElements returns which user elements should be displayed to the user.
	DisplayUserElements(context.Context, *connect.Request[v1alpha1.DisplayUserElementsRequest]) (*connect.Response[v1alpha1.DisplayUserElementsResponse], error)
	// DisplayServerElements returns which server elements should be displayed to the user.
	DisplayServerElements(context.Context, *connect.Request[v1alpha1.DisplayServerElementsRequest]) (*connect.Response[v1alpha1.DisplayServerElementsResponse], error)
	// DisplayOwnerEntitledElements returns which owner elements are entitled to be displayed to the user.
	DisplayOwnerEntitledElements(context.Context, *connect.Request[v1alpha1.DisplayOwnerEntitledElementsRequest]) (*connect.Response[v1alpha1.DisplayOwnerEntitledElementsResponse], error)
	// DisplayRepositoryEntitledElements returns which repository elements are entitled to be displayed to the user.
	DisplayRepositoryEntitledElements(context.Context, *connect.Request[v1alpha1.DisplayRepositoryEntitledElementsRequest]) (*connect.Response[v1alpha1.DisplayRepositoryEntitledElementsResponse], error)
	// ListManageableRepositoryRoles returns which roles should be displayed
	// to the user when they are managing contributors on the repository.
	ListManageableRepositoryRoles(context.Context, *connect.Request[v1alpha1.ListManageableRepositoryRolesRequest]) (*connect.Response[v1alpha1.ListManageableRepositoryRolesResponse], error)
	// ListManageableUserRepositoryRoles returns which roles should be displayed
	// to the user when they are managing a specific contributor on the repository.
	ListManageableUserRepositoryRoles(context.Context, *connect.Request[v1alpha1.ListManageableUserRepositoryRolesRequest]) (*connect.Response[v1alpha1.ListManageableUserRepositoryRolesResponse], error)
}

// NewDisplayServiceClient constructs a client for the buf.alpha.registry.v1alpha1.DisplayService
// service. By default, it uses the Connect protocol with the binary Protobuf Codec, asks for
// gzipped responses, and sends uncompressed requests. To use the gRPC or gRPC-Web protocols, supply
// the connect.WithGRPC() or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewDisplayServiceClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) DisplayServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	displayServiceMethods := v1alpha1.File_buf_alpha_registry_v1alpha1_display_proto.Services().ByName("DisplayService").Methods()
	return &displayServiceClient{
		displayOrganizationElements: connect.NewClient[v1alpha1.DisplayOrganizationElementsRequest, v1alpha1.DisplayOrganizationElementsResponse](
			httpClient,
			baseURL+DisplayServiceDisplayOrganizationElementsProcedure,
			connect.WithSchema(displayServiceMethods.ByName("DisplayOrganizationElements")),
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
		displayRepositoryElements: connect.NewClient[v1alpha1.DisplayRepositoryElementsRequest, v1alpha1.DisplayRepositoryElementsResponse](
			httpClient,
			baseURL+DisplayServiceDisplayRepositoryElementsProcedure,
			connect.WithSchema(displayServiceMethods.ByName("DisplayRepositoryElements")),
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
		displayPluginElements: connect.NewClient[v1alpha1.DisplayPluginElementsRequest, v1alpha1.DisplayPluginElementsResponse](
			httpClient,
			baseURL+DisplayServiceDisplayPluginElementsProcedure,
			connect.WithSchema(displayServiceMethods.ByName("DisplayPluginElements")),
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
		displayUserElements: connect.NewClient[v1alpha1.DisplayUserElementsRequest, v1alpha1.DisplayUserElementsResponse](
			httpClient,
			baseURL+DisplayServiceDisplayUserElementsProcedure,
			connect.WithSchema(displayServiceMethods.ByName("DisplayUserElements")),
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
		displayServerElements: connect.NewClient[v1alpha1.DisplayServerElementsRequest, v1alpha1.DisplayServerElementsResponse](
			httpClient,
			baseURL+DisplayServiceDisplayServerElementsProcedure,
			connect.WithSchema(displayServiceMethods.ByName("DisplayServerElements")),
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
		displayOwnerEntitledElements: connect.NewClient[v1alpha1.DisplayOwnerEntitledElementsRequest, v1alpha1.DisplayOwnerEntitledElementsResponse](
			httpClient,
			baseURL+DisplayServiceDisplayOwnerEntitledElementsProcedure,
			connect.WithSchema(displayServiceMethods.ByName("DisplayOwnerEntitledElements")),
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
		displayRepositoryEntitledElements: connect.NewClient[v1alpha1.DisplayRepositoryEntitledElementsRequest, v1alpha1.DisplayRepositoryEntitledElementsResponse](
			httpClient,
			baseURL+DisplayServiceDisplayRepositoryEntitledElementsProcedure,
			connect.WithSchema(displayServiceMethods.ByName("DisplayRepositoryEntitledElements")),
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
		listManageableRepositoryRoles: connect.NewClient[v1alpha1.ListManageableRepositoryRolesRequest, v1alpha1.ListManageableRepositoryRolesResponse](
			httpClient,
			baseURL+DisplayServiceListManageableRepositoryRolesProcedure,
			connect.WithSchema(displayServiceMethods.ByName("ListManageableRepositoryRoles")),
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
		listManageableUserRepositoryRoles: connect.NewClient[v1alpha1.ListManageableUserRepositoryRolesRequest, v1alpha1.ListManageableUserRepositoryRolesResponse](
			httpClient,
			baseURL+DisplayServiceListManageableUserRepositoryRolesProcedure,
			connect.WithSchema(displayServiceMethods.ByName("ListManageableUserRepositoryRoles")),
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
	}
}

// displayServiceClient implements DisplayServiceClient.
type displayServiceClient struct {
	displayOrganizationElements       *connect.Client[v1alpha1.DisplayOrganizationElementsRequest, v1alpha1.DisplayOrganizationElementsResponse]
	displayRepositoryElements         *connect.Client[v1alpha1.DisplayRepositoryElementsRequest, v1alpha1.DisplayRepositoryElementsResponse]
	displayPluginElements             *connect.Client[v1alpha1.DisplayPluginElementsRequest, v1alpha1.DisplayPluginElementsResponse]
	displayUserElements               *connect.Client[v1alpha1.DisplayUserElementsRequest, v1alpha1.DisplayUserElementsResponse]
	displayServerElements             *connect.Client[v1alpha1.DisplayServerElementsRequest, v1alpha1.DisplayServerElementsResponse]
	displayOwnerEntitledElements      *connect.Client[v1alpha1.DisplayOwnerEntitledElementsRequest, v1alpha1.DisplayOwnerEntitledElementsResponse]
	displayRepositoryEntitledElements *connect.Client[v1alpha1.DisplayRepositoryEntitledElementsRequest, v1alpha1.DisplayRepositoryEntitledElementsResponse]
	listManageableRepositoryRoles     *connect.Client[v1alpha1.ListManageableRepositoryRolesRequest, v1alpha1.ListManageableRepositoryRolesResponse]
	listManageableUserRepositoryRoles *connect.Client[v1alpha1.ListManageableUserRepositoryRolesRequest, v1alpha1.ListManageableUserRepositoryRolesResponse]
}

// DisplayOrganizationElements calls
// buf.alpha.registry.v1alpha1.DisplayService.DisplayOrganizationElements.
func (c *displayServiceClient) DisplayOrganizationElements(ctx context.Context, req *connect.Request[v1alpha1.DisplayOrganizationElementsRequest]) (*connect.Response[v1alpha1.DisplayOrganizationElementsResponse], error) {
	return c.displayOrganizationElements.CallUnary(ctx, req)
}

// DisplayRepositoryElements calls
// buf.alpha.registry.v1alpha1.DisplayService.DisplayRepositoryElements.
func (c *displayServiceClient) DisplayRepositoryElements(ctx context.Context, req *connect.Request[v1alpha1.DisplayRepositoryElementsRequest]) (*connect.Response[v1alpha1.DisplayRepositoryElementsResponse], error) {
	return c.displayRepositoryElements.CallUnary(ctx, req)
}

// DisplayPluginElements calls buf.alpha.registry.v1alpha1.DisplayService.DisplayPluginElements.
func (c *displayServiceClient) DisplayPluginElements(ctx context.Context, req *connect.Request[v1alpha1.DisplayPluginElementsRequest]) (*connect.Response[v1alpha1.DisplayPluginElementsResponse], error) {
	return c.displayPluginElements.CallUnary(ctx, req)
}

// DisplayUserElements calls buf.alpha.registry.v1alpha1.DisplayService.DisplayUserElements.
func (c *displayServiceClient) DisplayUserElements(ctx context.Context, req *connect.Request[v1alpha1.DisplayUserElementsRequest]) (*connect.Response[v1alpha1.DisplayUserElementsResponse], error) {
	return c.displayUserElements.CallUnary(ctx, req)
}

// DisplayServerElements calls buf.alpha.registry.v1alpha1.DisplayService.DisplayServerElements.
func (c *displayServiceClient) DisplayServerElements(ctx context.Context, req *connect.Request[v1alpha1.DisplayServerElementsRequest]) (*connect.Response[v1alpha1.DisplayServerElementsResponse], error) {
	return c.displayServerElements.CallUnary(ctx, req)
}

// DisplayOwnerEntitledElements calls
// buf.alpha.registry.v1alpha1.DisplayService.DisplayOwnerEntitledElements.
func (c *displayServiceClient) DisplayOwnerEntitledElements(ctx context.Context, req *connect.Request[v1alpha1.DisplayOwnerEntitledElementsRequest]) (*connect.Response[v1alpha1.DisplayOwnerEntitledElementsResponse], error) {
	return c.displayOwnerEntitledElements.CallUnary(ctx, req)
}

// DisplayRepositoryEntitledElements calls
// buf.alpha.registry.v1alpha1.DisplayService.DisplayRepositoryEntitledElements.
func (c *displayServiceClient) DisplayRepositoryEntitledElements(ctx context.Context, req *connect.Request[v1alpha1.DisplayRepositoryEntitledElementsRequest]) (*connect.Response[v1alpha1.DisplayRepositoryEntitledElementsResponse], error) {
	return c.displayRepositoryEntitledElements.CallUnary(ctx, req)
}

// ListManageableRepositoryRoles calls
// buf.alpha.registry.v1alpha1.DisplayService.ListManageableRepositoryRoles.
func (c *displayServiceClient) ListManageableRepositoryRoles(ctx context.Context, req *connect.Request[v1alpha1.ListManageableRepositoryRolesRequest]) (*connect.Response[v1alpha1.ListManageableRepositoryRolesResponse], error) {
	return c.listManageableRepositoryRoles.CallUnary(ctx, req)
}

// ListManageableUserRepositoryRoles calls
// buf.alpha.registry.v1alpha1.DisplayService.ListManageableUserRepositoryRoles.
func (c *displayServiceClient) ListManageableUserRepositoryRoles(ctx context.Context, req *connect.Request[v1alpha1.ListManageableUserRepositoryRolesRequest]) (*connect.Response[v1alpha1.ListManageableUserRepositoryRolesResponse], error) {
	return c.listManageableUserRepositoryRoles.CallUnary(ctx, req)
}

// DisplayServiceHandler is an implementation of the buf.alpha.registry.v1alpha1.DisplayService
// service.
type DisplayServiceHandler interface {
	// DisplayOrganizationElements returns which organization elements should be displayed to the user.
	DisplayOrganizationElements(context.Context, *connect.Request[v1alpha1.DisplayOrganizationElementsRequest]) (*connect.Response[v1alpha1.DisplayOrganizationElementsResponse], error)
	// DisplayRepositoryElements returns which repository elements should be displayed to the user.
	DisplayRepositoryElements(context.Context, *connect.Request[v1alpha1.DisplayRepositoryElementsRequest]) (*connect.Response[v1alpha1.DisplayRepositoryElementsResponse], error)
	// DisplayPluginElements returns which plugin elements should be displayed to the user.
	DisplayPluginElements(context.Context, *connect.Request[v1alpha1.DisplayPluginElementsRequest]) (*connect.Response[v1alpha1.DisplayPluginElementsResponse], error)
	// DisplayUserElements returns which user elements should be displayed to the user.
	DisplayUserElements(context.Context, *connect.Request[v1alpha1.DisplayUserElementsRequest]) (*connect.Response[v1alpha1.DisplayUserElementsResponse], error)
	// DisplayServerElements returns which server elements should be displayed to the user.
	DisplayServerElements(context.Context, *connect.Request[v1alpha1.DisplayServerElementsRequest]) (*connect.Response[v1alpha1.DisplayServerElementsResponse], error)
	// DisplayOwnerEntitledElements returns which owner elements are entitled to be displayed to the user.
	DisplayOwnerEntitledElements(context.Context, *connect.Request[v1alpha1.DisplayOwnerEntitledElementsRequest]) (*connect.Response[v1alpha1.DisplayOwnerEntitledElementsResponse], error)
	// DisplayRepositoryEntitledElements returns which repository elements are entitled to be displayed to the user.
	DisplayRepositoryEntitledElements(context.Context, *connect.Request[v1alpha1.DisplayRepositoryEntitledElementsRequest]) (*connect.Response[v1alpha1.DisplayRepositoryEntitledElementsResponse], error)
	// ListManageableRepositoryRoles returns which roles should be displayed
	// to the user when they are managing contributors on the repository.
	ListManageableRepositoryRoles(context.Context, *connect.Request[v1alpha1.ListManageableRepositoryRolesRequest]) (*connect.Response[v1alpha1.ListManageableRepositoryRolesResponse], error)
	// ListManageableUserRepositoryRoles returns which roles should be displayed
	// to the user when they are managing a specific contributor on the repository.
	ListManageableUserRepositoryRoles(context.Context, *connect.Request[v1alpha1.ListManageableUserRepositoryRolesRequest]) (*connect.Response[v1alpha1.ListManageableUserRepositoryRolesResponse], error)
}

// NewDisplayServiceHandler builds an HTTP handler from the service implementation. It returns the
// path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewDisplayServiceHandler(svc DisplayServiceHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	displayServiceMethods := v1alpha1.File_buf_alpha_registry_v1alpha1_display_proto.Services().ByName("DisplayService").Methods()
	displayServiceDisplayOrganizationElementsHandler := connect.NewUnaryHandler(
		DisplayServiceDisplayOrganizationElementsProcedure,
		svc.DisplayOrganizationElements,
		connect.WithSchema(displayServiceMethods.ByName("DisplayOrganizationElements")),
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	displayServiceDisplayRepositoryElementsHandler := connect.NewUnaryHandler(
		DisplayServiceDisplayRepositoryElementsProcedure,
		svc.DisplayRepositoryElements,
		connect.WithSchema(displayServiceMethods.ByName("DisplayRepositoryElements")),
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	displayServiceDisplayPluginElementsHandler := connect.NewUnaryHandler(
		DisplayServiceDisplayPluginElementsProcedure,
		svc.DisplayPluginElements,
		connect.WithSchema(displayServiceMethods.ByName("DisplayPluginElements")),
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	displayServiceDisplayUserElementsHandler := connect.NewUnaryHandler(
		DisplayServiceDisplayUserElementsProcedure,
		svc.DisplayUserElements,
		connect.WithSchema(displayServiceMethods.ByName("DisplayUserElements")),
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	displayServiceDisplayServerElementsHandler := connect.NewUnaryHandler(
		DisplayServiceDisplayServerElementsProcedure,
		svc.DisplayServerElements,
		connect.WithSchema(displayServiceMethods.ByName("DisplayServerElements")),
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	displayServiceDisplayOwnerEntitledElementsHandler := connect.NewUnaryHandler(
		DisplayServiceDisplayOwnerEntitledElementsProcedure,
		svc.DisplayOwnerEntitledElements,
		connect.WithSchema(displayServiceMethods.ByName("DisplayOwnerEntitledElements")),
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	displayServiceDisplayRepositoryEntitledElementsHandler := connect.NewUnaryHandler(
		DisplayServiceDisplayRepositoryEntitledElementsProcedure,
		svc.DisplayRepositoryEntitledElements,
		connect.WithSchema(displayServiceMethods.ByName("DisplayRepositoryEntitledElements")),
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	displayServiceListManageableRepositoryRolesHandler := connect.NewUnaryHandler(
		DisplayServiceListManageableRepositoryRolesProcedure,
		svc.ListManageableRepositoryRoles,
		connect.WithSchema(displayServiceMethods.ByName("ListManageableRepositoryRoles")),
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	displayServiceListManageableUserRepositoryRolesHandler := connect.NewUnaryHandler(
		DisplayServiceListManageableUserRepositoryRolesProcedure,
		svc.ListManageableUserRepositoryRoles,
		connect.WithSchema(displayServiceMethods.ByName("ListManageableUserRepositoryRoles")),
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	return "/buf.alpha.registry.v1alpha1.DisplayService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case DisplayServiceDisplayOrganizationElementsProcedure:
			displayServiceDisplayOrganizationElementsHandler.ServeHTTP(w, r)
		case DisplayServiceDisplayRepositoryElementsProcedure:
			displayServiceDisplayRepositoryElementsHandler.ServeHTTP(w, r)
		case DisplayServiceDisplayPluginElementsProcedure:
			displayServiceDisplayPluginElementsHandler.ServeHTTP(w, r)
		case DisplayServiceDisplayUserElementsProcedure:
			displayServiceDisplayUserElementsHandler.ServeHTTP(w, r)
		case DisplayServiceDisplayServerElementsProcedure:
			displayServiceDisplayServerElementsHandler.ServeHTTP(w, r)
		case DisplayServiceDisplayOwnerEntitledElementsProcedure:
			displayServiceDisplayOwnerEntitledElementsHandler.ServeHTTP(w, r)
		case DisplayServiceDisplayRepositoryEntitledElementsProcedure:
			displayServiceDisplayRepositoryEntitledElementsHandler.ServeHTTP(w, r)
		case DisplayServiceListManageableRepositoryRolesProcedure:
			displayServiceListManageableRepositoryRolesHandler.ServeHTTP(w, r)
		case DisplayServiceListManageableUserRepositoryRolesProcedure:
			displayServiceListManageableUserRepositoryRolesHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedDisplayServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedDisplayServiceHandler struct{}

func (UnimplementedDisplayServiceHandler) DisplayOrganizationElements(context.Context, *connect.Request[v1alpha1.DisplayOrganizationElementsRequest]) (*connect.Response[v1alpha1.DisplayOrganizationElementsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.DisplayService.DisplayOrganizationElements is not implemented"))
}

func (UnimplementedDisplayServiceHandler) DisplayRepositoryElements(context.Context, *connect.Request[v1alpha1.DisplayRepositoryElementsRequest]) (*connect.Response[v1alpha1.DisplayRepositoryElementsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.DisplayService.DisplayRepositoryElements is not implemented"))
}

func (UnimplementedDisplayServiceHandler) DisplayPluginElements(context.Context, *connect.Request[v1alpha1.DisplayPluginElementsRequest]) (*connect.Response[v1alpha1.DisplayPluginElementsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.DisplayService.DisplayPluginElements is not implemented"))
}

func (UnimplementedDisplayServiceHandler) DisplayUserElements(context.Context, *connect.Request[v1alpha1.DisplayUserElementsRequest]) (*connect.Response[v1alpha1.DisplayUserElementsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.DisplayService.DisplayUserElements is not implemented"))
}

func (UnimplementedDisplayServiceHandler) DisplayServerElements(context.Context, *connect.Request[v1alpha1.DisplayServerElementsRequest]) (*connect.Response[v1alpha1.DisplayServerElementsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.DisplayService.DisplayServerElements is not implemented"))
}

func (UnimplementedDisplayServiceHandler) DisplayOwnerEntitledElements(context.Context, *connect.Request[v1alpha1.DisplayOwnerEntitledElementsRequest]) (*connect.Response[v1alpha1.DisplayOwnerEntitledElementsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.DisplayService.DisplayOwnerEntitledElements is not implemented"))
}

func (UnimplementedDisplayServiceHandler) DisplayRepositoryEntitledElements(context.Context, *connect.Request[v1alpha1.DisplayRepositoryEntitledElementsRequest]) (*connect.Response[v1alpha1.DisplayRepositoryEntitledElementsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.DisplayService.DisplayRepositoryEntitledElements is not implemented"))
}

func (UnimplementedDisplayServiceHandler) ListManageableRepositoryRoles(context.Context, *connect.Request[v1alpha1.ListManageableRepositoryRolesRequest]) (*connect.Response[v1alpha1.ListManageableRepositoryRolesResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.DisplayService.ListManageableRepositoryRoles is not implemented"))
}

func (UnimplementedDisplayServiceHandler) ListManageableUserRepositoryRoles(context.Context, *connect.Request[v1alpha1.ListManageableUserRepositoryRolesRequest]) (*connect.Response[v1alpha1.ListManageableUserRepositoryRolesResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.DisplayService.ListManageableUserRepositoryRoles is not implemented"))
}
