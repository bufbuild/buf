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
// Source: buf/alpha/webhook/v1alpha1/event.proto

package webhookv1alpha1connect

import (
	connect "connectrpc.com/connect"
	context "context"
	errors "errors"
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/webhook/v1alpha1"
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
	// EventServiceName is the fully-qualified name of the EventService service.
	EventServiceName = "buf.alpha.webhook.v1alpha1.EventService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// EventServiceEventProcedure is the fully-qualified name of the EventService's Event RPC.
	EventServiceEventProcedure = "/buf.alpha.webhook.v1alpha1.EventService/Event"
)

// These variables are the protoreflect.Descriptor objects for the RPCs defined in this package.
var (
	eventServiceServiceDescriptor     = v1alpha1.File_buf_alpha_webhook_v1alpha1_event_proto.Services().ByName("EventService")
	eventServiceEventMethodDescriptor = eventServiceServiceDescriptor.Methods().ByName("Event")
)

// EventServiceClient is a client for the buf.alpha.webhook.v1alpha1.EventService service.
type EventServiceClient interface {
	// Event is the rpc which receives webhook events.
	Event(context.Context, *connect.Request[v1alpha1.EventRequest]) (*connect.Response[v1alpha1.EventResponse], error)
}

// NewEventServiceClient constructs a client for the buf.alpha.webhook.v1alpha1.EventService
// service. By default, it uses the Connect protocol with the binary Protobuf Codec, asks for
// gzipped responses, and sends uncompressed requests. To use the gRPC or gRPC-Web protocols, supply
// the connect.WithGRPC() or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewEventServiceClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) EventServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &eventServiceClient{
		event: connect.NewClient[v1alpha1.EventRequest, v1alpha1.EventResponse](
			httpClient,
			baseURL+EventServiceEventProcedure,
			connect.WithSchema(eventServiceEventMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
	}
}

// eventServiceClient implements EventServiceClient.
type eventServiceClient struct {
	event *connect.Client[v1alpha1.EventRequest, v1alpha1.EventResponse]
}

// Event calls buf.alpha.webhook.v1alpha1.EventService.Event.
func (c *eventServiceClient) Event(ctx context.Context, req *connect.Request[v1alpha1.EventRequest]) (*connect.Response[v1alpha1.EventResponse], error) {
	return c.event.CallUnary(ctx, req)
}

// EventServiceHandler is an implementation of the buf.alpha.webhook.v1alpha1.EventService service.
type EventServiceHandler interface {
	// Event is the rpc which receives webhook events.
	Event(context.Context, *connect.Request[v1alpha1.EventRequest]) (*connect.Response[v1alpha1.EventResponse], error)
}

// NewEventServiceHandler builds an HTTP handler from the service implementation. It returns the
// path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewEventServiceHandler(svc EventServiceHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	eventServiceEventHandler := connect.NewUnaryHandler(
		EventServiceEventProcedure,
		svc.Event,
		connect.WithSchema(eventServiceEventMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	return "/buf.alpha.webhook.v1alpha1.EventService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case EventServiceEventProcedure:
			eventServiceEventHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedEventServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedEventServiceHandler struct{}

func (UnimplementedEventServiceHandler) Event(context.Context, *connect.Request[v1alpha1.EventRequest]) (*connect.Response[v1alpha1.EventResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.webhook.v1alpha1.EventService.Event is not implemented"))
}
