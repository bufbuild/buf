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

package rpcauth

import (
	"context"

	"github.com/bufbuild/buf/private/pkg/rpc"
)

// WithToken adds the token to the context via a header.
func WithToken(ctx context.Context, token string) context.Context {
	if token != "" {
		return rpc.WithOutgoingHeader(ctx, rpc.AuthenticationHeader, rpc.AuthenticationTokenPrefix+token)
	}
	return ctx
}

// WithTokenIfNoneSet adds the token to the context via a header if none is already set.
// If a token is already set on the header, this function just returns the context as is.
func WithTokenIfNoneSet(ctx context.Context, token string) context.Context {
	if rpc.GetOutgoingHeader(ctx, rpc.AuthenticationHeader) != "" {
		return ctx
	}
	return WithToken(ctx, token)
}
