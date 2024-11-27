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

package bufconnect

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"google.golang.org/protobuf/proto"
)

const (
	// TokenEnvKey is the environment variable key for the auth token
	TokenEnvKey = "BUF_TOKEN"
)

// NewAugmentedConnectErrorInterceptor returns a new Connect Interceptor that wraps
// [connect.Error]s in an [AugmentedConnectError].
func NewAugmentedConnectErrorInterceptor() connect.UnaryInterceptorFunc {
	interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			resp, err := next(ctx, req)
			if err != nil {
				if connectErr := new(connect.Error); errors.As(err, &connectErr) {
					err = &AugmentedConnectError{
						// Using the original err to avoid throwing information away.
						cause:     err,
						procedure: req.Spec().Procedure,
						addr:      req.Peer().Addr,
					}
				}
			}
			return resp, err
		}
	}
	return interceptor
}

// NewSetCLIVersionInterceptor returns a new Connect Interceptor that sets the Buf CLI version into all request headers
func NewSetCLIVersionInterceptor(version string) connect.UnaryInterceptorFunc {
	interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set(CliVersionHeaderName, version)
			return next(ctx, req)
		}
	}
	return interceptor
}

// NewCLIWarningInterceptor returns a new Connect Interceptor that logs CLI warnings returned by server responses.
func NewCLIWarningInterceptor(container appext.LoggerContainer) connect.UnaryInterceptorFunc {
	interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			resp, err := next(ctx, req)
			if resp != nil {
				logWarningFromHeader(container, resp.Header())
			} else if err != nil {
				if connectErr := new(connect.Error); errors.As(err, &connectErr) {
					logWarningFromHeader(container, connectErr.Meta())
				}
			}
			return resp, err
		}
	}
	return interceptor
}

func logWarningFromHeader(container appext.LoggerContainer, header http.Header) {
	encoded := header.Get(CLIWarningHeaderName)
	if encoded != "" {
		warning, err := connect.DecodeBinaryHeader(encoded)
		if err != nil {
			container.Logger().Debug(fmt.Errorf("failed to decode warning header: %w", err).Error())
			return
		}
		if len(warning) > 0 {
			container.Logger().Warn(string(warning))
		}
	}
}

// TokenProvider finds the token for NewAuthorizationInterceptorProvider.
type TokenProvider interface {
	// RemoteToken returns the remote token from the remote address.
	RemoteToken(address string) string
	// IsFromEnvVar returns true if the TokenProvider is generated from an environment variable.
	IsFromEnvVar() bool
}

// NewAuthorizationInterceptorProvider returns a new provider function which, when invoked, returns an interceptor
// which will set the auth token into the request header by the provided option.
//
// Note that the interceptor returned from this provider is always applied LAST in the series of interceptors added to
// a client.
func NewAuthorizationInterceptorProvider(tokenProviders ...TokenProvider) func(string) connect.UnaryInterceptorFunc {
	return func(address string) connect.UnaryInterceptorFunc {
		interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
			return connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
				usingTokenEnvKey := false
				hasToken := false
				for _, tf := range tokenProviders {
					if token := tf.RemoteToken(address); token != "" {
						req.Header().Set(AuthenticationHeader, AuthenticationTokenPrefix+token)
						usingTokenEnvKey = tf.IsFromEnvVar()
						hasToken = true
						break
					}
				}
				response, err := next(ctx, req)
				if err != nil {
					var envKey string
					if usingTokenEnvKey {
						envKey = TokenEnvKey
					}
					err = &AuthError{
						cause:       err,
						remote:      address,
						hasToken:    hasToken,
						tokenEnvKey: envKey,
					}
				}
				return response, err
			})
		}
		return interceptor
	}
}

// NewDebugLoggingInterceptor returns a new Connect Interceptor that adds debug log
// statements for each rpc call.
//
// The following information is collected for logging: duration, status code, peer name,
// rpc system, request size, and response size.
func NewDebugLoggingInterceptor(container appext.LoggerContainer) connect.UnaryInterceptorFunc {
	interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			var requestSize int
			if req.Any() != nil {
				msg, ok := req.Any().(proto.Message)
				if ok {
					requestSize = proto.Size(msg)
				}
			}
			startTime := time.Now()
			resp, err := next(ctx, req)
			duration := time.Since(startTime)
			var status connect.Code
			if err != nil {
				status = connect.CodeOf(err)
			}
			var responseSize int
			if resp != nil && resp.Any() != nil {
				msg, ok := resp.Any().(proto.Message)
				if ok {
					responseSize = proto.Size(msg)
				}
			}
			attrs := []slog.Attr{
				slog.Duration("duration", duration),
				slog.String("status", status.String()),
				slog.String("net.peer.name", req.Peer().Addr),
				slog.String("rpc.system", req.Peer().Protocol),
				slog.Int("message.sent.uncompressed_size", requestSize),
				slog.Int("message.received.uncompressed_size", responseSize),
			}
			container.Logger().LogAttrs(
				ctx,
				slog.LevelDebug,
				// Remove the leading "/" from Procedure name
				strings.TrimPrefix(req.Spec().Procedure, "/"),
				attrs...,
			)
			return resp, err
		}
	}
	return interceptor
}
