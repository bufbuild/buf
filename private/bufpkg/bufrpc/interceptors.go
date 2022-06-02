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
	tokenEnvKey = "BUF_TOKEN"
)

func NewTokenReaderInterceptorProvider(container appflag.Container) func(string) (connect.UnaryInterceptorFunc, error) {
	return func(address string) (connect.UnaryInterceptorFunc, error) {
		token := container.Env(tokenEnvKey)
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
				if req.Header().Get(authenticationHeader) == "" {
					req.Header().Set(authenticationHeader, authenticationTokenPrefix+token)
				}
				return next(ctx, req)
			})
		}
		return interceptor, nil
	}
}

func NewWithVersionInterceptor(version string) connect.UnaryInterceptorFunc {
	interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
		return connect.UnaryFunc(func(
			ctx context.Context,
			req connect.AnyRequest,
		) (connect.AnyResponse, error) {
			WithOutgoingCLIVersionHeader(req, version)
			return next(ctx, req)
		})
	}
	return interceptor, nil
}

// This is trying to mimic the current logic which actually has the context modifier setting the token as well as manually
// setting it in the context.  See private/buf/cmd/buf/command/registry/registrylogin/registrylogin.go for an example
//
// The context modifier only sets the token if it's not set by i.e. something like this.
//
// The other option is to have two NewRegistryProvider functions -- one that has this interceptor and one that has the
// reader applied
//
func NewWithTokenInterceptor(token string) connect.UnaryInterceptorFunc {
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
	return interceptor
}
