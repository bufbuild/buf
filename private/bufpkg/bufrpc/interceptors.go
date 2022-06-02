package bufrpc

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/netrc"
	"github.com/bufbuild/connect-go"
)

const (
	// authenticationHeader is the standard OAuth header used for authenticating
	// a user. Ignore the misnomer.
	authenticationHeader = "Authorization"
	// authenticationTokenPrefix is the standard OAuth token prefix.
	// We use it for familiarity.
	authenticationTokenPrefix = "Bearer "
)

const (
	// Version is the CLI version of buf.
	Version = "1.6.0-dev"

	tokenEnvKey = "BUF_TOKEN"
)

func NewOutgoingHeaderInterceptorProvider(container appflag.Container) func(string) (connect.UnaryInterceptorFunc, error) {
	return func(address string) (connect.UnaryInterceptorFunc, error) {
		token := container.Env(tokenEnvKey)
		fmt.Println("reading token")
		if token == "" {
			machine, err := netrc.GetMachineForName(container, address)
			if err != nil {
				return nil, fmt.Errorf("failed to read server password from netrc: %w", err)
			}
			if machine != nil {
				token = machine.Password()
			}
		}
		interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
			return connect.UnaryFunc(func(
				ctx context.Context,
				req connect.AnyRequest,
			) (connect.AnyResponse, error) {
				WithOutgoingCLIVersionHeader(req, Version)
				if req.Header().Get(authenticationHeader) == "" {
					req.Header().Set(authenticationHeader, authenticationTokenPrefix+token)
				}
				return next(ctx, req)
			})
		}
		return interceptor, nil
	}
}

func NewWithTokenInterceptorProvider(token string) func(string) (connect.UnaryInterceptorFunc, error) {
	return func(token string) (connect.UnaryInterceptorFunc, error) {
		interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
			return connect.UnaryFunc(func(
				ctx context.Context,
				req connect.AnyRequest,
			) (connect.AnyResponse, error) {
				if token != "" {
					req.Header().Set(authenticationHeader, authenticationTokenPrefix+token)
				}
				return next(ctx, req)
			})
		}
		return interceptor, nil
	}
}
