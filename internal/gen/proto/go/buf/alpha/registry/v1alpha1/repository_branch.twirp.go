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
// source: buf/alpha/registry/v1alpha1/repository_branch.proto

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

// =================================
// RepositoryBranchService Interface
// =================================

// RepositoryBranchService is the Repository branch service.
type RepositoryBranchService interface {
	// CreateRepositoryBranch creates a new repository branch.
	CreateRepositoryBranch(context.Context, *CreateRepositoryBranchRequest) (*CreateRepositoryBranchResponse, error)

	// ListRepositoryBranches lists the repository branches associated with a Repository.
	ListRepositoryBranches(context.Context, *ListRepositoryBranchesRequest) (*ListRepositoryBranchesResponse, error)
}

// =======================================
// RepositoryBranchService Protobuf Client
// =======================================

type repositoryBranchServiceProtobufClient struct {
	client      HTTPClient
	urls        [2]string
	interceptor twirp.Interceptor
	opts        twirp.ClientOptions
}

// NewRepositoryBranchServiceProtobufClient creates a Protobuf client that implements the RepositoryBranchService interface.
// It communicates using Protobuf and can be configured with a custom HTTPClient.
func NewRepositoryBranchServiceProtobufClient(baseURL string, client HTTPClient, opts ...twirp.ClientOption) RepositoryBranchService {
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
	serviceURL += baseServicePath(pathPrefix, "buf.alpha.registry.v1alpha1", "RepositoryBranchService")
	urls := [2]string{
		serviceURL + "CreateRepositoryBranch",
		serviceURL + "ListRepositoryBranches",
	}

	return &repositoryBranchServiceProtobufClient{
		client:      client,
		urls:        urls,
		interceptor: twirp.ChainInterceptors(clientOpts.Interceptors...),
		opts:        clientOpts,
	}
}

func (c *repositoryBranchServiceProtobufClient) CreateRepositoryBranch(ctx context.Context, in *CreateRepositoryBranchRequest) (*CreateRepositoryBranchResponse, error) {
	ctx = ctxsetters.WithPackageName(ctx, "buf.alpha.registry.v1alpha1")
	ctx = ctxsetters.WithServiceName(ctx, "RepositoryBranchService")
	ctx = ctxsetters.WithMethodName(ctx, "CreateRepositoryBranch")
	caller := c.callCreateRepositoryBranch
	if c.interceptor != nil {
		caller = func(ctx context.Context, req *CreateRepositoryBranchRequest) (*CreateRepositoryBranchResponse, error) {
			resp, err := c.interceptor(
				func(ctx context.Context, req interface{}) (interface{}, error) {
					typedReq, ok := req.(*CreateRepositoryBranchRequest)
					if !ok {
						return nil, twirp.InternalError("failed type assertion req.(*CreateRepositoryBranchRequest) when calling interceptor")
					}
					return c.callCreateRepositoryBranch(ctx, typedReq)
				},
			)(ctx, req)
			if resp != nil {
				typedResp, ok := resp.(*CreateRepositoryBranchResponse)
				if !ok {
					return nil, twirp.InternalError("failed type assertion resp.(*CreateRepositoryBranchResponse) when calling interceptor")
				}
				return typedResp, err
			}
			return nil, err
		}
	}
	return caller(ctx, in)
}

func (c *repositoryBranchServiceProtobufClient) callCreateRepositoryBranch(ctx context.Context, in *CreateRepositoryBranchRequest) (*CreateRepositoryBranchResponse, error) {
	out := new(CreateRepositoryBranchResponse)
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

func (c *repositoryBranchServiceProtobufClient) ListRepositoryBranches(ctx context.Context, in *ListRepositoryBranchesRequest) (*ListRepositoryBranchesResponse, error) {
	ctx = ctxsetters.WithPackageName(ctx, "buf.alpha.registry.v1alpha1")
	ctx = ctxsetters.WithServiceName(ctx, "RepositoryBranchService")
	ctx = ctxsetters.WithMethodName(ctx, "ListRepositoryBranches")
	caller := c.callListRepositoryBranches
	if c.interceptor != nil {
		caller = func(ctx context.Context, req *ListRepositoryBranchesRequest) (*ListRepositoryBranchesResponse, error) {
			resp, err := c.interceptor(
				func(ctx context.Context, req interface{}) (interface{}, error) {
					typedReq, ok := req.(*ListRepositoryBranchesRequest)
					if !ok {
						return nil, twirp.InternalError("failed type assertion req.(*ListRepositoryBranchesRequest) when calling interceptor")
					}
					return c.callListRepositoryBranches(ctx, typedReq)
				},
			)(ctx, req)
			if resp != nil {
				typedResp, ok := resp.(*ListRepositoryBranchesResponse)
				if !ok {
					return nil, twirp.InternalError("failed type assertion resp.(*ListRepositoryBranchesResponse) when calling interceptor")
				}
				return typedResp, err
			}
			return nil, err
		}
	}
	return caller(ctx, in)
}

func (c *repositoryBranchServiceProtobufClient) callListRepositoryBranches(ctx context.Context, in *ListRepositoryBranchesRequest) (*ListRepositoryBranchesResponse, error) {
	out := new(ListRepositoryBranchesResponse)
	ctx, err := doProtobufRequest(ctx, c.client, c.opts.Hooks, c.urls[1], in, out)
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

// ===================================
// RepositoryBranchService JSON Client
// ===================================

type repositoryBranchServiceJSONClient struct {
	client      HTTPClient
	urls        [2]string
	interceptor twirp.Interceptor
	opts        twirp.ClientOptions
}

// NewRepositoryBranchServiceJSONClient creates a JSON client that implements the RepositoryBranchService interface.
// It communicates using JSON and can be configured with a custom HTTPClient.
func NewRepositoryBranchServiceJSONClient(baseURL string, client HTTPClient, opts ...twirp.ClientOption) RepositoryBranchService {
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
	serviceURL += baseServicePath(pathPrefix, "buf.alpha.registry.v1alpha1", "RepositoryBranchService")
	urls := [2]string{
		serviceURL + "CreateRepositoryBranch",
		serviceURL + "ListRepositoryBranches",
	}

	return &repositoryBranchServiceJSONClient{
		client:      client,
		urls:        urls,
		interceptor: twirp.ChainInterceptors(clientOpts.Interceptors...),
		opts:        clientOpts,
	}
}

func (c *repositoryBranchServiceJSONClient) CreateRepositoryBranch(ctx context.Context, in *CreateRepositoryBranchRequest) (*CreateRepositoryBranchResponse, error) {
	ctx = ctxsetters.WithPackageName(ctx, "buf.alpha.registry.v1alpha1")
	ctx = ctxsetters.WithServiceName(ctx, "RepositoryBranchService")
	ctx = ctxsetters.WithMethodName(ctx, "CreateRepositoryBranch")
	caller := c.callCreateRepositoryBranch
	if c.interceptor != nil {
		caller = func(ctx context.Context, req *CreateRepositoryBranchRequest) (*CreateRepositoryBranchResponse, error) {
			resp, err := c.interceptor(
				func(ctx context.Context, req interface{}) (interface{}, error) {
					typedReq, ok := req.(*CreateRepositoryBranchRequest)
					if !ok {
						return nil, twirp.InternalError("failed type assertion req.(*CreateRepositoryBranchRequest) when calling interceptor")
					}
					return c.callCreateRepositoryBranch(ctx, typedReq)
				},
			)(ctx, req)
			if resp != nil {
				typedResp, ok := resp.(*CreateRepositoryBranchResponse)
				if !ok {
					return nil, twirp.InternalError("failed type assertion resp.(*CreateRepositoryBranchResponse) when calling interceptor")
				}
				return typedResp, err
			}
			return nil, err
		}
	}
	return caller(ctx, in)
}

func (c *repositoryBranchServiceJSONClient) callCreateRepositoryBranch(ctx context.Context, in *CreateRepositoryBranchRequest) (*CreateRepositoryBranchResponse, error) {
	out := new(CreateRepositoryBranchResponse)
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

func (c *repositoryBranchServiceJSONClient) ListRepositoryBranches(ctx context.Context, in *ListRepositoryBranchesRequest) (*ListRepositoryBranchesResponse, error) {
	ctx = ctxsetters.WithPackageName(ctx, "buf.alpha.registry.v1alpha1")
	ctx = ctxsetters.WithServiceName(ctx, "RepositoryBranchService")
	ctx = ctxsetters.WithMethodName(ctx, "ListRepositoryBranches")
	caller := c.callListRepositoryBranches
	if c.interceptor != nil {
		caller = func(ctx context.Context, req *ListRepositoryBranchesRequest) (*ListRepositoryBranchesResponse, error) {
			resp, err := c.interceptor(
				func(ctx context.Context, req interface{}) (interface{}, error) {
					typedReq, ok := req.(*ListRepositoryBranchesRequest)
					if !ok {
						return nil, twirp.InternalError("failed type assertion req.(*ListRepositoryBranchesRequest) when calling interceptor")
					}
					return c.callListRepositoryBranches(ctx, typedReq)
				},
			)(ctx, req)
			if resp != nil {
				typedResp, ok := resp.(*ListRepositoryBranchesResponse)
				if !ok {
					return nil, twirp.InternalError("failed type assertion resp.(*ListRepositoryBranchesResponse) when calling interceptor")
				}
				return typedResp, err
			}
			return nil, err
		}
	}
	return caller(ctx, in)
}

func (c *repositoryBranchServiceJSONClient) callListRepositoryBranches(ctx context.Context, in *ListRepositoryBranchesRequest) (*ListRepositoryBranchesResponse, error) {
	out := new(ListRepositoryBranchesResponse)
	ctx, err := doJSONRequest(ctx, c.client, c.opts.Hooks, c.urls[1], in, out)
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

// ======================================
// RepositoryBranchService Server Handler
// ======================================

type repositoryBranchServiceServer struct {
	RepositoryBranchService
	interceptor      twirp.Interceptor
	hooks            *twirp.ServerHooks
	pathPrefix       string // prefix for routing
	jsonSkipDefaults bool   // do not include unpopulated fields (default values) in the response
	jsonCamelCase    bool   // JSON fields are serialized as lowerCamelCase rather than keeping the original proto names
}

// NewRepositoryBranchServiceServer builds a TwirpServer that can be used as an http.Handler to handle
// HTTP requests that are routed to the right method in the provided svc implementation.
// The opts are twirp.ServerOption modifiers, for example twirp.WithServerHooks(hooks).
func NewRepositoryBranchServiceServer(svc RepositoryBranchService, opts ...interface{}) TwirpServer {
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

	return &repositoryBranchServiceServer{
		RepositoryBranchService: svc,
		hooks:                   serverOpts.Hooks,
		interceptor:             twirp.ChainInterceptors(serverOpts.Interceptors...),
		pathPrefix:              pathPrefix,
		jsonSkipDefaults:        jsonSkipDefaults,
		jsonCamelCase:           jsonCamelCase,
	}
}

// writeError writes an HTTP response with a valid Twirp error format, and triggers hooks.
// If err is not a twirp.Error, it will get wrapped with twirp.InternalErrorWith(err)
func (s *repositoryBranchServiceServer) writeError(ctx context.Context, resp http.ResponseWriter, err error) {
	writeError(ctx, resp, err, s.hooks)
}

// handleRequestBodyError is used to handle error when the twirp server cannot read request
func (s *repositoryBranchServiceServer) handleRequestBodyError(ctx context.Context, resp http.ResponseWriter, msg string, err error) {
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

// RepositoryBranchServicePathPrefix is a convenience constant that may identify URL paths.
// Should be used with caution, it only matches routes generated by Twirp Go clients,
// with the default "/twirp" prefix and default CamelCase service and method names.
// More info: https://twitchtv.github.io/twirp/docs/routing.html
const RepositoryBranchServicePathPrefix = "/twirp/buf.alpha.registry.v1alpha1.RepositoryBranchService/"

func (s *repositoryBranchServiceServer) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	ctx = ctxsetters.WithPackageName(ctx, "buf.alpha.registry.v1alpha1")
	ctx = ctxsetters.WithServiceName(ctx, "RepositoryBranchService")
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
	if pkgService != "buf.alpha.registry.v1alpha1.RepositoryBranchService" {
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
	case "CreateRepositoryBranch":
		s.serveCreateRepositoryBranch(ctx, resp, req)
		return
	case "ListRepositoryBranches":
		s.serveListRepositoryBranches(ctx, resp, req)
		return
	default:
		msg := fmt.Sprintf("no handler for path %q", req.URL.Path)
		s.writeError(ctx, resp, badRouteError(msg, req.Method, req.URL.Path))
		return
	}
}

func (s *repositoryBranchServiceServer) serveCreateRepositoryBranch(ctx context.Context, resp http.ResponseWriter, req *http.Request) {
	header := req.Header.Get("Content-Type")
	i := strings.Index(header, ";")
	if i == -1 {
		i = len(header)
	}
	switch strings.TrimSpace(strings.ToLower(header[:i])) {
	case "application/json":
		s.serveCreateRepositoryBranchJSON(ctx, resp, req)
	case "application/protobuf":
		s.serveCreateRepositoryBranchProtobuf(ctx, resp, req)
	default:
		msg := fmt.Sprintf("unexpected Content-Type: %q", req.Header.Get("Content-Type"))
		twerr := badRouteError(msg, req.Method, req.URL.Path)
		s.writeError(ctx, resp, twerr)
	}
}

func (s *repositoryBranchServiceServer) serveCreateRepositoryBranchJSON(ctx context.Context, resp http.ResponseWriter, req *http.Request) {
	var err error
	ctx = ctxsetters.WithMethodName(ctx, "CreateRepositoryBranch")
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
	reqContent := new(CreateRepositoryBranchRequest)
	unmarshaler := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err = unmarshaler.Unmarshal(rawReqBody, reqContent); err != nil {
		s.handleRequestBodyError(ctx, resp, "the json request could not be decoded", err)
		return
	}

	handler := s.RepositoryBranchService.CreateRepositoryBranch
	if s.interceptor != nil {
		handler = func(ctx context.Context, req *CreateRepositoryBranchRequest) (*CreateRepositoryBranchResponse, error) {
			resp, err := s.interceptor(
				func(ctx context.Context, req interface{}) (interface{}, error) {
					typedReq, ok := req.(*CreateRepositoryBranchRequest)
					if !ok {
						return nil, twirp.InternalError("failed type assertion req.(*CreateRepositoryBranchRequest) when calling interceptor")
					}
					return s.RepositoryBranchService.CreateRepositoryBranch(ctx, typedReq)
				},
			)(ctx, req)
			if resp != nil {
				typedResp, ok := resp.(*CreateRepositoryBranchResponse)
				if !ok {
					return nil, twirp.InternalError("failed type assertion resp.(*CreateRepositoryBranchResponse) when calling interceptor")
				}
				return typedResp, err
			}
			return nil, err
		}
	}

	// Call service method
	var respContent *CreateRepositoryBranchResponse
	func() {
		defer ensurePanicResponses(ctx, resp, s.hooks)
		respContent, err = handler(ctx, reqContent)
	}()

	if err != nil {
		s.writeError(ctx, resp, err)
		return
	}
	if respContent == nil {
		s.writeError(ctx, resp, twirp.InternalError("received a nil *CreateRepositoryBranchResponse and nil error while calling CreateRepositoryBranch. nil responses are not supported"))
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

func (s *repositoryBranchServiceServer) serveCreateRepositoryBranchProtobuf(ctx context.Context, resp http.ResponseWriter, req *http.Request) {
	var err error
	ctx = ctxsetters.WithMethodName(ctx, "CreateRepositoryBranch")
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
	reqContent := new(CreateRepositoryBranchRequest)
	if err = proto.Unmarshal(buf, reqContent); err != nil {
		s.writeError(ctx, resp, malformedRequestError("the protobuf request could not be decoded"))
		return
	}

	handler := s.RepositoryBranchService.CreateRepositoryBranch
	if s.interceptor != nil {
		handler = func(ctx context.Context, req *CreateRepositoryBranchRequest) (*CreateRepositoryBranchResponse, error) {
			resp, err := s.interceptor(
				func(ctx context.Context, req interface{}) (interface{}, error) {
					typedReq, ok := req.(*CreateRepositoryBranchRequest)
					if !ok {
						return nil, twirp.InternalError("failed type assertion req.(*CreateRepositoryBranchRequest) when calling interceptor")
					}
					return s.RepositoryBranchService.CreateRepositoryBranch(ctx, typedReq)
				},
			)(ctx, req)
			if resp != nil {
				typedResp, ok := resp.(*CreateRepositoryBranchResponse)
				if !ok {
					return nil, twirp.InternalError("failed type assertion resp.(*CreateRepositoryBranchResponse) when calling interceptor")
				}
				return typedResp, err
			}
			return nil, err
		}
	}

	// Call service method
	var respContent *CreateRepositoryBranchResponse
	func() {
		defer ensurePanicResponses(ctx, resp, s.hooks)
		respContent, err = handler(ctx, reqContent)
	}()

	if err != nil {
		s.writeError(ctx, resp, err)
		return
	}
	if respContent == nil {
		s.writeError(ctx, resp, twirp.InternalError("received a nil *CreateRepositoryBranchResponse and nil error while calling CreateRepositoryBranch. nil responses are not supported"))
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

func (s *repositoryBranchServiceServer) serveListRepositoryBranches(ctx context.Context, resp http.ResponseWriter, req *http.Request) {
	header := req.Header.Get("Content-Type")
	i := strings.Index(header, ";")
	if i == -1 {
		i = len(header)
	}
	switch strings.TrimSpace(strings.ToLower(header[:i])) {
	case "application/json":
		s.serveListRepositoryBranchesJSON(ctx, resp, req)
	case "application/protobuf":
		s.serveListRepositoryBranchesProtobuf(ctx, resp, req)
	default:
		msg := fmt.Sprintf("unexpected Content-Type: %q", req.Header.Get("Content-Type"))
		twerr := badRouteError(msg, req.Method, req.URL.Path)
		s.writeError(ctx, resp, twerr)
	}
}

func (s *repositoryBranchServiceServer) serveListRepositoryBranchesJSON(ctx context.Context, resp http.ResponseWriter, req *http.Request) {
	var err error
	ctx = ctxsetters.WithMethodName(ctx, "ListRepositoryBranches")
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
	reqContent := new(ListRepositoryBranchesRequest)
	unmarshaler := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err = unmarshaler.Unmarshal(rawReqBody, reqContent); err != nil {
		s.handleRequestBodyError(ctx, resp, "the json request could not be decoded", err)
		return
	}

	handler := s.RepositoryBranchService.ListRepositoryBranches
	if s.interceptor != nil {
		handler = func(ctx context.Context, req *ListRepositoryBranchesRequest) (*ListRepositoryBranchesResponse, error) {
			resp, err := s.interceptor(
				func(ctx context.Context, req interface{}) (interface{}, error) {
					typedReq, ok := req.(*ListRepositoryBranchesRequest)
					if !ok {
						return nil, twirp.InternalError("failed type assertion req.(*ListRepositoryBranchesRequest) when calling interceptor")
					}
					return s.RepositoryBranchService.ListRepositoryBranches(ctx, typedReq)
				},
			)(ctx, req)
			if resp != nil {
				typedResp, ok := resp.(*ListRepositoryBranchesResponse)
				if !ok {
					return nil, twirp.InternalError("failed type assertion resp.(*ListRepositoryBranchesResponse) when calling interceptor")
				}
				return typedResp, err
			}
			return nil, err
		}
	}

	// Call service method
	var respContent *ListRepositoryBranchesResponse
	func() {
		defer ensurePanicResponses(ctx, resp, s.hooks)
		respContent, err = handler(ctx, reqContent)
	}()

	if err != nil {
		s.writeError(ctx, resp, err)
		return
	}
	if respContent == nil {
		s.writeError(ctx, resp, twirp.InternalError("received a nil *ListRepositoryBranchesResponse and nil error while calling ListRepositoryBranches. nil responses are not supported"))
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

func (s *repositoryBranchServiceServer) serveListRepositoryBranchesProtobuf(ctx context.Context, resp http.ResponseWriter, req *http.Request) {
	var err error
	ctx = ctxsetters.WithMethodName(ctx, "ListRepositoryBranches")
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
	reqContent := new(ListRepositoryBranchesRequest)
	if err = proto.Unmarshal(buf, reqContent); err != nil {
		s.writeError(ctx, resp, malformedRequestError("the protobuf request could not be decoded"))
		return
	}

	handler := s.RepositoryBranchService.ListRepositoryBranches
	if s.interceptor != nil {
		handler = func(ctx context.Context, req *ListRepositoryBranchesRequest) (*ListRepositoryBranchesResponse, error) {
			resp, err := s.interceptor(
				func(ctx context.Context, req interface{}) (interface{}, error) {
					typedReq, ok := req.(*ListRepositoryBranchesRequest)
					if !ok {
						return nil, twirp.InternalError("failed type assertion req.(*ListRepositoryBranchesRequest) when calling interceptor")
					}
					return s.RepositoryBranchService.ListRepositoryBranches(ctx, typedReq)
				},
			)(ctx, req)
			if resp != nil {
				typedResp, ok := resp.(*ListRepositoryBranchesResponse)
				if !ok {
					return nil, twirp.InternalError("failed type assertion resp.(*ListRepositoryBranchesResponse) when calling interceptor")
				}
				return typedResp, err
			}
			return nil, err
		}
	}

	// Call service method
	var respContent *ListRepositoryBranchesResponse
	func() {
		defer ensurePanicResponses(ctx, resp, s.hooks)
		respContent, err = handler(ctx, reqContent)
	}()

	if err != nil {
		s.writeError(ctx, resp, err)
		return
	}
	if respContent == nil {
		s.writeError(ctx, resp, twirp.InternalError("received a nil *ListRepositoryBranchesResponse and nil error while calling ListRepositoryBranches. nil responses are not supported"))
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

func (s *repositoryBranchServiceServer) ServiceDescriptor() ([]byte, int) {
	return twirpFileDescriptor6, 0
}

func (s *repositoryBranchServiceServer) ProtocGenTwirpVersion() string {
	return "v8.1.0"
}

// PathPrefix returns the base service path, in the form: "/<prefix>/<package>.<Service>/"
// that is everything in a Twirp route except for the <Method>. This can be used for routing,
// for example to identify the requests that are targeted to this service in a mux.
func (s *repositoryBranchServiceServer) PathPrefix() string {
	return baseServicePath(s.pathPrefix, "buf.alpha.registry.v1alpha1", "RepositoryBranchService")
}

var twirpFileDescriptor6 = []byte{
	// 536 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x9c, 0x54, 0xc1, 0x6e, 0xd3, 0x40,
	0x10, 0x95, 0x9d, 0x02, 0xed, 0x84, 0x40, 0x59, 0x04, 0x58, 0x89, 0xd2, 0x46, 0x41, 0x42, 0xbd,
	0xb0, 0x56, 0xd3, 0x1b, 0x39, 0x35, 0x9c, 0x90, 0x38, 0x54, 0x6e, 0xc5, 0x21, 0x8a, 0x88, 0xec,
	0x64, 0xe2, 0xac, 0x48, 0xbc, 0x66, 0x77, 0x1d, 0xd1, 0x0a, 0x3e, 0x82, 0x1b, 0xe2, 0xc8, 0x09,
	0xf1, 0x17, 0x5c, 0xf9, 0x00, 0xbe, 0x07, 0x79, 0x37, 0x5b, 0x8a, 0xdd, 0xba, 0x2a, 0x37, 0xfb,
	0xcd, 0xbc, 0x37, 0x6f, 0xdf, 0xac, 0x16, 0x0e, 0xa2, 0x6c, 0xe6, 0x87, 0x8b, 0x74, 0x1e, 0xfa,
	0x02, 0x63, 0x26, 0x95, 0x38, 0xf5, 0x57, 0xfb, 0x1a, 0xd8, 0xf7, 0x05, 0xa6, 0x5c, 0x32, 0xc5,
	0xc5, 0xe9, 0x38, 0x12, 0x61, 0x32, 0x99, 0xd3, 0x54, 0x70, 0xc5, 0x49, 0x2b, 0xca, 0x66, 0x54,
	0xf7, 0x50, 0x4b, 0xa2, 0x96, 0xd4, 0xdc, 0x8d, 0x39, 0x8f, 0x17, 0xe8, 0xeb, 0xd6, 0x5c, 0x5d,
	0xb1, 0x25, 0x4a, 0x15, 0x2e, 0x53, 0xc3, 0xee, 0x7e, 0x71, 0x60, 0x3b, 0x38, 0x57, 0x1e, 0x68,
	0x61, 0x72, 0x0f, 0x5c, 0x36, 0xf5, 0x9c, 0x8e, 0xb3, 0xb7, 0x15, 0xb8, 0x6c, 0x4a, 0xfa, 0x50,
	0x9f, 0x08, 0x0c, 0x15, 0x8e, 0x73, 0xba, 0xe7, 0x76, 0x9c, 0xbd, 0x7a, 0xaf, 0x49, 0x8d, 0x36,
	0xb5, 0xda, 0xf4, 0xc4, 0x6a, 0x07, 0x60, 0xda, 0x73, 0x80, 0x10, 0xd8, 0x48, 0xc2, 0x25, 0x7a,
	0x1b, 0x5a, 0x4e, 0x7f, 0x93, 0xa7, 0xd0, 0xb8, 0x70, 0x1c, 0x36, 0xf5, 0x6e, 0xe9, 0xe2, 0xdd,
	0xbf, 0xe0, 0xab, 0x69, 0xf7, 0x13, 0xb4, 0x5f, 0x6a, 0x99, 0xa2, 0xbf, 0x00, 0xdf, 0x67, 0x28,
	0x55, 0x59, 0xc5, 0x29, 0xab, 0x9c, 0x8f, 0x77, 0xff, 0x1d, 0x9f, 0x86, 0x02, 0x13, 0xb5, 0x4e,
	0xd2, 0xab, 0x19, 0xa2, 0x01, 0xcd, 0x90, 0xee, 0x47, 0xd8, 0xb9, 0x6a, 0xbc, 0x4c, 0x79, 0x22,
	0x91, 0x0c, 0xe1, 0x41, 0x69, 0x29, 0xda, 0x43, 0xbd, 0xf7, 0x9c, 0x56, 0x6c, 0x85, 0x96, 0x14,
	0xb7, 0x45, 0x01, 0xe9, 0x7e, 0x75, 0xa0, 0xfd, 0x9a, 0x49, 0x55, 0x6c, 0x45, 0x79, 0xa3, 0xd3,
	0xb7, 0x60, 0x2b, 0x0d, 0x63, 0x1c, 0x4b, 0x76, 0x66, 0x22, 0x68, 0x04, 0x9b, 0x39, 0x70, 0xcc,
	0xce, 0x90, 0xb4, 0x01, 0x74, 0x51, 0xf1, 0x77, 0x98, 0xac, 0x33, 0xd0, 0xed, 0x27, 0x39, 0x40,
	0x3c, 0xb8, 0x23, 0x70, 0x85, 0x42, 0x9a, 0xdd, 0x6d, 0x06, 0xf6, 0xb7, 0xfb, 0xdd, 0x81, 0x9d,
	0xab, 0xcc, 0xad, 0xb3, 0x79, 0x0b, 0x0f, 0x4b, 0xd9, 0xa0, 0xf4, 0x9c, 0x4e, 0xed, 0xe6, 0xe9,
	0x10, 0x51, 0x9a, 0x43, 0x9e, 0xc1, 0xfd, 0x04, 0x3f, 0xa8, 0xf1, 0x85, 0x03, 0x98, 0x0d, 0x37,
	0x72, 0xf8, 0xc8, 0x1e, 0xa2, 0xf7, 0xd3, 0x85, 0x27, 0x45, 0xc1, 0x63, 0x14, 0x2b, 0x36, 0x41,
	0xf2, 0xd9, 0x81, 0xc7, 0x97, 0xaf, 0x98, 0xbc, 0xa8, 0x74, 0x58, 0x79, 0x2d, 0x9b, 0xfd, 0xff,
	0xe2, 0xae, 0x73, 0xcb, 0x3d, 0x5d, 0x1e, 0xed, 0x35, 0x9e, 0x2a, 0x2f, 0xcb, 0x35, 0x9e, 0xaa,
	0x77, 0x39, 0xf8, 0xed, 0xc0, 0xee, 0x84, 0x2f, 0xab, 0x24, 0x06, 0x8f, 0x8a, 0xfc, 0xa3, 0xfc,
	0x55, 0x18, 0x0e, 0x63, 0xa6, 0xe6, 0x59, 0x44, 0x27, 0x7c, 0xe9, 0x47, 0xd9, 0x2c, 0xca, 0xd8,
	0x62, 0x9a, 0x7f, 0xf8, 0x2c, 0x51, 0x28, 0x92, 0x70, 0xe1, 0xc7, 0x98, 0x98, 0xd7, 0xc9, 0x8f,
	0xb9, 0x5f, 0xf1, 0xfe, 0xf5, 0x2d, 0x62, 0x81, 0x6f, 0x6e, 0x6d, 0x70, 0x18, 0xfc, 0x70, 0x5b,
	0x83, 0x6c, 0x46, 0x0f, 0xb5, 0xad, 0xc0, 0xda, 0x7a, 0xb3, 0xee, 0xf9, 0xa5, 0xab, 0x23, 0x5d,
	0x1d, 0xd9, 0xea, 0xc8, 0x56, 0xa3, 0xdb, 0x7a, 0xf0, 0xc1, 0x9f, 0x00, 0x00, 0x00, 0xff, 0xff,
	0xd2, 0xa5, 0xbd, 0x41, 0x78, 0x05, 0x00, 0x00,
}
