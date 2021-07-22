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
// source: buf/alpha/registry/v1alpha1/reference.proto

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

// ==========================
// ReferenceService Interface
// ==========================

// ReferenceService is a service that provides RPCs that allow the BSR to query
// for reference information.
type ReferenceService interface {
	// GetReferenceByName takes a reference name and returns the
	// reference either as a tag, branch, or commit.
	GetReferenceByName(context.Context, *GetReferenceByNameRequest) (*GetReferenceByNameResponse, error)
}

// ================================
// ReferenceService Protobuf Client
// ================================

type referenceServiceProtobufClient struct {
	client      HTTPClient
	urls        [1]string
	interceptor twirp.Interceptor
	opts        twirp.ClientOptions
}

// NewReferenceServiceProtobufClient creates a Protobuf client that implements the ReferenceService interface.
// It communicates using Protobuf and can be configured with a custom HTTPClient.
func NewReferenceServiceProtobufClient(baseURL string, client HTTPClient, opts ...twirp.ClientOption) ReferenceService {
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
	serviceURL += baseServicePath(pathPrefix, "buf.alpha.registry.v1alpha1", "ReferenceService")
	urls := [1]string{
		serviceURL + "GetReferenceByName",
	}

	return &referenceServiceProtobufClient{
		client:      client,
		urls:        urls,
		interceptor: twirp.ChainInterceptors(clientOpts.Interceptors...),
		opts:        clientOpts,
	}
}

func (c *referenceServiceProtobufClient) GetReferenceByName(ctx context.Context, in *GetReferenceByNameRequest) (*GetReferenceByNameResponse, error) {
	ctx = ctxsetters.WithPackageName(ctx, "buf.alpha.registry.v1alpha1")
	ctx = ctxsetters.WithServiceName(ctx, "ReferenceService")
	ctx = ctxsetters.WithMethodName(ctx, "GetReferenceByName")
	caller := c.callGetReferenceByName
	if c.interceptor != nil {
		caller = func(ctx context.Context, req *GetReferenceByNameRequest) (*GetReferenceByNameResponse, error) {
			resp, err := c.interceptor(
				func(ctx context.Context, req interface{}) (interface{}, error) {
					typedReq, ok := req.(*GetReferenceByNameRequest)
					if !ok {
						return nil, twirp.InternalError("failed type assertion req.(*GetReferenceByNameRequest) when calling interceptor")
					}
					return c.callGetReferenceByName(ctx, typedReq)
				},
			)(ctx, req)
			if resp != nil {
				typedResp, ok := resp.(*GetReferenceByNameResponse)
				if !ok {
					return nil, twirp.InternalError("failed type assertion resp.(*GetReferenceByNameResponse) when calling interceptor")
				}
				return typedResp, err
			}
			return nil, err
		}
	}
	return caller(ctx, in)
}

func (c *referenceServiceProtobufClient) callGetReferenceByName(ctx context.Context, in *GetReferenceByNameRequest) (*GetReferenceByNameResponse, error) {
	out := new(GetReferenceByNameResponse)
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

// ============================
// ReferenceService JSON Client
// ============================

type referenceServiceJSONClient struct {
	client      HTTPClient
	urls        [1]string
	interceptor twirp.Interceptor
	opts        twirp.ClientOptions
}

// NewReferenceServiceJSONClient creates a JSON client that implements the ReferenceService interface.
// It communicates using JSON and can be configured with a custom HTTPClient.
func NewReferenceServiceJSONClient(baseURL string, client HTTPClient, opts ...twirp.ClientOption) ReferenceService {
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
	serviceURL += baseServicePath(pathPrefix, "buf.alpha.registry.v1alpha1", "ReferenceService")
	urls := [1]string{
		serviceURL + "GetReferenceByName",
	}

	return &referenceServiceJSONClient{
		client:      client,
		urls:        urls,
		interceptor: twirp.ChainInterceptors(clientOpts.Interceptors...),
		opts:        clientOpts,
	}
}

func (c *referenceServiceJSONClient) GetReferenceByName(ctx context.Context, in *GetReferenceByNameRequest) (*GetReferenceByNameResponse, error) {
	ctx = ctxsetters.WithPackageName(ctx, "buf.alpha.registry.v1alpha1")
	ctx = ctxsetters.WithServiceName(ctx, "ReferenceService")
	ctx = ctxsetters.WithMethodName(ctx, "GetReferenceByName")
	caller := c.callGetReferenceByName
	if c.interceptor != nil {
		caller = func(ctx context.Context, req *GetReferenceByNameRequest) (*GetReferenceByNameResponse, error) {
			resp, err := c.interceptor(
				func(ctx context.Context, req interface{}) (interface{}, error) {
					typedReq, ok := req.(*GetReferenceByNameRequest)
					if !ok {
						return nil, twirp.InternalError("failed type assertion req.(*GetReferenceByNameRequest) when calling interceptor")
					}
					return c.callGetReferenceByName(ctx, typedReq)
				},
			)(ctx, req)
			if resp != nil {
				typedResp, ok := resp.(*GetReferenceByNameResponse)
				if !ok {
					return nil, twirp.InternalError("failed type assertion resp.(*GetReferenceByNameResponse) when calling interceptor")
				}
				return typedResp, err
			}
			return nil, err
		}
	}
	return caller(ctx, in)
}

func (c *referenceServiceJSONClient) callGetReferenceByName(ctx context.Context, in *GetReferenceByNameRequest) (*GetReferenceByNameResponse, error) {
	out := new(GetReferenceByNameResponse)
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

// ===============================
// ReferenceService Server Handler
// ===============================

type referenceServiceServer struct {
	ReferenceService
	interceptor      twirp.Interceptor
	hooks            *twirp.ServerHooks
	pathPrefix       string // prefix for routing
	jsonSkipDefaults bool   // do not include unpopulated fields (default values) in the response
	jsonCamelCase    bool   // JSON fields are serialized as lowerCamelCase rather than keeping the original proto names
}

// NewReferenceServiceServer builds a TwirpServer that can be used as an http.Handler to handle
// HTTP requests that are routed to the right method in the provided svc implementation.
// The opts are twirp.ServerOption modifiers, for example twirp.WithServerHooks(hooks).
func NewReferenceServiceServer(svc ReferenceService, opts ...interface{}) TwirpServer {
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

	return &referenceServiceServer{
		ReferenceService: svc,
		hooks:            serverOpts.Hooks,
		interceptor:      twirp.ChainInterceptors(serverOpts.Interceptors...),
		pathPrefix:       pathPrefix,
		jsonSkipDefaults: jsonSkipDefaults,
		jsonCamelCase:    jsonCamelCase,
	}
}

// writeError writes an HTTP response with a valid Twirp error format, and triggers hooks.
// If err is not a twirp.Error, it will get wrapped with twirp.InternalErrorWith(err)
func (s *referenceServiceServer) writeError(ctx context.Context, resp http.ResponseWriter, err error) {
	writeError(ctx, resp, err, s.hooks)
}

// handleRequestBodyError is used to handle error when the twirp server cannot read request
func (s *referenceServiceServer) handleRequestBodyError(ctx context.Context, resp http.ResponseWriter, msg string, err error) {
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

// ReferenceServicePathPrefix is a convenience constant that may identify URL paths.
// Should be used with caution, it only matches routes generated by Twirp Go clients,
// with the default "/twirp" prefix and default CamelCase service and method names.
// More info: https://twitchtv.github.io/twirp/docs/routing.html
const ReferenceServicePathPrefix = "/twirp/buf.alpha.registry.v1alpha1.ReferenceService/"

func (s *referenceServiceServer) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	ctx = ctxsetters.WithPackageName(ctx, "buf.alpha.registry.v1alpha1")
	ctx = ctxsetters.WithServiceName(ctx, "ReferenceService")
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
	if pkgService != "buf.alpha.registry.v1alpha1.ReferenceService" {
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
	case "GetReferenceByName":
		s.serveGetReferenceByName(ctx, resp, req)
		return
	default:
		msg := fmt.Sprintf("no handler for path %q", req.URL.Path)
		s.writeError(ctx, resp, badRouteError(msg, req.Method, req.URL.Path))
		return
	}
}

func (s *referenceServiceServer) serveGetReferenceByName(ctx context.Context, resp http.ResponseWriter, req *http.Request) {
	header := req.Header.Get("Content-Type")
	i := strings.Index(header, ";")
	if i == -1 {
		i = len(header)
	}
	switch strings.TrimSpace(strings.ToLower(header[:i])) {
	case "application/json":
		s.serveGetReferenceByNameJSON(ctx, resp, req)
	case "application/protobuf":
		s.serveGetReferenceByNameProtobuf(ctx, resp, req)
	default:
		msg := fmt.Sprintf("unexpected Content-Type: %q", req.Header.Get("Content-Type"))
		twerr := badRouteError(msg, req.Method, req.URL.Path)
		s.writeError(ctx, resp, twerr)
	}
}

func (s *referenceServiceServer) serveGetReferenceByNameJSON(ctx context.Context, resp http.ResponseWriter, req *http.Request) {
	var err error
	ctx = ctxsetters.WithMethodName(ctx, "GetReferenceByName")
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
	reqContent := new(GetReferenceByNameRequest)
	unmarshaler := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err = unmarshaler.Unmarshal(rawReqBody, reqContent); err != nil {
		s.handleRequestBodyError(ctx, resp, "the json request could not be decoded", err)
		return
	}

	handler := s.ReferenceService.GetReferenceByName
	if s.interceptor != nil {
		handler = func(ctx context.Context, req *GetReferenceByNameRequest) (*GetReferenceByNameResponse, error) {
			resp, err := s.interceptor(
				func(ctx context.Context, req interface{}) (interface{}, error) {
					typedReq, ok := req.(*GetReferenceByNameRequest)
					if !ok {
						return nil, twirp.InternalError("failed type assertion req.(*GetReferenceByNameRequest) when calling interceptor")
					}
					return s.ReferenceService.GetReferenceByName(ctx, typedReq)
				},
			)(ctx, req)
			if resp != nil {
				typedResp, ok := resp.(*GetReferenceByNameResponse)
				if !ok {
					return nil, twirp.InternalError("failed type assertion resp.(*GetReferenceByNameResponse) when calling interceptor")
				}
				return typedResp, err
			}
			return nil, err
		}
	}

	// Call service method
	var respContent *GetReferenceByNameResponse
	func() {
		defer ensurePanicResponses(ctx, resp, s.hooks)
		respContent, err = handler(ctx, reqContent)
	}()

	if err != nil {
		s.writeError(ctx, resp, err)
		return
	}
	if respContent == nil {
		s.writeError(ctx, resp, twirp.InternalError("received a nil *GetReferenceByNameResponse and nil error while calling GetReferenceByName. nil responses are not supported"))
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

func (s *referenceServiceServer) serveGetReferenceByNameProtobuf(ctx context.Context, resp http.ResponseWriter, req *http.Request) {
	var err error
	ctx = ctxsetters.WithMethodName(ctx, "GetReferenceByName")
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
	reqContent := new(GetReferenceByNameRequest)
	if err = proto.Unmarshal(buf, reqContent); err != nil {
		s.writeError(ctx, resp, malformedRequestError("the protobuf request could not be decoded"))
		return
	}

	handler := s.ReferenceService.GetReferenceByName
	if s.interceptor != nil {
		handler = func(ctx context.Context, req *GetReferenceByNameRequest) (*GetReferenceByNameResponse, error) {
			resp, err := s.interceptor(
				func(ctx context.Context, req interface{}) (interface{}, error) {
					typedReq, ok := req.(*GetReferenceByNameRequest)
					if !ok {
						return nil, twirp.InternalError("failed type assertion req.(*GetReferenceByNameRequest) when calling interceptor")
					}
					return s.ReferenceService.GetReferenceByName(ctx, typedReq)
				},
			)(ctx, req)
			if resp != nil {
				typedResp, ok := resp.(*GetReferenceByNameResponse)
				if !ok {
					return nil, twirp.InternalError("failed type assertion resp.(*GetReferenceByNameResponse) when calling interceptor")
				}
				return typedResp, err
			}
			return nil, err
		}
	}

	// Call service method
	var respContent *GetReferenceByNameResponse
	func() {
		defer ensurePanicResponses(ctx, resp, s.hooks)
		respContent, err = handler(ctx, reqContent)
	}()

	if err != nil {
		s.writeError(ctx, resp, err)
		return
	}
	if respContent == nil {
		s.writeError(ctx, resp, twirp.InternalError("received a nil *GetReferenceByNameResponse and nil error while calling GetReferenceByName. nil responses are not supported"))
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

func (s *referenceServiceServer) ServiceDescriptor() ([]byte, int) {
	return twirpFileDescriptor8, 0
}

func (s *referenceServiceServer) ProtocGenTwirpVersion() string {
	return "v8.1.0"
}

// PathPrefix returns the base service path, in the form: "/<prefix>/<package>.<Service>/"
// that is everything in a Twirp route except for the <Method>. This can be used for routing,
// for example to identify the requests that are targeted to this service in a mux.
func (s *referenceServiceServer) PathPrefix() string {
	return baseServicePath(s.pathPrefix, "buf.alpha.registry.v1alpha1", "ReferenceService")
}

var twirpFileDescriptor8 = []byte{
	// 393 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x92, 0xcf, 0x4e, 0xe3, 0x30,
	0x10, 0xc6, 0x9b, 0xed, 0x6e, 0xa5, 0xb8, 0x52, 0x77, 0x65, 0xed, 0xa1, 0xdb, 0x3d, 0x80, 0x7a,
	0x00, 0x04, 0x22, 0xa6, 0xad, 0x04, 0x07, 0x24, 0x0e, 0x01, 0xa9, 0x3d, 0x21, 0x64, 0x38, 0xf5,
	0x82, 0xec, 0x30, 0x49, 0x23, 0x35, 0x76, 0x70, 0x9c, 0xa2, 0x3e, 0x00, 0x47, 0x5e, 0x80, 0x47,
	0xe4, 0x29, 0x50, 0x9c, 0x3f, 0xad, 0x04, 0x84, 0xf6, 0xe6, 0x8c, 0xe7, 0xf7, 0xf9, 0xfb, 0x26,
	0x83, 0x8e, 0x78, 0xea, 0x13, 0x36, 0x8f, 0x67, 0x8c, 0x28, 0x08, 0xc2, 0x44, 0xab, 0x25, 0x59,
	0x0c, 0x4c, 0x61, 0x40, 0x14, 0xf8, 0xa0, 0x40, 0x78, 0xe0, 0xc4, 0x4a, 0x6a, 0x89, 0xff, 0xf3,
	0xd4, 0x77, 0xcc, 0x9d, 0x53, 0x36, 0x3b, 0x65, 0x73, 0x6f, 0x54, 0xaf, 0x14, 0xcb, 0x24, 0xd4,
	0x52, 0x2d, 0xef, 0xb9, 0x62, 0xc2, 0x9b, 0xe5, 0x8a, 0x1b, 0x43, 0x9e, 0x8c, 0xa2, 0x50, 0x17,
	0xd0, 0xc9, 0x86, 0x90, 0x66, 0x41, 0x4e, 0xf4, 0xdf, 0x2c, 0x64, 0xd3, 0x32, 0x0c, 0x1e, 0xa3,
	0x56, 0x6e, 0xa2, 0x6b, 0xed, 0x5a, 0x07, 0xed, 0xe1, 0xb1, 0x53, 0x93, 0xcb, 0xa1, 0x95, 0xa0,
	0x6b, 0xa0, 0x49, 0x83, 0x16, 0x38, 0xbe, 0x40, 0x4d, 0xcd, 0x82, 0xee, 0x0f, 0xa3, 0x72, 0xb8,
	0xa1, 0xca, 0x1d, 0x0b, 0x26, 0x0d, 0x9a, 0x81, 0x99, 0x91, 0x3c, 0x58, 0xb7, 0xb9, 0x95, 0x91,
	0x4b, 0x03, 0x65, 0x46, 0x72, 0xdc, 0x6d, 0x23, 0xbb, 0xfa, 0x57, 0x7d, 0x81, 0xfe, 0x8d, 0x41,
	0x57, 0x71, 0xdd, 0xe5, 0x35, 0x8b, 0x80, 0xc2, 0x63, 0x0a, 0x89, 0xc6, 0x18, 0xfd, 0x14, 0x2c,
	0x02, 0x93, 0xdc, 0xa6, 0xe6, 0x8c, 0xff, 0xa2, 0x5f, 0xf2, 0x49, 0x80, 0x32, 0x41, 0x6c, 0x9a,
	0x7f, 0xe0, 0x7d, 0xf4, 0x7b, 0x6d, 0x96, 0x06, 0x6a, 0x9a, 0xfb, 0xce, 0xaa, 0x9c, 0x29, 0xf7,
	0x39, 0xea, 0x7d, 0xf6, 0x5e, 0x12, 0x4b, 0x91, 0x00, 0xbe, 0x5a, 0xb3, 0x56, 0xcc, 0x7b, 0xef,
	0x9b, 0x98, 0x45, 0x37, 0x5d, 0x81, 0xc3, 0x57, 0x0b, 0xfd, 0xa9, 0x2e, 0x6e, 0x41, 0x2d, 0x42,
	0x0f, 0xf0, 0xb3, 0x85, 0xf0, 0xc7, 0x97, 0xf1, 0x69, 0xad, 0xfc, 0x97, 0xa3, 0xe9, 0x9d, 0x6d,
	0xcd, 0xe5, 0x11, 0xdd, 0x17, 0x0b, 0xed, 0x78, 0x32, 0xaa, 0xc3, 0xdd, 0x4e, 0x05, 0xdf, 0x64,
	0x1b, 0x39, 0x9d, 0x06, 0xa1, 0x9e, 0xa5, 0xdc, 0xf1, 0x64, 0x44, 0x78, 0xea, 0xf3, 0x34, 0x9c,
	0x3f, 0x64, 0x07, 0x12, 0x0a, 0x0d, 0x4a, 0xb0, 0x39, 0x09, 0x40, 0x10, 0xb3, 0xbd, 0x24, 0x90,
	0xa4, 0x66, 0xe3, 0xcf, 0xcb, 0x4a, 0x59, 0xe0, 0x2d, 0x83, 0x8d, 0xde, 0x03, 0x00, 0x00, 0xff,
	0xff, 0xfc, 0xdc, 0x01, 0xbe, 0xdc, 0x03, 0x00, 0x00,
}
