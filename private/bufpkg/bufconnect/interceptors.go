// Copyright 2020-2026 Buf Technologies, Inc.
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
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"

	"buf.build/go/app/appext"
	"connectrpc.com/connect/v2"
)

const (
	// TokenEnvKey is the environment variable key for the auth token
	TokenEnvKey = "BUF_TOKEN"
)

// NewAugmentedConnectErrorInterceptor returns a new Connect client interceptor
// that wraps [connect.Error]s in an [AugmentedConnectError].
func NewAugmentedConnectErrorInterceptor() connect.ClientInterceptor {
	return func(next connect.ClientFunc) connect.ClientFunc {
		return func(ctx context.Context, spec connect.Spec) (connect.ClientStream, error) {
			info, _ := connect.CallInfoForClientContext(ctx)
			mapError := func(err error) error {
				if connectErr := new(connect.Error); errors.As(err, &connectErr) {
					var addr string
					if info != nil {
						addr = info.PeerAddr
					}
					return &AugmentedConnectError{
						// Using the original err to avoid throwing information away.
						cause:     err,
						procedure: spec.Procedure,
						addr:      addr,
					}
				}
				return err
			}
			stream, err := next(ctx, spec)
			if err != nil {
				return nil, mapError(err)
			}
			return &wrappedClientStream{
				ClientStream: stream,
				mapError:     mapError,
			}, nil
		}
	}
}

// NewSetCLIVersionInterceptor returns a new Connect client interceptor that sets
// the Buf CLI version into all request headers
func NewSetCLIVersionInterceptor(version string) connect.ClientInterceptor {
	return func(next connect.ClientFunc) connect.ClientFunc {
		return func(ctx context.Context, spec connect.Spec) (connect.ClientStream, error) {
			if info, ok := connect.CallInfoForClientContext(ctx); ok {
				info.RequestHeader().Set(CliVersionHeaderName, version)
			}
			return next(ctx, spec)
		}
	}
}

// NewCLIWarningInterceptor returns a new Connect client interceptor that logs CLI
// warnings returned by server responses.
func NewCLIWarningInterceptor(container appext.LoggerContainer) connect.ClientInterceptor {
	return func(next connect.ClientFunc) connect.ClientFunc {
		return func(ctx context.Context, spec connect.Spec) (connect.ClientStream, error) {
			info, ok := connect.CallInfoForClientContext(ctx)
			if !ok {
				return next(ctx, spec)
			}
			stream, err := next(ctx, spec)
			if err != nil {
				logWarningFromHeader(container, info.ResponseHeader())
				return nil, err
			}
			return &wrappedClientStream{
				ClientStream: stream,
				onTerminal: func(error) {
					logWarningFromHeader(container, info.ResponseHeader())
					logWarningFromHeader(container, info.ResponseTrailer())
				},
			}, nil
		}
	}
}

func logWarningFromHeader(container appext.LoggerContainer, header *connect.Header) {
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
func NewAuthorizationInterceptorProvider(tokenProviders ...TokenProvider) func(string) connect.ClientInterceptor {
	return func(address string) connect.ClientInterceptor {
		return func(next connect.ClientFunc) connect.ClientFunc {
			return func(ctx context.Context, spec connect.Spec) (connect.ClientStream, error) {
				usingTokenEnvKey := false
				hasToken := false
				if info, ok := connect.CallInfoForClientContext(ctx); ok {
					for _, tokenProvider := range tokenProviders {
						if token := tokenProvider.RemoteToken(address); token != "" {
							info.RequestHeader().Set(AuthenticationHeader, AuthenticationTokenPrefix+token)
							usingTokenEnvKey = tokenProvider.IsFromEnvVar()
							hasToken = true
							break
						}
					}
				}
				mapError := func(err error) error {
					var envKey string
					if usingTokenEnvKey {
						envKey = TokenEnvKey
					}
					return &AuthError{
						cause:       err,
						remote:      address,
						hasToken:    hasToken,
						tokenEnvKey: envKey,
					}
				}
				stream, err := next(ctx, spec)
				if err != nil {
					return nil, mapError(err)
				}
				return &wrappedClientStream{
					ClientStream: stream,
					mapError:     mapError,
				}, nil
			}
		}
	}
}

// NewDebugLoggingInterceptor returns a new Connect client interceptor that adds
// debug log statements for each rpc call.
//
// The following information is collected for logging: duration, status code, peer name,
// rpc system, request size, and response size.
func NewDebugLoggingInterceptor(container appext.LoggerContainer) connect.ClientInterceptor {
	return func(next connect.ClientFunc) connect.ClientFunc {
		return func(ctx context.Context, spec connect.Spec) (connect.ClientStream, error) {
			info, ok := connect.CallInfoForClientContext(ctx)
			if !ok {
				return next(ctx, spec)
			}
			startTime := time.Now()
			logCall := func(err error) {
				var status connect.Code
				if err != nil && !errors.Is(err, io.EOF) {
					status = connect.CodeOf(err)
				}
				attrs := []slog.Attr{
					slog.Duration("duration", time.Since(startTime)),
					slog.String("status", status.String()),
					slog.String("net.peer.name", info.PeerAddr),
					slog.String("rpc.system", info.Protocol),
					slog.Int("message.sent.uncompressed_size", info.SendStats.Size),
					slog.Int("message.received.uncompressed_size", info.ReceiveStats.Size),
				}
				container.Logger().LogAttrs(
					ctx,
					slog.LevelDebug,
					// Remove the leading "/" from Procedure name
					strings.TrimPrefix(spec.Procedure, "/"),
					attrs...,
				)
			}
			stream, err := next(ctx, spec)
			if err != nil {
				logCall(err)
				return nil, err
			}
			return &wrappedClientStream{
				ClientStream: stream,
				onTerminal:   logCall,
			}, nil
		}
	}
}

// wrappedClientStream wraps a [connect.ClientStream] so interceptors can map
// errors returned by the stream and observe the end of the RPC.
//
// mapError, if non-nil, is applied to every non-nil error returned by the
// stream, except errors that report clean receive-side completion (matching
// [io.EOF]). onTerminal, if non-nil, is invoked at most once when the RPC
// reaches a terminal event: Receive returning a non-nil error (including
// [io.EOF]) or Close. It receives the raw (unmapped) error.
type wrappedClientStream struct {
	connect.ClientStream
	mapError   func(error) error
	onTerminal func(error)
	once       sync.Once
}

func (s *wrappedClientStream) SendHeaders() error {
	return s.wrapError(s.ClientStream.SendHeaders())
}

func (s *wrappedClientStream) Send(msg any) error {
	return s.wrapError(s.ClientStream.Send(msg))
}

func (s *wrappedClientStream) CloseSend() error {
	return s.wrapError(s.ClientStream.CloseSend())
}

func (s *wrappedClientStream) Receive(msg any) error {
	err := s.ClientStream.Receive(msg)
	if err != nil {
		s.terminal(err)
	}
	return s.wrapError(err)
}

func (s *wrappedClientStream) Close() error {
	err := s.ClientStream.Close()
	s.terminal(err)
	return s.wrapError(err)
}

func (s *wrappedClientStream) wrapError(err error) error {
	if err == nil || s.mapError == nil || errors.Is(err, io.EOF) {
		return err
	}
	return s.mapError(err)
}

func (s *wrappedClientStream) terminal(err error) {
	if s.onTerminal == nil {
		return
	}
	s.once.Do(func() {
		s.onTerminal(err)
	})
}
