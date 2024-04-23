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

package bufstudioagent

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/textproto"
	"net/url"

	"connectrpc.com/connect"
	studiov1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/studio/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"google.golang.org/protobuf/proto"
)

// MaxMessageSizeBytesDefault determines the maximum number of bytes to read
// from the request body.
const MaxMessageSizeBytesDefault = 1024 * 1024 * 5

// plainPostHandler implements a POST handler for forwarding requests that can
// be called with simple CORS requests.
//
// Simple CORS requests are limited [1] to certain headers and content types, so
// this handler expects base64 encoded protobuf messages in the body and writes
// out base64 encoded protobuf messages to be able to use Content-Type: text/plain.
//
// Because of the content-type restriction we do not define a protobuf service
// that gets served by connect but instead use a plain post handler.
//
// [1] https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS#simple_requests).
type plainPostHandler struct {
	Logger              *zap.Logger
	MaxMessageSizeBytes int64
	B64Encoding         *base64.Encoding
	TLSClient           *http.Client
	H2CClient           *http.Client
	DisallowedHeaders   map[string]struct{}
	ForwardHeaders      map[string]string
}

func newPlainPostHandler(
	logger *zap.Logger,
	disallowedHeaders map[string]struct{},
	forwardHeaders map[string]string,
	tlsClientConfig *tls.Config,
) *plainPostHandler {
	canonicalDisallowedHeaders := make(map[string]struct{}, len(disallowedHeaders))
	for k := range disallowedHeaders {
		canonicalDisallowedHeaders[textproto.CanonicalMIMEHeaderKey(k)] = struct{}{}
	}
	canonicalForwardHeaders := make(map[string]string, len(forwardHeaders))
	for k, v := range forwardHeaders {
		canonicalForwardHeaders[textproto.CanonicalMIMEHeaderKey(k)] = v
	}
	return &plainPostHandler{
		B64Encoding:       base64.StdEncoding,
		DisallowedHeaders: canonicalDisallowedHeaders,
		ForwardHeaders:    canonicalForwardHeaders,
		H2CClient: &http.Client{
			Transport: &http2.Transport{
				AllowHTTP: true,
				DialTLS: func(netw, addr string, config *tls.Config) (net.Conn, error) {
					return net.Dial(netw, addr)
				},
			},
		},
		Logger:              logger,
		MaxMessageSizeBytes: MaxMessageSizeBytesDefault,
		TLSClient: &http.Client{
			Transport: &http2.Transport{
				TLSClientConfig: tlsClientConfig,
			},
		},
	}
}

func (i *plainPostHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "", http.StatusMethodNotAllowed)
		return
	}
	if r.Header.Get("content-type") != "text/plain" {
		http.Error(w, "", http.StatusUnsupportedMediaType)
		return
	}
	bodyBytes, err := io.ReadAll(
		base64.NewDecoder(
			i.B64Encoding,
			http.MaxBytesReader(w, r.Body, i.MaxMessageSizeBytes),
		),
	)
	if err != nil {
		if b64Err := new(base64.CorruptInputError); errors.As(err, &b64Err) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
		return
	}
	envelopeRequest := &studiov1alpha1.InvokeRequest{}
	if err := protoencoding.NewWireUnmarshaler(nil).Unmarshal(bodyBytes, envelopeRequest); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	request := connect.NewRequest(bytes.NewBuffer(envelopeRequest.GetBody()))
	for _, header := range envelopeRequest.Headers {
		if _, ok := i.DisallowedHeaders[textproto.CanonicalMIMEHeaderKey(header.Key)]; ok {
			http.Error(w, fmt.Sprintf("header %q disallowed by agent", header.Key), http.StatusBadRequest)
			return
		}
		for _, value := range header.Value {
			request.Header().Add(header.Key, value)
		}
	}
	for fromHeader, toHeader := range i.ForwardHeaders {
		headerValues := r.Header.Values(fromHeader)
		if len(headerValues) > 0 {
			request.Header().Del(toHeader)
			for _, headerValue := range headerValues {
				request.Header().Add(toHeader, headerValue)
			}
		}
	}
	targetURL, err := url.Parse(envelopeRequest.GetTarget())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var httpClient *http.Client
	switch targetURL.Scheme {
	case "http":
		httpClient = i.H2CClient
	case "https":
		httpClient = i.TLSClient
	default:
		http.Error(w, fmt.Sprintf("must specify http or https url scheme, got %q", targetURL.Scheme), http.StatusBadRequest)
		return
	}
	clientOptions, err := connectClientOptionsFromContentType(request.Header().Get("Content-Type"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	client := connect.NewClient[bytes.Buffer, bytes.Buffer](
		httpClient,
		targetURL.String(),
		clientOptions...,
	)
	// TODO(rvanginkel) should this context be cloned to remove attached values (but keep timeout)?
	response, err := client.CallUnary(r.Context(), request)
	if err != nil {
		// We need to differentiate client errors from server errors. In the former,
		// trigger a `StatusBadGateway` result, and in the latter surface whatever
		// error information came back from the server.
		//
		// Any error here is expected to be wrapped in a `connect.Error` struct. We
		// need to check *first* if it's not a wire error, so we can assume the
		// request never left the client, or a response never arrived from the
		// server. In those scenarios we trigger a `StatusBadGateway` to signal
		// that the upstream server is unreachable or in a bad status...
		if !connect.IsWireError(err) {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		// ... but if a response was received from the server, we assume there's
		// error information from the server we can surface to the user by including
		// it in the headers response, unless it is a `CodeUnknown` error. Connect
		// marks any issues connecting with the `CodeUnknown` error.
		if connectErr := new(connect.Error); errors.As(err, &connectErr) {
			if connectErr.Code() == connect.CodeUnknown {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
			i.writeProtoMessage(w, &studiov1alpha1.InvokeResponse{
				// connectErr.Meta contains the trailers for the
				// caller to find out the error details.
				Headers: goHeadersToProtoHeaders(connectErr.Meta()),
			})
			return
		}
		i.Logger.Warn(
			"non_connect_unary_error",
			zap.Error(err),
		)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	i.writeProtoMessage(w, &studiov1alpha1.InvokeResponse{
		Headers:  goHeadersToProtoHeaders(response.Header()),
		Body:     response.Msg.Bytes(),
		Trailers: goHeadersToProtoHeaders(response.Trailer()),
	})
}

func connectClientOptionsFromContentType(contentType string) ([]connect.ClientOption, error) {
	switch contentType {
	case "application/grpc", "application/grpc+proto":
		return []connect.ClientOption{
			connect.WithGRPC(),
			connect.WithCodec(&bufferCodec{name: "proto"}),
		}, nil
	case "application/grpc+json":
		return []connect.ClientOption{
			connect.WithGRPC(),
			connect.WithCodec(&bufferCodec{name: "json"}),
		}, nil
	case "application/json":
		return []connect.ClientOption{
			connect.WithCodec(&bufferCodec{name: "json"}),
		}, nil
	case "application/proto":
		return []connect.ClientOption{
			connect.WithCodec(&bufferCodec{name: "proto"}),
		}, nil
	default:
		return nil, fmt.Errorf("unknown Content-Type: %q", contentType)
	}
}

func (i *plainPostHandler) writeProtoMessage(w http.ResponseWriter, message proto.Message) {
	responseProtoBytes, err := protoencoding.NewWireMarshaler().Marshal(message)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	responseB64Bytes := make([]byte, i.B64Encoding.EncodedLen(len(responseProtoBytes)))
	i.B64Encoding.Encode(responseB64Bytes, responseProtoBytes)
	w.Header().Set("Content-Type", "text/plain")
	if n, err := w.Write(responseB64Bytes); n != len(responseB64Bytes) && err != nil {
		i.Logger.Error(
			"write_error",
			zap.Int("expected_bytes", len(responseB64Bytes)),
			zap.Int("actual_bytes", n),
			zap.Error(err),
		)
	}
}

func goHeadersToProtoHeaders(in http.Header) []*studiov1alpha1.Headers {
	var out []*studiov1alpha1.Headers
	for k, v := range in {
		out = append(out, &studiov1alpha1.Headers{
			Key:   k,
			Value: v,
		})
	}
	return out
}
