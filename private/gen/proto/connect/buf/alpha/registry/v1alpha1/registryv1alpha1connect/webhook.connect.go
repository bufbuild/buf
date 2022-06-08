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
// Source: buf/alpha/registry/v1alpha1/webhook.proto

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
	// WebhookServiceName is the fully-qualified name of the WebhookService service.
	WebhookServiceName = "buf.alpha.registry.v1alpha1.WebhookService"
)

// WebhookServiceClient is a client for the buf.alpha.registry.v1alpha1.WebhookService service.
type WebhookServiceClient interface {
	CreateWebhook(context.Context, *connect_go.Request[v1alpha1.CreateWebhookRequest]) (*connect_go.Response[v1alpha1.CreateWebhookResponse], error)
	// Analogous APIs with likely very similar request/response structures.
	DeleteWebhook(context.Context, *connect_go.Request[v1alpha1.DeleteWebhookRequest]) (*connect_go.Response[v1alpha1.DeleteWebhookResponse], error)
	// Lists the subscriptions for a given repository. Will only return if the
	// the user has a role within the owner/repository.
	ListWebhooks(context.Context, *connect_go.Request[v1alpha1.ListWebhooksRequest]) (*connect_go.Response[v1alpha1.ListWebhooksResponse], error)
}

// NewWebhookServiceClient constructs a client for the buf.alpha.registry.v1alpha1.WebhookService
// service. By default, it uses the Connect protocol with the binary Protobuf Codec, asks for
// gzipped responses, and sends uncompressed requests. To use the gRPC or gRPC-Web protocols, supply
// the connect.WithGRPC() or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewWebhookServiceClient(httpClient connect_go.HTTPClient, baseURL string, opts ...connect_go.ClientOption) WebhookServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &webhookServiceClient{
		createWebhook: connect_go.NewClient[v1alpha1.CreateWebhookRequest, v1alpha1.CreateWebhookResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.WebhookService/CreateWebhook",
			opts...,
		),
		deleteWebhook: connect_go.NewClient[v1alpha1.DeleteWebhookRequest, v1alpha1.DeleteWebhookResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.WebhookService/DeleteWebhook",
			opts...,
		),
		listWebhooks: connect_go.NewClient[v1alpha1.ListWebhooksRequest, v1alpha1.ListWebhooksResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.WebhookService/ListWebhooks",
			opts...,
		),
	}
}

// webhookServiceClient implements WebhookServiceClient.
type webhookServiceClient struct {
	createWebhook *connect_go.Client[v1alpha1.CreateWebhookRequest, v1alpha1.CreateWebhookResponse]
	deleteWebhook *connect_go.Client[v1alpha1.DeleteWebhookRequest, v1alpha1.DeleteWebhookResponse]
	listWebhooks  *connect_go.Client[v1alpha1.ListWebhooksRequest, v1alpha1.ListWebhooksResponse]
}

// CreateWebhook calls buf.alpha.registry.v1alpha1.WebhookService.CreateWebhook.
func (c *webhookServiceClient) CreateWebhook(ctx context.Context, req *connect_go.Request[v1alpha1.CreateWebhookRequest]) (*connect_go.Response[v1alpha1.CreateWebhookResponse], error) {
	return c.createWebhook.CallUnary(ctx, req)
}

// DeleteWebhook calls buf.alpha.registry.v1alpha1.WebhookService.DeleteWebhook.
func (c *webhookServiceClient) DeleteWebhook(ctx context.Context, req *connect_go.Request[v1alpha1.DeleteWebhookRequest]) (*connect_go.Response[v1alpha1.DeleteWebhookResponse], error) {
	return c.deleteWebhook.CallUnary(ctx, req)
}

// ListWebhooks calls buf.alpha.registry.v1alpha1.WebhookService.ListWebhooks.
func (c *webhookServiceClient) ListWebhooks(ctx context.Context, req *connect_go.Request[v1alpha1.ListWebhooksRequest]) (*connect_go.Response[v1alpha1.ListWebhooksResponse], error) {
	return c.listWebhooks.CallUnary(ctx, req)
}

// WebhookServiceHandler is an implementation of the buf.alpha.registry.v1alpha1.WebhookService
// service.
type WebhookServiceHandler interface {
	CreateWebhook(context.Context, *connect_go.Request[v1alpha1.CreateWebhookRequest]) (*connect_go.Response[v1alpha1.CreateWebhookResponse], error)
	// Analogous APIs with likely very similar request/response structures.
	DeleteWebhook(context.Context, *connect_go.Request[v1alpha1.DeleteWebhookRequest]) (*connect_go.Response[v1alpha1.DeleteWebhookResponse], error)
	// Lists the subscriptions for a given repository. Will only return if the
	// the user has a role within the owner/repository.
	ListWebhooks(context.Context, *connect_go.Request[v1alpha1.ListWebhooksRequest]) (*connect_go.Response[v1alpha1.ListWebhooksResponse], error)
}

// NewWebhookServiceHandler builds an HTTP handler from the service implementation. It returns the
// path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewWebhookServiceHandler(svc WebhookServiceHandler, opts ...connect_go.HandlerOption) (string, http.Handler) {
	mux := http.NewServeMux()
	mux.Handle("/buf.alpha.registry.v1alpha1.WebhookService/CreateWebhook", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.WebhookService/CreateWebhook",
		svc.CreateWebhook,
		opts...,
	))
	mux.Handle("/buf.alpha.registry.v1alpha1.WebhookService/DeleteWebhook", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.WebhookService/DeleteWebhook",
		svc.DeleteWebhook,
		opts...,
	))
	mux.Handle("/buf.alpha.registry.v1alpha1.WebhookService/ListWebhooks", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.WebhookService/ListWebhooks",
		svc.ListWebhooks,
		opts...,
	))
	return "/buf.alpha.registry.v1alpha1.WebhookService/", mux
}

// UnimplementedWebhookServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedWebhookServiceHandler struct{}

func (UnimplementedWebhookServiceHandler) CreateWebhook(context.Context, *connect_go.Request[v1alpha1.CreateWebhookRequest]) (*connect_go.Response[v1alpha1.CreateWebhookResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.WebhookService.CreateWebhook is not implemented"))
}

func (UnimplementedWebhookServiceHandler) DeleteWebhook(context.Context, *connect_go.Request[v1alpha1.DeleteWebhookRequest]) (*connect_go.Response[v1alpha1.DeleteWebhookResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.WebhookService.DeleteWebhook is not implemented"))
}

func (UnimplementedWebhookServiceHandler) ListWebhooks(context.Context, *connect_go.Request[v1alpha1.ListWebhooksRequest]) (*connect_go.Response[v1alpha1.ListWebhooksResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.WebhookService.ListWebhooks is not implemented"))
}
