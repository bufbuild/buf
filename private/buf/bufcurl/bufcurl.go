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

package bufcurl

import (
	"context"
	"io"
	"net/http"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// Resolver is an interface for resolving symbols into descriptors and for
// looking up extensions.
//
// Note that resolver implementations must be thread-safe because they could be
// used by two goroutines concurrently during bidirectional streaming calls.
type Resolver interface {
	FindDescriptorByName(name protoreflect.FullName) (protoreflect.Descriptor, error)
	protoregistry.MessageTypeResolver
	protoregistry.ExtensionTypeResolver
}

// Invoker provides the ability to invoke RPCs dynamically.
type Invoker interface {
	// Invoke invokes an RPC method using the given input data and request headers.
	// The dataSource is a string that describes the input data (e.g. a filename).
	// The actual contents of the request data is read from the given reader.
	Invoke(ctx context.Context, dataSource string, data io.Reader, headers http.Header) error
}
