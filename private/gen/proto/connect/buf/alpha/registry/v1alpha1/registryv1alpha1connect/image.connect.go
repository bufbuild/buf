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
// Source: buf/alpha/registry/v1alpha1/image.proto

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
	// ImageServiceName is the fully-qualified name of the ImageService service.
	ImageServiceName = "buf.alpha.registry.v1alpha1.ImageService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// ImageServiceGetImageProcedure is the fully-qualified name of the ImageService's GetImage RPC.
	ImageServiceGetImageProcedure = "/buf.alpha.registry.v1alpha1.ImageService/GetImage"
)

// ImageServiceClient is a client for the buf.alpha.registry.v1alpha1.ImageService service.
type ImageServiceClient interface {
	// GetImage serves a compiled image for the local module. It automatically
	// downloads dependencies if necessary.
	GetImage(context.Context, *connect.Request[v1alpha1.GetImageRequest]) (*connect.Response[v1alpha1.GetImageResponse], error)
}

// NewImageServiceClient constructs a client for the buf.alpha.registry.v1alpha1.ImageService
// service. By default, it uses the Connect protocol with the binary Protobuf Codec, asks for
// gzipped responses, and sends uncompressed requests. To use the gRPC or gRPC-Web protocols, supply
// the connect.WithGRPC() or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewImageServiceClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) ImageServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	imageServiceMethods := v1alpha1.File_buf_alpha_registry_v1alpha1_image_proto.Services().ByName("ImageService").Methods()
	return &imageServiceClient{
		getImage: connect.NewClient[v1alpha1.GetImageRequest, v1alpha1.GetImageResponse](
			httpClient,
			baseURL+ImageServiceGetImageProcedure,
			connect.WithSchema(imageServiceMethods.ByName("GetImage")),
			connect.WithIdempotency(connect.IdempotencyNoSideEffects),
			connect.WithClientOptions(opts...),
		),
	}
}

// imageServiceClient implements ImageServiceClient.
type imageServiceClient struct {
	getImage *connect.Client[v1alpha1.GetImageRequest, v1alpha1.GetImageResponse]
}

// GetImage calls buf.alpha.registry.v1alpha1.ImageService.GetImage.
func (c *imageServiceClient) GetImage(ctx context.Context, req *connect.Request[v1alpha1.GetImageRequest]) (*connect.Response[v1alpha1.GetImageResponse], error) {
	return c.getImage.CallUnary(ctx, req)
}

// ImageServiceHandler is an implementation of the buf.alpha.registry.v1alpha1.ImageService service.
type ImageServiceHandler interface {
	// GetImage serves a compiled image for the local module. It automatically
	// downloads dependencies if necessary.
	GetImage(context.Context, *connect.Request[v1alpha1.GetImageRequest]) (*connect.Response[v1alpha1.GetImageResponse], error)
}

// NewImageServiceHandler builds an HTTP handler from the service implementation. It returns the
// path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewImageServiceHandler(svc ImageServiceHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	imageServiceMethods := v1alpha1.File_buf_alpha_registry_v1alpha1_image_proto.Services().ByName("ImageService").Methods()
	imageServiceGetImageHandler := connect.NewUnaryHandler(
		ImageServiceGetImageProcedure,
		svc.GetImage,
		connect.WithSchema(imageServiceMethods.ByName("GetImage")),
		connect.WithIdempotency(connect.IdempotencyNoSideEffects),
		connect.WithHandlerOptions(opts...),
	)
	return "/buf.alpha.registry.v1alpha1.ImageService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case ImageServiceGetImageProcedure:
			imageServiceGetImageHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedImageServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedImageServiceHandler struct{}

func (UnimplementedImageServiceHandler) GetImage(context.Context, *connect.Request[v1alpha1.GetImageRequest]) (*connect.Response[v1alpha1.GetImageResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.ImageService.GetImage is not implemented"))
}
