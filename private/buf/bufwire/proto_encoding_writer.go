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

	"github.com/bufbuild/buf/private/buf/buffetch"
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

func newProtoEncodingWriter(
	logger *zap.Logger,
	fetchWriter buffetch.Writer,
) *protoEncodingWriter {
	return &protoEncodingWriter{
		logger:      logger,
		fetchWriter: fetchWriter,
	}
}

func (i *protoEncodingWriter) PutMessage(
	ctx context.Context,
	container app.EnvStdoutContainer,
	message proto.Message,
	path string,
) (retErr error) {
	ctx, span := trace.StartSpan(ctx, "put_message")
	defer span.End()
	// Currently, this only support json format.
	data, err := protoencoding.NewJSONMarshalerIndent(nil).Marshal(message)
	if err != nil {
		return err
	}
	writeCloser, err := i.fetchWriter.PutFile(ctx, container, path)
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, writeCloser.Close())
	}()
	_, err = writeCloser.Write(data)
	return err
}
