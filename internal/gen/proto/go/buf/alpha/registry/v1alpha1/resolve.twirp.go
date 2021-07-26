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
// source: buf/alpha/registry/v1alpha1/resolve.proto

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

// ========================
// ResolveService Interface
// ========================

// ResolveService is the resolve service.
//
// This is the public service.
type ResolveService interface {
	// GetModulePins finds all the latest digests and respective dependencies of
	// the provided module references and picks a set of distinct modules pins.
	//
	// Note that module references with commits should still be passed to this function
	// to make sure this function can do dependency resolution.
	//
	// This function also deals with tiebreaking what ModulePin wins for the same repository.
	GetModulePins(context.Context, *GetModulePinsRequest) (*GetModulePinsResponse, error)
}

// ==============================
// ResolveService Protobuf Client
// ==============================

type resolveServiceProtobufClient struct {
	client      HTTPClient
	urls        [1]string
	interceptor twirp.Interceptor
	opts        twirp.ClientOptions
}

// NewResolveServiceProtobufClient creates a Protobuf client that implements the ResolveService interface.
// It communicates using Protobuf and can be configured with a custom HTTPClient.
func NewResolveServiceProtobufClient(baseURL string, client HTTPClient, opts ...twirp.ClientOption) ResolveService {
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
	serviceURL += baseServicePath(pathPrefix, "buf.alpha.registry.v1alpha1", "ResolveService")
	urls := [1]string{
		serviceURL + "GetModulePins",
	}

	return &resolveServiceProtobufClient{
		client:      client,
		urls:        urls,
		interceptor: twirp.ChainInterceptors(clientOpts.Interceptors...),
		opts:        clientOpts,
	}
}

func (c *resolveServiceProtobufClient) GetModulePins(ctx context.Context, in *GetModulePinsRequest) (*GetModulePinsResponse, error) {
	ctx = ctxsetters.WithPackageName(ctx, "buf.alpha.registry.v1alpha1")
	ctx = ctxsetters.WithServiceName(ctx, "ResolveService")
	ctx = ctxsetters.WithMethodName(ctx, "GetModulePins")
	caller := c.callGetModulePins
	if c.interceptor != nil {
		caller = func(ctx context.Context, req *GetModulePinsRequest) (*GetModulePinsResponse, error) {
			resp, err := c.interceptor(
				func(ctx context.Context, req interface{}) (interface{}, error) {
					typedReq, ok := req.(*GetModulePinsRequest)
					if !ok {
						return nil, twirp.InternalError("failed type assertion req.(*GetModulePinsRequest) when calling interceptor")
					}
					return c.callGetModulePins(ctx, typedReq)
				},
			)(ctx, req)
			if resp != nil {
				typedResp, ok := resp.(*GetModulePinsResponse)
				if !ok {
					return nil, twirp.InternalError("failed type assertion resp.(*GetModulePinsResponse) when calling interceptor")
				}
				return typedResp, err
			}
			return nil, err
		}
	}
	return caller(ctx, in)
}

func (c *resolveServiceProtobufClient) callGetModulePins(ctx context.Context, in *GetModulePinsRequest) (*GetModulePinsResponse, error) {
	out := new(GetModulePinsResponse)
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

// ==========================
// ResolveService JSON Client
// ==========================

type resolveServiceJSONClient struct {
	client      HTTPClient
	urls        [1]string
	interceptor twirp.Interceptor
	opts        twirp.ClientOptions
}

// NewResolveServiceJSONClient creates a JSON client that implements the ResolveService interface.
// It communicates using JSON and can be configured with a custom HTTPClient.
func NewResolveServiceJSONClient(baseURL string, client HTTPClient, opts ...twirp.ClientOption) ResolveService {
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
	serviceURL += baseServicePath(pathPrefix, "buf.alpha.registry.v1alpha1", "ResolveService")
	urls := [1]string{
		serviceURL + "GetModulePins",
	}

	return &resolveServiceJSONClient{
		client:      client,
		urls:        urls,
		interceptor: twirp.ChainInterceptors(clientOpts.Interceptors...),
		opts:        clientOpts,
	}
}

func (c *resolveServiceJSONClient) GetModulePins(ctx context.Context, in *GetModulePinsRequest) (*GetModulePinsResponse, error) {
	ctx = ctxsetters.WithPackageName(ctx, "buf.alpha.registry.v1alpha1")
	ctx = ctxsetters.WithServiceName(ctx, "ResolveService")
	ctx = ctxsetters.WithMethodName(ctx, "GetModulePins")
	caller := c.callGetModulePins
	if c.interceptor != nil {
		caller = func(ctx context.Context, req *GetModulePinsRequest) (*GetModulePinsResponse, error) {
			resp, err := c.interceptor(
				func(ctx context.Context, req interface{}) (interface{}, error) {
					typedReq, ok := req.(*GetModulePinsRequest)
					if !ok {
						return nil, twirp.InternalError("failed type assertion req.(*GetModulePinsRequest) when calling interceptor")
					}
					return c.callGetModulePins(ctx, typedReq)
				},
			)(ctx, req)
			if resp != nil {
				typedResp, ok := resp.(*GetModulePinsResponse)
				if !ok {
					return nil, twirp.InternalError("failed type assertion resp.(*GetModulePinsResponse) when calling interceptor")
				}
				return typedResp, err
			}
			return nil, err
		}
	}
	return caller(ctx, in)
}

func (c *resolveServiceJSONClient) callGetModulePins(ctx context.Context, in *GetModulePinsRequest) (*GetModulePinsResponse, error) {
	out := new(GetModulePinsResponse)
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

// =============================
// ResolveService Server Handler
// =============================

type resolveServiceServer struct {
	ResolveService
	interceptor      twirp.Interceptor
	hooks            *twirp.ServerHooks
	pathPrefix       string // prefix for routing
	jsonSkipDefaults bool   // do not include unpopulated fields (default values) in the response
	jsonCamelCase    bool   // JSON fields are serialized as lowerCamelCase rather than keeping the original proto names
}

// NewResolveServiceServer builds a TwirpServer that can be used as an http.Handler to handle
// HTTP requests that are routed to the right method in the provided svc implementation.
// The opts are twirp.ServerOption modifiers, for example twirp.WithServerHooks(hooks).
func NewResolveServiceServer(svc ResolveService, opts ...interface{}) TwirpServer {
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

	return &resolveServiceServer{
		ResolveService:   svc,
		hooks:            serverOpts.Hooks,
		interceptor:      twirp.ChainInterceptors(serverOpts.Interceptors...),
		pathPrefix:       pathPrefix,
		jsonSkipDefaults: jsonSkipDefaults,
		jsonCamelCase:    jsonCamelCase,
	}
}

// writeError writes an HTTP response with a valid Twirp error format, and triggers hooks.
// If err is not a twirp.Error, it will get wrapped with twirp.InternalErrorWith(err)
func (s *resolveServiceServer) writeError(ctx context.Context, resp http.ResponseWriter, err error) {
	writeError(ctx, resp, err, s.hooks)
}

// handleRequestBodyError is used to handle error when the twirp server cannot read request
func (s *resolveServiceServer) handleRequestBodyError(ctx context.Context, resp http.ResponseWriter, msg string, err error) {
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

// ResolveServicePathPrefix is a convenience constant that may identify URL paths.
// Should be used with caution, it only matches routes generated by Twirp Go clients,
// with the default "/twirp" prefix and default CamelCase service and method names.
// More info: https://twitchtv.github.io/twirp/docs/routing.html
const ResolveServicePathPrefix = "/twirp/buf.alpha.registry.v1alpha1.ResolveService/"

func (s *resolveServiceServer) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	ctx = ctxsetters.WithPackageName(ctx, "buf.alpha.registry.v1alpha1")
	ctx = ctxsetters.WithServiceName(ctx, "ResolveService")
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
	if pkgService != "buf.alpha.registry.v1alpha1.ResolveService" {
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
	case "GetModulePins":
		s.serveGetModulePins(ctx, resp, req)
		return
	default:
		msg := fmt.Sprintf("no handler for path %q", req.URL.Path)
		s.writeError(ctx, resp, badRouteError(msg, req.Method, req.URL.Path))
		return
	}
}

func (s *resolveServiceServer) serveGetModulePins(ctx context.Context, resp http.ResponseWriter, req *http.Request) {
	header := req.Header.Get("Content-Type")
	i := strings.Index(header, ";")
	if i == -1 {
		i = len(header)
	}
	switch strings.TrimSpace(strings.ToLower(header[:i])) {
	case "application/json":
		s.serveGetModulePinsJSON(ctx, resp, req)
	case "application/protobuf":
		s.serveGetModulePinsProtobuf(ctx, resp, req)
	default:
		msg := fmt.Sprintf("unexpected Content-Type: %q", req.Header.Get("Content-Type"))
		twerr := badRouteError(msg, req.Method, req.URL.Path)
		s.writeError(ctx, resp, twerr)
	}
}

func (s *resolveServiceServer) serveGetModulePinsJSON(ctx context.Context, resp http.ResponseWriter, req *http.Request) {
	var err error
	ctx = ctxsetters.WithMethodName(ctx, "GetModulePins")
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
	reqContent := new(GetModulePinsRequest)
	unmarshaler := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err = unmarshaler.Unmarshal(rawReqBody, reqContent); err != nil {
		s.handleRequestBodyError(ctx, resp, "the json request could not be decoded", err)
		return
	}

	handler := s.ResolveService.GetModulePins
	if s.interceptor != nil {
		handler = func(ctx context.Context, req *GetModulePinsRequest) (*GetModulePinsResponse, error) {
			resp, err := s.interceptor(
				func(ctx context.Context, req interface{}) (interface{}, error) {
					typedReq, ok := req.(*GetModulePinsRequest)
					if !ok {
						return nil, twirp.InternalError("failed type assertion req.(*GetModulePinsRequest) when calling interceptor")
					}
					return s.ResolveService.GetModulePins(ctx, typedReq)
				},
			)(ctx, req)
			if resp != nil {
				typedResp, ok := resp.(*GetModulePinsResponse)
				if !ok {
					return nil, twirp.InternalError("failed type assertion resp.(*GetModulePinsResponse) when calling interceptor")
				}
				return typedResp, err
			}
			return nil, err
		}
	}

	// Call service method
	var respContent *GetModulePinsResponse
	func() {
		defer ensurePanicResponses(ctx, resp, s.hooks)
		respContent, err = handler(ctx, reqContent)
	}()

	if err != nil {
		s.writeError(ctx, resp, err)
		return
	}
	if respContent == nil {
		s.writeError(ctx, resp, twirp.InternalError("received a nil *GetModulePinsResponse and nil error while calling GetModulePins. nil responses are not supported"))
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

func (s *resolveServiceServer) serveGetModulePinsProtobuf(ctx context.Context, resp http.ResponseWriter, req *http.Request) {
	var err error
	ctx = ctxsetters.WithMethodName(ctx, "GetModulePins")
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
	reqContent := new(GetModulePinsRequest)
	if err = proto.Unmarshal(buf, reqContent); err != nil {
		s.writeError(ctx, resp, malformedRequestError("the protobuf request could not be decoded"))
		return
	}

	handler := s.ResolveService.GetModulePins
	if s.interceptor != nil {
		handler = func(ctx context.Context, req *GetModulePinsRequest) (*GetModulePinsResponse, error) {
			resp, err := s.interceptor(
				func(ctx context.Context, req interface{}) (interface{}, error) {
					typedReq, ok := req.(*GetModulePinsRequest)
					if !ok {
						return nil, twirp.InternalError("failed type assertion req.(*GetModulePinsRequest) when calling interceptor")
					}
					return s.ResolveService.GetModulePins(ctx, typedReq)
				},
			)(ctx, req)
			if resp != nil {
				typedResp, ok := resp.(*GetModulePinsResponse)
				if !ok {
					return nil, twirp.InternalError("failed type assertion resp.(*GetModulePinsResponse) when calling interceptor")
				}
				return typedResp, err
			}
			return nil, err
		}
	}

	// Call service method
	var respContent *GetModulePinsResponse
	func() {
		defer ensurePanicResponses(ctx, resp, s.hooks)
		respContent, err = handler(ctx, reqContent)
	}()

	if err != nil {
		s.writeError(ctx, resp, err)
		return
	}
	if respContent == nil {
		s.writeError(ctx, resp, twirp.InternalError("received a nil *GetModulePinsResponse and nil error while calling GetModulePins. nil responses are not supported"))
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

func (s *resolveServiceServer) ServiceDescriptor() ([]byte, int) {
	return twirpFileDescriptor11, 0
}

func (s *resolveServiceServer) ProtocGenTwirpVersion() string {
	return "v8.1.0"
}

// PathPrefix returns the base service path, in the form: "/<prefix>/<package>.<Service>/"
// that is everything in a Twirp route except for the <Method>. This can be used for routing,
// for example to identify the requests that are targeted to this service in a mux.
func (s *resolveServiceServer) PathPrefix() string {
	return baseServicePath(s.pathPrefix, "buf.alpha.registry.v1alpha1", "ResolveService")
}

var twirpFileDescriptor11 = []byte{
	// 345 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x92, 0xcb, 0x4a, 0xc3, 0x40,
	0x18, 0x85, 0x49, 0x0b, 0x2e, 0xa6, 0x2a, 0x36, 0x2a, 0x94, 0x76, 0xa1, 0x14, 0x11, 0x75, 0x31,
	0x43, 0xeb, 0xd2, 0x55, 0x03, 0xe2, 0x4a, 0x28, 0xa3, 0x28, 0x74, 0x61, 0x69, 0xd2, 0x3f, 0xe9,
	0x40, 0x32, 0x13, 0xe7, 0x12, 0xf0, 0x0d, 0x7c, 0x0e, 0x97, 0x6e, 0x7c, 0x0a, 0xdf, 0x4b, 0x3a,
	0xc9, 0x18, 0x2b, 0x52, 0xe9, 0x2e, 0x39, 0x73, 0xfe, 0xef, 0x9c, 0xb9, 0xa0, 0xf3, 0xd0, 0xc4,
	0x64, 0x96, 0xe6, 0x8b, 0x19, 0x91, 0x90, 0x30, 0xa5, 0xe5, 0x0b, 0x29, 0x06, 0x56, 0x18, 0x10,
	0x09, 0x4a, 0xa4, 0x05, 0xe0, 0x5c, 0x0a, 0x2d, 0xfc, 0x5e, 0x68, 0x62, 0x6c, 0x57, 0xb0, 0xb3,
	0x62, 0x67, 0xed, 0x9e, 0xd6, 0x9c, 0x4c, 0xcc, 0x4d, 0x0a, 0x35, 0xa5, 0xfc, 0x2f, 0x21, 0xfd,
	0x4f, 0x0f, 0x1d, 0xdc, 0x80, 0xbe, 0xb5, 0xda, 0x98, 0x71, 0x45, 0xe1, 0xd9, 0x80, 0xd2, 0xfe,
	0x23, 0x6a, 0x97, 0xc6, 0xa9, 0x84, 0x18, 0x24, 0xf0, 0x08, 0x54, 0xc7, 0x3b, 0x6e, 0x9e, 0xb5,
	0x86, 0x17, 0xb8, 0x4e, 0xae, 0x60, 0x0e, 0x8e, 0x4b, 0x10, 0x75, 0x23, 0x74, 0x2f, 0x5b, 0x15,
	0x94, 0x7f, 0x8f, 0xf6, 0x23, 0x23, 0x25, 0x70, 0x3d, 0xad, 0x02, 0x72, 0xc6, 0x55, 0xa7, 0x61,
	0xd1, 0x27, 0xff, 0xa2, 0xc7, 0x8c, 0xd3, 0x76, 0x05, 0xa8, 0x5b, 0xf7, 0x9f, 0xd0, 0xe1, 0xaf,
	0x6d, 0xa8, 0x5c, 0x70, 0x05, 0xfe, 0x35, 0x6a, 0xfd, 0x8c, 0xf1, 0x36, 0x88, 0x41, 0xd9, 0x37,
	0x6e, 0xf8, 0xea, 0xa1, 0x5d, 0x5a, 0x1e, 0xff, 0x1d, 0xc8, 0x82, 0x45, 0xe0, 0x17, 0x68, 0x67,
	0x25, 0xd2, 0x1f, 0xe0, 0x35, 0x37, 0x82, 0xff, 0x3a, 0xe5, 0xee, 0x70, 0x93, 0x91, 0x72, 0x47,
	0xc1, 0x87, 0x87, 0x8e, 0x22, 0x91, 0xad, 0x9b, 0x0c, 0xb6, 0xab, 0xae, 0xe3, 0xe5, 0x25, 0x4f,
	0x26, 0x09, 0xd3, 0x0b, 0x13, 0xe2, 0x48, 0x64, 0x24, 0x34, 0x71, 0x68, 0x58, 0x3a, 0x5f, 0x7e,
	0x10, 0xc6, 0x35, 0x48, 0x3e, 0x4b, 0x49, 0x02, 0x9c, 0xd8, 0x07, 0x41, 0x12, 0x41, 0xd6, 0x3c,
	0xc1, 0x2b, 0xa7, 0x38, 0xe1, 0xad, 0xd1, 0x0c, 0x46, 0xf4, 0xbd, 0xd1, 0x0b, 0x4c, 0x8c, 0x47,
	0xb6, 0x0d, 0x75, 0x6d, 0x1e, 0x2a, 0x4f, 0xb8, 0x65, 0xd1, 0x97, 0x5f, 0x01, 0x00, 0x00, 0xff,
	0xff, 0x41, 0xaa, 0x0b, 0xd8, 0xdd, 0x02, 0x00, 0x00,
}
