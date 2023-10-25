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
// Source: buf/registry/module/v1beta1/module_service.proto

package modulev1beta1connect

import (
	connect "connectrpc.com/connect"
	context "context"
	errors "errors"
	v1beta1 "github.com/bufbuild/buf/private/gen/proto/go/buf/registry/module/v1beta1"
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
	// ModuleServiceName is the fully-qualified name of the ModuleService service.
	ModuleServiceName = "buf.registry.module.v1beta1.ModuleService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// ModuleServiceGetModulesProcedure is the fully-qualified name of the ModuleService's GetModules
	// RPC.
	ModuleServiceGetModulesProcedure = "/buf.registry.module.v1beta1.ModuleService/GetModules"
	// ModuleServiceListModulesProcedure is the fully-qualified name of the ModuleService's ListModules
	// RPC.
	ModuleServiceListModulesProcedure = "/buf.registry.module.v1beta1.ModuleService/ListModules"
	// ModuleServiceCreateModulesProcedure is the fully-qualified name of the ModuleService's
	// CreateModules RPC.
	ModuleServiceCreateModulesProcedure = "/buf.registry.module.v1beta1.ModuleService/CreateModules"
	// ModuleServiceUpdateModulesProcedure is the fully-qualified name of the ModuleService's
	// UpdateModules RPC.
	ModuleServiceUpdateModulesProcedure = "/buf.registry.module.v1beta1.ModuleService/UpdateModules"
	// ModuleServiceDeleteModulesProcedure is the fully-qualified name of the ModuleService's
	// DeleteModules RPC.
	ModuleServiceDeleteModulesProcedure = "/buf.registry.module.v1beta1.ModuleService/DeleteModules"
)

// ModuleServiceClient is a client for the buf.registry.module.v1beta1.ModuleService service.
type ModuleServiceClient interface {
	// Get Modules by id or name.
	GetModules(context.Context, *connect.Request[v1beta1.GetModulesRequest]) (*connect.Response[v1beta1.GetModulesResponse], error)
	// List Modules, usually for a specific User or Organization.
	ListModules(context.Context, *connect.Request[v1beta1.ListModulesRequest]) (*connect.Response[v1beta1.ListModulesResponse], error)
	// Create new Modules.
	//
	// When a Module is created, a Branch representing the release Branch
	// is created as well.
	CreateModules(context.Context, *connect.Request[v1beta1.CreateModulesRequest]) (*connect.Response[v1beta1.CreateModulesResponse], error)
	// Update existing Modules.
	UpdateModules(context.Context, *connect.Request[v1beta1.UpdateModulesRequest]) (*connect.Response[v1beta1.UpdateModulesResponse], error)
	// Delete existing Modules.
	DeleteModules(context.Context, *connect.Request[v1beta1.DeleteModulesRequest]) (*connect.Response[v1beta1.DeleteModulesResponse], error)
}

// NewModuleServiceClient constructs a client for the buf.registry.module.v1beta1.ModuleService
// service. By default, it uses the Connect protocol with the binary Protobuf Codec, asks for
// gzipped responses, and sends uncompressed requests. To use the gRPC or gRPC-Web protocols, supply
// the connect.WithGRPC() or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewModuleServiceClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) ModuleServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &moduleServiceClient{
		getModules: connect.NewClient[v1beta1.GetModulesRequest, v1beta1.GetModulesResponse](
			httpClient,
			baseURL+ModuleServiceGetModulesProcedure,
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
		listModules: connect.NewClient[v1beta1.ListModulesRequest, v1beta1.ListModulesResponse](
			httpClient,
			baseURL+ModuleServiceListModulesProcedure,
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
		createModules: connect.NewClient[v1beta1.CreateModulesRequest, v1beta1.CreateModulesResponse](
			httpClient,
			baseURL+ModuleServiceCreateModulesProcedure,
			connect.WithIdempotency(connect.IdempotencyIdempotent),
			connect.WithClientOptions(opts...),
		),
		updateModules: connect.NewClient[v1beta1.UpdateModulesRequest, v1beta1.UpdateModulesResponse](
			httpClient,
			baseURL+ModuleServiceUpdateModulesProcedure,
			connect.WithIdempotency(connect.IdempotencyIdempotent),
			connect.WithClientOptions(opts...),
		),
		deleteModules: connect.NewClient[v1beta1.DeleteModulesRequest, v1beta1.DeleteModulesResponse](
			httpClient,
			baseURL+ModuleServiceDeleteModulesProcedure,
			connect.WithIdempotency(connect.IdempotencyIdempotent),
			connect.WithClientOptions(opts...),
		),
	}
}

// moduleServiceClient implements ModuleServiceClient.
type moduleServiceClient struct {
	getModules    *connect.Client[v1beta1.GetModulesRequest, v1beta1.GetModulesResponse]
	listModules   *connect.Client[v1beta1.ListModulesRequest, v1beta1.ListModulesResponse]
	createModules *connect.Client[v1beta1.CreateModulesRequest, v1beta1.CreateModulesResponse]
	updateModules *connect.Client[v1beta1.UpdateModulesRequest, v1beta1.UpdateModulesResponse]
	deleteModules *connect.Client[v1beta1.DeleteModulesRequest, v1beta1.DeleteModulesResponse]
}

// GetModules calls buf.registry.module.v1beta1.ModuleService.GetModules.
func (c *moduleServiceClient) GetModules(ctx context.Context, req *connect.Request[v1beta1.GetModulesRequest]) (*connect.Response[v1beta1.GetModulesResponse], error) {
	return c.getModules.CallUnary(ctx, req)
}

// ListModules calls buf.registry.module.v1beta1.ModuleService.ListModules.
func (c *moduleServiceClient) ListModules(ctx context.Context, req *connect.Request[v1beta1.ListModulesRequest]) (*connect.Response[v1beta1.ListModulesResponse], error) {
	return c.listModules.CallUnary(ctx, req)
}

// CreateModules calls buf.registry.module.v1beta1.ModuleService.CreateModules.
func (c *moduleServiceClient) CreateModules(ctx context.Context, req *connect.Request[v1beta1.CreateModulesRequest]) (*connect.Response[v1beta1.CreateModulesResponse], error) {
	return c.createModules.CallUnary(ctx, req)
}

// UpdateModules calls buf.registry.module.v1beta1.ModuleService.UpdateModules.
func (c *moduleServiceClient) UpdateModules(ctx context.Context, req *connect.Request[v1beta1.UpdateModulesRequest]) (*connect.Response[v1beta1.UpdateModulesResponse], error) {
	return c.updateModules.CallUnary(ctx, req)
}

// DeleteModules calls buf.registry.module.v1beta1.ModuleService.DeleteModules.
func (c *moduleServiceClient) DeleteModules(ctx context.Context, req *connect.Request[v1beta1.DeleteModulesRequest]) (*connect.Response[v1beta1.DeleteModulesResponse], error) {
	return c.deleteModules.CallUnary(ctx, req)
}

// ModuleServiceHandler is an implementation of the buf.registry.module.v1beta1.ModuleService
// service.
type ModuleServiceHandler interface {
	// Get Modules by id or name.
	GetModules(context.Context, *connect.Request[v1beta1.GetModulesRequest]) (*connect.Response[v1beta1.GetModulesResponse], error)
	// List Modules, usually for a specific User or Organization.
	ListModules(context.Context, *connect.Request[v1beta1.ListModulesRequest]) (*connect.Response[v1beta1.ListModulesResponse], error)
	// Create new Modules.
	//
	// When a Module is created, a Branch representing the release Branch
	// is created as well.
	CreateModules(context.Context, *connect.Request[v1beta1.CreateModulesRequest]) (*connect.Response[v1beta1.CreateModulesResponse], error)
	// Update existing Modules.
	UpdateModules(context.Context, *connect.Request[v1beta1.UpdateModulesRequest]) (*connect.Response[v1beta1.UpdateModulesResponse], error)
	// Delete existing Modules.
	DeleteModules(context.Context, *connect.Request[v1beta1.DeleteModulesRequest]) (*connect.Response[v1beta1.DeleteModulesResponse], error)
}

// NewModuleServiceHandler builds an HTTP handler from the service implementation. It returns the
// path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewModuleServiceHandler(svc ModuleServiceHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	moduleServiceGetModulesHandler := connect.NewUnaryHandler(
		ModuleServiceGetModulesProcedure,
		svc.GetModules,
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	moduleServiceListModulesHandler := connect.NewUnaryHandler(
		ModuleServiceListModulesProcedure,
		svc.ListModules,
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	moduleServiceCreateModulesHandler := connect.NewUnaryHandler(
		ModuleServiceCreateModulesProcedure,
		svc.CreateModules,
		connect.WithIdempotency(connect.IdempotencyIdempotent),
		connect.WithHandlerOptions(opts...),
	)
	moduleServiceUpdateModulesHandler := connect.NewUnaryHandler(
		ModuleServiceUpdateModulesProcedure,
		svc.UpdateModules,
		connect.WithIdempotency(connect.IdempotencyIdempotent),
		connect.WithHandlerOptions(opts...),
	)
	moduleServiceDeleteModulesHandler := connect.NewUnaryHandler(
		ModuleServiceDeleteModulesProcedure,
		svc.DeleteModules,
		connect.WithIdempotency(connect.IdempotencyIdempotent),
		connect.WithHandlerOptions(opts...),
	)
	return "/buf.registry.module.v1beta1.ModuleService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case ModuleServiceGetModulesProcedure:
			moduleServiceGetModulesHandler.ServeHTTP(w, r)
		case ModuleServiceListModulesProcedure:
			moduleServiceListModulesHandler.ServeHTTP(w, r)
		case ModuleServiceCreateModulesProcedure:
			moduleServiceCreateModulesHandler.ServeHTTP(w, r)
		case ModuleServiceUpdateModulesProcedure:
			moduleServiceUpdateModulesHandler.ServeHTTP(w, r)
		case ModuleServiceDeleteModulesProcedure:
			moduleServiceDeleteModulesHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedModuleServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedModuleServiceHandler struct{}

func (UnimplementedModuleServiceHandler) GetModules(context.Context, *connect.Request[v1beta1.GetModulesRequest]) (*connect.Response[v1beta1.GetModulesResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.registry.module.v1beta1.ModuleService.GetModules is not implemented"))
}

func (UnimplementedModuleServiceHandler) ListModules(context.Context, *connect.Request[v1beta1.ListModulesRequest]) (*connect.Response[v1beta1.ListModulesResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.registry.module.v1beta1.ModuleService.ListModules is not implemented"))
}

func (UnimplementedModuleServiceHandler) CreateModules(context.Context, *connect.Request[v1beta1.CreateModulesRequest]) (*connect.Response[v1beta1.CreateModulesResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.registry.module.v1beta1.ModuleService.CreateModules is not implemented"))
}

func (UnimplementedModuleServiceHandler) UpdateModules(context.Context, *connect.Request[v1beta1.UpdateModulesRequest]) (*connect.Response[v1beta1.UpdateModulesResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.registry.module.v1beta1.ModuleService.UpdateModules is not implemented"))
}

func (UnimplementedModuleServiceHandler) DeleteModules(context.Context, *connect.Request[v1beta1.DeleteModulesRequest]) (*connect.Response[v1beta1.DeleteModulesResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.registry.module.v1beta1.ModuleService.DeleteModules is not implemented"))
}
