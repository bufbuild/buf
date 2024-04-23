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

package bufstudioagent

import (
	"bytes"
	"fmt"
	"io"

	connect "connectrpc.com/connect"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"google.golang.org/protobuf/proto"
)

// bufferCodec is a connect.Codec for use with clients of type
// connect.Client[bytes.Buffer, bytes.Buffer] which does not attempt to parse
// messages but instead allows the application layer to work on the buffers
// directly. This is useful for creating proxies.
type bufferCodec struct {
	name string
}

var _ connect.Codec = (*bufferCodec)(nil)

func (b *bufferCodec) Name() string { return b.name }

func (b *bufferCodec) Marshal(src any) ([]byte, error) {
	switch typedSrc := src.(type) {
	case *bytes.Buffer:
		return typedSrc.Bytes(), nil
	case proto.Message:
		// When the codec is named "proto", connect will assume that it
		// may also be used to unmarshal the errors in the
		// grpc-status-details-bin trailer. The type used is not
		// exported so we match against the general proto.Message.
		return protoencoding.NewWireMarshaler().Marshal(typedSrc)
	default:
		return nil, fmt.Errorf("marshal unexpected type %T", src)
	}
}

func (b *bufferCodec) Unmarshal(src []byte, dst any) error {
	switch destination := dst.(type) {
	case *bytes.Buffer:
		destination.Reset()
		_, err := io.Copy(destination, bytes.NewReader(src))
		return err
	case proto.Message:
		// When the codec is named "proto", connect will assume that it
		// may also be used to unmarshal the errors in the
		// grpc-status-details-bin trailer. The type used is not
		// exported so we match against the general proto.Message.
		return protoencoding.NewWireUnmarshaler(nil).Unmarshal(src, destination)
	default:
		return fmt.Errorf("unmarshal unexpected type %T", dst)
	}
}
