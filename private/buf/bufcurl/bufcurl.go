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

package bufcurl

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// Invoker provides the ability to invoke RPCs dynamically.
type Invoker interface {
	// Invoke invokes an RPC method using the given input data and request headers.
	// The dataSource is a string that describes the input data (e.g. a filename).
	// The actual contents of the request data is read from the given reader.
	Invoke(ctx context.Context, dataSource string, data io.Reader, headers http.Header) error
}

// ResolveMethodDescriptor uses the given resolver to find a descriptor for
// the requested service and method. The service name must be fully-qualified.
func ResolveMethodDescriptor(res protoencoding.Resolver, service, method string) (protoreflect.MethodDescriptor, error) {
	descriptor, err := res.FindDescriptorByName(protoreflect.FullName(service))
	if err == protoregistry.NotFound {
		return nil, fmt.Errorf("failed to find service named %q in schema", service)
	} else if err != nil {
		return nil, err
	}
	serviceDescriptor, ok := descriptor.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, fmt.Errorf("URL indicates service name %q, but that name is a %s", service, descriptorKind(descriptor))
	}
	methodDescriptor := serviceDescriptor.Methods().ByName(protoreflect.Name(method))
	if methodDescriptor == nil {
		return nil, fmt.Errorf("URL indicates method name %q, but service %q contains no such method", method, service)
	}
	return methodDescriptor, nil
}
