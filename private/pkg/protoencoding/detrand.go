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

package protoencoding

import _ "unsafe"

//go:linkname detrandDisable google.golang.org/protobuf/internal/detrand.Disable
func detrandDisable()

func init() {
	// Disable detrand so that it does not mess with json or text serialization.
	//
	// https://github.com/golang/protobuf/issues/1121
	// https://go-review.googlesource.com/c/protobuf/+/151340
	// https://developers.google.com/protocol-buffers/docs/reference/go/faq#unstable-json
	//
	// Google is doing this in their own libraries at this point https://github.com/google/starlark-go/blob/ee8ed142361c69d52fe8e9fb5e311d2a0a7c02de/lib/proto/proto.go#L774
	//
	// See the above issues - detrand.Disable should fundamentally be allowed to be called by external
	// libraries, this has been an issue for years.
	detrandDisable()
}
