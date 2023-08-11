// Copyright 2020-2023 Buf Technologies, Inc.
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

//go:build go1.20

package bufcli

import (
	"crypto/tls"
	"errors"
)

// wrappedTLSError returns an unwrapped TLS error or nil if the error is another type of error.
func wrappedTLSError(err error) error {
	if tlsErr := (&tls.CertificateVerificationError{}); errors.As(err, &tlsErr) {
		return tlsErr
	}
	return nil
}
