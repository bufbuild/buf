// Copyright 2020-2021 Buf Technologies, Inc.
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

// Code generated by protoc-gen-twirp v8.1.0, DO NOT EDIT.
// source: buf/alpha/registry/v1alpha1/push.proto

package registryv1alpha1

import context "context"
import fmt "fmt"
import http "net/http"
import ioutil "io/ioutil"
import json "encoding/json"
import strconv "strconv"
import strings "strings"

import protojson "google.golang.org/protobuf/encoding/protojson"
import proto "google.golang.org/protobuf/proto"
import twirp "github.com/twitchtv/twirp"
import ctxsetters "github.com/twitchtv/twirp/ctxsetters"

// Version compatibility assertion.
// If the constant is not defined in the package, that likely means
// the package needs to be updated to work with this generated code.
// See https://twitchtv.github.io/twirp/docs/version_matrix.html
const _ = twirp.TwirpPackageMinVersion_8_1_0

// =====================
// PushService Interface
// =====================

// PushService is the Push service.
type PushService interface {
	// Push pushes.
	Push(context.Context, *PushRequest) (*PushResponse, error)
}

// ===========================
// PushService Protobuf Client
// ===========================

type pushServiceProtobufClient struct {
	client      HTTPClient
	urls        [1]string
	interceptor twirp.Interceptor
	opts        twirp.ClientOptions
}

// NewPushServiceProtobufClient creates a Protobuf client that implements the PushService interface.
// It communicates using Protobuf and can be configured with a custom HTTPClient.
func NewPushServiceProtobufClient(baseURL string, client HTTPClient, opts ...twirp.ClientOption) PushService {
	if c, ok := client.(*http.Client); ok {
		client = withoutRedirects(c)
	}

	clientOpts := twirp.ClientOptions{}
	for _, o := range opts {
		o(&clientOpts)
	}

	// Using ReadOpt allows backwards and forwads compatibility with new options in the future
	literalURLs := false
	_ = clientOpts.ReadOpt("literalURLs", &literalURLs)
	var pathPrefix string
	if ok := clientOpts.ReadOpt("pathPrefix", &pathPrefix); !ok {
		pathPrefix = "/twirp" // default prefix
	}

	// Build method URLs: <baseURL>[<prefix>]/<package>.<Service>/<Method>
	serviceURL := sanitizeBaseURL(baseURL)
	serviceURL += baseServicePath(pathPrefix, "buf.alpha.registry.v1alpha1", "PushService")
	urls := [1]string{
		serviceURL + "Push",
	}

	return &pushServiceProtobufClient{
		client:      client,
		urls:        urls,
		interceptor: twirp.ChainInterceptors(clientOpts.Interceptors...),
		opts:        clientOpts,
	}
}

func (c *pushServiceProtobufClient) Push(ctx context.Context, in *PushRequest) (*PushResponse, error) {
	ctx = ctxsetters.WithPackageName(ctx, "buf.alpha.registry.v1alpha1")
	ctx = ctxsetters.WithServiceName(ctx, "PushService")
	ctx = ctxsetters.WithMethodName(ctx, "Push")
	caller := c.callPush
	if c.interceptor != nil {
		caller = func(ctx context.Context, req *PushRequest) (*PushResponse, error) {
			resp, err := c.interceptor(
				func(ctx context.Context, req interface{}) (interface{}, error) {
					typedReq, ok := req.(*PushRequest)
					if !ok {
						return nil, twirp.InternalError("failed type assertion req.(*PushRequest) when calling interceptor")
					}
					return c.callPush(ctx, typedReq)
				},
			)(ctx, req)
			if resp != nil {
				typedResp, ok := resp.(*PushResponse)
				if !ok {
					return nil, twirp.InternalError("failed type assertion resp.(*PushResponse) when calling interceptor")
				}
				return typedResp, err
			}
			return nil, err
		}
	}
	return caller(ctx, in)
}

func (c *pushServiceProtobufClient) callPush(ctx context.Context, in *PushRequest) (*PushResponse, error) {
	out := new(PushResponse)
	ctx, err := doProtobufRequest(ctx, c.client, c.opts.Hooks, c.urls[0], in, out)
	if err != nil {
		twerr, ok := err.(twirp.Error)
		if !ok {
			twerr = twirp.InternalErrorWith(err)
		}
		callClientError(ctx, c.opts.Hooks, twerr)
		return nil, err
	}

	callClientResponseReceived(ctx, c.opts.Hooks)

	return out, nil
}

// =======================
// PushService JSON Client
// =======================

type pushServiceJSONClient struct {
	client      HTTPClient
	urls        [1]string
	interceptor twirp.Interceptor
	opts        twirp.ClientOptions
}

// NewPushServiceJSONClient creates a JSON client that implements the PushService interface.
// It communicates using JSON and can be configured with a custom HTTPClient.
func NewPushServiceJSONClient(baseURL string, client HTTPClient, opts ...twirp.ClientOption) PushService {
	if c, ok := client.(*http.Client); ok {
		client = withoutRedirects(c)
	}

	clientOpts := twirp.ClientOptions{}
	for _, o := range opts {
		o(&clientOpts)
	}

	// Using ReadOpt allows backwards and forwads compatibility with new options in the future
	literalURLs := false
	_ = clientOpts.ReadOpt("literalURLs", &literalURLs)
	var pathPrefix string
	if ok := clientOpts.ReadOpt("pathPrefix", &pathPrefix); !ok {
		pathPrefix = "/twirp" // default prefix
	}

	// Build method URLs: <baseURL>[<prefix>]/<package>.<Service>/<Method>
	serviceURL := sanitizeBaseURL(baseURL)
	serviceURL += baseServicePath(pathPrefix, "buf.alpha.registry.v1alpha1", "PushService")
	urls := [1]string{
		serviceURL + "Push",
	}

	return &pushServiceJSONClient{
		client:      client,
		urls:        urls,
		interceptor: twirp.ChainInterceptors(clientOpts.Interceptors...),
		opts:        clientOpts,
	}
}

func (c *pushServiceJSONClient) Push(ctx context.Context, in *PushRequest) (*PushResponse, error) {
	ctx = ctxsetters.WithPackageName(ctx, "buf.alpha.registry.v1alpha1")
	ctx = ctxsetters.WithServiceName(ctx, "PushService")
	ctx = ctxsetters.WithMethodName(ctx, "Push")
	caller := c.callPush
	if c.interceptor != nil {
		caller = func(ctx context.Context, req *PushRequest) (*PushResponse, error) {
			resp, err := c.interceptor(
				func(ctx context.Context, req interface{}) (interface{}, error) {
					typedReq, ok := req.(*PushRequest)
					if !ok {
						return nil, twirp.InternalError("failed type assertion req.(*PushRequest) when calling interceptor")
					}
					return c.callPush(ctx, typedReq)
				},
			)(ctx, req)
			if resp != nil {
				typedResp, ok := resp.(*PushResponse)
				if !ok {
					return nil, twirp.InternalError("failed type assertion resp.(*PushResponse) when calling interceptor")
				}
				return typedResp, err
			}
			return nil, err
		}
	}
	return caller(ctx, in)
}

func (c *pushServiceJSONClient) callPush(ctx context.Context, in *PushRequest) (*PushResponse, error) {
	out := new(PushResponse)
	ctx, err := doJSONRequest(ctx, c.client, c.opts.Hooks, c.urls[0], in, out)
	if err != nil {
		twerr, ok := err.(twirp.Error)
		if !ok {
			twerr = twirp.InternalErrorWith(err)
		}
		callClientError(ctx, c.opts.Hooks, twerr)
		return nil, err
	}

	callClientResponseReceived(ctx, c.opts.Hooks)

	return out, nil
}

// ==========================
// PushService Server Handler
// ==========================

type pushServiceServer struct {
	PushService
	interceptor      twirp.Interceptor
	hooks            *twirp.ServerHooks
	pathPrefix       string // prefix for routing
	jsonSkipDefaults bool   // do not include unpopulated fields (default values) in the response
	jsonCamelCase    bool   // JSON fields are serialized as lowerCamelCase rather than keeping the original proto names
}

// NewPushServiceServer builds a TwirpServer that can be used as an http.Handler to handle
// HTTP requests that are routed to the right method in the provided svc implementation.
// The opts are twirp.ServerOption modifiers, for example twirp.WithServerHooks(hooks).
func NewPushServiceServer(svc PushService, opts ...interface{}) TwirpServer {
	serverOpts := newServerOpts(opts)

	// Using ReadOpt allows backwards and forwads compatibility with new options in the future
	jsonSkipDefaults := false
	_ = serverOpts.ReadOpt("jsonSkipDefaults", &jsonSkipDefaults)
	jsonCamelCase := false
	_ = serverOpts.ReadOpt("jsonCamelCase", &jsonCamelCase)
	var pathPrefix string
	if ok := serverOpts.ReadOpt("pathPrefix", &pathPrefix); !ok {
		pathPrefix = "/twirp" // default prefix
	}

	return &pushServiceServer{
		PushService:      svc,
		hooks:            serverOpts.Hooks,
		interceptor:      twirp.ChainInterceptors(serverOpts.Interceptors...),
		pathPrefix:       pathPrefix,
		jsonSkipDefaults: jsonSkipDefaults,
		jsonCamelCase:    jsonCamelCase,
	}
}

// writeError writes an HTTP response with a valid Twirp error format, and triggers hooks.
// If err is not a twirp.Error, it will get wrapped with twirp.InternalErrorWith(err)
func (s *pushServiceServer) writeError(ctx context.Context, resp http.ResponseWriter, err error) {
	writeError(ctx, resp, err, s.hooks)
}

// handleRequestBodyError is used to handle error when the twirp server cannot read request
func (s *pushServiceServer) handleRequestBodyError(ctx context.Context, resp http.ResponseWriter, msg string, err error) {
	if context.Canceled == ctx.Err() {
		s.writeError(ctx, resp, twirp.NewError(twirp.Canceled, "failed to read request: context canceled"))
		return
	}
	if context.DeadlineExceeded == ctx.Err() {
		s.writeError(ctx, resp, twirp.NewError(twirp.DeadlineExceeded, "failed to read request: deadline exceeded"))
		return
	}
	s.writeError(ctx, resp, twirp.WrapError(malformedRequestError(msg), err))
}

// PushServicePathPrefix is a convenience constant that may identify URL paths.
// Should be used with caution, it only matches routes generated by Twirp Go clients,
// with the default "/twirp" prefix and default CamelCase service and method names.
// More info: https://twitchtv.github.io/twirp/docs/routing.html
const PushServicePathPrefix = "/twirp/buf.alpha.registry.v1alpha1.PushService/"

func (s *pushServiceServer) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	ctx = ctxsetters.WithPackageName(ctx, "buf.alpha.registry.v1alpha1")
	ctx = ctxsetters.WithServiceName(ctx, "PushService")
	ctx = ctxsetters.WithResponseWriter(ctx, resp)

	var err error
	ctx, err = callRequestReceived(ctx, s.hooks)
	if err != nil {
		s.writeError(ctx, resp, err)
		return
	}

	if req.Method != "POST" {
		msg := fmt.Sprintf("unsupported method %q (only POST is allowed)", req.Method)
		s.writeError(ctx, resp, badRouteError(msg, req.Method, req.URL.Path))
		return
	}

	// Verify path format: [<prefix>]/<package>.<Service>/<Method>
	prefix, pkgService, method := parseTwirpPath(req.URL.Path)
	if pkgService != "buf.alpha.registry.v1alpha1.PushService" {
		msg := fmt.Sprintf("no handler for path %q", req.URL.Path)
		s.writeError(ctx, resp, badRouteError(msg, req.Method, req.URL.Path))
		return
	}
	if prefix != s.pathPrefix {
		msg := fmt.Sprintf("invalid path prefix %q, expected %q, on path %q", prefix, s.pathPrefix, req.URL.Path)
		s.writeError(ctx, resp, badRouteError(msg, req.Method, req.URL.Path))
		return
	}

	switch method {
	case "Push":
		s.servePush(ctx, resp, req)
		return
	default:
		msg := fmt.Sprintf("no handler for path %q", req.URL.Path)
		s.writeError(ctx, resp, badRouteError(msg, req.Method, req.URL.Path))
		return
	}
}

func (s *pushServiceServer) servePush(ctx context.Context, resp http.ResponseWriter, req *http.Request) {
	header := req.Header.Get("Content-Type")
	i := strings.Index(header, ";")
	if i == -1 {
		i = len(header)
	}
	switch strings.TrimSpace(strings.ToLower(header[:i])) {
	case "application/json":
		s.servePushJSON(ctx, resp, req)
	case "application/protobuf":
		s.servePushProtobuf(ctx, resp, req)
	default:
		msg := fmt.Sprintf("unexpected Content-Type: %q", req.Header.Get("Content-Type"))
		twerr := badRouteError(msg, req.Method, req.URL.Path)
		s.writeError(ctx, resp, twerr)
	}
}

func (s *pushServiceServer) servePushJSON(ctx context.Context, resp http.ResponseWriter, req *http.Request) {
	var err error
	ctx = ctxsetters.WithMethodName(ctx, "Push")
	ctx, err = callRequestRouted(ctx, s.hooks)
	if err != nil {
		s.writeError(ctx, resp, err)
		return
	}

	d := json.NewDecoder(req.Body)
	rawReqBody := json.RawMessage{}
	if err := d.Decode(&rawReqBody); err != nil {
		s.handleRequestBodyError(ctx, resp, "the json request could not be decoded", err)
		return
	}
	reqContent := new(PushRequest)
	unmarshaler := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err = unmarshaler.Unmarshal(rawReqBody, reqContent); err != nil {
		s.handleRequestBodyError(ctx, resp, "the json request could not be decoded", err)
		return
	}

	handler := s.PushService.Push
	if s.interceptor != nil {
		handler = func(ctx context.Context, req *PushRequest) (*PushResponse, error) {
			resp, err := s.interceptor(
				func(ctx context.Context, req interface{}) (interface{}, error) {
					typedReq, ok := req.(*PushRequest)
					if !ok {
						return nil, twirp.InternalError("failed type assertion req.(*PushRequest) when calling interceptor")
					}
					return s.PushService.Push(ctx, typedReq)
				},
			)(ctx, req)
			if resp != nil {
				typedResp, ok := resp.(*PushResponse)
				if !ok {
					return nil, twirp.InternalError("failed type assertion resp.(*PushResponse) when calling interceptor")
				}
				return typedResp, err
			}
			return nil, err
		}
	}

	// Call service method
	var respContent *PushResponse
	func() {
		defer ensurePanicResponses(ctx, resp, s.hooks)
		respContent, err = handler(ctx, reqContent)
	}()

	if err != nil {
		s.writeError(ctx, resp, err)
		return
	}
	if respContent == nil {
		s.writeError(ctx, resp, twirp.InternalError("received a nil *PushResponse and nil error while calling Push. nil responses are not supported"))
		return
	}

	ctx = callResponsePrepared(ctx, s.hooks)

	marshaler := &protojson.MarshalOptions{UseProtoNames: !s.jsonCamelCase, EmitUnpopulated: !s.jsonSkipDefaults}
	respBytes, err := marshaler.Marshal(respContent)
	if err != nil {
		s.writeError(ctx, resp, wrapInternal(err, "failed to marshal json response"))
		return
	}

	ctx = ctxsetters.WithStatusCode(ctx, http.StatusOK)
	resp.Header().Set("Content-Type", "application/json")
	resp.Header().Set("Content-Length", strconv.Itoa(len(respBytes)))
	resp.WriteHeader(http.StatusOK)

	if n, err := resp.Write(respBytes); err != nil {
		msg := fmt.Sprintf("failed to write response, %d of %d bytes written: %s", n, len(respBytes), err.Error())
		twerr := twirp.NewError(twirp.Unknown, msg)
		ctx = callError(ctx, s.hooks, twerr)
	}
	callResponseSent(ctx, s.hooks)
}

func (s *pushServiceServer) servePushProtobuf(ctx context.Context, resp http.ResponseWriter, req *http.Request) {
	var err error
	ctx = ctxsetters.WithMethodName(ctx, "Push")
	ctx, err = callRequestRouted(ctx, s.hooks)
	if err != nil {
		s.writeError(ctx, resp, err)
		return
	}

	buf, err := ioutil.ReadAll(req.Body)
	if err != nil {
		s.handleRequestBodyError(ctx, resp, "failed to read request body", err)
		return
	}
	reqContent := new(PushRequest)
	if err = proto.Unmarshal(buf, reqContent); err != nil {
		s.writeError(ctx, resp, malformedRequestError("the protobuf request could not be decoded"))
		return
	}

	handler := s.PushService.Push
	if s.interceptor != nil {
		handler = func(ctx context.Context, req *PushRequest) (*PushResponse, error) {
			resp, err := s.interceptor(
				func(ctx context.Context, req interface{}) (interface{}, error) {
					typedReq, ok := req.(*PushRequest)
					if !ok {
						return nil, twirp.InternalError("failed type assertion req.(*PushRequest) when calling interceptor")
					}
					return s.PushService.Push(ctx, typedReq)
				},
			)(ctx, req)
			if resp != nil {
				typedResp, ok := resp.(*PushResponse)
				if !ok {
					return nil, twirp.InternalError("failed type assertion resp.(*PushResponse) when calling interceptor")
				}
				return typedResp, err
			}
			return nil, err
		}
	}

	// Call service method
	var respContent *PushResponse
	func() {
		defer ensurePanicResponses(ctx, resp, s.hooks)
		respContent, err = handler(ctx, reqContent)
	}()

	if err != nil {
		s.writeError(ctx, resp, err)
		return
	}
	if respContent == nil {
		s.writeError(ctx, resp, twirp.InternalError("received a nil *PushResponse and nil error while calling Push. nil responses are not supported"))
		return
	}

	ctx = callResponsePrepared(ctx, s.hooks)

	respBytes, err := proto.Marshal(respContent)
	if err != nil {
		s.writeError(ctx, resp, wrapInternal(err, "failed to marshal proto response"))
		return
	}

	ctx = ctxsetters.WithStatusCode(ctx, http.StatusOK)
	resp.Header().Set("Content-Type", "application/protobuf")
	resp.Header().Set("Content-Length", strconv.Itoa(len(respBytes)))
	resp.WriteHeader(http.StatusOK)
	if n, err := resp.Write(respBytes); err != nil {
		msg := fmt.Sprintf("failed to write response, %d of %d bytes written: %s", n, len(respBytes), err.Error())
		twerr := twirp.NewError(twirp.Unknown, msg)
		ctx = callError(ctx, s.hooks, twerr)
	}
	callResponseSent(ctx, s.hooks)
}

func (s *pushServiceServer) ServiceDescriptor() ([]byte, int) {
	return twirpFileDescriptor4, 0
}

func (s *pushServiceServer) ProtocGenTwirpVersion() string {
	return "v8.1.0"
}

// PathPrefix returns the base service path, in the form: "/<prefix>/<package>.<Service>/"
// that is everything in a Twirp route except for the <Method>. This can be used for routing,
// for example to identify the requests that are targeted to this service in a mux.
func (s *pushServiceServer) PathPrefix() string {
	return baseServicePath(s.pathPrefix, "buf.alpha.registry.v1alpha1", "PushService")
}

var twirpFileDescriptor4 = []byte{
	// 365 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x92, 0x41, 0x6a, 0xe3, 0x30,
	0x14, 0x86, 0x71, 0xe2, 0x04, 0xa2, 0x0c, 0xc3, 0x20, 0x86, 0xc1, 0x64, 0x60, 0xc6, 0xf5, 0xa2,
	0xb8, 0x14, 0x24, 0x92, 0xae, 0x4a, 0x77, 0x59, 0xb7, 0x10, 0x5c, 0xba, 0xc9, 0x26, 0xc8, 0x8e,
	0x62, 0x0b, 0x1c, 0x49, 0x95, 0xac, 0x94, 0xdc, 0x20, 0x37, 0xe8, 0x1d, 0x7a, 0xca, 0x62, 0xc9,
	0xc6, 0xee, 0xc6, 0x74, 0xa7, 0xff, 0xd7, 0xff, 0xe9, 0x3d, 0x3d, 0x09, 0x5c, 0xa7, 0xe6, 0x80,
	0x49, 0x29, 0x0b, 0x82, 0x15, 0xcd, 0x99, 0xae, 0xd4, 0x19, 0x9f, 0x96, 0xd6, 0x58, 0x62, 0x69,
	0x74, 0x81, 0xa4, 0x12, 0x95, 0x80, 0x7f, 0x53, 0x73, 0x40, 0xd6, 0x46, 0x6d, 0x0e, 0xb5, 0xb9,
	0x45, 0xd8, 0x1d, 0x42, 0x24, 0xeb, 0x78, 0x22, 0x99, 0xc3, 0x17, 0xbd, 0x32, 0x47, 0xb1, 0x37,
	0x25, 0xed, 0x42, 0x4e, 0x37, 0xb9, 0x78, 0xa8, 0x9d, 0x7e, 0x32, 0xfa, 0xf0, 0xc0, 0x7c, 0x63,
	0x74, 0x91, 0xd0, 0x57, 0x43, 0x75, 0x05, 0x7f, 0x83, 0x89, 0x78, 0xe3, 0x54, 0x05, 0x5e, 0xe8,
	0xc5, 0xb3, 0xc4, 0x09, 0xf8, 0x0f, 0x00, 0x45, 0xa5, 0xd0, 0xac, 0x12, 0xea, 0x1c, 0x8c, 0xec,
	0x56, 0xcf, 0x81, 0x7f, 0xc0, 0x34, 0x55, 0x84, 0x67, 0x45, 0x30, 0xb6, 0x7b, 0x8d, 0x82, 0xf7,
	0x60, 0xea, 0xaa, 0x05, 0x7e, 0xe8, 0xc5, 0xf3, 0xd5, 0x15, 0xea, 0xee, 0xdf, 0xb4, 0xd1, 0xb6,
	0x85, 0x9e, 0xac, 0x4e, 0x1a, 0x00, 0x42, 0xe0, 0x57, 0x24, 0xd7, 0xc1, 0x24, 0x1c, 0xc7, 0xb3,
	0xc4, 0xae, 0x23, 0x0a, 0x7e, 0xb8, 0x5e, 0xb5, 0x14, 0x5c, 0x53, 0xf8, 0x02, 0x7e, 0x95, 0x22,
	0x23, 0xe5, 0xce, 0x31, 0x3b, 0xc9, 0x78, 0x30, 0xb1, 0x85, 0x6e, 0xd1, 0xc0, 0xa0, 0xd1, 0x63,
	0x0d, 0xb9, 0x7a, 0x1b, 0xc6, 0x93, 0x9f, 0xe5, 0x17, 0xbd, 0x92, 0x6e, 0x24, 0xcf, 0x54, 0x9d,
	0x58, 0x46, 0x21, 0x01, 0x7e, 0x2d, 0x61, 0x3c, 0x78, 0x66, 0x6f, 0x88, 0x8b, 0x9b, 0x6f, 0x24,
	0xdd, 0x15, 0x22, 0xff, 0xf2, 0x1e, 0x8d, 0xd6, 0x17, 0x0f, 0xfc, 0xcf, 0xc4, 0x71, 0x08, 0x5b,
	0xcf, 0x6a, 0x6e, 0x53, 0x3f, 0xda, 0x76, 0x9b, 0xb3, 0xaa, 0x30, 0x29, 0xca, 0xc4, 0x11, 0xa7,
	0xe6, 0x90, 0x1a, 0x56, 0xee, 0xeb, 0x05, 0x66, 0xbc, 0xa2, 0x8a, 0x93, 0x12, 0xe7, 0x94, 0x63,
	0xfb, 0xc0, 0x38, 0x17, 0x78, 0xe0, 0x33, 0x3c, 0xb4, 0x4e, 0x6b, 0xa4, 0x53, 0x8b, 0xdd, 0x7d,
	0x06, 0x00, 0x00, 0xff, 0xff, 0xa2, 0xe3, 0x64, 0x27, 0xd2, 0x02, 0x00, 0x00,
}
