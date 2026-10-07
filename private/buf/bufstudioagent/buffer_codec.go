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

package bufstudioagent

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"connectrpc.com/connect/v2"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"google.golang.org/protobuf/proto"
)

// bufferCodec is a connect.Codec for use with clients that pass raw
// *bytes.Buffer messages, which does not attempt to parse messages but
// instead allows the application layer to work on the buffers directly. This
// is useful for creating proxies.
type bufferCodec struct {
	name string
}

var _ connect.Codec = (*bufferCodec)(nil)

func (b *bufferCodec) Name() string { return b.name }

func (b *bufferCodec) MarshalWrite(_ context.Context, dst io.Writer, src any) error {
	switch typedSrc := src.(type) {
	case *bytes.Buffer:
		_, err := dst.Write(typedSrc.Bytes())
		return err
	case proto.Message:
		// When the codec is named "proto", connect will assume that it
		// may also be used to marshal the errors in the
		// grpc-status-details-bin trailer. The type used is not
		// exported so we match against the general proto.Message.
		data, err := protoencoding.NewWireMarshaler().Marshal(typedSrc)
		if err != nil {
			return err
		}
		_, err = dst.Write(data)
		return err
	default:
		return fmt.Errorf("marshal unexpected type %T", src)
	}
}

func (b *bufferCodec) UnmarshalRead(_ context.Context, src io.Reader, dst any) error {
	switch destination := dst.(type) {
	case *bytes.Buffer:
		destination.Reset()
		_, err := io.Copy(destination, src)
		return err
	case proto.Message:
		// When the codec is named "proto", connect will assume that it
		// may also be used to unmarshal the errors in the
		// grpc-status-details-bin trailer. The type used is not
		// exported so we match against the general proto.Message.
		data, err := io.ReadAll(src)
		if err != nil {
			return err
		}
		return protoencoding.NewWireUnmarshaler(nil).Unmarshal(data, destination)
	default:
		return fmt.Errorf("unmarshal unexpected type %T", dst)
	}
}
