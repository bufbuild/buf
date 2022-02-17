// Copyright 2020-2022 Buf Technologies, Inc.
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
	"fmt"

	"github.com/bufbuild/buf/private/buf/bufencoding"
	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"go.opencensus.io/trace"
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
	encoding bufencoding.MessageEncoding,
	path string,
) (retErr error) {
	ctx, span := trace.StartSpan(ctx, "put_message")
	defer span.End()
	// Currently, this support bin and JSON format.
	resolver, err := protoencoding.NewResolver(
		bufimage.ImageToFileDescriptors(
			image,
		)...,
	)
	if err != nil {
		return err
	}
	var marshaler protoencoding.Marshaler
	switch encoding {
	case bufencoding.MessageEncodingBin:
		marshaler = protoencoding.NewWireMarshaler()
	case bufencoding.MessageEncodingJSON:
		marshaler = protoencoding.NewJSONMarshalerIndent(resolver)
	default:
		return fmt.Errorf("unknown message encoding type")
	}
	data, err := marshaler.Marshal(message)
	if err != nil {
		return err
	}
	writeCloser, err := p.fetchWriter.PutSingleFile(ctx, container, path)
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, writeCloser.Close())
	}()
	_, err = writeCloser.Write(data)
	return err
}
