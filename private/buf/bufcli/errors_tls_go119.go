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

//go:build !go1.20

package bufcli

import (
	"errors"
	"strings"
)

// wrappedTLSError returns an unwrapped TLS error or nil if the error is another type of error.
// This method is a workaround until we can switch to use errors.As(err, *tls.CertificateVerificationError),
// which is a new error type introduced in Go 1.20. This can be removed when we upgrade to support Go 1.20/1.21+.
func wrappedTLSError(err error) error {
	wrapped := errors.Unwrap(err)
	if wrapped == nil {
		return nil
	}
	if strings.HasPrefix(wrapped.Error(), "x509:") && strings.HasSuffix(wrapped.Error(), "certificate is not trusted") {
		return wrapped
	}
	return nil
}
