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
// Source: buf/alpha/audit/v1alpha1/service.proto

package auditv1alpha1connect

import (
	context "context"
	errors "errors"
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/audit/v1alpha1"
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
	// AuditServiceName is the fully-qualified name of the AuditService service.
	AuditServiceName = "buf.alpha.audit.v1alpha1.AuditService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// AuditServiceListAuditedEventsProcedure is the fully-qualified name of the AuditService's
	// ListAuditedEvents RPC.
	AuditServiceListAuditedEventsProcedure = "/buf.alpha.audit.v1alpha1.AuditService/ListAuditedEvents"
)

// AuditServiceClient is a client for the buf.alpha.audit.v1alpha1.AuditService service.
type AuditServiceClient interface {
	// ListAuditedEvents lists audited events recorded in the BSR instance.
	ListAuditedEvents(context.Context, *connect_go.Request[v1alpha1.ListAuditedEventsRequest]) (*connect_go.Response[v1alpha1.ListAuditedEventsResponse], error)
}

// NewAuditServiceClient constructs a client for the buf.alpha.audit.v1alpha1.AuditService service.
// By default, it uses the Connect protocol with the binary Protobuf Codec, asks for gzipped
// responses, and sends uncompressed requests. To use the gRPC or gRPC-Web protocols, supply the
// connect.WithGRPC() or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewAuditServiceClient(httpClient connect_go.HTTPClient, baseURL string, opts ...connect_go.ClientOption) AuditServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &auditServiceClient{
		listAuditedEvents: connect_go.NewClient[v1alpha1.ListAuditedEventsRequest, v1alpha1.ListAuditedEventsResponse](
			httpClient,
			baseURL+AuditServiceListAuditedEventsProcedure,
			connect_go.WithIdempotency(connect_go.IdempotencyNoSideEffects),
			connect_go.WithClientOptions(opts...),
		),
	}
}

// auditServiceClient implements AuditServiceClient.
type auditServiceClient struct {
	listAuditedEvents *connect_go.Client[v1alpha1.ListAuditedEventsRequest, v1alpha1.ListAuditedEventsResponse]
}

// ListAuditedEvents calls buf.alpha.audit.v1alpha1.AuditService.ListAuditedEvents.
func (c *auditServiceClient) ListAuditedEvents(ctx context.Context, req *connect_go.Request[v1alpha1.ListAuditedEventsRequest]) (*connect_go.Response[v1alpha1.ListAuditedEventsResponse], error) {
	return c.listAuditedEvents.CallUnary(ctx, req)
}

// AuditServiceHandler is an implementation of the buf.alpha.audit.v1alpha1.AuditService service.
type AuditServiceHandler interface {
	// ListAuditedEvents lists audited events recorded in the BSR instance.
	ListAuditedEvents(context.Context, *connect_go.Request[v1alpha1.ListAuditedEventsRequest]) (*connect_go.Response[v1alpha1.ListAuditedEventsResponse], error)
}

// NewAuditServiceHandler builds an HTTP handler from the service implementation. It returns the
// path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewAuditServiceHandler(svc AuditServiceHandler, opts ...connect_go.HandlerOption) (string, http.Handler) {
	auditServiceListAuditedEventsHandler := connect_go.NewUnaryHandler(
		AuditServiceListAuditedEventsProcedure,
		svc.ListAuditedEvents,
		connect_go.WithIdempotency(connect_go.IdempotencyNoSideEffects),
		connect_go.WithHandlerOptions(opts...),
	)
	return "/buf.alpha.audit.v1alpha1.AuditService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case AuditServiceListAuditedEventsProcedure:
			auditServiceListAuditedEventsHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedAuditServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedAuditServiceHandler struct{}

func (UnimplementedAuditServiceHandler) ListAuditedEvents(context.Context, *connect_go.Request[v1alpha1.ListAuditedEventsRequest]) (*connect_go.Response[v1alpha1.ListAuditedEventsResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.audit.v1alpha1.AuditService.ListAuditedEvents is not implemented"))
}
