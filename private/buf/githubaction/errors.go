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

package githubaction

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/rpc"
)

// NewErrorInterceptor returns a CLI interceptor that wraps Buf CLI errors.
func NewErrorInterceptor() appflag.Interceptor {
	return func(next func(context.Context, appflag.Container) error) func(context.Context, appflag.Container) error {
		return func(ctx context.Context, container appflag.Container) error {
			return wrapError(next(ctx, container))
		}
	}
}

// wrapError is used when a CLI command fails, regardless of its error code.
// This is similar to wrapError in bufcli except it prefixes the error message with "::error::" so that it will be
// seen as an error message by github actions. It also has a different message for rpc.ErrorCodeUnauthenticated
func wrapError(err error) error {
	const prefix = "::error::"
	const unauthenticatedMessage = `you are not authenticated. Add a token to inputs.buf_token.`

	if err == nil || (err.Error() == "" && !rpc.IsError(err)) {
		// If the error is nil or empty and not an rpc error, we return it as-is.
		// This is especially relevant for commands like lint and breaking.
		return err
	}
	if err.Error() == "" && rpc.GetErrorCode(err) == rpc.ErrorCodeUnknown {
		return fmt.Errorf("%s%s: %w", prefix, unauthenticatedMessage, err)
	}
	switch rpc.GetErrorCode(err) {
	case rpc.ErrorCodeUnauthenticated:
		return fmt.Errorf("%s%s: %w", prefix, unauthenticatedMessage, err)
	case rpc.ErrorCodeUnavailable:
		return fmt.Errorf("%s%s: %w", prefix, "the server hosted at that remote is unavailable", err)
	}
	return fmt.Errorf("::error:: %w", err)
}
