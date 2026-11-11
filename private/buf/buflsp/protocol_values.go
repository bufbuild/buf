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

package buflsp

import (
	"encoding/json"

	"go.lsp.dev/protocol"
)

// newOptionalNonZero returns a [protocol.Optional] that is set only if value is
// not the zero value, so that zero values are omitted from the wire.
func newOptionalNonZero[T comparable](value T) protocol.Optional[T] {
	var zero T
	if value == zero {
		return protocol.Optional[T]{}
	}
	return protocol.NewOptional(value)
}

// newStringLSPAny returns value encoded as a JSON string [protocol.LSPAny].
func newStringLSPAny(value string) protocol.LSPAny {
	// Marshaling a string never fails.
	data, _ := json.Marshal(value)
	return data
}
