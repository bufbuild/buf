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

package bufwire

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/bufbuild/buf/private/buf/bufconvert"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufreflect"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"go.opentelemetry.io/otel"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type protoEncodingReader struct {
	logger *zap.Logger
}

var _ ProtoEncodingReader = &protoEncodingReader{}

func newProtoEncodingReader(
	logger *zap.Logger,
) *protoEncodingReader {
	return &protoEncodingReader{
		logger: logger,
	}
}

func (p *protoEncodingReader) GetMessage(
	ctx context.Context,
	container app.EnvStdinContainer,
	image bufimage.Image,
	typeName string,
	messageRef bufconvert.MessageEncodingRef,
) (_ proto.Message, retErr error) {
	ctx, span := otel.GetTracerProvider().Tracer("bufbuild/buf").Start(ctx, "get_message")
	defer span.End()
	// Currently, this support bin and JSON format.
	resolver, err := protoencoding.NewResolver(
		bufimage.ImageToFileDescriptors(
			image,
		)...,
	)
	if err != nil {
		return nil, err
	}
	var unmarshaler protoencoding.Unmarshaler
	switch messageRef.MessageEncoding() {
	case bufconvert.MessageEncodingBin:
		unmarshaler = protoencoding.NewWireUnmarshaler(resolver)
	case bufconvert.MessageEncodingJSON:
		unmarshaler = protoencoding.NewJSONUnmarshaler(resolver)
	default:
		return nil, errors.New("unknown message encoding type")
	}
	readCloser := io.NopCloser(container.Stdin())
	if messageRef.Path() != "-" {
		var err error
		readCloser, err = os.Open(messageRef.Path())
		if err != nil {
			return nil, err
		}
	}
	defer func() {
		retErr = multierr.Append(retErr, readCloser.Close())
	}()
	data, err := io.ReadAll(readCloser)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, errors.New("size of input message must not be zero")
	}
	message, err := bufreflect.NewMessage(ctx, image, typeName)
	if err != nil {
		return nil, err
	}
	if err := unmarshaler.Unmarshal(data, message); err != nil {
		return nil, fmt.Errorf("unable to unmarshal the message: %v", err)
	}
	return message, nil
}
