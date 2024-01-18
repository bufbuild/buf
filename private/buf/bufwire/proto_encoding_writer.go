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

package bufwire

import (
	"context"
	"errors"

	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type protoEncodingWriter struct {
	logger      *zap.Logger
	fetchWriter buffetch.Writer
}

var _ ProtoEncodingWriter = &protoEncodingWriter{}

func newProtoEncodingWriter(
	logger *zap.Logger,
	fetchWriter buffetch.Writer,
) *protoEncodingWriter {
	return &protoEncodingWriter{
		logger:      logger,
		fetchWriter: fetchWriter,
	}
}

func (p *protoEncodingWriter) PutMessage(
	ctx context.Context,
	container app.EnvStdoutContainer,
	image bufimage.Image,
	message proto.Message,
	messageRef buffetch.MessageRef,
) (retErr error) {
	// Currently, this support binpb and JSON format.
	resolver, err := protoencoding.NewResolver(
		bufimage.ImageToFileDescriptorProtos(image)...,
	)
	if err != nil {
		return err
	}
	var marshaler protoencoding.Marshaler
	switch messageRef.MessageEncoding() {
	case buffetch.MessageEncodingBinpb:
		marshaler = protoencoding.NewWireMarshaler()
	case buffetch.MessageEncodingJSON:
		marshaler = newJSONMarshaler(resolver, messageRef)
	case buffetch.MessageEncodingTxtpb:
		marshaler = protoencoding.NewTxtpbMarshaler(resolver)
	case buffetch.MessageEncodingYAML:
		marshaler = newYAMLMarshaler(resolver, messageRef)
	default:
		return errors.New("unknown message encoding type")
	}
	data, err := marshaler.Marshal(message)
	if err != nil {
		return err
	}
	writeCloser, err := p.fetchWriter.PutMessageFile(ctx, container, messageRef)
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, writeCloser.Close())
	}()
	_, err = writeCloser.Write(data)
	return err
}
