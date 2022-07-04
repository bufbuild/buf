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

	studiov1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/studio/v1alpha1"
	"github.com/bufbuild/connect-go"
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
	if err := proto.Unmarshal(bodyBytes, envelopeRequest); err != nil {
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
	clientOptions, err := connectClientOptionsFromContentType(request.Header().Get("Content-Type"), len(request.Msg.Bytes()))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	client := connect.NewClient[bytes.Buffer, bytes.Buffer](
		httpClient,
		targetURL.String(),
		clientOptions...,
	)
	// TODO: should this context be cloned to remove attached values (but keep timeout)?``
	response, err := client.CallUnary(r.Context(), request)
	if err != nil {
		// Connect marks any issues connecting with the Unavailable
		// status code. We need to differentiate between server sent
		// errors with the Unavailable code and client connection
		// errors.
		if netErr := new(net.OpError); errors.As(err, &netErr) {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		if urlErr := new(url.Error); errors.As(err, &urlErr) {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
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
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	i.writeProtoMessage(w, &studiov1alpha1.InvokeResponse{
		Headers:  goHeadersToProtoHeaders(response.Header()),
		Body:     response.Msg.Bytes(),
		Trailers: goHeadersToProtoHeaders(response.Trailer()),
	})
}

func connectClientOptionsFromContentType(contentType string, requestMsgSize int) ([]connect.ClientOption, error) {
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
	case "":
		if requestMsgSize == 0 {
			// For zero-length outgoing requests where the content
			// type has not been specified, we default to gRPC + proto
			// so any incoming response just goes into the buffer while
			// we also allow parsing the proto error details from trailers.
			return []connect.ClientOption{
				connect.WithGRPC(),
				connect.WithCodec(&bufferCodec{name: "proto"}),
			}, nil
		}
		return nil, fmt.Errorf("missing Content-Type while body size is %d", requestMsgSize)
	default:
		return nil, fmt.Errorf("unknown Content-Type: %q", contentType)
	}
}

func (i *plainPostHandler) writeProtoMessage(w http.ResponseWriter, message proto.Message) {
	responseProtoBytes, err := proto.Marshal(message)
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
