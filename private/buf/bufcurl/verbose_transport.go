// Copyright 2020-2024 Buf Technologies, Inc.
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

package bufcurl

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"sort"
	"strings"

	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/pkg/verbose"
	"go.uber.org/atomic"
)

type skipUploadFinishedMessageKey struct{}

func skippingUploadFinishedMessage(ctx context.Context) context.Context {
	return context.WithValue(ctx, skipUploadFinishedMessageKey{}, true)
}

type userAgentKey struct{}

func withUserAgent(ctx context.Context, headers http.Header) context.Context {
	if userAgentHeaders := headers.Values("user-agent"); len(userAgentHeaders) > 0 {
		return context.WithValue(ctx, userAgentKey{}, userAgentHeaders)
	}
	return ctx // no change
}

// DefaultUserAgent returns the default user agent for the given protocol.
func DefaultUserAgent(protocol string, bufVersion string) string {
	// mirror the default user agent for the Connect client library, but
	// add "buf/<version>" in front of it.
	libUserAgent := "connect-go"
	if strings.Contains(protocol, "grpc") {
		libUserAgent = "grpc-go-connect"
	}
	return fmt.Sprintf("%s/%s (%s) buf/%s", libUserAgent, connect.Version, runtime.Version(), bufVersion)
}

// NewVerboseHTTPClient creates a new HTTP client with the given transport and
// printing verbose trace information to the given printer.
func NewVerboseHTTPClient(transport http.RoundTripper, printer verbose.Printer) connect.HTTPClient {
	return &verboseClient{transport: transport, printer: printer}
}

// TraceTrailersInterceptor returns an interceptor that will print information
// about trailers for streaming calls to the given printer. This is used with
// the Connect and gRPC-web protocols since these protocols include trailers in
// the request body, instead of using actual HTTP trailers. (For the gRPC
// protocol, which uses actual HTTP trailers, the verbose HTTP client suffices
// since it already prints information about the trailers.)
func TraceTrailersInterceptor(printer verbose.Printer) connect.Interceptor {
	return traceTrailersInterceptor{printer: printer}
}

type verboseClient struct {
	transport http.RoundTripper
	printer   verbose.Printer
	reqNum    atomic.Int32
}

type reqNumAddrKey struct{}

func (v *verboseClient) Do(req *http.Request) (*http.Response, error) {
	if host := req.Header.Get("Host"); host != "" {
		// Set based on host header. This way it is also correctly used as
		// the ":authority" meta-header in HTTP/2.
		req.Host = host
	}
	if userAgentHeaders, _ := req.Context().Value(userAgentKey{}).([]string); len(userAgentHeaders) > 0 {
		req.Header.Del("user-agent")
		for _, val := range userAgentHeaders {
			req.Header.Add("user-agent", val)
		}
	}

	reqNum := v.reqNum.Add(1)
	if reqNumAddr, _ := req.Context().Value(reqNumAddrKey{}).(*int32); reqNumAddr != nil {
		*reqNumAddr = reqNum
	}
	rawBody := req.Body
	if rawBody == nil {
		rawBody = io.NopCloser(bytes.NewBuffer(nil))
	}
	var atEnd func(error)
	if skip, _ := req.Context().Value(skipUploadFinishedMessageKey{}).(bool); !skip {
		atEnd = func(err error) {
			if errors.Is(err, io.EOF) {
				v.printer.Printf("* (#%d) Finished upload", reqNum)
			}
		}
	}
	req.Body = &verboseReader{
		ReadCloser: rawBody,
		callback: func(count int) {
			v.traceWriteRequestBytes(reqNum, count)
		},
		whenDone: atEnd,
		whenStart: func() {
			// we defer this until body is read so that our HTTP client's dialer and TLS
			// config can potentially log useful things about connection setup *before*
			// we print the request info.
			v.traceRequest(req, reqNum)
		},
	}
	resp, err := v.transport.RoundTrip(req)
	if resp != nil {
		v.traceResponse(resp, reqNum)
		if resp.Body != nil {
			resp.Body = &verboseReader{
				ReadCloser: resp.Body,
				callback: func(count int) {
					v.traceReadResponseBytes(reqNum, count)
				},
				whenDone: func(err error) {
					traceTrailers(v.printer, resp.Trailer, false, reqNum)
					v.printer.Printf("* (#%d) Call complete", reqNum)
				},
			}
		}
	}

	return resp, err
}

func (v *verboseClient) traceRequest(r *http.Request, reqNum int32) {
	// we look at the *raw* http headers, in case any get added by the
	// Connect client impl or an interceptor after we could otherwise
	// inspect them from an interceptor
	var queryString string
	if r.URL.RawQuery != "" {
		queryString = "?" + r.URL.RawQuery
	} else if r.URL.ForceQuery {
		queryString = "?"
	}
	v.printer.Printf("> (#%d) %s %s%s\n", reqNum, r.Method, r.URL.Path, queryString)
	traceMetadata(v.printer, r.Header, fmt.Sprintf("> (#%d) ", reqNum))
	v.printer.Printf("> (#%d)\n", reqNum)
}

func (v *verboseClient) traceWriteRequestBytes(reqNum int32, count int) {
	v.printer.Printf("} (#%d) [%d bytes data]", reqNum, count)
}

func (v *verboseClient) traceResponse(r *http.Response, reqNum int32) {
	v.printer.Printf("< (#%d) %s %s\n", reqNum, r.Proto, r.Status)
	traceMetadata(v.printer, r.Header, fmt.Sprintf("< (#%d) ", reqNum))
	v.printer.Printf("< (#%d)\n", reqNum)
}

func (v *verboseClient) traceReadResponseBytes(reqNum int32, count int) {
	v.printer.Printf("{ (#%d) [%d bytes data]", reqNum, count)
}

func traceMetadata(printer verbose.Printer, meta http.Header, prefix string) {
	keys := make([]string, 0, len(meta))
	for key := range meta {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		vals := meta[key]
		for _, val := range vals {
			printer.Printf("%s%s: %s\n", prefix, key, val)
		}
	}
}

func traceTrailers(printer verbose.Printer, trailers http.Header, synthetic bool, reqNum int32) {
	if len(trailers) == 0 {
		return
	}
	printer.Printf("< (#%d)\n", reqNum)
	prefix := fmt.Sprintf("< (#%d) ", reqNum)
	if synthetic {
		// mark synthetic trailers with an asterisk
		prefix += "[*] "
	}
	traceMetadata(printer, trailers, prefix)
}

type verboseReader struct {
	io.ReadCloser
	callback  func(int)
	whenStart func()
	whenDone  func(error)
	started   atomic.Bool
	done      atomic.Bool
}

func (v *verboseReader) Read(dest []byte) (n int, err error) {
	if v.started.CompareAndSwap(false, true) && v.whenStart != nil {
		v.whenStart()
	}
	n, err = v.ReadCloser.Read(dest)
	if n > 0 && v.callback != nil {
		v.callback(n)
	}
	if err != nil {
		if v.done.CompareAndSwap(false, true) && v.whenDone != nil {
			v.whenDone(err)
		}
	}
	return n, err
}

func (v *verboseReader) Close() error {
	err := v.ReadCloser.Close()
	if v.done.CompareAndSwap(false, true) && v.whenDone != nil {
		reportError := err
		if reportError == nil {
			reportError = io.EOF
		}
		v.whenDone(reportError)
	}
	return err
}

type traceTrailersInterceptor struct {
	printer verbose.Printer
}

func (t traceTrailersInterceptor) WrapUnary(unaryFunc connect.UnaryFunc) connect.UnaryFunc {
	return unaryFunc
}

func (t traceTrailersInterceptor) WrapStreamingClient(clientFunc connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		var reqNum int32
		ctx = context.WithValue(ctx, reqNumAddrKey{}, &reqNum)
		return &traceTrailersStream{StreamingClientConn: clientFunc(ctx, spec), reqNum: &reqNum, printer: t.printer}
	}
}

func (t traceTrailersInterceptor) WrapStreamingHandler(handlerFunc connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return handlerFunc
}

type traceTrailersStream struct {
	connect.StreamingClientConn
	reqNum  *int32
	printer verbose.Printer
	done    atomic.Bool
}

func (s *traceTrailersStream) Receive(msg any) error {
	err := s.StreamingClientConn.Receive(msg)
	if err != nil && s.done.CompareAndSwap(false, true) {
		traceTrailers(s.printer, s.ResponseTrailer(), true, *s.reqNum)
	}
	return err
}
