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
// Source: buf/alpha/registry/v1alpha1/download.proto

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
const _ = connect_go.IsAtLeastVersion0_0_1

const (
	// DownloadServiceName is the fully-qualified name of the DownloadService service.
	DownloadServiceName = "buf.alpha.registry.v1alpha1.DownloadService"
)

// DownloadServiceClient is a client for the buf.alpha.registry.v1alpha1.DownloadService service.
type DownloadServiceClient interface {
	// Download downloads.
	Download(context.Context, *connect_go.Request[v1alpha1.DownloadRequest]) (*connect_go.Response[v1alpha1.DownloadResponse], error)
}

// NewDownloadServiceClient constructs a client for the buf.alpha.registry.v1alpha1.DownloadService
// service. By default, it uses the binary Protobuf Codec, asks for gzipped responses, and sends
// uncompressed requests. It doesn't have a default protocol; you must supply either the
// connect.WithGRPC() or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewDownloadServiceClient(httpClient connect_go.HTTPClient, baseURL string, opts ...connect_go.ClientOption) DownloadServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &downloadServiceClient{
		download: connect_go.NewClient[v1alpha1.DownloadRequest, v1alpha1.DownloadResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.DownloadService/Download",
			opts...,
		),
	}
}

// downloadServiceClient implements DownloadServiceClient.
type downloadServiceClient struct {
	download *connect_go.Client[v1alpha1.DownloadRequest, v1alpha1.DownloadResponse]
}

// Download calls buf.alpha.registry.v1alpha1.DownloadService.Download.
func (c *downloadServiceClient) Download(ctx context.Context, req *connect_go.Request[v1alpha1.DownloadRequest]) (*connect_go.Response[v1alpha1.DownloadResponse], error) {
	return c.download.CallUnary(ctx, req)
}

// DownloadServiceHandler is an implementation of the buf.alpha.registry.v1alpha1.DownloadService
// service.
type DownloadServiceHandler interface {
	// Download downloads.
	Download(context.Context, *connect_go.Request[v1alpha1.DownloadRequest]) (*connect_go.Response[v1alpha1.DownloadResponse], error)
}

// NewDownloadServiceHandler builds an HTTP handler from the service implementation. It returns the
// path on which to mount the handler and the handler itself.
//
// By default, handlers support the gRPC and gRPC-Web protocols with the binary Protobuf and JSON
// codecs.
func NewDownloadServiceHandler(svc DownloadServiceHandler, opts ...connect_go.HandlerOption) (string, http.Handler) {
	mux := http.NewServeMux()
	mux.Handle("/buf.alpha.registry.v1alpha1.DownloadService/Download", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.DownloadService/Download",
		svc.Download,
		opts...,
	))
	return "/buf.alpha.registry.v1alpha1.DownloadService/", mux
}

// UnimplementedDownloadServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedDownloadServiceHandler struct{}

func (UnimplementedDownloadServiceHandler) Download(context.Context, *connect_go.Request[v1alpha1.DownloadRequest]) (*connect_go.Response[v1alpha1.DownloadResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.DownloadService.Download is not implemented"))
}
